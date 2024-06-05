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
	n = LastChild(n)
	last := n
	n.WalkDown(func(k Node) bool {
		last = k
		return Continue
	})
	return last
}

// LastChild returns the last child under the given node,
// or the node itself if it has no children.
func LastChild(n Node) Node {
	if n.HasChildren() {
		return LastChild(n.Child(n.NumChildren() - 1))
	}
	return n
}

// Previous returns the previous node in the tree,
// or nil if this is the root node.
func Previous(n Node) Node {
	if n.Parent() == nil {
		return nil
	}
	myidx := n.AsTree().IndexInParent()
	if myidx > 0 {
		nn := n.Parent().Child(myidx - 1)
		return LastChild(nn)
	}
	return n.Parent()
}

// Next returns next node in the tree,
// or nil if this is the last node.
func Next(n Node) Node {
	if !n.HasChildren() {
		return NextSibling(n)
	}
	return n.Child(0)
}

// NextSibling returns the next sibling of this node,
// or nil if it has none.
func NextSibling(n Node) Node {
	if n.Parent() == nil {
		return nil
	}
	myidx := n.AsTree().IndexInParent()
	if myidx >= 0 && myidx < n.Parent().NumChildren()-1 {
		return n.Parent().Child(myidx + 1)
	}
	return NextSibling(n.Parent())
}
