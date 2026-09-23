package file

import (
	"encoding/json"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/worker"
)

func Handle(uri string, sessionUser string, sudoConfig *config.SudoConfig) (string, string, error) {
	path := strings.TrimPrefix(uri, "file://")

	// The schema (internal/rpc/resources.go) advertises /stat, /content, and
	// /type suffixes for metadata/content/MIME-type reads respectively. A
	// bare path with none of these defaults to /content, matching the
	// template's original (suffix-less) behavior.
	tool := "files/content"
	if trimmed, ok := strings.CutSuffix(path, "/stat"); ok {
		tool, path = "files/stat", trimmed
	} else if trimmed, ok := strings.CutSuffix(path, "/type"); ok {
		tool, path = "files/filetype", trimmed
	} else if trimmed, ok := strings.CutSuffix(path, "/content"); ok {
		path = trimmed
	}

	// Determine if we need to run as root based on mcp-sudo.yaml paths list
	isPrivileged := sudoConfig.CanReadResourceAsRoot(sessionUser, "file://", path)

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"path": path,
	})

	content, readErr := worker.SpawnWorker(sessionUser, tool, argsJSON, isPrivileged, sudoConfig, 30)

	mimeType := "text/plain"
	switch {
	case tool == "files/stat":
		mimeType = "application/json"
	case strings.HasSuffix(path, ".json"):
		mimeType = "application/json"
	}

	return content, mimeType, readErr
}
