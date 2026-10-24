// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

// walkStack holds the current child index of each node on the path of a
// depth-first walk, in place of a map from node to index: the path has few
// nodes, and the node looked up is nearly always the deepest one, so it is
// searched from the end. It behaves as the map would, including for nodes
// that are not on it.
type walkStack []walkEntry

// walkEntry is the current child index of a node in a [walkStack].
type walkEntry struct {
	node  Node
	child int
}

// get returns the child index of n, or 0 if n is not in the stack.
func (ws walkStack) get(n Node) int {
	for i := len(ws) - 1; i >= 0; i-- {
		if ws[i].node == n {
			return ws[i].child
		}
	}
	return 0
}

// set sets the child index of n, adding n to the end if it is not in the stack.
func (ws *walkStack) set(n Node, child int) {
	s := *ws
	for i := len(s) - 1; i >= 0; i-- {
		if s[i].node == n {
			s[i].child = child
			return
		}
	}
	*ws = append(s, walkEntry{n, child})
}

// push adds n with the given child index to the end of the stack, for a
// node that is known not to be in it, such as a child of the deepest node,
// which cannot also be on the path above it.
func (ws *walkStack) push(n Node, child int) {
	*ws = append(*ws, walkEntry{n, child})
}

// delete removes n from the stack, if it is in it.
func (ws *walkStack) delete(n Node) {
	s := *ws
	for i := len(s) - 1; i >= 0; i-- {
		if s[i].node == n {
			*ws = append(s[:i], s[i+1:]...)
			return
		}
	}
}
