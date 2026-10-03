package main

import (
	"fmt"
	"io"
	"os"
)

// applyContentFlags handles the client-side flags around a tool's `content`
// argument, so text never has to pass through the caller's shell:
//
//	--content-file PATH   read the text from a local file ("-" = stdin)
//	--clear               needed to send an EMPTY crontab (cron/manage only)
//
// The text is sent as-is; nothing in it is interpreted here or by the daemon.
// An empty crontab is refused without --clear because the usual way to get one
// by accident is `--content "$(cat missing.txt)"`, where the shell hands over
// an empty string after cat failed.
func applyContentFlags(tool string, schema map[string]interface{}, flags map[string]interface{}, stdin io.Reader) error {
	props, _ := schema["properties"].(map[string]interface{})
	_, hasContent := props["content"]

	if v, ok := flags["content-file"]; ok {
		delete(flags, "content-file")
		if !hasContent {
			return fmt.Errorf("--content-file: %s has no content argument", tool)
		}
		if _, dup := flags["content"]; dup {
			return fmt.Errorf("use either --content or --content-file, not both")
		}
		path := fmt.Sprint(v) // flagValue may have typed "123" as a number
		if path == "true" {
			return fmt.Errorf("--content-file needs a path (or - for stdin)")
		}
		var b []byte
		var err error
		if path == "-" {
			b, err = io.ReadAll(stdin)
		} else {
			b, err = os.ReadFile(path)
		}
		if err != nil {
			return fmt.Errorf("--content-file: %v", err)
		}
		flags["content"] = string(b)
	}

	clear, hasClear := flags["clear"]
	if tool != "cron/manage" {
		return nil
	}
	delete(flags, "clear")
	content, hasC := flags["content"]
	wantClear := hasClear && clear == true
	if hasC && fmt.Sprint(content) == "" && !wantClear {
		return fmt.Errorf("refusing to replace the crontab with an empty one (a missing or empty file?): to delete it on purpose, pass --clear")
	}
	if wantClear {
		if hasC && fmt.Sprint(content) != "" {
			return fmt.Errorf("--clear deletes the crontab; do not combine it with content")
		}
		flags["content"] = ""
	}
	return nil
}
