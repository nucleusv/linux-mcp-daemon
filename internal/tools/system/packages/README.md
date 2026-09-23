# system/packages

Lists installed packages by natively parsing the host's package database - `/var/lib/dpkg/status` on Debian/Ubuntu, `/lib/apk/db/installed` on Alpine. RPM-based systems (Fedora/RHEL/CentOS) aren't supported yet: their package database isn't a plain-text format safe to hand-parse without a real RPM parser or wrapping `rpm`/`dnf`, so the tool returns a clear error naming the detected distro instead of guessing.

## Host vs. container filesystem
By default this reports on whatever filesystem the worker process sees. When `mcpd` runs containerized (`worker.containerized: true` in `configs/daemon.yaml` - see `internal/worker/hostns.go`), passing `privileged: true` automatically also joins the host's real mount namespace before reading the package database, so paths like `/var/lib/dpkg/status` resolve to the host's real file rather than this daemon's own container image. There's no separate flag to request this - it follows from `privileged: true` plus how the daemon itself is deployed.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `name` (string, optional): Only packages whose name matches this glob or exact name (`openssh-*`, `*ssl*`, `curl`).
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload (array of `{name, version, architecture}`). The default text output is the list itself - a count line, then `NAME VERSION ARCH` columns (it used to be only the count). Packages removed with only their config files left (`deinstall ... config-files`) are not listed, matching `dpkg -l`'s `ii` entries.
- `privileged` (boolean, optional): Run the worker as root (must be authorized in `mcp-sudo.yaml`). See "Host vs. container filesystem" above for what this means when `mcpd` runs containerized.
