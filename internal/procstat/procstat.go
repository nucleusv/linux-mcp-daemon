// Package procstat parses the parts of /proc that process tools share:
// system CPU time, memory, load and uptime, and per-process stat/statm/
// status. Everything is read natively - no ps/top binaries.
package procstat

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Root is the procfs mount point. A variable only so tests can point it
// at a fake tree.
var Root = "/proc"

// UtmpPath is where logged-in sessions are recorded.
var UtmpPath = "/var/run/utmp"

// ClockTicks is USER_HZ, the unit of every CPU time in /proc. It is 100 on
// every Linux architecture Go supports (the kernel fixes it for userspace
// ABI stability regardless of CONFIG_HZ), and reading it properly would
// need sysconf(_SC_CLK_TCK) via cgo.
const ClockTicks = 100

// CPUTimes is the aggregate "cpu" line of /proc/stat, in clock ticks.
type CPUTimes struct {
	User, Nice, System, Idle, IOWait, IRQ, SoftIRQ, Steal uint64
}

// Total excludes guest time, which the kernel already counts in User/Nice.
func (c CPUTimes) Total() uint64 {
	return c.User + c.Nice + c.System + c.Idle + c.IOWait + c.IRQ + c.SoftIRQ + c.Steal
}

// ReadCPU returns the aggregate CPU times from /proc/stat.
func ReadCPU() (CPUTimes, error) {
	f, err := os.Open(filepath.Join(Root, "stat"))
	if err != nil {
		return CPUTimes{}, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 9 || fields[0] != "cpu" {
			continue
		}
		v := make([]uint64, 8)
		for i := range v {
			v[i], _ = strconv.ParseUint(fields[i+1], 10, 64)
		}
		return CPUTimes{v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7]}, nil
	}
	return CPUTimes{}, fmt.Errorf("no cpu line in %s/stat", Root)
}

// Memory holds /proc/meminfo values in KiB.
type Memory struct {
	Total, Free, Buffers, Cached, SReclaimable, Available, SwapTotal, SwapFree uint64
}

// BuffCache is what top reports as buff/cache.
func (m Memory) BuffCache() uint64 { return m.Buffers + m.Cached + m.SReclaimable }

// Used is what top reports as used: total minus free minus buff/cache.
func (m Memory) Used() uint64 {
	if used := int64(m.Total) - int64(m.Free) - int64(m.BuffCache()); used > 0 {
		return uint64(used)
	}
	return 0
}

// ReadMemory parses /proc/meminfo.
func ReadMemory() (Memory, error) {
	data, err := os.ReadFile(filepath.Join(Root, "meminfo"))
	if err != nil {
		return Memory{}, err
	}
	var m Memory
	fields := map[string]*uint64{
		"MemTotal": &m.Total, "MemFree": &m.Free, "Buffers": &m.Buffers, "Cached": &m.Cached,
		"SReclaimable": &m.SReclaimable, "MemAvailable": &m.Available,
		"SwapTotal": &m.SwapTotal, "SwapFree": &m.SwapFree,
	}
	for _, line := range strings.Split(string(data), "\n") {
		name, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if p, ok := fields[name]; ok {
			f := strings.Fields(rest)
			if len(f) > 0 {
				*p, _ = strconv.ParseUint(f[0], 10, 64)
			}
		}
	}
	return m, nil
}

// ReadLoadAvg returns the 1/5/15-minute load averages.
func ReadLoadAvg() ([3]float64, error) {
	var l [3]float64
	data, err := os.ReadFile(filepath.Join(Root, "loadavg"))
	if err != nil {
		return l, err
	}
	f := strings.Fields(string(data))
	if len(f) < 3 {
		return l, fmt.Errorf("malformed loadavg")
	}
	for i := range l {
		l[i], _ = strconv.ParseFloat(f[i], 64)
	}
	return l, nil
}

// ReadUptime returns seconds since boot.
func ReadUptime() (float64, error) {
	data, err := os.ReadFile(filepath.Join(Root, "uptime"))
	if err != nil {
		return 0, err
	}
	f := strings.Fields(string(data))
	if len(f) < 1 {
		return 0, fmt.Errorf("malformed uptime")
	}
	return strconv.ParseFloat(f[0], 64)
}

