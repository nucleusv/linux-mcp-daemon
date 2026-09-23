package main

// Implements the verb/group grammar from plan/linuxctl-redesign.md:
//
//   linuxctl <verb> <group> [target-keyword] [name/args]
//
// The registry is built fresh from live tools/list, resources/list, and
// resources/templates/list responses on every invocation - there is no
// static per-tool table in this client. A new tool or resource added
// server-side (tools_group + linuxctl_verb, or group + linuxctl_verb for
// resources) is automatically reachable with zero client changes, the same
// way an MCP-speaking AI client can call it the moment it exists.

import (
	"fmt"
	"strconv"
	"strings"
)

type ToolDef struct {
	Name         string
	ToolsGroup   string
	LinuxctlVerb string
	Description  string
	InputSchema  map[string]interface{}
}

type ResourceDef struct {
	URI          string
	Name         string
	Group        string
	LinuxctlVerb string
	Description  string
}

type TemplateDef struct {
	URITemplate  string
	Name         string
	Group        string
	LinuxctlVerb string
	Description  string
}

type Registry struct {
	Tools     []ToolDef
	Resources []ResourceDef
	Templates []TemplateDef
}

// Action is what Resolve produces - main.go executes it without needing to
// know which of these three shapes it came from.
type Action struct {
	Kind        string // "tool_call", "resource_read", "template_read", "top_snapshot"
	Tool        ToolDef
	ResourceURI string
	Template    TemplateDef
	Args        map[string]interface{}
	Positional  []string // leftover positional args after keyword/preset consumption
}

