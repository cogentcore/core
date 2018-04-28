// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/rcoreilly/goki/ki/kit"
)

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

var KiT_NodeEmbed = kit.Types.AddType(&NodeEmbed{}, NodeEmbedProps)

func (n *NodeEmbed) New() Ki { return &NodeEmbed{} }

type NodeWithField struct {
	NodeEmbed
	Field1 NodeEmbed
}

var KiT_NodeWithField = kit.Types.AddType(&NodeWithField{}, nil)

func (n *NodeWithField) New() Ki { return &NodeWithField{} }

type NodeField2 struct {
	NodeWithField
	Field2    NodeEmbed
	PtrIgnore *NodeEmbed
}

var KiT_NodeField2 = kit.Types.AddType(&NodeField2{}, nil)

func (n *NodeField2) New() Ki { return &NodeField2{} }

func TestNodeAddChild(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	child := NodeEmbed{}
	// Note: must pass child.KiNode as a pointer  -- if it is a plain Node it is ok but
	// as a member of a struct, for somewhat obscure reasons having to do with the
	// fact that an interface is implicitly a pointer, you need to pass as a pointer here
	parent.AddChild(&child)
	child.SetName("child1")
	if len(parent.Kids) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.Kids))
	}
	if child.Parent() == nil {
		t.Errorf("child parent is nil")
	}
	if child.Path() != "/par1/child1" {
		t.Errorf("child path != correct, was %v", child.Path())
	}
}

func TestNodeEmbedAddChild(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	child := NodeEmbed{}
	// Note: must pass child as a pointer  -- if it is a plain Node it is ok but
	// as a member of a struct, for somewhat obscure reasons having to do with the
	// fact that an interface is implicitly a pointer, you need to pass as a pointer here
	parent.AddChild(&child)
	child.SetName("child1")
	if len(parent.Kids) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.Kids))
	}
	if child.Path() != "/par1/child1" {
		t.Errorf("child path != correct, was %v", child.Path())
	}
}

func TestNodeEmbedAddNewChild(t *testing.T) {
	// nod := Node{}
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	// parent.SetChildType(reflect.TypeOf(nod))
	err := parent.SetChildType(reflect.TypeOf(parent))
	if err != nil {
		t.Error(err)
	}
	child := parent.AddNewChild(nil, "child1")
	if len(parent.Kids) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.Kids))
	}
	if child.Path() != "/par1/child1" {
		t.Errorf("child path != correct, was %v", child.Path())
	}
	if reflect.TypeOf(child).Elem() != parent.ChildType.T {
		t.Errorf("child type != correct, was %T", child)
	}
}

func TestNodeUniqueNames(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	child := parent.AddNewChild(nil, "child1")
	child2 := parent.AddNewChild(nil, "child1")
	child3 := parent.AddNewChild(nil, "child1")
	if len(parent.Kids) != 3 {
		t.Errorf("Children length != 3, was %d", len(parent.Kids))
	}
	if pth := child.PathUnique(); pth != "/par1/child1" {
		t.Errorf("child path != correct, was %v", pth)
	}
	if pth := child2.PathUnique(); pth != "/par1/child1_1" {
		t.Errorf("child2 path != correct, was %v", pth)
	}
	if pth := child3.PathUnique(); pth != "/par1/child1_2" {
		t.Errorf("child3 path != correct, was %v", pth)
	}

}

func TestNodeDeleteChild(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	child := parent.AddNewChild(nil, "child1")
	parent.DeleteChild(child, true)
	if len(parent.Kids) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.Kids))
	}
	if len(DelMgr.Dels) != 0 { // note: even though using destroy, UpdateEnd does destroy
		t.Errorf("Deleted length != 0, was %d", len(DelMgr.Dels))
	}
}

func TestNodeDeleteChildName(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.AddNewChild(nil, "child1")
	parent.DeleteChildByName("child1", true)
	if len(parent.Kids) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.Kids))
	}
	if len(DelMgr.Dels) != 0 { // note: even though using destroy, UpdateEnd does destroy
		t.Errorf("Deleted length != 0, was %d", len(DelMgr.Dels))
	}
}

