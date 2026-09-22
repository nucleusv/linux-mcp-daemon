package disks

import (
	"encoding/json"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/worker"
)

func Handle(uri string, sessionUser string, sudoConfig *config.SudoConfig) (string, string, error) {
	devName := strings.TrimSuffix(strings.TrimPrefix(uri, "disks://"), "/stats")
	
	// Right now we don't strictly require root for /proc/diskstats but we can pass it if we ever change backend
	argsJSON, _ := json.Marshal(map[string]interface{}{
		"device":        devName,
		"output_format": "json",
	})
	
	content, readErr := worker.SpawnWorker(sessionUser, "disks/performance", argsJSON, false, sudoConfig, 30)
	
	return content, "application/json", readErr
}
