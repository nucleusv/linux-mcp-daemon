package filetype

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// FileTypeArgs defines the parameters for the file://{path}/type resource.
type FileTypeArgs struct {
	Path string `json:"path"` // Path is the absolute path to the file. Required.
}

func Type(argsJSON []byte) (string, error) {
	// Actually, resources typically get invoked with a path directly from rpc.go 
	// but we'll accept the JSON for consistency with tools.
	// Wait, resources don't use argsJSON in the same way, they are passed as raw strings or JSON.
	// We'll just assume standard JSON mapping like `stat.go` and `content.go` use.
	var args FileTypeArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	// We use the 'file' command to determine the type
	cmd := exec.Command("file", "-b", "--mime-type", args.Path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("file command error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
