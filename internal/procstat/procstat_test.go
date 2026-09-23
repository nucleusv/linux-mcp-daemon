package procstat

import "testing"

func TestParseStatCommWithSpacesAndParens(t *testing.T) {
	// comm may contain spaces and ")" - fields start after the LAST ")".
	line := []byte("1234 (tmux: server) x) S 1 1234 1234 0 -1 4194560 100 0 0 0 250 50 0 0 20 0 3 0 12345 10000 500 18446744073709551615")
	p, err := parseStat(line)
	if err != nil {
		t.Fatal(err)
	}
	if p.Comm != "tmux: server) x" || p.State != 'S' || p.PPID != 1 || p.UTime != 250 || p.STime != 50 ||
		p.Priority != 20 || p.Nice != 0 || p.Threads != 3 {
		t.Errorf("parsed %+v", p)
	}
	if _, err := parseStat([]byte("garbage")); err == nil {
		t.Error("parsed garbage")
	}
}

func TestMemoryDerived(t *testing.T) {
	m := Memory{Total: 1000, Free: 100, Buffers: 50, Cached: 300, SReclaimable: 50}
	if m.BuffCache() != 400 || m.Used() != 500 {
		t.Errorf("buff/cache %d used %d", m.BuffCache(), m.Used())
	}
}
