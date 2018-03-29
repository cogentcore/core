// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English

package ki

import (
	"encoding/json"
	"encoding/xml"
	"github.com/json-iterator/go"
	"github.com/rcoreilly/goki/ki/atomctr"
	"github.com/rcoreilly/goki/ki/kit"
	//	"errors"
	"fmt"
	// "github.com/cznic/mathutil"
	"log"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	// "unsafe"
)

// use this to switch between using standard json vs. faster jsoniter
// right now jsoniter does not continue with the MarshalIndent beyond first level,
// even when called specifically in the Slice code
var UseJsonIter bool = false

// todo:

/*
The Node implements the Ki interface and provides the core functionality for the GoKi tree -- use the Node as an embedded struct or as a struct field -- the embedded version supports full JSON save / load.

The desc: key for fields is used by the GoGr GUI viewer for help / tooltip info -- add these to all your derived struct's fields.  See relevant docs for other such tags controlling a wide range of GUI and other functionality -- Ki makes extensive use of such tags.
*/
type Node struct {
	Name       string                 `desc:"user-supplied name of this node -- can be empty or non-unique"`
	UniqueName string                 `desc:"automatically-updated version of Name that is guaranteed to be unique within the slice of Children within one Node -- used e.g., for saving Unique Paths in Ptr pointers"`
	Props      map[string]interface{} `xml:"-" desc:"property map for arbitrary extensible properties, including style properties"`
	Parent     Ki                     `json:"-" xml:"-" desc:"parent of this node -- set automatically when this node is added as a child of parent"`
	ChildType  kit.Type               `desc:"default type of child to create -- if nil then same type as node itself is used"`
	Children   Slice                  `desc:"list of children of this node -- all are set to have this node as their parent -- can reorder etc but generally use KiNode methods to Add / Delete to ensure proper usage"`
	NodeSig    Signal                 `json:"-" xml:"-" desc:"signal for node structure / state changes -- emits NodeSignals signals -- can also extend to custom signals (see signal.go) but in general better to create a new Signal instead"`
	Updating   atomctr.Ctr            `json:"-" xml:"-" desc:"updating counter used in UpdateStart / End calls -- atomic for thread safety -- read using Value() method (not a good idea to modify)"`
	Deleted    []Ki                   `json:"-" xml:"-" desc:"keeps track of deleted nodes until destroyed"`
	This       Ki                     `json:"-" xml:"-" desc:"we need a pointer to ourselves as a Ki, which can always be used to extract the true underlying type of object when Node is embedded in other structs -- function receivers do not have this ability so this is necessary"`
	TmpProps   map[string]interface{} `json:"-" xml:"-" desc:"temporary properties that are not saved -- e.g., used by Gi views to store view properties"`
}

// must register all new types so type names can be looked up by name -- also props
var KiT_Node = kit.Types.AddType(&Node{}, nil)

// check for interface implementation
var _ Ki = &Node{}

//////////////////////////////////////////////////////////////////////////
//  Stringer

// stringer interface -- basic indented tree representation
func (n Node) String() string {
	str := ""
	n.FunDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		for i := 0; i < level; i++ {
			str += "\t"
		}
		str += k.KiName() + "\n"
		return true
	})
	return str
}

//////////////////////////////////////////////////////////////////////////
//  Basic Ki fields

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

func (n *Node) Type() reflect.Type {
	return reflect.TypeOf(n.This).Elem()
}

