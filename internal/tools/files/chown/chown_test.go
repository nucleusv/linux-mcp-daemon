package chown

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOwner(t *testing.T) {
	d := t.TempDir()
	passwd, group := filepath.Join(d, "passwd"), filepath.Join(d, "group")
	os.WriteFile(passwd, []byte("root:x:0:0::/root:/bin/bash\nalice:x:1000:1000::/home/alice:/bin/bash\n"), 0o644)
	os.WriteFile(group, []byte("root:x:0:\nalice:x:1000:\nadm:x:4:alice\n"), 0o644)
	cases := []struct {
		spec     string
		uid, gid int
	}{
		{"alice", 1000, -1},
		{"alice:adm", 1000, 4},
		{":adm", -1, 4},
		{"alice:", 1000, 1000}, // login group
		{"1234:5678", 1234, 5678},
		{"root:root", 0, 0},
	}
	for _, c := range cases {
		uid, gid, err := ParseOwner(c.spec, passwd, group)
		if err != nil || uid != c.uid || gid != c.gid {
			t.Errorf("ParseOwner(%q) = %d, %d, %v; want %d, %d", c.spec, uid, gid, err, c.uid, c.gid)
		}
	}
	for _, bad := range []string{"", ":", "nobody-here", "alice:nogroup"} {
		if _, _, err := ParseOwner(bad, passwd, group); err == nil {
			t.Errorf("ParseOwner(%q) accepted", bad)
		}
	}
}