func TestNodeFindName(t *testing.T) {
	names := [...]string{"name0", "name1", "name2", "name3", "name4", "name5"}
	parent := Node{}
	parent.InitName(&parent, "par")
	for _, nm := range names {
		parent.AddNewChild(nil, nm)
	}
	if len(parent.Kids) != len(names) {
		t.Errorf("Children length != n, was %d", len(parent.Kids))
	}
	for i, nm := range names {
		for st, _ := range names { // test all starting indexes
			idx := parent.ChildIndexByName(nm, st)
			if idx != i {
				t.Errorf("find index was not correct val of %d, was %d", i, idx)
			}
		}
	}
}

func TestNodeFindNameUnique(t *testing.T) {
	names := [...]string{"child", "child_1", "child_2", "child_3", "child_4", "child_5"}
	parent := Node{}
	parent.InitName(&parent, "par")
	for range names {
		parent.AddNewChild(nil, "child")
	}
	if len(parent.Kids) != len(names) {
		t.Errorf("Children length != n, was %d", len(parent.Kids))
	}
	for i, nm := range names {
		for st, _ := range names { // test all starting indexes
			idx := parent.ChildIndexByUniqueName(nm, st)
			if idx != i {
				t.Errorf("find index was not correct val of %d, was %d", i, idx)
			}
		}
	}
}

func TestNodeFindType(t *testing.T) {
	parent := Node{}
	parent.InitName(&parent, "par")
	parent.AddNewChild(KiT_NodeEmbed, "child1")
	parent.AddNewChild(KiT_Node, "child2")
	idx := parent.ChildIndexByType(KiT_NodeEmbed, false, 0)
	if idx != 0 {
		t.Errorf("find index was not correct val of %d, was %d", 0, idx)
	}
	idx = parent.ChildIndexByType(KiT_Node, false, 0)
	if idx != 1 {
		t.Errorf("find index was not correct val of %d, was %d", 1, idx)
	}
	cn := parent.ChildByType(KiT_Node, false, 0)
	if cn == nil {
		t.Error("find child by type was nil")
	}
}

