package top

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/procstat"
)

func TestTimePlus(t *testing.T) {
	for ticks, want := range map[uint64]string{0: "0:00.00", 512: "0:05.12", 6000: "1:00.00", 3723456: "620:34.56"} {
		if got := timePlus(ticks); got != want {
			t.Errorf("timePlus(%d) = %q, want %q", ticks, got, want)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	for secs, want := range map[int64]string{
		42 * 60:                  "up 42 min",
		3*3600 + 2*60:            "up  3:02",
		86400 + 3600:             "up 1 day,  1:00",
		5*86400 + 30*60:          "up 5 days, 30 min",
		839*86400 + 7*3600 + 120: "up 839 days,  7:02",
	} {
		if got := formatUptime(secs); got != want {
			t.Errorf("formatUptime(%d) = %q, want %q", secs, got, want)
		}
	}
}

func TestScaleKiB(t *testing.T) {
	cases := []struct {
		kib   uint64
		width int
		want  string
	}{
		{13440, 6, "13440"},
		{999999, 6, "999999"},
		{1048576, 6, "1.0g"}, // "1024.0m" would be 7 chars, so g
		{2621440, 7, "2621440"},
		{104857600, 7, "100.0g"},
	}
	for _, c := range cases {
		if got := scaleKiB(c.kib, c.width); got != c.want || len(got) > c.width {
			t.Errorf("scaleKiB(%d, %d) = %q, want %q", c.kib, c.width, got, c.want)
		}
	}
}

// fakeProc builds a minimal /proc with two processes and returns a func
// that bumps process 200's utime - simulating it burning CPU.
func fakeProc(t *testing.T) func(extraTicks int) {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("stat", "cpu  1000 0 500 8000 100 0 0 0 0 0\n")
	write("meminfo", "MemTotal: 2048000 kB\nMemFree: 512000 kB\nMemAvailable: 1500000 kB\nBuffers: 100000 kB\nCached: 800000 kB\nSReclaimable: 100000 kB\nSwapTotal: 1024000 kB\nSwapFree: 1024000 kB\n")
	write("loadavg", "0.17 0.24 0.52 1/123 4567\n")
	write("uptime", "10920.55 20000.00\n")
	stat := func(pid int, comm, state string, utime int) string {
		return fmt.Sprintf("%d (%s) %s 1 %d %d 0 -1 0 0 0 0 0 %d 10 0 0 20 0 1 0 100 0 0\n", pid, comm, state, pid, pid, utime)
	}
	write("1/stat", stat(1, "systemd", "S", 500))
	write("1/statm", "5634 3360 2336 0 0 0 0\n")
	write("1/status", "Name:\tsystemd\nUid:\t0\t0\t0\t0\n")
	write("1/cmdline", "/sbin/init\x00splash\x00")
	write("200/stat", stat(200, "burner", "R", 0))
	write("200/statm", "25000 12500 1000 0 0 0 0\n")
	write("200/status", "Name:\tburner\nUid:\t0\t0\t0\t0\n")
	old := procstat.Root
	procstat.Root = root
	t.Cleanup(func() { procstat.Root = old })
	return func(extra int) { write("200/stat", stat(200, "burner", "R", extra)) }
}

func TestTakeComputesCPUFromTwoSamples(t *testing.T) {
	burn := fakeProc(t)
	go func() {
		time.Sleep(50 * time.Millisecond)
		burn(20) // 20 ticks = 0.2s of CPU during a 0.2s interval = ~100%
	}()
	snap, err := Take(200 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	SortRows(snap.Processes, "cpu")
	if len(snap.Processes) != 2 || snap.Processes[0].PID != 200 {
		t.Fatalf("rows: %+v", snap.Processes)
	}
	if c := snap.Processes[0].CPUPercent; c < 80 || c > 105 {
		t.Errorf("burner %%CPU = %.1f, want ~100", c)
	}
	if c := snap.Processes[1].CPUPercent; c != 0 {
		t.Errorf("idle process %%CPU = %.1f, want 0", c)
	}
	s := snap.Summary
	if s.Tasks.Total != 2 || s.Tasks.Running != 1 || s.Tasks.Sleeping != 1 {
		t.Errorf("tasks %+v", s.Tasks)
	}
	if s.Uptime != "up  3:02" || s.LoadAverage[2] != 0.52 {
		t.Errorf("header %+v", s)
	}
	if s.MemMiB.Total != 2000 || s.MemMiB.BuffCache != 976.6 {
		t.Errorf("mem %+v", s.MemMiB)
	}
	if s.Mem.Total != 2000*1024*1024 || s.Swap.Total != s.kib.swapTotal*1024 {
		t.Errorf("mem bytes %+v, swap bytes %+v", s.Mem, s.Swap)
	}
	if r := snap.Processes[1]; r.ResBytes != r.ResKiB*1024 || r.VirtBytes != r.VirtKiB*1024 {
		t.Errorf("row bytes %+v", r)
	}
	// JSON carries bytes only - no KiB/MiB fields.
	j, _ := json.Marshal(snap)
	for _, key := range []string{`"mem_bytes"`, `"swap_bytes"`, `"res_bytes"`} {
		if !strings.Contains(string(j), key) {
			t.Errorf("JSON lacks %s: %s", key, j)
		}
	}
	for _, key := range []string{"_mib", "_kib"} {
		if strings.Contains(string(j), key) {
			t.Errorf("JSON still has a %s field: %s", key, j)
		}
	}
	// systemd: RES 3360 pages * pagesize / 2048000 KiB
	sysd := snap.Processes[1]
	if sysd.Command != "systemd" || sysd.Cmdline != "/sbin/init splash" || sysd.TimePlus != "0:05.10" || sysd.PR != "20" {
		t.Errorf("systemd row %+v", sysd)
	}
}

func TestFormatLayout(t *testing.T) {
	fakeProc(t)
	// human_readable: top's own layout.
	out, err := Top([]byte(`{"interval_ms": 50, "sort_by": "pid", "human_readable": true}`))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out, "\n")
	for i, prefix := range []string{"top - ", "Tasks:   2 total,", "%Cpu(s):", "MiB Mem :   2000.0 total,", "MiB Swap:   1000.0 total,", "", "    PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND"} {
		if !strings.HasPrefix(lines[i], prefix) {
			t.Errorf("line %d = %q, want prefix %q", i, lines[i], prefix)
		}
	}
	if !strings.Contains(lines[0], "load average: 0.17, 0.24, 0.52") {
		t.Errorf("header line: %q", lines[0])
	}
	if !strings.HasPrefix(lines[7], "      1 root") || !strings.HasSuffix(lines[7], "systemd") {
		t.Errorf("first row: %q", lines[7])
	}

	js, _ := Top([]byte(`{"interval_ms": 50, "output_format": "json", "limit": 1}`))
	var snap Snapshot
	if err := json.Unmarshal([]byte(js), &snap); err != nil || len(snap.Processes) != 1 {
		t.Errorf("json: %v %s", err, js)
	}
	if _, err := Top([]byte(`{"sort_by": "bogus"}`)); err == nil {
		t.Error("bad sort_by accepted")
	}
}

func TestTableAndWideFormats(t *testing.T) {
	fakeProc(t)
	table, err := Top([]byte(`{"interval_ms": 50, "sort_by": "pid", "output_format": "table"}`))
	if err != nil || !strings.HasPrefix(table, "top - ") || !strings.Contains(table, " systemd\n") {
		t.Errorf("table should be top's text layout: %v\n%s", err, table)
	}
	wide, err := Top([]byte(`{"interval_ms": 50, "sort_by": "pid", "output_format": "wide"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(wide, "   PPID  THR COMMAND") {
		t.Errorf("wide header missing PPID/THR:\n%s", wide)
	}
	if !strings.Contains(wide, "/sbin/init splash") || !strings.Contains(wide, "[burner]") {
		t.Errorf("wide should show cmdline, [comm] for no cmdline:\n%s", wide)
	}
}

func TestWideAlignsLongUsernames(t *testing.T) {
	snap := Snapshot{Processes: []Row{
		{PID: 1, User: "root", PR: "20", Command: "a", Cmdline: "/a"},
		{PID: 2, User: "privileged", PR: "20", Command: "b", Cmdline: "/b"},
	}}
	lines := strings.Split(FormatWide(snap, true), "\n")
	head, r1, r2 := lines[6], lines[7], lines[8]
	// The PR value must sit under the PR header in every row, whatever
	// the length of the user name before it.
	want := strings.Index(head, " PR ")
	if strings.Index(r1, " 20 ") != want || strings.Index(r2, " 20 ") != want {
		t.Errorf("misaligned:\n%s\n%s\n%s", head, r1, r2)
	}
	if !strings.Contains(r2, "privileged") {
		t.Errorf("wide must not truncate user names: %q", r2)
	}
}

func TestFormatBytes(t *testing.T) {
	fakeProc(t)
	out, err := Top([]byte(`{"interval_ms": 50, "sort_by": "pid"}`))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out, "\n")
	// 2000 MiB exactly, from the fake meminfo's kB value - not a rounded MiB.
	if !strings.HasPrefix(lines[3], "B Mem : 2097152000 total,") || !strings.HasPrefix(lines[4], "B Swap: 1048576000 total,") {
		t.Errorf("header in bytes:\n%s\n%s", lines[3], lines[4])
	}
	if !strings.Contains(lines[6], "VIRT") || strings.Contains(out, "MiB") {
		t.Errorf("unexpected byte layout:\n%s", out)
	}
}
