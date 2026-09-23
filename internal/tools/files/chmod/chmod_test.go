package chmod

import "testing"

func TestParseModeUnits(t *testing.T) {
	cases := []struct {
		spec string
		old  uint32
		dir  bool
		want uint32
	}{
		{"644", 0o755, false, 0o644},
		{"0755", 0o600, false, 0o755},
		{"4755", 0o755, false, 0o4755},
		{"u+x", 0o644, false, 0o744},
		{"go-w", 0o666, false, 0o644},
		{"a=r", 0o755, false, 0o444},
		{"u+x,g-r", 0o644, false, 0o704},
		{"+X", 0o644, false, 0o644}, // X: file with no x bit stays non-executable
		{"+X", 0o644, true, 0o755},  // ... but a directory gets x
		{"+X", 0o744, false, 0o755}, // ... and so does an already-executable file
		{"u+s", 0o755, false, 0o4755},
		{"+t", 0o777, true, 0o1777},
		{"u=rx", 0o4755, false, 0o555},  // = clears setuid on a file
		{"u=rwx", 0o2775, true, 0o2775}, // ... but a dir keeps setgid
		{"755", 0o2775, true, 0o2755},   // numeric keeps a dir's setgid (GNU)
		{"00755", 0o2775, true, 0o755},  // ... unless given as five digits
		{"755", 0o4755, false, 0o755},   // files: numeric is exact
	}
	for _, c := range cases {
		f, err := ParseMode(c.spec)
		if err != nil {
			t.Errorf("ParseMode(%q): %v", c.spec, err)
			continue
		}
		if got := f(c.old, c.dir); got != c.want {
			t.Errorf("ParseMode(%q)(%04o, dir=%v) = %04o, want %04o", c.spec, c.old, c.dir, got, c.want)
		}
	}
	for _, bad := range []string{"", "9", "77777", "007777", "u", "u+q", "z+x", "u+x;rm"} {
		if _, err := ParseMode(bad); err == nil {
			t.Errorf("ParseMode(%q) accepted", bad)
		}
	}
}
