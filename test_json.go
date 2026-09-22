package main
import (
	"encoding/json"
	"fmt"
)
type GetListOfFilesArgs struct {
	OutputFormat string `json:"output_format,omitempty"`
	Path       string `json:"path"`
}
func main() {
	j := []byte(`{"path": "/var/log", "output_format": "json"}`)
	var args GetListOfFilesArgs
	json.Unmarshal(j, &args)
	fmt.Printf("OutputFormat: %s\n", args.OutputFormat)
}
