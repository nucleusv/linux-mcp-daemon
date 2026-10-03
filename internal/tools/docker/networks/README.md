# docker/networks

Lists Docker networks over the Engine API (`GET /networks`): driver, scope, subnet, gateway, the flags that are set, and the containers attached to each with their addresses.

Read-only by construction, not by a flag — there is no code path here that creates, removes, connects or disconnects. Removing an unused network is `docker/prune`'s job; a single container's membership is already in `docker-container://<name>/inspect`.

`GET /networks` returns an empty `Containers` map however many containers are attached — only an inspect fills it — so the attached containers come from the container list, the same detour `docker/volumes` makes to find a volume's users. Stopped containers are listed too, without an address.

## Parameters
- `pattern` (string, optional): only networks whose name matches this glob.
- `driver` (string, optional): only networks on this driver (`bridge`, `host`, `none`, `overlay`, `macvlan`, …), matched case-insensitively.
- `output_format` (string, optional): `json`, `yaml`, `table`, `wide`; default text.

Text output prints only the flags that are set — `internal`, `attachable`, `ingress`, `ipv6` — because `Internal: false` on every network on a host is noise. The JSON form has them all as booleans, plus IPAM options and labels.

## Usage & Permissions
⚠ Root or nothing: needs `docker/networks: {allowed: true}` in the caller's grant in `configs/mcp-sudo.yaml`. The Docker socket is root-owned and all-or-nothing, so read-only is no reason to fall back to the caller's uid — a uid outside the `docker` group cannot open the socket at all. Being granted at all means an agent can see every network on the host, including its subnets and every attached container's address.

No `containers:` list: this tool names no container, so one on the grant is a config error (see `internal/config/sudo.go`).

`linuxctl get docker networks` calls this tool. One network's full IPAM, options and per-container MAC addresses are `docker-network://<name>/inspect` — **not** `network://`, which is already the host's own networking (`network://interfaces`, `network://routes`). That asymmetry with `docker-container://`/`docker-image://`/`docker-volume://` is deliberate.
