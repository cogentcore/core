// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"testing"
)

var testTree *Node

func init() {
	testTree = &Node{}
	typ := testTree.Type()
	testTree.InitName(testTree, "root")
	// child1 :=
	testTree.NewChild(typ, "child0")
	var child2 = testTree.NewChild(typ, "child1")
	// child3 :=
	testTree.NewChild(typ, "child2")
	schild2 := child2.NewChild(typ, "subchild1")
	// sschild2 :=
	schild2.NewChild(typ, "subsubchild1")
	// child4 :=
	testTree.NewChild(typ, "child3")
}

func TestDown(t *testing.T) {
	cur := testTree
	for {
		fmt.Println(cur.Path())
		curi := Next(cur)
		if curi == nil {
			break
		}
		cur = curi.(*Node)
	}
}

func TestUp(t *testing.T) {
	cur := Last(testTree)
	for {
		fmt.Println(cur.Path())
		curi := Prev(cur)
		if curi == nil {
			break
		}
		cur = curi.(*Node)
	}
}
