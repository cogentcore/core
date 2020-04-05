// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/goki/ki/kit"
	"github.com/goki/prof"
)

type NodeEmbed struct {
	Node
	Mbr1 string
	Mbr2 int
}

var NodeEmbedProps = Props{
	"intprop":    -17,
	"floatprop":  3.1415,
	"stringprop": "type string",
}

var KiT_NodeEmbed = kit.Types.AddType(&NodeEmbed{}, NodeEmbedProps)

type NodeField struct {
	NodeEmbed
	Field1 NodeEmbed
}

var KiT_NodeField = kit.Types.AddType(&NodeField{}, nil)

type NodeField2 struct {
	NodeField
	Field2    NodeEmbed
	PtrIgnore *NodeEmbed
}

var KiT_NodeField2 = kit.Types.AddType(&NodeField2{}, nil)

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
	if reflect.TypeOf(child).Elem() != parent.Type() {
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
	if pth := child2.PathUnique(); pth != "/par1/child1_001" {
		t.Errorf("child2 path != correct, was %v", pth)
	}
	if pth := child3.PathUnique(); pth != "/par1/child1_002" {
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
		for st := range names { // test all starting indexes
			idx, ok := parent.Children().IndexByName(nm, st)
			if !ok || idx != i {
				t.Errorf("find index was not correct val of %d, was %d", i, idx)
			}
		}
	}
}

