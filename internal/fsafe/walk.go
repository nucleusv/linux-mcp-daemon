package fsafe

import (
	"errors"
	"sort"
)

// WalkResult summarizes a recursive change.
type WalkResult struct {
	Changed, Unchanged int
	SkippedSymlinks    []string
	Errors             []string
}

// Walk calls fn on root and, if root is a directory, on everything below
// it, depth-first in name order. Each child is opened relative to its
// verified parent directory, so the walk can never be redirected through
// a symlink: symlinks met along the way are skipped and reported, never
// followed. fn returns whether it changed anything.
func Walk(root *Node, fn func(*Node) (bool, error)) WalkResult {
	var res WalkResult
	var visit func(n *Node)
	visit = func(n *Node) {
		changed, err := fn(n)
		switch {
		case err != nil:
			res.Errors = append(res.Errors, err.Error())
		case changed:
			res.Changed++
		default:
			res.Unchanged++
		}
		if !n.IsDir() {
			return
		}
		names, err := n.Names()
		if err != nil {
			res.Errors = append(res.Errors, n.Path+": "+err.Error())
			return
		}
		sort.Strings(names)
		for _, name := range names {
			child, err := n.Child(name)
			if errors.Is(err, ErrSymlink) {
				res.SkippedSymlinks = append(res.SkippedSymlinks, n.Path+"/"+name)
				continue
			}
			if err != nil {
				res.Errors = append(res.Errors, err.Error())
				continue
			}
			visit(child)
			child.Close()
		}
	}
	visit(root)
	return res
}
