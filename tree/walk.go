// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
This file provides basic tree walking functions for iterative traversal
of the tree in up / down directions.  As compared to the core walk methods,
these are for more dynamic, piecemeal processing.
*/

package tree

// Last returns the last node in the tree
func Last(nd Node) Node {
	nd = LastChild(nd)
	last := nd
	nd.WalkPre(func(k Node) bool {
		last = k
		return Continue
	})
	return last
}

// LastChild returns the last child under given node, or node itself if no children
func LastChild(nd Node) Node {
	if nd.HasChildren() {
		ek, err := nd.Children().ElemFromEndTry(0)
		if err == nil {
			return LastChild(ek)
		}
	}
	return nd
}

// Prev returns previous node in the tree -- nil if top
func Prev(nd Node) Node {
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

// Next returns next node in the tree, nil if end
func Next(nd Node) Node {
	if !nd.HasChildren() {
		return NextSibling(nd)
	}
	return nd.Child(0)
}

// NextSibling returns next sibling or nil if none
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