func TestNodeFindNameUnique(t *testing.T) {
	names := [...]string{"child", "child_001", "child_002", "child_003", "child_004", "child_005"}
	parent := Node{}
	parent.InitName(&parent, "par")
	for range names {
		parent.AddNewChild(nil, "child")
	}
	if len(parent.Kids) != len(names) {
		t.Errorf("Children length != n, was %d", len(parent.Kids))
	}
	for i, nm := range names {
		for st := range names { // test all starting indexes
			idx, ok := parent.Children().IndexByUniqueName(nm, st)
			if !ok || idx != i {
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
	idx, ok := parent.Children().IndexByType(KiT_NodeEmbed, NoEmbeds, 0)
	if !ok || idx != 0 {
		t.Errorf("find index was not correct val of %d, was %d", 0, idx)
	}
	idx, ok = parent.Children().IndexByType(KiT_Node, NoEmbeds, 0)
	if !ok || idx != 1 {
		t.Errorf("find index was not correct val of %d, was %d", 1, idx)
	}
	_, err := parent.Children().ElemByTypeTry(KiT_Node, NoEmbeds, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestNodeMove(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(nil, "child0")
	var child2 = parent.AddNewChild(nil, "child1").(*NodeEmbed)
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
[/par1/child0 /par1/child1 /par1/child2 /par1/child3]
`
	if bf != bft {
		t.Errorf("move error\n%v !=\n%v", bf, bft)
	}
	a31t := `mv 3 -> 1:
[/par1/child0 /par1/child3 /par1/child1 /par1/child2]
`
	if a31 != a31t {
		t.Errorf("move error\n%v !=\n%v", a31, a31t)
	}
	a03t := `mv 0 -> 3:
[/par1/child3 /par1/child1 /par1/child2 /par1/child0]
`
	if a03 != a03t {
		t.Errorf("move error\n%v !=\n%v", a03, a03t)
	}
	a12t := `mv 1 -> 2:
[/par1/child3 /par1/child2 /par1/child1 /par1/child0]
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
	var child2 = parent.AddNewChild(nil, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(nil, "child2")
	//schild2 :=
	child2.AddNewChild(nil, "subchild1")
	// child4 :=
	parent.AddNewChild(nil, "child3")

	config1 := kit.TypeAndNameList{
		{Type: KiT_NodeEmbed, Name: "child2"},
		{Type: KiT_NodeEmbed, Name: "child3"},
		{Type: KiT_NodeEmbed, Name: "child1"},
	}

	// bf := fmt.Sprintf("mv before:\n%v\n", parent.Kids)

	mods, updt := parent.ConfigChildren(config1, NonUniqueNames)
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
	netn := kit.Types.TypeName(KiT_NodeEmbed)
	ntn := kit.Types.TypeName(KiT_Node)
	err := config3.SetFromString("{" + netn + ", child4}, {" + ntn + ", child1}, {" + netn + ", child5}, {" + netn + ", child3}, {" + netn + ", child6}")
	if err != nil {
		t.Errorf("%v", err)
	}

	mods, updt = parent.ConfigChildren(config3, NonUniqueNames)
	if mods {
		parent.UpdateEnd(updt)
	}

	cf2 := fmt.Sprintf("config2:\n%v\n", parent.Kids)

	cf1t := `config1:
[/par1/child2 /par1/child3 /par1/child1]
`
	if cf1 != cf1t {
		t.Errorf("config error\n%v !=\n%v", cf1, cf1t)
	}

	cf2t := `config2:
[/par1/child4 /par1/child1 /par1/child5 /par1/child3 /par1/child6]
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
	var child2 = parent.AddNewChild(nil, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(nil, "child1")
	child2.AddNewChild(nil, "subchild1")

	var buf bytes.Buffer
	err := parent.WriteJSON(&buf, true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output:\n%v\n", string(buf.Bytes()))
	}
	b := buf.Bytes()

	tstload := NodeEmbed{}
	tstload.InitName(&tstload, "")
	err = tstload.ReadJSON(bytes.NewReader(b))
	if err != nil {
		t.Error(err)
	} else {
		var buf2 bytes.Buffer
		err = tstload.WriteJSON(&buf2, true)
		if err != nil {
			t.Error(err)
		}
		tstb := buf2.Bytes()
		// fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
		}
	}

	nwnd, err := ReadNewJSON(bytes.NewReader(b))
	if err != nil {
		t.Error(err)
	} else {
		var buf2 bytes.Buffer
		err = nwnd.WriteJSON(&buf2, true)
		if err != nil {
			t.Error(err)
		}
		tstb := buf2.Bytes()
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
	var child2 = parent.AddNewChild(nil, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(nil, "child1")
	child2.AddNewChild(nil, "subchild1")

	var buf bytes.Buffer
	err := parent.WriteXML(&buf, true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("xml output:\n%v\n", string(buf.Bytes()))
	}
	b := buf.Bytes()

	tstload := NodeEmbed{}
	tstload.InitName(&tstload, "")
	err = tstload.ReadXML(bytes.NewReader(b))
	if err != nil {
		t.Error(err)
	} else {
		var buf2 bytes.Buffer
		if err != nil {
			t.Error(err)
		}
		err := tstload.WriteXML(&buf2, true)
		if err != nil {
			t.Error(err)
		}
		tstb := buf2.Bytes()
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
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.UniqueName(), d, level))
		return true
	})

	trg := []string{"[par1, fun_down, lev 0]", "[child1, fun_down, lev 1]", "[child1_001, fun_down, lev 1]", "[subchild1, fun_down, lev 2]", "[child1_002, fun_down, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncDown error -- results:\n%v\n != target:\n%v\n", res, trg)
	}
	res = res[:0]

	// test return = false case
	parent.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.UniqueName(), d, level))
		if k.UniqueName() == "child1_001" {
			return false
		}
		return true
	})

	trg = []string{"[par1, fun_down, lev 0]", "[child1, fun_down, lev 1]", "[child1_001, fun_down, lev 1]", "[child1_002, fun_down, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncDown return false error -- results:\n%v\n != target:\n%v\n", res, trg)
	}
	res = res[:0]

	schild2.FuncUp(0, "fun_up", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("%v, %v", k.UniqueName(), d))
		return true
	})
	//	fmt.Printf("result: %v\n", res)

	trg = []string{"subchild1, fun_up", "child1_001, fun_up", "par1, fun_up"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncUp error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	parent.FuncDownMeLast(0, "fun_down_me_last", func(k Ki, level int, d interface{}) bool {
		return true
	},
		func(k Ki, level int, d interface{}) bool {
			res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.UniqueName(), d, level))
			return true
		})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[child1, fun_down_me_last, lev 1]", "[subchild1, fun_down_me_last, lev 2]", "[child1_001, fun_down_me_last, lev 1]", "[child1_002, fun_down_me_last, lev 1]", "[par1, fun_down_me_last, lev 0]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDownDepthFirst error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]

	parent.FuncDownBreadthFirst(0, "fun_breadth", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.UniqueName(), d, level))
		return true
	})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[par1, fun_breadth, lev 0]", "[child1, fun_breadth, lev 1]", "[child1_001, fun_breadth, lev 1]", "[child1_002, fun_breadth, lev 1]", "[subchild1, fun_breadth, lev 2]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDownBreadthFirst error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]
}

func TestNodeUpdate(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	res := make([]string, 0, 10)
	parent.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d interface{}) {
		res = append(res, fmt.Sprintf("%v sig %v flags %v", s.Name(), NodeSignals(sig),
			kit.BitFlagsToString((s.Flags()), FlagsN)))
	})
	// child1 :=
	parent.AddNewChild(nil, "child1")
	child2 := parent.AddNewChild(nil, "child1")
	// child3 :=
	updt := parent.UpdateStart()
	parent.AddNewChild(nil, "child1")
	parent.UpdateEnd(updt)
	schild2 := child2.AddNewChild(nil, "subchild1")

	for ri := range res {
		res[ri] = strings.Replace(res[ri], "HasNoKiFields|", "", -1)
	}
	// fmt.Printf("res: %v\n", res)
	trg := []string{"par1 sig NodeSignalUpdated flags ChildAdded", "par1 sig NodeSignalUpdated flags ChildAdded", "par1 sig NodeSignalUpdated flags ChildAdded"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("Add child sigs error -- results:\n%v\n!= target:\n%v\n", res, trg)
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

	trg = []string{"par1 false", "child1 false", "child1_001 false", "subchild1 false", "child1_002 false"}
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
	pprop, ok := kit.ToInt(parent.Prop("intprop"))
	if !ok || pprop != 42 {
		t.Errorf("TestProps error -- pprop %v != %v\n", pprop, 42)
	}
	sprop, ok := schild2.PropInherit("intprop", Inherit, NoTypeProps)
	if !ok {
		t.Errorf("TestProps error -- intprop inherited not found\n")
	}
	sint, ok := kit.ToInt(sprop)
	if !ok || sprop != 42 {
		t.Errorf("TestProps error -- intprop inherited %v != %v\n", sint, 42)
	}
	sprop, ok = schild2.PropInherit("intprop", NoInherit, NoTypeProps)
	if ok {
		t.Errorf("TestProps error -- intprop should not be found!  was: %v\n", sprop)
	}

	parent.SetProp("floatprop", 42.0)
	sprop, ok = schild2.PropInherit("floatprop", Inherit, NoTypeProps)
	if !ok {
		t.Errorf("TestProps error -- floatprop inherited not found\n")
	}
	spropf, ok := kit.ToFloat(sprop)
	if !ok || spropf != 42.0 {
		t.Errorf("TestProps error -- floatprop inherited %v != %v\n", spropf, 42.0)
	}

	tstr := "test string"
	parent.SetProp("stringprop", tstr)
	sprop, ok = schild2.PropInherit("stringprop", Inherit, NoTypeProps)
	if !ok {
		t.Errorf("TestProps error -- stringprop not found\n")
	}
	sprops := kit.ToString(sprop)
	if sprops != tstr {
		t.Errorf("TestProps error -- sprops inherited %v != %v\n", sprops, tstr)
	}

	parent.DeleteProp("floatprop")
	sprop, ok = schild2.PropInherit("floatprop", Inherit, NoTypeProps)
	if ok {
		t.Errorf("TestProps error -- floatprop should be gone\n")
	}

	sprop, ok = parent.PropInherit("floatprop", Inherit, TypeProps)
	if !ok {
		t.Errorf("TestProps error -- floatprop on type not found\n")
	}
	spropf, ok = kit.ToFloat(sprop)
	if !ok || spropf != 3.1415 {
		t.Errorf("TestProps error -- floatprop from type %v != %v\n", spropf, 3.1415)
	}
}

func TestTreeMod(t *testing.T) {
	SignalTrace = true
	sigs := ""
	SignalTraceString = &sigs

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

	mvsigs := `ki.Signal Emit from: tree1 sig: NodeSignalUpdated data: 4100
ki.Signal Emit from: tree2 sig: NodeSignalUpdated data: 1028
`

	_ = mvsigs
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
ki.Signal Emit from: subchild12 sig: NodeSignalDeleting data: <nil>
ki.Signal Emit from: subchild12 sig: NodeSignalUpdated data: 8196
ki.Signal Emit from: child12 sig: NodeSignalUpdated data: 8196
ki.Signal Emit from: tree2 sig: NodeSignalUpdated data: 4100
`

	_ = delsigs
	// fmt.Printf("Delete Signals:\n%v", sigs)
	if sigs != delsigs {
		t.Errorf("TestTreeMod child12 delete signals:\n%v\nnot as expected:\n%v\n", sigs, delsigs)
	}
	sigs = ""

}

func TestNodeFieldFunc(t *testing.T) {
	parent := NodeField{}
	parent.InitName(&parent, "par1")
	res := make([]string, 0, 10)
	parent.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.UniqueName(), d, level))
		return true
	})
	// fmt.Printf("node field fun result: %v\n", res)

	trg := []string{"[par1, fun_down, lev 0]", "[Field1, fun_down, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDown error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	par2 := NodeField2{}
	par2.InitName(&par2, "par2")
	par2.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d interface{}) bool {
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.UniqueName(), d, level))
		return true
	})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[par2, fun_down, lev 0]", "[Field1, fun_down, lev 1]", "[Field2, fun_down, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDown error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	par2.FuncDownMeLast(0, "fun_down_me_last", func(k Ki, level int, d interface{}) bool {
		return true
	},
		func(k Ki, level int, d interface{}) bool {
			res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.UniqueName(), d, level))
			return true
		})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[Field1, fun_down_me_last, lev 1]", "[Field2, fun_down_me_last, lev 1]", "[par2, fun_down_me_last, lev 0]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDownDepthFirst error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
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
	child2.AddNewChild(nil, "subchild1")

	var buf bytes.Buffer
	err := parent.WriteJSON(&buf, true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output:\n%v\n", string(buf.Bytes()))
	}
	b := buf.Bytes()

	tstload := NodeField2{}
	tstload.InitName(&tstload, "")
	err = tstload.ReadJSON(bytes.NewReader(b))
	if err != nil {
		t.Error(err)
	} else {
		var buf2 bytes.Buffer
		err = tstload.WriteJSON(&buf2, true)
		if err != nil {
			t.Error(err)
		}
		tstb := buf2.Bytes()
		// fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
			ioutil.WriteFile("/tmp/jsonout1", b, 0644)
			ioutil.WriteFile("/tmp/jsonout2", tstb, 0644)
		}
	}

	nwnd, err := ReadNewJSON(bytes.NewReader(b))
	if err != nil {
		t.Error(err)
	} else {
		var buf2 bytes.Buffer
		err = nwnd.WriteJSON(&buf2, true)
		if err != nil {
			t.Error(err)
		}
		tstb := buf2.Bytes()
		// fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
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
	child2.AddNewChild(nil, "subchild1")

	ts := "child2 is nice"
	err := child2.SetField("Mbr1", ts)
	if err != nil {
		t.Error(err)
	}
	fs := kit.NonPtrInterface(child2.FieldByName("Mbr1"))
	if fs != ts {
		t.Errorf("Set field error: %+v != %+v\n", fs, ts)
	}

	ts = "45.21"
	err = child2.SetField("Mbr1", 45.21)
	if err != nil {
		t.Error(err)
	}
	fs = kit.NonPtrInterface(child2.FieldByName("Mbr1"))
	if fs != ts {
		t.Errorf("Set field error: %+v != %+v\n", fs, ts)
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
	child2.AddNewChild(nil, "subchild1")

	var buf bytes.Buffer
	err := parent.WriteJSON(&buf, true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output:\n%v\n", string(buf.Bytes()))
	}
	b := buf.Bytes()

	tstload := parent.Clone()
	var buf2 bytes.Buffer
	err = tstload.WriteJSON(&buf2, true)
	if err != nil {
		t.Error(err)
	}
	tstb := buf2.Bytes()
	// fmt.Printf("test loaded json output: %v\n", string(tstb))
	if !bytes.Equal(tstb, b) {
		t.Error("original and unmarshal'd json rep are not equivalent")
		ioutil.WriteFile("/tmp/jsonout1", b, 0644)
		ioutil.WriteFile("/tmp/jsonout2", tstb, 0644)
	}
}

// BuildGuiTreeSlow builds a tree that is typical of GUI structures where there are
// many widgets in a container and each widget has some number of parts.
// Uses slow AddChild method instead of fast one.
func BuildGuiTreeSlow(widgets, parts int, typ reflect.Type) Ki {
	win := NewOfType(typ)
	win.InitName(win, "window")
	updt := win.UpdateStart()

	vp := win.AddNewChild(typ, "vp")
	frame := vp.AddNewChild(typ, "frame")
	for wi := 0; wi < widgets; wi++ {
		widg := frame.AddNewChild(typ, fmt.Sprintf("widg_%d", wi))

		for pi := 0; pi < parts; pi++ {
			widg.AddNewChild(typ, fmt.Sprintf("part_%d", pi))
		}
	}
	win.UpdateEnd(updt)
	return win
}

// BuildGuiTree builds a tree that is typical of GUI structures where there are
// many widgets in a container and each widget has some number of parts.
// Uses slow AddChild method instead of fast one.
func BuildGuiTree(widgets, parts int, typ reflect.Type) Ki {
	win := NewOfType(typ)
	win.InitName(win, "window")
	updt := win.UpdateStart()

	vp := win.AddNewChildFast(typ, "vp")
	frame := vp.AddNewChildFast(typ, "frame")
	for wi := 0; wi < widgets; wi++ {
		widg := frame.AddNewChildFast(typ, fmt.Sprintf("widg_%d", wi))

		for pi := 0; pi < parts; pi++ {
			widg.AddNewChildFast(typ, fmt.Sprintf("part_%d", pi))
		}
	}
	win.UpdateEnd(updt)
	return win
}

var TotNodes int
var TestGUITree_NodeEmbed Ki
var TestGUITree_NodeField Ki
var TestGUITree_NodeField2 Ki

var NWidgets = 10000
var NParts = 5

func BenchmarkBuildGuiTree_NodeEmbed(b *testing.B) {
	prof.Reset()
	prof.Profiling = true
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTree(NWidgets, NParts, KiT_NodeEmbed)
		TestGUITree_NodeEmbed = wt
	}
	prof.Report(time.Millisecond)
	prof.Profiling = false
}

