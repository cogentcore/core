// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	//	"fmt"
	"testing"
)

type HasNode struct {
	node Node
	mbr1 string
	mbr2 int
}

func TestNodeAddChild(t *testing.T) {
	parent := HasNode{}
	parent.node.SetName("par1")
	child := HasNode{}
	err := parent.node.AddChild(&child.node)
	if err != nil {
		t.Error(err)
	}
	child.node.SetName("child1")
	if len(parent.node.Children) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.node.Children))
	}
	if child.node.Path() != ".par1.child1" {
		t.Errorf("child path != correct, was %v", child.node.Path())
	}
}

func TestNodeUniqueNames(t *testing.T) {
	parent := HasNode{}
	parent.node.SetName("par1")
	child := HasNode{}
	parent.node.AddChildNamed(&child.node, "child1")
	child2 := HasNode{}
	parent.node.AddChildNamed(&child2.node, "child1")
	child3 := HasNode{}
	parent.node.AddChildNamed(&child3.node, "child1")
	if len(parent.node.Children) != 3 {
		t.Errorf("Children length != 3, was %d", len(parent.node.Children))
	}
	if pth := child.node.PathUnique(); pth != ".par1.child1" {
		t.Errorf("child path != correct, was %v", pth)
	}
	if pth := child2.node.PathUnique(); pth != ".par1.child1_1" {
		t.Errorf("child2 path != correct, was %v", pth)
	}
	if pth := child3.node.PathUnique(); pth != ".par1.child1_2" {
		t.Errorf("child3 path != correct, was %v", pth)
	}

}

func TestNodeRemoveChild(t *testing.T) {
	parent := HasNode{}
	parent.node.SetName("par1")
	child := HasNode{}
	parent.node.AddChildNamed(&child.node, "child1")
	parent.node.RemoveChild(&child.node, true)
	if len(parent.node.Children) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.node.Children))
	}
	if len(parent.node.deleted) != 1 {
		t.Errorf("deleted length != 1, was %d", len(parent.node.Children))
	}
}

func TestNodeRemoveChildName(t *testing.T) {
	parent := HasNode{}
	parent.node.SetName("par1")
	child := HasNode{}
	parent.node.AddChildNamed(&child.node, "child1")
	parent.node.RemoveChildName("child1", true)
	if len(parent.node.Children) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.node.Children))
	}
	if len(parent.node.deleted) != 1 {
		t.Errorf("deleted length != 1, was %d", len(parent.node.Children))
	}
}

func TestNodeFindName(t *testing.T) {
	names := [...]string{"name0", "name1", "name2", "name3", "name4", "name5"}
	parent := NewNode()
	for _, nm := range names {
		child := NewNode()
		parent.AddChildNamed(child, nm)
	}
	if len(parent.Children) != len(names) {
		t.Errorf("Children length != n, was %d", len(parent.Children))
	}
	for i, nm := range names {
		for st, _ := range names { // test all starting indexes
			idx := parent.FindChildNameIndex(nm, st)
			if idx != i {
				t.Errorf("find index was not correct val of %d, was %d", i, idx)
			}
		}
	}
}

func TestNodeFindNameUnique(t *testing.T) {
	names := [...]string{"child", "child_1", "child_2", "child_3", "child_4", "child_5"}
	parent := NewNode()
	for range names {
		child := NewNode()
		parent.AddChildNamed(child, "child")
	}
	if len(parent.Children) != len(names) {
		t.Errorf("Children length != n, was %d", len(parent.Children))
	}
	for i, nm := range names {
		for st, _ := range names { // test all starting indexes
			idx := parent.FindChildUniqueNameIndex(nm, st)
			if idx != i {
				t.Errorf("find index was not correct val of %d, was %d", i, idx)
			}
		}
	}
}