func TestNodeMove(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(nil, "child0")
	var child2 *NodeEmbed = parent.AddNewChild(nil, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(nil, "child2")
	//schild2 :=
	child2.AddNewChild(nil, "subchild1")
	// child4 :=
	parent.AddNewChild(nil, "child3")

	bf := fmt.Sprintf("mv before:\n%v\n", parent.Kids)
	parent.MoveChild(3, 1)
	a31 := fmt.Sprintf("mv 3 -> 1:\n%v\n", parent.Kids)
	parent.MoveChild(0, 3)
	a03 := fmt.Sprintf("mv 0 -> 3:\n%v\n", parent.Kids)
	parent.MoveChild(1, 2)
	a12 := fmt.Sprintf("mv 1 -> 2:\n%v\n", parent.Kids)

	bft := `mv before:
[child0
 child1
	subchild1
 child2
 child3
]
`
	if bf != bft {
		t.Errorf("move error\n%v !=\n%v", bf, bft)
	}
	a31t := `mv 3 -> 1:
[child0
 child3
 child1
	subchild1
 child2
]
`
	if a31 != a31t {
		t.Errorf("move error\n%v !=\n%v", a31, a31t)
	}
	a03t := `mv 0 -> 3:
[child3
 child1
	subchild1
 child2
 child0
]
`
	if a03 != a03t {
		t.Errorf("move error\n%v !=\n%v", a03, a03t)
	}
	a12t := `mv 1 -> 2:
[child3
 child2
 child1
	subchild1
 child0
]
`
	if a12 != a12t {
		t.Errorf("move error\n%v !=\n%v", a12, a12t)
	}
}

func TestNodeConfig(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(nil, "child0")
	var child2 *NodeEmbed = parent.AddNewChild(nil, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(nil, "child2")
	//schild2 :=
	child2.AddNewChild(nil, "subchild1")
	// child4 :=
	parent.AddNewChild(nil, "child3")

	config1 := kit.TypeAndNameList{
		{KiT_NodeEmbed, "child2"},
		{KiT_NodeEmbed, "child3"},
		{KiT_NodeEmbed, "child1"},
	}

	// bf := fmt.Sprintf("mv before:\n%v\n", parent.Kids)

	mods, updt := parent.ConfigChildren(config1, false)
	if mods {
		parent.UpdateEnd(updt)
	}

	cf1 := fmt.Sprintf("config1:\n%v\n", parent.Kids)

	// config2 := kit.TypeAndNameList{
	// 	{KiT_NodeEmbed, "child4"},
	// 	{KiT_Node, "child1"}, // note: changing this to Node type removes child1.subchild1
	// 	{KiT_NodeEmbed, "child5"},
	// 	{KiT_NodeEmbed, "child3"},
	// 	{KiT_NodeEmbed, "child6"},
	// }

	config3 := kit.TypeAndNameList{}
	// fmt.Printf("NodeEmbed type name: %v\n", kit.FullTypeName(KiT_NodeEmbed))
	netn := kit.FullTypeName(KiT_NodeEmbed)
	ntn := kit.FullTypeName(KiT_Node)
	err := config3.SetFromString("{" + netn + ", child4}, {" + ntn + ", child1}, {" + netn + ", child5}, {" + netn + ", child3}, {" + netn + ", child6}")
	if err != nil {
		t.Errorf("%v", err)
	}

	mods, updt = parent.ConfigChildren(config3, false)
	if mods {
		parent.UpdateEnd(updt)
	}

	cf2 := fmt.Sprintf("config2:\n%v\n", parent.Kids)

	cf1t := `config1:
[child2
 child3
 child1
	subchild1
]
`
	if cf1 != cf1t {
		t.Errorf("config error\n%v !=\n%v", cf1, cf1t)
	}

	cf2t := `config2:
[child4
 child1
 child5
 child3
 child6
]
`
	if cf2 != cf2t {
		t.Errorf("config error\n%v !=\n%v", cf2, cf2t)
	}
}

//////////////////////////////////////////
//  JSON I/O

func TestNodeJSonSave(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(nil, "child1")
	var child2 *NodeEmbed = parent.AddNewChild(nil, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(nil, "child1")
	schild2 := child2.AddNewChild(nil, "subchild1")

	parent.Ptr.Ptr = child2
	child2.Ptr.Ptr = schild2

	b, err := parent.SaveJSON(true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output: %v\n", string(b))
	}

	tstload := NodeEmbed{}
	tstload.InitName(&tstload, "")
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

func TestNodeXMLSave(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(nil, "child1")
	var child2 *NodeEmbed = parent.AddNewChild(nil, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(nil, "child1")
	schild2 := child2.AddNewChild(nil, "subchild1")

	parent.Ptr.Ptr = child2
	child2.Ptr.Ptr = schild2

	b, err := parent.SaveXML(true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("xml output:\n%v\n", string(b))
	}

	tstload := NodeEmbed{}
	tstload.InitName(&tstload, "")
	err = tstload.LoadXML(b)
	if err != nil {
		t.Error(err)
	} else {
		tstb, _ := tstload.SaveXML(true)
		// fmt.Printf("test loaded json output:\n%v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd XML rep are not equivalent")
		}
	}
}

//////////////////////////////////////////
//  function calling

func TestNodeCallFun(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(nil, "child1")
	child2 := parent.AddNewChild(nil, "child1")
	// child3 :=
	parent.AddNewChild(nil, "child1")
	schild2 := child2.AddNewChild(nil, "subchild1")

	res := make([]string, 0, 10)
	parent.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("%v, %v, lev %v", k.UniqueName(), d, level))
		return true
	})
	// fmt.Printf("result: %v\n", res)

	trg := []string{"par1, fun_down, lev 0", "child1, fun_down, lev 1", "child1_1, fun_down, lev 1", "subchild1, fun_down, lev 2", "child1_2, fun_down, lev 1"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncDown error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	schild2.FuncUp(0, "fun_up", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("%v, %v", k.UniqueName(), d))
		return true
	})
	//	fmt.Printf("result: %v\n", res)

	trg = []string{"subchild1, fun_up", "child1_1, fun_up", "par1, fun_up"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncUp error -- results: %v != target: %v\n", res, trg)
	}
}

