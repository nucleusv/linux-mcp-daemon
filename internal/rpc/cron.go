package rpc

// cron/manage: the schema and the one place its calls are prepared.
//
// Whose crontab a call touches is decided here, in the master, and handed to
// the worker in reserved "_" arguments that a caller's own values can never
// reach: the caller's own crontab is run as the caller, another account's
// needs root and a rule naming that account, and root's is view-only.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/user"
	"strconv"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/worker"
)

var cronReserved = []string{"_mode", "_target_user", "_target_uid", "_target_gid", "_target_groups", "_view_users", "_info"}

// cronPlan is a prepared call: the arguments for the worker and whether the
// worker runs as root (true for another account's crontab and for the list).
type cronPlan struct {
	Args       json.RawMessage
	Privileged bool
	Write      bool
	Target     string
}

// accountLookup resolves an account name (os/user.Lookup in production).
type accountLookup func(name string) (*user.User, error)

// prepareCronCall authorizes one cron/manage call for caller and builds the
// worker's arguments. info asks for the metadata view (the /info template).
func prepareCronCall(sudoCfg *config.SudoConfig, caller string, raw json.RawMessage, info bool, lookup accountLookup) (cronPlan, error) {
	var m map[string]interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &m); err != nil {
			return cronPlan{}, fmt.Errorf("invalid arguments: %v", err)
		}
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	for _, k := range cronReserved { // never the caller's to set
		delete(m, k)
	}
	target, _ := m["user"].(string)
	if v, has := m["user"]; has && v != nil {
		if _, ok := v.(string); !ok {
			return cronPlan{}, fmt.Errorf("user must be a string")
		}
	}
	if target != "" && !config.ValidAccountName(target) {
		return cronPlan{}, fmt.Errorf("invalid account name %q", target)
	}
	content, hasContent := m["content"]
	if hasContent {
		if _, ok := content.(string); !ok {
			return cronPlan{}, fmt.Errorf("content must be a string (an empty string clears the crontab)")
		}
	}
	priv, _ := m["privileged"].(bool)
	write := hasContent

	finish := func(p cronPlan, args map[string]interface{}) (cronPlan, error) {
		b, err := json.Marshal(args)
		p.Args = b
		return p, err
	}

	// The list: privileged, no account named, no content.
	if target == "" && priv && !hasContent && !info {
		if !sudoCfg.CanRunAsRoot(caller, config.CronTool) {
			return cronPlan{}, fmt.Errorf("user %s is not authorized to list crontabs: cron/manage needs `allowed: true` and a users: rule for each account in your grant in mcp-sudo.yaml", caller)
		}
		m["_mode"] = "list"
		m["_view_users"] = append(sudoCfg.CronViewAccounts(caller), caller)
		return finish(cronPlan{Privileged: true}, m)
	}

	// Your own crontab: no root, no rule.
	if target == "" || target == caller {
		m["_mode"] = "self"
		m["_target_user"] = caller
		m["_info"] = info
		return finish(cronPlan{Write: write, Target: caller}, m)
	}

	// Another account's crontab.
	if !priv {
		return cronPlan{}, fmt.Errorf("another account's crontab needs privileged: true and a rule for %s in your cron/manage grant (users: {%s: {view: true}})", target, target)
	}
	if !sudoCfg.CanRunAsRoot(caller, config.CronTool) {
		return cronPlan{}, fmt.Errorf("user %s is not authorized to run cron/manage on another account: it needs `allowed: true` and a users: rule for %s in your grant in mcp-sudo.yaml", caller, target)
	}
	rule := sudoCfg.CronRuleFor(caller, target)
	if write && !rule.Edit {
		if rule.View {
			return cronPlan{}, fmt.Errorf("user %s may not edit %s's crontab: the cron/manage grant gives %s view only", caller, target, target)
		}
		return cronPlan{}, fmt.Errorf("user %s may not edit %s's crontab: no rule for it in the cron/manage grant (users:)", caller, target)
	}
	if !write && !rule.View {
		return cronPlan{}, fmt.Errorf("user %s may not view %s's crontab: no rule for it in the cron/manage grant (users:)", caller, target)
	}
	u, err := lookup(target)
	if err != nil {
		return cronPlan{}, fmt.Errorf("no account named %s on this host", target)
	}
	uid, _ := strconv.ParseUint(u.Uid, 10, 32)
	gid, _ := strconv.ParseUint(u.Gid, 10, 32)
	if uid == 0 && write { // root, under any name: view only, whatever a rule says
		return cronPlan{}, fmt.Errorf("root's crontab may be viewed but never edited through mcpd")
	}
	var groups []uint32
	if ids, err := u.GroupIds(); err == nil {
		for _, id := range ids {
			if n, err := strconv.ParseUint(id, 10, 32); err == nil {
				groups = append(groups, uint32(n))
			}
		}
	}
	m["_mode"] = "other"
	m["_target_user"] = target
	m["_target_uid"], m["_target_gid"], m["_target_groups"] = uint32(uid), uint32(gid), groups
	m["_info"] = info
	return finish(cronPlan{Privileged: true, Write: write, Target: target}, m)
}

