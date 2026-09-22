package locale

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Read returns the system's configured locale settings (LANG, LC_*), from
// /etc/default/locale (Debian/Ubuntu) or /etc/locale.conf (systemd/RHEL
// style), whichever exists first. Falls back to this process's own
// environment variables if neither file is present - a best-effort signal,
// not necessarily what an interactive login shell would see.
func Read() (string, string, error) {
	vars := make(map[string]string)
	source := ""

	for _, path := range []string{"/etc/default/locale", "/etc/locale.conf"} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		source = path
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			vars[k] = strings.Trim(v, `"'`)
		}
		f.Close()
		break
	}

	if len(vars) == 0 {
		for _, key := range []string{"LANG", "LC_ALL", "LC_CTYPE"} {
			if v := os.Getenv(key); v != "" {
				vars[key] = v
			}
		}
		source = "this process's environment (no /etc/default/locale or /etc/locale.conf found)"
	}

	if len(vars) == 0 {
		return "", "", fmt.Errorf("could not determine locale: no /etc/default/locale, /etc/locale.conf, or LANG/LC_* environment variables")
	}

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Source: %s\n", source))
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%s=%s\n", k, vars[k]))
	}
	return sb.String(), "text/plain", nil
}