func TestNodeUpdate(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	res := make([]string, 0, 10)
	parent.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d interface{}) {
		res = append(res, fmt.Sprintf("%v sig %v flags %v", s.Name(), NodeSignals(sig),
			kit.BitFlagsToString(*(s.Flags()), FlagsN)))
	})
	// child1 :=
	parent.AddNewChild(nil, "child1")
	child2 := parent.AddNewChild(nil, "child1")
	// child3 :=
	updt := parent.UpdateStart()
	parent.AddNewChild(nil, "child1")
	parent.UpdateEnd(updt)
	schild2 := child2.AddNewChild(nil, "subchild1")

	// fmt.Printf("res: %v\n", res)
	trg := []string{"par1 sig NodeSignalUpdated flags ChildAdded", "par1 sig NodeSignalUpdated flags ChildAdded", "par1 sig NodeSignalUpdated flags ChildAdded"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("Add child sigs error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	child2.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d interface{}) {
		res = append(res, fmt.Sprintf("%v sig %v", s.Name(), NodeSignals(sig)))
	})
	schild2.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d interface{}) {
		res = append(res, fmt.Sprintf("%v sig %v", s.Name(), NodeSignals(sig)))
	})

	// fmt.Print("\nnode update top starting\n")
	updt = child2.UpdateStart()
	updt2 := schild2.UpdateStart()
	schild2.UpdateEnd(updt2)
	child2.UpdateEnd(updt)

	// fmt.Printf("res: %v\n", res)
	trg = []string{"child1 sig NodeSignalUpdated"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("update signal only top error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	parent.FuncDownMeFirst(0, "upcnt", func(n Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("%v %v", n.UniqueName(), n.IsUpdating()))
		return true
	})
	// fmt.Printf("res: %v\n", res)

	trg = []string{"par1 false", "child1 false", "child1_1 false", "subchild1 false", "child1_2 false"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("update counts error -- results: %v != target: %v\n", res, trg)
	}

}

func TestProps(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	res := make([]string, 0, 10)
	parent.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d interface{}) {
		res = append(res, fmt.Sprintf("%v sig %v", s.Name(), sig))
	})
	// child1 :=
	parent.AddNewChild(nil, "child1")
	child2 := parent.AddNewChild(nil, "child1")
	// child3 :=
	updt := parent.UpdateStart()
	parent.AddNewChild(nil, "child1")
	parent.UpdateEnd(updt)
	schild2 := child2.AddNewChild(nil, "subchild1")

	parent.SetProp("intprop", 42)
	pprop, ok := kit.ToInt(parent.Prop("intprop", false, false))
	if !ok || pprop != 42 {
		t.Errorf("TestProps error -- pprop %v != %v\n", pprop, 42)
	}
	sprop, ok := kit.ToInt(schild2.Prop("intprop", true, false))
	if !ok || sprop != 42 {
		t.Errorf("TestProps error -- sprop inherited %v != %v\n", sprop, 42)
	}
	sprop, ok = kit.ToInt(schild2.Prop("intprop", false, false))
	if ok || sprop != 0 {
		t.Errorf("TestProps error -- sprop not inherited %v != %v\n", sprop, 0)
	}

	parent.SetProp("floatprop", 42.0)
	spropf, ok := kit.ToFloat(schild2.Prop("floatprop", true, false))
	if !ok || spropf != 42.0 {
		t.Errorf("TestProps error -- spropf inherited %v != %v\n", spropf, 42.0)
	}

	tstr := "test string"
	parent.SetProp("stringprop", tstr)
	sprops := kit.ToString(schild2.Prop("stringprop", true, false))
	if sprops != tstr {
		t.Errorf("TestProps error -- sprops inherited %v != %v\n", sprops, tstr)
	}

	parent.DeleteProp("floatprop")
	spropf, ok = kit.ToFloat(schild2.Prop("floatprop", true, false))
	if ok || spropf != 0 {
		t.Errorf("TestProps error -- spropf inherited %v != %v\n", spropf, 0)
	}

	spropf, ok = kit.ToFloat(parent.Prop("floatprop", true, true))
	if !ok || spropf != 3.1415 {
		t.Errorf("TestProps error -- spropf from type %v != %v\n", spropf, 3.1415)
	}

}

