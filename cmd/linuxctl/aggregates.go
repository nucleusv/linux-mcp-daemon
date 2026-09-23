package main

import (
	"fmt"
	"strings"
)

// runDescribe implements the "describe" verb: rich, aggregated single-object
// detail. Most templates need only one resources/read; two (file, process)
// combine several reads into one report - see buildDescribeURIs.
func runDescribe(authToken string, tpl TemplateDef, positional []string, outputFormat string) {
	uris := buildDescribeURIs(tpl, positional)

	if len(uris) == 1 {
		respRPC := callMethod(authToken, nextID(), "resources/read", map[string]interface{}{"uri": uris[0]})
		renderResponse(respRPC, outputFormat, "contents")
		return
	}

	for _, uri := range uris {
		label := uri
		if idx := strings.LastIndex(uri, "/"); idx != -1 {
			label = uri[idx+1:]
		}
		fmt.Printf("=== %s ===\n", label)
		respRPC := callMethod(authToken, nextID(), "resources/read", map[string]interface{}{"uri": uri})
		if respRPC.Error != nil {
			fmt.Printf("(unavailable: %s)\n\n", respRPC.Error.Message)
			continue
		}
		renderResponse(respRPC, outputFormat, "contents")
		fmt.Println()
	}
}

// buildDescribeURIs returns the concrete resource URI(s) a describe call
// needs. Deliberately excludes process://{pid}/environ - see
// plan/linuxctl-redesign.md's verb rule: describe never leaks secret-shaped
// data by default.
func buildDescribeURIs(tpl TemplateDef, positional []string) []string {
	switch tpl.URITemplate {
	case "file:///{path}":
		base := fillTemplate(tpl.URITemplate, positional)
		return []string{base + "/stat", base + "/type"}

	case "process://{pid}/{target}":
		var pidArg []string
		if len(positional) > 0 {
			pidArg = positional[:1]
		}
		base := fillTemplate("process://{pid}/{target}", pidArg) // leaves "{target}" unfilled
		var uris []string
		for _, target := range []string{"status", "cmdline", "limits"} {
			uris = append(uris, strings.Replace(base, "{target}", target, 1))
		}
		return uris

	case "service://{name}/status":
		name := ""
		if len(positional) > 0 {
			name = positional[0]
		}
		if name != "" && !strings.Contains(name, ".") {
			name += ".service"
		}
		return []string{fillTemplate(tpl.URITemplate, []string{name})}

	default:
		return []string{fillTemplate(tpl.URITemplate, positional)}
	}
}
