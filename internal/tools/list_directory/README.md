# `list_directory`

This package implements the `list_directory` tool. It uses Go's native `os.ReadDir` to securely list the contents of a directory. It is executed by the Ephemeral Worker under strict UID isolation.
