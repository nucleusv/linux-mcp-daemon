package stat

import (
	"encoding/json"
	"fmt"
	"os"
)

type StatArgs struct {
	Path string `json:"path"`
}

type FileStat struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"modified_time"`
	IsDir   bool   `json:"is_dir"`
}

func Stat(args []byte) (string, error) {
	var parsedArgs StatArgs
	if err := json.Unmarshal(args, &parsedArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if parsedArgs.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	info, err := os.Stat(parsedArgs.Path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file %s: %v", parsedArgs.Path, err)
	}

	statObj := FileStat{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    info.Mode().String(),
		ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		IsDir:   info.IsDir(),
	}

	b, err := json.MarshalIndent(statObj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal stat: %v", err)
	}

	return string(b), nil
}
