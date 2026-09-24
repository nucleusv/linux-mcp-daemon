package usage

import "testing"

func TestHuman(t *testing.T) {
	for b, want := range map[uint64]string{0: "0B", 512: "512B", 2048: "2.0Ki", 1984507904: "1.8Gi", 560111616: "534Mi"} {
		if got := human(b); got != want {
			t.Errorf("human(%d) = %s, want %s", b, got, want)
		}
	}
}
