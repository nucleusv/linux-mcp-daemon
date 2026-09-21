import os
import glob

# Find all tool READMEs
readmes = glob.glob("internal/tools/*/*/README.md")

for readme in readmes:
    parts = readme.split(os.sep)
    # internal/tools/<group>/<command>/README.md
    if len(parts) >= 5:
        group = parts[2]
        command = parts[3]
        correct_title = f"# {group}/{command}\n"
        
        with open(readme, "r") as f:
            lines = f.readlines()
        
        # Replace the first line if it's a title
        if lines and lines[0].startswith("# "):
            lines[0] = correct_title
            with open(readme, "w") as f:
                f.writelines(lines)
                print(f"Updated {readme} -> {correct_title.strip()}")
