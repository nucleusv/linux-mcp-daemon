package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
	"github.com/nucleusv/linux-mcp-daemon/internal/netpolicy"
)

type SudoConfig struct {
	Users map[string]UserSudo `yaml:"users"`
}

type UserSudo struct {
	Privileged PrivilegedConfig `yaml:"privileged"`
}

type PrivilegedConfig struct {
	Tools     map[string]ToolPrivilege `yaml:"tools"`
	Resources map[string][]string      `yaml:"resources,omitempty"`
}

type ToolPrivilege struct {
	Allowed bool     `yaml:"allowed"`
	Paths   []string `yaml:"paths"`
	// Containers restricts which containers a docker/* tool that names one
	// may touch (name or ID globs). It is per-tool, like Paths, so a
	// docker/exec grant and a docker/manage grant for the same user name
	// different containers. Absent on an allowed grant = refuse everything;
	// containers: ["*"] is the explicit everywhere, like paths: ["/"].
	//
	// Unlike Paths, this is the *only* gate: the docker socket is
	// all-or-nothing, the Engine API has no per-container authorization, and
	// the worker is root - so there is no kernel check behind this list.
	Containers []string `yaml:"containers,omitempty"`
	// Prune restricts which kinds of unused object docker/prune may reclaim
	// (see PruneTargets). Same shape as Containers and for the same reason:
	// prune names no container, it names a category of garbage, and the
	// Engine API has no authorization of its own. Absent on an allowed grant
	// = refuse everything; there is deliberately no "all".
	Prune []string `yaml:"prune,omitempty"`
	// Users are the per-account rules of cron/manage, and only its: which
	// accounts' crontabs a privileged call may view and edit. Keys are single
	// account names - no wildcards - so root is covered only when named, and
	// then for view only. A caller's own crontab needs no rule.
	Users map[string]CronRule `yaml:"users,omitempty"`
	// Network restricts where an outbound network tool (network/curl,
	// network/ping) may connect. Unlike Allowed, which only governs
	// running as root, it applies to every call of the tool - network
	// access doesn't depend on the worker's uid. Absent = unrestricted.
	Network *netpolicy.Policy `yaml:"network,omitempty"`
	// Sysctl restricts kernel/system-control writes. Absent = unrestricted
	// (writes allowed wherever the tool may run as root, as before).
	Sysctl *SysctlPolicy `yaml:"sysctl,omitempty"`
}

// SysctlPolicy limits which kernel parameters a user may change. Writing
// some parameters is equivalent to running code as root (e.g.
// kernel.core_pattern, kernel.modprobe), so it's worth being able to allow
// writes to only a known set of keys. (For read-only access, don't grant
// `allowed` at all: reading needs no root, and without root the OS refuses
// every write.)
type SysctlPolicy struct {
	// RemovedReadOnly catches the former read_only option. It was removed
	// as redundant with simply not granting `allowed`, and is kept only so
	// a leftover `read_only: true` fails loudly at load - a silently ignored
	// key would leave writes open while the config reads as closed.
	RemovedReadOnly *bool `yaml:"read_only"`
	// WriteKeys, if non-empty, is the only set of keys that may be
	// written. Entries are dotted keys or glob patterns ("vm.*",
	// "net.ipv4.conf.*.rp_filter"); "*" matches within one dotted
	// component.
	WriteKeys []string `yaml:"write_keys,omitempty"`
}

// CanWriteSysctl reports whether username may write key (dotted form)
// through kernel/system-control, and why not if they can't.
func (c *SudoConfig) CanWriteSysctl(username, key string) (bool, string) {
	userSudo, ok := c.Users[username]
	if !ok {
		return true, ""
	}
	privs, ok := userSudo.Privileged.Tools["kernel/system-control"]
	if !ok || privs.Sysctl == nil {
		return true, ""
	}
	if len(privs.Sysctl.WriteKeys) == 0 {
		return true, ""
	}
	for _, pattern := range privs.Sysctl.WriteKeys {
		if sysctlKeyMatch(pattern, key) {
			return true, ""
		}
	}
	return false, fmt.Sprintf("writing %s is not permitted for this user (not in sysctl.write_keys in mcp-sudo.yaml)", key)
}

func validSysctlPattern(pattern string) error {
	_, err := path.Match(strings.ReplaceAll(pattern, ".", "/"), "")
	return err
}

// sysctlKeyMatch matches a dotted key against a dotted glob, treating "."
// as the separator so "*" never spans components ("vm.*" matches
// "vm.swappiness" but not "vm.a.b").
func sysctlKeyMatch(pattern, key string) bool {
	ok, err := path.Match(strings.ReplaceAll(pattern, ".", "/"), strings.ReplaceAll(key, ".", "/"))
	return err == nil && ok
}

