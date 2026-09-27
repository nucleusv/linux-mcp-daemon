package docker

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeDocker serves a Docker Engine API on a throwaway unix socket. Every
// test that must prove something is refused *before* the socket is touched
// asserts on hits == 0.
func fakeDocker(t *testing.T, containers map[string]Container) (*Client, *int) {
	t.Helper()
	// Not t.TempDir(): its path is long enough to blow the ~104-byte limit
	// on a unix socket address, which fails as "bind: invalid argument".
	dir, err := os.MkdirTemp("", "mcpd")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "d.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	hits := 0
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		// Only /containers/{ref}/json is needed here.
		ref := r.URL.Path
		if len(ref) > len("/containers/") {
			ref = ref[len("/containers/"):]
		}
		if i := len(ref) - len("/json"); i > 0 && ref[i:] == "/json" {
			ref = ref[:i]
		}
		ct, ok := containers[ref]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "No such container: " + ref})
			return
		}
		_ = json.NewEncoder(w).Encode(ct)
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return New(socket, 5*time.Second), &hits
}

// T5e - a malicious identifier is refused, never cleaned up. Each of these
// would match the glob `web-*` while addressing something else.
func TestValidIdentifierRejects(t *testing.T) {
	bad := []string{
		"",
		"web-1/../../db-1",    // path traversal, matches web-*
		"web-1%2F..%2Fdb-1",   // percent-encoded traversal
		"web-1;rm -rf /",      // shell metacharacters
		"web-1 db-1",          // space
		"web-1/json?all=true", // query injection
		"-web-1",              // Docker names never start with -
		".web-1",              // nor with .
		"web-1\n",             // newline
		"web/1",               // bare slash
	}
	for _, s := range bad {
		if err := ValidIdentifier(s); err == nil {
			t.Errorf("ValidIdentifier(%q) = nil, want refusal", s)
		}
	}

	good := []string{
		"web-1",
		"web_1.a-B9",
		"3f2c1b0a9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8b7a6f5e4d3c2b", // full ID
		"3f2c1b0a9e8d",
	}
	for _, s := range good {
		if err := ValidIdentifier(s); err != nil {
			t.Errorf("ValidIdentifier(%q) = %v, want nil", s, err)
		}
	}
}

func TestValidImageRef(t *testing.T) {
	good := []string{"nginx", "nginx:1.25", "ghcr.io/org/img:tag", "img@sha256:abc123", "sha256:abc123"}
	for _, s := range good {
		if err := ValidImageRef(s); err != nil {
			t.Errorf("ValidImageRef(%q) = %v, want nil", s, err)
		}
	}
	bad := []string{"", "/nginx", "../../etc/passwd", "ghcr.io/../img", "nginx;x", "nginx image"}
	for _, s := range bad {
		if err := ValidImageRef(s); err == nil {
			t.Errorf("ValidImageRef(%q) = nil, want refusal", s)
		}
	}
}

// T5 - the allowlist matches the canonical name, the full ID and the short
// ID, and an empty list allows nothing.
func TestAllowed(t *testing.T) {
	const id = "3f2c1b0a9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8b7a6f5e4d3c2b"
	cases := []struct {
		name, id string
		allow    []string
		want     bool
	}{
		{"web-1", id, nil, false},                     // no list = nothing
		{"web-1", id, []string{}, false},              // explicit empty = nothing
		{"web-1", id, []string{"*"}, true},            // the explicit everywhere
		{"web-1", id, []string{"web-*"}, true},        // glob on name
		{"db-1", id, []string{"web-*"}, false},        // glob misses
		{"web-1", id, []string{"web-1"}, true},        // exact name
		{"web-1", id, []string{id}, true},             // full ID
		{"web-1", id, []string{"3f2c1b0a9e8d"}, true}, // short ID
		{"/web-1", id, []string{"web-1"}, true},       // Docker's leading slash
		{"web-1", id, []string{"db-*", "web-1"}, true},
	}
	for _, c := range cases {
		if got := Allowed(c.name, c.id, c.allow); got != c.want {
			t.Errorf("Allowed(%q, id, %v) = %v, want %v", c.name, c.allow, got, c.want)
		}
	}
}

// T5f - authorization matches the *canonical* container, so an ID cannot be
// used to dodge a name glob and a name cannot be used to dodge an ID grant.
func TestAuthorizeResolvesBeforeMatching(t *testing.T) {
	const webID = "3f2c1b0a9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8b7a6f5e4d3c2b"
	const dbID = "aaaa1b0a9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8b7a6f5e4d3c2b"
	web := Container{ID: webID, Name: "/web-1"}
	db := Container{ID: dbID, Name: "/db-1"}
	containers := map[string]Container{
		"web-1": web, webID: web, "3f2c1b0a9e8d": web,
		"db-1": db, dbID: db, "aaaa1b0a9e8d": db,
	}

	// A name grant covers the same container addressed by its ID.
	c, hits := fakeDocker(t, containers)
	id, name, err := Authorize(c, "3f2c1b0a9e8d", []string{"web-*"})
	if err != nil || id != webID || name != "web-1" {
		t.Errorf("short ID under a name glob: got (%q, %q, %v), want the web-1 container", id, name, err)
	}
	if *hits == 0 {
		t.Error("expected the socket to be consulted")
	}

	// An ID grant covers the same container addressed by its name.
	c, _ = fakeDocker(t, containers)
	if id, _, err := Authorize(c, "web-1", []string{webID}); err != nil || id != webID {
		t.Errorf("name under an ID grant: got (%q, %v), want %q", id, err, webID)
	}

	// A container outside the glob is refused even by its real name.
	c, _ = fakeDocker(t, containers)
	if _, _, err := Authorize(c, "db-1", []string{"web-*"}); err == nil {
		t.Error("db-1 under containers: [web-*] should be refused")
	}

	// ...and by its ID, which the glob would never have caught anyway.
	c, _ = fakeDocker(t, containers)
	if _, _, err := Authorize(c, "aaaa1b0a9e8d", []string{"web-*"}); err == nil {
		t.Error("db-1's ID under containers: [web-*] should be refused")
	}
}

// An empty allowlist and an invalid identifier are both refused before the
// socket is dialled at all.
func TestAuthorizeRefusesBeforeDialing(t *testing.T) {
	c, hits := fakeDocker(t, map[string]Container{})

	if _, _, err := Authorize(c, "web-1", nil); err == nil {
		t.Error("empty allowlist should be refused")
	}
	if _, _, err := Authorize(c, "web-1/../../db-1", []string{"*"}); err == nil {
		t.Error("traversal identifier should be refused")
	}
	if *hits != 0 {
		t.Errorf("socket was contacted %d times; both refusals must happen before any request", *hits)
	}
}

// A missing container surfaces Docker's own message, not a generic failure.
func TestAuthorizeUnknownContainer(t *testing.T) {
	c, _ := fakeDocker(t, map[string]Container{})
	_, _, err := Authorize(c, "nope", []string{"*"})
	if err == nil {
		t.Fatal("unknown container should error")
	}
	if want := "No such container: nope"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should carry Docker's message %q", err, want)
	}
}
