package exec

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

const fullID = "aaaaaaaaaaaa1111111111"

func engine(t *testing.T, exit string) *fakeengine.Engine {
	t.Helper()
	e, err := fakeengine.New(map[string]string{
		"GET /containers/web-1/json":           fakeengine.Inspect(fullID, "web-1"),
		"POST /containers/" + fullID + "/exec": `{"Id":"exec123"}`,
		"POST /exec/exec123/start":             fakeengine.Frame(1, "hello\n") + fakeengine.Frame(2, "oops\n") + fakeengine.Frame(1, "bye\n"),
		"GET /exec/exec123/json":               exit,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func call(e *fakeengine.Engine, args map[string]interface{}) (string, error) {
	args["_docker_socket"] = e.Socket
	b, _ := json.Marshal(args)
	return Exec(b)
}

// T3: one command, its argv, demuxed stdout and stderr, and the exit code
// read back from the exec inspect rather than guessed.
func TestExec(t *testing.T) {
	e := engine(t, `{"ExitCode":3,"Running":false}`)
	defer e.Close()

	out, err := call(e, map[string]interface{}{
		"_containers": []string{"web-*"},
		"container":   "web-1",
		"command":     []string{"sh", "-c", "echo hello"},
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	for _, want := range []string{"Container web-1 (aaaaaaaaaaaa): sh -c echo hello", "Exit code: 3", "--- stdout ---\nhello\nbye\n", "--- stderr ---\noops\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	// The create request must be a non-interactive, non-TTY exec of the argv
	// as given - never a shell line assembled here.
	var body map[string]interface{}
	for _, r := range e.Requests() {
		if r.Path == "/containers/"+fullID+"/exec" {
			if err := json.Unmarshal([]byte(r.Body), &body); err != nil {
				t.Fatalf("exec create body: %v", err)
			}
		}
	}
	if body == nil {
		t.Fatal("no exec create request")
	}
	if body["Tty"] != false || body["AttachStdin"] != false {
		t.Errorf("exec must be non-TTY with no stdin, got %v", body)
	}
	cmd, _ := json.Marshal(body["Cmd"])
	if string(cmd) != `["sh","-c","echo hello"]` {
		t.Errorf("argv should pass through unchanged, got %s", cmd)
	}
}

// T3, timeout half: the client gives up, and the error says the command may
// still be running - the Engine API has no way to cancel an exec.
func TestExecTimeout(t *testing.T) {
	e := engine(t, `{"ExitCode":0,"Running":true}`)
	defer e.Close()
	e.Delay("POST", "/exec/exec123/start", 2*time.Second)

	_, err := call(e, map[string]interface{}{
		"_containers": []string{"*"},
		"container":   "web-1",
		"command":     []string{"sleep", "60"},
		"timeout":     1,
	})
	if err == nil {
		t.Fatal("a command that outlives the timeout should fail")
	}
	if !strings.Contains(err.Error(), "did not finish within 1s") || !strings.Contains(err.Error(), "may still be running") {
		t.Errorf("the timeout error should not imply the command died, got %v", err)
	}
}

func TestExecJSONAndValidation(t *testing.T) {
	e := engine(t, `{"ExitCode":0,"Running":false}`)
	defer e.Close()

	out, err := call(e, map[string]interface{}{
		"_containers":   []string{"*"},
		"container":     "web-1",
		"command":       []string{"true"},
		"output_format": "json",
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output_format json is not JSON: %v", err)
	}
	if got["exit_code"] != float64(0) || got["stdout"] != "hello\nbye\n" || got["stderr"] != "oops\n" {
		t.Errorf("unexpected structured output: %v", got)
	}

	if _, err := call(e, map[string]interface{}{"_containers": []string{"*"}, "container": "web-1"}); err == nil || !strings.Contains(err.Error(), "argv array") {
		t.Errorf("a missing command should say it wants an argv array, got %v", err)
	}
}
