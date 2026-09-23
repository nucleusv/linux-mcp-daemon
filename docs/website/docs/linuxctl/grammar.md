---
sidebar_label: 'Grammar & Commands'
sidebar_position: 2
---

# Grammar & Commands

## The grammar

```
linuxctl <verb> <group> [target-keyword] [name/args]
```

- **`<group>`** is always present, always the tool's real `tools_group` (`files`, `disks`, `processes`, `network`, `devices`, `kernel`, `logs`, `system`, `users`, `cpu`, `memory`, `auth`). There is no top-level `services` group - `services/list` and `services/manage` live under `system`, reached as `system services`.
- **`<verb>`** is one of exactly two universal words for reads-and-detail, plus a tool's own action word for mutations:
  - **`get`** - the sole read verb, one result or many, no separate `list`. This mirrors `kubectl` exactly: `kubectl get pods` and `kubectl get pod nginx` both use `get`, disambiguated by whether a specific name/argument is given, not by a different verb.
  - **`describe`** - rich, aggregated single-object detail, backed by resource templates. Not redundant with `get <group> <name>`: `get` returns the same lightweight format the many-result case uses, just filtered to one match; `describe` combines several reads into one report (e.g. a process's status + cmdline + limits) and deliberately excludes secret-shaped data (`environ`) by default.
  - **Mutation words** keep their own verb directly - `create`, `update`, `delete`, `restart`, `start`, `stop`, `enable`, `disable`. The last five are enum values from a tool's own schema (`services/manage`'s `action` field), not hardcoded in the client - a future sixth action works immediately with no client changes.
- **`[target-keyword]`** disambiguates which operation within the group, only needed when the group has more than one candidate under that verb (`network` has five different `get`-shaped operations, so `nslookup`/`ping`/`curl`/`arp`/`trace-path` must be named; `files` has exactly one bare-reachable read, so no keyword is needed for it).
- **`top`** is a target keyword, not a verb - `get processes top` runs a fixed client-side recipe (`cpu/load-average` + `memory/usage` + `processes/list`), the one deliberate hardcoded exception in an otherwise fully schema-driven client.

## Standalone commands (no `<group>`)

### `ping`
Pings the `mcpd` daemon to verify connectivity and authentication.

### `explain <group>`
Lists every tool, resource, and template `linuxctl` can reach in a group, with the exact verb/keyword to use for each - dynamically generated from the daemon's live schema, not a static help page.

```bash
$ linuxctl explain files
  get list                 -> tool files/list                Lists contents of a directory.
  get                      -> tool files/read                Precision reading of file contents ... (bare - no keyword needed)
  create                   -> tool files/create              Create a new file or replace file contents.
  update                   -> tool files/update              Programmatically edit a file ...
  get find                 -> tool files/find                Search for files in a directory hierarchy.
  get filetype             -> tool files/filetype            Determines a file's MIME type ...
  describe <name>          -> template file:///{path}         Reads any file on the system. ...
```

### `resource <uri>`
Reads the exact contents of an MCP resource directly by URI. Resource URIs always use `scheme://path` syntax, e.g. `os://uname`, `system://hostname`, `file:///etc/hosts` (note the triple slash - an empty authority followed by an absolute path).

### `tool <group>/<command> [--flag val ...]`
Calls a tool directly by its literal name, symmetric with `resource <uri>` - both call the underlying MCP method (`tools/call` vs `resources/read`) directly by its exact identifier, bypassing the verb/group grammar entirely. Also doubles as the pre-redesign flat-syntax escape hatch for anything the verb resolver doesn't yet cover (see `plan/linuxctl-redesign.md`'s "Open decisions").

```bash
linuxctl tool files/list --path /tmp
```

See [MCP Meta-Group](./mcp-meta-group) for `get mcp tools/resources/prompts/info`, and [Daemon User Administration](./mcpd-admin) for the local-only `mcpd` group.

## `<verb> <group> [target-keyword] [args]`

The main grammar. Arguments can be positional or explicit flags:

```bash
linuxctl get files /etc/hosts
linuxctl get files list /var/log --privileged true
```

Append `--output table`, `--output wide`, `--output yaml`, or `--output json` to format the result (this also sets the underlying tool's own `output_format` argument where it has one, since several tools return meaningfully different data in JSON mode, not just a different rendering of the same data).

For the complete list of every group's verbs and keywords with real output, see the [Full Command Reference](./command-reference).
