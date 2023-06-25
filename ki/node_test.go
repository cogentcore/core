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

	"github.com/goki/ki/kit"
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
	typ := reflect.TypeOf(parent)
	child := parent.AddNewChild(typ, "child1")
	if len(parent.Kids) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.Kids))
	}
	if child.Path() != "/par1/child1" {
		t.Errorf("child path != correct, was %v", child.Path())
	}
	if reflect.TypeOf(child).Elem() != Type(parent.This()) {
		t.Errorf("child type != correct, was %T", child)
	}
}

func TestNodeUniqueNames(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := reflect.TypeOf(parent)
	child := parent.AddNewChild(typ, "child1")
	child2 := parent.AddNewChild(typ, "child1")
	child3 := parent.AddNewChild(typ, "child1")
	if len(parent.Kids) != 3 {
		t.Errorf("Children length != 3, was %d", len(parent.Kids))
	}
	UniquifyNamesAll(parent.This())
	if pth := child.Path(); pth != "/par1/child1" {
		t.Errorf("child path != correct, was %v", pth)
	}
	if pth := child2.Path(); pth != "/par1/child1_001" {
		t.Errorf("child2 path != correct, was %v", pth)
	}
	if pth := child3.Path(); pth != "/par1/child1_002" {
		t.Errorf("child3 path != correct, was %v", pth)
	}

}

func TestNodeEscapePaths(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := reflect.TypeOf(parent)
	child := parent.AddNewChild(typ, "child1.go")
	child2 := parent.AddNewChild(typ, "child1/child1")
	child3 := parent.AddNewChild(typ, "child1/child1.go")
	schild2 := child2.AddNewChild(typ, "subchild1")
	if len(parent.Kids) != 3 {
		t.Errorf("Children length != 3, was %d", len(parent.Kids))
	}
	if pth := child.Path(); pth != `/par1/child1\,go` {
		t.Errorf("child path != correct, was %v", pth)
	}
	if pth := child2.Path(); pth != `/par1/child1\\child1` {
		t.Errorf("child2 path != correct, was %v", pth)
	}
	if pth := child3.Path(); pth != `/par1/child1\\child1\,go` {
		t.Errorf("child3 path != correct, was %v", pth)
	}
	ch := parent.FindPath(child.Path())
	if ch != child {
		t.Errorf("child path not found in parent")
	}
	ch = parent.FindPath(child3.Path())
	if ch != child3 {
		t.Errorf("child3 path not found in parent")
	}
	ch = parent.FindPath(child3.Path())
	if ch != child3 {
		t.Errorf("child3 path not found in parent")
	}
	ch = parent.FindPath(schild2.Path())
	if ch != schild2 {
		t.Errorf("schild2 path not found in parent")
	}
	ch = child2.FindPath(schild2.Path())
	if ch != schild2 {
		t.Errorf("schild2 path not found in child2")
	}
}

