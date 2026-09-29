# services/status (internal worker)

The worker behind `service://{name}/status`: the properties of one systemd unit as JSON - `name`, `description`, `load_state`, `active_state`, `sub_state`, `fragment_path` - read over DBus (`go-systemd`, no `systemctl`). Not a tool an agent can call; the `service://` template handler runs it.

It uses the private systemd socket as root and the system bus otherwise, which any user may read on a host with systemd; a host without it gets a named connection error. To list units use `services/list`, for timers `timers/list`.
