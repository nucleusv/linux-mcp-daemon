// Package exec runs one command inside a running container and returns its
// output. There is no TTY and no interactive attach: one command, its argv,
// stdout, stderr and an exit code.
//
// This is the sharpest tool in the docker group - arbitrary code as root
// inside whatever container it targets - which is why its containers:
// allowlist is per-tool and separate from docker/manage's by construction.
package exec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

const (
	maxOutputBytes = 1 << 20
	defaultTimeout = 30
	maxTimeout     = 300
)

type Args struct {
	docker.CommonArgs
	Container    string   `json:"container,omitempty"`
	Command      []string `json:"command,omitempty"`
	User         string   `json:"user,omitempty"`
	WorkingDir   string   `json:"working_dir,omitempty"`
	Timeout      int      `json:"timeout,omitempty"`
	OutputFormat string   `json:"output_format,omitempty"`
}

func Exec(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}
	if len(args.Command) == 0 {
		return "", fmt.Errorf("command is required: an argv array, e.g. [\"sh\", \"-c\", \"ls /app\"]")
	}

	timeout := args.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	c := args.Client(time.Duration(timeout) * time.Second)
	id, name, err := docker.Authorize(c, args.Container, args.Containers)
	if err != nil {
		return "", err
	}

	var created struct {
		ID string `json:"Id"`
	}
	create := map[string]interface{}{
		"AttachStdout": true,
		"AttachStderr": true,
		"AttachStdin":  false,
		"Tty":          false,
		"Cmd":          args.Command,
	}
	if args.User != "" {
		create["User"] = args.User
	}
	if args.WorkingDir != "" {
		create["WorkingDir"] = args.WorkingDir
	}
	if err := c.PostJSON("/containers/"+id+"/exec", create, &created); err != nil {
		return "", err
	}

	body, err := c.PostStream("/exec/"+created.ID+"/start", map[string]interface{}{"Detach": false, "Tty": false})
	if err != nil {
		// A command that produces nothing and never exits trips the timeout
		// here, before any header - same situation, same explanation.
		return "", timedOut(name, timeout, err)
	}
	stdout, stderr, err := docker.Demux(body, maxOutputBytes)
	body.Close()
	if err != nil {
		return "", timedOut(name, timeout, err)
	}

	var state struct {
		ExitCode int  `json:"ExitCode"`
		Running  bool `json:"Running"`
	}
	if err := c.GetJSON("/exec/"+created.ID+"/json", &state); err != nil {
		return "", err
	}

	if docker.Structured(args.OutputFormat) {
		b, _ := json.Marshal(map[string]interface{}{
			"container": name,
			"id":        id,
			"command":   args.Command,
			"exit_code": state.ExitCode,
			"running":   state.Running,
			"stdout":    stdout,
			"stderr":    stderr,
		})
		return string(b), nil
	}

	var text strings.Builder
	fmt.Fprintf(&text, "Container %s (%s): %s\nExit code: %d\n", name, id[:12], strings.Join(args.Command, " "), state.ExitCode)
	if stdout != "" {
		fmt.Fprintf(&text, "\n--- stdout ---\n%s", stdout)
		if !strings.HasSuffix(stdout, "\n") {
			text.WriteString("\n")
		}
	}
	if stderr != "" {
		fmt.Fprintf(&text, "\n--- stderr ---\n%s", stderr)
		if !strings.HasSuffix(stderr, "\n") {
			text.WriteString("\n")
		}
	}
	if stdout == "" && stderr == "" {
		text.WriteString("\n(no output)\n")
	}
	return text.String(), nil
}

// timedOut explains a cut-off exec. The Engine API has no endpoint to cancel a
// running exec, so the message must not imply the command died with it.
func timedOut(name string, timeout int, err error) error {
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "Client.Timeout") {
		return err
	}
	return fmt.Errorf("exec in %s did not finish within %ds: %v (the command may still be running inside the container - the Engine API has no way to cancel an exec)", name, timeout, err)
}
