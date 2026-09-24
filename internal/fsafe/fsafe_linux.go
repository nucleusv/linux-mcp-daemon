//go:build linux

// Package fsafe opens paths for metadata changes (chmod, chown) without
// ever following a symbolic link, in any component.
//
// A plain check-then-act ("lstat each component, then chmod the path") is
// racy: a directory can be swapped for a symlink between the check and the
// act. Instead the path is walked one component at a time with
// openat(O_PATH|O_NOFOLLOW), each step relative to the directory fd the
// previous step opened, so what gets changed is exactly the object that
// was checked. Any symlink along the way - including the target itself -
// is refused.
package fsafe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// ErrSymlink is returned when a path component is a symbolic link.
var ErrSymlink = errors.New("refusing to follow a symbolic link")

// Node is an open, verified filesystem object: an O_PATH descriptor plus
// the stat taken through that same descriptor.
type Node struct {
	FD   int
	Stat unix.Stat_t
	Path string
}

// Close releases the descriptor.
func (n *Node) Close() { unix.Close(n.FD) }

// IsDir reports whether the node is a directory.
func (n *Node) IsDir() bool { return n.Stat.Mode&unix.S_IFMT == unix.S_IFDIR }

// Open resolves an absolute path without following symlinks anywhere.
func Open(path string) (*Node, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("path must be absolute: %q", path)
	}
	clean := filepath.Clean(path)
	dirfd, err := unix.Open("/", unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	if clean == "/" {
		return stat(dirfd, clean)
	}
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	for i, name := range parts {
		fd, err := unix.Openat(dirfd, name, unix.O_PATH|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		unix.Close(dirfd)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", "/"+strings.Join(parts[:i+1], "/"), err)
		}
		n, err := stat(fd, "/"+strings.Join(parts[:i+1], "/"))
		if err != nil {
			return nil, err
		}
		last := i == len(parts)-1
		if !last && !n.IsDir() {
			n.Close()
			return nil, fmt.Errorf("%s: not a directory", n.Path)
		}
		if last {
			return n, nil
		}
		dirfd = fd
	}
	return nil, fmt.Errorf("unreachable")
}

// Child opens a directory entry relative to an already-verified directory,
// again without following symlinks.
func (n *Node) Child(name string) (*Node, error) {
	fd, err := unix.Openat(n.FD, name, unix.O_PATH|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(n.Path, name), err)
	}
	return stat(fd, filepath.Join(n.Path, name))
}

// Names lists a verified directory's entries.
func (n *Node) Names() ([]string, error) {
	// O_PATH descriptors can't be read; reopen this exact directory for
	// reading through its /proc magic link - which refers to the same
	// inode, not to a path that could be swapped.
	fd, err := unix.Open(n.procPath(), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)
	var names []string
	buf := make([]byte, 64*1024)
	for {
		nb, err := unix.Getdents(fd, buf)
		if err != nil {
			return nil, err
		}
		if nb <= 0 {
			break
		}
		_, _, found := unix.ParseDirent(buf[:nb], -1, nil)
		for _, name := range found {
			if name != "." && name != ".." {
				names = append(names, name)
			}
		}
	}
	return names, nil
}

// Chmod sets the permission bits of exactly this object.
func (n *Node) Chmod(mode uint32) error {
	// fchmod() rejects O_PATH descriptors; chmod through the descriptor's
	// /proc magic link reaches the same inode (what glibc's fchmodat does).
	return unix.Chmod(n.procPath(), mode)
}

// Chown sets the owner and group of exactly this object (-1 = unchanged).
func (n *Node) Chown(uid, gid int) error {
	return unix.Fchownat(n.FD, "", uid, gid, unix.AT_EMPTY_PATH)
}

func (n *Node) procPath() string { return "/proc/self/fd/" + strconv.Itoa(n.FD) }

