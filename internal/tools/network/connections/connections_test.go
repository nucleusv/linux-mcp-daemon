package connections

import "testing"

func TestSSState(t *testing.T) {
	cases := map[string]string{
		"LISTEN":      "listening",
		"listening":   "listening",
		"ESTABLISHED": "established",
		"TIME_WAIT":   "time-wait",
		"close_wait":  "close-wait",
		"SYN_RECV":    "syn-recv",
		" Listen ":    "listening",
	}
	for in, want := range cases {
		got, err := ssState(in)
		if err != nil || got != want {
			t.Errorf("ssState(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ssState("bogus; rm -rf /"); err == nil {
		t.Error("ssState accepted an unknown state")
	}
}
