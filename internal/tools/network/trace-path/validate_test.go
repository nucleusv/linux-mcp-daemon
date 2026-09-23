package tracepath

import "testing"

func TestHostValidation(t *testing.T) {
	for _, h := range []string{"-i", "--help", "-s 1.2.3.4", "a b", "host;id", ""} {
		if validHost.MatchString(h) {
			t.Errorf("accepted %q", h)
		}
	}
	for _, h := range []string{"1.1.1.1", "example.com", "2606:4700::1111", "my-host_1.local"} {
		if !validHost.MatchString(h) {
			t.Errorf("rejected %q", h)
		}
	}
}
