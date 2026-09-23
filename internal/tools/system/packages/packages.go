package packages

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
)

// GetPackagesArgs are the tool's input arguments.
type GetPackagesArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	Name         string `json:"name,omitempty"`          // Name filters packages by name: a glob ("linux-*", "*ssl*") or an exact name.
}

// Package describes a single installed package, normalized across the
// different native package database formats this tool understands.
type Package struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Architecture string `json:"architecture,omitempty"`
}

// List returns the packages installed on the filesystem this worker
// currently sees. With "privileged": true, and this daemon deployed
// containerized (see worker.Containerized), that's automatically the real
// host's package state rather than this container's own image.
func List(argsJSON []byte) (string, error) {
	var args GetPackagesArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	manager, err := detectManager()
	if err != nil {
		return "", err
	}

	var pkgs []Package
	switch manager {
	case "dpkg":
		pkgs, err = parseDpkgStatus("/var/lib/dpkg/status")
	case "apk":
		pkgs, err = parseApkInstalled("/lib/apk/db/installed")
	default:
		return "", fmt.Errorf("package listing for %s-based systems isn't implemented natively yet - its package database isn't a plain-text format safe to hand-parse, so this needs a documented CLI-wrapping exception (see ARCHITECTURE.md) rather than guesswork", manager)
	}
	if err != nil {
		return "", err
	}

	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })

	if args.Name != "" {
		if _, err := path.Match(args.Name, ""); err != nil {
			return "", fmt.Errorf("invalid name pattern %q: %v", args.Name, err)
		}
		var kept []Package
		for _, p := range pkgs {
			if ok, _ := path.Match(args.Name, p.Name); ok {
				kept = append(kept, p)
			}
		}
		pkgs = kept
	}
	if pkgs == nil {
		pkgs = []Package{}
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		b, err := json.Marshal(pkgs)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %v", err)
		}
		return string(b), nil
	}

	// Text: the actual list, dpkg -l style. It used to be only a count
	// ("397 packages installed"), so a caller asking without a format got
	// no package names at all.
	nameW, verW := len("NAME"), len("VERSION")
	for _, p := range pkgs {
		nameW = max(nameW, len(p.Name))
		verW = max(verW, len(p.Version))
	}
	var b strings.Builder
	what := "packages installed"
	if args.Name != "" {
		what = fmt.Sprintf("installed packages matching %q", args.Name)
	}
	fmt.Fprintf(&b, "%d %s (%s)\n", len(pkgs), what, manager)
	if len(pkgs) > 0 {
		fmt.Fprintf(&b, "%-*s  %-*s  %s\n", nameW, "NAME", verW, "VERSION", "ARCH")
		for _, p := range pkgs {
			fmt.Fprintf(&b, "%-*s  %-*s  %s\n", nameW, p.Name, verW, p.Version, p.Architecture)
		}
	}
	return b.String(), nil
}

// detectManager identifies the package manager in scope by checking for its
// canonical database file/directory. A host runs exactly one of these.
func detectManager() (string, error) {
	if _, err := os.Stat("/var/lib/dpkg/status"); err == nil {
		return "dpkg", nil
	}
	if _, err := os.Stat("/lib/apk/db/installed"); err == nil {
		return "apk", nil
	}
	if _, err := os.Stat("/var/lib/rpm"); err == nil {
		return "rpm", nil
	}
	return "", fmt.Errorf("no known package database found (checked dpkg, apk, rpm) - if this was meant to query the real host, check that \"privileged\" was set to true and that this daemon's worker.containerized config setting is correct")
}

// parseDpkgStatus natively parses dpkg's status database (Debian/Ubuntu).
// The file is a sequence of RFC822-style stanzas separated by blank lines;
// each stanza describes one package. Multi-line fields (like Description)
// continue on lines starting with whitespace, which we skip since we only
// need the single-line fields below.
func parseDpkgStatus(path string) ([]Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer f.Close()

	var pkgs []Package
	var name, version, arch, status string

	flush := func() {
		// Status is "<want> <flag> <status>", e.g. "install ok installed".
		// Only the fully "installed" state counts - "half-installed",
		// "config-files", "unpacked" etc. are not currently-installed
		// packages, even though some contain "install" as a substring.
		if name != "" && status == "installed" {
			pkgs = append(pkgs, Package{Name: name, Version: version, Architecture: arch})
		}
		name, version, arch, status = "", "", "", ""
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch key {
		case "Package":
			name = val
		case "Version":
			version = val
		case "Architecture":
			arch = val
		case "Status":
			fields := strings.Fields(val)
			if len(fields) == 3 {
				status = fields[2]
			}
		}
	}
	flush() // the final stanza has no trailing blank line to trigger on
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", path, err)
	}
	return pkgs, nil
}

// parseApkInstalled natively parses Alpine's apk installed-packages database:
// one field per line as "<letter>:<value>", blank line separates packages.
func parseApkInstalled(path string) ([]Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer f.Close()

	var pkgs []Package
	var name, version, arch string

	flush := func() {
		if name != "" {
			pkgs = append(pkgs, Package{Name: name, Version: version, Architecture: arch})
		}
		name, version, arch = "", "", ""
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}
		if len(line) < 2 || line[1] != ':' {
			continue
		}
		val := line[2:]
		switch line[0] {
		case 'P':
			name = val
		case 'V':
			version = val
		case 'A':
			arch = val
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", path, err)
	}
	return pkgs, nil
}
