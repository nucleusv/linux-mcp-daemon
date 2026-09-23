package ping

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestPingNetworkPolicy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port

	run := func(policy interface{}) PingResponse {
		args := map[string]interface{}{"host": "127.0.0.1", "port": port}
		if policy != nil {
			args["_network_policy"] = policy
		}
		b, _ := json.Marshal(args)
		out, err := Ping(b)
		if err != nil {
			t.Fatal(err)
		}
		var r PingResponse
		json.Unmarshal([]byte(out), &r)
		return r
	}

	if r := run(nil); !r.Success {
		t.Errorf("unrestricted ping to 127.0.0.1:%d failed: %s", port, r.Error)
	}
	if r := run(map[string]interface{}{"deny_private": true}); r.Success || !strings.Contains(r.Error, "deny_private") {
		t.Errorf("deny_private ping not blocked: %+v", r)
	}
	if r := run(map[string]interface{}{"deny_private": true, "allow": []string{"127.0.0.1"}}); !r.Success {
		t.Errorf("allowlisted ping blocked: %s (port %s)", r.Error, strconv.Itoa(port))
	}
}
