// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

type HasNode struct {
	//	Node ki.Node // others will use it this way
	KiNode Node
	Mbr1   string
	Mbr2   int
}

type NodeEmbed struct {
	Node
	Mbr1 string
	Mbr2 int
}

func TestNodeAddChild(t *testing.T) {
	parent := HasNode{}
	parent.KiNode.SetName("par1")
	child := HasNode{}
	// Note: must pass child.KiNode as a pointer  -- if it is a plain Node it is ok but
	// as a member of a struct, for somewhat obscure reasons having to do with the
	// fact that an interface is implicitly a pointer, you need to pass as a pointer here
	parent.KiNode.AddChild(&child.KiNode)
	child.KiNode.SetName("child1")
	if len(parent.KiNode.Children) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.KiNode.Children))
	}
	if child.KiNode.Path() != ".par1.child1" {
		t.Errorf("child path != correct, was %v", child.KiNode.Path())
	}
}

func TestNodeEmbedAddChild(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetName("par1")
	child := NodeEmbed{}
	// Note: must pass child.KiNode as a pointer  -- if it is a plain Node it is ok but
	// as a member of a struct, for somewhat obscure reasons having to do with the
	// fact that an interface is implicitly a pointer, you need to pass as a pointer here
	parent.AddChild(&child.Node)
	child.SetName("child1")
	if len(parent.Children) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.Children))
	}
	if child.Path() != ".par1.child1" {
		t.Errorf("child path != correct, was %v", child.Path())
	}
}

func TestNodeEmbedAddNewChild(t *testing.T) {
	// nod := Node{}
	parent := NodeEmbed{}
	parent.SetName("par1")
	// parent.SetChildType(reflect.TypeOf(nod))
	parent.SetChildType(reflect.TypeOf(parent))
	child, err := parent.AddNewChild()
	if err != nil {
		t.Error(err)
	}
	child.SetName("child1")
	if len(parent.Children) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.Children))
	}
	if child.Path() != ".par1.child1" {
		t.Errorf("child path != correct, was %v", child.Path())
	}
}

func TestNodeUniqueNames(t *testing.T) {
	parent := HasNode{}
	parent.KiNode.SetName("par1")
	child := HasNode{}
	parent.KiNode.AddChildNamed(&child.KiNode, "child1")
	child2 := HasNode{}
	parent.KiNode.AddChildNamed(&child2.KiNode, "child1")
	child3 := HasNode{}
	parent.KiNode.AddChildNamed(&child3.KiNode, "child1")
	if len(parent.KiNode.Children) != 3 {
		t.Errorf("Children length != 3, was %d", len(parent.KiNode.Children))
	}
	if pth := child.KiNode.PathUnique(); pth != ".par1.child1" {
		t.Errorf("child path != correct, was %v", pth)
	}
	if pth := child2.KiNode.PathUnique(); pth != ".par1.child1_1" {
		t.Errorf("child2 path != correct, was %v", pth)
	}
	if pth := child3.KiNode.PathUnique(); pth != ".par1.child1_2" {
		t.Errorf("child3 path != correct, was %v", pth)
	}

}

func TestNodeRemoveChild(t *testing.T) {
	parent := HasNode{}
	parent.KiNode.SetName("par1")
	child := HasNode{}
	parent.KiNode.AddChildNamed(&child.KiNode, "child1")
	parent.KiNode.RemoveChild(&child.KiNode, true)
	if len(parent.KiNode.Children) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.KiNode.Children))
	}
	if len(parent.KiNode.deleted) != 1 {
		t.Errorf("deleted length != 1, was %d", len(parent.KiNode.Children))
	}
}

func TestNodeRemoveChildName(t *testing.T) {
	parent := HasNode{}
	parent.KiNode.SetName("par1")
	child := HasNode{}
	parent.KiNode.AddChildNamed(&child.KiNode, "child1")
	parent.KiNode.RemoveChildName("child1", true)
	if len(parent.KiNode.Children) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.KiNode.Children))
	}
	if len(parent.KiNode.deleted) != 1 {
		t.Errorf("deleted length != 1, was %d", len(parent.KiNode.Children))
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

//////////////////////////////////////////
//  JSON I/O

func TestNodeJSonSave(t *testing.T) {
	parent := HasNode{}
	parent.KiNode.SetName("par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	child := HasNode{}
	parent.KiNode.AddChildNamed(&child.KiNode, "child1")
	child2 := HasNode{}
	parent.KiNode.AddChildNamed(&child2.KiNode, "child1")
	child3 := HasNode{}
	parent.KiNode.AddChildNamed(&child3.KiNode, "child1")

	b, err := json.Marshal(parent)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Printf("json output: %v\n", string(b))
	}

	tstload := HasNode{}
	err = json.Unmarshal(b, &tstload)
	if err != nil {
		t.Error(err)
	} else {
		tstb, _ := json.Marshal(tstload)
		// fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
		}
	}
}

func TestNodeEmbedJSonSave(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetName("par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	child := NodeEmbed{}
	parent.AddChildNamed(&child.Node, "child1")
	child2 := NodeEmbed{}
	parent.AddChildNamed(&child2.Node, "child1")
	child3 := NodeEmbed{}
	parent.AddChildNamed(&child3.Node, "child1")

	b, err := json.Marshal(parent)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output: %v\n", string(b))
	}

	tstload := NodeEmbed{}
	err = json.Unmarshal(b, &tstload)
	if err != nil {
		t.Error(err)
	} else {
		tstb, _ := json.Marshal(tstload)
		// fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
		}
	}
}
