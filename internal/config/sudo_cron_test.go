package config

import (
	"strings"
	"testing"
)

const cronGrant = `users:
  alice:
    privileged:
      tools:
        cron/manage:
          allowed: true
          users:
            test_user: {edit: true}
            www-data:  {view: true}
            root:      {view: true}
`

func TestCronRules(t *testing.T) {
	c, err := ParseSudoConfig([]byte(cronGrant), true)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		target     string
		view, edit bool
	}{
		{"test_user", true, true}, // edit implies view
		{"www-data", true, false},
		{"root", true, false},
		{"postgres", false, false}, // not named: refused
	}
	for _, x := range cases {
		r := c.CronRuleFor("alice", x.target)
		if r.View != x.view || r.Edit != x.edit {
			t.Errorf("%s: got %+v, want view=%v edit=%v", x.target, r, x.view, x.edit)
		}
	}
	if r := c.CronRuleFor("bob", "test_user"); r.View || r.Edit {
		t.Errorf("a user without a grant must have no rights, got %+v", r)
	}
	if got := strings.Join(c.CronViewAccounts("alice"), ","); got != "root,test_user,www-data" {
		t.Errorf("view accounts: %q", got)
	}
}

func TestCronConfigValidation(t *testing.T) {
	bad := map[string]string{
		"root with edit":  "test_user: {view: true}\n            root: {edit: true}",
		"a glob key":      `"*": {view: true}`,
		"a pattern key":   `"ops-*": {view: true}`,
		"an option-like":  `"-l": {view: true}`,
		"neither flag":    "test_user: {}",
		"explicit falses": "test_user: {view: false, edit: false}",
	}
	for name, users := range bad {
		yaml := "users:\n  alice:\n    privileged:\n      tools:\n        cron/manage:\n          allowed: true\n          users:\n            " + users + "\n"
		if _, err := ParseSudoConfig([]byte(yaml), true); err == nil {
			t.Errorf("%s: want a load error", name)
		}
	}
	// allowed without users: strict rejects, non-strict loads and refuses everything at runtime
	noUsers := "users:\n  alice:\n    privileged:\n      tools:\n        cron/manage: {allowed: true}\n"
	if _, err := ParseSudoConfig([]byte(noUsers), true); err == nil {
		t.Error("strict: allowed without users must be rejected")
	}
	c, err := ParseSudoConfig([]byte(noUsers), false)
	if err != nil {
		t.Fatal(err)
	}
	if r := c.CronRuleFor("alice", "test_user"); r.View || r.Edit {
		t.Errorf("non-strict, no users: must refuse, got %+v", r)
	}
	// users on another tool
	other := "users:\n  alice:\n    privileged:\n      tools:\n        files/list: {allowed: true, paths: [/], users: {a: {view: true}}}\n"
	if _, err := ParseSudoConfig([]byte(other), true); err == nil {
		t.Error("users on a non-cron tool must be rejected")
	}
}
