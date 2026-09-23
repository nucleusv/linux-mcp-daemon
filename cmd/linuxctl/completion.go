package main

// Shell completion. `linuxctl completion bash|zsh` prints a script that
// calls back into the hidden `linuxctl __complete -- <words...>` command on
// every Tab press. Candidates come from the same live registry Resolve uses
// (tools/list + resources/list + resources/templates/list), so a tool or
// resource added server-side completes with zero client changes - the same
// principle as the resolver itself. The registry is cached on disk for a
// few minutes, since each fetch is four round trips to the daemon.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	completionCacheTTL = 5 * time.Minute
	completionTimeout  = 3 * time.Second
)

// completionState is set when running as `__complete`: real stdout is saved
// here and os.Stdout is pointed at /dev/null, so connection errors printed by
// the normal startup path never leak into the shell's candidate list.
var (
	completionOut   *os.File
	completionWords []string
	completionOnce  sync.Once
)

const bashCompletionScript = `# linuxctl bash completion
# Load with:  source <(linuxctl completion bash)
_linuxctl() {
    local cur words cword
    if declare -F _get_comp_words_by_ref >/dev/null 2>&1; then
        # Keep URIs like network://interfaces as one word.
        _get_comp_words_by_ref -n : cur words cword
    else
        cur="${COMP_WORDS[COMP_CWORD]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    fi
    local bin="${words[0]}"
    type -P "$bin" >/dev/null 2>&1 || bin=linuxctl
    local IFS=$'\n'
    COMPREPLY=($(compgen -W "$("$bin" __complete -- "${words[@]:1:cword}" 2>/dev/null)" -- "$cur"))
    if declare -F __ltrim_colon_completions >/dev/null 2>&1; then
        __ltrim_colon_completions "$cur"
    fi
}
complete -F _linuxctl linuxctl
`

const zshCompletionScript = `#compdef linuxctl
# linuxctl zsh completion
# Load with:  source <(linuxctl completion zsh)
_linuxctl() {
    local bin=${words[1]}
    (( $+commands[$bin] )) || [[ -x $bin ]] || bin=linuxctl
    local -a candidates
    candidates=("${(@f)$(command $bin __complete -- "${(@)words[2,CURRENT]}" 2>/dev/null)}")
    candidates=(${candidates:#})
    compadd -Q -- $candidates
}
if (( $+functions[compdef] )); then
    compdef _linuxctl linuxctl
else
    autoload -Uz compinit && compinit && compdef _linuxctl linuxctl
fi
`

func runCompletionScript(args []string) {
	shell := ""
	if len(args) > 0 {
		shell = args[0]
	}
	switch shell {
	case "bash":
		fmt.Print(bashCompletionScript)
	case "zsh":
		fmt.Print(zshCompletionScript)
	default:
		fmt.Println("Usage: linuxctl completion <bash|zsh>")
		fmt.Println("  bash:  source <(linuxctl completion bash)   # add to ~/.bashrc")
		fmt.Println("  zsh:   source <(linuxctl completion zsh)    # add to ~/.zshrc")
		os.Exit(1)
	}
}

// startCompletion handles everything `__complete` can do before a daemon
// connection: strips the global flags out of the typed words (applying
// -server/-token so the connection uses them), answers from cache or with
// static candidates when possible, and arms a timeout so a slow or
// unreachable daemon never hangs the shell. Returns only when the caller
// must connect and fetch a fresh registry.
func startCompletion(args []string) {
	completionOut = os.Stdout
	if devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
		os.Stdout = devNull
	}

	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		args = []string{""}
	}

	// Leading global flags, as typed before the verb.
	i := 0
	for i < len(args)-1 && strings.HasPrefix(args[i], "-") {
		name := strings.TrimLeft(args[i], "-")
		value := ""
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			name, value = name[:eq], name[eq+1:]
			i++
		} else if name == "server" || name == "token" || name == "config-path" {
			if i+1 == len(args)-1 {
				// Cursor is on this flag's value - nothing sensible to offer.
				emitCompletions(nil)
			}
			value = args[i+1]
			i += 2
		} else {
			i++
		}
		switch name {
		case "server":
			*serverURL = value
		case "token":
			*token = value
		}
	}
	completionWords = args[i:]

	if len(completionWords) == 1 && strings.HasPrefix(completionWords[0], "-") {
		emitCompletions([]string{"-server", "-token", "-config-path"})
	}

	// Anything answerable without the registry.
	if cands, ok := staticCompletions(completionWords); ok {
		emitCompletions(cands)
	}

	authToken := *token
	if authToken == "" {
		authToken = os.Getenv("MCP_TOKEN")
	}
	if reg, ok := loadCachedRegistry(authToken); ok {
		emitCompletions(completeWords(reg, completionWords))
	}
	if authToken == "" {
		emitCompletions(completeWords(Registry{}, completionWords))
	}

	go func() {
		time.Sleep(completionTimeout)
		emitCompletions(completeWords(Registry{}, completionWords))
	}()
}

