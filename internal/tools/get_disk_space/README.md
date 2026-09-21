# `get_disk_space`

This package implements the `get_disk_space` tool (equivalent to `df`). It uses `syscall.Statfs` to calculate available disk space and inodes without shelling out.
