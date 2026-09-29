package rpc

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
)

func cronGrant(t *testing.T) *config.SudoConfig {
	t.Helper()
	c, err := config.ParseSudoConfig([]byte(`users:
  alice:
    privileged:
      tools:
        cron/manage:
          allowed: true
          users:
            test_user: {view: true, edit: true}
            www-data:  {view: true}
            root:      {view: true}
            toor:      {view: true, edit: true}
  bob: {}
`), true)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func plan(t *testing.T, caller, args string, info bool) (cronPlan, map[string]interface{}, error) {
	t.Helper()
	p, err := prepareCronCall(cronGrant(t), caller, json.RawMessage(args), info)
	var m map[string]interface{}
	if err == nil {
		json.Unmarshal(p.Args, &m)
	}
	return p, m, err
}

func TestCronOwnCrontabNeedsNoGrantAndNoRoot(t *testing.T) {
	for _, args := range []string{`{}`, `{"user":"bob"}`, `{"user":"bob","privileged":true,"content":"x\n"}`, `{"content":""}`} {
		p, m, err := plan(t, "bob", args, false) // bob has no cron grant at all
		if err != nil || p.Privileged || m["_mode"] != "self" || m["_target_user"] != "bob" {
			t.Errorf("%s: plan=%+v args=%v err=%v", args, p, m, err)
		}
	}
	// a caller can never set the reserved keys
	_, m, err := plan(t, "bob", `{"_mode":"other","_target_uid":0,"_target_user":"root","_view_users":["root"],"_info":true}`, false)
	if err != nil || m["_mode"] != "self" || m["_target_user"] != "bob" || m["_view_users"] != nil || m["_target_uid"] != nil || m["_info"] != false {
		t.Errorf("reserved keys not replaced: %v %v", m, err)
	}
}

func TestCronAnotherAccount(t *testing.T) {
	// no privileged flag
	if _, _, err := plan(t, "alice", `{"user":"test_user"}`, false); err == nil || !strings.Contains(err.Error(), "privileged: true") {
		t.Errorf("without privileged: %v", err)
	}
	// no grant at all
	if _, _, err := plan(t, "bob", `{"user":"test_user","privileged":true}`, false); err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Errorf("no grant: %v", err)
	}
	// view and edit through a rule: root worker, target credentials injected
	p, m, err := plan(t, "alice", `{"user":"test_user","privileged":true,"content":"0 3 * * * x"}`, false)
	if err != nil || !p.Privileged || !p.Write || m["_mode"] != "other" || m["_target_user"] != "test_user" || m["_target_uid"] != nil {
		t.Errorf("edit allowed: %+v %v %v", p, m, err)
	}
	// view-only account: read fine, write refused naming view only
	if _, _, err := plan(t, "alice", `{"user":"www-data","privileged":true}`, false); err != nil {
		t.Errorf("view of www-data: %v", err)
	}
	if _, _, err := plan(t, "alice", `{"user":"www-data","privileged":true,"content":"x"}`, false); err == nil || !strings.Contains(err.Error(), "view only") {
		t.Errorf("edit of a view-only account: %v", err)
	}
	// an account with no rule
	if _, _, err := plan(t, "alice", `{"user":"nobody","privileged":true}`, false); err == nil || !strings.Contains(err.Error(), "no rule") {
		t.Errorf("unnamed account: %v", err)
	}
	// no numeric ids ever travel from the master: the worker resolves the name where it runs
	if _, m, _ := plan(t, "alice", `{"user":"test_user","privileged":true,"_target_uid":0}`, false); m["_target_uid"] != nil || m["_target_gid"] != nil || m["_target_groups"] != nil {
		t.Errorf("uid/gid/groups must not be set by the master: %v", m)
	}
}

func TestCronRootIsViewOnly(t *testing.T) {
	// root may be viewed when named ...
	if _, m, err := plan(t, "alice", `{"user":"root","privileged":true}`, false); err != nil || m["_target_user"] != "root" {
		t.Errorf("root view: %v %v", m, err)
	}
	// ... but the rule can only say view (an edit rule for root is a load error, see config tests),
	// so a write to root is refused in the master; a uid-0 alias under another name is refused by the worker.
	if _, _, err := plan(t, "alice", `{"user":"root","privileged":true,"content":"x"}`, false); err == nil || !strings.Contains(err.Error(), "view only") {
		t.Errorf("edit of root must be refused, got %v", err)
	}
}

func TestCronList(t *testing.T) {
	p, m, err := plan(t, "alice", `{"privileged":true}`, false)
	if err != nil || !p.Privileged || m["_mode"] != "list" {
		t.Fatalf("list: %+v %v %v", p, m, err)
	}
	got := fmt.Sprint(m["_view_users"])
	for _, want := range []string{"alice", "test_user", "www-data", "root"} {
		if !strings.Contains(got, want) {
			t.Errorf("view users %s lack %s", got, want)
		}
	}
	if _, _, err := plan(t, "bob", `{"privileged":true}`, false); err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Errorf("list without a grant: %v", err)
	}
}

func TestCronBadInput(t *testing.T) {
	for _, args := range []string{`{"user":"-l","privileged":true}`, `{"user":"a b","privileged":true}`, `{"user":"*","privileged":true}`, `{"user":5}`, `{"content":5}`, `not json`} {
		if _, _, err := plan(t, "alice", args, false); err == nil {
			t.Errorf("%s must be refused", args)
		}
	}
}

func TestCronAuditNeverHoldsTheContent(t *testing.T) {
	a := cronAuditAttrs(json.RawMessage(`{"content":"0 3 * * * curl -u me:secret x\n"}`), "test_user")
	joined := fmt.Sprint(a...)
	if strings.Contains(joined, "secret") || !strings.Contains(joined, "content_sha256") || !strings.Contains(joined, "test_user") {
		t.Errorf("audit attrs: %v", a)
	}
	if cronAuditAttrs(json.RawMessage(`{}`), "x") != nil {
		t.Error("a read must not produce audit attrs")
	}
}
