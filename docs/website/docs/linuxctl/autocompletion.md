---
sidebar_label: 'Autocompletion (bash/zsh)'
sidebar_position: 6
---
# Shell Autocompletion (bash & zsh)

`linuxctl completion <bash|zsh>` prints a completion script for your shell. Once loaded, `Tab` completes every part of a command:

| Position | Example | Completes |
|---|---|---|
| Verb | `linuxctl <Tab>` | `get`, `describe`, `explain`, `tool`, `resource`, `create`, `update`, `delete`, action verbs like `restart`/`stop`, ... |
| Group | `linuxctl get <Tab>` | `cpu`, `disks`, `files`, `network`, `system`, ..., plus `mcp-api` |
| Target keyword | `linuxctl get disks <Tab>` | `free`, `health`, `mounts`, `partitions`, `performance`, `usage` |
| Flags | `linuxctl get disks --<Tab>` | the resolved tool's own parameters (`--all`, `--privileged`, ...) plus `--output`/`-o` |
| Output format | `linuxctl get processes -o <Tab>` | `json`, `yaml`, `table`, `wide` |
| Tool name | `linuxctl tool <Tab>` | `disks/list`, `files/read`, ... |
| Resource URI | `linuxctl resource <Tab>` | `network://interfaces`, `os://uname`, `process://{pid}/{target}`, ... |
| Catalog | `linuxctl get mcp-api <Tab>` | `tools`, `resources`, `prompts`, `info` |

Candidates come from the daemon's **live registry** (`tools/list`, `resources/list`, `resources/templates/list`) - the same data the [command resolver](./grammar) uses - so a tool added server-side completes immediately, with no client update. Each user only sees completions for the tools their token can list.

## Prerequisites

Completion runs `linuxctl` itself on every `Tab`, so:

1. **`linuxctl` must be on your `PATH`** (or invoked by path, e.g. `./executables/linuxctl`). Build it with `scripts/build-cli.sh`, which writes `executables/linuxctl`.
2. **Point it at your daemon with environment variables, not an alias.** Completion can't see through a shell alias like `alias linuxctl="linuxctl -server ..."` - use `MCP_SERVER` (default for `-server`) and `MCP_TOKEN` instead.

```bash
export PATH="$HOME/path/to/linux-mcp-daemon/executables:$PATH"
export MCP_SERVER="http://my-host:9091"
export MCP_TOKEN="your_token_here"
```

:::tip Keep the token out of your shell rc file
Rather than hard-coding the token in `~/.bashrc`/`~/.zshrc` (often world-readable), store it in a private file and read it at startup:

```bash
mkdir -p ~/.config/linuxctl && chmod 700 ~/.config/linuxctl
umask 077; printf '%s' 'your_token_here' > ~/.config/linuxctl/token
```

```bash
# in ~/.bashrc or ~/.zshrc
[[ -r ~/.config/linuxctl/token ]] && export MCP_TOKEN="$(<~/.config/linuxctl/token)"
```
:::

## bash

Add to `~/.bashrc` (or `~/.bash_profile` on macOS):

```bash
source <(linuxctl completion bash)
```

Install the **bash-completion** package too (`apt install bash-completion`, `brew install bash-completion@2`) - without it, bash splits words on `:`, so resource URIs like `network://interfaces` won't complete as one word. Everything else works without it.

## zsh

Add to `~/.zshrc`, **after** `compinit` if your config already runs it (e.g. via oh-my-zsh):

```zsh
source <(linuxctl completion zsh)
```

The script calls `compinit` itself if it hasn't been loaded yet.

### Full zsh example

```zsh
# linuxctl (linux-mcp-daemon CLI)
export PATH="$HOME/path/to/linux-mcp-daemon/executables:$PATH"
export MCP_SERVER='http://my-host:9091'
[[ -r ~/.config/linuxctl/token ]] && export MCP_TOKEN="$(<~/.config/linuxctl/token)"
unalias linuxctl 2>/dev/null
command -v linuxctl >/dev/null && source <(linuxctl completion zsh)
```

Reload with `exec zsh` (or `exec bash`), then try `linuxctl get di<Tab>`.

## How it works

The completion script calls a hidden command, passing the words typed so far:

```bash
linuxctl __complete -- get disks ""
# free
# health
# mounts
# ...
```

You can run this directly to debug what completion will offer.

- **Caching:** the registry is cached for 5 minutes per server+token, under the user cache directory (`~/Library/Caches/linuxctl/` on macOS, `~/.cache/linuxctl/` on Linux), so most `Tab` presses don't touch the network. Delete that directory to force a refresh after a server-side change.
- **Offline / slow daemon:** if the daemon doesn't answer within 3 seconds (or no token is set), completion falls back to the candidates it can offer statically - output formats, `completion bash|zsh`, `mcp-api` catalogs - rather than hanging the shell.
- **Global flags** typed before the verb (`linuxctl -server http://other:9091 get <Tab>`) are honored, so completion queries the same daemon the command will.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| No completions at all | `linuxctl` isn't on `PATH`, or the script wasn't sourced - check `type linuxctl` and (zsh) `echo $_comps[linuxctl]`. |
| Only `json yaml table wide`-style static candidates | No `MCP_TOKEN`, or the daemon is unreachable - test with `linuxctl ping`. |
| New server-side tool doesn't complete | Stale 5-minute cache - `rm -rf ~/Library/Caches/linuxctl` (macOS) or `~/.cache/linuxctl` (Linux). |
| `network://interfaces` splits at `:` in bash | Install bash-completion (see above). |
| Worked with an alias before, not now | Replace the alias with `MCP_SERVER` - completion bypasses aliases. |
