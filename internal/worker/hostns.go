package worker

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

// sameNamespace reports whether two namespace files (e.g. two /proc/*/ns/mnt
// entries) refer to the same kernel namespace, by comparing device+inode -
// the standard, cheap way to compare Linux namespace identities without
// actually joining them.
func sameNamespace(a, b string) (bool, error) {
	var stA, stB syscall.Stat_t
	if err := syscall.Stat(a, &stA); err != nil {
		return false, fmt.Errorf("stat %s: %w", a, err)
	}
	if err := syscall.Stat(b, &stB); err != nil {
		return false, fmt.Errorf("stat %s: %w", b, err)
	}
	return stA.Dev == stB.Dev && stA.Ino == stB.Ino, nil
}

// IsContainerized reports whether this process appears to be running in a
// separate mount namespace from its own PID 1 - i.e. whether there's an
// actual container boundary for JoinHostMountNamespace to cross. It's a
// heuristic (see caveat below), used only for a startup sanity check against
// configs/daemon.yaml's worker.containerized setting; JoinHostMountNamespace
// makes its own authoritative check every time it actually runs.
//
// Caveat: this compares against PID 1 of whatever PID namespace this process
// is in. With hostPID: true (this daemon's actual deployment), that's
// genuinely the host's real init, so the check is meaningful. Without
// hostPID, PID 1 would just be this container's own init, which normally
// shares this process's mount namespace regardless of whether the container
// itself is isolated from a real host one level up - so in that
// configuration this heuristic can under-report containerization. It's a
// best-effort warning, not a security boundary.
func IsContainerized() (bool, error) {
	same, err := sameNamespace("/proc/self/ns/mnt", "/proc/1/ns/mnt")
	if err != nil {
		return false, err
	}
	return !same, nil
}

// JoinHostMountNamespace switches the calling OS thread into the host's real
// mount namespace and chroots into it, so subsequent native file operations
// (os.Stat, os.ReadDir, our own /proc and /sys parsing) see the true host
// filesystem instead of this container's own root.
//
// This must only be called from a freshly re-exec'd, single-purpose `mcpd
// worker` process, immediately before running the requested tool's logic -
// never from the master daemon loop. setns(2) operates per-OS-thread, so the
// calling thread is permanently repurposed here; that's fine because the
// worker process exits right after handling its one tool call.
//
// Requires CAP_SYS_ADMIN and CAP_SYS_CHROOT (granted via the pod's
// privileged securityContext) and hostPID: true at the pod level, so
// /proc/1 is the host's real init rather than a namespaced view of it.
func JoinHostMountNamespace() error {
	// If this worker's mount namespace already is PID 1's (bare-metal mcpd,
	// or a misconfigured worker.containerized: true on a non-containerized
	// host), setns/chroot would be redundant work that still demands
	// CAP_SYS_ADMIN/CAP_SYS_CHROOT for no benefit. Skip it - already at the
	// host root is success, not a case to fail on.
	if same, err := sameNamespace("/proc/self/ns/mnt", "/proc/1/ns/mnt"); err == nil && same {
		return nil
	}

	runtime.LockOSThread() // setns/chroot are per-thread; deliberately never unlocked.

	// setns(CLONE_NEWNS) refuses to run (EINVAL) on any thread that still
	// shares filesystem attributes (fs_struct: root/cwd/umask) with other
	// threads - which, by default, is every OS thread the Go runtime spawns
	// (they're created with CLONE_FS, like standard pthreads). unshare it
	// first to give this thread its own private fs_struct.
	if err := syscall.Unshare(syscall.CLONE_FS); err != nil {
		return fmt.Errorf("unshare(CLONE_FS): %w", err)
	}

	fd, err := os.Open("/proc/1/ns/mnt")
	if err != nil {
		return fmt.Errorf("open host mount namespace: %w", err)
	}
	defer fd.Close()

	// syscall.SYS_SETNS is undefined on linux/amd64 in the standard library
	// (defined only for arm, arm64, mips*, ppc64*, s390x, riscv64, loong64) -
	// use x/sys/unix's Setns wrapper, which has the correct syscall number
	// for every Linux architecture, amd64 included.
	if err := unix.Setns(int(fd.Fd()), unix.CLONE_NEWNS); err != nil {
		return fmt.Errorf("setns(CLONE_NEWNS): %w", err)
	}

	// setns only swaps which mount table this thread consults; the process's
	// root/cwd stay pinned to whatever dentry they already pointed at. Chroot
	// into /proc/1/root - which, now that we've joined the namespace PID 1
	// itself lives in, correctly resolves to the host's real "/" - to
	// actually become the host's filesystem view. Same technique `nsenter
	// --root` uses under the hood.
	if err := syscall.Chdir("/proc/1/root"); err != nil {
		return fmt.Errorf("chdir to host root: %w", err)
	}
	if err := syscall.Chroot("."); err != nil {
		return fmt.Errorf("chroot to host root: %w", err)
	}
	return syscall.Chdir("/")
}
