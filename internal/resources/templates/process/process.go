package process

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/worker"
)

func Handle(uri string, sessionUser string, sudoConfig *config.SudoConfig) (string, string, error) {
	rawPath := strings.TrimPrefix(uri, "process://")
	parts := strings.SplitN(rawPath, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid process URI format, expected process://{pid}/{target}")
	}
	
	pidStr, target := parts[0], parts[1]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid PID: %v", err)
	}

	isPrivileged := sudoConfig.CanReadResourceAsRoot(sessionUser, "process://", pidStr)
	
	argsJSON, _ := json.Marshal(map[string]interface{}{
		"pid":    pid,
		"target": target,
	})
	
	content, readErr := worker.SpawnWorker(sessionUser, "processes/read", argsJSON, isPrivileged, sudoConfig, 30)
	
	return content, "text/plain", readErr
}
