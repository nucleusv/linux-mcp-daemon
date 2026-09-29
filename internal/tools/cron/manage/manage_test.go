package manage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeCrontab installs a shell script standing in for `crontab`: it keeps the
// "crontab" in a file, prints it for -l ("no crontab for x" and exit 1 when
// there is none), replaces it for -, and removes it for -r.
func fakeCrontab(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	store := filepath.Join(dir, "store")
	script := "#!/bin/sh\nS=" + store + "\ncase \"$1\" in\n" +
		"-l) [ -f $S ] && cat $S || { echo 'no crontab for tester' >&2; exit 1; } ;;\n" +
		"-r) [ -f $S ] && rm $S || { echo 'no crontab for tester' >&2; exit 1; } ;;\n" +
		"-) cat > $S.new; if grep -q BAD $S.new; then echo '\"-\":1: bad minute' >&2; rm $S.new; exit 1; fi; mv $S.new $S ;;\nesac\n"
	bin := filepath.Join(dir, "crontab")
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	crontabBin = bin
	t.Cleanup(func() { crontabBin = "" })
	return store
}

func call(t *testing.T, a Args) (string, error) {
	t.Helper()
	b, _ := json.Marshal(a)
	return Manage(b)
}

func str(s string) *string { return &s }

func TestReadWriteRoundTripSelf(t *testing.T) {
	fakeCrontab(t)
	// no crontab yet: an empty read, not an error
	if out, err := call(t, Args{Mode: "self", User: "alice"}); err != nil || out != "" {
		t.Fatalf("read of a missing crontab: %q, %v", out, err)
	}
	text := "MAILTO=ops\n*/15 * * * * /home/alice/bin/sync.sh\n"
	out, err := call(t, Args{Mode: "self", User: "alice", Content: str(text)})
	if err != nil || !strings.Contains(out, "2 lines") {
		t.Fatalf("write: %q, %v", out, err)
	}
	if got, err := call(t, Args{Mode: "self", User: "alice"}); err != nil || got != text {
		t.Fatalf("round trip: %q, %v", got, err)
	}
	// a missing trailing newline is added
	if _, err := call(t, Args{Mode: "self", User: "alice", Content: str("0 3 * * * x")}); err != nil {
		t.Fatal(err)
	}
	if got, _ := call(t, Args{Mode: "self", User: "alice"}); got != "0 3 * * * x\n" {
		t.Errorf("newline not added: %q", got)
	}
	// json read carries the hash and counts
	js, _ := call(t, Args{Mode: "self", User: "alice", OutputFormat: "json"})
	var r crontabResult
	if err := json.Unmarshal([]byte(js), &r); err != nil || !r.Exists || r.Lines != 1 || r.Jobs != 1 || r.SHA256 != sum("0 3 * * * x\n") {
		t.Errorf("json read: %+v, %v", r, err)
	}
}

func TestInvalidFileKeepsTheOldOne(t *testing.T) {
	fakeCrontab(t)
	call(t, Args{Mode: "self", User: "alice", Content: str("0 3 * * * ok\n")})
	if _, err := call(t, Args{Mode: "self", User: "alice", Content: str("BAD line\n")}); err == nil || !strings.Contains(err.Error(), "old crontab is unchanged") {
		t.Errorf("want a rejection naming the unchanged crontab, got %v", err)
	}
	if got, _ := call(t, Args{Mode: "self", User: "alice"}); got != "0 3 * * * ok\n" {
		t.Errorf("the old crontab must remain, got %q", got)
	}
}

func TestIfMatchAndClear(t *testing.T) {
	fakeCrontab(t)
	call(t, Args{Mode: "self", User: "alice", Content: str("0 3 * * * a\n")})
	if _, err := call(t, Args{Mode: "self", User: "alice", Content: str("0 4 * * * b\n"), IfMatch: "deadbeef"}); err == nil || !strings.Contains(err.Error(), "changed since you read it") {
		t.Errorf("if_match mismatch must refuse, got %v", err)
	}
	ok := sum("0 3 * * * a\n")
	if _, err := call(t, Args{Mode: "self", User: "alice", Content: str("0 4 * * * b\n"), IfMatch: ok}); err != nil {
		t.Errorf("matching if_match must write: %v", err)
	}
	// an explicit empty content clears the crontab; clearing an absent one is a no-op
	if _, err := call(t, Args{Mode: "self", User: "alice", Content: str("")}); err != nil {
		t.Fatal(err)
	}
	if got, _ := call(t, Args{Mode: "self", User: "alice"}); got != "" {
		t.Errorf("not cleared: %q", got)
	}
	if _, err := call(t, Args{Mode: "self", User: "alice", Content: str("")}); err != nil {
		t.Errorf("clearing an absent crontab must not fail: %v", err)
	}
}

func TestLimitsAndModes(t *testing.T) {
	fakeCrontab(t)
	if _, err := call(t, Args{Mode: "self", Content: str(strings.Repeat("x", MaxContent+1))}); err == nil {
		t.Error("oversized content must be refused")
	}
	if _, err := call(t, Args{Mode: "self", Content: str("a\x00b")}); err == nil {
		t.Error("a NUL byte must be refused")
	}
	if _, err := call(t, Args{}); err == nil {
		t.Error("a call the daemon did not prepare (no mode) must be refused")
	}
	if _, err := call(t, Args{Mode: "other", TargetUser: "-l"}); err == nil {
		t.Error("an option-like account name must be refused")
	}
}

func TestListFiltersByViewAndSkipsNonFiles(t *testing.T) {
	dir := t.TempDir()
	spoolDirs = []string{dir}
	t.Cleanup(func() { spoolDirs = []string{"/var/spool/cron/crontabs", "/var/spool/cron"} })
	os.WriteFile(filepath.Join(dir, "alice"), []byte("# c\n0 3 * * * x\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "bob"), []byte("0 1 * * * y\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "carol"), []byte("0 2 * * * z\n"), 0o600)
	os.Symlink("/etc/passwd", filepath.Join(dir, "dave")) // a symlink is never followed
	out, err := call(t, Args{Mode: "list", ViewUsers: []string{"alice", "carol", "dave", "nobody"}, OutputFormat: "json"})
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		User  string `json:"user"`
		Lines int    `json:"lines"`
	}
	json.Unmarshal([]byte(out), &rows)
	if len(rows) != 2 || rows[0].User != "alice" || rows[0].Lines != 2 || rows[1].User != "carol" {
		t.Errorf("want alice and carol only (bob not viewable, dave a symlink): %+v", rows)
	}
}

func TestCounts(t *testing.T) {
	s := "# comment\nMAILTO=ops\n\n*/5 * * * * a\n@daily b\n"
	if countLines(s) != 5 || countJobs(s) != 2 {
		t.Errorf("lines %d jobs %d", countLines(s), countJobs(s))
	}
}
