import re

with open("internal/tools/processes/get/processes/get_processes.go", "r") as f:
    content = f.read()

# E.g.
replacement = """
	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		b, _ := json.Marshal(processList)
		return string(b), nil
	}
"""

content = re.sub(
    r"\tvar sb strings\.Builder\n\tsb\.WriteString.*?\n\tfor _, p := range processList {",
    replacement + "\n\tvar sb strings.Builder\n\tsb.WriteString(fmt.Sprintf(\"%-8s %-10s %-8s %-8s %-8s %s\\\\n\", \"PID\", \"USER\", \"CPU%\", \"MEM%\", \"VSZ\", \"COMMAND\"))\n\tfor _, p := range processList {",
    content,
    flags=re.DOTALL
)

with open("internal/tools/processes/get/processes/get_processes.go", "w") as f:
    f.write(content)
