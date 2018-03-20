// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
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

var KiT_HasNode = Types.AddType(&HasNode{}, nil)

type NodeEmbed struct {
	Node
	Ptr  Ptr
	Mbr1 string
	Mbr2 int
}

var NodeEmbedProps = map[string]interface{}{
	"intprop":    -17,
	"floatprop":  3.1415,
	"stringprop": "type string",
}

var KiT_NodeEmbed = Types.AddType(&NodeEmbed{}, NodeEmbedProps)

func TestNodeAddChild(t *testing.T) {
	parent := HasNode{}
	parent.KiNode.SetThisName(&parent.KiNode, "par1")
	child := HasNode{}
	// Note: must pass child.KiNode as a pointer  -- if it is a plain Node it is ok but
	// as a member of a struct, for somewhat obscure reasons having to do with the
	// fact that an interface is implicitly a pointer, you need to pass as a pointer here
	parent.KiNode.AddChild(&child.KiNode)
	child.KiNode.SetName("child1")
	if len(parent.KiNode.Children) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.KiNode.Children))
	}
	if child.KiNode.KiParent() == nil {
		t.Errorf("child parent is nil")
	}
	if child.KiNode.Path() != ".par1.child1" {
		t.Errorf("child path != correct, was %v", child.KiNode.Path())
	}
}

func TestNodeEmbedAddChild(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetThisName(&parent, "par1")
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
	parent.SetThisName(&parent, "par1")
	// parent.SetChildType(reflect.TypeOf(nod))
	err := parent.SetChildType(reflect.TypeOf(parent))
	if err != nil {
		t.Error(err)
	}
	child := parent.AddNewChild(nil)
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
	parent.KiNode.SetThisName(&parent.KiNode, "par1")
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

func TestNodeDeleteChild(t *testing.T) {
	parent := HasNode{}
	parent.KiNode.SetThisName(&parent.KiNode, "par1")
	child := HasNode{}
	parent.KiNode.AddChildNamed(&child.KiNode, "child1")
	parent.KiNode.DeleteChild(&child.KiNode, true)
	if len(parent.KiNode.Children) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.KiNode.Children))
	}
	if len(parent.KiNode.Deleted) != 1 {
		t.Errorf("Deleted length != 1, was %d", len(parent.KiNode.Children))
	}
}

func TestNodeDeleteChildName(t *testing.T) {
	parent := HasNode{}
	parent.KiNode.SetThisName(&parent.KiNode, "par1")
	child := HasNode{}
	parent.KiNode.AddChildNamed(&child.KiNode, "child1")
	parent.KiNode.DeleteChildByName("child1", true)
	if len(parent.KiNode.Children) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.KiNode.Children))
	}
	if len(parent.KiNode.Deleted) != 1 {
		t.Errorf("Deleted length != 1, was %d", len(parent.KiNode.Children))
	}
}

func TestNodeFindName(t *testing.T) {
	names := [...]string{"name0", "name1", "name2", "name3", "name4", "name5"}
	parent := Node{}
	parent.SetThisName(&parent, "par")
	for _, nm := range names {
		child := Node{}
		parent.AddChildNamed(&child, nm)
	}
	if len(parent.Children) != len(names) {
		t.Errorf("Children length != n, was %d", len(parent.Children))
	}
	for i, nm := range names {
		for st, _ := range names { // test all starting indexes
			idx := parent.FindChildIndexByName(nm, st)
			if idx != i {
				t.Errorf("find index was not correct val of %d, was %d", i, idx)
			}
		}
	}
}

func TestNodeFindNameUnique(t *testing.T) {
	names := [...]string{"child", "child_1", "child_2", "child_3", "child_4", "child_5"}
	parent := Node{}
	parent.SetThisName(&parent, "par")
	for range names {
		child := Node{}
		parent.AddChildNamed(&child, "child")
	}
	if len(parent.Children) != len(names) {
		t.Errorf("Children length != n, was %d", len(parent.Children))
	}
	for i, nm := range names {
		for st, _ := range names { // test all starting indexes
			idx := parent.FindChildIndexByUniqueName(nm, st)
			if idx != i {
				t.Errorf("find index was not correct val of %d, was %d", i, idx)
			}
		}
	}
}

func TestNodeFindType(t *testing.T) {
	parent := Node{}
	parent.SetThisName(&parent, "par")
	parent.AddNewChildNamed(KiT_NodeEmbed, "child1")
	parent.AddNewChildNamed(KiT_Node, "child2")
	idx := parent.FindChildIndexByType(KiT_NodeEmbed)
	if idx != 0 {
		t.Errorf("find index was not correct val of %d, was %d", 0, idx)
	}
	idx = parent.FindChildIndexByType(KiT_Node)
	if idx != 1 {
		t.Errorf("find index was not correct val of %d, was %d", 1, idx)
	}
	cn := parent.FindChildByType(KiT_Node)
	if cn == nil {
		t.Error("find child by type was nil")
	}
}