// finishCompletion runs once the daemon connection is up.
func finishCompletion(authToken string) {
	reg := fetchRegistry(authToken)
	saveCachedRegistry(authToken, reg)
	emitCompletions(completeWords(reg, completionWords))
}

func emitCompletions(cands []string) {
	completionOnce.Do(func() {
		seen := map[string]bool{}
		for _, c := range cands {
			if c != "" && !seen[c] {
				seen[c] = true
				fmt.Fprintln(completionOut, c)
			}
		}
		os.Exit(0)
	})
	// Another goroutine already emitted and is exiting.
	select {}
}

func registryCachePath(authToken string) string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	// Keyed by server+token: different users see different tools.
	sum := sha256.Sum256([]byte(*serverURL + "\x00" + authToken))
	return filepath.Join(dir, "linuxctl", "registry-"+hex.EncodeToString(sum[:8])+".json")
}

func loadCachedRegistry(authToken string) (Registry, bool) {
	path := registryCachePath(authToken)
	if path == "" {
		return Registry{}, false
	}
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > completionCacheTTL {
		return Registry{}, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Registry{}, false
	}
	var reg Registry
	if err := json.Unmarshal(b, &reg); err != nil {
		return Registry{}, false
	}
	return reg, true
}

func saveCachedRegistry(authToken string, reg Registry) {
	path := registryCachePath(authToken)
	if path == "" || len(reg.Tools) == 0 {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	if b, err := json.Marshal(reg); err == nil {
		os.WriteFile(path, b, 0o600)
	}
}

var outputFormats = []string{"json", "yaml", "table", "wide"}

// staticCompletions covers positions whose candidates don't depend on the
// daemon at all.
func staticCompletions(words []string) ([]string, bool) {
	n := len(words)
	if n >= 2 && (words[n-2] == "-o" || words[n-2] == "--output") {
		return outputFormats, true
	}
	positional := positionalWords(words[:n-1])
	if len(positional) == 1 && positional[0] == "completion" {
		return []string{"bash", "zsh"}, true
	}
	if len(positional) == 2 && positional[0] == "get" && positional[1] == "mcp-api" {
		return []string{"tools", "resources", "prompts", "info"}, true
	}
	return nil, false
}

// positionalWords drops --flag value pairs, mirroring splitFlagsAndPositional.
func positionalWords(words []string) []string {
	var out []string
	for i := 0; i < len(words); i++ {
		w := words[i]
		if w == "-o" {
			i++
			continue
		}
		if strings.HasPrefix(w, "--") {
			if i+1 < len(words) && !strings.HasPrefix(words[i+1], "-") {
				i++
			}
			continue
		}
		out = append(out, w)
	}
	return out
}

var mcpdAdminVerbs = []string{"create", "delete", "update", "list", "describe"}

// completeWords returns candidates for the last word in words (the one
// being typed), given the ones before it.
func completeWords(reg Registry, words []string) []string {
	n := len(words)
	cur := words[n-1]
	positional := positionalWords(words[:n-1])

	if strings.HasPrefix(cur, "-") {
		return completeFlags(reg, positional)
	}

	switch len(positional) {
	case 0:
		return sorted(topLevelVerbs(reg))
	case 1:
		return sorted(groupsForVerb(reg, positional[0]))
	case 2:
		return sorted(keywordsFor(reg, positional[0], positional[1]))
	}
	return nil
}

func topLevelVerbs(reg Registry) []string {
	verbs := []string{"get", "describe", "explain", "tool", "resource", "completion", "ping", "create", "update", "delete", "list"}
	for _, t := range reg.Tools {
		if isMutationOnly(t) {
			verbs = append(verbs, requiredEnumValues(t)...)
		}
	}
	return verbs
}

func groupsForVerb(reg Registry, verb string) []string {
	var groups []string
	switch verb {
	case "tool":
		for _, t := range reg.Tools {
			groups = append(groups, t.Name)
		}
		return groups
	case "resource":
		for _, r := range reg.Resources {
			groups = append(groups, r.URI)
		}
		for _, t := range reg.Templates {
			groups = append(groups, t.URITemplate)
		}
		return groups
	case "explain":
		for _, t := range reg.Tools {
			groups = append(groups, t.ToolsGroup)
		}
		for _, r := range reg.Resources {
			groups = append(groups, r.Group)
		}
		for _, t := range reg.Templates {
			groups = append(groups, t.Group)
		}
		return groups
	case "get":
		groups = append(groups, "mcp-api")
		for _, t := range reg.Tools {
			if !isMutationOnly(t) {
				groups = append(groups, t.ToolsGroup)
			}
		}
		for _, r := range reg.Resources {
			groups = append(groups, r.Group)
		}
		return groups
	case "describe":
		groups = append(groups, "mcpd")
		for _, t := range reg.Templates {
			groups = append(groups, t.Group)
		}
		return groups
	}

	// Mutation verbs: groups where Resolve's Case A would match.
	for _, v := range mcpdAdminVerbs {
		if verb == v {
			groups = append(groups, "mcpd")
		}
	}
	for _, t := range reg.Tools {
		if t.LinuxctlVerb == verb || contains(requiredEnumValues(t), verb) ||
			(verb == "update" && hasOptionalValueField(t)) {
			groups = append(groups, t.ToolsGroup)
		}
	}
	return groups
}

func keywordsFor(reg Registry, verb, group string) []string {
	var kws []string
	if group == "mcpd" {
		return []string{"user"}
	}
	switch verb {
	case "get":
		for _, t := range reg.Tools {
			if t.ToolsGroup == group && !isMutationOnly(t) && t.LinuxctlVerb != "" && t.LinuxctlVerb != "get" {
				kws = append(kws, t.LinuxctlVerb)
			}
		}
		for _, r := range reg.Resources {
			if r.Group == group && r.LinuxctlVerb != "" {
				kws = append(kws, r.LinuxctlVerb)
			}
		}
		if group == "processes" {
			kws = append(kws, "top")
		}
	case "describe":
		for _, t := range reg.Templates {
			if t.Group == group && t.LinuxctlVerb != "" {
				kws = append(kws, t.LinuxctlVerb)
			}
		}
	default:
		// e.g. "restart system services <name>", "update kernel sysctl <key> <value>"
		for _, t := range reg.Tools {
			if t.ToolsGroup != group || t.LinuxctlVerb == "" || t.LinuxctlVerb == verb || t.LinuxctlVerb == "get" {
				continue
			}
			if contains(requiredEnumValues(t), verb) || (verb == "update" && hasOptionalValueField(t)) {
				kws = append(kws, t.LinuxctlVerb)
			}
		}
	}
	return kws
}

// completeFlags offers --<param> for the tool the typed words resolve to.
func completeFlags(reg Registry, positional []string) []string {
	flags := []string{"--output", "-o"}
	var tool *ToolDef
	if len(positional) >= 2 && positional[0] == "tool" {
		for i := range reg.Tools {
			if reg.Tools[i].Name == positional[1] {
				tool = &reg.Tools[i]
			}
		}
	} else if len(positional) >= 2 {
		if action, err := Resolve(reg, positional[0], positional[1], positional[2:]); err == nil && action.Kind == "tool_call" {
			tool = &action.Tool
		}
	}
	if tool != nil {
		props, _ := tool.InputSchema["properties"].(map[string]interface{})
		for name := range props {
			if name != "output_format" {
				flags = append(flags, "--"+name)
			}
		}
	}
	return sorted(flags)
}

// requiredEnumValues returns the enum values of a tool's required enum
// parameters - the "action as verb" values (start, stop, restart, ...).
func requiredEnumValues(t ToolDef) []string {
	var vals []string
	reqRaw, _ := t.InputSchema["required"].([]interface{})
	props, _ := t.InputSchema["properties"].(map[string]interface{})
	for _, r := range reqRaw {
		name, _ := r.(string)
		prop, _ := props[name].(map[string]interface{})
		enumRaw, _ := prop["enum"].([]interface{})
		for _, ev := range enumRaw {
			if s, ok := ev.(string); ok {
				vals = append(vals, s)
			}
		}
	}
	return vals
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func sorted(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range list {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
