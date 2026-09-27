package prune

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

func call(e *fakeengine.Engine, targets, allow []string, format string) (string, error) {
	b, _ := json.Marshal(map[string]interface{}{
		"_docker_socket": e.Socket,
		"_prune":         allow,
		"targets":        targets,
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

	out, err := call(e, []string{"images"}, []string{"images"}, "")
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

// T2: several targets in one call hit each endpoint, in dependency order
// whatever order they were asked for.
func TestMultipleTargetsInDependencyOrder(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"POST /containers/prune": `{"ContainersDeleted":["web-1"],"SpaceReclaimed":100}`,
		"POST /images/prune":     `{"ImagesDeleted":[{"Deleted":"sha256:aaa"}],"SpaceReclaimed":200}`,
		"POST /build/prune":      `{"CachesDeleted":["c1","c2"],"SpaceReclaimed":300}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, []string{"build-cache", "images", "containers"}, []string{"containers", "images", "build-cache"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Total reclaimed: 600.0 B") {
		t.Errorf("total not summed: %q", out)
	}
	var got []string
	for _, r := range e.Requests() {
		got = append(got, r.Path)
	}
	want := []string{"/containers/prune", "/images/prune", "/build/prune"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("wrong order: got %v, want %v", got, want)
	}
}

// T3: a target outside the grant refuses the whole call - including the
// targets that *were* granted, so a partly-unauthorized request deletes
// nothing at all.
func TestUnauthorizedTargetDeletesNothing(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"POST /containers/prune": `{"ContainersDeleted":["web-1"],"SpaceReclaimed":100}`,
		"POST /volumes/prune":    `{"VolumesDeleted":["data"],"SpaceReclaimed":900}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	_, err = call(e, []string{"containers", "volumes"}, []string{"containers"}, "")
	if err == nil || !strings.Contains(err.Error(), "not authorized to prune volumes") {
		t.Fatalf("expected refusal, got %v", err)
	}
	if len(e.Requests()) != 0 {
		t.Errorf("refused call still reached the socket: %v", e.Requests())
	}

	// An empty prune: list (a grant without one) refuses everything.
	if _, err := call(e, []string{"containers"}, nil, ""); err == nil || !strings.Contains(err.Error(), "no prune: list") {
		t.Errorf("empty allowlist should refuse, got %v", err)
	}
	// No targets is never "all".
	if _, err := call(e, nil, []string{"containers"}, ""); err == nil || !strings.Contains(err.Error(), "targets is required") {
		t.Errorf("missing targets should refuse, got %v", err)
	}
	// An unknown target is a typo, not a no-op.
	if _, err := call(e, []string{"everything"}, []string{"containers"}, ""); err == nil || !strings.Contains(err.Error(), "unknown prune target") {
		t.Errorf("unknown target should refuse, got %v", err)
	}
	if len(e.Requests()) != 0 {
		t.Errorf("refused calls still reached the socket: %v", e.Requests())
	}
}

// A partial failure still reports what was deleted; an all-failed call is an
// error, not a report.
func TestPartialFailure(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"POST /containers/prune": `{"ContainersDeleted":["web-1"],"SpaceReclaimed":100}`,
		"POST /networks/prune":   `500 {"message":"driver busy"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, []string{"containers", "networks"}, []string{"containers", "networks"}, "json")
	if err != nil {
		t.Fatalf("partial failure should still report: %v", err)
	}
	var got struct {
		Results []targetResult `json:"results"`
		Total   int64          `json:"total_space_reclaimed_bytes"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 2 || got.Results[0].Count != 1 || got.Results[1].Error == "" {
		t.Errorf("unexpected results %+v", got.Results)
	}
	if got.Total != 100 {
		t.Errorf("total = %d, want 100", got.Total)
	}

	if _, err := call(e, []string{"networks"}, []string{"networks"}, ""); err == nil {
		t.Error("a call where every target failed should be an error")
	}
}
