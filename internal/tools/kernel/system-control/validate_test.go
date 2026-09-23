package system_control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeProcSys builds a small /proc/sys-like tree and points procSys at it.
func fakeProcSys(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"net/ipv4/ip_forward":              "1\n",
		"net/ipv4/tcp_syncookies":          "1\n",
		"net/ipv4/conf/eth0.100/rp_filter": "2\n",
		"vm/swappiness":                    "60\n",
		"kernel/hostname":                  "vps\n",
		"kernel/multi":                     "a\nb\n",
	}
	for rel, v := range files {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(v), 0o644)
	}
	old := procSys
	procSys = root
	t.Cleanup(func() { procSys = old })
	return root
}

func call(t *testing.T, args map[string]interface{}) (string, error) {
	t.Helper()
	b, _ := json.Marshal(args)
	return SystemControl(b)
}

func TestReadDottedAndSlashKeys(t *testing.T) {
	fakeProcSys(t)
	for _, key := range []string{"net.ipv4.ip_forward", "net/ipv4/ip_forward"} {
		out, err := call(t, map[string]interface{}{"key": key})
		if err != nil || out != "net.ipv4.ip_forward = 1\n" {
			t.Errorf("read %q = %q, %v", key, out, err)
		}
	}
	// A component containing a dot is reachable via the slash form.
	out, err := call(t, map[string]interface{}{"key": "net/ipv4/conf/eth0.100/rp_filter"})
	if err != nil || !strings.Contains(out, "= 2") {
		t.Errorf("slash key with dotted component: %q, %v", out, err)
	}
}

func TestWriteThenReadBack(t *testing.T) {
	root := fakeProcSys(t)
	out, err := call(t, map[string]interface{}{"key": "vm.swappiness", "value": "10"})
	if err != nil || out != "vm.swappiness = 10\n" {
		t.Errorf("write = %q, %v", out, err)
	}
	// A real /proc/sys file replaces its value on write; this plain temp
	// file just gets its first bytes overwritten ("60\n" -> "10\n").
	if b, _ := os.ReadFile(filepath.Join(root, "vm/swappiness")); !strings.HasPrefix(string(b), "10") {
		t.Errorf("file holds %q", b)
	}
	// Writing a key that doesn't exist must fail, never create a file.
	if _, err := call(t, map[string]interface{}{"key": "vm.no_such_key", "value": "1"}); err == nil {
		t.Error("write to a missing key succeeded")
	}
	if _, err := os.Stat(filepath.Join(root, "vm/no_such_key")); err == nil {
		t.Error("write created a new file")
	}
}

func TestSubtreeAndReadAll(t *testing.T) {
	fakeProcSys(t)
	out, err := call(t, map[string]interface{}{"key": "net.ipv4"})
	if err != nil || !strings.Contains(out, "net.ipv4.ip_forward = 1") || !strings.Contains(out, "net.ipv4.tcp_syncookies = 1") {
		t.Errorf("subtree read: %q, %v", out, err)
	}
	out, err = call(t, map[string]interface{}{"read_all": true})
	if err != nil || !strings.Contains(out, "vm.swappiness = 60") || !strings.Contains(out, "kernel.multi = b") {
		t.Errorf("read_all: %q, %v", out, err)
	}
}

func TestKeysCannotLeaveProcSys(t *testing.T) {
	fakeProcSys(t)
	for _, key := range []string{"-p/etc/shadow", "../../etc/passwd", "net/../../../etc", "a b", "*", ""} {
		if key == "" {
			continue
		}
		if _, err := KeyPath(key); err == nil {
			t.Errorf("KeyPath accepted %q", key)
		}
	}
	if _, err := call(t, map[string]interface{}{"key": "kernel.hostname", "value": "x\ny"}); err == nil {
		t.Error("multi-line value accepted")
	}
}

func TestNormalizeKey(t *testing.T) {
	for in, want := range map[string]string{"net/ipv4/ip_forward": "net.ipv4.ip_forward", "vm.swappiness": "vm.swappiness", "/vm/x/": "vm.x"} {
		if got := NormalizeKey(in); got != want {
			t.Errorf("NormalizeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValueAcceptsNumbersAndBools(t *testing.T) {
	fakeProcSys(t)
	for raw, want := range map[string]string{`"10"`: "10", `10`: "10", `true`: "1", `false`: "0"} {
		args, err := ParseArgs([]byte(`{"key":"vm.swappiness","value":` + raw + `}`))
		if err != nil || string(args.Value) != want {
			t.Errorf("value %s parsed as %q, %v; want %q", raw, args.Value, err, want)
		}
	}
	if _, err := ParseArgs([]byte(`{"key":"vm.swappiness","value":{"x":1}}`)); err == nil {
		t.Error("object value accepted")
	}
	out, err := call(t, map[string]interface{}{"key": "vm.swappiness", "value": 7})
	if err != nil || !strings.HasPrefix(out, "vm.swappiness = 7") {
		t.Errorf("numeric write: %q, %v", out, err)
	}
}
