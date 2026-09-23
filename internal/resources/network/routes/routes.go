package routes

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"
	"net"
	"os"
	"strconv"
	"strings"
)

// Route is one IPv4 route from /proc/net/route, decoded - the kernel file
// stores addresses as little-endian hex ("01D4A7DE" is 222.167.212.1) and
// flags as a hex bitmask, neither of which is useful to read directly.
type Route struct {
	Destination string   `json:"destination"` // CIDR, e.g. "0.0.0.0/0" for the default route
	Gateway     string   `json:"gateway"`     // "" when directly connected
	Iface       string   `json:"iface"`
	Metric      int      `json:"metric"`
	Flags       []string `json:"flags"` // decoded, e.g. ["up", "gateway"]
	MTU         int      `json:"mtu"`
	Window      int      `json:"window"`
	IRTT        int      `json:"irtt"`
	Default     bool     `json:"default"`
}

// RTF_* flag bits from linux/route.h.
var routeFlags = []struct {
	bit  uint64
	name string
}{
	{0x0001, "up"},
	{0x0002, "gateway"},
	{0x0004, "host"},
	{0x0008, "reinstate"},
	{0x0010, "dynamic"},
	{0x0020, "modified"},
	{0x0200, "reject"},
}

func ReadRoutes(args []byte) (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/net/route: %v", err)
	}
	defer file.Close()

	routes := []Route{}
	scanner := bufio.NewScanner(file)
	scanner.Scan() // header line
	for scanner.Scan() {
		if r, ok := parseRoute(scanner.Text()); ok {
			routes = append(routes, r)
		}
	}

	b, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal routes: %v", err)
	}
	return string(b), nil
}

// parseRoute decodes one /proc/net/route line:
// Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT
func parseRoute(line string) (Route, bool) {
	fields := strings.Fields(line)
	if len(fields) < 11 {
		return Route{}, false
	}
	dest, ok1 := hexIPv4(fields[1])
	gw, ok2 := hexIPv4(fields[2])
	mask, ok3 := hexIPv4(fields[7])
	flags, err := strconv.ParseUint(fields[3], 16, 32)
	if !ok1 || !ok2 || !ok3 || err != nil {
		return Route{}, false
	}

	r := Route{
		Destination: fmt.Sprintf("%s/%d", dest, bits.OnesCount32(binary.BigEndian.Uint32(mask))),
		Iface:       fields[0],
		Metric:      atoi(fields[6]),
		MTU:         atoi(fields[8]),
		Window:      atoi(fields[9]),
		IRTT:        atoi(fields[10]),
		Flags:       []string{},
	}
	if !gw.Equal(net.IPv4zero) {
		r.Gateway = gw.String()
	}
	r.Default = r.Destination == "0.0.0.0/0"
	for _, f := range routeFlags {
		if flags&f.bit != 0 {
			r.Flags = append(r.Flags, f.name)
		}
	}
	return r, true
}

// hexIPv4 decodes /proc/net/route's little-endian hex address format.
func hexIPv4(s string) (net.IP, bool) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 4 {
		return nil, false
	}
	return net.IPv4(b[3], b[2], b[1], b[0]).To4(), true
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
