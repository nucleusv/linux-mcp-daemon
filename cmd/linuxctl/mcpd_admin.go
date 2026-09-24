package main

// linuxctl's "mcpd" group edits users.yaml + mcp-sudo.yaml directly on
// disk, locally. This is a security boundary, not a shortcut - user/token
// administration is a privilege-escalation-relevant surface, and there is
// deliberately no MCP tool that edits these files, so no remote agent can
// ever reach it, regardless of what any bearer token is authorized for.
// The only network call is the one after a change: asking the running
// daemon to re-read its files (daemon/reload-config), which changes
// nothing on disk.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/yamledit"
	"gopkg.in/yaml.v3"
)

const (
	mcpdAdminTokenBytes = 32
	mcpdAdminSaltBytes  = 16
)

func generateHexSecret(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		fmt.Printf("Error: failed to generate random secret: %v\n", err)
		os.Exit(1)
	}
	return hex.EncodeToString(b)
}

func hashToken(token, salt string) string {
	sum := sha256.Sum256([]byte(salt + token))
	return hex.EncodeToString(sum[:])
}

func loadYAMLDoc(path string) (*yaml.Node, error) {
	doc, err := yamledit.LoadDoc(path)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s is empty or not a valid YAML document", path)
	}
	return doc, nil
}

func daemonUsersSeq(root *yaml.Node, create bool) *yaml.Node {
	seq := yamledit.MapGet(root, "users")
	if seq == nil && create {
		seq = &yaml.Node{Kind: yaml.SequenceNode}
		yamledit.MapSet(root, "users", seq)
	}
	if seq != nil && create {
		// An empty list can only be written "users: []" (flow style);
		// entries appended to it would inherit that and collapse onto one
		// line. Switch it to block style so each user gets readable lines.
		seq.Style &^= yaml.FlowStyle
	}
	return seq
}

func daemonFindUser(seq *yaml.Node, username string) *yaml.Node {
	if seq == nil {
		return nil
	}
	for _, item := range seq.Content {
		if u := yamledit.MapGet(item, "username"); u != nil && u.Value == username {
			return item
		}
	}
	return nil
}

// usersFileFor returns users.yaml in cfgPath, first moving a legacy users:
// list out of daemon.yaml into it if there is one. Every command that
// writes users goes through this, so configs migrate on the first change.
func usersFileFor(cfgPath string) string {
	usersPath := filepath.Join(cfgPath, "users.yaml")
	if _, err := os.Stat(usersPath); err == nil {
		return usersPath
	}
	daemonPath := filepath.Join(cfgPath, "daemon.yaml")
	daemonDoc, err := loadYAMLDoc(daemonPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	legacy := daemonUsersSeq(daemonDoc.Content[0], false)

	usersDoc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{yamledit.EmptyMapNode()}}
	root := usersDoc.Content[0]
	root.Style &^= yaml.FlowStyle
	root.HeadComment = "mcpd users: who may connect, and with which token (only a salted hash is\n" +
		"stored). Managed with `linuxctl create|update|delete mcpd user`. Each user\n" +
		"must also exist as an OS account: tools run as that account. What a user\n" +
		"may do as root is set in mcp-sudo.yaml."
	seq := legacy
	if seq == nil {
		seq = &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
	}
	yamledit.MapSet(root, "users", seq)

	// users.yaml holds token hashes: create it readable by its owner only,
	// before anything is written to it.
	if err := os.WriteFile(usersPath, nil, 0600); err != nil {
		fmt.Printf("Error creating %s: %v\n", usersPath, err)
		os.Exit(1)
	}
	if err := yamledit.SaveDoc(usersPath, usersDoc); err != nil {
		fmt.Printf("Error writing %s: %v\n", usersPath, err)
		os.Exit(1)
	}
	if legacy != nil {
		yamledit.MapDelete(daemonDoc.Content[0], "users")
		if err := yamledit.SaveDoc(daemonPath, daemonDoc); err != nil {
			fmt.Printf("Error writing %s: %v\n", daemonPath, err)
			os.Exit(1)
		}
		fmt.Printf("Moved the users list from %s to %s.\n", daemonPath, usersPath)
	}
	return usersPath
}

