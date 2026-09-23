package system_control

import "testing"

func TestKeyInjectionRejected(t *testing.T) {
	for _, key := range []string{"-p/etc/shadow", "-p", "--load=/etc/shadow", "-a", "net.ipv4 -w x=1", ""} {
		if key != "" && validKey.MatchString(key) {
			t.Errorf("validKey accepted %q", key)
		}
	}
	for _, key := range []string{"net.ipv4.ip_forward", "kernel/hostname", "vm.swappiness", "net.ipv4.conf.all.rp_filter"} {
		if !validKey.MatchString(key) {
			t.Errorf("validKey rejected legit %q", key)
		}
	}
	if _, err := SystemControl([]byte(`{"key":"-p/etc/shadow"}`)); err == nil {
		t.Error("SystemControl accepted an option as key")
	}
}
