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
func Last(nd Node) Node {
	nd = LastChild(nd)
	last := nd
	nd.WalkDown(func(k Node) bool {
		last = k
		return Continue
	})
	return last
}

// LastChild returns the last child under the given node,
// or the node itself if it has no children.
func LastChild(nd Node) Node {
	if nd.HasChildren() {
		ek, err := nd.Children().ElemFromEndTry(0)
		if err == nil {
			return LastChild(ek)
		}
	}
	return nd
}

// Previous returns the previous node in the tree,
// or nil if this is the root node.
func Previous(nd Node) Node {
	if nd.Parent() == nil {
		return nil
	}
	myidx := nd.IndexInParent()
	if myidx > 0 {
		nn := nd.Parent().Child(myidx - 1)
		return LastChild(nn)
	}
	return nd.Parent()
}

// Next returns next node in the tree,
// or nil if this is the last node.
func Next(nd Node) Node {
	if !nd.HasChildren() {
		return NextSibling(nd)
	}
	return nd.Child(0)
}

// NextSibling returns the next sibling of this node,
// or nil if it has none.
func NextSibling(nd Node) Node {
	if nd.Parent() == nil {
		return nil
	}
	myidx := nd.IndexInParent()
	if myidx >= 0 && myidx < nd.Parent().NumChildren()-1 {
		return nd.Parent().Child(myidx + 1)
	}
	return NextSibling(nd.Parent())
}
