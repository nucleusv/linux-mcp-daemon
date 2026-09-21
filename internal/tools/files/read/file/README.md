# read_file

Read the contents of a file on the filesystem.

*Note: This internal tool is strictly intended for isolated worker dispatcher use when accessing the `file:///{path}` resource URN. The MCP daemon exposes this securely to clients via the resources subsystem.*

## Input Schema

```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "The absolute path of the file to read."
    }
  },
  "required": ["path"]
}
```
