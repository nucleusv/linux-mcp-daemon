# file resource template

Handler for `file:///{path}`: reads any file, its metadata or its MIME type through the same workers the `files/*` tools use.

| URI | Returns | Worker |
|---|---|---|
| `file:///etc/hosts` or `file:///etc/hosts/content` | the file's contents as text | `files/content` |
| `file:///etc/hosts/stat` | metadata as JSON (size, mode, owner, times, symlink target) | `files/stat` |
| `file:///etc/hosts/type` | the MIME type, detected from the first bytes | `files/filetype` |

A bare path has no suffix and means `/content`. The path is cleaned (`..` resolved) before both the permission check and the read, so the worker reads exactly the path that was authorized.

## Permissions
Unprivileged by default: the worker runs as the caller's OS account and the kernel decides. A read runs as root only when the user's `resources:` grant for `file://` lists a path that covers it (`resources: {"file://": ["/var/log"]}`; an empty prefix `[""]` grants the whole scheme); with a grant narrower than the whole filesystem the worker does not follow a symlink out of the allowed tree. A `.json` file is served as `application/json`, everything else as `text/plain`.

`linuxctl`: `get files /etc/hosts`, `describe files /etc/hosts` (stat and type together).
