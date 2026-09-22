package file

import (
	"encoding/json"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/worker"
)

func Handle(uri string, sessionUser string, sudoConfig *config.SudoConfig) (string, string, error) {
	path := strings.TrimPrefix(uri, "file://")

	// Determine if we need to run as root based on mcp-sudo.yaml paths list
	isPrivileged := sudoConfig.CanReadResourceAsRoot(sessionUser, "file://", path)

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"path": path,
	})

	content, readErr := worker.SpawnWorker(sessionUser, "files/content", argsJSON, isPrivileged, sudoConfig, 30)
	
	mimeType := "text/plain"
	if strings.HasSuffix(path, ".json") {
		mimeType = "application/json"
	}
	
	return content, mimeType, readErr
}