//////////////////////////////////////////
//  JSON I/O

func TestNodeEmbedJSonSave(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetThisName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChildNamed(nil, "child1")
	var child2 *NodeEmbed = parent.AddNewChildNamed(nil, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChildNamed(nil, "child1")
	schild2 := child2.AddNewChildNamed(nil, "subchild1")

	parent.Ptr.Ptr = child2
	child2.Ptr.Ptr = schild2

	b, err := parent.SaveJSON(true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output: %v\n", string(b))
	}

	tstload := NodeEmbed{}
	tstload.SetThisName(&tstload, "")
	err = tstload.LoadJSON(b)
	if err != nil {
		t.Error(err)
	} else {
		tstb, _ := tstload.SaveJSON(true)
		// fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
		}
	}
}

//////////////////////////////////////////
//  function calling

func TestNodeCallFun(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetThisName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChildNamed(nil, "child1")
	child2 := parent.AddNewChildNamed(nil, "child1")
	// child3 :=
	parent.AddNewChildNamed(nil, "child1")
	schild2 := child2.AddNewChildNamed(nil, "subchild1")

	res := make([]string, 0, 10)
	parent.FunDownMeFirst(0, "fun_down", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("%v, %v, lev %v", k.KiUniqueName(), d, level))
		return true
	})
	// fmt.Printf("result: %v\n", res)

	trg := []string{"par1, fun_down, lev 0", "child1, fun_down, lev 1", "child1_1, fun_down, lev 1", "subchild1, fun_down, lev 2", "child1_2, fun_down, lev 1"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FunDown error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	schild2.FunUp(0, "fun_up", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("%v, %v", k.KiUniqueName(), d))
		return true
	})
	//	fmt.Printf("result: %v\n", res)

	trg = []string{"subchild1, fun_up", "child1_1, fun_up", "par1, fun_up"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FunUp error -- results: %v != target: %v\n", res, trg)
	}
}

func TestNodeUpdate(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetThisName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	res := make([]string, 0, 10)
	parent.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d interface{}) {
		res = append(res, fmt.Sprintf("%v sig %v", s.KiName(), NodeSignals(sig)))
	})
	// child1 :=
	parent.AddNewChildNamed(nil, "child1")
	child2 := parent.AddNewChildNamed(nil, "child1")
	// child3 :=
	parent.UpdateStart()
	parent.AddNewChildNamed(nil, "child1")
	parent.UpdateEnd()
	schild2 := child2.AddNewChildNamed(nil, "subchild1")

	// fmt.Printf("res: %v\n", res)
	trg := []string{"par1 sig NodeSignalChildAdded", "par1 sig NodeSignalChildAdded", "par1 sig NodeSignalUpdated"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("Add child sigs error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	child2.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d interface{}) {
		res = append(res, fmt.Sprintf("%v sig %v", s.KiName(), NodeSignals(sig)))
	})
	schild2.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d interface{}) {
		res = append(res, fmt.Sprintf("%v sig %v", s.KiName(), NodeSignals(sig)))
	})

	// fmt.Print("\nnode update all starting\n")
	child2.UpdateStart()
	schild2.UpdateStart()
	schild2.UpdateEndAll()
	child2.UpdateEndAll()

	// fmt.Printf("res: %v\n", res)
	trg = []string{"child1 sig NodeSignalUpdated", "subchild1 sig NodeSignalUpdated"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("update signal all error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	// fmt.Print("\nnode update top starting\n")
	child2.UpdateStart()
	schild2.UpdateStart()
	schild2.UpdateEnd()
	child2.UpdateEnd()

	// fmt.Printf("res: %v\n", res)
	trg = []string{"child1 sig NodeSignalUpdated"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("update signal only top error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	parent.FunDownMeFirst(0, "upcnt", func(n Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("%v %v", n.KiUniqueName(), *n.UpdateCtr()))
		return true
	})
	// fmt.Printf("res: %v\n", res)

	trg = []string{"par1 0", "child1 0", "child1_1 0", "subchild1 0", "child1_2 0"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("update counts error -- results: %v != target: %v\n", res, trg)
	}

}

func TestProps(t *testing.T) {
	parent := NodeEmbed{}
	parent.SetThisName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	res := make([]string, 0, 10)
	parent.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d interface{}) {
		res = append(res, fmt.Sprintf("%v sig %v", s.KiName(), sig))
	})
	// child1 :=
	parent.AddNewChildNamed(nil, "child1")
	child2 := parent.AddNewChildNamed(nil, "child1")
	// child3 :=
	parent.UpdateStart()
	parent.AddNewChildNamed(nil, "child1")
	parent.UpdateEnd()
	schild2 := child2.AddNewChildNamed(nil, "subchild1")

	parent.SetProp("intprop", 42)
	pprop, ok := ToInt(parent.Prop("intprop", false, false))
	if !ok || pprop != 42 {
		t.Errorf("TestProps error -- pprop %v != %v\n", pprop, 42)
	}
	sprop, ok := ToInt(schild2.Prop("intprop", true, false))
	if !ok || sprop != 42 {
		t.Errorf("TestProps error -- sprop inherited %v != %v\n", sprop, 42)
	}
	sprop, ok = ToInt(schild2.Prop("intprop", false, false))
	if ok || sprop != 0 {
		t.Errorf("TestProps error -- sprop not inherited %v != %v\n", sprop, 0)
	}

	parent.SetProp("floatprop", 42.0)
	spropf, ok := ToFloat(schild2.Prop("floatprop", true, false))
	if !ok || spropf != 42.0 {
		t.Errorf("TestProps error -- spropf inherited %v != %v\n", spropf, 42.0)
	}

	tstr := "test string"
	parent.SetProp("stringprop", tstr)
	sprops, ok := ToString(schild2.Prop("stringprop", true, false))
	if !ok || sprops != tstr {
		t.Errorf("TestProps error -- sprops inherited %v != %v\n", sprops, tstr)
	}

	parent.DeleteProp("floatprop")
	spropf, ok = ToFloat(schild2.Prop("floatprop", true, false))
	if ok || spropf != 0 {
		t.Errorf("TestProps error -- spropf inherited %v != %v\n", spropf, 0)
	}

	spropf, ok = ToFloat(parent.Prop("floatprop", true, true))
	if !ok || spropf != 3.1415 {
		t.Errorf("TestProps error -- spropf from type %v != %v\n", spropf, 3.1415)
	}

}

