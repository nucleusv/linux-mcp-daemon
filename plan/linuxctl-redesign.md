# linuxctl Redesign

Converged design for a from-scratch rework of `cmd/linuxctl/main.go`'s command grammar. Not yet implemented - this document is the spec to review before touching code, per this project's usual plan-first cadence.

## The problem with today's grammar

Today's `linuxctl <group>/<command> [--flag val ...]` is a flat, literal passthrough of mcpd's internal `<group>/<command>` tool-naming convention (see `CLAUDE.md`). It works, but every group invents its own command vocabulary with no shared pattern (`disks/free` vs `files/list` vs `network/trace-path`), there's no consistent way to get a single object's full detail (resources are bolted on as a separate special-cased `linuxctl resource <uri>` path), and nothing is discoverable by analogy - you have to already know a command exists to find it.

## Core grammar

```
linuxctl <verb> <group> [target-keyword] [name/args]
```

- **`<group>`** is always present, always the literal, unchanging `tools_group` string (`files`, `disks`, `processes`, `network`, `devices`, `kernel`, `logs`, `system`, `users`, `cpu`, `memory`, `auth`) - never abbreviated, never singularized, never dropped even when it feels implied. Dropping it was tried and rejected during design (`linuxctl restart nginx` - "restart what?" - fails the moment the tool surface grows past one restartable domain). **Note: there is no top-level `services` group** - `services/list` and `services/manage`'s real `tools_group` is `system` (confirmed against the live schema, not assumed), so services are reached as a `system` sub-target (`system services`), same as `packages`/`hostname`.
- **`<verb>`** is always one of a small universal set for reads (`get`, `list`), plus the tool's own specific action word for mutations. See the rule below - this is the part that took several rounds to converge on correctly.
- **`[target-keyword]`** disambiguates *which* operation within the group, only when the group has more than one candidate under that verb (`network` has five different `get`-shaped operations, so `nslookup`/`ping`/`curl`/`arp`/`trace-path` must be named; `files` has exactly one `get`-shaped operation, so no keyword is needed there).

### The verb rule (converged after correcting an earlier draft)

**Every read operation - tool-backed or resource-backed, doesn't matter which - is always prefixed with the universal verb (`get` for one result, `list` for many). The tool's own specific name becomes a target keyword, never the leading word.** An earlier draft of this document used the tool's own name directly as the leading verb (`linuxctl nslookup network example.com`, `linuxctl free disks /`) - this was corrected because it reads backwards: "nslookup" already means "look up a name," so putting the group word between the verb and its real target made commands read like nonsense. Wrapping every read in `get`/`list` fixes this and gives exactly two words to remember for the entire read-side surface, regardless of how many tools exist behind them.

**Mutations keep their own word directly, no wrapper**: `create`, `update`, `delete`, `restart`, `start`, `stop`, `enable`, `disable`. These don't get wrapped because there's no sensible generic verb to wrap them in - "create" and "delete" already are the generic words, and the enum-derived action verbs (`restart`/`start`/`stop`/`enable`/`disable`) are unambiguous actions with no broader category to fold into.

**`describe` and `top` are their own universal verbs**, same status as `get`/`list`, not wrapped further:
- **`describe`** - rich, aggregated single-object detail, backed by resource templates. Where the underlying data spans multiple targets (e.g. `process://{pid}/{status,cmdline,limits}`), the client makes multiple resource reads and renders one combined report. **Deliberately excludes `environ`/other secret-shaped data by default** - same instinct as this project's access-logging work earlier this session (don't leak secrets into output by default; require an explicit flag).
- **`top`** - the one deliberate, hardcoded exception to "everything is schema-derived": no tool is named `top`; it's a fixed client recipe combining `cpu/load-average` + `memory/usage` + `processes/list`. Filed under `processes` (`linuxctl top processes`), not top-level, since the subject is unambiguously process resource consumption.
- **Enum-value-as-verb** - values of an enum-typed parameter become their own mutation verbs generically. `services/manage`'s `action` enum (`start`/`stop`/`restart`/`reload`/`enable`/`disable`) makes `linuxctl restart system services nginx` resolve by scanning tool schemas in the `system` group for an enum parameter containing `restart`, not by hardcoding "restart" anywhere. A future fifth action added server-side works immediately with zero client changes.