func LoadSudoConfig(path string) (*SudoConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := ParseSudoConfig(data, false)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// LoadSudoConfigStrict is LoadSudoConfig rejecting unknown keys too: a
// misspelled key (`path:` for `paths:`) is an error rather than a rule
// silently left out. Used by daemon/reload-config and linuxctl edit.
func LoadSudoConfigStrict(path string) (*SudoConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := ParseSudoConfig(data, true)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// ParseSudoConfig parses and validates mcp-sudo.yaml; strict rejects
// unknown keys.
func ParseSudoConfig(data []byte, strict bool) (*SudoConfig, error) {
	var cfg SudoConfig
	if err := decodeYAML(data, &cfg, strict); err != nil {
		return nil, err
	}

	// Validation
	for username, userSudo := range cfg.Users {
		for toolName, privs := range userSudo.Privileged.Tools {
			if privs.Sysctl != nil {
				if privs.Sysctl.RemovedReadOnly != nil {
					return nil, fmt.Errorf("user '%s', tool '%s': sysctl.read_only is no longer supported - for read-only access remove `allowed` (reading needs no root, and without root the OS refuses writes); to limit writes use sysctl.write_keys", username, toolName)
				}
				for _, p := range privs.Sysctl.WriteKeys {
					if err := validSysctlPattern(p); err != nil {
						return nil, fmt.Errorf("user '%s', tool '%s': invalid sysctl.write_keys pattern %q: %v", username, toolName, p, err)
					}
				}
			}
			if privs.Network != nil {
				if err := privs.Network.Validate(); err != nil {
					return nil, fmt.Errorf("user '%s', tool '%s': %v", username, toolName, err)
				}
			}
			if PathTools[toolName] && privs.Allowed && len(privs.Paths) == 0 && strict {
				return nil, fmt.Errorf("user '%s', tool '%s': allowed without paths - as root it could only be refused; list the directories it may reach, or paths: [\"/\"] for the whole filesystem", username, toolName)
			}
			if len(privs.Paths) > 0 && !PathTools[toolName] && strict {
				return nil, fmt.Errorf("user '%s', tool '%s': paths has no effect - %s takes no path argument (paths can limit: %s)", username, toolName, toolName, strings.Join(sortedPathTools(), ", "))
			}
			if ContainerTools[toolName] && privs.Allowed && len(privs.Containers) == 0 && strict {
				return nil, fmt.Errorf("user '%s', tool '%s': allowed without containers - as root it could only be refused; list the containers it may touch (name or ID globs), or containers: [\"*\"] for all of them", username, toolName)
			}
			if len(privs.Containers) > 0 && !ContainerTools[toolName] && strict {
				return nil, fmt.Errorf("user '%s', tool '%s': containers has no effect - %s names no container (containers can limit: %s)", username, toolName, toolName, strings.Join(sortedContainerTools(), ", "))
			}
			for _, t := range privs.Prune {
				if !validPruneTarget(t) {
					return nil, fmt.Errorf("user '%s', tool '%s': unknown prune target %q (one of: %s)", username, toolName, t, strings.Join(PruneTargets, ", "))
				}
			}
			if toolName == PruneTool && privs.Allowed && len(privs.Prune) == 0 && strict {
				return nil, fmt.Errorf("user '%s', tool '%s': allowed without prune - as root it could only be refused; list the targets it may reclaim (%s). There is no \"all\" on purpose: pruning volumes deletes data nothing can rebuild", username, toolName, strings.Join(PruneTargets, ", "))
			}
			if len(privs.Prune) > 0 && toolName != PruneTool && strict {
				return nil, fmt.Errorf("user '%s', tool '%s': prune has no effect - only %s reclaims unused objects", username, toolName, PruneTool)
			}
			if len(privs.Users) > 0 && toolName != CronTool && strict {
				return nil, fmt.Errorf("user '%s', tool '%s': users has no effect - only %s takes per-account crontab rules", username, toolName, CronTool)
			}
			if toolName == CronTool && privs.Allowed && len(privs.Users) == 0 && strict {
				return nil, fmt.Errorf("user '%s', tool '%s': allowed without users - a call on another account's crontab could only be refused; name the accounts and what may be done: users: {test_user: {view: true, edit: true}} (your own crontab needs no rule)", username, toolName)
			}
			for target, rule := range privs.Users {
				if err := validateCronRule(target, rule); err != nil {
					return nil, fmt.Errorf("user '%s', tool '%s', users.%s: %v", username, toolName, target, err)
				}
				if rule.Edit && !rule.View { // edit implies view
					rule.View = true
					privs.Users[target] = rule
				}
			}
		}
	}

	return &cfg, nil
}

// CanRunAsRoot checks if a specific user is authorized to run a specific tool as root.
func (c *SudoConfig) CanRunAsRoot(username, toolName string) bool {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[toolName]; ok {
			return privs.Allowed
		}
	}
	return false
}

// NetworkPolicy returns the network restrictions configured for this
// user's use of toolName, or nil for none.
func (c *SudoConfig) NetworkPolicy(username, toolName string) *netpolicy.Policy {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[toolName]; ok {
			return privs.Network
		}
	}
	return nil
}