func TestTreeMod(t *testing.T) {
	NodeSignalTrace = true
	sigs := ""
	NodeSignalTraceString = &sigs

	tree1 := Node{}
	tree1.SetThisName(&tree1, "tree1")
	// child11 :=
	tree1.AddNewChildNamed(nil, "child11")
	child12 := tree1.AddNewChildNamed(nil, "child12")
	// child13 :=
	tree1.AddNewChildNamed(nil, "child13")
	// schild12 :=
	child12.AddNewChildNamed(nil, "subchild12")

	tree2 := Node{}
	tree2.SetThisName(&tree2, "tree2")
	// child21 :=
	tree2.AddNewChildNamed(nil, "child21")
	child22 := tree2.AddNewChildNamed(nil, "child22")
	// child23 :=
	tree2.AddNewChildNamed(nil, "child23")
	// schild22 :=
	child22.AddNewChildNamed(nil, "subchild22")

	// fmt.Printf("Setup Signals:\n%v", sigs)
	sigs = ""

	// fmt.Printf("#################################\n")

	// fmt.Printf("Trees before:\n%v%v", tree1, tree2)
	tree2.AddChild(child12)

	// fmt.Printf("#################################\n")
	// fmt.Printf("Trees after add child12 move:\n%v%v", tree1, tree2)

	mvsigs := `ki.Signal EmitGo from: tree1 sig: NodeSignalChildDeleted data: child12
	subchild12

ki.Signal EmitGo from: child12 sig: NodeSignalMoved data: tree1
	child11
	child13

ki.Signal EmitGo from: tree2 sig: NodeSignalChildAdded data: child12
	subchild12

`
	// fmt.Printf("Move Signals:\n%v", sigs)
	if sigs != mvsigs {
		t.Errorf("TestTreeMod child12 move signals not as expected: %v", sigs)
	}
	sigs = ""

	tree2.UpdateStart()
	tree2.DeleteChild(child12, true)
	tree2.UpdateEnd()

	// fmt.Printf("#################################\n")

	delsigs := `ki.Signal EmitGo from: child12 sig: NodeSignalDeleting data: <nil>
ki.Signal EmitGo from: tree2 sig: NodeSignalUpdated data: <nil>
ki.Signal EmitGo from: child12 sig: NodeSignalDestroying data: <nil>
ki.Signal EmitGo from: subchild12 sig: NodeSignalDeleting data: <nil>
ki.Signal EmitGo from: subchild12 sig: NodeSignalDestroying data: <nil>
`

	// fmt.Printf("Delete Signals:\n%v", sigs)
	if sigs != delsigs {
		t.Errorf("TestTreeMod child12 move signals not as expected: %v", sigs)
	}
	sigs = ""

}
