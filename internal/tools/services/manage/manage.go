package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
)

// ManageServiceArgs defines the parameters for the services/manage tool.
type ManageServiceArgs struct {
	Service string `json:"service"` // Service name (e.g., "kubelet.service"). Required.
	Action  string `json:"action"`  // Action to perform: start, stop, restart, reload, enable, disable. Required.
}

func Manage(argsJSON []byte) (string, error) {
	var args ManageServiceArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.Service == "" {
		return "", fmt.Errorf("service argument is required")
	}
	if !strings.HasSuffix(args.Service, ".service") {
		args.Service += ".service"
	}

	if args.Action == "" {
		return "", fmt.Errorf("action argument is required")
	}

	conn, err := dbus.NewSystemdConnectionContext(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to connect to systemd dbus: %v", err)
	}
	defer conn.Close()

	ch := make(chan string)
	var jobID int
	var startErr error

	switch args.Action {
	case "start":
		jobID, startErr = conn.StartUnitContext(context.Background(), args.Service, "replace", ch)
	case "stop":
		jobID, startErr = conn.StopUnitContext(context.Background(), args.Service, "replace", ch)
	case "restart":
		jobID, startErr = conn.RestartUnitContext(context.Background(), args.Service, "replace", ch)
	case "reload":
		jobID, startErr = conn.ReloadUnitContext(context.Background(), args.Service, "replace", ch)
	case "enable":
		install, changes, err := conn.EnableUnitFilesContext(context.Background(), []string{args.Service}, false, true)
		if err != nil {
			return "", fmt.Errorf("failed to enable unit: %v", err)
		}
		return fmt.Sprintf("Enabled %s (install: %v, changes: %d)", args.Service, install, len(changes)), nil
	case "disable":
		changes, err := conn.DisableUnitFilesContext(context.Background(), []string{args.Service}, false)
		if err != nil {
			return "", fmt.Errorf("failed to disable unit: %v", err)
		}
		return fmt.Sprintf("Disabled %s (changes: %d)", args.Service, len(changes)), nil
	default:
		return "", fmt.Errorf("unsupported action: %s", args.Action)
	}

	if startErr != nil {
		return "", fmt.Errorf("failed to initiate %s job: %v", args.Action, startErr)
	}

	// Wait for job completion
	status := <-ch
	return fmt.Sprintf("Job %d completed with status: %s", jobID, status), nil
}