// GetAllowedPaths fetches the restricted paths for a tool.
// PathTools are the tools that take a filesystem `path` argument. Running
// one as root requires `paths:` in its grant - there is no "allowed
// everywhere" default; `paths: ["/"]` says so explicitly. `paths:` on any
// other tool would restrict nothing, so the config loader rejects it.
var PathTools = map[string]bool{
	"files/list": true, "files/read": true, "files/create": true, "files/update": true,
	"files/find": true, "files/filetype": true, "files/chmod": true, "files/chown": true,
	"disks/free": true, "disks/usage": true,
}

func (c *SudoConfig) GetAllowedPaths(username, toolName string) []string {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[toolName]; ok {
			return privs.Paths
		}
	}
	return nil
}

// DockerTools are every tool reaching the Docker Engine API. The socket is
// root-owned and all-or-nothing, so each of these runs as root or not at all:
// the daemon forces `privileged: true` and refuses the call outright when
// `allowed: true` is missing from the grant. ContainerTools below is the
// subset that also needs a `containers:` list.
var DockerTools = map[string]bool{
	"docker/containers": true, "docker/manage": true, "docker/logs": true,
	"docker/exec": true, "docker/images": true, "docker/volumes": true,
	"docker/networks": true, "docker/inspect": true, "docker/prune": true,
}

// PruneTool is the one tool taking a `prune:` list, and PruneTargets the kinds
// of unused object it can reclaim - the same relation PathTools has to `paths:`.
// Kept as a name rather than a map because there is exactly one such tool: a
// second one would mean a second blast radius nobody asked for.
const PruneTool = "docker/prune"

// The vocabulary itself lives with the tool that speaks it, so the config
// loader and the worker can never disagree about what a target is called.
var PruneTargets = docker.PruneTargets

func validPruneTarget(t string) bool { return docker.PruneAllowed(t, PruneTargets) }

// GetAllowedPruneTargets returns the prune targets this user is granted. An
// empty result refuses everything, like GetAllowedContainers.
func (c *SudoConfig) GetAllowedPruneTargets(username, toolName string) []string {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[toolName]; ok {
			return privs.Prune
		}
	}
	return nil
}

// ContainerTools are the docker/* tools that name one container. Each needs
// `containers:` in its grant; the plural listings (docker/containers,
// docker/images, docker/volumes, docker/networks) name none, so `containers:`
// on those is a config error rather than a silent no-op.
//
// docker/inspect is the internal worker behind the container:// templates, so
// its grant is what scopes those reads.
var ContainerTools = map[string]bool{
	"docker/manage": true, "docker/logs": true, "docker/exec": true, "docker/inspect": true,
}

// GetAllowedContainers returns the container globs this user may touch with
// toolName. An empty result refuses everything - the runtime counterpart of
// the strict check above, so a grant missing `containers:` is inert rather
// than permissive.
func (c *SudoConfig) GetAllowedContainers(username, toolName string) []string {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[toolName]; ok {
			return privs.Containers
		}
	}
	return nil
}

// CoversRoot reports whether an allowed-paths list includes "/" - then
// every path is allowed and following symlinks can't lead anywhere new.
func CoversRoot(allowed []string) bool {
	for _, a := range allowed {
		if strings.HasPrefix(a, "/") && filepath.Clean(a) == "/" {
			return true
		}
	}
	return false
}

