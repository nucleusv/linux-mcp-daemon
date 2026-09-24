//go:build !linux

package fsafe

import (
	"errors"
	"os"
)

// ErrSymlink is returned when a path component is a symbolic link.
var ErrSymlink = errors.New("refusing to follow a symbolic link")

var errUnsupported = errors.New("fsafe is Linux-only (openat O_PATH)")

// Node mirrors the Linux type so callers compile on other platforms.
type Node struct {
	Path string
}

func Open(path string) (*Node, error)             { return nil, errUnsupported }
func (n *Node) Close()                            {}
func (n *Node) IsDir() bool                       { return false }
func (n *Node) Child(name string) (*Node, error)  { return nil, errUnsupported }
func (n *Node) Names() ([]string, error)          { return nil, errUnsupported }
func (n *Node) Chmod(mode uint32) error           { return errUnsupported }
func (n *Node) Chown(uid, gid int) error          { return errUnsupported }
func (n *Node) ModeBits() uint32                  { return 0 }
func (n *Node) Owner() (uid, gid int)             { return -1, -1 }
func (n *Node) ProcPath() string                  { return "" }
func (n *Node) Reopen(flag int) (*os.File, error) { return nil, errUnsupported }
func OpenFile(path string, flag int) (*os.File, error) {
	return nil, errUnsupported
}
func MkdirAll(path string, perm uint32) (*Node, error) { return nil, errUnsupported }
func CreateFile(path string, flag int, perm uint32, mkdirs bool) (*os.File, error) {
	return nil, errUnsupported
}
func (n *Node) Chdir() error { return errUnsupported }
