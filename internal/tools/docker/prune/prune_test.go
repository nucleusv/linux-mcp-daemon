package prune

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

func call(e *fakeengine.Engine, target string, allow []string, format string) (string, error) {
	b, _ := json.Marshal(map[string]interface{}{
		"_docker_socket": e.Socket,
		"_prune":         allow,
		"target":         target,
		"output_format":  format,
	})
	return Prune(b)
}

// T1: images prune hits /images/prune and reports count and reclaimed space.
func TestPruneImages(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"POST /images/prune": `{"ImagesDeleted":[{"Untagged":"nginx:old"},{"Deleted":"sha256:abcdef0123456789"}],"SpaceReclaimed":1048576}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, "images", []string{"images"}, "")
	if err != nil {
		t.Fatal(err)
	}
	// Two API entries, one untag and one delete: one image removed, not two.
	if !strings.Contains(out, "images: 1 removed, 1.0 MiB reclaimed") {
		t.Errorf("unexpected output %q", out)
	}
	query, ok := e.Called("POST", "/images/prune")
	if !ok {
		t.Fatal("did not POST /images/prune")
	}
	// The dangling=false filter would remove every unused image, not just
	// the dangling ones. This tool must never send filters at all.
	if query != "" {
		t.Errorf("prune sent filters %q - the Engine API defaults are the safety margin", query)
	}
}

// T2: one call reclaims one kind. Pruning containers is what makes their
// images dangling and their anonymous volumes unused, so a call that named
// several would delete more than any one request described - there is no way
// to ask for that, and a call for one kind must leave the others alone.
func TestOneTargetPerCall(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"POST /containers/prune": `{"ContainersDeleted":["web-1"],"SpaceReclaimed":100}`,
		"POST /images/prune":     `{"ImagesDeleted":[{"Deleted":"sha256:aaa"}],"SpaceReclaimed":200}`,
		"POST /volumes/prune":    `{"VolumesDeleted":["data"],"SpaceReclaimed":900}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, "containers", []string{"containers", "images", "volumes"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "containers: 1 removed, 100.0 B reclaimed") {
		t.Errorf("unexpected output %q", out)
	}
	var got []string
	for _, r := range e.Requests() {
		got = append(got, r.Path)
	}
	if strings.Join(got, ",") != "/containers/prune" {
		t.Errorf("a one-kind prune touched %v", got)
	}

	// Neither spelling of "two kinds" is a partial success: an array does not
	// unmarshal into the string field, and a joined string is not a target.
	b, _ := json.Marshal(map[string]interface{}{
		"_docker_socket": e.Socket, "_prune": []string{"containers", "images"},
		"target": []string{"containers", "images"},
	})
	if _, err := Prune(b); err == nil || !strings.Contains(err.Error(), "invalid arguments") {
		t.Errorf("a list of targets should refuse, got %v", err)
	}
	if _, err := call(e, "containers,images", []string{"containers", "images"}, ""); err == nil ||
		!strings.Contains(err.Error(), "unknown prune target") {
		t.Errorf("a joined target should refuse, got %v", err)
	}
	if len(e.Requests()) != 1 {
		t.Errorf("a refused call reached the socket: %v", e.Requests())
	}
}

// T3: a target outside the grant is refused before the socket is dialled, so
// an unauthorized request deletes nothing at all.
func TestUnauthorizedTargetDeletesNothing(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"POST /containers/prune": `{"ContainersDeleted":["web-1"],"SpaceReclaimed":100}`,
		"POST /volumes/prune":    `{"VolumesDeleted":["data"],"SpaceReclaimed":900}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	_, err = call(e, "volumes", []string{"containers"}, "")
	if err == nil || !strings.Contains(err.Error(), "not authorized to prune volumes") {
		t.Fatalf("expected refusal, got %v", err)
	}
	if len(e.Requests()) != 0 {
		t.Errorf("refused call still reached the socket: %v", e.Requests())
	}

	// An empty prune: list (a grant without one) refuses everything.
	if _, err := call(e, "containers", nil, ""); err == nil || !strings.Contains(err.Error(), "no prune: list") {
		t.Errorf("empty allowlist should refuse, got %v", err)
	}
	// No target is never "all".
	if _, err := call(e, "", []string{"containers"}, ""); err == nil || !strings.Contains(err.Error(), "target is required") {
		t.Errorf("missing target should refuse, got %v", err)
	}
	// An unknown target is a typo, not a no-op.
	if _, err := call(e, "everything", []string{"containers"}, ""); err == nil || !strings.Contains(err.Error(), "unknown prune target") {
		t.Errorf("unknown target should refuse, got %v", err)
	}
	if len(e.Requests()) != 0 {
		t.Errorf("refused calls still reached the socket: %v", e.Requests())
	}
}

// The structured format is the single result, flat; a failing endpoint is an
// error rather than a report of nothing.
func TestJSONResultAndFailure(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"POST /containers/prune": `{"ContainersDeleted":["web-1"],"SpaceReclaimed":100}`,
		"POST /networks/prune":   `500 {"message":"driver busy"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, "containers", []string{"containers", "networks"}, "json")
	if err != nil {
		t.Fatal(err)
	}
	var got targetResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Target != "containers" || got.Count != 1 || got.Reclaimed != 100 {
		t.Errorf("unexpected result %+v", got)
	}

	if _, err := call(e, "networks", []string{"networks"}, ""); err == nil {
		t.Error("a failed prune should be an error, not an empty report")
	}
}
