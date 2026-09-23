package list

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

type GetBlockDevicesArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	All          bool   `json:"all"`                     // All includes empty devices and RAM disks, like `lsblk -a`.
	Privileged   bool   `json:"privileged,omitempty"`    // Privileged runs as root - in containerized deployments this reads the host's mount table for MOUNTPOINTS.
}

// BlockDevice mirrors one entry of `lsblk -J`'s "blockdevices" tree, plus
// size_bytes so callers don't have to parse the human-readable size.
type BlockDevice struct {
	Name        string         `json:"name"`
	KName       string         `json:"kname"`
	MajMin      string         `json:"maj:min"`
	RM          bool           `json:"rm"`
	Size        string         `json:"size"`
	SizeBytes   uint64         `json:"size_bytes"`
	RO          bool           `json:"ro"`
	Type        string         `json:"type"`
	MountPoints []string       `json:"mountpoints"`
	Children    []*BlockDevice `json:"children,omitempty"`

	parent string // kernel name of the parent disk, for partitions only
}

const sysBlock = "/sys/class/block"

// List returns block devices as a tree, equivalent to `lsblk`: partitions
// nest under their disk, and device-mapper/LVM/RAID volumes nest under the
// devices they're built on (via each device's holders/ directory).
// Everything is read natively from sysfs and procfs - no lsblk dependency.
func List(argsJSON []byte) (string, error) {
	var args GetBlockDevicesArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	entries, err := os.ReadDir(sysBlock)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %v", sysBlock, err)
	}

	mounts := readMountPoints()

	devices := map[string]*BlockDevice{}
	var names []string
	for _, entry := range entries {
		dev := readDevice(entry.Name(), mounts)
		// Same default filtering as lsblk: hide empty devices and RAM
		// disks (major 1) unless --all.
		if !args.All && (dev.SizeBytes == 0 || strings.HasPrefix(dev.MajMin, "1:")) {
			continue
		}
		devices[dev.KName] = dev
		names = append(names, dev.KName)
	}
	sort.Slice(names, func(i, j int) bool { return lessKName(names[i], names[j]) })

	// Link children to parents: partitions by sysfs nesting, stacked
	// devices (dm, md) by the lower device's holders/ entries.
	hasParent := map[string]bool{}
	for _, name := range names {
		dev := devices[name]
		if dev.parent != "" {
			if p, ok := devices[dev.parent]; ok {
				p.Children = append(p.Children, dev)
				hasParent[name] = true
			}
		}
		holders, _ := os.ReadDir(filepath.Join(sysBlock, name, "holders"))
		var holderNames []string
		for _, h := range holders {
			holderNames = append(holderNames, h.Name())
		}
		sort.Slice(holderNames, func(i, j int) bool { return lessKName(holderNames[i], holderNames[j]) })
		for _, h := range holderNames {
			if child, ok := devices[h]; ok {
				dev.Children = append(dev.Children, child)
				hasParent[h] = true
			}
		}
	}

	var roots []*BlockDevice
	for _, name := range names {
		if !hasParent[name] {
			roots = append(roots, devices[name])
		}
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		b, err := json.Marshal(map[string]interface{}{"blockdevices": roots})
		if err != nil {
			return "", fmt.Errorf("failed to encode block devices: %v", err)
		}
		return string(b), nil
	}

	return formatTree(roots), nil
}

func readSysfs(name, attr string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(sysBlock, name, attr))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(b)), true
}

