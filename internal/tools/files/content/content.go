package content

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// ContentArgs defines the parameters for the files/content resource template.
type ContentArgs struct {
	Path string `json:"path"` // Path is the absolute path to the file to read. Required.
}

const maxReadBytes = 10240 // 10KB

func Content(args []byte) (string, error) {
	var parsedArgs ContentArgs
	if err := json.Unmarshal(args, &parsedArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if parsedArgs.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	file, err := os.Open(parsedArgs.Path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %v", parsedArgs.Path, err)
	}
	defer file.Close()

	// Detect if file is binary
	{
		sniffBuf := make([]byte, 512)
		n, err := file.Read(sniffBuf)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read file for type detection: %v", err)
		}
		if n > 0 {
			contentType := http.DetectContentType(sniffBuf[:n])
			if !strings.HasPrefix(contentType, "text/") && contentType != "application/json" {
				isBinary := false
				for _, b := range sniffBuf[:n] {
					if b == 0 {
						isBinary = true
						break
					}
				}
				if isBinary {
					return "", fmt.Errorf("cannot read binary file (detected type: %s)", contentType)
				}
			}
			// Reset file pointer
			if _, err := file.Seek(0, io.SeekStart); err != nil {
				return "", fmt.Errorf("failed to reset file pointer: %v", err)
			}
		}
	}

	buf := make([]byte, maxReadBytes)
	n, err := io.ReadFull(file, buf)

	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("failed to read file contents: %v", err)
	}

	contentStr := string(buf[:n])

	// If we read exactly maxReadBytes, there might be more data.
	if n == maxReadBytes {
		// Check if there is literally anything else in the file
		extraBuf := make([]byte, 1)
		_, extraErr := file.Read(extraBuf)
		if extraErr == nil {
			contentStr += "\n\n[WARNING: File truncated at 10KB context limit]"
		}
	}

	return contentStr, nil
}
