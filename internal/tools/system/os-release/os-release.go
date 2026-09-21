package osrelease

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
)

type GetOSReleaseArgs struct {
	OutputFormat string `json:"output_format,omitempty"`
}

func OSRelease(argsJSON []byte) (string, error) {
	var args GetOSReleaseArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("failed to parse args: %v", err)
	}
	var out bytes.Buffer

	// Try reading /etc/os-release
	osRelease, err := os.ReadFile("/etc/os-release")
	if err == nil {
		out.WriteString("OS Release Info:\n")
		out.WriteString(string(osRelease))
		out.WriteString("\n")
	}

	// Use native syscall.Uname instead of exec.Command
	var uts syscall.Utsname
	err = syscall.Uname(&uts)
	var kernelInfo string
	if err == nil {
		sysname := charsToString(uts.Sysname[:])
		nodename := charsToString(uts.Nodename[:])
		release := charsToString(uts.Release[:])
		version := charsToString(uts.Version[:])
		machine := charsToString(uts.Machine[:])
		
		kernelInfo = fmt.Sprintf("%s %s %s %s %s", sysname, nodename, release, version, machine)
		out.WriteString("Kernel Info:\n")
		out.WriteString(kernelInfo)
		out.WriteString("\n")
	} else {
		out.WriteString(fmt.Sprintf("Failed to get uname sysinfo: %v\n", err))
	}

	if out.Len() == 0 {
		return "", fmt.Errorf("failed to retrieve any OS release information")
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		data := map[string]string{
			"os_release": string(osRelease),
			"kernel":     kernelInfo,
		}
		b, _ := json.Marshal(data)
		return string(b), nil
	}

	return out.String(), nil
}

func charsToString(ca []int8) string {
	s := make([]byte, len(ca))
	var lens int
	for ; lens < len(ca); lens++ {
		if ca[lens] == 0 {
			break
		}
		s[lens] = uint8(ca[lens])
	}
	return string(s[0:lens])
}
