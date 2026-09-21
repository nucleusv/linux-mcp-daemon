# AI Testing Guide for Linux MCP Daemon

Welcome, AI Agent! If you are reading this, your task is to act as an MCP Client and test the capabilities of the `linux-mcp-daemon`. This document outlines how you can connect to the daemon natively via your internal tools, and lists a set of complex challenges for you to execute on the host machine.

## How to Connect (For AI Agents)

Since you are an AI, you already have the ability to run terminal commands. To test this daemon, you do NOT need to write Python or Node.js MCP clients. You can interact with the daemon directly via HTTP/SSE using `curl` or by leveraging the built-in `linuxctl` CLI tool.

### Method 1: Using `linuxctl` (Recommended)
The `linuxctl` binary is an MCP client wrapper. You can use it to natively query the daemon.
```bash
# Ensure you are in the project root
export DAEMON_URL="http://localhost:9090"

# The daemon maps tokens to RBAC roles (defined in configs/mcp-sudo.yaml)
export TOKEN_PRIV="my-test-token-123"           # Maps to 'testuser' (Privileged)
export TOKEN_UNPRIV="my-unprivileged-token-123" # Maps to 'unpriviliged' (Restricted)

# Example: List CPU information (Privileged User)
./linuxctl -token $TOKEN_PRIV cpu list --output json

# Example: Attempt to read protected USB devices (Unprivileged User)
./linuxctl -token $TOKEN_UNPRIV resource devices://usb --output json
```

### Method 2: Standard MCP Clients (e.g., MCP Inspector)
If you prefer not to use `linuxctl`, you must connect using a legitimate MCP Client over the SSE transport layer. DO NOT use raw `curl` for JSON-RPC, as it violates the standard MCP streaming paradigms. 
- Use the official **MCP Inspector** (`npx @modelcontextprotocol/inspector`).
- Write a quick script using the official `@modelcontextprotocol/sdk` (Node or Python).

## AI Desktop Client Configurations

If you want to configure **Claude Desktop** or **Antigravity IDE** to permanently connect to this remote Linux daemon, you need an SSE-to-Stdio proxy (since desktop clients natively expect local `stdio` binaries).

You can use the official MCP SSE client proxy in your config file:

### Claude Desktop (`claude_desktop_config.json`)
```json
{
  "mcpServers": {
    "linux-remote": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/client-sse",
        "--url",
        "http://<LINUX_MCP_HOST_IP>:9090/sse",
        "--header",
        "Authorization: Bearer my-test-token-123"
      ]
    }
  }
}
```

### Antigravity IDE (`.agents/mcp_config.json`)
Antigravity IDE uses the exact same universal `mcpServers` format:
```json
{
  "mcpServers": {
    "linux-remote": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/client-sse",
        "--url",
        "http://<LINUX_MCP_HOST_IP>:9090/sse",
        "--header",
        "Authorization: Bearer my-test-token-123"
      ]
    }
  }
}
```

### Claude Code CLI
If you are using the terminal-based **Claude Code**, you can dynamically add the remote server using the `mcp add` command:
```bash
claude mcp add linux-remote npx -y @modelcontextprotocol/client-sse --url http://<LINUX_MCP_HOST_IP>:9090/sse --header "Authorization: Bearer my-test-token-123"
```
## AI Task Challenges

Your goal is to complete the following tasks using the MCP Daemon. Do not use standard linux shell utilities (like `ls`, `ps`, `cat`) directly. You MUST retrieve all information exclusively through the MCP daemon tools and resources.

### Task 1: Hardware Introspection
1. Query the daemon's **Resources** to find the URI for PCI devices.
2. Read the PCI devices resource.
3. Query the daemon's **Tools** to get CPU and Disk statistics.
4. **Deliverable**: Synthesize a comprehensive JSON report describing the host's hardware topology.

### Task 2: Network Forensics
1. Execute the `network/connections` tool to find all active listening ports.
2. Identify the process ID running the `mcpd` daemon on port `9090`.
3. Execute a `network/ping` via the daemon to verify connectivity to `1.1.1.1`.
4. **Deliverable**: A summary of network anomalies or active listening services.

### Task 3: Privilege Boundary Testing (RBAC)
The daemon supports granular Role-Based Access Control (RBAC) via the `mcp-sudo.yaml` config. Your task is to verify that the security boundary holds between users.
1. Attempt to execute `files/list` on `/root/` using the daemon with `$TOKEN_UNPRIV` (this should fail).
2. Attempt to execute the exact same command, but pass the `--privileged true` argument to the tool. Observe that it STILL fails for `$TOKEN_UNPRIV`.
3. Now switch to `$TOKEN_PRIV`. Execute `files/list` on `/root/` with `--privileged true`. Observe that it succeeds!
4. Attempt to read the `devices://usb` resource using `$TOKEN_UNPRIV` (should be denied).
5. Read the `devices://usb` resource again using `$TOKEN_PRIV` (should succeed).
6. **Deliverable**: A security audit report of the daemon's current active Sudo rules, detailing the exact results of the unprivileged vs privileged tool executions.

### Task 4: Process and Resource Management
1. Execute `processes/list` to find the top 5 CPU-consuming tasks.
2. Identify a safe, non-critical dummy process (or spawn one like `sleep 1000 &`).
3. Use the `processes/delete` tool to terminate that process.
4. **Deliverable**: Confirm successful termination by re-listing the processes (the dummy process should be missing) or explain why it failed based on container isolation.

### Task 5: Exhaustive Tool Matrix Validation
Perform real-world logical checks using the remaining tools. Use different parameters to prove the tools work dynamically.
1. **`network/nslookup`**: Perform a DNS lookup for `github.com` to verify that the container's nameservers (e.g., CoreDNS) are resolving external domains correctly.
2. **`network/curl`**: Perform a GET request to `http://ident.me` or `https://api.github.com/zen` to prove external HTTP egress routing is functional. Check the HTTP response status code in the output.
3. **`network/arp`**: Retrieve the ARP table and identify the MAC address of the default gateway (usually the first `.1` IP in the subnet).
4. **`cpu/load-average`**: Retrieve the system load average. Compare the 1-minute load against the number of CPU cores (from `cpu/list`) to logically determine if the host is overloaded.
5. **`memory/usage`**: Retrieve memory statistics. Calculate the exact percentage of RAM currently in use (`(Total - Free) / Total * 100`).
6. **`disks/free`**: Query the free space specifically on the `/tmp` directory. If it's less than 1GB, flag a logical warning in your final report.
7. **`files/read`**: Read the contents of `/etc/resolv.conf` to identify exactly which nameservers the container is using.
8. **`system/os-release`**: Retrieve the OS information to prove whether the daemon is running on Debian, Alpine, or Ubuntu.
9. **Deliverable**: A comprehensive "Health & Connectivity" dashboard report containing the calculated memory percentage, load status, external IP/egress status, and DNS health.
---
**Agent Instructions:** When the human user asks you to "Run the AI Test Plan", start from Task 1 and work your way through, reporting your findings at each step using beautiful Markdown tables and structured analysis!
