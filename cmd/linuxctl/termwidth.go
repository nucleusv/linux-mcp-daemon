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

// yaml11Ambiguous matches plain scalars YAML 1.1 resolves to non-strings:
// sexagesimal numbers (8:0, 1:30:00) and the extended booleans.
var yaml11Ambiguous = regexp.MustCompile(`^([-+]?[0-9][0-9_]*(:[0-5]?[0-9])+(\.[0-9_]*)?|[yY]|[nN]|[yY]es|YES|[nN]o|NO|[oO]n|ON|[oO]ff|OFF)$`)

// objectKeys returns the keys of a JSON object in document order (nil if
// data isn't an object) - decoding into a Go map would lose that order.
func objectKeys(data []byte) []string {
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return keys
		}
		key, ok := tok.(string)
		if !ok {
			return keys
		}
		keys = append(keys, key)
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return keys
		}
	}
	return keys
}