func str(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func BuildRegistry(toolsRaw, resourcesRaw, templatesRaw []interface{}) Registry {
	var reg Registry
	for _, t := range toolsRaw {
		m, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		schema, _ := m["inputSchema"].(map[string]interface{})
		reg.Tools = append(reg.Tools, ToolDef{
			Name:         str(m, "name"),
			ToolsGroup:   str(m, "tools_group"),
			LinuxctlVerb: str(m, "linuxctl_verb"),
			Description:  str(m, "description"),
			InputSchema:  schema,
		})
	}
	for _, r := range resourcesRaw {
		m, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		reg.Resources = append(reg.Resources, ResourceDef{
			URI:          str(m, "uri"),
			Name:         str(m, "name"),
			Group:        str(m, "group"),
			LinuxctlVerb: str(m, "linuxctl_verb"),
			Description:  str(m, "description"),
		})
	}
	for _, tpl := range templatesRaw {
		m, ok := tpl.(map[string]interface{})
		if !ok {
			continue
		}
		reg.Templates = append(reg.Templates, TemplateDef{
			URITemplate:  str(m, "uriTemplate"),
			Name:         str(m, "name"),
			Group:        str(m, "group"),
			LinuxctlVerb: str(m, "linuxctl_verb"),
			Description:  str(m, "description"),
		})
	}
	return reg
}

// findEnumParamContaining scans a tool's inputSchema for a string-typed
// parameter whose "enum" array contains verb (case-insensitive). This is
// what makes `linuxctl restart system services nginx` work without
// hardcoding "restart" anywhere - a future fifth action added to
// services/manage's action enum works immediately, zero client changes.
func findEnumParamContaining(schema map[string]interface{}, verb string) (string, bool) {
	if schema == nil {
		return "", false
	}
	props, _ := schema["properties"].(map[string]interface{})
	for name, raw := range props {
		prop, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		enumRaw, ok := prop["enum"].([]interface{})
		if !ok {
			continue
		}
		for _, ev := range enumRaw {
			if s, ok := ev.(string); ok && strings.EqualFold(s, verb) {
				return name, true
			}
		}
	}
	return "", false
}

// isMutationOnly reports whether a tool can only be meaningfully invoked
// with an action already selected - i.e. one of its *required* parameters
// is enum-constrained (services/manage's required "action" enum). Such a
// tool must never be a candidate for Case B's "get" matching, even when its
// LinuxctlVerb (a plain target-keyword noun like "services") happens to
// collide with a genuine read-shaped tool's own keyword in the same group
// (services/list also uses "services", for `get system services`) - without
// this check, whichever of the two happened to be registered first would
// silently win every time, and callers would occasionally get "service
// argument is required" from a plain read. Tools like logs/logins also have
// an enum ("type": success/failed) but it's optional, not required, so
// they're correctly left eligible for Case B.
func isMutationOnly(t ToolDef) bool {
	if t.InputSchema == nil {
		return false
	}
	// JSON-decoded arrays are always []interface{}, never []string, even
	// though the server encodes "required" as a Go []string literal.
	reqRaw, ok := t.InputSchema["required"].([]interface{})
	if !ok || len(reqRaw) == 0 {
		return false
	}
	var required []string
	for _, r := range reqRaw {
		if s, ok := r.(string); ok {
			required = append(required, s)
		}
	}
	props, _ := t.InputSchema["properties"].(map[string]interface{})
	for _, reqField := range required {
		prop, ok := props[reqField].(map[string]interface{})
		if !ok {
			continue
		}
		if _, hasEnum := prop["enum"]; hasEnum {
			return true
		}
	}
	return false
}

// hasOptionalValueField reports whether a tool has a "value" property that
// is not in its required list - the dual-purpose read/write shape "update"
// resolves against (see the Case A comment above).
func hasOptionalValueField(t ToolDef) bool {
	if t.InputSchema == nil {
		return false
	}
	props, _ := t.InputSchema["properties"].(map[string]interface{})
	if _, hasValue := props["value"]; !hasValue {
		return false
	}
	if reqRaw, ok := t.InputSchema["required"].([]interface{}); ok {
		for _, r := range reqRaw {
			if s, ok := r.(string); ok && s == "value" {
				return false
			}
		}
	}
	return true
}

// matchToolByLinuxctlVerb finds the tool in group that rest[0] refers to.
// If rest[0] literally matches some tool's LinuxctlVerb, that tool wins and
// rest[0] is consumed. Otherwise, the tool in this group whose LinuxctlVerb
// equals the ambient ubiquitous read verb "get" (the bare-reachable default
// - see plan/linuxctl-redesign.md's "files/read stays bare" note) is used,
// and nothing is consumed from rest.
func matchToolByLinuxctlVerb(reg Registry, group string, rest []string) (ToolDef, []string, bool) {
	var candidates []ToolDef
	for _, t := range reg.Tools {
		if t.ToolsGroup == group && !isMutationOnly(t) {
			candidates = append(candidates, t)
		}
	}
	if len(rest) > 0 {
		for _, t := range candidates {
			if t.LinuxctlVerb != "" && t.LinuxctlVerb == rest[0] {
				return t, rest[1:], true
			}
		}
	}
	for _, t := range candidates {
		if t.LinuxctlVerb == "get" {
			return t, rest, true
		}
	}
	return ToolDef{}, rest, false
}

func findResourceByTarget(reg Registry, group string, rest []string) (ResourceDef, bool) {
	if len(rest) == 0 {
		return ResourceDef{}, false
	}
	for _, r := range reg.Resources {
		if r.Group == group && r.LinuxctlVerb == rest[0] {
			return r, true
		}
	}
	return ResourceDef{}, false
}

func findTemplateByTarget(reg Registry, group string, rest []string) (TemplateDef, []string, bool) {
	var candidates []TemplateDef
	for _, t := range reg.Templates {
		if t.Group == group {
			candidates = append(candidates, t)
		}
	}
	if len(candidates) == 1 && (len(rest) == 0 || candidates[0].LinuxctlVerb == "" || candidates[0].LinuxctlVerb != rest[0]) {
		// Only one template in this group - no keyword needed to disambiguate.
		return candidates[0], rest, true
	}
	if len(rest) > 0 {
		for _, t := range candidates {
			if t.LinuxctlVerb == rest[0] {
				return t, rest[1:], true
			}
		}
	}
	if len(candidates) > 0 {
		return candidates[0], rest, true
	}
	return TemplateDef{}, rest, false
}

// Resolve implements the four resolution cases from
// plan/linuxctl-redesign.md's resolver algorithm.
func Resolve(reg Registry, verb, group string, rest []string) (Action, error) {
	// Case A: mutation verb - direct linuxctl_verb match first, then
	// enum-value-as-verb scanning. Must never fire for the universal verbs
	// ("get", "describe") - "get" is deliberately also used as the
	// LinuxctlVerb sentinel for bare-reachable tools (see
	// matchToolByLinuxctlVerb), which would otherwise make this loop match
	// e.g. files/read immediately on *every* "get files ..." command,
	// short-circuiting Case B's keyword matching before it ever runs.
	if verb != "get" && verb != "describe" {
		for _, t := range reg.Tools {
			if t.ToolsGroup == group && t.LinuxctlVerb == verb {
				return Action{Kind: "tool_call", Tool: t, Args: map[string]interface{}{}, Positional: rest}, nil
			}
		}
		for _, t := range reg.Tools {
			if t.ToolsGroup != group {
				continue
			}
			if param, ok := findEnumParamContaining(t.InputSchema, verb); ok {
				args := map[string]interface{}{param: verb}
				positional := rest
				if t.LinuxctlVerb != "" && len(rest) > 0 && rest[0] == t.LinuxctlVerb {
					positional = rest[1:]
				}
				return Action{Kind: "tool_call", Tool: t, Args: args, Positional: positional}, nil
			}
		}

		// A third mutation pattern, distinct from a direct verb match or an
		// enum-derived action: some tools (kernel/system-control) are
		// dual-purpose - the same tool reads when its "value" field is
		// omitted and writes when it's given, rather than having a separate
		// tool or an enum of named actions. "update" resolves to whichever
		// tool in this group has that shape, using the same keyword-matching
		// Case B uses for reads, since it's genuinely the same underlying
		// tool either way (`get kernel sysctl x` and `update kernel sysctl x
		// 1` are the same tool, just with or without a trailing value).
		if verb == "update" {
			if tool, remaining, ok := matchToolByLinuxctlVerb(reg, group, rest); ok && hasOptionalValueField(tool) {
				return Action{Kind: "tool_call", Tool: tool, Args: map[string]interface{}{}, Positional: remaining}, nil
			}
		}
	}

	// Case B: get - the sole universal read verb, one result or many.
	// "top" is handled here as the one keyword in "processes" that doesn't
	// map to a single tool/resource call.
	if verb == "get" {
		if group == "processes" && len(rest) > 0 && rest[0] == "top" {
			return Action{Kind: "top_snapshot"}, nil
		}
		if tool, remaining, ok := matchToolByLinuxctlVerb(reg, group, rest); ok {
			return Action{Kind: "tool_call", Tool: tool, Args: map[string]interface{}{}, Positional: remaining}, nil
		}
		if res, ok := findResourceByTarget(reg, group, rest); ok {
			return Action{Kind: "resource_read", ResourceURI: res.URI}, nil
		}
	}

	// Case C: describe - resource template aggregation.
	if verb == "describe" {
		if tpl, remaining, ok := findTemplateByTarget(reg, group, rest); ok {
			return Action{Kind: "template_read", Template: tpl, Positional: remaining}, nil
		}
	}

	return Action{}, fmt.Errorf("no verb %q in group %q - try: linuxctl explain %s", verb, group, group)
}

// fillTemplate substitutes {name}/{pid}/{path}/{type} placeholders in a
// uriTemplate with positional args, in order of appearance.
func fillTemplate(uriTemplate string, positional []string) string {
	result := uriTemplate
	idx := 0
	for {
		start := strings.Index(result, "{")
		if start == -1 || idx >= len(positional) {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start
		result = result[:start] + positional[idx] + result[end+1:]
		idx++
	}
	return result
}

// mapPositionalArgs assigns leftover positional args to a tool's own
// single-target schema property, tried in priority order. This generalizes
// the "map first positional to path" heuristic the original client already
// used for files/list, extended to every other single-identifier tool
// param this schema surface actually uses.
var positionalFieldPriority = []string{"path", "pid", "device", "host", "url", "key", "value", "service", "name", "content"}

func mapPositionalArgs(schema map[string]interface{}, args map[string]interface{}, positional []string) []string {
	if len(positional) == 0 || schema == nil {
		return positional
	}
	props, _ := schema["properties"].(map[string]interface{})
	for _, field := range positionalFieldPriority {
		if len(positional) == 0 {
			break
		}
		if _, exists := props[field]; !exists {
			continue
		}
		if _, alreadySet := args[field]; alreadySet {
			continue
		}
		prop, _ := props[field].(map[string]interface{})
		val := positional[0]
		if str(prop, "type") == "integer" {
			if n, err := strconv.Atoi(val); err == nil {
				args[field] = n
				positional = positional[1:]
				continue
			}
		}
		args[field] = val
		positional = positional[1:]
	}
	return positional
}
