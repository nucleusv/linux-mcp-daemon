package main

import (
	"fmt"
	"os"
	"strings"
)

// runDescribe implements the "describe" verb: rich, aggregated single-object
// detail. Most templates need only one resources/read; two (file, process)
// combine several reads into one report - see buildDescribeURIs.
func runDescribe(authToken string, tpl TemplateDef, positional []string, outputFormat string) {
	uris, extra := buildDescribeURIs(tpl, positional)
	if len(extra) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: %d extra argument(s) ignored: %s\n", len(extra), strings.Join(extra, " "))
	}

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
// data by default. The second result is the positional words nothing consumed.
func buildDescribeURIs(tpl TemplateDef, positional []string) ([]string, []string) {
	switch tpl.URITemplate {
	case "file:///{path}":
		base, extra := fillTemplate(tpl.URITemplate, positional)
		return []string{base + "/stat", base + "/type"}, extra

	case "process://{pid}/{target}":
		var pidArg, extra []string
		if len(positional) > 0 {
			pidArg, extra = positional[:1], positional[1:]
		}
		base, _ := fillTemplate("process://{pid}/{target}", pidArg) // leaves "{target}" unfilled
		var uris []string
		for _, target := range []string{"status", "cmdline", "limits"} {
			uris = append(uris, strings.Replace(base, "{target}", target, 1))
		}
		return uris, extra

	case "crontab://{user}/{view}":
		// One report: the metadata (with the hash `update --if_match` needs),
		// then the raw crontab.
		name := ""
		if len(positional) > 0 {
			name = positional[0]
		}
		var extra []string
		if len(positional) > 1 {
			extra = positional[1:]
		}
		return []string{"crontab://" + name + "/info", "crontab://" + name + "/text"}, extra

	case "docker-container://{name}/{view}":
		// Two placeholders, usually one argument: default to the computed
		// summary, which is what "describe" means everywhere else.
		name, view := "", "status"
		if len(positional) > 0 {
			name = positional[0]
		}
		if len(positional) > 1 {
			view = positional[1]
		}
		uri, _ := fillTemplate(tpl.URITemplate, []string{name, view})
		var extra []string
		if len(positional) > 2 {
			extra = positional[2:]
		}
		return []string{uri}, extra

	case "service://{name}/status":
		name := ""
		if len(positional) > 0 {
			name = positional[0]
		}
		if name != "" && !strings.Contains(name, ".") {
			name += ".service"
		}
		uri, _ := fillTemplate(tpl.URITemplate, []string{name})
		var extra []string
		if len(positional) > 1 {
			extra = positional[1:]
		}
		return []string{uri}, extra

	default:
		uri, extra := fillTemplate(tpl.URITemplate, positional)
		return []string{uri}, extra
	}
}
