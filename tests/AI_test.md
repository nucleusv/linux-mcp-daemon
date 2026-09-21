# AI Testing Guide for Linux MCP Daemon

Welcome, AI Agent! If you are reading this, your task is to act as an MCP Client and test the capabilities of the `linux-mcp-daemon`. This document outlines how you can connect to the daemon natively via your internal tools, and lists a set of complex challenges for you to execute on the host machine.

## How to Connect (For AI Agents)

Since you are an AI, you already have the ability to run terminal commands. To test this daemon, you do NOT need to write Python or Node.js MCP clients. You can interact with the daemon directly via HTTP/SSE using `curl` or by leveraging the built-in `linuxctl` CLI tool.

### Method 1: Using `linuxctl` (Recommended)
The `linuxctl` binary is an MCP client wrapper. You can use it to natively query the daemon.
```bash
# Ensure you are in the project root
export MCP_TOKEN="my-test-token-123"
export DAEMON_URL="http://localhost:9090"

# Example: List CPU information (Regular User)
./linuxctl cpu list --output json

# Example: Read protected USB devices (Privileged User)
./linuxctl -privileged devices usb --output json
```

### Method 2: Direct HTTP JSON-RPC via `curl`
If you want to test the raw MCP protocol over HTTP:
1. Hit the SSE endpoint to retrieve a session ID: `curl -s -N -H "Authorization: Bearer my-test-token-123" http://localhost:9090/sse`
2. Extract the `endpoint` URI from the `event: endpoint` payload.
3. Send a POST request to that URI with standard MCP JSON-RPC payloads (e.g., `tools/call`, `resources/list`, `resources/read`).

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
The daemon supports granular Role-Based Access Control (RBAC). Your task is to verify that the security boundary holds.
1. Attempt to execute `files/list` on `/root/` using the daemon as a **regular user** (this should fail with a permission error).
2. Attempt to execute the exact same command using the `-privileged` flag. Observe if it succeeds or is blocked by `mcp-sudo.yaml`.
3. Attempt to read the `devices://usb` resource as a **regular user** (should be denied).
4. Read the `devices://usb` resource again as a **privileged user** (should succeed).
5. Query `auth/sudo-rules` to retrieve the current active RBAC ruleset and analyze why your requests succeeded or failed.
6. **Deliverable**: A security audit report of the daemon's current active Sudo rules, detailing the exact results of the unprivileged vs privileged tests.

### Task 4: Process and Resource Management
1. Execute `processes/list` to find the top 5 CPU-consuming tasks.
2. Identify a safe, non-critical dummy process (or spawn one like `sleep 1000 &`).
3. Use the `processes/delete` tool to terminate that process.
4. **Deliverable**: Confirm successful termination by re-listing the processes.

---
**Agent Instructions:** When the human user asks you to "Run the AI Test Plan", start from Task 1 and work your way through, reporting your findings at each step using beautiful Markdown tables and structured analysis!
