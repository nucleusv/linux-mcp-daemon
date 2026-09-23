package packages

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDpkgAndFilter(t *testing.T) {
	status := filepath.Join(t.TempDir(), "status")
	os.WriteFile(status, []byte(`Package: openssh-server
Status: install ok installed
Architecture: amd64
Version: 1:9.6p1-3ubuntu13.19

Package: openssl
Status: install ok installed
Architecture: amd64
Version: 3.0.13-0ubuntu3.15

Package: removed-one
Status: deinstall ok config-files
Architecture: all
Version: 1.0
`), 0o644)
	pkgs, err := parseDpkgStatus(status)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range pkgs {
		names = append(names, p.Name)
	}
	if strings.Join(names, ",") != "openssh-server,openssl" {
		t.Errorf("parsed %v (config-files-only packages must be excluded)", names)
	}
}

func TestNamePatternValidated(t *testing.T) {
	b, _ := json.Marshal(map[string]string{"name": "[unclosed"})
	if _, err := List(b); err == nil || !strings.Contains(err.Error(), "invalid name pattern") {
		// On a host without dpkg/apk List fails earlier; only check when it got that far.
		if err != nil && strings.Contains(err.Error(), "package") && !strings.Contains(err.Error(), "pattern") {
			t.Skip("no supported package manager on this machine:", err)
		}
		t.Errorf("bad glob accepted: %v", err)
	}
}
