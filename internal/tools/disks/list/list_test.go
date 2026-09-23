package list

import (
	"sort"
	"strings"
	"testing"
)

func TestHumanSize(t *testing.T) {
	cases := map[uint64]string{
		0:           "0B",
		512:         "512B",
		2048:        "2K",
		2146435072:  "2G",   // 1.999G rounds up, like lsblk
		3955228672:  "3.7G", // vg4114-swap
		58317602816: "54.3G",
		64424509440: "60G",
		1073741824:  "1G",
	}
	for in, want := range cases {
		if got := humanSize(in); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestLessKName(t *testing.T) {
	names := []string{"sda10", "dm-10", "sda2", "sr0", "dm-2", "sda", "nvme0n1"}
	sort.Slice(names, func(i, j int) bool { return lessKName(names[i], names[j]) })
	want := "dm-2 dm-10 nvme0n1 sda sda2 sda10 sr0"
	if got := strings.Join(names, " "); got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
}

// TestFormatTree checks the text layout against real `lsblk` output from an
// Ubuntu 24.04 LVM host.
func TestFormatTree(t *testing.T) {
	swap := &BlockDevice{Name: "vg4114-swap", MajMin: "252:0", Size: "3.7G", Type: "lvm", MountPoints: []string{"[SWAP]"}}
	root := &BlockDevice{Name: "vg4114-root", MajMin: "252:1", Size: "54.3G", Type: "lvm", MountPoints: []string{"/"}}
	roots := []*BlockDevice{
		{Name: "sda", MajMin: "8:0", Size: "60G", Type: "disk", Children: []*BlockDevice{
			{Name: "sda1", MajMin: "8:1", Size: "2G", Type: "part", MountPoints: []string{"/boot"}},
			{Name: "sda2", MajMin: "8:2", Size: "58G", Type: "part", Children: []*BlockDevice{swap, root}},
		}},
		{Name: "sr0", MajMin: "11:0", RM: true, Size: "2K", RO: true, Type: "rom"},
	}
	want := `NAME            MAJ:MIN RM  SIZE RO TYPE MOUNTPOINTS
sda               8:0    0   60G  0 disk
├─sda1            8:1    0    2G  0 part /boot
└─sda2            8:2    0   58G  0 part
  ├─vg4114-swap 252:0    0  3.7G  0 lvm  [SWAP]
  └─vg4114-root 252:1    0 54.3G  0 lvm  /
sr0              11:0    1    2K  1 rom
`
	if got := formatTree(roots); got != want {
		t.Errorf("formatTree mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}
