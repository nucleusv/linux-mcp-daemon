import os

updates = {
    "internal/tools/disks/get/blocks/get_blocks.go": [
        (
            'return string(out), nil',
            'if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {\n\t\t// lsblk -J returns JSON natively\n\t\tcmd := exec.Command("lsblk", "-J")\n\t\toutJSON, err := cmd.CombinedOutput()\n\t\tif err != nil {\n\t\t\treturn "", fmt.Errorf("lsblk json error: %v, output: %s", err, string(outJSON))\n\t\t}\n\t\treturn string(outJSON), nil\n\t}\n\treturn string(out), nil'
        )
    ],
    "internal/tools/disks/get/usage/get_usage.go": [
        (
            'return fmt.Sprintf("Total size of %s: %s\\n", args.Path, formatBytes(totalBytes)), nil',
            'if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {\n\t\tdata := map[string]interface{}{\n\t\t\t"path": args.Path,\n\t\t\t"total_bytes": totalBytes,\n\t\t\t"human_readable": formatBytes(totalBytes),\n\t\t}\n\t\timport_json, _ := __import__("encoding/json").Marshal(data)\n\t\t// Wait, I cannot use __import__ in Go string. I will just rely on encoding/json being imported.\n\t\tb, _ := json.Marshal(data)\n\t\treturn string(b), nil\n\t}\n\treturn fmt.Sprintf("Total size of %s: %s\\n", args.Path, formatBytes(totalBytes)), nil'
        ),
        (
            'return fmt.Sprintf("Total size of %s: %d bytes\\n", args.Path, totalBytes), nil',
            'if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {\n\t\tdata := map[string]interface{}{\n\t\t\t"path": args.Path,\n\t\t\t"total_bytes": totalBytes,\n\t\t}\n\t\tb, _ := json.Marshal(data)\n\t\treturn string(b), nil\n\t}\n\treturn fmt.Sprintf("Total size of %s: %d bytes\\n", args.Path, totalBytes), nil'
        )
    ],
    "internal/tools/cpu/get/load_average/get_load_average.go": [
        (
            'return string(b), nil',
            'if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {\n\t\tparts := strings.Fields(string(b))\n\t\tif len(parts) >= 3 {\n\t\t\tdata := map[string]interface{}{\n\t\t\t\t"1_min": parts[0],\n\t\t\t\t"5_min": parts[1],\n\t\t\t\t"15_min": parts[2],\n\t\t\t}\n\t\t\tjsonB, _ := json.Marshal(data)\n\t\t\treturn string(jsonB), nil\n\t\t}\n\t}\n\treturn string(b), nil'
        )
    ],
    "internal/tools/system/get/os_release/get_os_release.go": [
        (
            'return sb.String(), nil',
            'if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {\n\t\tb, _ := json.Marshal(releaseInfo)\n\t\treturn string(b), nil\n\t}\n\treturn sb.String(), nil'
        )
    ],
    "internal/tools/network/get/interfaces/get_interfaces.go": [
        (
            'return sb.String(), nil',
            'if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {\n\t\tb, _ := json.Marshal(results)\n\t\treturn string(b), nil\n\t}\n\treturn sb.String(), nil'
        )
    ],
    "internal/tools/processes/delete/process/delete_process.go": [
        (
            'return fmt.Sprintf("Successfully sent signal %d to PID %d", args.Signal, args.PID), nil',
            'if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {\n\t\tdata := map[string]interface{}{\n\t\t\t"success": true,\n\t\t\t"pid": args.PID,\n\t\t\t"signal": args.Signal,\n\t\t}\n\t\tb, _ := json.Marshal(data)\n\t\treturn string(b), nil\n\t}\n\treturn fmt.Sprintf("Successfully sent signal %d to PID %d", args.Signal, args.PID), nil'
        )
    ]
}

for file_path, replacements in updates.items():
    if not os.path.exists(file_path):
        print(f"Skipping {file_path}")
        continue
    with open(file_path, "r") as f:
        content = f.read()
    
    # ensure encoding/json and strings are imported if used
    if '"encoding/json"' not in content:
        content = content.replace('import (\n', 'import (\n\t"encoding/json"\n')
    
    for old, new in replacements:
        content = content.replace(old, new)
        
    with open(file_path, "w") as f:
        f.write(content)
