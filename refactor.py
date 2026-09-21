import re
import os

with open("cmd/mcpd/rpc.go", "r") as f:
    lines = f.readlines()

new_lines = []
in_process = False

for line in lines:
    new_lines.append(line)

# Actually, doing this with a python script might be hard if we just blindly parse braces.