func (n *Node) IsType(t ...reflect.Type) bool {
	if err := n.ThisCheck(); err != nil {
		return false
	}
	ttyp := n.Type()
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

func (n *Node) KiChildren() Slice {
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

func (n *Node) KiProps() map[string]interface{} {
	return n.Props
}

func (n *Node) SetProp(key string, val interface{}) {
	if n.Props == nil {
		n.Props = make(map[string]interface{})
	}
	n.Props[key] = val
}

func (n *Node) Prop(key string, inherit, typ bool) interface{} {
	if n.Props != nil {
		v, ok := n.Props[key]
		if ok {
			return v
		}
	}
	if inherit && n.Parent != nil {
		pv := n.Parent.Prop(key, inherit, typ)
		if pv != nil {
			return pv
		}
	}
	if typ {
		return kit.Types.Prop(n.Type().Name(), key)
	}
	return nil
}

func (n *Node) DeleteProp(key string) {
	if n.Props == nil {
		return
	}
	delete(n.Props, key)
}

//////////////////////////////////////////////////////////////////////////
//  Parent / Child Functionality

// set parent of node -- does not remove from existing parent -- use Add / Insert / Delete
func (n *Node) SetParent(parent Ki) {
	n.Parent = parent
	if parent != nil {
		upc := parent.UpdateCtr().Value() // we need parent's update counter b/c they will end
		n.FunDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.UpdateCtr().Set(upc)
			return true
		})
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

// check if it is safe to add child -- it cannot be a parent of us -- prevent loops!
func (n *Node) AddChildCheck(kid Ki) error {
	var err error
	n.FunUp(0, n, func(k Ki, level int, d interface{}) bool {
		if k == kid {
			err = fmt.Errorf("KiNode Attempt to add child to node %v that is my own parent -- no cycles permitted!\n", (d.(Ki)).PathUnique())
			log.Printf("%v", err)
			return false
		}
		return true
	})
	return err
}

// after adding child -- signals etc
func (n *Node) addChildImplPost(kid Ki) {
	oldPar := kid.KiParent()
	kid.SetParent(n.This) // key to set new parent before deleting: indicates move instead of delete
	if oldPar != nil {
		oldPar.DeleteChild(kid, false)
		kid.NodeSignal().Emit(kid, int64(NodeSignalMoved), oldPar)
	} else {
		kid.NodeSignal().Emit(kid, int64(NodeSignalAdded), nil)
	}
}

func (n *Node) AddChildImpl(kid Ki) error {
	if err := n.ThisCheck(); err != nil {
		return err
	}
	kid.SetThis(kid)
	if err := n.AddChildCheck(kid); err != nil {
		return err
	}
	n.Children = append(n.Children, kid)
	n.addChildImplPost(kid)
	return nil
}

func (n *Node) InsertChildImpl(kid Ki, at int) error {
	if err := n.ThisCheck(); err != nil {
		return err
	}
	kid.SetThis(kid)
	if err := n.AddChildCheck(kid); err != nil {
		return err
	}
	n.Children.InsertKi(kid, at)
	n.addChildImplPost(kid)
	return nil
}

func (n *Node) EmitChildAddedSignal(kid Ki) {
	if n.Updating.Value() == 0 {
		n.NodeSig.Emit(n.This, int64(NodeSignalChildAdded), kid)
	}
}

func (n *Node) AddChild(kid Ki) error {
	err := n.AddChildImpl(kid)
	n.EmitChildAddedSignal(kid)
	return err
}

func (n *Node) InsertChild(kid Ki, at int) error {
	err := n.InsertChildImpl(kid, at)
	n.EmitChildAddedSignal(kid)
	return err
}

func (n *Node) AddChildNamed(kid Ki, name string) error {
	err := n.AddChildImpl(kid)
	kid.SetName(name)
	n.EmitChildAddedSignal(kid)
	return err
}

func (n *Node) InsertChildNamed(kid Ki, at int, name string) error {
	err := n.InsertChildImpl(kid, at)
	kid.SetName(name)
	n.EmitChildAddedSignal(kid)
	return err
}

func (n *Node) MakeNewChild(typ reflect.Type) Ki {
	if err := n.ThisCheck(); err != nil {
		return nil
	}
	if typ == nil {
		typ = n.ChildType.T
	}
	if typ == nil {
		typ = n.Type() // make us by default
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
		n.NodeSig.Emit(n.This, int64(NodeSignalChildDeleted), kid)
	}
}

func (n *Node) DeleteChildAtIndex(idx int, destroy bool) {
	idx, err := n.Children.ValidIndex(idx)
	if err != nil {
		log.Print("KiNode DeleteChildAtIndex -- attempt to delete item in empty children slice")
		return
	}
	child := n.Children[idx]
	if child.KiParent() == n.This {
		// only deleting if we are still parent -- change parent first to signal move
		child.NodeSignal().Emit(child, int64(NodeSignalDeleting), nil)
		child.SetParent(nil)
	}
	_ = n.Children.DeleteAtIndex(idx)
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
		n.NodeSig.Emit(n.This, int64(NodeSignalChildrenDeleted), nil)
	}
}

func (n *Node) DeleteChildren(destroy bool) {
	for _, child := range n.Children {
		child.NodeSignal().Emit(child, int64(NodeSignalDeleting), nil)
		child.SetParent(nil)
	}
	if destroy {
		n.Deleted = append(n.Deleted, n.Children...)
	}
	n.Children = n.Children[:0] // preserves capacity of list
	n.EmitChildrenDeletedSignal()
}

func (n *Node) DeleteMe(destroy bool) {
	if n.Parent == nil {
		if destroy {
			n.DestroyKi()
		}
	} else {
		n.Parent.DeleteChild(n.This, destroy)
	}
}

func (n *Node) DestroyDeleted() {
	for _, child := range n.Deleted {
		child.DestroyKi()
	}
	n.Deleted = n.Deleted[:0]
}

func (n *Node) DestroyAllDeleted() {
	n.FunDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		k.DestroyDeleted()
		return true
	})
	runtime.GC() // this is a great time to call the GC!
}

