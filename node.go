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
	"github.com/cznic/mathutil"
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
The Node implements the Ki interface and provides the core functionality for the GoKi tree -- use the Node as an embedded struct or as a struct field -- the embedded version supports full JSON save / load
*/
type Node struct {
	Name       string
	UniqueName string
	Properties map[string]interface{}
	Parent     Ki     `json:"-"`
	ChildType  KiType `desc:"default type of child to create"`
	Children   KiSlice
	NodeSig    Signal  `json:"-", desc:"signal for node structure changes -- emits SignalType signals"`
	Updating   AtomCtr `json:"-", desc:"updating counter used in UpdateStart / End calls -- atomic for thread safety"`
	deleted    []Ki    `desc:"keeps track of deleted nodes until destroyed"`
	this       Ki      `desc:"we need a pointer to ourselves as a Ki, which can always be used to extract the true underlying type of object -- function receivers do not have this ability"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtNode = KiTypes.AddType(&Node{})

//////////////////////////////////////////////////////////////////////////
//  Basic Ki properties

func (n *Node) This() Ki {
	return n.this
}

func (n *Node) SetThis(ki Ki) {
	n.this = ki
}

func (n *Node) ThisCheck() error {
	if n.this == nil {
		return fmt.Errorf("KiNode ThisCheck: node has null 'this' pointer -- must call SetRoot on root nodes!  Name: %v", n.Name)
	}
	return nil
}

func (n *Node) KiParent() Ki {
	return n.Parent
}

func (n *Node) KiChild(idx int) (Ki, error) {
	// todo range checking?
	if idx > len(n.Children) || idx < 0 {
		return nil, fmt.Errorf("ki Node Child: index out of range: %d, n children: %d", idx, len(n.Children))
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

func (n *Node) GetProp(key string, inherit bool) interface{} {
	if n.Properties != nil {
		v, ok := n.Properties[key]
		if ok {
			return v
		}
	}
	if !inherit || n.Parent == nil {
		return nil
	}
	return n.Parent.GetProp(key, inherit)
}

func (n *Node) GetPropBool(key string, inherit bool) (bool, error) {
	v := n.GetProp(key, inherit)
	if v == nil {
		return false, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("KiNode GetPropBool -- property %v exists but is not a bool, is: %T", key, v)
	}
	return b, nil
}

func (n *Node) GetPropInt(key string, inherit bool) (int, error) {
	v := n.GetProp(key, inherit)
	if v == nil {
		return 0, nil
	}
	b, ok := v.(int)
	if !ok {
		return 0, fmt.Errorf("KiNode GetPropInt -- property %v exists but is not an int, is: %T", key, v)
	}
	return b, nil
}

func (n *Node) GetPropFloat64(key string, inherit bool) (float64, error) {
	v := n.GetProp(key, inherit)
	if v == nil {
		return 0, nil
	}
	b, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("KiNode GetPropFloat64 -- property %v exists but is not a float64, is: %T", key, v)
	}
	return b, nil
}

func (n *Node) GetPropString(key string, inherit bool) (string, error) {
	v := n.GetProp(key, inherit)
	if v == nil {
		return "", nil
	}
	b, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("KiNode GetPropString -- property %v exists but is not a string, is: %T", key, v)
	}
	return b, nil
}

func (n *Node) DelProp(key string) {
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
		n.Parent.RemoveChild(n, false)
	}
	n.Parent = parent
	if parent != nil {
		n.Updating.Set(parent.UpdateCtr().Value()) // we need parent's update counter b/c they will end
		n.DelProp("root")                          // can't be root anymore!
	}
}

func (n *Node) SetRoot(ths Ki) {
	n.SetThis(ths)
	n.SetProp("root", true)
}

func (n *Node) IsRoot() bool {
	b, _ := n.GetPropBool("root", false) // not inherit
	return b
}

func (n *Node) SetChildType(t reflect.Type) error {
	if !reflect.PtrTo(t).Implements(reflect.TypeOf((*Ki)(nil)).Elem()) {
		return fmt.Errorf("KiNode SetChildType: type does not implement the Ki interface -- must -- type passed is: %v", t.Name())
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
	kid.SetParent(n.this)
}

func (n *Node) InsertChildImpl(kid Ki, at int) {
	if err := n.ThisCheck(); err != nil {
		return
	}
	at = mathutil.Min(at, len(n.Children))
	// this avoids extra garbage collection
	n.Children = append(n.Children, nil)
	copy(n.Children[at+1:], n.Children[at:])
	kid.SetThis(kid)
	n.Children[at] = kid
	kid.SetParent(n.this)
}

func (n *Node) EmitChildAddedSignal(kid Ki) {
	if n.Updating.Value() == 0 {
		n.NodeSig.Emit(n.this, SignalChildAdded, kid)
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

func (n *Node) MakeNewChild() Ki {
	if err := n.ThisCheck(); err != nil {
		return nil
	}
	typ := n.ChildType.T
	if typ == nil {
		typ = reflect.TypeOf(n.this).Elem() // make us by default
	}
	nkid := reflect.New(typ).Interface()
	// fmt.Printf("nkid is new obj of type %T val: %+v\n", nkid, nkid)
	kid, _ := nkid.(Ki)
	// fmt.Printf("kid is new obj of type %T val: %+v\n", kid, kid)
	return kid
}

func (n *Node) AddNewChild() Ki {
	kid := n.MakeNewChild()
	n.AddChild(kid)
	return kid
}

func (n *Node) InsertNewChild(at int) Ki {
	kid := n.MakeNewChild()
	n.InsertChild(kid, at)
	return kid
}

func (n *Node) AddNewChildNamed(name string) Ki {
	kid := n.MakeNewChild()
	n.AddChildNamed(kid, name)
	return kid
}

func (n *Node) InsertNewChildNamed(at int, name string) Ki {
	kid := n.MakeNewChild()
	n.InsertChildNamed(kid, at, name)
	return kid
}

// find index of child -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
func (n *Node) FindChildIndex(kid Ki, start_idx int) int {
	if start_idx == 0 {
		for idx, child := range n.Children {
			if child == kid {
				return idx
			}
		}
	} else {
		upi := start_idx + 1
		dni := start_idx
		upo := false
		sz := len(n.Children)
		for {
			if !upo && upi < sz {
				if n.Children[upi] == kid {
					return upi
				}
				upi++
			} else {
				upo = true
			}
			if dni >= 0 {
				if n.Children[dni] == kid {
					return dni
				}
				dni--
			} else if upo {
				break
			}
		}
	}
	return -1
}

// find index of child from name -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
func (n *Node) FindChildNameIndex(name string, start_idx int) int {
	if start_idx == 0 {
		for idx, child := range n.Children {
			if child.KiName() == name {
				return idx
			}
		}
	} else {
		upi := start_idx + 1
		dni := start_idx
		upo := false
		sz := len(n.Children)
		for {
			if !upo && upi < sz {
				if n.Children[upi].KiName() == name {
					return upi
				}
				upi++
			} else {
				upo = true
			}
			if dni >= 0 {
				if n.Children[dni].KiName() == name {
					return dni
				}
				dni--
			} else if upo {
				break
			}
		}
	}
	return -1
}

// find index of child from unique name -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
func (n *Node) FindChildUniqueNameIndex(name string, start_idx int) int {
	if start_idx == 0 {
		for idx, child := range n.Children {
			if child.KiUniqueName() == name {
				return idx
			}
		}
	} else {
		upi := start_idx + 1
		dni := start_idx
		upo := false
		sz := len(n.Children)
		for {
			if !upo && upi < sz {
				if n.Children[upi].KiUniqueName() == name {
					return upi
				}
				upi++
			} else {
				upo = true
			}
			if dni >= 0 {
				if n.Children[dni].KiUniqueName() == name {
					return dni
				}
				dni--
			} else if upo {
				break
			}
		}
	}
	return -1
}

// find child from name -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
func (n *Node) FindChildName(name string, start_idx int) Ki {
	idx := n.FindChildNameIndex(name, start_idx)
	if idx < 0 {
		return nil
	}
	return n.Children[idx]
}

func (n *Node) EmitChildRemovedSignal(kid Ki) {
	if n.Updating.Value() == 0 {
		n.NodeSig.Emit(n.this, SignalChildRemoved, kid)
	}
}

// Remove child at index -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
func (n *Node) RemoveChildIndex(idx int, destroy bool) {
	child := n.Children[idx]
	// this copy makes sure there are no memory leaks
	copy(n.Children[idx:], n.Children[idx+1:])
	n.Children[len(n.Children)-1] = nil
	n.Children = n.Children[:len(n.Children)-1]
	child.SetParent(nil)
	if destroy {
		n.deleted = append(n.deleted, child)
	}
	n.EmitChildRemovedSignal(child)
}

// Remove child node -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
func (n *Node) RemoveChild(child Ki, destroy bool) {
	idx := n.FindChildIndex(child, 0)
	if idx < 0 {
		return
	}
	n.RemoveChildIndex(idx, destroy)
}

// Remove child node by name -- returns child -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
func (n *Node) RemoveChildName(name string, destroy bool) Ki {
	idx := n.FindChildNameIndex(name, 0)
	if idx < 0 {
		return nil
	}
	child := n.Children[idx]
	n.RemoveChildIndex(idx, destroy)
	return child
}

func (n *Node) EmitChildrenResetSignal() {
	if n.Updating.Value() == 0 {
		n.NodeSig.Emit(n.this, SignalChildrenReset, nil)
	}
}

// Remove all children nodes -- destroy will add removed children to deleted list, to be destroyed later -- otherwise children remain intact but parent is nil -- could be inserted elsewhere, but you better have kept a slice of them before calling this
func (n *Node) RemoveAllChildren(destroy bool) {
	for _, child := range n.Children {
		child.SetParent(nil)
	}
	if destroy {
		n.deleted = append(n.deleted, n.Children...)
	}
	n.Children = n.Children[:0] // preserves capacity of list
	n.EmitChildrenResetSignal()
}

// second-pass actually delete all previously-removed children: causes them to remove all their children and then destroy them
func (n *Node) DestroyDeleted() {
	for _, child := range n.deleted {
		child.DestroyKi()
	}
	n.deleted = n.deleted[:0]
}

// remove all children and their childrens-children, etc
func (n *Node) DestroyKi() {
	for _, child := range n.Children {
		child.DestroyKi()
	}
	n.RemoveAllChildren(true)
	n.DestroyDeleted()
}

// is this a terminal node in the tree?  i.e., has no children
func (n *Node) IsLeaf() bool {
	return len(n.Children) == 0
}

// does this node have children (i.e., non-terminal)
func (n *Node) HasChildren() bool {
	return len(n.Children) > 0
}

//////////////////////////////////////////////////////////////////////////
//  Tree walking and state updating

// call function on given node and all the way up to its parents, and so on..
func (n *Node) FunUp(data interface{}, fun KiFun) {
	if !fun(n.this, data) { // false return means stop
		return
	}
	if n.KiParent() != nil {
		n.KiParent().FunUp(data, fun)
	}
}

// call function on given node and all the way down to its children, and so on..
func (n *Node) FunDown(data interface{}, fun KiFun) {
	if !fun(n.this, data) { // false return means stop
		return
	}
	for _, child := range n.KiChildren() {
		child.FunDown(data, fun)
	}
}

// concurrent go function on given node and all the way down to its children, and so on..
func (n *Node) GoFunDown(data interface{}, fun KiFun) {
	// todo: think about a channel here to coordinate
	go fun(n.this, data)
	for _, child := range n.KiChildren() {
		child.GoFunDown(data, fun)
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
		idx := curn.FindChildUniqueNameIndex(pe, 0)
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
	n.FunDown(nil, func(k Ki, d interface{}) bool { k.UpdateCtr().Inc(); return true })
}

func (n *Node) UpdateEnd(updtall bool) {
	par_updt := false
	n.FunDown(&par_updt, func(k Ki, d interface{}) bool {
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
			return jsoniter.MarshalIndent(n.this, "", " ")
		} else {
			return json.MarshalIndent(n.this, "", " ")
		}
	} else {
		if UseJsonIter {
			return jsoniter.Marshal(n.this)
		} else {
			return json.Marshal(n.this)
		}
	}
}

func (n *Node) LoadJSON(b []byte) error {
	var err error
	if err = n.ThisCheck(); err != nil {
		return err
	}
	if UseJsonIter {
		err = jsoniter.Unmarshal(b, n.this) // key use of this!
	} else {
		err = json.Unmarshal(b, n.this) // key use of this!
	}
	if err != nil {
		return nil
	}
	n.UnmarshalPost()
	return nil
}

func (n *Node) SetKiPtrsFmPaths() {
	top := n.this
	n.FunDown(nil, func(k Ki, d interface{}) bool {
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
						pv = top.FindPathUnique(kp.Path)
						if pv == nil {
							log.Printf("KiNode SetKiPtrsFmPaths: could not find path: %v in top obj: %v", kp.Path, top.KiName())
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
	n.FunDown(nil, func(k Ki, d interface{}) bool {
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