func stat(fd int, path string) (*Node, error) {
	n := &Node{FD: fd, Path: path}
	if err := unix.Fstat(fd, &n.Stat); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if n.Stat.Mode&unix.S_IFMT == unix.S_IFLNK {
		unix.Close(fd)
		return nil, fmt.Errorf("%s: %w", path, ErrSymlink)
	}
	return n, nil
}

// ModeBits returns the permission bits, including setuid/setgid/sticky.
func (n *Node) ModeBits() uint32 { return n.Stat.Mode & 07777 }

// Owner returns the object's uid and gid.
func (n *Node) Owner() (uid, gid int) { return int(n.Stat.Uid), int(n.Stat.Gid) }

// ProcPath is a path that reaches exactly this object ("/proc/self/fd/N"),
// for APIs that take a path: an entry below it ("<ProcPath>/name") is
// resolved relative to this verified directory, never through a path that
// could have been swapped since.
func (n *Node) ProcPath() string { return n.procPath() }

// Reopen opens this exact object for reading or writing (flag: O_RDONLY,
// O_WRONLY|O_APPEND, O_RDWR, ...). O_PATH descriptors can't be read or
// written; reopening through the /proc magic link reaches the same inode.
func (n *Node) Reopen(flag int) (*os.File, error) {
	fd, err := unix.Open(n.procPath(), flag|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", n.Path, err)
	}
	return os.NewFile(uintptr(fd), n.Path), nil
}

// OpenFile opens an existing file without following a symlink in any
// component, including the file itself.
func OpenFile(path string, flag int) (*os.File, error) {
	n, err := Open(path)
	if err != nil {
		return nil, err
	}
	defer n.Close()
	return n.Reopen(flag)
}

// MkdirAll opens directory path, creating missing components (mode perm),
// without following a symlink in any component.
func MkdirAll(path string, perm uint32) (*Node, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("path must be absolute: %q", path)
	}
	dir, err := Open("/")
	if err != nil {
		return nil, err
	}
	clean := filepath.Clean(path)
	if clean == "/" {
		return dir, nil
	}
	for _, name := range strings.Split(strings.TrimPrefix(clean, "/"), "/") {
		next, err := dir.Child(name)
		if errors.Is(err, unix.ENOENT) {
			if mkErr := unix.Mkdirat(dir.FD, name, perm); mkErr != nil && !errors.Is(mkErr, unix.EEXIST) {
				dir.Close()
				return nil, fmt.Errorf("%s: %w", filepath.Join(dir.Path, name), mkErr)
			}
			next, err = dir.Child(name)
		}
		dir.Close()
		if err != nil {
			return nil, err
		}
		if !next.IsDir() {
			next.Close()
			return nil, fmt.Errorf("%s: not a directory", next.Path)
		}
		dir = next
	}
	return dir, nil
}

// CreateFile opens path for writing, creating it (mode perm) if flag has
// O_CREAT - and the missing parent directories if mkdirs - without
// following a symlink in any component: a symlink planted where the file
// should be is refused (ELOOP), not written through.
func CreateFile(path string, flag int, perm uint32, mkdirs bool) (*os.File, error) {
	clean := filepath.Clean(path)
	dirPath, name := filepath.Split(clean)
	if name == "" {
		return nil, fmt.Errorf("%s: not a file path", path)
	}
	var parent *Node
	var err error
	if mkdirs {
		parent, err = MkdirAll(dirPath, 0755)
	} else {
		parent, err = Open(dirPath)
	}
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	if !parent.IsDir() {
		return nil, fmt.Errorf("%s: not a directory", parent.Path)
	}
	fd, err := unix.Openat(parent.FD, name, flag|unix.O_NOFOLLOW|unix.O_CLOEXEC, perm)
	if errors.Is(err, unix.ELOOP) {
		return nil, fmt.Errorf("%s: %w", clean, ErrSymlink)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", clean, err)
	}
	return os.NewFile(uintptr(fd), clean), nil
}

// Chdir makes this directory the process's working directory, so relative
// paths are resolved from it rather than through a path that could have
// been swapped since it was checked.
func (n *Node) Chdir() error { return unix.Fchdir(n.FD) }