func readDevice(kname string, mounts map[string][]string) *BlockDevice {
	dev := &BlockDevice{Name: kname, KName: kname, MountPoints: []string{}}

	if v, ok := readSysfs(kname, "dev"); ok {
		dev.MajMin = v
	}
	if v, ok := readSysfs(kname, "size"); ok {
		if sectors, err := strconv.ParseUint(v, 10, 64); err == nil {
			dev.SizeBytes = sectors * 512
		}
	}
	dev.Size = humanSize(dev.SizeBytes)
	if v, ok := readSysfs(kname, "ro"); ok {
		dev.RO = v == "1"
	}

	_, isPartition := readSysfs(kname, "partition")
	if isPartition {
		// A partition's sysfs dir is nested inside its disk's
		// (.../block/sda/sda1); it has no "removable" file of its own.
		if realPath, err := filepath.EvalSymlinks(filepath.Join(sysBlock, kname)); err == nil {
			dev.parent = filepath.Base(filepath.Dir(realPath))
		}
	}
	rmName := kname
	if isPartition && dev.parent != "" {
		rmName = dev.parent
	}
	if v, ok := readSysfs(rmName, "removable"); ok {
		dev.RM = v == "1"
	}

	dev.Type = deviceType(kname, isPartition)
	if dmName, ok := readSysfs(kname, "dm/name"); ok && dmName != "" {
		dev.Name = dmName
	}
	if mp, ok := mounts[dev.MajMin]; ok {
		dev.MountPoints = mp
	}
	return dev
}

// deviceType follows lsblk's TYPE column conventions.
func deviceType(kname string, isPartition bool) string {
	if isPartition {
		return "part"
	}
	if uuid, ok := readSysfs(kname, "dm/uuid"); ok {
		switch {
		case strings.HasPrefix(uuid, "LVM-"):
			return "lvm"
		case strings.HasPrefix(uuid, "CRYPT-"):
			return "crypt"
		case strings.HasPrefix(uuid, "mpath-"):
			return "mpath"
		case strings.HasPrefix(uuid, "part"):
			return "part"
		}
		return "dm"
	}
	if level, ok := readSysfs(kname, "md/level"); ok && level != "" {
		return level
	}
	if strings.HasPrefix(kname, "loop") {
		return "loop"
	}
	// SCSI peripheral type 5 is a CD/DVD drive.
	if t, ok := readSysfs(kname, "device/type"); ok && t == "5" {
		return "rom"
	}
	return "disk"
}

// mountinfoEscape matches the octal escapes procfs uses for whitespace and
// backslashes in mount paths.
var mountinfoEscape = regexp.MustCompile(`\\[0-7]{3}`)

func unescape(s string) string {
	return mountinfoEscape.ReplaceAllStringFunc(s, func(m string) string {
		n, err := strconv.ParseInt(m[1:], 8, 32)
		if err != nil {
			return m
		}
		return string(rune(n))
	})
}

// readMountPoints maps "major:minor" to that device's mount points, plus
// "[SWAP]" for active swap devices, the same way lsblk's MOUNTPOINTS does.
// Uses /proc/thread-self for the same reason as disks/mounts: after a
// per-thread setns into the host mount namespace, it reliably reflects the
// calling thread's namespace.
func readMountPoints() map[string][]string {
	result := map[string][]string{}
	seen := map[string]bool{}
	add := func(majMin, mp string) {
		if seen[majMin+"\x00"+mp] {
			return
		}
		seen[majMin+"\x00"+mp] = true
		result[majMin] = append(result[majMin], mp)
	}

	if content, err := os.ReadFile("/proc/thread-self/mountinfo"); err == nil {
		// Format: id parent-id major:minor root mount-point options ...
		for _, line := range strings.Split(string(content), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}
			add(fields[2], unescape(fields[4]))
		}
	}

	if content, err := os.ReadFile("/proc/swaps"); err == nil {
		for i, line := range strings.Split(string(content), "\n") {
			fields := strings.Fields(line)
			if i == 0 || len(fields) < 2 || fields[1] != "partition" {
				continue
			}
			var st unix.Stat_t
			if err := unix.Stat(unescape(fields[0]), &st); err != nil {
				continue
			}
			add(fmt.Sprintf("%d:%d", unix.Major(uint64(st.Rdev)), unix.Minor(uint64(st.Rdev))), "[SWAP]")
		}
	}
	return result
}

// humanSize formats bytes the way lsblk does: powers of 1024, one decimal
// place, trailing ".0" dropped (60G, 3.7G, 2K).
func humanSize(bytes uint64) string {
	units := []string{"B", "K", "M", "G", "T", "P", "E"}
	value := float64(bytes)
	i := 0
	for value >= 1024 && i < len(units)-1 {
		value /= 1024
		i++
	}
	rounded := math.Round(value*10) / 10
	if rounded >= 1024 && i < len(units)-1 {
		rounded = math.Round(rounded/1024*10) / 10
		i++
	}
	s := strconv.FormatFloat(rounded, 'f', 1, 64)
	s = strings.TrimSuffix(s, ".0")
	return s + units[i]
}

