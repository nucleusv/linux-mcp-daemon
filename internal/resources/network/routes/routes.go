package routes

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Route struct {
	Iface       string `json:"iface"`
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Flags       string `json:"flags"`
	RefCnt      string `json:"ref_cnt"`
	Use         string `json:"use"`
	Metric      string `json:"metric"`
	Mask        string `json:"mask"`
	MTU         string `json:"mtu"`
	Window      string `json:"window"`
	IRTT        string `json:"irtt"`
}

func ReadRoutes(args []byte) (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/net/route: %v", err)
	}
	defer file.Close()

	var routes []Route
	scanner := bufio.NewScanner(file)
	
	// Skip header
	if scanner.Scan() {
		// Header line
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 11 {
			routes = append(routes, Route{
				Iface:       fields[0],
				Destination: fields[1],
				Gateway:     fields[2],
				Flags:       fields[3],
				RefCnt:      fields[4],
				Use:         fields[5],
				Metric:      fields[6],
				Mask:        fields[7],
				MTU:         fields[8],
				Window:      fields[9],
				IRTT:        fields[10],
			})
		}
	}

	if len(routes) == 0 {
		return "[]", nil
	}

	b, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal routes: %v", err)
	}
	return string(b), nil
}
