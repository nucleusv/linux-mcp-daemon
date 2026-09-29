package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// toolText returns the first text of a tools/call result and whether it is an error.
func toolText(r JSONRPCResponse) (string, bool) {
	if r.Error != nil {
		return r.Error.Message, true
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	json.Unmarshal(r.Result, &res)
	if len(res.Content) == 0 {
		return "", res.IsError
	}
	return res.Content[0].Text, res.IsError
}

// renderRawText prints a tool's text exactly as returned (or its error to
// stderr with a non-zero exit): for a crontab, every byte counts.
func renderRawText(r JSONRPCResponse) {
	text, isErr := toolText(r)
	if isErr {
		fmt.Fprintln(os.Stderr, text)
		os.Exit(1)
	}
	fmt.Print(text)
}

// editCrontab implements `linuxctl edit crontabs [user] [--privileged true]`,
// like `crontab -e`: read the crontab and remember its hash, edit a temporary
// copy in $EDITOR, and write it back only if it changed - refused by the
// daemon if someone changed the crontab in the meantime (if_match).
func editCrontab(authToken string, positional []string, flags map[string]interface{}) {
	args := map[string]interface{}{"output_format": "json"}
	if len(positional) > 0 {
		args["user"] = positional[0]
	}
	if v, ok := flags["privileged"]; ok {
		args["privileged"] = v
	}
	text, isErr := toolText(callMethod(authToken, nextID(), "tools/call", map[string]interface{}{"name": "cron/manage", "arguments": args}))
	if isErr {
		fmt.Fprintln(os.Stderr, text)
		os.Exit(1)
	}
	var cur struct {
		User    string `json:"user"`
		SHA256  string `json:"sha256"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(text), &cur); err != nil {
		fmt.Fprintf(os.Stderr, "Error: unexpected answer from cron/manage: %v\n", err)
		os.Exit(1)
	}

	tmp, err := os.CreateTemp("", "crontab-*.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmp.Name())
	os.Chmod(tmp.Name(), 0o600) // a crontab can hold secrets
	tmp.WriteString(cur.Content)
	tmp.Close()

	if err := runEditor(tmp.Name()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v - the crontab is unchanged\n", err)
		os.Exit(1)
	}
	edited, err := os.ReadFile(tmp.Name())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if string(edited) == cur.Content || sumHex(string(edited)) == sumHex(cur.Content) {
		fmt.Printf("No changes to the crontab of %s.\n", cur.User)
		return
	}
	args["content"] = string(edited)
	args["if_match"] = cur.SHA256
	delete(args, "output_format")
	out, isErr := toolText(callMethod(authToken, nextID(), "tools/call", map[string]interface{}{"name": "cron/manage", "arguments": args}))
	if isErr {
		msg := out
		if strings.Contains(msg, "changed since you read it") {
			msg += "\nYour edit was not saved; it is kept in " + tmp.Name() + " (read the crontab again, merge, and update)."
		}
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
	fmt.Println(out)
}

func sumHex(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
