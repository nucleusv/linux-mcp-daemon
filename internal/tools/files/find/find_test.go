package findfile

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildArgsRejectsInjection(t *testing.T) {
	bad := []FindFileArgs{
		{Path: "-delete"},
		{Path: "relative/dir"},
		{Path: "/tmp", Type: "f -delete"},
		{Path: "/tmp", Mtime: "+1 -exec"},
		{Path: "/tmp", Size: "-fprint"},
	}
	for _, a := range bad {
		if _, err := buildArgs(a); err == nil {
			t.Errorf("buildArgs(%+v) accepted unsafe input", a)
		}
	}
}

func TestBuildArgsPrunesVirtualFS(t *testing.T) {
	got, err := buildArgs(FindFileArgs{Path: "/", Type: "f", Size: "+50M"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/", "(", "-path", "/proc", "-o", "-path", "/sys", "-o", "-path", "/dev", "-o", "-path", "/run", ")", "-prune", "-o",
		"-type", "f", "-size", "+50M", "-printf", `%s\t%y\t%p\n`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}

	// Searching inside a virtual FS explicitly must still work.
	got, _ = buildArgs(FindFileArgs{Path: "/proc/1"})
	if strings.Contains(strings.Join(got, " "), "-prune") {
		t.Errorf("pruned inside an explicitly requested /proc path: %q", got)
	}
	// And a subtree that contains none of them adds no prune clause.
	got, _ = buildArgs(FindFileArgs{Path: "/etc", Name: "hosts*"})
	if strings.Contains(strings.Join(got, " "), "-prune") {
		t.Errorf("unexpected prune for /etc: %q", got)
	}
}

func TestParseOutput(t *testing.T) {
	got := parseOutput("1024\tf\t/etc/hosts\n4096\td\t/etc/with space\n")
	want := []Match{{"/etc/hosts", 1024, "f"}, {"/etc/with space", 4096, "d"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}
