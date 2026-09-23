// Package version holds the build's version, set at link time:
//
//	go build -ldflags "-X github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/version.Version=0.1.0"
//
// Release builds (GoReleaser, the release Docker image) set all three;
// local builds report "dev".
package version

import "fmt"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// String is the one-line form printed by `mcpd --version` and
// `linuxctl --version`.
func String(program string) string {
	return fmt.Sprintf("%s %s (commit %s, built %s)", program, Version, Commit, Date)
}
