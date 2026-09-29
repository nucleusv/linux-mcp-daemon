# files/content (internal worker)

The worker behind `file:///{path}` and `file:///{path}/content`: it reads one file's contents as text. It is not a tool an agent can call (it is absent from `tools/list`); the `file://` template handler runs it.

The read is capped: past 10 KB the output ends with `[WARNING: File truncated at 10KB context limit]`. For a byte or line range, or a larger read, use the `files/read` tool. Arguments: `path` (absolute), and `_no_follow`, which the daemon sets when a `resources:` grant narrower than the whole filesystem makes the worker refuse to follow a symlink out of the allowed tree.

Permissions are those of the `file://` template: the caller's own account by default, root only where the user's `resources:` grant for `file://` covers the path.
