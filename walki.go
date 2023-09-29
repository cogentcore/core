// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
This file provides basic tree walking functions for iterative traversal
of the tree in up / down directions.  As compared to the core Func methods,
these are for more dynamic, piecemeal processing.
*/

package ki

// Last returns the last node in the tree
func Last(nd Ki) Ki {
	var last Ki
	nd.WalkPre(func(k Ki) bool {
		last = k
		return Continue
	})
	return last
}

// LastChild returns the last child under given node, or node itself if no children
func LastChild(nd Ki) Ki {
	if nd.HasChildren() {
		ek, err := nd.Children().ElemFromEndTry(0)
		if err == nil {
			return LastChild(ek)
		}
	}
	return nd
}

// Prev returns previous node in the tree -- nil if top
func Prev(nd Ki) Ki {
	if nd.Parent() == nil {
		return nil
	}
	myidx, ok := nd.IndexInParent()
	if ok && myidx > 0 {
		nn := nd.Parent().Child(myidx - 1)
		return LastChild(nn)
	}
	return nd.Parent()
}

// Next returns next node in the tree, nil if end
func Next(nd Ki) Ki {
	if !nd.HasChildren() {
		return NextSibling(nd)
	}
	if nd.HasChildren() {
		return nd.Child(0)
	}
	return nil
}

// NextSibling returns next sibling or nil if none
func NextSibling(nd Ki) Ki {
	if nd.Parent() == nil {
		return nil
	}
	myidx, ok := nd.IndexInParent()
	if ok {
		if myidx < nd.Parent().NumChildren()-1 {
			return nd.Parent().Child(myidx + 1)
		}
	}
	return NextSibling(nd.Parent())
}