func (n *Node) DestroyKi() {
	n.NodeSig.Emit(n.This, int64(NodeSignalDestroying), nil)
	// todo: traverse struct and un-set all Ptr's!
	n.DeleteChildren(true) // first delete all my children
	n.DestroyDeleted()     // then destroy all those kids
}

func (n *Node) IsLeaf() bool {
	return len(n.Children) == 0
}

func (n *Node) HasChildren() bool {
	return len(n.Children) > 0
}

//////////////////////////////////////////////////////////////////////////
//  Tree walking and state updating

func (n *Node) FunUp(level int, data interface{}, fun KiFun) bool {
	if !fun(n.This, level, data) { // false return means stop
		return false
	}
	level++
	if n.KiParent() != nil {
		return n.KiParent().FunUp(level, data, fun)
	}
	return true
}

func (n *Node) FunUpParent(level int, data interface{}, fun KiFun) bool {
	if n.IsRoot() {
		return true
	}
	if !fun(n.KiParent(), level, data) { // false return means stop
		return false
	}
	level++
	return n.KiParent().FunUpParent(level, data, fun)
}

func (n *Node) FunDownMeFirst(level int, data interface{}, fun KiFun) bool {
	if !fun(n.This, level, data) { // false return means stop
		return false
	}
	level++
	for _, child := range n.KiChildren() {
		child.FunDownMeFirst(level, data, fun) // don't care about their return values
	}
	return true
}

func (n *Node) FunDownDepthFirst(level int, data interface{}, doChildTestFun KiFun, fun KiFun) {
	level++
	for _, child := range n.KiChildren() {
		if doChildTestFun(n.This, level, data) { // test if we should run on this child
			child.FunDownDepthFirst(level, data, doChildTestFun, fun)
		}
	}
	level--
	fun(n.This, level, data) // can't use the return value at this point
}

