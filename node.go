// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English

package ki

import (
	"encoding/json"
	"github.com/json-iterator/go"
	//	"errors"
	"fmt"
	// "github.com/cznic/mathutil"
	"log"
	"reflect"
	"strconv"
	"strings"
	// "unsafe"
)

// use this to switch between using standard json vs. faster jsoniter
// right now jsoniter does not continue with the MarshalIndent beyond first level,
// even when called specifically in the KiSlice code
var UseJsonIter bool = false

// todo:

/*
The Node implements the Ki interface and provides the core functionality for the GoKi tree -- use the Node as an embedded struct or as a struct field -- the embedded version supports full JSON save / load.

The desc: key for fields is used by the GoGr GUI viewer for help / tooltip info -- add these to all your derived struct's fields.  See relevant docs for other such tags controlling a wide range of GUI and other functionality -- Ki makes extensive use of such tags.
*/
type Node struct {
	Name       string `desc:"user-supplied name of this node -- can be empty or non-unique"`
	UniqueName string `desc:"automatically-updated version of Name that is guaranteed to be unique within the slice of Children within one Node -- used e.g., for saving Unique Paths in KiPtr pointers"`
	Properties map[string]interface{}
	Parent     Ki      `json:"-",desc:"parent of this node -- set automatically when this node is added as a child of parent"`
	ChildType  KiType  `desc:"default type of child to create -- if nil then same type as node itself is used"`
	Children   KiSlice `desc:"list of children of this node -- all are set to have this node as their parent -- can reorder etc but generally use KiNode methods to Add / Delete to ensure proper usage"`
	NodeSig    Signal  `json:"-",desc:"signal for node structure / state changes -- emits SignalType signals"`
	Updating   AtomCtr `json:"-",desc:"updating counter used in UpdateStart / End calls -- atomic for thread safety -- read using Value() method (not a good idea to modify)"`
	Deleted    []Ki    `json:"-",desc:"keeps track of deleted nodes until destroyed"`
	This       Ki      `json:"-",desc:"we need a pointer to ourselves as a Ki, which can always be used to extract the true underlying type of object when Node is embedded in other structs -- function receivers do not have this ability so this is necessary"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtNode = KiTypes.AddType(&Node{})

//////////////////////////////////////////////////////////////////////////
//  Basic Ki properties

func (n *Node) ThisKi() Ki {
	return n.This
}

func (n *Node) SetThis(ki Ki) {
	n.This = ki
}

func (n *Node) ThisCheck() error {
	if n.This == nil {
		err := fmt.Errorf("KiNode %v ThisCheck: node has null 'this' pointer -- must call SetThis/Name on root nodes!", n.PathUnique())
		log.Print(err)
		return err
	}
	return nil
}

func (n *Node) IsType(t ...reflect.Type) bool {
	if err := n.ThisCheck(); err != nil {
		return false
	}
	ttyp := reflect.TypeOf(n.This).Elem()
	for _, typ := range t {
		if typ == ttyp {
			return true
		}
	}
	return false
}

func (n *Node) KiParent() Ki {
	return n.Parent
}

func (n *Node) KiChild(idx int) (Ki, error) {
	idx, err := n.Children.ValidIndex(idx)
	if err != nil {
		return nil, err
	}
	return n.Children[idx], nil
}

func (n *Node) KiChildren() KiSlice {
	return n.Children
}

func (n *Node) KiName() string {
	return n.Name
}

func (n *Node) KiUniqueName() string {
	return n.UniqueName
}

// set name and unique name, ensuring unique name is unique..
func (n *Node) SetName(name string) {
	n.Name = name
	n.UniqueName = name
	if n.Parent != nil {
		n.Parent.UniquifyNames()
	}
}

func (n *Node) SetThisName(ki Ki, name string) {
	n.SetThis(ki)
	n.SetName(name)
}

func (n *Node) SetUniqueName(name string) {
	n.UniqueName = name
}

// make sure that the names are unique -- n^2 ish
func (n *Node) UniquifyNames() {
	for i, child := range n.Children {
		if len(child.KiUniqueName()) == 0 {
			if n.Parent != nil {
				child.SetUniqueName(n.Parent.KiUniqueName())
			} else {
				child.SetUniqueName(fmt.Sprintf("c%04d", i))
			}
		}
		for { // changed
			changed := false
			for j := i - 1; j >= 0; j-- { // check all prior
				if child.KiUniqueName() == n.Children[j].KiUniqueName() {
					if idx := strings.LastIndex(child.KiUniqueName(), "_"); idx >= 0 {
						curnum, err := strconv.ParseInt(child.KiUniqueName()[idx+1:], 10, 64)
						if err == nil { // it was a number
							curnum++
							child.SetUniqueName(child.KiUniqueName()[:idx+1] +
								strconv.FormatInt(curnum, 10))
							changed = true
							break
						}
					}
					child.SetUniqueName(child.KiUniqueName() + "_1")
					changed = true
					break
				}
			}
			if !changed {
				break
			}
		}
	}
}

//////////////////////////////////////////////////////////////////////////
//  Property interface with inheritance -- nodes can inherit props from parents

func (n *Node) KiProperties() map[string]interface{} {
	return n.Properties
}

func (n *Node) SetProp(key string, val interface{}) {
	if n.Properties == nil {
		n.Properties = make(map[string]interface{})
	}
	n.Properties[key] = val
}

func (n *Node) Prop(key string, inherit bool) interface{} {
	if n.Properties != nil {
		v, ok := n.Properties[key]
		if ok {
			return v
		}
	}
	if !inherit || n.Parent == nil {
		return nil
	}
	return n.Parent.Prop(key, inherit)
}

func (n *Node) PropBool(key string, inherit bool) (bool, error) {
	v := n.Prop(key, inherit)
	if v == nil {
		return false, nil
	}
	b, ok := v.(bool)
	if !ok {
		err := fmt.Errorf("KiNode %v PropBool -- property %v exists but is not a bool, is: %T", n.PathUnique(), key, v)
		log.Print(err)
		return false, err
	}
	return b, nil
}

func (n *Node) PropInt(key string, inherit bool) (int, error) {
	v := n.Prop(key, inherit)
	if v == nil {
		return 0, nil
	}
	b, ok := v.(int)
	if !ok {
		err := fmt.Errorf("KiNode %v PropInt -- property %v exists but is not an int, is: %T", n.PathUnique(), key, v)
		log.Print(err)
		return 0, err
	}
	return b, nil
}

func (n *Node) PropFloat64(key string, inherit bool) (float64, error) {
	v := n.Prop(key, inherit)
	if v == nil {
		return 0, nil
	}
	b, ok := v.(float64)
	if !ok {
		err := fmt.Errorf("KiNode %v PropFloat64 -- property %v exists but is not a float64, is: %T", n.PathUnique(), key, v)
		log.Print(err)
		return 0, err
	}
	return b, nil
}

func (n *Node) PropString(key string, inherit bool) (string, error) {
	v := n.Prop(key, inherit)
	if v == nil {
		return "", nil
	}
	b, ok := v.(string)
	if !ok {
		err := fmt.Errorf("KiNode %v PropString -- property %v exists but is not a string, is: %T", n.PathUnique(), key, v)
		log.Print(err)
		return "", err
	}
	return b, nil
}

func (n *Node) DeleteProp(key string) {
	if n.Properties == nil {
		return
	}
	delete(n.Properties, key)
}

//////////////////////////////////////////////////////////////////////////
//  Parent / Child Functionality

// set parent of node -- if parent is already set, then removes from that parent first -- nodes can ONLY have one parent -- only for true Tree structures, not DAG's or other such graphs that do not enforce a strict single-parent relationship
func (n *Node) SetParent(parent Ki) {
	if n.Parent != nil {
		n.Parent.DeleteChild(n, false)
	}
	n.Parent = parent
	if parent != nil {
		n.Updating.Set(parent.UpdateCtr().Value()) // we need parent's update counter b/c they will end
		n.DeleteProp("root")                       // can't be root anymore!
	}
}

func (n *Node) IsRoot() bool {
	return (n.Parent == nil)
}

func (n *Node) Root() Ki {
	if n.IsRoot() {
		return n.This
	}
	return n.Parent.Root()
}

func (n *Node) SetChildType(t reflect.Type) error {
	if !reflect.PtrTo(t).Implements(reflect.TypeOf((*Ki)(nil)).Elem()) {
		err := fmt.Errorf("KiNode %v SetChildType: type does not implement the Ki interface -- must -- type passed is: %v", n.PathUnique(), t.Name())
		log.Print(err)
		return err
	}
	n.ChildType.T = t
	return nil
}

func (n *Node) AddChildImpl(kid Ki) {
	if err := n.ThisCheck(); err != nil {
		return
	}
	kid.SetThis(kid)
	n.Children = append(n.Children, kid)
	kid.SetParent(n.This)
}

func (n *Node) InsertChildImpl(kid Ki, at int) {
	if err := n.ThisCheck(); err != nil {
		return
	}
	n.Children.InsertKi(kid, at)
	kid.SetThis(kid)
	kid.SetParent(n.This)
}

func (n *Node) EmitChildAddedSignal(kid Ki) {
	if n.Updating.Value() == 0 {
		n.NodeSig.Emit(n.This, SignalChildAdded, kid)
	}
}

func (n *Node) AddChild(kid Ki) {
	n.AddChildImpl(kid)
	n.EmitChildAddedSignal(kid)
}

func (n *Node) InsertChild(kid Ki, at int) {
	n.InsertChildImpl(kid, at)
	n.EmitChildAddedSignal(kid)
}

func (n *Node) AddChildNamed(kid Ki, name string) {
	n.AddChildImpl(kid)
	kid.SetName(name)
	n.EmitChildAddedSignal(kid)
}

func (n *Node) InsertChildNamed(kid Ki, at int, name string) {
	n.InsertChildImpl(kid, at)
	kid.SetName(name)
	n.EmitChildAddedSignal(kid)
}

func (n *Node) MakeNewChild(typ reflect.Type) Ki {
	if err := n.ThisCheck(); err != nil {
		return nil
	}
	if typ == nil {
		typ = n.ChildType.T
	}
	if typ == nil {
		typ = reflect.TypeOf(n.This).Elem() // make us by default
	}
	nkid := reflect.New(typ).Interface()
	// fmt.Printf("nkid is new obj of type %T val: %+v\n", nkid, nkid)
	kid, _ := nkid.(Ki)
	// fmt.Printf("kid is new obj of type %T val: %+v\n", kid, kid)
	return kid
}

func (n *Node) AddNewChild(typ reflect.Type) Ki {
	kid := n.MakeNewChild(typ)
	n.AddChild(kid)
	return kid
}

func (n *Node) InsertNewChild(typ reflect.Type, at int) Ki {
	kid := n.MakeNewChild(typ)
	n.InsertChild(kid, at)
	return kid
}

func (n *Node) AddNewChildNamed(typ reflect.Type, name string) Ki {
	kid := n.MakeNewChild(typ)
	n.AddChildNamed(kid, name)
	return kid
}

func (n *Node) InsertNewChildNamed(typ reflect.Type, at int, name string) Ki {
	kid := n.MakeNewChild(typ)
	n.InsertChildNamed(kid, at, name)
	return kid
}

func (n *Node) FindChildIndexByFun(start_idx int, match func(ki Ki) bool) int {
	return n.Children.FindIndexByFun(start_idx, match)
}

func (n *Node) FindChildIndex(kid Ki, start_idx int) int {
	return n.Children.FindIndex(kid, start_idx)
}

func (n *Node) FindChildIndexByName(name string, start_idx int) int {
	return n.Children.FindIndexByName(name, start_idx)
}

func (n *Node) FindChildIndexByUniqueName(name string, start_idx int) int {
	return n.Children.FindIndexByUniqueName(name, start_idx)
}

func (n *Node) FindChildIndexByType(t ...reflect.Type) int {
	return n.Children.FindIndexByType(t...)
}

func (n *Node) FindChildByName(name string, start_idx int) Ki {
	idx := n.Children.FindIndexByName(name, start_idx)
	if idx < 0 {
		return nil
	}
	return n.Children[idx]
}

func (n *Node) FindChildByType(t ...reflect.Type) Ki {
	idx := n.Children.FindIndexByType(t...)
	if idx < 0 {
		return nil
	}
	return n.Children[idx]
}

func (n *Node) FindParentByName(name string) Ki {
	if n.IsRoot() {
		return nil
	}
	if n.Parent.KiName() == name {
		return n.Parent
	}
	return n.Parent.FindParentByName(name)
}

func (n *Node) FindParentByType(t ...reflect.Type) Ki {
	if n.IsRoot() {
		return nil
	}
	if n.Parent.IsType(t...) {
		return n.Parent
	}
	return n.Parent.FindParentByType(t...)
}

func (n *Node) EmitChildDeletedSignal(kid Ki) {
	if n.Updating.Value() == 0 {
		n.NodeSig.Emit(n.This, SignalChildDeleted, kid)
	}
}

func (n *Node) DeleteChildAtIndex(idx int, destroy bool) {
	idx, err := n.Children.ValidIndex(idx)
	if err != nil {
		log.Print("KiNode DeleteChildAtIndex -- attempt to delete item in empty children slice")
		return
	}
	child := n.Children[idx]
	_ = n.Children.DeleteAtIndex(idx)
	child.SetParent(nil)
	if destroy {
		n.Deleted = append(n.Deleted, child)
	}
	n.EmitChildDeletedSignal(child)
}

func (n *Node) DeleteChild(child Ki, destroy bool) {
	idx := n.FindChildIndex(child, 0)
	if idx < 0 {
		return
	}
	n.DeleteChildAtIndex(idx, destroy)
}

func (n *Node) DeleteChildByName(name string, destroy bool) Ki {
	idx := n.FindChildIndexByName(name, 0)
	if idx < 0 {
		return nil
	}
	child := n.Children[idx]
	n.DeleteChildAtIndex(idx, destroy)
	return child
}

func (n *Node) EmitChildrenDeletedSignal() {
	if n.Updating.Value() == 0 {
		n.NodeSig.Emit(n.This, SignalChildrenDeleted, nil)
	}
}

func (n *Node) DeleteChildren(destroy bool) {
	for _, child := range n.Children {
		child.SetParent(nil)
	}
	if destroy {
		n.Deleted = append(n.Deleted, n.Children...)
	}
	n.Children = n.Children[:0] // preserves capacity of list
	n.EmitChildrenDeletedSignal()
}

func (n *Node) DestroyDeleted() {
	for _, child := range n.Deleted {
		child.DestroyKi()
	}
	n.Deleted = n.Deleted[:0]
}

func (n *Node) DestroyKi() {
	for _, child := range n.Children {
		child.DestroyKi()
	}
	n.DeleteChildren(true)
	n.DestroyDeleted()
}

func (n *Node) IsLeaf() bool {
	return len(n.Children) == 0
}

func (n *Node) HasChildren() bool {
	return len(n.Children) > 0
}

//////////////////////////////////////////////////////////////////////////
//  Tree walking and state updating

func (n *Node) FunUp(level int, data interface{}, fun KiFun) {
	if !fun(n.This, level, data) { // false return means stop
		return
	}
	level++
	if n.KiParent() != nil {
		n.KiParent().FunUp(level, data, fun)
	}
}

func (n *Node) FunDown(level int, data interface{}, fun KiFun) {
	if !fun(n.This, level, data) { // false return means stop
		return
	}
	level++
	for _, child := range n.KiChildren() {
		child.FunDown(level, data, fun)
	}
}

func (n *Node) FunDownBreadthFirst(level int, data interface{}, fun KiFun) {
	level++
	for _, child := range n.KiChildren() {
		if !fun(child, level, data) { // false return means stop
			return
		}
	}
	for _, child := range n.KiChildren() {
		child.FunDownBreadthFirst(level, data, fun)
	}
}

func (n *Node) GoFunDown(level int, data interface{}, fun KiFun) {
	go fun(n.This, level, data)
	level++
	for _, child := range n.KiChildren() {
		child.GoFunDown(level, data, fun)
	}
}

func (n *Node) GoFunDownWait(level int, data interface{}, fun KiFun) {
	// todo: use channel or something to wait
	go fun(n.This, level, data)
	level++
	for _, child := range n.KiChildren() {
		child.GoFunDown(level, data, fun)
	}
}

func (n *Node) Path() string {
	if n.Parent != nil {
		return n.Parent.Path() + "." + n.Name
	}
	return "." + n.Name
}

func (n *Node) PathUnique() string {
	if n.Parent != nil {
		return n.Parent.PathUnique() + "." + n.UniqueName
	}
	return "." + n.UniqueName
}

func (n *Node) FindPathUnique(path string) Ki {
	curn := Ki(n)
	pels := strings.Split(strings.Trim(strings.TrimSpace(path), "\""), ".")
	for i, pe := range pels {
		if len(pe) == 0 {
			continue
		}
		// fmt.Printf("pe: %v\n", pe)
		if i <= 1 && curn.KiUniqueName() == pe {
			continue
		}
		idx := curn.FindChildIndexByUniqueName(pe, 0)
		if idx < 0 {
			return nil
		}
		curn, _ = curn.KiChild(idx)
	}
	return curn
}

//////////////////////////////////////////////////////////////////////////
//  State update signaling -- automatically consolidates all changes across
//   levels so there is only one update at end (optionally per node or only
//   at highest level)
//   All modification starts with UpdateStart() and ends with UpdateEnd()

func (n *Node) NodeSignal() *Signal {
	return &n.NodeSig
}

func (n *Node) UpdateCtr() *AtomCtr {
	return &n.Updating
}

func (n *Node) UpdateStart() {
	n.FunDown(0, nil, func(k Ki, level int, d interface{}) bool { k.UpdateCtr().Inc(); return true })
}

func (n *Node) UpdateEnd(updtall bool) {
	par_updt := false
	n.FunDown(0, &par_updt, func(k Ki, level int, d interface{}) bool {
		par_updt := d.(*bool)           // did the parent already update?
		if k.UpdateCtr().Value() == 1 { // we will go to 0 -- but don't do yet so !updtall works
			if updtall {
				k.UpdateCtr().Dec()
				k.NodeSignal().Emit(k, SignalNodeUpdated, d)
			} else {
				if k.KiParent() == nil || (!*par_updt && k.KiParent().UpdateCtr().Value() == 0) {
					k.UpdateCtr().Dec()
					k.NodeSignal().Emit(k, SignalNodeUpdated, d)
					*par_updt = true // we updated so nobody else can!
				} else {
					k.UpdateCtr().Dec()
				}
			}
		} else {
			if k.UpdateCtr().Value() <= 0 {
				log.Printf("KiNode UpdateEnd called with Updating <= 0: %d in node: %v\n", *(k.UpdateCtr()), k.PathUnique())
			} else {
				k.UpdateCtr().Dec()
			}
		}
		return true
	})
}

//////////////////////////////////////////////////////////////////////////
//  Marshal / Unmarshal support -- mostly in KiSlice

func (n *Node) SaveJSON(indent bool) ([]byte, error) {
	if err := n.ThisCheck(); err != nil {
		return nil, err
	}
	if indent {
		if UseJsonIter {
			return jsoniter.MarshalIndent(n.This, "", " ")
		} else {
			return json.MarshalIndent(n.This, "", " ")
		}
	} else {
		if UseJsonIter {
			return jsoniter.Marshal(n.This)
		} else {
			return json.Marshal(n.This)
		}
	}
}

func (n *Node) LoadJSON(b []byte) error {
	var err error
	if err = n.ThisCheck(); err != nil {
		return err
	}
	if UseJsonIter {
		err = jsoniter.Unmarshal(b, n.This) // key use of this!
	} else {
		err = json.Unmarshal(b, n.This) // key use of this!
	}
	if err != nil {
		return nil
	}
	n.UnmarshalPost()
	return nil
}

func (n *Node) SetKiPtrsFmPaths() {
	root := n.This
	n.FunDown(0, nil, func(k Ki, level int, d interface{}) bool {
		v := reflect.ValueOf(k).Elem()
		// fmt.Printf("v: %v\n", v.Type())
		for i := 0; i < v.NumField(); i++ {
			vf := v.Field(i)
			// fmt.Printf("vf: %v\n", vf.Type())
			if vf.CanInterface() {
				kp, ok := (vf.Interface()).(KiPtr)
				if ok {
					var pv Ki
					if len(kp.Path) > 0 {
						pv = root.FindPathUnique(kp.Path)
						if pv == nil {
							log.Printf("KiNode SetKiPtrsFmPaths: could not find path: %v in root obj: %v", kp.Path, root.KiName())
						}
						vf.FieldByName("Ptr").Set(reflect.ValueOf(pv))
					}
					// todo: should set Ptr to nil but can't seem to do that here -- complains when above set is called with pv = nil
				}
			}
		}
		return true
	})
}

func (n *Node) ParentAllChildren() {
	n.FunDown(0, nil, func(k Ki, level int, d interface{}) bool {
		for _, child := range k.KiChildren() {
			child.SetParent(k)
		}
		return true
	})
}

func (n *Node) UnmarshalPost() {
	n.ParentAllChildren()
	n.SetKiPtrsFmPaths()
}