func TestNodeDeleteChild(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := reflect.TypeOf(parent)
	child := parent.AddNewChild(typ, "child1")
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
	typ := reflect.TypeOf(parent)
	parent.AddNewChild(typ, "child1")
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
	typ := reflect.TypeOf(parent)
	for _, nm := range names {
		parent.AddNewChild(typ, nm)
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
	typ := reflect.TypeOf(parent)
	for range names {
		parent.AddNewChild(typ, "child")
	}
	if len(parent.Kids) != len(names) {
		t.Errorf("Children length != n, was %d", len(parent.Kids))
	}
	if UniqueNameCheckAll(parent.This()) {
		t.Errorf("UniqeNameCheckAll failed: Children are not unique!")
	}
	UniquifyNamesAll(parent.This())
	for i, nm := range names {
		for st := range names { // test all starting indexes
			idx, ok := parent.Children().IndexByName(nm, st)
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
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(typ, "child0")
	var child2 = parent.AddNewChild(typ, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(typ, "child2")
	//schild2 :=
	child2.AddNewChild(typ, "subchild1")
	// child4 :=
	parent.AddNewChild(typ, "child3")

	bf := fmt.Sprintf("mv before:\n%v\n", parent.Kids)
	parent.Children().Move(3, 1)
	a31 := fmt.Sprintf("mv 3 -> 1:\n%v\n", parent.Kids)
	parent.Children().Move(0, 3)
	a03 := fmt.Sprintf("mv 0 -> 3:\n%v\n", parent.Kids)
	parent.Children().Move(1, 2)
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
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(typ, "child0")
	var child2 = parent.AddNewChild(typ, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(typ, "child2")
	//schild2 :=
	child2.AddNewChild(typ, "subchild1")
	// child4 :=
	parent.AddNewChild(typ, "child3")

	config1 := kit.TypeAndNameList{
		{Type: KiT_NodeEmbed, Name: "child2"},
		{Type: KiT_NodeEmbed, Name: "child3"},
		{Type: KiT_NodeEmbed, Name: "child1"},
	}

	// bf := fmt.Sprintf("mv before:\n%v\n", parent.Kids)

	mods, updt := parent.ConfigChildren(config1)
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

	mods, updt = parent.ConfigChildren(config3)
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

func TestNodeJSONSave(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(typ, "child1")
	var child2 = parent.AddNewChild(typ, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(typ, "child1")
	child2.AddNewChild(typ, "subchild1")

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
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(typ, "child1")
	var child2 = parent.AddNewChild(typ, "child1").(*NodeEmbed)
	// child3 :=
	parent.AddNewChild(typ, "child1")
	child2.AddNewChild(typ, "subchild1")

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
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(typ, "child1")
	child2 := parent.AddNewChild(typ, "child1")
	// child3 :=
	parent.AddNewChild(typ, "child1")
	schild2 := child2.AddNewChild(typ, "subchild1")
	UniquifyNames(parent.This())

	res := make([]string, 0, 10)
	parent.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d any) bool {
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.Name(), d, level))
		return true
	})

	trg := []string{"[par1, fun_down, lev 0]", "[child1, fun_down, lev 1]", "[child1_001, fun_down, lev 1]", "[subchild1, fun_down, lev 2]", "[child1_002, fun_down, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncDown error -- results:\n%v\n != target:\n%v\n", res, trg)
	}
	res = res[:0]

	// test return = false case
	parent.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d any) bool {
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.Name(), d, level))
		if k.Name() == "child1_001" {
			return Break
		}
		return Continue
	})

	trg = []string{"[par1, fun_down, lev 0]", "[child1, fun_down, lev 1]", "[child1_001, fun_down, lev 1]", "[child1_002, fun_down, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncDown return false error -- results:\n%v\n != target:\n%v\n", res, trg)
	}
	res = res[:0]

	schild2.FuncUp(0, "fun_up", func(k Ki, level int, d any) bool {
		res = append(res, fmt.Sprintf("%v, %v", k.Name(), d))
		return Continue
	})
	//	fmt.Printf("result: %v\n", res)

	trg = []string{"subchild1, fun_up", "child1_001, fun_up", "par1, fun_up"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncUp error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	parent.FuncDownMeLast(0, "fun_down_me_last", func(k Ki, level int, d any) bool {
		return Continue
	},
		func(k Ki, level int, d any) bool {
			res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.Name(), d, level))
			return Continue
		})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[child1, fun_down_me_last, lev 1]", "[subchild1, fun_down_me_last, lev 2]", "[child1_001, fun_down_me_last, lev 1]", "[child1_002, fun_down_me_last, lev 1]", "[par1, fun_down_me_last, lev 0]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDownMeLast error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]

	// test for return = false working
	parent.FuncDownMeLast(0, "fun_down_me_last", func(k Ki, level int, d any) bool {
		if k.Name() == "child1_001" {
			return Break
		}
		return Continue
	},
		func(k Ki, level int, d any) bool {
			if k.Name() == "child1_001" {
				return Break
			}
			res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.Name(), d, level))
			return Continue
		})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[child1, fun_down_me_last, lev 1]", "[child1_002, fun_down_me_last, lev 1]", "[par1, fun_down_me_last, lev 0]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDownMeLast error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]

	parent.FuncDownBreadthFirst(0, "fun_breadth", func(k Ki, level int, d any) bool {
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.Name(), d, level))
		return Continue
	})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[par1, fun_breadth, lev 0]", "[child1, fun_breadth, lev 1]", "[child1_001, fun_breadth, lev 1]", "[child1_002, fun_breadth, lev 1]", "[subchild1, fun_breadth, lev 2]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDownBreadthFirst error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]

	// test for return false
	parent.FuncDownBreadthFirst(0, "fun_breadth", func(k Ki, level int, d any) bool {
		if k.Name() == "child1_001" {
			return Break
		}
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.Name(), d, level))
		return Continue
	})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[par1, fun_breadth, lev 0]", "[child1, fun_breadth, lev 1]", "[child1_002, fun_breadth, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDownBreadthFirst error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]
}

