package docker

import (
	"fmt"
	"path"
	"strings"
	"time"
)

// This file is the whole boundary for docker/* tools, and it is worth
// knowing why it carries more weight than the equivalent check for files/*.
//
// For every other privileged tool, mcp-sudo.yaml only chooses the worker's
// uid and the kernel independently enforces access: a bug in the `paths:`
// matcher still cannot read /etc/shadow as uid 1001. The docker socket has
// no such second gate - it is all-or-nothing, the Engine API has no
// per-container authorization at all, and docker/* workers run as root. So
// `containers:` is an application-level filter with nothing behind it.
//
// Hence the two rules below, which are not defensive style but the
// difference between working and not:
//
//  1. Validate the identifier and REJECT it; never sanitize. The identifier
//     goes into a URL path (POST /containers/{id}/stop), so with
//     containers: ["web-*"] the string "web-1/../../db-1" *matches the
//     glob* and addresses a different container. Cleaning it would silently
//     change which container is acted on; refusing it cannot.
//  2. Resolve, then check - never check, then resolve. Docker accepts a
//     name, a full ID or an ID prefix for the same container, so filtering
//     the caller's raw string lets a name glob be dodged with an ID prefix,
//     and an ID grant be dodged with a name. Resolve to the canonical
//     container first, then match its real name AND its real ID.

// maxIdentifier is generous next to Docker's 64-hex IDs and its name limit;
// it exists so a megabyte of junk is refused before anything parses it.
const maxIdentifier = 255

// ValidIdentifier reports whether s is a syntactically valid Docker
// container or volume identifier: Docker's own charset,
// [a-zA-Z0-9][a-zA-Z0-9_.-]*. Anything else - a slash, a percent escape, a
// semicolon, a space - is refused outright rather than cleaned.
func ValidIdentifier(s string) error {
	if s == "" {
		return fmt.Errorf("container name or ID is required")
	}
	if len(s) > maxIdentifier {
		return fmt.Errorf("container identifier is too long (%d characters, maximum %d)", len(s), maxIdentifier)
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			continue
		case i > 0 && (c == '_' || c == '.' || c == '-'):
			continue
		}
		return fmt.Errorf("invalid container identifier %q: Docker names and IDs match [a-zA-Z0-9][a-zA-Z0-9_.-]+, and %q is refused rather than cleaned up", s, string(c))
	}
	return nil
}

// ValidImageRef checks an image reference. Image refs are not container
// names: they legitimately contain "/" (ghcr.io/org/img), ":" (tag), and
// "@" (digest), so the container charset would refuse valid input. What
// must still be impossible is escaping the endpoint's path, so ".."
// segments and a leading "/" are refused.
func ValidImageRef(s string) error {
	if s == "" {
		return fmt.Errorf("image name or ID is required")
	}
	if len(s) > maxIdentifier*2 {
		return fmt.Errorf("image reference is too long (%d characters)", len(s))
	}
	if strings.HasPrefix(s, "/") {
		return fmt.Errorf("invalid image reference %q: must not start with /", s)
	}
	for _, seg := range strings.Split(s, "/") {
		if seg == "." || seg == ".." || seg == "" {
			return fmt.Errorf("invalid image reference %q: path segments like %q are refused", s, seg)
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '_' || c == '.' || c == '-' || c == '/' || c == ':' || c == '@' || c == '+':
		default:
			return fmt.Errorf("invalid image reference %q: character %q is not allowed", s, string(c))
		}
	}
	return nil
}

// Allowed reports whether a container whose canonical name and ID are given
// is covered by the allowlist. Both are matched, so neither a name glob nor
// an ID grant can be sidestepped by naming the container the other way. An
// empty allowlist allows nothing: a grant without `containers:` is refused,
// never treated as "everywhere" (`containers: ["*"]` says that explicitly).
func Allowed(name, id string, allow []string) bool {
	if len(allow) == 0 {
		return false
	}
	name = strings.TrimPrefix(name, "/")
	short := id
	if len(short) > 12 {
		short = short[:12]
	}
	for _, pattern := range allow {
		for _, candidate := range []string{name, id, short} {
			if candidate == "" {
				continue
			}
			if ok, err := path.Match(pattern, candidate); err == nil && ok {
				return true
			}
		}
	}
	return false
}

// Container is the part of an inspect answer needed to identify one.
type Container struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

// Resolve asks Docker which container ref means, returning its canonical
// name (without the leading slash) and full ID. ref must already have
// passed ValidIdentifier.
func Resolve(c *Client, ref string) (name, id string, err error) {
	var ct Container
	if err := c.GetJSON("/containers/"+ref+"/json", &ct); err != nil {
		return "", "", err
	}
	if ct.ID == "" {
		return "", "", fmt.Errorf("container %q could not be resolved", ref)
	}
	return strings.TrimPrefix(ct.Name, "/"), ct.ID, nil
}

// Authorize is the single entry point every container-scoped tool uses. It
// validates the identifier, refuses an empty allowlist before touching the
// socket, resolves ref to the canonical container, matches both its name and
// ID against the allowlist, and returns the full ID.
//
// Callers must act on the returned ID, not on ref: the ID is what was
// authorized, and it cannot be re-pointed at another container by a
// concurrent `docker rename`.
func Authorize(c *Client, ref string, allow []string) (id, name string, err error) {
	if err := ValidIdentifier(ref); err != nil {
		return "", "", err
	}
	if len(allow) == 0 {
		return "", "", fmt.Errorf("not authorized: no containers: list in this tool's grant in mcp-sudo.yaml - list the containers it may touch, or containers: [\"*\"] for all of them")
	}
	name, id, err = Resolve(c, ref)
	if err != nil {
		return "", "", err
	}
	if !Allowed(name, id, allow) {
		return "", "", fmt.Errorf("not authorized to act on container %s (%s): it is not in this tool's containers: list in mcp-sudo.yaml", name, id[:min(12, len(id))])
	}
	return id, name, nil
}

// PruneTargets are the kinds of unused object docker/prune can reclaim, in
// the order it reclaims them: containers first (that is what makes their
// images dangling and their volumes unused), build cache last. A grant's
// prune: list names a subset of these; there is no "all".
var PruneTargets = []string{"containers", "images", "volumes", "networks", "build-cache"}

// PruneAllowed reports whether target is in this grant's prune: list. Exact
// matches only - the targets are five fixed words, so a glob would buy
// nothing and "*" would be the implicit everything the ticket rules out.
func PruneAllowed(target string, allow []string) bool {
	for _, a := range allow {
		if a == target {
			return true
		}
	}
	return false
}

// CommonArgs are the fields the daemon injects into every docker/* worker
// call. Neither is ever accepted from the caller: the master deletes them
// from the incoming arguments and sets them itself, the same way
// _network_policy is handled for network/curl.
type CommonArgs struct {
	// Socket is the path from daemon.yaml (worker.docker_socket).
	Socket string `json:"_docker_socket,omitempty"`
	// Containers is this user's containers: allowlist for this specific
	// tool. Empty means refuse everything.
	Containers []string `json:"_containers,omitempty"`
}

// Client builds the Engine API client for this call.
func (a CommonArgs) Client(timeout time.Duration) *Client {
	return New(a.Socket, timeout)
}

// Structured reports whether output_format asks for machine-readable output,
// the same set every other tool in this daemon accepts.
func Structured(format string) bool {
	switch format {
	case "json", "yaml", "table", "wide":
		return true
	}
	return false
}
