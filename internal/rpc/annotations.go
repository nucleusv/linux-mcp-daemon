package rpc

// MCP tool annotations (readOnlyHint, destructiveHint, idempotentHint,
// openWorldHint, title) for every tool tools/list returns. Clients use them to
// decide what needs a confirmation, and Glama's quality score reads them
// instead of making each description spell out "read-only" or "destructive"
// (FR-024, GUIDELINES.md section 9). A new tool needs a row here: the test in
// annotations_test.go fails without one.

type toolAnnotation struct {
	Title                                   string
	ReadOnly, Destructive, Idempotent, Open bool
}

var toolAnnotations = map[string]toolAnnotation{
	"files/list":            {"List directory", true, false, true, false},
	"files/read":            {"Read file contents", true, false, true, false},
	"files/create":          {"Create or overwrite file", false, true, true, false},
	"files/update":          {"Edit file (append or replace lines)", false, true, false, false},
	"files/find":            {"Find files", true, false, true, false},
	"files/filetype":        {"Detect file MIME type", true, false, true, false},
	"files/chmod":           {"Change file permissions", false, true, true, false},
	"files/chown":           {"Change file owner/group", false, true, true, false},
	"disks/free":            {"Filesystem free space", true, false, true, false},
	"disks/usage":           {"Directory disk usage", true, false, true, false},
	"processes/top":         {"Process snapshot (top)", true, false, true, false},
	"processes/list":        {"List processes", true, false, true, false},
	"processes/delete":      {"Signal a process", false, true, false, false},
	"network/nslookup":      {"DNS lookup", true, false, true, true},
	"network/curl":          {"HTTP request", false, true, false, true},
	"network/arp":           {"ARP cache", true, false, true, false},
	"network/ping":          {"TCP connect ping", true, false, true, true},
	"network/connections":   {"List sockets", true, false, true, false},
	"memory/usage":          {"Memory and swap usage", true, false, true, false},
	"services/manage":       {"Control systemd service", false, true, false, false},
	"services/list":         {"List systemd services", true, false, true, false},
	"timers/list":           {"List systemd timers", true, false, true, false},
	"logs/journal-control":  {"Query systemd journal", true, false, true, false},
	"logs/dmesg":            {"Kernel ring buffer", true, false, true, false},
	"logs/logins":           {"Login history", true, false, true, false},
	"kernel/system-control": {"Read/write sysctl", false, true, true, false},
	"cpu/list":              {"CPU info", true, false, true, false},
	"cpu/load-average":      {"Load average", true, false, true, false},
	"disks/list":            {"List block devices", true, false, true, false},
	"disks/mounts":          {"List mounts", true, false, true, false},
	"disks/performance":     {"Disk I/O counters", true, false, true, false},
	"disks/health":          {"SMART disk health", true, false, true, false},
	"disks/partitions":      {"Partition geometry", true, false, true, false},
	"network/trace-path":    {"Trace network path", true, false, true, true},
	"system/os-release":     {"OS and kernel version", true, false, true, false},
	"system/packages":       {"List installed packages", true, false, true, false},
	"users/list":            {"List user accounts", true, false, true, false},
	"auth/sudo-rules":       {"My root grants", true, false, true, false},
	"docker/containers":     {"List containers", true, false, true, false},
	"docker/manage":         {"Container lifecycle", false, true, false, false},
	"docker/logs":           {"Container logs", true, false, true, false},
	"docker/exec":           {"Run command in container", false, true, false, false},
	"docker/images":         {"List images", true, false, true, false},
	"docker/volumes":        {"List volumes", true, false, true, false},
	"docker/networks":       {"List Docker networks", true, false, true, false},
	"docker/prune":          {"Prune unused Docker objects", false, true, true, false},
	"daemon/reload-config":  {"Reload mcpd config", false, false, true, false},
}

// annotateTools adds the "annotations" object to each tool schema that has a row.
func annotateTools(tools []interface{}) {
	for _, t := range tools {
		tool, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := tool["name"].(string)
		a, ok := toolAnnotations[name]
		if !ok {
			continue
		}
		tool["annotations"] = map[string]interface{}{
			"title":           a.Title,
			"readOnlyHint":    a.ReadOnly,
			"destructiveHint": a.Destructive,
			"idempotentHint":  a.Idempotent,
			"openWorldHint":   a.Open,
		}
	}
}
