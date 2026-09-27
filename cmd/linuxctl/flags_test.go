package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// The three ways a flag's value used to be misread, all found live against the
// docker tools: --stdout=false became a flag named "stdout=false" (so the
// boolean read as true, the opposite of what was typed), "-o" after a boolean
// flag was swallowed as that flag's value, and an array parameter had no
// client syntax at all.
func TestSplitFlagsAndPositional(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		wantFlags  map[string]interface{}
		wantPos    []string
		wantOutput string
	}{
		{
			name:      "equals form",
			args:      []string{"--stdout=false", "--lines=4"},
			wantFlags: map[string]interface{}{"stdout": false, "lines": 4},
		},
		{
			name:      "space form",
			args:      []string{"--stdout", "false", "--lines", "4"},
			wantFlags: map[string]interface{}{"stdout": false, "lines": 4},
		},
		{
			name:       "-o after a boolean flag is not its value",
			args:       []string{"docker", "--all", "-o", "table"},
			wantFlags:  map[string]interface{}{"all": true, "output_format": "table"},
			wantPos:    []string{"docker"},
			wantOutput: "table",
		},
		{
			name:       "--output=json",
			args:       []string{"--output=json"},
			wantFlags:  map[string]interface{}{"output_format": "json"},
			wantOutput: "json",
		},
		{
			name:      "negative numbers stay values",
			args:      []string{"--boot-offset", "-1"},
			wantFlags: map[string]interface{}{"boot-offset": -1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags, pos, out := splitFlagsAndPositional(tc.args)
			if !reflect.DeepEqual(flags, tc.wantFlags) {
				t.Errorf("flags = %#v, want %#v", flags, tc.wantFlags)
			}
			if !reflect.DeepEqual(pos, tc.wantPos) {
				t.Errorf("positional = %#v, want %#v", pos, tc.wantPos)
			}
			if out != tc.wantOutput {
				t.Errorf("outputFormat = %q, want %q", out, tc.wantOutput)
			}
		})
	}
}

// A call must fail with a message when the SSE session drops mid-flight.
// Waiting on the response channel alone made the Go runtime abort the client
// with "all goroutines are asleep - deadlock!" - seen live by stopping the
// container mcpd itself runs in.
func TestCallGivesUpWhenSessionEnds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted) // accepted; the answer would come over SSE
	}))
	defer srv.Close()

	postEndpoint = srv.URL
	ended := make(chan struct{})
	close(ended)
	sessionEnded = ended

	_, err := tryCallMethod("token", "1", "tools/call", nil, 0)
	if err == nil || !strings.Contains(err.Error(), "closed the connection") {
		t.Fatalf("err = %v, want the session-ended message", err)
	}
}
