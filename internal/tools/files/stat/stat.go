package stat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nucleusv/linux-mcp-daemon/internal/fsafe"
)

// StatArgs defines the parameters for the files/stat resource template.
type StatArgs struct {
	Path string `json:"path"` // Path is the absolute path to the file or directory to stat. Required.
	// NoFollow is set by the daemon (never the caller) for a root read
	// under a narrower-than-"/" grant: no symlink is followed in any
	// component of Path.
	NoFollow bool `json:"_no_follow,omitempty"`
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

	var info os.FileInfo
	var err error
	if parsedArgs.NoFollow {
		var n *fsafe.Node
		if n, err = fsafe.Open(parsedArgs.Path); err == nil {
			info, err = os.Stat(n.ProcPath()) // this exact, symlink-free inode
			n.Close()
		}
	} else {
		info, err = os.Stat(parsedArgs.Path)
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat file %s: %v", parsedArgs.Path, err)
	}

	statObj := FileStat{
		Name:    filepath.Base(parsedArgs.Path),
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
