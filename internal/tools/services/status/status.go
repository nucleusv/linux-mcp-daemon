package status

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
)

// ServiceStatusArgs defines the parameters for the service://{name}/status resource.
type ServiceStatusArgs struct {
	Service string `json:"service"` // Service name. Required.
}

type ServiceStatus struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	LoadState    string `json:"load_state"`
	ActiveState  string `json:"active_state"`
	SubState     string `json:"sub_state"`
	FragmentPath string `json:"fragment_path"`
}

func Status(argsJSON []byte) (string, error) {
	var args ServiceStatusArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.Service == "" {
		return "", fmt.Errorf("service argument is required")
	}
	if !strings.HasSuffix(args.Service, ".service") {
		args.Service += ".service"
	}

	conn, err := dbus.NewSystemdConnectionContext(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to connect to systemd dbus: %v", err)
	}
	defer conn.Close()

	units, err := conn.ListUnitsByNamesContext(context.Background(), []string{args.Service})
	if err != nil {
		return "", fmt.Errorf("failed to list units: %v", err)
	}

	if len(units) == 0 {
		return "", fmt.Errorf("service not found: %s", args.Service)
	}

	unit := units[0]

	// Get additional properties
	props, err := conn.GetAllPropertiesContext(context.Background(), args.Service)
	if err != nil {
		return "", fmt.Errorf("failed to get properties: %v", err)
	}

	desc, _ := props["Description"].(string)
	fragmentPath, _ := props["FragmentPath"].(string)

	resp := ServiceStatus{
		Name:         unit.Name,
		Description:  desc,
		LoadState:    unit.LoadState,
		ActiveState:  unit.ActiveState,
		SubState:     unit.SubState,
		FragmentPath: fragmentPath,
	}

	out, _ := json.MarshalIndent(resp, "", "  ")
	return string(out), nil
}