// usersFileForReading returns the file the users are in, without
// migrating anything: users.yaml, or daemon.yaml in older configs.
func usersFileForReading(cfgPath string) string {
	usersPath := filepath.Join(cfgPath, "users.yaml")
	if _, err := os.Stat(usersPath); err == nil {
		return usersPath
	}
	return filepath.Join(cfgPath, "daemon.yaml")
}

// --- command handlers ---

func handleMcpdAdmin(args []string) {
	if len(args) < 2 {
		printMcpdAdminUsage()
		os.Exit(1)
	}
	verb, target, rest := args[0], args[1], args[2:]
	isConfig := verb == "edit" && target == "config"
	if !strings.HasPrefix(target, "user") && !isConfig {
		fmt.Printf("Error: unknown mcpd target %q (expected \"user\", \"users\" or, for edit, \"config\")\n", target)
		os.Exit(1)
	}

	var username, setToken string
	var grants []string
	cfgPath := *configPath
	reload := true
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--set-token":
			if i+1 < len(rest) {
				setToken = rest[i+1]
				i++
			}
		case "--config-path":
			if i+1 < len(rest) {
				cfgPath = rest[i+1]
				i++
			}
		case "--no-reload":
			reload = false
		case "--grant":
			if i+1 < len(rest) {
				for _, g := range strings.Split(rest[i+1], ",") {
					if g = strings.TrimSpace(g); g != "" {
						grants = append(grants, g)
					}
				}
				i++
			}
		default:
			if !strings.HasPrefix(rest[i], "-") && username == "" {
				username = rest[i]
			}
		}
	}

	sudoPath := filepath.Join(cfgPath, "mcp-sudo.yaml")

	switch verb {
	case "edit":
		if !isConfig {
			fmt.Println("Error: expected `linuxctl edit mcpd config sudo|users|daemon`")
			os.Exit(1)
		}
		which := username // the positional argument
		if which == "" {
			which = "sudo"
		}
		if !mcpdEditConfig(cfgPath, which) {
			return
		}
	case "create":
		mcpdCreateUser(usersFileFor(cfgPath), sudoPath, username, setToken, grants)
	case "delete":
		mcpdDeleteUser(usersFileFor(cfgPath), sudoPath, username)
	case "update":
		mcpdUpdateUser(usersFileFor(cfgPath), username, setToken)
	case "list":
		mcpdListUsers(usersFileForReading(cfgPath))
		return
	case "describe":
		mcpdDescribeUser(sudoPath, username)
		return
	default:
		fmt.Printf("Error: unknown mcpd verb %q (expected create, delete, update, list, describe or edit)\n", verb)
		os.Exit(1)
	}
	if reload {
		reloadDaemon()
	} else {
		fmt.Println("\nNot applied yet (--no-reload). Apply with: linuxctl reload daemon")
	}
}

func printMcpdAdminUsage() {
	fmt.Println(`Usage:
  linuxctl create   mcpd user <username> [--set-token VALUE] [--grant TOOL,...] [--config-path DIR] [--no-reload]
  linuxctl delete   mcpd user <username> [--config-path DIR] [--no-reload]
  linuxctl update   mcpd user <username> [--set-token VALUE] [--config-path DIR] [--no-reload]
  linuxctl list     mcpd users           [--config-path DIR]
  linuxctl describe mcpd user <username> [--config-path DIR]
  linuxctl edit     mcpd config [sudo|users|daemon] [--config-path DIR] [--no-reload]

Edits users.yaml + mcp-sudo.yaml locally (--config-path, default ./configs).
edit opens the file in $VISUAL/$EDITOR (like visudo) and saves it only if
it passes the same strict check mcpd applies. After a change, asks the running mcpd (MCP_SERVER, MCP_TOKEN)
to re-read its config with daemon/reload-config; --no-reload skips that.`)
}

