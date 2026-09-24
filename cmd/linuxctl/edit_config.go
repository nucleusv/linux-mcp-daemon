package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
)

// editConfigFiles maps `linuxctl edit mcpd config <which>` to a file name.
var editConfigFiles = map[string]string{
	"sudo":   config.SudoFile,
	"users":  config.UsersFile,
	"daemon": config.DaemonFile,
}

// mcpdEditConfig edits one mcpd config file the way visudo edits sudoers:
// on a temporary copy, validated with the same strict parser mcpd uses on
// reload before it replaces the real file. An invalid edit never reaches
// the real file - the choice is to edit again or discard it.
func mcpdEditConfig(cfgPath, which string) bool {
	name, ok := editConfigFiles[which]
	if !ok {
		fmt.Printf("Error: unknown config %q (expected sudo, users or daemon)\n", which)
		os.Exit(1)
	}
	path := filepath.Join(cfgPath, name)
	if which == "users" {
		path = usersFileFor(cfgPath) // creates it, moving a legacy list
	}
	orig, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// The copy sits next to the real file, so the final rename is atomic
	// (same filesystem), and is readable only by its owner - users.yaml
	// holds token hashes.
	tmp, err := os.CreateTemp(cfgPath, "."+name+".edit-*")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(orig); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	tmp.Close()

	for {
		if err := runEditor(tmpPath); err != nil {
			fmt.Printf("Error: %v - %s left unchanged\n", err, path)
			os.Exit(1)
		}
		edited, err := os.ReadFile(tmpPath)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if bytes.Equal(edited, orig) {
			fmt.Printf("No changes to %s.\n", path)
			return false
		}
		warnings, err := validateConfigEdit(cfgPath, which, edited)
		if err != nil {
			fmt.Printf("%s is not valid: %v\n", name, err)
			if askEditAgain() {
				continue
			}
			fmt.Printf("Discarded the edit - %s left unchanged.\n", path)
			os.Exit(1) // the edit asked for didn't happen
		}
		for _, w := range warnings {
			fmt.Printf("Warning: %s\n", w)
		}

		if err := os.Chmod(tmpPath, info.Mode().Perm()); err != nil {
			fmt.Printf("Error: %v - %s left unchanged\n", err, path)
			os.Exit(1)
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			// Keep the file's owner; only possible (and only needed) as root.
			_ = os.Chown(tmpPath, int(st.Uid), int(st.Gid))
		}
		if err := os.Rename(tmpPath, path); err != nil {
			fmt.Printf("Error: %v - %s left unchanged\n", err, path)
			os.Exit(1)
		}
		fmt.Printf("Saved %s.\n", path)
		return true
	}
}

// runEditor opens path in $VISUAL, $EDITOR or vi, attached to the terminal.
func runEditor(path string) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	// $EDITOR may carry arguments ("code --wait").
	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %q: %w", editor, err)
	}
	return nil
}

func askEditAgain() bool {
	fmt.Print("(e)dit again or (d)iscard the edit? [e] ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "" || answer == "e"
}

// validateConfigEdit checks an edited file with the parser mcpd itself
// uses on reload (strict: unknown keys are errors), plus how it fits the
// other files. Warnings are consistent-but-suspicious findings.
func validateConfigEdit(cfgPath, which string, data []byte) (warnings []string, err error) {
	var users []string
	haveUsers := false
	var sudo *config.SudoConfig
	switch which {
	case "daemon":
		c, err := config.ParseDaemonConfig(data, true)
		if err != nil {
			return nil, err
		}
		if len(c.Users) > 0 {
			if _, statErr := os.Stat(filepath.Join(cfgPath, config.UsersFile)); statErr == nil {
				return nil, fmt.Errorf("users: belongs in %s now, not in %s", config.UsersFile, config.DaemonFile)
			}
		}
		return nil, nil
	case "users":
		u, err := config.ParseUsersConfig(data, true)
		if err != nil {
			return nil, err
		}
		for _, x := range u.Users {
			users = append(users, x.Username)
		}
		haveUsers = true
		sudo, _ = config.LoadSudoConfigStrict(filepath.Join(cfgPath, config.SudoFile))
	case "sudo":
		if sudo, err = config.ParseSudoConfig(data, true); err != nil {
			return nil, err
		}
		if c, _, err := config.LoadConfigDir(cfgPath, false); err == nil {
			for _, x := range c.Users {
				users = append(users, x.Username)
			}
			haveUsers = true
		}
	}
	if sudo != nil && haveUsers {
		known := map[string]bool{}
		for _, u := range users {
			known[u] = true
		}
		var orphans []string
		for u := range sudo.Users {
			if !known[u] {
				orphans = append(orphans, u)
			}
		}
		sort.Strings(orphans)
		for _, u := range orphans {
			warnings = append(warnings, fmt.Sprintf("mcp-sudo.yaml has grants for %q, which isn't an mcpd user", u))
		}
	}
	return warnings, nil
}
