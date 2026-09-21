package readfile

import (
	"encoding/json"
	"fmt"
	"os"
)

type ReadFileArgs struct {
	Path string `json:"path"`
}

func Read(argsJSON []byte) (string, error) {
	var args ReadFileArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	data, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	return string(data), nil
}