func mcpdCreateUser(usersPath, sudoPath, username, setToken string, grants []string) {
	if username == "" {
		fmt.Println("Error: username required")
		os.Exit(1)
	}

	doc, err := loadYAMLDoc(usersPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	root := doc.Content[0]
	seq := daemonUsersSeq(root, true)
	if daemonFindUser(seq, username) != nil {
		fmt.Printf("Error: user %q already exists in %s\n", username, usersPath)
		os.Exit(1)
	}

	generated := setToken == ""
	token := setToken
	if generated {
		token = generateHexSecret(mcpdAdminTokenBytes)
	}
	salt := generateHexSecret(mcpdAdminSaltBytes)
	hash := hashToken(token, salt)
	createdAt := time.Now().UTC().Format(time.RFC3339)

	entry := yamledit.EmptyMapNode()
	yamledit.MapSet(entry, "username", yamledit.ScalarNode(username))
	yamledit.MapSet(entry, "token_salt", yamledit.ScalarNode(salt))
	yamledit.MapSet(entry, "token_hash", yamledit.ScalarNode(hash))
	yamledit.MapSet(entry, "created_at", yamledit.ScalarNode(createdAt))
	// pinned_uid/os_uid are deliberately left unset here, not set to a
	// guessed value - they get populated by mcpd itself, trust-on-first-use,
	// the first time this user successfully authenticates against a real
	// running daemon (see cmd/mcpd/uid_pin.go). linuxctl can't safely set
	// them at create time even if it wanted to: the OS account this
	// username will map to often doesn't exist yet (create → useradd in the
	// Dockerfile → rebuild is still an unavoidable three-step process - see
	// the "Next steps" printed below), and even when it does exist locally,
	// linuxctl may be running on a different machine entirely from wherever
	// mcpd is actually deployed. Only the daemon, at the moment it resolves
	// user.Lookup() for a real authenticated request, knows the UID that
	// actually matters.
	seq.Content = append(seq.Content, entry)

	if err := yamledit.SaveDoc(usersPath, doc); err != nil {
		fmt.Printf("Error writing %s: %v\n", usersPath, err)
		os.Exit(1)
	}

	sudoDoc, err := loadYAMLDoc(sudoPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	sudoRoot := sudoDoc.Content[0]
	usersMap := yamledit.MapGet(sudoRoot, "users")
	if usersMap == nil {
		usersMap = yamledit.EmptyMapNode()
		yamledit.MapSet(sudoRoot, "users", usersMap)
	}
	usersMap.Style &^= yaml.FlowStyle // "users: {}" -> block, as for users.yaml
	if yamledit.MapGet(usersMap, username) != nil {
		fmt.Printf("Note: %s had a stale entry for %q - replacing it with a fresh, empty grant block (this is the point of create/delete being atomic: a recreated user never inherits old grants)\n", sudoPath, username)
	}
	freshBlock := yamledit.EmptyMapNode()
	freshBlock.Style &^= yaml.FlowStyle
	privileged := yamledit.EmptyMapNode()
	privileged.Style &^= yaml.FlowStyle
	tools := yamledit.EmptyMapNode()
	if len(grants) > 0 {
		tools.Style &^= yaml.FlowStyle
	}
	for _, g := range grants {
		grant := yamledit.EmptyMapNode()
		yamledit.MapSet(grant, "allowed", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
		yamledit.MapSet(tools, g, grant)
	}
	yamledit.MapSet(privileged, "tools", tools)
	yamledit.MapSet(privileged, "resources", yamledit.EmptyMapNode())
	yamledit.MapSet(freshBlock, "privileged", privileged)
	yamledit.MapSet(usersMap, username, freshBlock)

	if err := yamledit.SaveDoc(sudoPath, sudoDoc); err != nil {
		fmt.Printf("Error writing %s: %v\n", sudoPath, err)
		os.Exit(1)
	}

	fmt.Printf("Created mcpd user %q.\n", username)
	if len(grants) > 0 {
		fmt.Printf("Granted as root: %s\n", strings.Join(grants, ", "))
	}
	if generated {
		fmt.Printf("\nToken (shown once - not stored in plaintext anywhere, save it now):\n  %s\n\n", token)
	}
	fmt.Println("Next steps:")
	fmt.Printf("  1. Make sure an OS account %q exists on the mcpd host - each call runs as that account:\n       useradd --system --shell /usr/sbin/nologin %s\n     (for the container image, add it to the Dockerfile instead)\n", username, username)
	fmt.Printf("  2. Optionally grant root for specific tools in %s - without grants %q can use\n     every tool, but only as its own OS account, never as root\n", sudoPath, username)
}

func mcpdDeleteUser(usersPath, sudoPath, username string) {
	if username == "" {
		fmt.Println("Error: username required")
		os.Exit(1)
	}

	doc, err := loadYAMLDoc(usersPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	root := doc.Content[0]
	seq := daemonUsersSeq(root, false)
	removed := false
	if seq != nil {
		for i, item := range seq.Content {
			if u := yamledit.MapGet(item, "username"); u != nil && u.Value == username {
				seq.Content = append(seq.Content[:i], seq.Content[i+1:]...)
				removed = true
				break
			}
		}
	}
	if !removed {
		fmt.Printf("Error: user %q not found in %s\n", username, usersPath)
		os.Exit(1)
	}
	if err := yamledit.SaveDoc(usersPath, doc); err != nil {
		fmt.Printf("Error writing %s: %v\n", usersPath, err)
		os.Exit(1)
	}

	sudoDoc, err := loadYAMLDoc(sudoPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	sudoRoot := sudoDoc.Content[0]
	if usersMap := yamledit.MapGet(sudoRoot, "users"); usersMap != nil {
		yamledit.MapDelete(usersMap, username)
	}
	if err := yamledit.SaveDoc(sudoPath, sudoDoc); err != nil {
		fmt.Printf("Error writing %s: %v\n", sudoPath, err)
		os.Exit(1)
	}

	fmt.Printf("Deleted mcpd user %q from %s and %s.\n", username, usersPath, sudoPath)
	fmt.Println("Remove the OS account too if it's no longer needed (for the container image: its useradd line in the Dockerfile).")
}

func mcpdUpdateUser(usersPath, username, setToken string) {
	if username == "" {
		fmt.Println("Error: username required")
		os.Exit(1)
	}

	doc, err := loadYAMLDoc(usersPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	root := doc.Content[0]
	seq := daemonUsersSeq(root, false)
	entry := daemonFindUser(seq, username)
	if entry == nil {
		fmt.Printf("Error: user %q not found in %s (use create instead)\n", username, usersPath)
		os.Exit(1)
	}

	generated := setToken == ""
	token := setToken
	if generated {
		token = generateHexSecret(mcpdAdminTokenBytes)
	}
	salt := generateHexSecret(mcpdAdminSaltBytes)
	hash := hashToken(token, salt)

	// Migrates a legacy plaintext `token:` entry (pre-dating salted-hash
	// storage) the same way a normal rotation does - drop the old field,
	// write the new ones. created_at is left untouched if already present,
	// so a rotation doesn't look like a brand-new account in `list`.
	yamledit.MapDelete(entry, "token")
	yamledit.MapSet(entry, "token_salt", yamledit.ScalarNode(salt))
	yamledit.MapSet(entry, "token_hash", yamledit.ScalarNode(hash))
	if yamledit.MapGet(entry, "created_at") == nil {
		yamledit.MapSet(entry, "created_at", yamledit.ScalarNode(time.Now().UTC().Format(time.RFC3339)))
	}

	// Rotating the token is treated as "I am consciously re-provisioning
	// this identity" - the same signal that should reset the pinned OS UID
	// (see cmd/mcpd/uid_pin.go). Without this, rotating a token for a
	// legitimately reissued OS account (e.g. after a deliberate rebuild)
	// would leave a stale pin in place that the next real request then
	// immediately - and incorrectly - trips as a security mismatch.
	//
	// SECURITY NOTE / KNOWN LIMITATION: clearing the pin here re-arms
	// trust-on-first-use, which is necessary for legitimate re-provisioning
	// but is exactly as strong as TOFU ever is - if an attacker can rotate
	// this token (e.g. by compromising whoever runs linuxctl locally) *and*
	// control which OS account authenticates next, they get a fresh, silently
	// accepted pin. This is a deliberate, bounded trust: linuxctl already
	// requires local filesystem access to configs/, which this project
	// treats as the trust boundary for all `mcpd` admin operations (see the
	// package comment above) - it does not add a new one.
	uidWasPinned := yamledit.MapDelete(entry, "pinned_uid")
	yamledit.MapDelete(entry, "os_uid")

	if err := yamledit.SaveDoc(usersPath, doc); err != nil {
		fmt.Printf("Error writing %s: %v\n", usersPath, err)
		os.Exit(1)
	}

	fmt.Printf("Rotated token for mcpd user %q.\n", username)
	if uidWasPinned {
		fmt.Println("Cleared its pinned OS UID - mcpd will re-pin whatever UID this username resolves to on its next successful call.")
	}
	if generated {
		fmt.Printf("\nNew token (shown once - not stored in plaintext anywhere, save it now):\n  %s\n\n", token)
	}
	fmt.Println("The old token stops working as soon as mcpd reloads its config.")
}

func mcpdListUsers(usersPath string) {
	doc, err := loadYAMLDoc(usersPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	root := doc.Content[0]
	seq := daemonUsersSeq(root, false)
	if seq == nil || len(seq.Content) == 0 {
		fmt.Println("No mcpd users configured.")
		return
	}
	fmt.Printf("%-20s %-25s %-30s %s\n", "USERNAME", "CREATED_AT", "AUTH", "PINNED_UID / OS_UID")
	for _, item := range seq.Content {
		username := "?"
		if u := yamledit.MapGet(item, "username"); u != nil {
			username = u.Value
		}
		createdAt := "-"
		if c := yamledit.MapGet(item, "created_at"); c != nil {
			createdAt = c.Value
		}
		auth := "salted-hash"
		if yamledit.MapGet(item, "token") != nil {
			auth = "PLAINTEXT (legacy - run update to migrate)"
		}
		pinned := "-"
		if p := yamledit.MapGet(item, "pinned_uid"); p != nil {
			pinned = p.Value
		}
		observed := "-"
		if o := yamledit.MapGet(item, "os_uid"); o != nil {
			observed = o.Value
		}
		uidStatus := fmt.Sprintf("%s / %s", pinned, observed)
		if pinned != "-" && observed != "-" && pinned != observed {
			uidStatus += "  ⚠ MISMATCH"
		}
		fmt.Printf("%-20s %-25s %-30s %s\n", username, createdAt, auth, uidStatus)
	}
}

func mcpdDescribeUser(sudoPath, username string) {
	if username == "" {
		fmt.Println("Error: username required")
		os.Exit(1)
	}
	doc, err := loadYAMLDoc(sudoPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	root := doc.Content[0]
	usersMap := yamledit.MapGet(root, "users")
	entry := yamledit.MapGet(usersMap, username)
	if entry == nil {
		fmt.Printf("No mcp-sudo.yaml grants found for %q (denies everything by default).\n", username)
		return
	}
	out, err := yaml.Marshal(entry)
	if err != nil {
		fmt.Printf("Error encoding grants: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Grants for %q (from %s):\n%s", username, sudoPath, string(out))
}