// lessKName sorts kernel names naturally (sda2 before sda10, dm-2 before dm-10).
func lessKName(a, b string) bool {
	pa, na := splitTrailingNumber(a)
	pb, nb := splitTrailingNumber(b)
	if pa != pb {
		return pa < pb
	}
	return na < nb
}

func splitTrailingNumber(s string) (string, int) {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	n, err := strconv.Atoi(s[i:])
	if err != nil {
		return s, -1
	}
	return s[:i], n
}

type row struct {
	name, major, minor, rm, size, ro, typ string
	mounts                                []string
	cont                                  string // tree prefix for continuation lines
}

// flatten walks the tree depth-first, building lsblk's "├─"/"└─" name
// prefixes. Top-level devices get no branch glyph.
func flatten(devs []*BlockDevice, prefix string, top bool, rows *[]row) {
	for i, d := range devs {
		branch, cont := "", ""
		if !top {
			branch, cont = "├─", "│ "
			if i == len(devs)-1 {
				branch, cont = "└─", "  "
			}
		}
		major, minor := d.MajMin, ""
		if parts := strings.SplitN(d.MajMin, ":", 2); len(parts) == 2 {
			major, minor = parts[0], parts[1]
		}
		*rows = append(*rows, row{
			name:   prefix + branch + d.Name,
			major:  major,
			minor:  minor,
			rm:     boolDigit(d.RM),
			size:   d.Size,
			ro:     boolDigit(d.RO),
			typ:    d.Type,
			mounts: d.MountPoints,
			cont:   prefix + cont,
		})
		flatten(d.Children, prefix+cont, false, rows)
	}
}

func boolDigit(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func padRight(s string, w int) string {
	if n := utf8.RuneCountInString(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func padLeft(s string, w int) string {
	if n := utf8.RuneCountInString(s); n < w {
		return strings.Repeat(" ", w-n) + s
	}
	return s
}

// formatTree renders the same layout as plain `lsblk`.
func formatTree(roots []*BlockDevice) string {
	var rows []row
	flatten(roots, "", true, &rows)

	nameW, majW, minW, sizeW, typeW := len("NAME"), len("MAJ"), len("MIN"), len("SIZE"), len("TYPE")
	for _, r := range rows {
		nameW = max(nameW, utf8.RuneCountInString(r.name))
		majW = max(majW, len(r.major))
		minW = max(minW, len(r.minor))
		sizeW = max(sizeW, len(r.size))
		typeW = max(typeW, len(r.typ))
	}

	var sb strings.Builder
	writeLine := func(s string) {
		sb.WriteString(strings.TrimRight(s, " "))
		sb.WriteString("\n")
	}
	writeLine(strings.Join([]string{
		padRight("NAME", nameW),
		padLeft("MAJ", majW) + ":" + padRight("MIN", minW),
		"RM",
		padLeft("SIZE", sizeW),
		"RO",
		padRight("TYPE", typeW),
		"MOUNTPOINTS",
	}, " "))

	for _, r := range rows {
		first := ""
		if len(r.mounts) > 0 {
			first = r.mounts[0]
		}
		writeLine(strings.Join([]string{
			padRight(r.name, nameW),
			padLeft(r.major, majW) + ":" + padRight(r.minor, minW),
			padLeft(r.rm, 2),
			padLeft(r.size, sizeW),
			padLeft(r.ro, 2),
			padRight(r.typ, typeW),
			first,
		}, " "))
		// Extra mount points go on their own lines under MOUNTPOINTS, as
		// lsblk does.
		indent := nameW + 1 + majW + 1 + minW + 1 + 2 + 1 + sizeW + 1 + 2 + 1 + typeW + 1
		for _, mp := range r.mounts[min(1, len(r.mounts)):] {
			writeLine(padRight(r.cont, indent) + mp)
		}
	}
	return sb.String()
}
