package manage

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

const fullID = "aaaaaaaaaaaa1111111111"

func call(e *fakeengine.Engine, container, action string, allow []string) (string, error) {
	b, _ := json.Marshal(map[string]interface{}{
		"_docker_socket": e.Socket,
		"_containers":    allow,
		"container":      container,
		"action":         action,
	})
	return Manage(b)
}

// T2: every lifecycle verb POSTs its own endpoint, on the resolved ID.
func TestLifecycleEndpoints(t *testing.T) {
	routes := map[string]string{"GET /containers/web-1/json": fakeengine.Inspect(fullID, "web-1")}
	for _, verb := range []string{"start", "stop", "restart", "kill", "pause", "unpause"} {
		routes["POST /containers/"+fullID+"/"+verb] = ""
	}
	e, err := fakeengine.New(routes)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	for _, verb := range []string{"start", "stop", "restart", "kill", "pause", "unpause"} {
		out, err := call(e, "web-1", verb, []string{"web-*"})
		if err != nil {
			t.Fatalf("%s: %v", verb, err)
		}
		if !strings.Contains(out, "web-1 (aaaaaaaaaaaa): "+verb+" succeeded") {
			t.Errorf("%s: unexpected output %q", verb, out)
		}
		if _, ok := e.Called("POST", "/containers/"+fullID+"/"+verb); !ok {
			t.Errorf("%s did not POST /containers/<id>/%s", verb, verb)
		}
	}

	if _, err := call(e, "web-1", "destroy", []string{"*"}); err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("unknown action should be refused, got %v", err)
	}
}

// T5d: remove is a bare DELETE - never force, never v - and a running
// container comes back as advice to stop or kill it first.
func TestRemoveIsFenced(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"GET /containers/web-1/json":   fakeengine.Inspect(fullID, "web-1"),
		"DELETE /containers/" + fullID: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	if _, err := call(e, "web-1", "remove", []string{"web-1"}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	q, ok := e.Called("DELETE", "/containers/"+fullID)
	if !ok {
		t.Fatal("remove did not DELETE the container")
	}
	if q != "" {
		t.Errorf("remove must send no query parameters at all (no force, no v), got %q", q)
	}

	busy, err := fakeengine.New(map[string]string{
		"GET /containers/web-1/json":   fakeengine.Inspect(fullID, "web-1"),
		"DELETE /containers/" + fullID: `409 {"message":"You cannot remove a running container"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()

	_, err = call(busy, "web-1", "remove", []string{"web-1"})
	if err == nil {
		t.Fatal("removing a running container should fail")
	}
	if !strings.Contains(err.Error(), "stop or kill it first") {
		t.Errorf("409 should say what to do instead, got %v", err)
	}
}

// T5b/T5c: the allowlist is this tool's own, and a container outside it is
// refused after resolution without any lifecycle call being made.
func TestAllowlistBlocksBeforeActing(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"GET /containers/db-1/json": fakeengine.Inspect("bbbbbbbbbbbb2222", "db-1"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	if _, err := call(e, "db-1", "stop", []string{"web-*"}); err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("container outside the list should be refused, got %v", err)
	}
	for _, r := range e.Requests() {
		if r.Method != "GET" {
			t.Errorf("nothing but the resolve read should have happened, saw %s %s", r.Method, r.Path)
		}
	}

	// An empty list refuses before the socket is touched at all.
	empty, err := fakeengine.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer empty.Close()
	if _, err := call(empty, "web-1", "stop", nil); err == nil || !strings.Contains(err.Error(), "no containers: list") {
		t.Fatalf("missing containers: list should be refused, got %v", err)
	}
	if len(empty.Requests()) != 0 {
		t.Errorf("an empty allowlist must refuse before dialing, saw %v", empty.Requests())
	}
}

// Docker answers 304 when the container is already in the requested state -
// a no-op, not an error the agent should retry.
func TestAlreadyInStateIsNotAnError(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"GET /containers/web-1/json":            fakeengine.Inspect(fullID, "web-1"),
		"POST /containers/" + fullID + "/start": "304 {}",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, "web-1", "start", []string{"web-*"})
	if err != nil {
		t.Fatalf("304 should not be an error: %v", err)
	}
	if !strings.Contains(out, "already in that state") {
		t.Errorf("unexpected output %q", out)
	}
}