// PathAllowed reports whether path lies inside one of the allowed
// directories, returning the cleaned path the caller must then operate on.
// The check is lexical and boundary-aware: path is cleaned first (so
// "/tmp/../etc/shadow" is judged as "/etc/shadow", not waved through by a
// raw prefix match on "/tmp"), and an allowed "/tmp" covers "/tmp" and
// "/tmp/x" but not "/tmpfoo". Relative paths are always rejected - they'd be
// resolved against the worker's working directory, which no allowlist entry
// describes.
func PathAllowed(path string, allowed []string) (string, bool) {
	if !strings.HasPrefix(path, "/") {
		return "", false
	}
	clean := filepath.Clean(path)
	for _, a := range allowed {
		if !strings.HasPrefix(a, "/") {
			continue
		}
		a = filepath.Clean(a)
		if a == "/" || clean == a || strings.HasPrefix(clean, a+"/") {
			return clean, true
		}
	}
	return "", false
}

// ResourceGrantCoversRoot reports whether username's grant for scheme
// covers every path - "" (the whole scheme) or "/" - so following a
// symlink can't reach anything the grant doesn't already allow.
func (c *SudoConfig) ResourceGrantCoversRoot(username, scheme string) bool {
	userSudo, ok := c.Users[username]
	if !ok {
		return false
	}
	for _, p := range userSudo.Privileged.Resources[scheme] {
		if p == "" || (strings.HasPrefix(p, "/") && filepath.Clean(p) == "/") {
			return true
		}
	}
	return false
}

// CanReadResourceAsRoot checks if a specific user is authorized to read a resource path as root.
func (c *SudoConfig) CanReadResourceAsRoot(username, scheme, resourcePath string) bool {
	if userSudo, ok := c.Users[username]; ok {
		if resPaths, ok := userSudo.Privileged.Resources[scheme]; ok {
			for _, p := range resPaths {
				switch {
				// Exact match - how exact-match resources (devices://usb,
				// os://uname, ...) are granted: callers pass the literal
				// sentinel "*" and the config lists "*".
				case resourcePath == p:
					return true
				// Empty prefix grants a whole prefix-matched scheme (file://,
				// service://, process://). A literal "*" deliberately does
				// NOT - see the comment in configs/mcp-sudo.yaml.
				case p == "":
					return true
				// Filesystem path grants are matched on the cleaned path with
				// a directory boundary (see PathAllowed), so neither
				// "/var/log/../../etc/shadow" nor "/var/logs-private" gets
				// through a "/var/log" grant.
				case strings.HasPrefix(p, "/"):
					if _, ok := PathAllowed(resourcePath, []string{p}); ok {
						return true
					}
				// Non-path prefixes (service names, PIDs) keep plain prefix
				// matching, as before.
				case !strings.HasPrefix(resourcePath, "/") && strings.HasPrefix(resourcePath, p):
					return true
				}
			}
		}
	}
	return false
}

func sortedPathTools() []string {
	names := make([]string, 0, len(PathTools))
	for n := range PathTools {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func sortedContainerTools() []string {
	names := make([]string, 0, len(ContainerTools))
	for n := range ContainerTools {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// CronTool is the one tool taking a `users:` list, and CronRule what a grant
// allows on one account's crontab.
const CronTool = "cron/manage"

// CronRule: View allows reading (and listing) the account's crontab, Edit
// replacing it as a whole. Edit implies View.
type CronRule struct {
	View bool `yaml:"view"`
	Edit bool `yaml:"edit"`
}

var accountNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,63}$`)

// ValidAccountName reports whether name is a plausible OS account name: no
// leading dash (it could become an option), no glob characters, no path bits.
func ValidAccountName(name string) bool { return accountNameRE.MatchString(name) }

func validateCronRule(target string, r CronRule) error {
	if !ValidAccountName(target) {
		return fmt.Errorf("%q is not a single account name (no wildcards or globs - name each account)", target)
	}
	if !r.View && !r.Edit {
		return fmt.Errorf("rule allows nothing: set view: true, edit: true, or both")
	}
	if target == "root" && r.Edit {
		return fmt.Errorf("root's crontab may be viewed but never edited through mcpd: remove edit: true")
	}
	return nil
}

// CronRuleFor returns what username's cron/manage grant allows on target's
// crontab; the zero rule (nothing allowed) when the grant names no such account.
func (c *SudoConfig) CronRuleFor(username, target string) CronRule {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[CronTool]; ok && privs.Allowed {
			r := privs.Users[target]
			if r.Edit {
				r.View = true
			}
			if target == "root" {
				r.Edit = false
			}
			return r
		}
	}
	return CronRule{}
}

// CronViewAccounts lists the accounts whose crontab username may view, sorted.
func (c *SudoConfig) CronViewAccounts(username string) []string {
	var out []string
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[CronTool]; ok && privs.Allowed {
			for name, r := range privs.Users {
				if r.View || r.Edit {
					out = append(out, name)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}
