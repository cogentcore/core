// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package walki provides basic tree walking functions for iterative traversal
of the tree in up / down directions.  As compared to the core Func methods
defined in ki package, these are for more dynamic, piecemeal processing.
*/
package walki

import (
	"goki.dev/ki/v2/ki"
)

// Last returns the last node in the tree
func Last(nd ki.Ki) ki.Ki {
	var last ki.Ki
	nd.FuncDownMeFirst(0, nd, func(k ki.Ki, level int, d any) bool {
		last = k
		return ki.Continue
	})
	return last
}

// LastChild returns the last child under given node, or node itself if no children
func LastChild(nd ki.Ki) ki.Ki {
	if nd.HasChildren() {
		ek, err := nd.Children().ElemFromEndTry(0)
		if err == nil {
			return LastChild(ek)
		}
	}
	return nd
}

// Prev returns previous node in the tree -- nil if top
func Prev(nd ki.Ki) ki.Ki {
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
func Next(nd ki.Ki) ki.Ki {
	if !nd.HasChildren() {
		return NextSibling(nd)
	}
	if nd.HasChildren() {
		return nd.Child(0)
	}
	return nil
}

// NextSibling returns next sibling or nil if none
func NextSibling(nd ki.Ki) ki.Ki {
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
