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

func (t *HasNode) Ki() Ki { return &t.KiNode }

var KtHasNode = KiTypes.AddType(&HasNode{})

type NodeEmbed struct {
	Node
	Ptr  KiPtr
	Mbr1 string
	Mbr2 int
}

// func (t *NodeEmbed) Ki() Ki { return t }

var KtNodeEmbed = KiTypes.AddType(&NodeEmbed{})

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
	parent.SetRoot()
	child := NodeEmbed{}
	// Note: must pass child.KiNode as a pointer  -- if it is a plain Node it is ok but
	// as a member of a struct, for somewhat obscure reasons having to do with the
	// fact that an interface is implicitly a pointer, you need to pass as a pointer here
	parent.AddChild(&child)
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
	parent.SetRoot()
	// parent.SetChildType(reflect.TypeOf(nod))
	err := parent.SetChildType(reflect.TypeOf(parent))
	if err != nil {
		t.Error(err)
	}
	child := parent.AddNewChild()
	child.SetName("child1")
	if len(parent.Children) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.Children))
	}
	if child.Path() != ".par1.child1" {
		t.Errorf("child path != correct, was %v", child.Path())
	}
	if reflect.TypeOf(child).Elem() != parent.ChildType.T {
		t.Errorf("child type != correct, was %T", child)
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

func TestNodeEmbedJSonSave(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetName("par1")
	parent.SetRoot()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	parent.SetChildType(reflect.TypeOf(parent))
	// child1 :=
	parent.AddNewChildNamed("child1")
	var child2 *NodeEmbed = parent.AddNewChildNamed("child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChildNamed("child1")

	child2.SetChildType(reflect.TypeOf(parent))
	schild2 := child2.AddNewChildNamed("subchild1")

	parent.Ptr.Ptr = child2
	child2.Ptr.Ptr = schild2

	b, err := json.MarshalIndent(parent, "", "  ")
	if err != nil {
		t.Error(err)
	} else {
		fmt.Printf("json output: %v\n", string(b))
	}

	tstload := NodeEmbed{}
	err = json.Unmarshal(b, &tstload)
	if err != nil {
		t.Error(err)
	} else {
		tstload.UnmarshalPost(&tstload)
		tstb, _ := json.MarshalIndent(tstload, "", "  ")
		fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
		}
	}
}

func NodeTestFun1(n Ki, d interface{}) bool {
	fmt.Printf("node fun1 on: %v, data %v\n", n.KiUniqueName(), d)
	return true
}

func TestNodeCallFun(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetName("par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	parent.SetChildType(reflect.TypeOf(parent))
	// child1 :=
	parent.AddNewChildNamed("child1")
	child2 := parent.AddNewChildNamed("child1")
	// child3 :=
	parent.AddNewChildNamed("child1")

	child2.SetChildType(reflect.TypeOf(parent))
	schild2 := child2.AddNewChildNamed("subchild1")

	parent.FunDown(&parent, NodeTestFun1, "fun down")
	schild2.FunUp(schild2, NodeTestFun1, "fun up")
	schild2.FunUp(schild2, func(n Ki, d interface{}) bool {
		fmt.Printf("node anon fun on: %v, data %v\n", n.KiUniqueName(), d)
		return true
	}, "fun up2")

}

func TestNodeUpdate(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetName("par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	parent.SetChildType(reflect.TypeOf(parent))
	parent.NodeSignal().Connect(&parent, func(r, s Ki, sig SignalType, d interface{}) {
		fmt.Printf("node %v was updated sig %v\n", s.KiName(), sig)
	})
	// child1 :=
	parent.AddNewChildNamed("child1")
	child2 := parent.AddNewChildNamed("child1")
	// child3 :=
	parent.UpdateStart()
	parent.AddNewChildNamed("child1")
	parent.UpdateEnd(false)

	parent.FunDown(&parent, func(n Ki, d interface{}) bool {
		fmt.Printf("node %v updt count %v\n", n.KiUniqueName(), *n.UpdateCtr())
		return true
	}, "upcnt")

	child2.SetChildType(reflect.TypeOf(parent))
	schild2 := child2.AddNewChildNamed("subchild1")

	child2.NodeSignal().Connect(&parent, func(r, s Ki, sig SignalType, d interface{}) {
		fmt.Printf("node %v was updated sig %v\n", s.KiName(), sig)
	})
	schild2.NodeSignal().Connect(&parent, func(r, s Ki, sig SignalType, d interface{}) {
		fmt.Printf("node %v was updated sig %v\n", s.KiName(), sig)
	})

	fmt.Print("\nnode update all starting\n")
	child2.UpdateStart()
	schild2.UpdateStart()
	schild2.UpdateEnd(true)
	child2.UpdateEnd(true)

	// parent.FunDown(func(n Ki, d interface{}) bool {
	// 	fmt.Printf("node %v updt count %v\n", n.KiUniqueName(), *n.UpdateCtr())
	// 	return true
	// }, "upcnt")

	fmt.Print("\nnode update top starting\n")
	child2.UpdateStart()
	schild2.UpdateStart()
	schild2.UpdateEnd(false)
	child2.UpdateEnd(false)

	fmt.Print("\nfinal node counts should all be zero\n")
	parent.FunDown(&parent, func(n Ki, d interface{}) bool {
		fmt.Printf("node %v updt count %v\n", n.KiUniqueName(), *n.UpdateCtr())
		return true
	}, "upcnt")
}
