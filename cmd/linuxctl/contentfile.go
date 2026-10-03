package main

import (
	"fmt"
	"io"
	"os"
)

// maxContentFile bounds what --content-file will read, so a stray path (or a
// stream that never ends) cannot fill memory. The daemon applies its own,
// tighter limit per tool (64 KiB for a crontab).
const maxContentFile = 16 << 20

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
		var src io.Reader = stdin
		if path != "-" {
			fi, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("--content-file: %v", err)
			}
			if !fi.Mode().IsRegular() { // not a device, pipe or directory
				return fmt.Errorf("--content-file: %s is not a regular file", path)
			}
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("--content-file: %v", err)
			}
			defer f.Close()
			src = f
		}
		b, err := io.ReadAll(io.LimitReader(src, maxContentFile+1))
		if err != nil {
			return fmt.Errorf("--content-file: %v", err)
		}
		if len(b) > maxContentFile {
			return fmt.Errorf("--content-file: more than %d MiB; the text is not sent", maxContentFile>>20)
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
