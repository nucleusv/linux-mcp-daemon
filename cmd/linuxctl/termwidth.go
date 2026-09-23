package main

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

// terminalWidth returns the width of the user's terminal, or 0 if there
// isn't one. Falls back to /dev/tty when stdout is piped, so
// `linuxctl ... -o table | more` still fits the screen; COLUMNS is the last
// resort.
func terminalWidth() int {
	if ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ); err == nil && ws.Col > 0 {
		return int(ws.Col)
	}
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		if ws, err := unix.IoctlGetWinsize(int(tty.Fd()), unix.TIOCGWINSZ); err == nil && ws.Col > 0 {
			return int(ws.Col)
		}
	}
	if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n > 0 {
		return n
	}
	return 0
}

// truncateLine cuts a line to width runes, marking the cut with "…".
// width <= 0 means no limit.
func truncateLine(line string, width int) string {
	body := strings.TrimSuffix(line, "\n")
	if width <= 0 || utf8.RuneCountInString(body) <= width {
		return line
	}
	runes := []rune(body)
	out := string(runes[:width-1]) + "…"
	if strings.HasSuffix(line, "\n") {
		out += "\n"
	}
	return out
}

// jsonKeyOrder maps each object key in a JSON document to the position it
// first appears at, so table columns can follow the server's field order
// (decoding into map[string]interface{} loses it).
func jsonKeyOrder(data []byte) map[string]int {
	type frame struct{ isObject, expectKey bool }
	order := map[string]int{}
	var stack []frame
	// valueDone marks that the enclosing object's next token is a key.
	valueDone := func() {
		if n := len(stack); n > 0 && stack[n-1].isObject {
			stack[n-1].expectKey = true
		}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return order
		}
		switch v := tok.(type) {
		case json.Delim:
			if v == '{' || v == '[' {
				stack = append(stack, frame{isObject: v == '{', expectKey: v == '{'})
			} else {
				stack = stack[:len(stack)-1]
				valueDone()
			}
		case string:
			if n := len(stack); n > 0 && stack[n-1].isObject && stack[n-1].expectKey {
				if _, seen := order[v]; !seen {
					order[v] = len(order)
				}
				stack[n-1].expectKey = false
			} else {
				valueDone()
			}
		default:
			valueDone()
		}
	}
}

// yaml11Ambiguous matches plain scalars YAML 1.1 resolves to non-strings:
// sexagesimal numbers (8:0, 1:30:00) and the extended booleans.
var yaml11Ambiguous = regexp.MustCompile(`^([-+]?[0-9][0-9_]*(:[0-5]?[0-9])+(\.[0-9_]*)?|[yY]|[nN]|[yY]es|YES|[nN]o|NO|[oO]n|ON|[oO]ff|OFF)$`)