func TestTreeMod(t *testing.T) {
	NodeSignalTrace = true
	sigs := ""
	NodeSignalTraceString = &sigs

	tree1 := Node{}
	tree1.InitName(&tree1, "tree1")
	// child11 :=
	tree1.AddNewChild(nil, "child11")
	child12 := tree1.AddNewChild(nil, "child12")
	// child13 :=
	tree1.AddNewChild(nil, "child13")
	// schild12 :=
	child12.AddNewChild(nil, "subchild12")

	tree2 := Node{}
	tree2.InitName(&tree2, "tree2")
	// child21 :=
	tree2.AddNewChild(nil, "child21")
	child22 := tree2.AddNewChild(nil, "child22")
	// child23 :=
	tree2.AddNewChild(nil, "child23")
	// schild22 :=
	child22.AddNewChild(nil, "subchild22")

	// fmt.Printf("Setup Signals:\n%v", sigs)
	sigs = ""

	// fmt.Printf("#################################\n")

	// fmt.Printf("Trees before:\n%v%v", tree1, tree2)
	tree2.AddChild(child12)

	// fmt.Printf("#################################\n")
	// fmt.Printf("Trees after add child12 move:\n%v%v", tree1, tree2)

	mvsigs := `ki.Signal Emit from: tree1 sig: NodeSignalUpdated data: 1024
ki.Signal Emit from: tree2 sig: NodeSignalUpdated data: 256
`
	// fmt.Printf("Move Signals:\n%v", sigs)
	if sigs != mvsigs {
		t.Errorf("TestTreeMod child12 move signals:\n%v\nnot as expected:\n%v\n", sigs, mvsigs)
	}
	sigs = ""

	updt := tree2.UpdateStart()
	tree2.DeleteChild(child12, true)
	tree2.UpdateEnd(updt)

	// fmt.Printf("#################################\n")

	delsigs := `ki.Signal Emit from: child12 sig: NodeSignalDeleting data: <nil>
ki.Signal Emit from: child12 sig: NodeSignalDestroying data: <nil>
ki.Signal Emit from: subchild12 sig: NodeSignalDeleting data: <nil>
ki.Signal Emit from: subchild12 sig: NodeSignalDestroying data: <nil>
ki.Signal Emit from: tree2 sig: NodeSignalUpdated data: 1024
`

	// fmt.Printf("Delete Signals:\n%v", sigs)
	if sigs != delsigs {
		t.Errorf("TestTreeMod child12 delete signals:\n%v\nnot as expected:\n%v\n", sigs, delsigs)
	}
	sigs = ""

}

