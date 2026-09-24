package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ToolsConfig holds per-tool overrides from daemon.yaml's tools: block.
type ToolsConfig map[string]struct {
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

// DaemonConfig is daemon.yaml.
type DaemonConfig struct {
	Server struct {
		Port int `yaml:"port"`
		TLS  struct {
			Enabled  bool   `yaml:"enabled"`
			Port     int    `yaml:"port"`
			CertFile string `yaml:"cert_file"`
			KeyFile  string `yaml:"key_file"`
		} `yaml:"tls"`
	} `yaml:"server"`
	RateLimits struct {
		DefaultRPS   float64 `yaml:"default_rps"`
		DefaultBurst int     `yaml:"default_burst"`
	} `yaml:"rate_limits"`
	Worker struct {
		TimeoutSeconds int `yaml:"timeout_seconds"`
		// Containerized indicates this daemon process itself runs inside a
		// container with its own private root filesystem (e.g. Kubernetes),
		// as opposed to running directly on the host with no container
		// boundary. When true, every privileged (root) worker call also
		// joins the real host's mount namespace before running - see
		// internal/worker/hostns.go. Set false when mcpd runs directly on
		// the host.
		Containerized bool `yaml:"containerized"`
	} `yaml:"worker"`
	Tools ToolsConfig `yaml:"tools"`
	// Users is the legacy location of the user list; it now lives in
	// users.yaml (see LoadConfigDir), which is read instead when present.
	Users []DaemonUser `yaml:"users"`
}

// DaemonUser is one mcpd user: a name and its token (users.yaml).
type DaemonUser struct {
	Username string `yaml:"username"`
	// Token is the legacy plaintext field. New/rotated users
	// (via `linuxctl create|update mcpd user`) use TokenSalt+TokenHash
	// instead - see authenticateRequest in cmd/mcpd/http.go. Both are supported
	// simultaneously so migration doesn't require a hard cutover.
	Token     string `yaml:"token,omitempty"`
	TokenSalt string `yaml:"token_salt,omitempty"`
	TokenHash string `yaml:"token_hash,omitempty"`
	CreatedAt string `yaml:"created_at,omitempty"`
	// PinnedUID and OSUID implement trust-on-first-use OS identity
	// pinning - see cmd/mcpd/uid_pin.go for the full mechanism, rationale,
	// and its known limitation (it does not reliably catch a username
	// being reused for a different real person if the OS happens to
	// reissue the exact same UID, which useradd's gap-filling behavior
	// makes plausible - see ARCHITECTURE.md). Never set these fields by
	// hand; they're written only by the daemon itself.
	PinnedUID string `yaml:"pinned_uid,omitempty"`
	OSUID     string `yaml:"os_uid,omitempty"`
}

// UsersConfig is users.yaml: who may connect, and with which token. It's
// kept apart from daemon.yaml because it's the one file holding secrets
// (token hashes) and the file user administration rewrites.
type UsersConfig struct {
	Users []DaemonUser `yaml:"users"`
}

// Config file names inside the config directory.
const (
	DaemonFile = "daemon.yaml"
	UsersFile  = "users.yaml"
	SudoFile   = "mcp-sudo.yaml"
)

// LoadConfigDir loads daemon.yaml together with the user list: from
// users.yaml, or - for configs predating it - from daemon.yaml's own
// users: list. usersPath is the file the users came from, which is where
// updates to them (UID pins) must go. Users in both files is an error:
// it's ambiguous which list is meant.
func LoadConfigDir(dir string, strict bool) (c DaemonConfig, usersPath string, err error) {
	daemonPath := filepath.Join(dir, DaemonFile)
	c, err = LoadDaemonConfig(daemonPath, strict)
	if err != nil {
		return c, "", err
	}
	usersPath = filepath.Join(dir, UsersFile)
	data, err := os.ReadFile(usersPath)
	if errors.Is(err, fs.ErrNotExist) {
		return c, daemonPath, nil
	}
	if err != nil {
		return c, "", err
	}
	u, err := ParseUsersConfig(data, strict)
	if err != nil {
		return c, "", fmt.Errorf("%s: %w", usersPath, err)
	}
	if len(c.Users) > 0 {
		return c, "", fmt.Errorf("users are listed in both %s and %s - move any missing from %s into it, then delete users: from %s", daemonPath, usersPath, UsersFile, DaemonFile)
	}
	c.Users = u.Users
	return c, usersPath, nil
}

// ParseUsersConfig parses and validates users.yaml; strict rejects
// unknown keys.
func ParseUsersConfig(data []byte, strict bool) (UsersConfig, error) {
	var u UsersConfig
	if err := decodeYAML(data, &u, strict); err != nil {
		return u, err
	}
	return u, validateUsers(u.Users)
}

// LoadDaemonConfig reads and validates daemon.yaml. See ParseDaemonConfig.
func LoadDaemonConfig(path string, strict bool) (DaemonConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DaemonConfig{}, err
	}
	c, err := ParseDaemonConfig(data, strict)
	if err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// ParseDaemonConfig parses daemon.yaml, fills in defaults and validates the
// users list. strict rejects unknown keys, so a misspelled key is an error
// instead of a setting silently left at its default.
func ParseDaemonConfig(data []byte, strict bool) (DaemonConfig, error) {
	var c DaemonConfig
	if err := decodeYAML(data, &c, strict); err != nil {
		return c, err
	}
	if c.Server.Port == 0 {
		c.Server.Port = 9091
	}
	if c.Server.TLS.Enabled && c.Server.TLS.Port == 0 {
		c.Server.TLS.Port = 9443
	}
	if c.Worker.TimeoutSeconds == 0 {
		c.Worker.TimeoutSeconds = 30
	}
	return c, validateUsers(c.Users)
}

func validateUsers(users []DaemonUser) error {
	seen := map[string]bool{}
	for _, u := range users {
		if u.Username == "" {
			return fmt.Errorf("a user has no username")
		}
		if seen[u.Username] {
			return fmt.Errorf("user %q is listed twice", u.Username)
		}
		seen[u.Username] = true
		if u.Token == "" && u.TokenHash == "" {
			return fmt.Errorf("user %q has no token (token_hash)", u.Username)
		}
		if u.TokenHash != "" && u.TokenSalt == "" {
			return fmt.Errorf("user %q has token_hash but no token_salt", u.Username)
		}
	}
	return nil
}

// decodeYAML unmarshals one YAML document; strict rejects keys the target
// struct doesn't have. An empty document decodes to the zero value.
func decodeYAML(data []byte, out interface{}, strict bool) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(strict)
	if err := dec.Decode(out); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