// CountUsers counts logged-in sessions (USER_PROCESS records) in utmp, the
// same number top and uptime report. Returns 0 if utmp is unavailable (as
// in a container without the host's /run).
func CountUsers() int {
	data, err := os.ReadFile(UtmpPath)
	if err != nil {
		return 0
	}
	// glibc struct utmp on Linux (x86_64, arm64 and others): 384 bytes,
	// ut_type int16 at offset 0, ut_user[32] at offset 44.
	const recSize, userProcess, userOff = 384, 7, 44
	n := 0
	for off := 0; off+recSize <= len(data); off += recSize {
		rec := data[off : off+recSize]
		if int16(binary.LittleEndian.Uint16(rec[0:2])) == userProcess && rec[userOff] != 0 {
			n++
		}
	}
	return n
}

// Proc is one process, from /proc/<pid>/stat, statm and status.
type Proc struct {
	PID      int
	PPID     int
	Comm     string
	State    byte
	UTime    uint64 // clock ticks
	STime    uint64 // clock ticks
	Priority int64
	Nice     int64
	Threads  int64
	VirtKiB  uint64
	ResKiB   uint64
	ShrKiB   uint64
	EUID     int
}

// CPUTicks is total CPU time used, in clock ticks.
func (p Proc) CPUTicks() uint64 { return p.UTime + p.STime }

// ListPIDs returns every numeric entry under /proc.
func ListPIDs() ([]int, error) {
	entries, err := os.ReadDir(Root)
	if err != nil {
		return nil, err
	}
	var pids []int
	for _, e := range entries {
		if pid, err := strconv.Atoi(e.Name()); err == nil && e.IsDir() {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

// ReadProc reads one process. It fails if the process has exited.
func ReadProc(pid int) (Proc, error) {
	dir := filepath.Join(Root, strconv.Itoa(pid))
	stat, err := os.ReadFile(filepath.Join(dir, "stat"))
	if err != nil {
		return Proc{}, err
	}
	p, err := parseStat(stat)
	if err != nil {
		return Proc{}, err
	}
	p.PID = pid

	pageKiB := uint64(os.Getpagesize() / 1024)
	if statm, err := os.ReadFile(filepath.Join(dir, "statm")); err == nil {
		f := strings.Fields(string(statm))
		if len(f) >= 3 {
			size, _ := strconv.ParseUint(f[0], 10, 64)
			res, _ := strconv.ParseUint(f[1], 10, 64)
			shr, _ := strconv.ParseUint(f[2], 10, 64)
			p.VirtKiB, p.ResKiB, p.ShrKiB = size*pageKiB, res*pageKiB, shr*pageKiB
		}
	}

	p.EUID = -1
	if status, err := os.ReadFile(filepath.Join(dir, "status")); err == nil {
		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "Uid:") {
				// Real, effective, saved, filesystem - top shows effective.
				if f := strings.Fields(line); len(f) >= 3 {
					p.EUID, _ = strconv.Atoi(f[2])
				}
				break
			}
		}
	}
	return p, nil
}

// parseStat parses /proc/<pid>/stat. The command name is in parentheses
// and may itself contain spaces or ")", so fields are split after the
// LAST ")".
func parseStat(data []byte) (Proc, error) {
	open := bytes.IndexByte(data, '(')
	closeIdx := bytes.LastIndexByte(data, ')')
	if open < 0 || closeIdx < open || closeIdx+2 > len(data) {
		return Proc{}, fmt.Errorf("malformed stat")
	}
	p := Proc{Comm: string(data[open+1 : closeIdx])}
	f := strings.Fields(string(data[closeIdx+2:]))
	// f[0] is field 3 (state) in proc(5) numbering.
	if len(f) < 20 {
		return Proc{}, fmt.Errorf("malformed stat")
	}
	if len(f[0]) > 0 {
		p.State = f[0][0]
	}
	p.PPID, _ = strconv.Atoi(f[1])
	p.UTime, _ = strconv.ParseUint(f[11], 10, 64)
	p.STime, _ = strconv.ParseUint(f[12], 10, 64)
	p.Priority, _ = strconv.ParseInt(f[15], 10, 64)
	p.Nice, _ = strconv.ParseInt(f[16], 10, 64)
	p.Threads, _ = strconv.ParseInt(f[17], 10, 64)
	return p, nil
}

// Cmdline returns the process's command line, arguments space-joined.
func Cmdline(pid int) string {
	data, err := os.ReadFile(filepath.Join(Root, strconv.Itoa(pid), "cmdline"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", " "))
}

// Usernames maps uid to name from /etc/passwd - one read instead of a
// lookup per process.
func Usernames() map[int]string {
	names := map[int]string{}
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return names
	}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Split(line, ":")
		if len(f) >= 3 {
			if uid, err := strconv.Atoi(f[2]); err == nil {
				names[uid] = f[0]
			}
		}
	}
	return names
}