func (n *Node) FunDownBreadthFirst(level int, data interface{}, fun KiFun) {
	dontMap := make(map[int]bool) // map of who NOT to process further -- default is false for map so reverse
	level++
	for i, child := range n.KiChildren() {
		if !fun(child, level, data) { // false return means stop
			dontMap[i] = true
		}
	}
	for i, child := range n.KiChildren() {
		if dontMap[i] {
			continue
		}
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

func (n *Node) FunPrev(level int, data interface{}, fun KiFun) bool {
	return true
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

// todo: send NodeSignalDestroying before destroying, etc
// general logic: child added / deleted events are blocked by updating count
// but NodeDeleted and NodeDestroying are NOT

// after an UpdateEnd, DestroyDeleted is called

func (n *Node) NodeSignal() *Signal {
	return &n.NodeSig
}

func (n *Node) UpdateCtr() *atomctr.Ctr {
	return &n.Updating
}

func (n *Node) UpdateStart() {
	n.FunDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		k.UpdateCtr().Inc()
		return true
	})
}

func (n *Node) UpdateEnd() {
	par_updt := false
	n.FunDownMeFirst(0, &par_updt, func(k Ki, level int, d interface{}) bool {
		par_up := d.(*bool)             // did the parent already update?
		if k.UpdateCtr().Value() == 1 { // we will go to 0 -- but don't do yet so !updtall works
			if k.KiParent() == nil || (!*par_up && k.KiParent().UpdateCtr().Value() == 0) {
				k.UpdateCtr().Dec()
				k.NodeSignal().Emit(k, int64(NodeSignalUpdated), nil)
				k.DestroyAllDeleted()
				*par_up = true // we updated so nobody else can!
			} else {
				k.UpdateCtr().Dec()
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

func (n *Node) UpdateEndAll() {
	n.FunDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		if k.UpdateCtr().Value() == 1 { // we will go to 0 -- but don't do yet so !updtall works
			k.UpdateCtr().Dec()
			k.NodeSignal().Emit(k, int64(NodeSignalUpdated), nil)
			k.DestroyDeleted()
		} else {
			if k.UpdateCtr().Value() <= 0 {
				log.Printf("KiNode UpdateEndAll called with Updating <= 0: %d in node: %v\n", *(k.UpdateCtr()), k.PathUnique())
			} else {
				k.UpdateCtr().Dec()
			}
		}
		return true
	})
}

//////////////////////////////////////////////////////////////////////////
//  Marshal / Unmarshal support -- mostly in Slice

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

func (n *Node) SaveXML(indent bool) ([]byte, error) {
	if err := n.ThisCheck(); err != nil {
		return nil, err
	}
	if indent {
		return xml.MarshalIndent(n.This, "", "  ")
	} else {
		return xml.Marshal(n.This)
	}
}

func (n *Node) LoadXML(b []byte) error {
	var err error
	if err = n.ThisCheck(); err != nil {
		return err
	}
	err = xml.Unmarshal(b, n.This) // key use of this!
	if err != nil {
		return nil
	}
	n.UnmarshalPost()
	return nil
}

func (n *Node) SetPtrsFmPaths() {
	root := n.This
	n.FunDownMeFirst(0, root, func(k Ki, level int, d interface{}) bool {
		v := reflect.ValueOf(k).Elem()
		// fmt.Printf("v: %v\n", v.Type())
		for i := 0; i < v.NumField(); i++ {
			vf := v.Field(i)
			// fmt.Printf("vf: %v\n", vf.Type())
			if vf.CanInterface() {
				kp, ok := (vf.Interface()).(Ptr)
				if ok {
					var pv Ki
					if len(kp.Path) > 0 {
						rt := d.(Ki)
						pv = rt.FindPathUnique(kp.Path)
						if pv == nil {
							log.Printf("KiNode SetPtrsFmPaths: could not find path: %v in root obj: %v", kp.Path, rt.KiName())
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
	n.FunDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		for _, child := range k.KiChildren() {
			if child != nil {
				child.SetParent(k)
			} else {
				return false
			}
		}
		return true
	})
}

func (n *Node) UnmarshalPost() {
	n.ParentAllChildren()
	n.SetPtrsFmPaths()
}
