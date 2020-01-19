// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package walki

import (
	"fmt"
	"testing"

	"github.com/goki/ki/ki"
)

var testTree *ki.Node

func init() {
	testTree = &ki.Node{}
	testTree.InitName(testTree, "root")
	// child1 :=
	testTree.AddNewChild(nil, "child0")
	var child2 = testTree.AddNewChild(nil, "child1")
	// child3 :=
	testTree.AddNewChild(nil, "child2")
	schild2 := child2.AddNewChild(nil, "subchild1")
	// sschild2 :=
	schild2.AddNewChild(nil, "subsubchild1")
	// child4 :=
	testTree.AddNewChild(nil, "child3")
}

func TestDown(t *testing.T) {
	cur := testTree
	for {
		fmt.Println(cur.PathUnique())
		curi := Next(cur)
		if curi == nil {
			break
		}
		cur = curi.(*ki.Node)
	}
}

func TestUp(t *testing.T) {
	cur := Last(testTree)
	for {
		fmt.Println(cur.PathUnique())
		curi := Prev(cur)
		if curi == nil {
			break
		}
		cur = curi.(*ki.Node)
	}
}
