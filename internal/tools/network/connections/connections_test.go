package connections

import (
	"encoding/json"
	"net"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/kernel"
)

func TestStateSet(t *testing.T) {
	cases := map[string][]string{
		"LISTEN":      {"LISTEN", "UNCONN"},
		"listening":   {"LISTEN", "UNCONN"},
		"ESTABLISHED": {"ESTAB"},
		"TIME_WAIT":   {"TIME-WAIT"},
		"close_wait":  {"CLOSE-WAIT"},
		"SYN_RECV":    {"SYN-RECV"},
		" Listen ":    {"LISTEN", "UNCONN"},
	}
	for in, want := range cases {
		got, err := stateSet(in)
		if err != nil || len(got) != len(want) {
			t.Errorf("stateSet(%q) = %v, %v; want %v", in, got, err, want)
			continue
		}
		for _, s := range want {
			if !got[s] {
				t.Errorf("stateSet(%q) lacks %s", in, s)
			}
		}
	}
	if set, err := stateSet("all"); err != nil || set != nil {
		t.Errorf("all: %v, %v", set, err)
	}
	if _, err := stateSet("bogus; rm -rf /"); err == nil {
		t.Error("stateSet accepted an unknown state")
	}
}

// A listener opened by the test itself must show up with its owner, and
// the port filter must find it.
func TestConnectionsFindsOwnListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	args, _ := json.Marshal(GetConnectionsArgs{State: "LISTEN", Port: port, OutputFormat: "json"})
	out, err := Connections(args)
	if err != nil {
		t.Skip(err) // no /proc/net (not Linux)
	}
	var socks []kernel.Socket
	if err := json.Unmarshal([]byte(out), &socks); err != nil {
		t.Fatal(err)
	}
	if len(socks) != 1 || socks[0].Netid != "tcp" || socks[0].State != "LISTEN" || socks[0].LocalAddr != "127.0.0.1" || socks[0].LocalPort != port {
		t.Fatalf("got %+v", socks)
	}
	if len(socks[0].Processes) == 0 {
		t.Error("the test's own listener has no owning process")
	}

	text, _ := Connections([]byte(`{"state": "listening"}`))
	if !strings.HasPrefix(text, "Netid") || !strings.Contains(text, "127.0.0.1:"+itoa(port)) {
		t.Errorf("text output lacks the listener:\n%s", text)
	}
}

// TestMatchesSS compares the set of sockets with `ss -tuan`, where ss is
// installed: the same (netid, state, local, peer) tuples.
func TestMatchesSS(t *testing.T) {
	if _, err := exec.LookPath("ss"); err != nil {
		t.Skip("ss not installed")
	}
	ln, err := net.Listen("tcp", "[::]:0")
	if err == nil {
		defer ln.Close()
	}
	u, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err == nil {
		defer u.Close()
	}

	out, err := exec.Command("ss", "-tuanH").Output()
	if err != nil {
		t.Skip(err)
	}
	var want []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(line)
		if len(f) >= 6 {
			// ss prints a dual-stack (not IPV6_V6ONLY) wildcard socket as
			// "*:port"; /proc/net doesn't carry that flag, so we print the
			// socket's actual address, "[::]:port".
			// It also appends "%iface" for sockets bound to a device
			// (SO_BINDTODEVICE), which only netlink reports.
			for i := 4; i <= 5; i++ {
				if strings.HasPrefix(f[i], "*:") {
					f[i] = "[::]" + f[i][1:]
				}
				if pct := strings.IndexByte(f[i], '%'); pct >= 0 {
					if colon := strings.LastIndexByte(f[i], ':'); colon > pct {
						f[i] = f[i][:pct] + f[i][colon:]
					}
				}
			}
			want = append(want, strings.Join([]string{f[0], f[1], f[4], f[5]}, " "))
		}
	}
	socks, err := kernel.ReadSockets("/proc")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range socks {
		got = append(got, strings.Join([]string{s.Netid, s.State, s.Endpoint(true), s.Endpoint(false)}, " "))
	}
	sort.Strings(want)
	sort.Strings(got)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("sockets differ from ss -tuan\nours:\n%s\nss:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	t.Logf("compared %d sockets", len(want))
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }
