package main

// linuxctl's "mcpd" group is deliberately local-only: every command here
// reads and writes configs/daemon.yaml + configs/mcp-sudo.yaml directly on
// disk and never makes a network call to the running daemon. This is a
// security boundary, not a shortcut - user/token administration is a
// privilege-escalation-relevant surface, and keeping it off the network
// tools/call path means no remote agent can ever reach it, regardless of
// what any bearer token is authorized for.

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

// --- command handlers ---

func handleMcpdAdmin(args []string) {
	if len(args) < 2 {
		printMcpdAdminUsage()
		os.Exit(1)
	}
	verb, target, rest := args[0], args[1], args[2:]
	if !strings.HasPrefix(target, "user") {
		fmt.Printf("Error: unknown mcpd target %q (expected \"user\" or \"users\")\n", target)
		os.Exit(1)
	}

	var username, setToken string
	cfgPath := *configPath
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
		default:
			if !strings.HasPrefix(rest[i], "-") && username == "" {
				username = rest[i]
			}
		}
	}

	daemonPath := filepath.Join(cfgPath, "daemon.yaml")
	sudoPath := filepath.Join(cfgPath, "mcp-sudo.yaml")

	switch verb {
	case "create":
		mcpdCreateUser(daemonPath, sudoPath, username, setToken)
	case "delete":
		mcpdDeleteUser(daemonPath, sudoPath, username)
	case "update":
		mcpdUpdateUser(daemonPath, username, setToken)
	case "list":
		mcpdListUsers(daemonPath)
	case "describe":
		mcpdDescribeUser(sudoPath, username)
	default:
		fmt.Printf("Error: unknown mcpd verb %q (expected create, delete, update, list, or describe)\n", verb)
		os.Exit(1)
	}
}

func printMcpdAdminUsage() {
	fmt.Println(`Usage:
  linuxctl create   mcpd user <username> [--set-token VALUE] [--config-path DIR]
  linuxctl delete   mcpd user <username> [--config-path DIR]
  linuxctl update   mcpd user <username> [--set-token VALUE] [--config-path DIR]
  linuxctl list     mcpd users           [--config-path DIR]
  linuxctl describe mcpd user <username> [--config-path DIR]

All local-only: reads/writes daemon.yaml + mcp-sudo.yaml directly, no
network call to mcpd. --config-path defaults to ./configs.`)
}

func mcpdCreateUser(daemonPath, sudoPath, username, setToken string) {
	if username == "" {
		fmt.Println("Error: username required")
		os.Exit(1)
	}

	doc, err := loadYAMLDoc(daemonPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	root := doc.Content[0]
	seq := daemonUsersSeq(root, true)
	if daemonFindUser(seq, username) != nil {
		fmt.Printf("Error: user %q already exists in %s\n", username, daemonPath)
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

	if err := yamledit.SaveDoc(daemonPath, doc); err != nil {
		fmt.Printf("Error writing %s: %v\n", daemonPath, err)
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
	if yamledit.MapGet(usersMap, username) != nil {
		fmt.Printf("Note: %s had a stale entry for %q - replacing it with a fresh, empty grant block (this is the point of create/delete being atomic: a recreated user never inherits old grants)\n", sudoPath, username)
	}
	freshBlock := yamledit.EmptyMapNode()
	privileged := yamledit.EmptyMapNode()
	yamledit.MapSet(privileged, "tools", yamledit.EmptyMapNode())
	yamledit.MapSet(privileged, "resources", yamledit.EmptyMapNode())
	yamledit.MapSet(freshBlock, "privileged", privileged)
	yamledit.MapSet(usersMap, username, freshBlock)

	if err := yamledit.SaveDoc(sudoPath, sudoDoc); err != nil {
		fmt.Printf("Error writing %s: %v\n", sudoPath, err)
		os.Exit(1)
	}

	fmt.Printf("Created mcpd user %q.\n", username)
	if generated {
		fmt.Printf("\nToken (shown once - not stored in plaintext anywhere, save it now):\n  %s\n\n", token)
	}
	fmt.Println("Next steps:")
	fmt.Printf("  1. Add a matching OS account (useradd -m -s /bin/bash %s in the Dockerfile) - workers run\n     as a real OS user via user.Lookup(), so this account must exist before %s can make any call.\n", username, username)
	fmt.Printf("  2. Grant tools/resources for %q in %s (currently empty - denies everything by default)\n", username, sudoPath)
	fmt.Println("  3. Run scripts/deploy.sh to rebuild and apply")
}

func mcpdDeleteUser(daemonPath, sudoPath, username string) {
	if username == "" {
		fmt.Println("Error: username required")
		os.Exit(1)
	}

	doc, err := loadYAMLDoc(daemonPath)
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
		fmt.Printf("Error: user %q not found in %s\n", username, daemonPath)
		os.Exit(1)
	}
	if err := yamledit.SaveDoc(daemonPath, doc); err != nil {
		fmt.Printf("Error writing %s: %v\n", daemonPath, err)
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

	fmt.Printf("Deleted mcpd user %q from %s and %s.\n", username, daemonPath, sudoPath)
	fmt.Println("Run scripts/deploy.sh to apply. Remove the matching useradd line from the Dockerfile too if this account is no longer needed at all.")
}

func mcpdUpdateUser(daemonPath, username, setToken string) {
	if username == "" {
		fmt.Println("Error: username required")
		os.Exit(1)
	}

	doc, err := loadYAMLDoc(daemonPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	root := doc.Content[0]
	seq := daemonUsersSeq(root, false)
	entry := daemonFindUser(seq, username)
	if entry == nil {
		fmt.Printf("Error: user %q not found in %s (use create instead)\n", username, daemonPath)
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

	if err := yamledit.SaveDoc(daemonPath, doc); err != nil {
		fmt.Printf("Error writing %s: %v\n", daemonPath, err)
		os.Exit(1)
	}

	fmt.Printf("Rotated token for mcpd user %q.\n", username)
	if uidWasPinned {
		fmt.Println("Cleared its pinned OS UID - mcpd will re-pin whatever UID this username resolves to on its next successful call.")
	}
	if generated {
		fmt.Printf("\nNew token (shown once - not stored in plaintext anywhere, save it now):\n  %s\n\n", token)
	}
	fmt.Println("Run scripts/deploy.sh to apply. The old token stops working the moment this is deployed.")
}

func mcpdListUsers(daemonPath string) {
	doc, err := loadYAMLDoc(daemonPath)
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
