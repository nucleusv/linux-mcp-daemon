package service

import (
	"encoding/json"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/worker"
)

func Handle(uri string, sessionUser string, sudoConfig *config.SudoConfig) (string, string, error) {
	serviceName := strings.TrimPrefix(uri, "service://")
	serviceName = strings.TrimSuffix(serviceName, "/status")

	isPrivileged := sudoConfig.CanReadResourceAsRoot(sessionUser, "service://", serviceName)
	
	argsJSON, _ := json.Marshal(map[string]interface{}{
		"service": serviceName,
	})
	
	content, readErr := worker.SpawnWorker(sessionUser, "services/status", argsJSON, isPrivileged, sudoConfig, 30)
	
	return content, "application/json", readErr
}
