// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import "iter"

// All returns an iterator over this node and all of its descendants, in
// depth-first order, as [NodeBase.WalkDown] visits them (including the
// nodes that [Node.NodeWalkDown] adds, such as the parts of widgets).
// Unlike with WalkDown, where [Break] only skips the children of a node,
// breaking out of the loop stops the iteration:
//
//	for n := range root.All() {
//		if isTarget(n) {
//			return n
//		}
//	}
//
// To skip the children of some nodes, use WalkDown.
func (n *NodeBase) All() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		stopped := false
		n.WalkDown(func(c Node) bool {
			if stopped || !yield(c) {
				stopped = true // the remaining nodes are skipped
				return Break
			}
			return Continue
		})
	}
}

// Ancestors returns an iterator over the parents of this node, from its
// parent up to the root, as [NodeBase.WalkUpParent] visits them.
func (n *NodeBase) Ancestors() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		n.WalkUpParent(func(p Node) bool {
			return yield(p)
		})
	}
}
