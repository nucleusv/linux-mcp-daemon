package logs

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

const fullID = "aaaaaaaaaaaa1111111111"

func call(e *fakeengine.Engine, args map[string]interface{}) (string, error) {
	args["_docker_socket"] = e.Socket
	if _, set := args["_containers"]; !set {
		args["_containers"] = []string{"web-*"}
	}
	b, _ := json.Marshal(args)
	return Logs(b)
}

// Both streams come back interleaved in recorded order - splitting them the
// way exec does would lose the chronology `docker logs` shows.
func TestLogsInterleaveStreams(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"GET /containers/web-1/json":          fakeengine.Inspect(fullID, "web-1"),
		"GET /containers/" + fullID + "/logs": fakeengine.Frame(1, "start\n") + fakeengine.Frame(2, "warn\n") + fakeengine.Frame(1, "ready\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, map[string]interface{}{"container": "web-1"})
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if out != "start\nwarn\nready\n" {
		t.Errorf("streams should stay in order, got %q", out)
	}

	q, _ := e.Called("GET", "/containers/"+fullID+"/logs")
	for _, want := range []string{"stdout=true", "stderr=true", "tail=100"} {
		if !strings.Contains(q, want) {
			t.Errorf("default query should contain %s, got %q", want, q)
		}
	}
	if strings.Contains(q, "follow") {
		t.Errorf("this tool never follows a log stream, query was %q", q)
	}
}

func TestLogsParameters(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"GET /containers/web-1/json":          fakeengine.Inspect(fullID, "web-1"),
		"GET /containers/" + fullID + "/logs": fakeengine.Frame(1, "x\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, map[string]interface{}{
		"container":     "web-1",
		"lines":         5,
		"since":         "2026-09-27T10:00:00Z",
		"timestamps":    true,
		"stderr":        false,
		"output_format": "json",
	})
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output_format json is not JSON: %v", err)
	}
	if got["logs"] != "x\n" || got["container"] != "web-1" {
		t.Errorf("unexpected structured output: %v", got)
	}
	if got["lines"] != float64(5) {
		t.Errorf("lines should be the number asked for, got %#v", got["lines"])
	}

	// The Engine API takes only a unix timestamp for since/until - sending the
	// RFC3339 the schema advertises verbatim gets a 400 back from Docker.
	q, _ := e.Called("GET", "/containers/"+fullID+"/logs")
	for _, want := range []string{"tail=5", "timestamps=true", "since=1790503200", "stdout=true"} {
		if !strings.Contains(q, want) {
			t.Errorf("query should contain %s, got %q", want, q)
		}
	}
	if strings.Contains(q, "stderr") {
		t.Errorf("stderr:false should be omitted, not sent as false, got %q", q)
	}
}

func TestLogsNeedsGrant(t *testing.T) {
	e, err := fakeengine.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	if _, err := call(e, map[string]interface{}{"container": "web-1", "_containers": []string{}}); err == nil || !strings.Contains(err.Error(), "no containers: list") {
		t.Fatalf("logs carry secrets and need their own list, got %v", err)
	}
	if len(e.Requests()) != 0 {
		t.Errorf("must refuse before dialing, saw %v", e.Requests())
	}
}
