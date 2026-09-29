# service resource template

Handler for `service://{name}/status`: the state of one systemd unit as JSON, through the `services/status` worker over DBus (no `systemctl`).

```text
$ linuxctl describe system cron --privileged true
{ "name": "cron.service", "description": "Regular background program processing daemon",
  "load_state": "loaded", "active_state": "active", "sub_state": "running",
  "fragment_path": "/usr/lib/systemd/system/cron.service" }
```

The name is the unit, `.service` is added when it is missing. To list units use the `services/list` tool (services) or `timers/list` (timers); to start or stop one use `services/manage`.

## Permissions
The worker runs as the caller and reads the system bus, which any user may read on a host with systemd. Inside a container that bus does not exist, so the read needs root through the user's `resources:` grant for `service://`, which joins the host. A host without systemd gets a named connection error.