func TestNodeUpdate(t *testing.T) {
	parent := NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	res := make([]string, 0, 10)
	parent.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d any) {
		res = append(res, fmt.Sprintf("%v sig %v flags %v", s.Name(), NodeSignals(sig),
			kit.BitFlagsToString((s.Flags()), FlagsN)))
	})
	// child1 :=
	updt := parent.UpdateStart()
	parent.SetChildAdded()
	parent.AddNewChild(typ, "child1")
	child2 := parent.AddNewChild(typ, "child1")
	// child3 :=
	parent.UpdateEnd(updt)
	updt = parent.UpdateStart()
	parent.SetChildAdded()
	parent.AddNewChild(typ, "child1")
	parent.UpdateEnd(updt)
	schild2 := child2.AddNewChild(typ, "subchild1")
	child2.SetChildAdded()
	parent.UpdateEnd(updt)

	for ri := range res {
		res[ri] = strings.Replace(res[ri], "HasNoKiFields|", "", -1)
	}
	// fmt.Printf("res: %v\n", res)
	trg := []string{"par1 sig NodeSignalUpdated flags ChildAdded", "par1 sig NodeSignalUpdated flags ChildAdded", "par1 sig NodeSignalUpdated flags ChildAdded"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("Add child sigs error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]

	child2.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d any) {
		res = append(res, fmt.Sprintf("%v sig %v", s.Name(), NodeSignals(sig)))
	})
	schild2.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d any) {
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

	UniquifyNamesAll(parent.This())

	parent.FuncDownMeFirst(0, "upcnt", func(n Ki, level int, d any) bool {
		res = append(res, fmt.Sprintf("%v %v", n.Name(), n.IsUpdating()))
		return Continue
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
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	res := make([]string, 0, 10)
	parent.NodeSignal().Connect(&parent, func(r, s Ki, sig int64, d any) {
		res = append(res, fmt.Sprintf("%v sig %v", s.Name(), sig))
	})
	// child1 :=
	parent.AddNewChild(typ, "child1")
	child2 := parent.AddNewChild(typ, "child1")
	// child3 :=
	updt := parent.UpdateStart()
	parent.AddNewChild(typ, "child1")
	parent.UpdateEnd(updt)
	schild2 := child2.AddNewChild(typ, "subchild1")

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
	typ := reflect.TypeOf(tree1)
	tree1.InitName(&tree1, "tree1")
	// child11 :=
	tree1.AddNewChild(typ, "child11")
	child12 := tree1.AddNewChild(typ, "child12")
	// child13 :=
	tree1.AddNewChild(typ, "child13")
	// schild12 :=
	child12.AddNewChild(typ, "subchild12")

	tree2 := Node{}
	tree2.InitName(&tree2, "tree2")
	// child21 :=
	tree2.AddNewChild(typ, "child21")
	child22 := tree2.AddNewChild(typ, "child22")
	// child23 :=
	tree2.AddNewChild(typ, "child23")
	// schild22 :=
	child22.AddNewChild(typ, "subchild22")

	// fmt.Printf("Setup Signals:\n%v", sigs)
	sigs = ""

	// fmt.Printf("#################################\n")

	// fmt.Printf("Trees before:\n%v%v", tree1, tree2)
	updt := tree2.UpdateStart()
	tree2.SetChildAdded()
	MoveToParent(child12.This(), tree2.This())
	tree2.UpdateEnd(updt)

	// fmt.Printf("#################################\n")
	// fmt.Printf("Trees after add child12 move:\n%v%v", tree1, tree2)

	mvsigs := `ki.Signal Emit from: tree1 sig: NodeSignalUpdated data: 260
ki.Signal Emit from: tree2 sig: NodeSignalUpdated data: 132
`

	_ = mvsigs
	// fmt.Printf("Move Signals:\n%v", sigs)
	if sigs != mvsigs {
		t.Errorf("TestTreeMod child12 move signals:\n%v\nnot as expected:\n%v\n", sigs, mvsigs)
	}
	sigs = ""

	updt = tree2.UpdateStart()
	tree2.DeleteChild(child12, true)
	tree2.UpdateEnd(updt)

	// fmt.Printf("#################################\n")

	delsigs := `ki.Signal Emit from: child12 sig: NodeSignalDeleting data: <nil>
ki.Signal Emit from: subchild12 sig: NodeSignalDeleting data: <nil>
ki.Signal Emit from: tree2 sig: NodeSignalUpdated data: 260
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
	parent.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d any) bool {
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.Name(), d, level))
		return Continue
	})
	// fmt.Printf("node field fun result: %v\n", res)

	trg := []string{"[par1, fun_down, lev 0]", "[Field1, fun_down, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDown error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	par2 := NodeField2{}
	par2.InitName(&par2, "par2")
	par2.FuncDownMeFirst(0, "fun_down", func(k Ki, level int, d any) bool {
		res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.Name(), d, level))
		return Continue
	})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[par2, fun_down, lev 0]", "[Field1, fun_down, lev 1]", "[Field2, fun_down, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDown error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	par2.FuncDownMeLast(0, "fun_down_me_last", func(k Ki, level int, d any) bool {
		return Continue
	},
		func(k Ki, level int, d any) bool {
			res = append(res, fmt.Sprintf("[%v, %v, lev %v]", k.Name(), d, level))
			return Continue
		})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[Field1, fun_down_me_last, lev 1]", "[Field2, fun_down_me_last, lev 1]", "[par2, fun_down_me_last, lev 0]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField FuncDownMeLast error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]
}

func TestNodeFieldJSONSave(t *testing.T) {
	parent := NodeField2{}
	parent.InitName(&parent, "par1")
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(typ, "child1")
	child2 := parent.AddNewChild(typ, "child1").(*NodeField2)
	// child3 :=
	parent.AddNewChild(typ, "child1")
	child2.AddNewChild(typ, "subchild1")

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
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(typ, "child1")
	child2 := parent.AddNewChild(typ, "child1").(*NodeField2)
	// child3 :=
	parent.AddNewChild(typ, "child1")
	child2.AddNewChild(typ, "subchild1")

	ts := "child2 is nice"
	err := child2.SetField("Mbr1", ts)
	if err != nil {
		t.Error(err)
	}
	fs := kit.NonPtrInterface(FieldByName(child2.This(), "Mbr1"))
	if fs != ts {
		t.Errorf("Set field error: %+v != %+v\n", fs, ts)
	}

	ts = "45.21"
	err = child2.SetField("Mbr1", 45.21)
	if err != nil {
		t.Error(err)
	}
	fs = kit.NonPtrInterface(FieldByName(child2.This(), "Mbr1"))
	if fs != ts {
		t.Errorf("Set field error: %+v != %+v\n", fs, ts)
	}
}

func TestClone(t *testing.T) {
	parent := NodeField2{}
	parent.InitName(&parent, "par1")
	typ := reflect.TypeOf(parent)
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.AddNewChild(typ, "child1")
	child2 := parent.AddNewChild(typ, "child1").(*NodeField2)
	// child3 :=
	parent.AddNewChild(typ, "child1")
	child2.AddNewChild(typ, "subchild1")

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
func BuildGuiTree(widgets, parts int, typ reflect.Type) Ki {
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

var TotNodes int
var TestGUITree_NodeEmbed Ki
var TestGUITree_NodeField Ki
var TestGUITree_NodeField2 Ki

var NWidgets = 10000
var NParts = 5

func BenchmarkBuildGuiTree_NodeEmbed(b *testing.B) {
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTree(NWidgets, NParts, KiT_NodeEmbed)
		TestGUITree_NodeEmbed = wt
	}
}

func BenchmarkBuildGuiTree_NodeField(b *testing.B) {
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTree(NWidgets, NParts, KiT_NodeField)
		TestGUITree_NodeField = wt
	}
}

func BenchmarkBuildGuiTree_NodeField2(b *testing.B) {
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTree(NWidgets, NParts, KiT_NodeField2)
		TestGUITree_NodeField2 = wt
	}
}

func BenchmarkBuildGuiTreeSlow_NodeEmbed(b *testing.B) {
	// prof.Reset()
	// prof.Profiling = true
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTreeSlow(NWidgets, NParts, KiT_NodeEmbed)
		TestGUITree_NodeEmbed = wt
	}
	// prof.Report(time.Millisecond)
	// prof.Profiling = false
}

func BenchmarkFuncDownMeFirst_NodeEmbed(b *testing.B) {
	wt := TestGUITree_NodeEmbed
	nnodes := 0
	for n := 0; n < b.N; n++ {
		wt.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
			k.ClearFlag(int(Updating))
			nnodes++
			return Continue
		})
	}
	TotNodes = nnodes
	// fmt.Printf("tot nodes: %d\n", TotNodes)
}

func BenchmarkFuncDownMeFirst_NodeField(b *testing.B) {
	wt := TestGUITree_NodeField
	nnodes := 0
	for n := 0; n < b.N; n++ {
		wt.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
			k.ClearFlag(int(Updating))
			nnodes++
			return Continue
		})
	}
	TotNodes = nnodes
	// fmt.Printf("tot nodes: %d\n", TotNodes)
}

func BenchmarkFuncDownMeFirst_NodeField2(b *testing.B) {
	wt := TestGUITree_NodeField2
	nnodes := 0
	for n := 0; n < b.N; n++ {
		wt.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
			k.ClearFlag(int(Updating))
			nnodes++
			return Continue
		})
	}
	TotNodes = nnodes
	// fmt.Printf("tot nodes: %d\n", TotNodes)
}
