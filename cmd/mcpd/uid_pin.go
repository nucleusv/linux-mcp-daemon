package main

import (
	"fmt"
	"log"
	"os/user"
	"sync"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/yamledit"
)

// uidPinMu serializes writes to daemon.yaml from checkAndPinUID. Multiple
// requests (from the same or different users) can hit this concurrently;
// without a lock, two concurrent first-use pins could race and corrupt the
// file via interleaved read-modify-write cycles.
var uidPinMu sync.Mutex

// checkAndPinUID implements trust-on-first-use OS identity pinning for a
// username that has already passed the bearer-token check in
// authenticateRequest. It exists to close a gap that token verification
// alone cannot: a bearer token authenticates "someone who knows this
// secret," but SpawnWorker (internal/worker/spawner.go) separately does
// user.Lookup(username) and runs the actual worker process as whatever UID
// that resolves to *right now* - and daemon.yaml/mcp-sudo.yaml key
// everything off the username string, with zero awareness of the
// underlying OS account's identity. If the OS account behind a username is
// ever deleted and recreated (a role change, an offboarding/onboarding
// pair reusing a company username convention, a system rebuild), the token
// and mcp-sudo.yaml grants keep working completely unchanged, silently
// applied to whatever now answers to that name.
//
// The check: the first time a username successfully authenticates, pin
// whatever UID it currently resolves to (PinnedUID). On every later call,
// compare freshly - if user.Lookup() now returns a *different* UID for the
// same username, refuse the request rather than silently proceeding.
//
// SECURITY NOTE / KNOWN LIMITATION - read this before relying on it:
// this is trust-on-first-use, not a guarantee. It only catches a UID
// *changing*. On Debian/Ubuntu (this project's own base image),
// `useradd` without an explicit -u does not simply increment from the
// highest existing UID - it fills the lowest unused UID in range (see
// shadow-utils' find_new_uid()). That means the most likely real-world
// version of the attack this defends against - delete an employee's OS
// account, promptly create a new one under the *same* username for their
// replacement - is exactly the case most likely to reissue the *same* UID
// (the slot that was just freed is now the lowest gap), which this check
// cannot detect: resolved UID == pinned UID, nothing looks wrong. It DOES
// catch UID drift from other causes (other accounts created in between
// shifting which gap is lowest, an explicit -u during a migration, moving
// to a different host/image entirely). Treat this as one extra layer, not
// a substitute for the actual reliable control: retire a username for good
// when its person leaves, and run `linuxctl delete mcpd user <name>` as
// part of offboarding - see ARCHITECTURE.md and
// docs/website/docs/configuration/daemon.md.
//
// SECOND KNOWN LIMITATION, specific to this daemon's actual deployment
// (see k8s/deployment.yaml): configs/daemon.yaml is baked into the
// container image (Dockerfile's COPY --from=builder .../configs), not
// mounted from anything durable - there is no volume for it. Every write
// persistUIDPin makes lands on that specific running container's own
// ephemeral filesystem. That means a pin survives for the life of the
// current pod (so it *does* catch someone using `kubectl exec` to swap an
// OS account without redeploying, which is arguably the more realistic
// live-tampering scenario), but a redeploy or crash resets every user back
// to an unpinned state, and the next authentication after that silently
// re-pins whatever UID is current at that moment as the new trusted
// baseline. This is not a bug to "fix" casually - making the pin durable
// across redeploys would mean the daemon writing back into the same
// configs/ directory that scripts/deploy.sh's next `docker build` reads
// from, i.e. the running container mutating the git-tracked source of
// truth for its own next build, which is a much bigger design commitment
// than this feature currently makes.
func checkAndPinUID(username string) bool {
	osUser, err := user.Lookup(username)
	if err != nil {
		// Can't resolve at all - not this function's problem to report.
		// SpawnWorker will hit the exact same user.Lookup() and fail with
		// its own clear "unknown user" error for every call regardless of
		// what we do here, so there's nothing to gain by duplicating or
		// pre-empting that failure path.
		return true
	}
	currentUID := osUser.Uid

	uidPinMu.Lock()
	defer uidPinMu.Unlock()

	idx := -1
	for i := range daemonConfig.Users {
		if daemonConfig.Users[i].Username == username {
			idx = i
			break
		}
	}
	if idx == -1 {
		return true // shouldn't happen - authenticateRequest already matched this username
	}

	pinned := daemonConfig.Users[idx].PinnedUID

	if pinned == "" {
		// First successful use - trust-on-first-use.
		daemonConfig.Users[idx].PinnedUID = currentUID
		daemonConfig.Users[idx].OSUID = currentUID
		if err := persistUIDPin(username, currentUID, currentUID); err != nil {
			log.Printf("[SECURITY] failed to persist UID pin for %q (uid %s): %v - continuing without a persisted pin this time", username, currentUID, err)
		} else {
			log.Printf("[SECURITY] pinned OS UID %s for mcpd user %q (first successful authentication)", currentUID, username)
		}
		return true
	}

	if pinned == currentUID {
		return true // matches - nothing to persist, nothing to warn about
	}

	// Mismatch: the username now resolves to a different OS account than
	// the one this token was first used with. Refuse, and persist the
	// observed value so it's visible in `linuxctl list mcpd users` even
	// after this log line scrolls away.
	daemonConfig.Users[idx].OSUID = currentUID
	if err := persistUIDPin(username, pinned, currentUID); err != nil {
		log.Printf("[SECURITY] failed to persist UID mismatch observation for %q: %v", username, err)
	}
	log.Printf("[SECURITY] REFUSING auth for %q: pinned UID %s does not match currently-resolved UID %s - the OS account behind this username may have been deleted and recreated. Run `linuxctl update mcpd user %s` to re-pin if this is an intentional change.", username, pinned, currentUID, username)
	return false
}

// persistUIDPin writes pinned_uid/os_uid for username into daemon.yaml,
// preserving every other field and comment in the file (see
// internal/yamledit). Caller must hold uidPinMu.
func persistUIDPin(username, pinnedUID, osUID string) error {
	doc, err := yamledit.LoadDoc(daemonConfigPath)
	if err != nil {
		return fmt.Errorf("loading %s: %w", daemonConfigPath, err)
	}
	if len(doc.Content) == 0 {
		return fmt.Errorf("%s is empty", daemonConfigPath)
	}
	root := doc.Content[0]
	seq := yamledit.MapGet(root, "users")
	if seq == nil {
		return fmt.Errorf("%s has no users: list", daemonConfigPath)
	}
	for _, item := range seq.Content {
		if u := yamledit.MapGet(item, "username"); u != nil && u.Value == username {
			yamledit.MapSet(item, "pinned_uid", yamledit.ScalarNode(pinnedUID))
			yamledit.MapSet(item, "os_uid", yamledit.ScalarNode(osUID))
			return yamledit.SaveDoc(daemonConfigPath, doc)
		}
	}
	return fmt.Errorf("user %q not found in %s", username, daemonConfigPath)
}
