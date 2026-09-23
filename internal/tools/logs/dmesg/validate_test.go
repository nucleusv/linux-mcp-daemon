package dmesg

import "testing"

func TestLevelValidation(t *testing.T) {
	for _, l := range []string{"err,warn", "debug", "emerg,alert,crit"} {
		if !validLevels.MatchString(l) {
			t.Errorf("rejected %q", l)
		}
	}
	for _, l := range []string{"-F/etc/shadow", "err;id", "err,", "ERR"} {
		if validLevels.MatchString(l) {
			t.Errorf("accepted %q", l)
		}
	}
}
