package main

import "testing"

func TestJSONKeyOrder(t *testing.T) {
	order := jsonKeyOrder([]byte(`[{"pid":1,"user":"root","nested":{"z":1,"a":[1,"x"]},"cmdline":"init"},{"pid":2,"user":"x","nested":{},"cmdline":""}]`))
	want := []string{"pid", "user", "nested", "z", "a", "cmdline"}
	for i, k := range want {
		if order[k] != i {
			t.Errorf("order[%q] = %d, want %d (full: %v)", k, order[k], i, order)
		}
	}
}

func TestTruncateLine(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"short\n", 10, "short\n"},
		{"exactly10!\n", 10, "exactly10!\n"},
		{"this is too long\n", 10, "this is t…\n"},
		{"no limit at all\n", 0, "no limit at all\n"},
		{"├─ünïcödé-long\n", 5, "├─ün…\n"},
	}
	for _, c := range cases {
		if got := truncateLine(c.in, c.width); got != c.want {
			t.Errorf("truncateLine(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}