func TestNodeFieldFunc(t *testing.T) {
	parent := NodeWithField{}
	parent.InitName(&parent, "par1")
	res := make([]string, 0, 10)
	parent.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("%v, %v, lev %v", k.UniqueName(), d, level))
		return true
	})
	// fmt.Printf("node field fun result: %v\n", res)

	trg := []string{"par1, fun_down, lev 0", "Field1, fun_down, lev 1"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeWithField FuncDown error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	par2 := NodeField2{}
	par2.InitName(&par2, "par2")
	par2.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("%v, %v, lev %v", k.UniqueName(), d, level))
		return true
	})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"par2, fun_down, lev 0", "Field1, fun_down, lev 1", "Field2, fun_down, lev 1"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeWithField FuncDown error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	par2.AddNewChild(nil, "child0")
	child1 := par2.AddNewChild(nil, "child1")
	child1.AddNewChild(nil, "subchild0")
	child1.AddNewChild(nil, "subchild1")

	par2.FuncDownDepthFirst(0, "fun_down_depth", func(k Ki, level int, d interface{}) bool {
		return true
	},
		func(k Ki, level int, d interface{}) bool {
			res = append(res, fmt.Sprintf("%v, %v, lev %v", k.UniqueName(), d, level))
			return true
		})
	// fmt.Printf("node field fun result: %v\n", res)
	// trg = []string{"par2, fun_down, lev 0", "Field1, fun_down, lev 1", "Field2, fun_down, lev 1"}
	// if !reflect.DeepEqual(res, trg) {
	// 	t.Errorf("NodeWithField FuncDown error -- results: %v != target: %v\n", res, trg)
	// }
	res = res[:0]
}

func TestNodeFieldJSonSave(t *testing.T) {
	parent := NodeField2{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(nil, "child1")
	child2 := parent.AddNewChild(nil, "child1").(*NodeField2)
	// child3 :=
	parent.AddNewChild(nil, "child1")
	schild2 := child2.AddNewChild(nil, "subchild1").(*NodeField2)

	parent.Ptr.Ptr = &child2.Field1
	child2.Ptr.Ptr = &schild2.Field2

	b, err := parent.SaveJSON(true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output: %v\n", string(b))
	}

	tstload := NodeField2{}
	tstload.InitName(&tstload, "")
	err = tstload.LoadJSON(b)
	if err != nil {
		t.Error(err)
	} else {
		tstb, _ := tstload.SaveJSON(true)
		// fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
			ioutil.WriteFile("/tmp/jsonout1", b, 0644)
			ioutil.WriteFile("/tmp/jsonout2", tstb, 0644)
		}
	}
}

func TestNodeFieldSet(t *testing.T) {
	parent := NodeField2{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(nil, "child1")
	child2 := parent.AddNewChild(nil, "child1").(*NodeField2)
	// child3 :=
	parent.AddNewChild(nil, "child1")
	schild2 := child2.AddNewChild(nil, "subchild1").(*NodeField2)

	parent.Ptr.Ptr = &child2.Field1
	child2.Ptr.Ptr = &schild2.Field2

	ts := "child2 is nice"
	ok := child2.SetField("Mbr1", ts)
	fs := kit.NonPtrInterface(child2.FieldByName("Mbr1"))
	if !ok || fs != ts {
		t.Errorf("Set field error: %+v != %+v, ok: %v\n", fs, ts, ok)
	}

	ts = "45.21"
	ok = child2.SetField("Mbr1", 45.21)
	fs = kit.NonPtrInterface(child2.FieldByName("Mbr1"))
	if !ok || fs != ts {
		t.Errorf("Set field error: %+v != %+v, ok: %v\n", fs, ts, ok)
	}
}

func TestClone(t *testing.T) {
	parent := NodeField2{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(nil, "child1")
	child2 := parent.AddNewChild(nil, "child1").(*NodeField2)
	// child3 :=
	parent.AddNewChild(nil, "child1")
	schild2 := child2.AddNewChild(nil, "subchild1").(*NodeField2)

	parent.Ptr.Ptr = &child2.Field1
	child2.Ptr.Ptr = &schild2.Field2

	b, err := parent.SaveJSON(true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output: %v\n", string(b))
	}

	tstload := parent.Clone()
	tstb, _ := tstload.SaveJSON(true)
	// fmt.Printf("test loaded json output: %v\n", string(tstb))
	if !bytes.Equal(tstb, b) {
		t.Error("original and unmarshal'd json rep are not equivalent")
		ioutil.WriteFile("/tmp/jsonout1", b, 0644)
		ioutil.WriteFile("/tmp/jsonout2", tstb, 0644)
	}
}
