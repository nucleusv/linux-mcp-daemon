import os
import re

with open("cmd/mcpd/main.go", "r") as f:
    lines = f.readlines()

# Extract header (imports)
import_end = 0
for i, line in enumerate(lines):
    if line.strip() == ")":
        import_end = i + 1
        break

header = "".join(lines[:import_end]) + "\n"

# types.go: 47 to 146
with open("cmd/mcpd/types.go", "w") as f:
    f.write(header + "".join(lines[46:146]))

# main.go: 147 to 264
with open("cmd/mcpd/main.go", "w") as f:
    f.write(header + "".join(lines[146:264]))

# http.go: 265 to 386
with open("cmd/mcpd/http.go", "w") as f:
    f.write(header + "".join(lines[264:386]))

# rpc.go: 387 to EOF
with open("cmd/mcpd/rpc.go", "w") as f:
    f.write(header + "".join(lines[386:]))
