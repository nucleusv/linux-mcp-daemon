# files/stat (internal worker)

The worker behind `file:///{path}/stat`: metadata of one file or directory as JSON - `name`, `size` (bytes), `mode`, `modified_time`, `is_dir`. It is not a tool an agent can call; the `file://` template handler runs it. `stat` follows a symlink and describes its target, unless the daemon sets `_no_follow` (a narrow `resources:` grant), in which case the exact inode is described and a path through a symlink is refused.

For a directory listing use the `files/list` tool; for the MIME type, `file:///{path}/type` (worker `files/filetype`).
