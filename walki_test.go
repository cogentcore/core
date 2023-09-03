// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"reflect"
	"testing"
)

var testTree *Node

func init() {
	testTree = &Node{}
	typ := reflect.TypeOf(testTree).Elem()
	testTree.InitName(testTree, "root")
	// child1 :=
	testTree.AddNewChild(typ, "child0")
	var child2 = testTree.AddNewChild(typ, "child1")
	// child3 :=
	testTree.AddNewChild(typ, "child2")
	schild2 := child2.AddNewChild(typ, "subchild1")
	// sschild2 :=
	schild2.AddNewChild(typ, "subsubchild1")
	// child4 :=
	testTree.AddNewChild(typ, "child3")
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
