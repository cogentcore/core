// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
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
	child.node.SetName("child1")
	err := parent.node.AddChild(&child.node)
	if err != nil {
		t.Error(err)
	}
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
	parent.node.AddChild(&child.node)
	child.node.SetName("child1")
	child2 := HasNode{}
	parent.node.AddChild(&child2.node)
	child2.node.SetName("child1")
	child3 := HasNode{}
	parent.node.AddChild(&child3.node)
	child3.node.SetName("child1")
	if len(parent.node.Children) != 3 {
		t.Errorf("Children length != 3, was %d", len(parent.node.Children))
	}
	fmt.Print(child.node.PathUnique() + "\n")
	fmt.Print(child2.node.PathUnique() + "\n")
	fmt.Print(child3.node.PathUnique() + "\n")
}

func TestNodeRemoveChild(t *testing.T) {
	parent := HasNode{
		node: Node{Name: "par1"},
	}
	child := HasNode{
		node: Node{Name: "child1"},
	}
	err := parent.node.AddChild(&child.node)
	if err != nil {
		t.Error(err)
	}
	parent.node.RemoveChild(&child.node, true)
	if len(parent.node.Children) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.node.Children))
	}
	if len(parent.node.deleted) != 1 {
		t.Errorf("deleted length != 1, was %d", len(parent.node.Children))
	}
}

func TestNodeRemoveChildName(t *testing.T) {
	parent := HasNode{
		node: Node{Name: "par1"},
	}
	child := HasNode{
		node: Node{Name: "child1"},
	}
	parent.node.AddChild(&child.node)
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
		child.Name = nm
		parent.AddChild(child)
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
