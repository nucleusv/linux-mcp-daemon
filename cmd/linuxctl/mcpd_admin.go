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

// --- minimal yaml.Node helpers (gopkg.in/yaml.v3) ---
// These edit the parsed document tree surgically so that comments and
// formatting elsewhere in the file (e.g. daemon.yaml's worker: section) are
// preserved - a plain struct unmarshal/marshal round trip would silently
// drop every comment in the file.

func mapGet(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func mapSet(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
}

func mapDelete(mapping *yaml.Node, key string) bool {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return true
		}
	}
	return false
}

func scalarNode(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: v}
}

func emptyMapNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode}
}

func loadYAMLDoc(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s is empty or not a valid YAML document", path)
	}
	return &doc, nil
}

func saveYAMLDoc(path string, doc *yaml.Node) error {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2) // matches this project's existing YAML style (yaml.v3 defaults to 4)
	if err := enc.Encode(doc); err != nil {
		enc.Close()
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(buf.String()), 0644)
}

func daemonUsersSeq(root *yaml.Node, create bool) *yaml.Node {
	seq := mapGet(root, "users")
	if seq == nil && create {
		seq = &yaml.Node{Kind: yaml.SequenceNode}
		mapSet(root, "users", seq)
	}
	return seq
}

func daemonFindUser(seq *yaml.Node, username string) *yaml.Node {
	if seq == nil {
		return nil
	}
	for _, item := range seq.Content {
		if u := mapGet(item, "username"); u != nil && u.Value == username {
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

	entry := emptyMapNode()
	mapSet(entry, "username", scalarNode(username))
	mapSet(entry, "token_salt", scalarNode(salt))
	mapSet(entry, "token_hash", scalarNode(hash))
	mapSet(entry, "created_at", scalarNode(createdAt))
	seq.Content = append(seq.Content, entry)

	if err := saveYAMLDoc(daemonPath, doc); err != nil {
		fmt.Printf("Error writing %s: %v\n", daemonPath, err)
		os.Exit(1)
	}

	sudoDoc, err := loadYAMLDoc(sudoPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	sudoRoot := sudoDoc.Content[0]
	usersMap := mapGet(sudoRoot, "users")
	if usersMap == nil {
		usersMap = emptyMapNode()
		mapSet(sudoRoot, "users", usersMap)
	}
	if mapGet(usersMap, username) != nil {
		fmt.Printf("Note: %s had a stale entry for %q - replacing it with a fresh, empty grant block (this is the point of create/delete being atomic: a recreated user never inherits old grants)\n", sudoPath, username)
	}
	freshBlock := emptyMapNode()
	privileged := emptyMapNode()
	mapSet(privileged, "tools", emptyMapNode())
	mapSet(privileged, "resources", emptyMapNode())
	mapSet(freshBlock, "privileged", privileged)
	mapSet(usersMap, username, freshBlock)

	if err := saveYAMLDoc(sudoPath, sudoDoc); err != nil {
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
			if u := mapGet(item, "username"); u != nil && u.Value == username {
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
	if err := saveYAMLDoc(daemonPath, doc); err != nil {
		fmt.Printf("Error writing %s: %v\n", daemonPath, err)
		os.Exit(1)
	}

	sudoDoc, err := loadYAMLDoc(sudoPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	sudoRoot := sudoDoc.Content[0]
	if usersMap := mapGet(sudoRoot, "users"); usersMap != nil {
		mapDelete(usersMap, username)
	}
	if err := saveYAMLDoc(sudoPath, sudoDoc); err != nil {
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
	mapDelete(entry, "token")
	mapSet(entry, "token_salt", scalarNode(salt))
	mapSet(entry, "token_hash", scalarNode(hash))
	if mapGet(entry, "created_at") == nil {
		mapSet(entry, "created_at", scalarNode(time.Now().UTC().Format(time.RFC3339)))
	}

	if err := saveYAMLDoc(daemonPath, doc); err != nil {
		fmt.Printf("Error writing %s: %v\n", daemonPath, err)
		os.Exit(1)
	}

	fmt.Printf("Rotated token for mcpd user %q.\n", username)
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
	fmt.Printf("%-20s %-25s %s\n", "USERNAME", "CREATED_AT", "AUTH")
	for _, item := range seq.Content {
		username := "?"
		if u := mapGet(item, "username"); u != nil {
			username = u.Value
		}
		createdAt := "-"
		if c := mapGet(item, "created_at"); c != nil {
			createdAt = c.Value
		}
		auth := "salted-hash"
		if mapGet(item, "token") != nil {
			auth = "PLAINTEXT (legacy - run update to migrate)"
		}
		fmt.Printf("%-20s %-25s %s\n", username, createdAt, auth)
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
	usersMap := mapGet(root, "users")
	entry := mapGet(usersMap, username)
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
