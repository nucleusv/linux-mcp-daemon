---
sidebar_position: 1
---

# list_directory

This tool allows the AI to securely explore the filesystem. It uses Go's native `os.ReadDir` to list the contents of a directory.

## Execution
This tool is executed by the Ephemeral Worker under strict UID isolation. It will only ever be able to list directories that the authenticated user's UID has `r-x` permissions to.

## Parameters
- `path` (string): Absolute path to list.
- `privileged` (boolean): Set to true to execute the worker as the root user.

### Example JSON-RPC
```json
{
  "method": "tools/call",
  "params": {
    "name": "list_directory",
    "arguments": {
      "path": "/var/log",
      "privileged": true
    }
  }
}
```
