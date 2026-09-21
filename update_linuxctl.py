import re

with open("cmd/linuxctl/main.go", "r") as f:
    content = f.read()

# 1. Add imports for formatting
if '"text/tabwriter"' not in content:
    content = content.replace(
        '"strings"\n\t"sync"',
        '"strings"\n\t"sync"\n\t"text/tabwriter"\n\t"gopkg.in/yaml.v3"'
    )

# 2. Add -o parsing
if 'key == "o" || key == "output"' not in content:
    content = content.replace(
        'key := strings.TrimPrefix(arg, "--")',
        'key := strings.TrimPrefix(arg, "--")\n\t\t\tkey = strings.TrimPrefix(key, "-")\n\t\t\tif key == "o" || key == "output" {\n\t\t\t\tkey = "output_format"\n\t\t\t}'
    )

# 3. Add formatting logic
format_logic = """
	if contentList, ok := result["content"].([]interface{}); ok {
		var outputFormat string
		if of, ok := toolArgs["output_format"].(string); ok {
			outputFormat = of
		}

		for _, c := range contentList {
			content := c.(map[string]interface{})
			if text, ok := content["text"].(string); ok {
				if outputFormat == "json" {
					// Prettify JSON if it is valid JSON
					var obj interface{}
					if err := json.Unmarshal([]byte(text), &obj); err == nil {
						b, _ := json.MarshalIndent(obj, "", "  ")
						fmt.Println(string(b))
					} else {
						fmt.Print(text)
					}
				} else if outputFormat == "yaml" {
					var obj interface{}
					if err := json.Unmarshal([]byte(text), &obj); err == nil {
						b, _ := yaml.Marshal(obj)
						fmt.Print(string(b))
					} else {
						fmt.Print(text)
					}
				} else if outputFormat == "table" || outputFormat == "wide" {
					// Basic dynamic table formatting for JSON arrays/objects
					var obj interface{}
					if err := json.Unmarshal([]byte(text), &obj); err == nil {
						w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
						
						if arr, ok := obj.([]interface{}); ok && len(arr) > 0 {
							// Array of objects
							if first, ok := arr[0].(map[string]interface{}); ok {
								var keys []string
								for k := range first {
									keys = append(keys, k)
								}
								fmt.Fprintln(w, strings.ToUpper(strings.Join(keys, "\\t")))
								for _, item := range arr {
									if m, ok := item.(map[string]interface{}); ok {
										var vals []string
										for _, k := range keys {
											vals = append(vals, fmt.Sprintf("%v", m[k]))
										}
										fmt.Fprintln(w, strings.Join(vals, "\\t"))
									}
								}
							} else {
								// Array of primitives
								for _, item := range arr {
									fmt.Fprintln(w, fmt.Sprintf("%v", item))
								}
							}
						} else if m, ok := obj.(map[string]interface{}); ok {
							// Single object
							for k, v := range m {
								fmt.Fprintf(w, "%s\\t%v\\n", strings.ToUpper(k), v)
							}
						}
						w.Flush()
					} else {
						fmt.Print(text)
					}
				} else {
					fmt.Print(text)
				}
			}
		}
"""
content = re.sub(
    r"\tif contentList, ok := result\[\"content\"\].*?}\n\t} else if result\[\"isError\"\] == true {",
    format_logic + "\t} else if result[\"isError\"] == true {",
    content,
    flags=re.DOTALL
)

with open("cmd/linuxctl/main.go", "w") as f:
    f.write(content)