### Why not just mirror kubectl, or mirror real Unix utility names

Two approaches were explored and rejected in favor of this one, both for the same underlying reason: `linuxctl` talks to `mcpd`'s JSON-RPC API (tools, resources, resource templates) - it never touches a Linux binary or the Kubernetes API directly. Naming decisions based on "what would kubectl call this" or "what does the real Unix tool call this, and do we happen to wrap that exact binary" are both answering the wrong question. `services/manage` doesn't wrap `systemctl` (`ARCHITECTURE.md` confirms it talks to `go-systemd/dbus` directly) - calling it `linuxctl systemctl` would be a false claim about the implementation. The real utilities were useful purely as a *brainstorming aid* for discovering which domains and verbs matter to an admin, not as a naming template to copy.

### True exceptions - no group at all

Only two, each for a distinct, principled reason:
- **`explain <group> <verb>`** - a meta-verb *about* the schema itself, not about a data domain. `kubectl explain` has the identical shape (never nested under a resource type).
- **`config`** - pure client-local state (server URL, bearer token). Has no corresponding server-side tool or resource at all, so it structurally cannot fit the `<group> <verb>` resolver no matter how the resolver is designed.

## Required server-side schema additions

Small, additive, non-breaking changes (extra JSON fields any existing MCP client, including Claude itself, will simply ignore). **The `linuxctl_verb` field is implemented as of this session** - it holds the tool's own specific name (used as the target keyword the client assembles around, per the rule above, not as a standalone leading verb):

1. **`"linuxctl_verb"` field on every tool schema entry** in `internal/rpc/tools.go` - done. Also corrected a real, pre-existing bug found while adding it: `services/manage` and `services/list` had `tools_group: "system"` - initially mistaken for a bug and "fixed" to `"services"` mid-session, then correctly reverted once the `system services`-as-target-keyword pattern was confirmed. The live schema's `tools_group: "system"` for both was correct all along.
2. **Add a `"group"` field to every resource and resource template** in `internal/rpc/resources.go` - not yet done. Mirrors `tools_group`; needed because `service://{name}/status` has no way to declare "I belong to the `system` group" today.
3. **Convert free-text-only enumerations to real JSON Schema `"enum"` arrays** - not yet done. `services/manage`'s `action` and `logs/logins`'s `type` currently only list valid values in the English `description` string. Worth doing independent of `linuxctl` - it's the exact same problem for me (Claude) calling these tools today; I'm parsing prose to guess valid values instead of reading a real constraint.

## The resolver algorithm

```go
func Resolve(reg Registry, verb, group string, rest []string) (Action, error) {
    // Case A: verb is a genuine mutation word - matched directly against a
    // tool's own linuxctl_verb (create/update/delete/manage) or, failing
    // that, against an enum value inside some tool's parameters in this
    // group (restart/start/stop/enable/disable -> services/manage's
    // `action` enum). Fully generic - a future fifth enum value works with
    // zero client changes.
    if tool, ok := reg.ToolsByGroupVerb[group][verb]; ok {
        return buildToolCall(tool, rest), nil
    }
    for _, tool := range reg.ToolsByGroupVerb[group] {
        if param, ok := findEnumParamContaining(tool.InputSchema, verb); ok {
            return buildToolCall(tool, rest, withPreset(param, verb)), nil
        }
    }

    // Case B: verb is "get"/"list" - universal read wrappers. rest[0], if
    // present, is the target keyword disambiguating which operation; if the
    // group has only one candidate under this verb, the keyword is optional.
    if verb == "get" || verb == "list" {
        if tool, ok := matchToolByLinuxctlVerb(reg, group, rest); ok {
            return buildToolCall(tool, remainingArgs(rest)), nil
        }
        if res, ok := findResourceByTarget(reg, group, rest); ok {
            return buildResourceRead(res, rest), nil
        }
    }

    // Case C: describe - resource template aggregation.
    if verb == "describe" {
        if tpl, ok := findTemplateByTarget(reg, group, rest); ok {
            return buildTemplateRead(tpl, rest), nil
        }
    }

    // Case D: the one hardcoded exception.
    if verb == "top" && group == "processes" {
        return buildTopSnapshot(reg), nil
    }

    return nil, fmt.Errorf("no verb %q in group %q - try: linuxctl explain %s", verb, group, group)
}
```

