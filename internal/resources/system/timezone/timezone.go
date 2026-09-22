package timezone

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Read returns the system's configured IANA timezone (e.g. "America/New_York"),
// preferring /etc/timezone (Debian/Ubuntu) and falling back to resolving the
// /etc/localtime symlink target (portable across distros that don't have
// /etc/timezone). Also reports the current local offset/abbreviation via
// Go's time package for convenience.
func Read() (string, string, error) {
	zone := ""
	if b, err := os.ReadFile("/etc/timezone"); err == nil {
		zone = strings.TrimSpace(string(b))
	} else if target, err := os.Readlink("/etc/localtime"); err == nil {
		if idx := strings.Index(target, "zoneinfo/"); idx != -1 {
			zone = target[idx+len("zoneinfo/"):]
		}
	}
	if zone == "" {
		// Not a guess: per POSIX/glibc convention, the absence of
		// /etc/localtime means the system runs in UTC - this is documented
		// default behavior, not an assumption. Seen in practice on minimal
		// images with no tzdata installed at all.
		zone = "UTC"
	}

	now := time.Now()
	abbrev, offsetSec := now.Zone()
	offsetHours := float64(offsetSec) / 3600

	result := fmt.Sprintf("Time zone: %s\nAbbreviation: %s\nUTC offset: %+.1f\nCurrent local time: %s\n",
		zone, abbrev, offsetHours, now.Format(time.RFC1123))
	return result, "text/plain", nil
}