// cronTools is the cron/manage schema.
func cronTools() []interface{} {
	return []interface{}{map[string]interface{}{
		"name":          config.CronTool,
		"tools_group":   "crontabs",
		"linuxctl_verb": "get",
		"description":   "Reads or replaces a user's crontab, the list of commands cron runs for that account on a schedule, and lists which accounts have one. Without `content` it reads: your own crontab as raw text exactly as `crontab -l` prints it (empty when you have none), or with `privileged: true` and no `user`, a table of the accounts that have a crontab and that your grant lets you view. With `content` it writes: the WHOLE crontab is replaced (an empty string clears it); the `crontab` command rejects a file it cannot parse and then the old crontab is unchanged; a missing final newline is added; the limit is 64 KiB. Your own crontab needs no grant and runs as you. Another account's needs `privileged: true` and a rule naming that account in your cron/manage grant (`users: {name: {view: true, edit: true}}`); `edit` implies `view`, root's crontab can be viewed but never edited, and cron.allow/cron.deny still apply to that account. WARNING: writing a crontab schedules commands as that user, and the job outlives this session and the revocation of your token; every write is audit-logged (size, lines, hash, never the content). To avoid overwriting a change made meanwhile, pass `if_match` with the sha256 you read (`output_format: json` returns it). For systemd timers use `timers/list`. Needs the `crontab` command on the host; inside a container a call sees the container's own crontabs.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user":          map[string]interface{}{"type": "string", "description": "Account whose crontab to read or replace; default is your own. Another account needs privileged: true and a rule for it"},
				"content":       map[string]interface{}{"type": "string", "description": "The complete new crontab; present = write (empty string clears it), absent = read. Standard crontab syntax: `m h dom mon dow command`, `@daily`, and NAME=value lines"},
				"if_match":      map[string]interface{}{"type": "string", "description": "sha256 of the crontab as you read it; the write is refused if it has changed since"},
				"output_format": map[string]interface{}{"type": "string", "description": "'json' (also yaml/table/wide, which return the same JSON) returns {user, exists, lines, jobs, bytes, sha256, content} for a read; default is the raw text"},
				"privileged":    map[string]interface{}{"type": "boolean", "description": "Needed to list crontabs or to act on another account (needs a cron/manage grant); ignored for your own crontab"},
			},
			"required": []string{},
		},
	}}
}

// cronAuditAttrs describes a write for the audit line - size, lines and a
// hash, never the content, which may hold secrets.
func cronAuditAttrs(raw json.RawMessage, target string) []any {
	var a struct {
		Content *string `json:"content"`
	}
	if json.Unmarshal(raw, &a) != nil || a.Content == nil {
		return nil
	}
	return []any{"target_user", target, "content_bytes", len(*a.Content), "content_lines", cronLines(*a.Content), "content_sha256", cronSum(*a.Content)}
}

func cronSum(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func cronLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimSuffix(s, "\n"), "\n") + 1
}

// readCrontabResource serves crontab://{user}/{view}: the same preparation as
// the tool, so one rule set decides who may see whose crontab. Another
// account's crontab is read by a root worker acting for the caller's `view`
// rule; there is no separate resources: grant.
func (h *RPCHandler) readCrontabResource(session *Session, sudoCfg *config.SudoConfig, uri string) (string, string, error) {
	rest := strings.TrimPrefix(uri, "crontab://")
	target, view, _ := strings.Cut(rest, "/")
	if view == "" {
		view = "text"
	}
	if target == "" || (view != "text" && view != "info") {
		return "", "", fmt.Errorf("unknown resource %s (use crontab://<user>/text or crontab://<user>/info)", uri)
	}
	args := map[string]interface{}{"user": target, "privileged": target != session.User}
	if view == "info" {
		args["output_format"] = "json"
	}
	raw, _ := json.Marshal(args)
	plan, err := prepareCronCall(sudoCfg, session.User, raw, view == "info", user.Lookup)
	if err != nil {
		return "", "", err
	}
	out, err := worker.SpawnWorker(session.User, config.CronTool, plan.Args, plan.Privileged, sudoCfg, h.timeoutFor(config.CronTool))
	if err != nil {
		return "", "", err
	}
	if view == "info" {
		return out, "application/json", nil
	}
	return out, "text/plain", nil
}