func BenchmarkBuildGuiTree_NodeField(b *testing.B) {
	prof.Reset()
	prof.Profiling = true
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTree(NWidgets, NParts, KiT_NodeField)
		TestGUITree_NodeField = wt
	}
	prof.Report(time.Millisecond)
	prof.Profiling = false
}

func BenchmarkBuildGuiTree_NodeField2(b *testing.B) {
	prof.Reset()
	prof.Profiling = true
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTree(NWidgets, NParts, KiT_NodeField2)
		TestGUITree_NodeField2 = wt
	}
	prof.Report(time.Millisecond)
	prof.Profiling = false
}

func BenchmarkBuildGuiTreeSlow_NodeEmbed(b *testing.B) {
	prof.Reset()
	prof.Profiling = true
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTreeSlow(NWidgets, NParts, KiT_NodeEmbed)
		TestGUITree_NodeEmbed = wt
	}
	prof.Report(time.Millisecond)
	prof.Profiling = false
}

func BenchmarkFuncDownMeFirst_NodeEmbed(b *testing.B) {
	wt := TestGUITree_NodeEmbed
	nnodes := 0
	for n := 0; n < b.N; n++ {
		nnodes = 0
		wt.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlag(int(Updating))
			return true
		})
	}
	TotNodes = nnodes
	fmt.Printf("tot nodes: %d\n", TotNodes)
}

func BenchmarkFuncDownMeFirst_NodeField(b *testing.B) {
	wt := TestGUITree_NodeField
	nnodes := 0
	for n := 0; n < b.N; n++ {
		nnodes = 0
		wt.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlag(int(Updating))
			return true
		})
	}
	TotNodes = nnodes
	fmt.Printf("tot nodes: %d\n", TotNodes)
}

func BenchmarkFuncDownMeFirst_NodeField2(b *testing.B) {
	wt := TestGUITree_NodeField2
	nnodes := 0
	for n := 0; n < b.N; n++ {
		nnodes = 0
		wt.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlag(int(Updating))
			return true
		})
	}
	TotNodes = nnodes
	fmt.Printf("tot nodes: %d\n", TotNodes)
}
