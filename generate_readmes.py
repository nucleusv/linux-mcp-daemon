import os

tools = {
    "auth/get/sudo_rules": ("get_sudo_rules", "Reads the authorized privileged tools for the user from mcp-sudo.yaml."),
    "cpu/get/info": ("get_info", "Returns native CPU topology and architecture metadata from /proc/cpuinfo."),
    "cpu/get/load_average": ("get_load_average", "Returns the 1, 5, and 15 minute system load averages using native syscall.Sysinfo."),
    "disks/get/blocks": ("get_blocks", "Lists block devices natively by reading /sys/class/block/."),
    "disks/get/free": ("get_free", "Returns filesystem disk space usage for a given path using syscall.Statfs."),
    "disks/get/usage": ("get_usage", "Calculates file space usage natively via filepath.WalkDir."),
    "files/get/list_of_files": ("get_list_of_files", "Lists the contents of a directory natively via os.ReadDir."),
    "memory/get/usage": ("get_memory_usage", "Returns detailed memory usage metrics by parsing /proc/meminfo."),
    "network/get/connections": ("get_connections", "Lists active network sockets and connections."),
    "network/get/interfaces": ("get_interfaces", "Lists native network interfaces and IP addresses via net.Interfaces."),
    "processes/delete/process": ("delete_process", "Sends a signal to a process using syscall.Kill."),
    "processes/get/processes": ("get_processes", "Lists running processes by iterating over /proc natively."),
    "system/get/os_release": ("get_os_release", "Returns system and kernel release info using syscall.Uname and /etc/os-release.")
}

template = """# `{tool_name}`

{desc}

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload instead of human-readable text.
"""

for path, (name, desc) in tools.items():
    full_path = os.path.join("internal/tools", path)
    if not os.path.exists(full_path):
        os.makedirs(full_path)
    readme_path = os.path.join(full_path, "README.md")
    
    with open(readme_path, "w") as f:
        f.write(template.format(tool_name=name, desc=desc))
