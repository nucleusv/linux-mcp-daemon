// Package list implements timers/list: the systemd timers of the host, with
// what they trigger, their schedule and their next and last run. Read-only,
// over the same DBus connection services/list uses.
package list

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"golang.org/x/sys/unix"
)

type GetListArgs struct {
	Pattern      string `json:"pattern,omitempty"`
	ActiveState  string `json:"active_state,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// timer is one row of the result.
type timer struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	LoadState   string   `json:"load_state"`
	ActiveState string   `json:"active_state"`
	SubState    string   `json:"sub_state"`
	Unit        string   `json:"unit"`       // the unit the timer starts
	NextRun     string   `json:"next_run"`   // RFC 3339 UTC, or "never"
	LastRun     string   `json:"last_run"`   // RFC 3339 UTC, or "never"
	Schedule    []string `json:"schedule"`   // OnCalendar=... / OnBootSec=... entries
	Persistent  bool     `json:"persistent"` // catches up on runs missed while the host was off
	Result      string   `json:"result"`
}

func List(argsJSON []byte) (string, error) {
	var args GetListArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	ctx := context.Background()
	conn, err := dbus.NewWithContext(ctx) // private socket as root, the system bus otherwise
	if err != nil {
		return "", fmt.Errorf("failed to connect to systemd dbus: %v", err)
	}
	defer conn.Close()

	units, err := conn.ListUnitsContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list units: %v", err)
	}

	var rows []timer
	for _, u := range units {
		if !strings.HasSuffix(u.Name, ".timer") {
			continue
		}
		if args.ActiveState != "" && u.ActiveState != args.ActiveState {
			continue
		}
		if !matchPattern(u.Name, args.Pattern) {
			continue
		}
		t := timer{
			Name: u.Name, Description: u.Description, LoadState: u.LoadState,
			ActiveState: u.ActiveState, SubState: u.SubState,
			NextRun: "never", LastRun: "never", Schedule: []string{},
		}
		// A timer whose properties cannot be read is still listed, with what ListUnits knows.
		if props, err := conn.GetUnitTypePropertiesContext(ctx, u.Name, "Timer"); err == nil {
			fillFromProperties(&t, props)
		}
		rows = append(rows, t)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

	if isStructured(args.OutputFormat) {
		if rows == nil {
			return "[]", nil
		}
		b, err := json.Marshal(rows)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %v", err)
		}
		return string(b), nil
	}
	if len(rows) == 0 {
		return "No timers found matching the criteria.", nil
	}
	var sb strings.Builder
	for _, t := range rows {
		fmt.Fprintf(&sb, "[%s] %s\n  Next: %s | Last: %s | Runs: %s\n", t.ActiveState, t.Name, t.NextRun, t.LastRun, orDash(t.Unit))
		if len(t.Schedule) > 0 {
			fmt.Fprintf(&sb, "  Schedule: %s\n", strings.Join(t.Schedule, "; "))
		}
		fmt.Fprintf(&sb, "  Desc: %s\n\n", t.Description)
	}
	return sb.String(), nil
}

func isStructured(f string) bool { return f == "json" || f == "yaml" || f == "table" || f == "wide" }

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// matchPattern is the wildcard match services/list uses: a leading and/or
// trailing * anywhere else is literal.
func matchPattern(name, pattern string) bool {
	switch {
	case pattern == "":
		return true
	case strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") && len(pattern) > 1:
		return strings.Contains(name, strings.Trim(pattern, "*"))
	case strings.HasPrefix(pattern, "*"):
		return strings.HasSuffix(name, strings.TrimPrefix(pattern, "*"))
	case strings.HasSuffix(pattern, "*"):
		return strings.HasPrefix(name, strings.TrimSuffix(pattern, "*"))
	}
	return name == pattern
}

// fillFromProperties copies the org.freedesktop.systemd1.Timer properties into t.
func fillFromProperties(t *timer, props map[string]interface{}) {
	if s, ok := props["Unit"].(string); ok {
		t.Unit = s
	}
	t.NextRun = usecToTime(props["NextElapseUSecRealtime"])
	if t.NextRun == "never" { // OnBootSec=/OnUnitActiveSec=... timers keep their next run on the monotonic clock
		t.NextRun = monotonicNext(props["NextElapseUSecMonotonic"], time.Now(), monotonicNow())
	}
	t.LastRun = usecToTime(props["LastTriggerUSec"])
	if b, ok := props["Persistent"].(bool); ok {
		t.Persistent = b
	}
	if s, ok := props["Result"].(string); ok {
		t.Result = s
	}
	t.Schedule = append(calendarSpecs(props["TimersCalendar"]), monotonicSpecs(props["TimersMonotonic"])...)
	if t.Schedule == nil {
		t.Schedule = []string{}
	}
}

// usecToTime turns systemd's microseconds since the epoch into an RFC 3339 UTC
// timestamp; 0 (and the "infinity" values systemd uses for unset) is "never".
func usecToTime(v interface{}) string {
	u, ok := v.(uint64)
	if !ok || u == 0 || u >= math.MaxInt64/1000 {
		return "never"
	}
	return time.UnixMicro(int64(u)).UTC().Format(time.RFC3339)
}

// monotonicNow is the current CLOCK_MONOTONIC reading, the clock systemd's
// monotonic timers count on (0 if it cannot be read).
func monotonicNow() time.Duration {
	var ts unix.Timespec
	if unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts) != nil {
		return 0
	}
	return time.Duration(ts.Nano())
}

// monotonicNext converts NextElapseUSecMonotonic (microseconds on CLOCK_MONOTONIC)
// to an RFC 3339 UTC time: now + (next - nowMono). 0 or unreadable is "never".
func monotonicNext(v interface{}, now time.Time, nowMono time.Duration) string {
	u, ok := v.(uint64)
	if !ok || u == 0 || u >= math.MaxInt64/1000 || nowMono == 0 {
		return "never"
	}
	next := time.Duration(u) * time.Microsecond
	return now.Add(next - nowMono).UTC().Format(time.RFC3339)
}

// calendarSpecs reads TimersCalendar, an array of (base, spec, next_elapse).
func calendarSpecs(v interface{}) []string {
	var out []string
	for _, e := range structs(v) {
		if len(e) >= 2 {
			base, _ := e[0].(string)
			spec, _ := e[1].(string)
			out = append(out, base+"="+spec)
		}
	}
	return out
}

// monotonicSpecs reads TimersMonotonic, an array of (base, value_usec, next_elapse).
func monotonicSpecs(v interface{}) []string {
	var out []string
	for _, e := range structs(v) {
		if len(e) >= 2 {
			base, _ := e[0].(string)
			usec, _ := e[1].(uint64)
			// DBus names them OnBootUSec etc.; unit files (and people) write OnBootSec.
			out = append(out, strings.TrimSuffix(base, "USec")+"Sec="+(time.Duration(usec)*time.Microsecond).String())
		}
	}
	return out
}

// structs flattens the DBus array-of-struct value (its Go shape depends on the
// decoder: [][]interface{} or []interface{} of []interface{}) into rows.
func structs(v interface{}) [][]interface{} {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return nil
	}
	var out [][]interface{}
	for i := 0; i < rv.Len(); i++ {
		el := rv.Index(i)
		for el.Kind() == reflect.Interface {
			el = el.Elem()
		}
		if el.Kind() != reflect.Slice && el.Kind() != reflect.Array {
			continue
		}
		row := make([]interface{}, el.Len())
		for j := range row {
			row[j] = el.Index(j).Interface()
		}
		out = append(out, row)
	}
	return out
}
