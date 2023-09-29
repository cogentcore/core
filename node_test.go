// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki_test

import (
	"fmt"
	"reflect"
	"testing"

	"goki.dev/gti"
	. "goki.dev/ki/v2"
	"goki.dev/ki/v2/testdata"
)

func TestNodeAddChild(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	child := testdata.NodeEmbed{}
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
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	child := testdata.NodeEmbed{}
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

func TestNodeEmbedNewChild(t *testing.T) {
	// nod := Node{}
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	child := parent.NewChild(typ, "child1")
	if len(parent.Kids) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.Kids))
	}
	if child.Path() != "/par1/child1" {
		t.Errorf("child path != correct, was %v", child.Path())
	}
	if child.KiType() != parent.KiType() {
		t.Errorf("child type != correct, was %T", child)
	}
}

func TestNodeUniqueNames(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	child := parent.NewChild(typ, "child1")
	child2 := parent.NewChild(typ, "child1")
	child3 := parent.NewChild(typ, "child1")
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
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	child := parent.NewChild(typ, "child1.go")
	child2 := parent.NewChild(typ, "child1/child1")
	child3 := parent.NewChild(typ, "child1/child1.go")
	schild2 := child2.NewChild(typ, "subchild1")
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
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	child := parent.NewChild(typ, "child1")
	parent.DeleteChild(child, true)
	if len(parent.Kids) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.Kids))
	}
	if len(DelMgr.Dels) != 0 { // note: even though using destroy, UpdateEnd does destroy
		t.Errorf("Deleted length != 0, was %d", len(DelMgr.Dels))
	}
}

