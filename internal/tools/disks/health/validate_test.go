package health

import "testing"

func TestDeviceValidation(t *testing.T) {
	for _, d := range []string{"../etc/shadow", "sda/../../etc", "-a", "sda -d sat", "/dev/sda"} {
		if validDevice.MatchString(d) {
			t.Errorf("accepted %q", d)
		}
	}
	for _, d := range []string{"sda", "nvme0n1", "sg1", "vda"} {
		if !validDevice.MatchString(d) {
			t.Errorf("rejected %q", d)
		}
	}
}
