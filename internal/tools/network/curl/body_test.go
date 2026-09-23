package curl

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyIsSentAndHTMLNotEscaped(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.Write([]byte("<html>ok</html>"))
	}))
	defer srv.Close()

	args, _ := json.Marshal(map[string]interface{}{"url": srv.URL, "method": "POST", "body": `{"a":1}`})
	out, err := Curl(args)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"a":1}` {
		t.Errorf("server received body %q - the schema's \"body\" param was dropped", got)
	}
	if !strings.Contains(out, "<html>ok</html>") {
		t.Errorf("response body HTML-escaped: %s", out)
	}
}

func TestNetworkPolicyEnforced(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("internal"))
	}))
	defer srv.Close()

	// No policy: unchanged behavior, loopback reachable.
	args, _ := json.Marshal(map[string]interface{}{"url": srv.URL})
	if _, err := Curl(args); err != nil {
		t.Fatalf("unrestricted curl failed: %v", err)
	}

	// deny_private: the same loopback server is refused.
	args, _ = json.Marshal(map[string]interface{}{"url": srv.URL, "_network_policy": map[string]interface{}{"deny_private": true}})
	if _, err := Curl(args); err == nil || !strings.Contains(err.Error(), "deny_private") {
		t.Errorf("deny_private did not block loopback: %v", err)
	}

	// Redirect hops go through the same policy dialer.
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL, http.StatusFound)
	}))
	defer redirector.Close()
	redirHost := strings.TrimPrefix(redirector.URL, "http://")
	redirIP := strings.Split(redirHost, ":")[0]
	// Both test servers listen on 127.0.0.1, so the redirect chain is
	// allowed when that IP is allowlisted, and every hop is refused once
	// it's on the deny list.
	policy := map[string]interface{}{"deny_private": true, "allow": []string{redirIP + "/32"}}
	args, _ = json.Marshal(map[string]interface{}{"url": redirector.URL, "_network_policy": policy})
	if _, err := Curl(args); err != nil {
		t.Errorf("allowlisted redirect chain failed: %v", err)
	}
	// ... and with a deny rule on that IP, every hop is refused.
	policy["deny"] = []string{redirIP}
	args, _ = json.Marshal(map[string]interface{}{"url": redirector.URL, "_network_policy": policy})
	if _, err := Curl(args); err == nil {
		t.Error("deny rule not enforced")
	}
}