func TestNodeDeleteChildName(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.NewChild(typ, "child1")
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
	typ := parent.KiType()
	for _, nm := range names {
		parent.NewChild(typ, nm)
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
	typ := parent.KiType()
	for range names {
		parent.NewChild(typ, "child")
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
	ne := parent.NewChild(testdata.NodeEmbedType, "child1")
	parent.NewChild(NodeType, "child2")

	emb := ne.KiType().HasEmbed(NodeType)
	if !emb {
		t.Errorf("HasEmbed of NodeEmbedType failed")
	}

	idx, ok := parent.Children().IndexByType(testdata.NodeEmbedType, NoEmbeds, 0)
	if !ok || idx != 0 {
		t.Errorf("find index was not correct val of %d, was %d", 0, idx)
	}
	idx, ok = parent.Children().IndexByType(NodeType, NoEmbeds, 0)
	if !ok || idx != 1 {
		t.Errorf("find index was not correct val of %d, was %d", 1, idx)
	}
	_, err := parent.Children().ElemByTypeTry(NodeType, NoEmbeds, 0)
	if err != nil {
		t.Error(err)
	}
	idx, ok = parent.Children().IndexByType(NodeType, Embeds, 0)
	if !ok || idx != 0 {
		t.Errorf("find index was not correct val of %d, was %d", 0, idx)
	}
}

func TestNodeMove(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.NewChild(typ, "child0")
	var child2 = parent.NewChild(typ, "child1").(*testdata.NodeEmbed)
	// child3 :=
	parent.NewChild(typ, "child2")
	//schild2 :=
	child2.NewChild(typ, "subchild1")
	// child4 :=
	parent.NewChild(typ, "child3")

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
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.NewChild(typ, "child0")
	var child2 = parent.NewChild(typ, "child1").(*testdata.NodeEmbed)
	// child3 :=
	parent.NewChild(typ, "child2")
	//schild2 :=
	child2.NewChild(typ, "subchild1")
	// child4 :=
	parent.NewChild(typ, "child3")

	config1 := Config{
		{Type: testdata.NodeEmbedType, Name: "child2"},
		{Type: testdata.NodeEmbedType, Name: "child3"},
		{Type: testdata.NodeEmbedType, Name: "child1"},
	}

	// bf := fmt.Sprintf("mv before:\n%v\n", parent.Kids)

	mods, updt := parent.ConfigChildren(config1)
	if mods {
		parent.UpdateEnd(updt)
	}

	cf1 := fmt.Sprintf("config1:\n%v\n", parent.Kids)

	config2 := Config{
		{testdata.NodeEmbedType, "child4"},
		{NodeType, "child1"}, // note: changing this to Node type removes child1.subchild1
		{testdata.NodeEmbedType, "child5"},
		{testdata.NodeEmbedType, "child3"},
		{testdata.NodeEmbedType, "child6"},
	}

	mods, updt = parent.ConfigChildren(config2)
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

/*

func TestNodeJSONSave(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.NewChild(typ, "child1")
	var child2 = parent.NewChild(typ, "child1").(*NodeEmbed)
	// child3 :=
	parent.NewChild(typ, "child1")
	child2.NewChild(typ, "subchild1")

	var buf bytes.Buffer
	err := parent.WriteJSON(&buf, true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output:\n%v\n", string(buf.Bytes()))
	}
	b := buf.Bytes()

	tstload := testdata.NodeEmbed{}
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
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.NewChild(typ, "child1")
	var child2 = parent.NewChild(typ, "child1").(*NodeEmbed)
	// child3 :=
	parent.NewChild(typ, "child1")
	child2.NewChild(typ, "subchild1")

	var buf bytes.Buffer
	err := parent.WriteXML(&buf, true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("xml output:\n%v\n", string(buf.Bytes()))
	}
	b := buf.Bytes()

	tstload := testdata.NodeEmbed{}
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

*/

//////////////////////////////////////////
//  function calling

func TestNodeCallFun(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.NewChild(typ, "child1")
	child2 := parent.NewChild(typ, "child1")
	// child3 :=
	parent.NewChild(typ, "child1")
	schild2 := child2.NewChild(typ, "subchild1")
	UniquifyNames(parent.This())

	res := make([]string, 0, 10)
	parent.WalkPreLevel(func(k Ki, level int) bool {
		res = append(res, fmt.Sprintf("[%v, lev %v]", k.Name(), level))
		return true
	})

	trg := []string{"[par1, lev 0]", "[child1, lev 1]", "[child1_001, lev 1]", "[subchild1, lev 2]", "[child1_002, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncDown error -- results:\n%v\n != target:\n%v\n", res, trg)
	}
	res = res[:0]

	// test return = false case
	parent.WalkPreLevel(func(k Ki, level int) bool {
		res = append(res, fmt.Sprintf("[%v, lev %v]", k.Name(), level))
		if k.Name() == "child1_001" {
			return Break
		}
		return Continue
	})

	trg = []string{"[par1, lev 0]", "[child1, lev 1]", "[child1_001, lev 1]", "[child1_002, lev 1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("FuncDown return false error -- results:\n%v\n != target:\n%v\n", res, trg)
	}
	res = res[:0]

	schild2.WalkUp(func(k Ki) bool {
		res = append(res, fmt.Sprintf("%v", k.Name()))
		return Continue
	})
	//	fmt.Printf("result: %v\n", res)

	trg = []string{"subchild1", "child1_001", "par1"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("WalkUp error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	parent.WalkPost(func(k Ki) bool {
		return Continue
	},
		func(k Ki) bool {
			res = append(res, fmt.Sprintf("[%v]", k.Name()))
			return Continue
		})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[child1]", "[subchild1]", "[child1_001]", "[child1_002]", "[par1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField WalkPost error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]

	// test for return = false working
	parent.WalkPost(func(k Ki) bool {
		if k.Name() == "child1_001" {
			return Break
		}
		return Continue
	},
		func(k Ki) bool {
			if k.Name() == "child1_001" {
				return Break
			}
			res = append(res, fmt.Sprintf("[%v]", k.Name()))
			return Continue
		})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[child1]", "[child1_002]", "[par1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField WalkPost error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]

	parent.WalkBreadth(func(k Ki) bool {
		res = append(res, fmt.Sprintf("[%v]", k.Name()))
		return Continue
	})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[par1]", "[child1]", "[child1_001]", "[child1_002]", "[subchild1]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField WalkBreadth error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]

	// test for return false
	parent.WalkBreadth(func(k Ki) bool {
		if k.Name() == "child1_001" {
			return Break
		}
		res = append(res, fmt.Sprintf("[%v]", k.Name()))
		return Continue
	})
	// fmt.Printf("node field fun result: %v\n", res)
	trg = []string{"[par1]", "[child1]", "[child1_002]"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("NodeField WalkBreadth error -- results:\n%v\n!= target:\n%v\n", res, trg)
	}
	res = res[:0]
}

func TestNodeUpdate(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	res := make([]string, 0, 10)
	// child1 :=
	updt := parent.UpdateStart()
	parent.SetChildAdded()
	parent.NewChild(typ, "child1")
	child2 := parent.NewChild(typ, "child1")
	// child3 :=
	parent.UpdateEnd(updt)
	updt = parent.UpdateStart()
	parent.SetChildAdded()
	parent.NewChild(typ, "child1")
	parent.UpdateEnd(updt)
	schild2 := child2.NewChild(typ, "subchild1")
	child2.SetChildAdded()
	parent.UpdateEnd(updt)

	// fmt.Print("\nnode update top starting\n")
	updt = child2.UpdateStart()
	updt2 := schild2.UpdateStart()
	schild2.UpdateEnd(updt2)
	child2.UpdateEnd(updt)

	UniquifyNamesAll(parent.This())

	parent.WalkPre(func(n Ki) bool {
		res = append(res, fmt.Sprintf("%v %v", n.Name(), n.Is(Updating)))
		return Continue
	})
	// fmt.Printf("res: %v\n", res)

}

func TestProps(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	// child1 :=
	parent.NewChild(typ, "child1")
	child2 := parent.NewChild(typ, "child1")
	// child3 :=
	updt := parent.UpdateStart()
	parent.NewChild(typ, "child1")
	parent.UpdateEnd(updt)
	schild2 := child2.NewChild(typ, "subchild1")

	parent.SetProp("intprop", 42)
	pprop, ok := parent.Prop("intprop").(int)
	if !ok || pprop != 42 {
		t.Errorf("TestProps error -- pprop %v != %v\n", pprop, 42)
	}
	sprop, ok := schild2.PropInherit("intprop", Inherit)
	if !ok {
		t.Errorf("TestProps error -- intprop inherited not found\n")
	}
	sint, ok := sprop.(int)
	if !ok || sprop != 42 {
		t.Errorf("TestProps error -- intprop inherited %v != %v\n", sint, 42)
	}
	sprop, ok = schild2.PropInherit("intprop", NoInherit)
	if ok {
		t.Errorf("TestProps error -- intprop should not be found!  was: %v\n", sprop)
	}

	parent.SetProp("floatprop", 42.0)
	sprop, ok = schild2.PropInherit("floatprop", Inherit)
	if !ok {
		t.Errorf("TestProps error -- floatprop inherited not found\n")
	}
	spropf, ok := sprop.(float64)
	if !ok || spropf != 42.0 {
		t.Errorf("TestProps error -- floatprop inherited %v != %v\n", spropf, 42.0)
	}

	tstr := "test string"
	parent.SetProp("stringprop", tstr)
	sprop, ok = schild2.PropInherit("stringprop", Inherit)
	if !ok {
		t.Errorf("TestProps error -- stringprop not found\n")
	}
	sprops := sprop.(string)
	if sprops != tstr {
		t.Errorf("TestProps error -- sprops inherited %v != %v\n", sprops, tstr)
	}

	parent.DeleteProp("floatprop")
	sprop, ok = schild2.PropInherit("floatprop", Inherit)
	if ok {
		t.Errorf("TestProps error -- floatprop should be gone\n")
	}

	// test type directives: replacement for type props
	tdirs := typ.Directives.ForTool("direct")
	if len(tdirs) == 0 {
		t.Errorf("Type directives error: tool 'direct' not found\n")
	}
	if tdirs[0].Directive != "value" {
		t.Errorf("Type directives error: directive should be `value`, got: %s\n", tdirs[0].Directive)
	}
}

func TestTreeMod(t *testing.T) {
	tree1 := Node{}
	typ := tree1.KiType()
	tree1.InitName(&tree1, "tree1")
	// child11 :=
	tree1.NewChild(typ, "child11")
	child12 := tree1.NewChild(typ, "child12")
	// child13 :=
	tree1.NewChild(typ, "child13")
	// schild12 :=
	child12.NewChild(typ, "subchild12")

	tree2 := Node{}
	tree2.InitName(&tree2, "tree2")
	// child21 :=
	tree2.NewChild(typ, "child21")
	child22 := tree2.NewChild(typ, "child22")
	// child23 :=
	tree2.NewChild(typ, "child23")
	// schild22 :=
	child22.NewChild(typ, "subchild22")

	// fmt.Printf("#################################\n")

	// fmt.Printf("Trees before:\n%v%v", tree1, tree2)
	updt := tree2.UpdateStart()
	tree2.SetChildAdded()
	MoveToParent(child12.This(), tree2.This())
	tree2.UpdateEnd(updt)

	// fmt.Printf("#################################\n")
	// fmt.Printf("Trees after add child12 move:\n%v%v", tree1, tree2)

	updt = tree2.UpdateStart()
	tree2.DeleteChild(child12, true)
	tree2.UpdateEnd(updt)

	// fmt.Printf("#################################\n")

	// todo need actual tests in here!
}

/*

func TestNodeFieldJSONSave(t *testing.T) {
	parent := testdata.NodeField2{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.NewChild(typ, "child1")
	child2 := parent.NewChild(typ, "child1").(*testdata.NodeField2)
	// child3 :=
	parent.NewChild(typ, "child1")
	child2.NewChild(typ, "subchild1")

	var buf bytes.Buffer
	err := parent.WriteJSON(&buf, true)
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("json output:\n%v\n", string(buf.Bytes()))
	}
	b := buf.Bytes()

	tstload := testdata.NodeField2{}
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

*/

func TestClone(t *testing.T) {
	parent := testdata.NodeField2{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.NewChild(typ, "child1")
	child2 := parent.NewChild(typ, "child1").(*testdata.NodeField2)
	// child3 :=
	parent.NewChild(typ, "child1")
	child2.NewChild(typ, "subchild1")

	/*
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
	*/
}

// BuildGuiTreeSlow builds a tree that is typical of GUI structures where there are
// many widgets in a container and each widget has some number of parts.
// Uses slow AddChild method instead of fast one.
func BuildGuiTreeSlow(widgets, parts int, typ *gti.Type) Ki {
	win := NewOfType(typ)
	win.InitName(win, "window")
	updt := win.UpdateStart()

	vp := win.NewChild(typ, "vp")
	frame := vp.NewChild(typ, "frame")
	for wi := 0; wi < widgets; wi++ {
		widg := frame.NewChild(typ, fmt.Sprintf("widg_%d", wi))

		for pi := 0; pi < parts; pi++ {
			widg.NewChild(typ, fmt.Sprintf("part_%d", pi))
		}
	}
	win.UpdateEnd(updt)
	return win
}

// BuildGuiTree builds a tree that is typical of GUI structures where there are
// many widgets in a container and each widget has some number of parts.
func BuildGuiTree(widgets, parts int, typ *gti.Type) Ki {
	win := NewOfType(typ)
	win.InitName(win, "window")
	updt := win.UpdateStart()

	vp := win.NewChild(typ, "vp")
	frame := vp.NewChild(typ, "frame")
	for wi := 0; wi < widgets; wi++ {
		widg := frame.NewChild(typ, fmt.Sprintf("widg_%d", wi))

		for pi := 0; pi < parts; pi++ {
			widg.NewChild(typ, fmt.Sprintf("part_%d", pi))
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
		wt := BuildGuiTree(NWidgets, NParts, testdata.NodeEmbedType)
		TestGUITree_NodeEmbed = wt
	}
}

func BenchmarkBuildGuiTree_NodeField(b *testing.B) {
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTree(NWidgets, NParts, testdata.NodeFieldType)
		TestGUITree_NodeField = wt
	}
}

func BenchmarkBuildGuiTree_NodeField2(b *testing.B) {
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTree(NWidgets, NParts, testdata.NodeField2Type)
		TestGUITree_NodeField2 = wt
	}
}

func BenchmarkBuildGuiTreeSlow_NodeEmbed(b *testing.B) {
	// prof.Reset()
	// prof.Profiling = true
	for n := 0; n < b.N; n++ {
		wt := BuildGuiTreeSlow(NWidgets, NParts, testdata.NodeEmbedType)
		TestGUITree_NodeEmbed = wt
	}
	// prof.Report(time.Millisecond)
	// prof.Profiling = false
}

func BenchmarkWalkPre_NodeEmbed(b *testing.B) {
	wt := TestGUITree_NodeEmbed
	nnodes := 0
	for n := 0; n < b.N; n++ {
		wt.WalkPre(func(k Ki) bool {
			k.SetFlag(false, Updating)
			nnodes++
			return Continue
		})
	}
	TotNodes = nnodes
	// fmt.Printf("tot nodes: %d\n", TotNodes)
}

func BenchmarkWalkPre_NodeField(b *testing.B) {
	wt := TestGUITree_NodeField
	nnodes := 0
	for n := 0; n < b.N; n++ {
		wt.WalkPre(func(k Ki) bool {
			k.SetFlag(false, Updating)
			nnodes++
			return Continue
		})
	}
	TotNodes = nnodes
	// fmt.Printf("tot nodes: %d\n", TotNodes)
}

func BenchmarkWalkPre_NodeField2(b *testing.B) {
	wt := TestGUITree_NodeField2
	nnodes := 0
	for n := 0; n < b.N; n++ {
		wt.WalkPre(func(k Ki) bool {
			k.SetFlag(false, Updating)
			nnodes++
			return Continue
		})
	}
	TotNodes = nnodes
	// fmt.Printf("tot nodes: %d\n", TotNodes)
}

func BenchmarkNewOfType(b *testing.B) {
	for n := 0; n < b.N; n++ {
		n := NewOfType(NodeType)
		n.InitName(n, "")
	}
}

func BenchmarkStdNew(b *testing.B) {
	for n := 0; n < b.N; n++ {
		n := new(Node)
		n.InitName(n, "")
	}
}
