# network/connections

Lists TCP and UDP sockets in every state with their owning processes - `ss -tuanp`'s view - read natively from `/proc/net/{tcp,tcp6,udp,udp6}` (parsed in `internal/kernel/netstat.go`) and `/proc/<pid>/fd` (socket inode to process), without running `ss`.

## Parameters
- `state` (string, optional): `LISTEN`/`listening` (TCP listeners and unconnected UDP sockets), `ESTABLISHED`, `TIME_WAIT`, `CLOSE_WAIT`, `SYN_SENT`, ... in ss or kernel spelling, case-insensitive; or ss's groups `connected`, `synchronized`, `all`.
- `port` (int, optional): sockets whose local or peer port is this.
- `output_format` (string, optional): `json`, `yaml`, `table` or `wide` return a JSON array of sockets (`netid`, `state`, `recv_q`, `send_q`, `local_address`, `local_port`, `peer_address`, `peer_port`, `uid`, `inode`, `processes`); otherwise ss-style text.
- `privileged` (bool, optional): run as root, so every socket's owning process is shown - unprivileged, only the calling user's own processes are.

## Differences from ss

`connections_test.go` compares the socket list with `ss -tuan` where `ss` is installed (identical on Ubuntu 24.04), except two things `/proc/net` doesn't carry - only netlink's `sock_diag` does:
- ss prints a dual-stack IPv6 wildcard socket (not `IPV6_V6ONLY`) as `*:port`; this prints its real address, `[::]:port`.
- ss appends `%iface` for sockets bound to a device (`SO_BINDTODEVICE`), e.g. `127.0.0.53%lo:53`; this prints `127.0.0.53:53`.

For a listening TCP socket, `Send-Q` is what `/proc` reports (0), not the backlog limit ss shows.
