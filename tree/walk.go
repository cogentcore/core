// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
This file provides basic tree walking functions for iterative traversal
of the tree in up / down directions.  As compared to the Node walk methods,
these are for more dynamic, piecemeal processing.
*/

package tree

// Last returns the last node in the tree.
func Last(n Node) Node {
	n = lastChild(n)
	last := n
	n.AsTree().WalkDown(func(k Node) bool {
		last = k
		return Continue
	})
	return last
}

// lastChild returns the last child under the given node,
// or the node itself if it has no children.
func lastChild(n Node) Node {
	nb := n.AsTree()
	if nb.HasChildren() {
		return lastChild(nb.Child(nb.NumChildren() - 1))
	}
	return n
}

// Previous returns the previous node in the tree,
// or nil if this is the root node.
func Previous(n Node) Node {
	nb := n.AsTree()
	if nb.Parent == nil {
		return nil
	}
	myidx := n.AsTree().IndexInParent()
	if myidx > 0 {
		nn := nb.Parent.AsTree().Child(myidx - 1)
		return lastChild(nn)
	}
	return nb.Parent
}

// Next returns next node in the tree,
// or nil if this is the last node.
func Next(n Node) Node {
	if !n.AsTree().HasChildren() {
		return NextSibling(n)
	}
	return n.AsTree().Child(0)
}

// NextSibling returns the next sibling of this node,
// or nil if it has none.
func NextSibling(n Node) Node {
	nb := n.AsTree()
	if nb.Parent == nil {
		return nil
	}
	myidx := n.AsTree().IndexInParent()
	if myidx >= 0 && myidx < nb.Parent.AsTree().NumChildren()-1 {
		return nb.Parent.AsTree().Child(myidx + 1)
	}
	return NextSibling(nb.Parent)
}
