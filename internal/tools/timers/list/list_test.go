package list

import (
	"math"
	"reflect"
	"testing"
)

func TestUsecToTime(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{uint64(0), "never"},
		{uint64(math.MaxUint64), "never"},
		{nil, "never"},
		{"x", "never"},
		{uint64(1_700_000_000_000_000), "2023-11-14T22:13:20Z"},
	}
	for _, c := range cases {
		if got := usecToTime(c.in); got != c.want {
			t.Errorf("usecToTime(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestScheduleParsing(t *testing.T) {
	// The two shapes a DBus decoder can hand back for array-of-struct.
	cal := [][]interface{}{{"OnCalendar", "*-*-* 06,18:00:00", uint64(1)}}
	if got := calendarSpecs(cal); !reflect.DeepEqual(got, []string{"OnCalendar=*-*-* 06,18:00:00"}) {
		t.Errorf("calendar: %v", got)
	}
	cal2 := []interface{}{[]interface{}{"OnCalendar", "daily", uint64(1)}}
	if got := calendarSpecs(cal2); !reflect.DeepEqual(got, []string{"OnCalendar=daily"}) {
		t.Errorf("calendar (nested interface): %v", got)
	}
	mono := [][]interface{}{{"OnBootUSec", uint64(15 * 60 * 1_000_000), uint64(0)}}
	if got := monotonicSpecs(mono); !reflect.DeepEqual(got, []string{"OnBootSec=15m0s"}) {
		t.Errorf("monotonic: %v", got)
	}
	if calendarSpecs(nil) != nil || monotonicSpecs("nope") != nil {
		t.Error("garbage input must yield no specs")
	}
}

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		name, pat string
		want      bool
	}{
		{"apt-daily.timer", "", true},
		{"apt-daily.timer", "apt*", true},
		{"apt-daily.timer", "*.timer", true},
		{"apt-daily.timer", "*daily*", true},
		{"apt-daily.timer", "logrotate*", false},
		{"apt-daily.timer", "apt-daily.timer", true},
		{"apt-daily.timer", "apt", false},
	}
	for _, c := range cases {
		if got := matchPattern(c.name, c.pat); got != c.want {
			t.Errorf("matchPattern(%q, %q) = %v", c.name, c.pat, got)
		}
	}
}

func TestFillFromPropertiesTimerNeverRun(t *testing.T) {
	var tm timer
	tm.NextRun, tm.LastRun = "never", "never"
	fillFromProperties(&tm, map[string]interface{}{
		"Unit": "logrotate.service", "Persistent": true, "Result": "success",
		"NextElapseUSecRealtime": uint64(0), "LastTriggerUSec": uint64(1_700_000_000_000_000),
	})
	if tm.Unit != "logrotate.service" || !tm.Persistent || tm.NextRun != "never" || tm.LastRun != "2023-11-14T22:13:20Z" || tm.Schedule == nil {
		t.Errorf("got %+v", tm)
	}
}