`Registry` is built once per invocation from live `tools/list` + `resources/list` + `resources/templates/list` responses - no static per-tool table anywhere. Adding a new tool server-side (as this session did repeatedly with `disks/mounts`, `system/packages`, `users/list`) makes it automatically usable via `linuxctl <group> <verb>` with zero client-side code changes, the same way I (Claude) could call a brand-new tool the moment it existed without anyone teaching me a new verb.

## Full command reference

Every tool, resource, and resource template that exists today (33 tools, 11 static resources, 6 templates), mapped one-for-one. This is the acceptance criteria for the redesign.

```
linuxctl list files /var/log
linuxctl get files /etc/hosts
linuxctl create files /tmp/x --content ".."
linuxctl update files /etc/nginx/nginx.conf
linuxctl get files find /var/log --name "*.log"
linuxctl get files filetype /usr/bin/python3
linuxctl describe files /etc/hosts

linuxctl list disks
linuxctl get disks free /
linuxctl get disks usage /var/log
linuxctl list disks mounts
linuxctl get disks health sda
linuxctl list disks partitions vda
linuxctl list disks performance          # all devices (many results)
linuxctl get disks performance vda       # one device (one result)
linuxctl describe disks sda

linuxctl list processes --sort_by mem
linuxctl top processes
linuxctl describe processes 1234
linuxctl delete processes 1234

linuxctl list system services
linuxctl describe system services nginx
linuxctl restart system services nginx
linuxctl start system services nginx
linuxctl stop system services nginx
linuxctl enable system services nginx
linuxctl disable system services nginx
linuxctl get logs journal --unit nginx.service

linuxctl list network connections --state listening
linuxctl get network ping 8.8.8.8
linuxctl get network curl https://example.com
linuxctl get network nslookup example.com
linuxctl list network arp
linuxctl get network trace-path 8.8.8.8
linuxctl list network interfaces
linuxctl list network routes
linuxctl describe network interfaces eth0

linuxctl list devices usb
linuxctl list devices pci
linuxctl list devices dmi

linuxctl get kernel sysctl net.ipv4.ip_forward
linuxctl update kernel sysctl net.ipv4.ip_forward 1
linuxctl list kernel modules

linuxctl get logs dmesg
linuxctl list logs logins --type failed
linuxctl list logs journal --since "1 hour ago"

linuxctl get system hostname
linuxctl get system timezone
linuxctl get system locale
linuxctl get system release
linuxctl get system uname
linuxctl list system packages

linuxctl list users --min_uid 1000
linuxctl list cpu
linuxctl get cpu load-average
linuxctl get memory usage
linuxctl get auth sudo-rules
```

## Open decisions not yet settled

1. **Backward compatibility.** Should the old `<group>/<command>` positional syntax be kept as an explicit escape hatch during transition (mirroring `kubectl get --raw /api/v1/...`, e.g. `linuxctl raw disks/free --path /`), or fully replaced? Recommendation: keep a `raw` escape hatch, not the old syntax as a parallel first-class path - guarantees coverage of anything the resolver doesn't yet handle without fragmenting the UX into two competing styles.
2. **`config` subcommand design** - `set-token`/`set-server` mechanics (where the config file lives, `~/.linuxctl/config` vs env vars vs both) not yet specified in detail.
3. **`-w`/`--watch` polling** - deliberately not building this into `linuxctl` itself; the real Unix `watch` utility already does this generically (`watch linuxctl list processes`), matching this whole design's "don't reimplement what already exists" principle.
4. **`stdio` mode** - today's `linuxctl`'s existing MCP-stdio-transport bridge (for Claude Desktop-style clients) is unrelated to this verb/group redesign and should be preserved unchanged; it's a different transport-bridging concern, not part of the human-facing command grammar.
5. **`--output`/`-o` flag** (`json`/`yaml`/`table`/`wide`) - unchanged from today's implementation, not part of this redesign's scope.
6. **`get`/`list` choice per tool** - a handful of calls (`disks/usage`, `disks/performance`, `logs/dmesg`) could reasonably be argued either way (single summary vs. an array under the hood); the table above picks one consistently but this is a judgment call worth a second look once real usage shows which reads better.
