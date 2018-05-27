// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English

package ki

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"sync"
	"unsafe"

	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
	"github.com/jinzhu/copier"
)

// The Node implements the Ki interface and provides the core functionality
// for the GoKi tree -- use the Node as an embedded struct or as a struct
// field -- the embedded version supports full JSON save / load.
//
// The desc: key for fields is used by the GoGr GUI viewer for help / tooltip
// info -- add these to all your derived struct's fields.  See relevant docs
// for other such tags controlling a wide range of GUI and other functionality
// -- Ki makes extensive use of such tags.
//
type Node struct {
	Nm       string     `copy:"-" label:"Name" desc:"Ki.Name() user-supplied name of this node -- can be empty or non-unique"`
	UniqueNm string     `copy:"-" view:"-" label:"UniqueName" desc:"Ki.UniqueName() automatically-updated version of Name that is guaranteed to be unique within the slice of Children within one Node -- used e.g., for saving Unique Paths in Ptr pointers"`
	Flag     int64      `copy:"-" json:"-" xml:"-" view:"-" desc:"bit flags for internal node state"`
	Props    Props      `xml:"-" copy:"-" label:"Properties" desc:"Ki.Properties() property map for arbitrary extensible properties, including style properties"`
	Par      Ki         `copy:"-" json:"-" xml:"-" label:"Parent" view:"-" desc:"Ki.Parent() parent of this node -- set automatically when this node is added as a child of parent"`
	Kids     Slice      `copy:"-" label:"Children" desc:"Ki.Children() list of children of this node -- all are set to have this node as their parent -- can reorder etc but generally use Ki Node methods to Add / Delete to ensure proper usage"`
	NodeSig  Signal     `copy:"-" json:"-" xml:"-" desc:"Ki.NodeSignal() signal for node structure / state changes -- emits NodeSignals signals -- can also extend to custom signals (see signal.go) but in general better to create a new Signal instead"`
	This     Ki         `copy:"-" json:"-" xml:"-" view:"-" desc:"we need a pointer to ourselves as a Ki, which can always be used to extract the true underlying type of object when Node is embedded in other structs -- function receivers do not have this ability so this is necessary"`
	FlagMu   sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"mutex protecting flag updates"`
	index    int        `desc:"last value of our index -- used as a starting point for finding us in our parent next time -- is not guaranteed to be accurate!  use Index() method`
}

// must register all new types so type names can be looked up by name -- also props
var KiT_Node = kit.Types.AddType(&Node{}, nil)

//////////////////////////////////////////////////////////////////////////
//  fmt.Stringer

// fmt.stringer interface -- basic indented tree representation
func (n Node) String() string {
	str := ""
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		for i := 0; i < level; i++ {
			str += "\t"
		}
		str += k.Name() + "\n"
		return true
	})
	return str
}

//////////////////////////////////////////////////////////////////////////
//  Basic Ki fields

func (n *Node) Init(this Ki) {
	kitype := KiType()
	bitflag.ClearMask(n.Flags(), int64(UpdateFlagsMask))
	if n.This != this {
		n.This = this
		// we need to call this directly instead of FuncFields because we need the field name
		FlatFieldsValueFunc(n.This, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			if fieldVal.Kind() == reflect.Struct && kit.EmbeddedTypeImplements(field.Type, kitype) {
				fk := kit.PtrValue(fieldVal).Interface().(Ki)
				if fk != nil {
					bitflag.Set(fk.Flags(), int(IsField))
					fk.InitName(fk, field.Name)
					fk.SetParent(n.This)
				}
			}
			return true
		})
	}
}

func (n *Node) InitName(ki Ki, name string) {
	n.Init(ki)
	n.SetName(name)
}

func (n *Node) ThisCheck() error {
	if n.This == nil {
		err := fmt.Errorf("Ki Node %v ThisCheck: node has null 'this' pointer -- must call SetThis/Name on root nodes!", n.PathUnique())
		log.Print(err)
		return err
	}
	return nil
}

func (n *Node) ThisOk() bool {
	return n.This != nil
}

func (n *Node) Type() reflect.Type {
	return reflect.TypeOf(n.This).Elem()
}

func (n *Node) TypeEmbeds(t reflect.Type) bool {
	return kit.TypeEmbeds(n.Type(), t)
}

func (n *Node) EmbeddedStruct(t reflect.Type) Ki {
	if n == nil {
		return nil
	}
	es := kit.EmbeddedStruct(n.This, t)
	if es != nil {
		k, ok := es.(Ki)
		if ok {
			return k
		}
		log.Printf("ki.EmbeddedStruct on: %v embedded struct is not a Ki type -- use kit.EmbeddedStruct for a more general version\n", n.PathUnique())
		return nil
	}
	return nil
}

func (n *Node) Parent() Ki {
	return n.Par
}

func (n *Node) HasParent(par Ki) bool {
	gotPar := false
	n.FuncUpParent(0, n, func(k Ki, level int, d interface{}) bool {
		if k == par {
			gotPar = true
			return false
		}
		return true
	})
	return gotPar
}

func (n *Node) Child(idx int) Ki {
	return n.Kids.Elem(idx)
}

func (n *Node) Children() Slice {
	return n.Kids
}

func (n *Node) IsValidIndex(idx int) bool {
	return n.Kids.IsValidIndex(idx)
}

func (n *Node) Name() string {
	return n.Nm
}

func (n *Node) UniqueName() string {
	return n.UniqueNm
}

// set name and unique name, ensuring unique name is unique..
func (n *Node) SetName(name string) bool {
	if n.Nm == name {
		return false
	}
	n.Nm = name
	n.UniqueNm = name
	if n.Par != nil {
		n.Par.UniquifyNames()
	}
	return true
}

func (n *Node) SetNameRaw(name string) {
	n.Nm = name
}

func (n *Node) SetUniqueName(name string) {
	n.UniqueNm = name
}

// make sure that the names are unique -- n^2 ish
func (n *Node) UniquifyNames() {
	if len(n.Kids) > 50 { // todo: figure out a better strategy for this many
		return
	}
	pr := prof.Start("ki.Node.UniquifyNames")
	for i, child := range n.Kids {
		if len(child.UniqueName()) == 0 {
			if n.Par != nil {
				child.SetUniqueName(n.Par.UniqueName())
			} else {
				child.SetUniqueName(fmt.Sprintf("c%04d", i))
			}
		}
		for { // changed
			changed := false
			for j := i - 1; j >= 0; j-- { // check all prior
				if child.UniqueName() == n.Kids[j].UniqueName() {
					if idx := strings.LastIndex(child.UniqueName(), "_"); idx >= 0 {
						curnum, err := strconv.ParseInt(child.UniqueName()[idx+1:], 10, 64)
						if err == nil { // it was a number
							curnum++
							child.SetUniqueName(child.UniqueName()[:idx+1] +
								strconv.FormatInt(curnum, 10))
							changed = true
							break
						}
					}
					child.SetUniqueName(child.UniqueName() + "_1")
					changed = true
					break
				}
			}
			if !changed {
				break
			}
		}
	}
	pr.End()
}

//////////////////////////////////////////////////////////////////////////
//  Flags

func (n *Node) Flags() *int64 {
	return &n.Flag
}

func (n *Node) SetFlagMu(flag ...int) {
	n.FlagMu.Lock()
	bitflag.Set(&n.Flag, flag...)
	n.FlagMu.Unlock()
}

func (n *Node) SetFlagStateMu(on bool, flag ...int) {
	n.FlagMu.Lock()
	bitflag.SetState(&n.Flag, on, flag...)
	n.FlagMu.Unlock()
}

func (n *Node) ClearFlagMu(flag ...int) {
	n.FlagMu.Lock()
	bitflag.Clear(&n.Flag, flag...)
	n.FlagMu.Unlock()
}

func (n *Node) IsUpdatingMu() bool {
	n.FlagMu.Lock()
	rval := bitflag.Has(n.Flag, int(Updating))
	n.FlagMu.Unlock()
	return rval
}

func (n *Node) IsUpdating() bool {
	return bitflag.Has(n.Flag, int(Updating))
}

func (n *Node) IsField() bool {
	return bitflag.Has(n.Flag, int(IsField))
}

func (n *Node) OnlySelfUpdate() bool {
	return bitflag.Has(n.Flag, int(OnlySelfUpdate))
}

func (n *Node) SetOnlySelfUpdate() {
	bitflag.Set(&n.Flag, int(OnlySelfUpdate))
}

func (n *Node) IsDeleted() bool {
	return bitflag.Has(n.Flag, int(NodeDeleted))
}

func (n *Node) IsDestroyed() bool {
	return bitflag.Has(n.Flag, int(NodeDestroyed))
}

//////////////////////////////////////////////////////////////////////////
//  Property interface with inheritance -- nodes can inherit props from parents

func (n *Node) Properties() Props {
	return n.Props
}

func (n *Node) SetProp(key string, val interface{}) {
	if n.Props == nil {
		n.Props = make(Props)
	}
	n.Props[key] = val
}

func (n *Node) SetProps(props Props, update bool) {
	if n.Props == nil {
		n.Props = make(Props)
	}
	for key, val := range props {
		n.Props[key] = val
	}
	if update {
		bitflag.Set(n.Flags(), int(PropUpdated))
		n.UpdateSig()
	}
}

func (n *Node) SetPropUpdate(key string, val interface{}) {
	bitflag.Set(n.Flags(), int(PropUpdated))
	n.SetProp(key, val)
	n.UpdateSig()
}

func (n *Node) SetPropChildren(key string, val interface{}) {
	for _, k := range n.Kids {
		k.SetProp(key, val)
	}
}

func (n *Node) Prop(key string, inherit, typ bool) interface{} {
	if n.Props != nil {
		v, ok := n.Props[key]
		if ok {
			return v
		}
	}
	if inherit && n.Par != nil {
		pv := n.Par.Prop(key, inherit, typ)
		if pv != nil {
			return pv
		}
	}
	if typ {
		return kit.Types.Prop(n.Type(), key)
	}
	return nil
}

func (n *Node) DeleteProp(key string) {
	if n.Props == nil {
		return
	}
	delete(n.Props, key)
}

func (n *Node) DeleteAllProps(cap int) {
	if n.Props != nil {
		n.Props = make(Props, cap)
	}
}

func init() {
	gob.Register(Props{})
}

func (n *Node) CopyPropsFrom(from Ki, deep bool) error {
	if from.Properties() == nil {
		return nil
	}
	if n.Props == nil {
		n.Props = make(Props)
	}
	fmP := from.Properties()
	if deep {
		// code from https://gist.github.com/soroushjp/0ec92102641ddfc3ad5515ca76405f4d
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		dec := gob.NewDecoder(&buf)
		err := enc.Encode(fmP)
		if err != nil {
			return err
		}
		err = dec.Decode(&n.Props)
		if err != nil {
			return err
		}
		return nil
	} else {
		for k, v := range fmP {
			n.Props[k] = v
		}
	}
	return nil
}

//////////////////////////////////////////////////////////////////////////
//  Parent / Child Functionality

// set parent of node -- does not remove from existing parent -- use Add / Insert / Delete
func (n *Node) SetParent(parent Ki) {
	n.Par = parent
	if parent != nil && !parent.OnlySelfUpdate() {
		parup := parent.IsUpdating()
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			bitflag.SetState(k.Flags(), parup, int(Updating))
			return true
		})
	}
}

func (n *Node) IsRoot() bool {
	return (n.Par == nil || !n.Par.ThisOk() || n.This == nil) // extra safe
}

func (n *Node) Root() Ki {
	if n.IsRoot() {
		return n.This
	}
	return n.Par.Root()
}

func (n *Node) FieldRoot() Ki {
	var root Ki
	gotField := false
	n.FuncUpParent(0, n, func(k Ki, level int, d interface{}) bool {
		if !gotField {
			if k.IsField() {
				gotField = true
			}
			return true
		} else {
			if !k.IsField() {
				root = k
				return false
			}
		}
		return true
	})
	return root
}

func (n *Node) HasChildren() bool {
	return len(n.Kids) > 0
}

func (n *Node) Index() int {
	if n.Par == nil {
		return -1
	}
	n.index = n.Par.ChildIndex(n.This, n.index) // very fast if index is close..
	return n.index
}

func (n *Node) SetChildType(t reflect.Type) error {
	if !reflect.PtrTo(t).Implements(reflect.TypeOf((*Ki)(nil)).Elem()) {
		err := fmt.Errorf("Ki Node %v SetChildType: type does not implement the Ki interface -- must -- type passed is: %v", n.PathUnique(), t.Name())
		log.Print(err)
		return err
	}
	n.SetProp("ChildType", t)
	return nil
}

// check if it is safe to add child -- it cannot be a parent of us -- prevent loops!
func (n *Node) AddChildCheck(kid Ki) error {
	var err error
	n.FuncUp(0, n, func(k Ki, level int, d interface{}) bool {
		if k == kid {
			err = fmt.Errorf("Ki Node Attempt to add child to node %v that is my own parent -- no cycles permitted!\n", (d.(Ki)).PathUnique())
			log.Printf("%v", err)
			return false
		}
		return true
	})
	return err
}

// after adding child -- signals etc
func (n *Node) addChildImplPost(kid Ki) {
	oldPar := kid.Parent()
	kid.SetParent(n.This) // key to set new parent before deleting: indicates move instead of delete
	if oldPar != nil {
		oldPar.DeleteChild(kid, false)
		bitflag.Set(kid.Flags(), int(ChildMoved))
	} else {
		bitflag.Set(kid.Flags(), int(ChildAdded))
	}
}

func (n *Node) AddChildImpl(kid Ki) error {
	if err := n.ThisCheck(); err != nil {
		return err
	}
	if err := n.AddChildCheck(kid); err != nil {
		return err
	}
	kid.Init(kid)
	n.Kids = append(n.Kids, kid)
	n.addChildImplPost(kid)
	return nil
}

func (n *Node) InsertChildImpl(kid Ki, at int) error {
	if err := n.ThisCheck(); err != nil {
		return err
	}
	if err := n.AddChildCheck(kid); err != nil {
		return err
	}
	kid.Init(kid)
	n.Kids.Insert(kid, at)
	n.addChildImplPost(kid)
	return nil
}

func (n *Node) AddChild(kid Ki) error {
	updt := n.UpdateStart()
	err := n.AddChildImpl(kid)
	if err == nil {
		bitflag.Set(&n.Flag, int(ChildAdded))
		if kid.UniqueName() == "" {
			kid.SetUniqueName(kid.Name())
		}
		n.UniquifyNames()
	}
	n.UpdateEnd(updt)
	return err
}

func (n *Node) InsertChild(kid Ki, at int) error {
	updt := n.UpdateStart()
	err := n.InsertChildImpl(kid, at)
	if err == nil {
		bitflag.Set(&n.Flag, int(ChildAdded))
		if kid.UniqueName() == "" {
			kid.SetUniqueName(kid.Name())
		}
		n.UniquifyNames()
	}
	n.UpdateEnd(updt)
	return err
}

func (n *Node) NewOfType(typ reflect.Type) Ki {
	if err := n.ThisCheck(); err != nil {
		return nil
	}
	if typ == nil {
		ct := n.Prop("ChildType", false, true) // no inherit but yes from type
		if ct != nil {
			if ctt, ok := ct.(reflect.Type); ok {
				typ = ctt
			}
		}
	}
	if typ == nil {
		typ = n.Type() // make us by default
	}
	nkid := reflect.New(typ).Interface()
	kid, _ := nkid.(Ki)
	return kid
}

func (n *Node) AddNewChild(typ reflect.Type, name string) Ki {
	updt := n.UpdateStart()
	kid := n.NewOfType(typ)
	err := n.AddChildImpl(kid)
	if err == nil {
		kid.SetName(name)
		bitflag.Set(&n.Flag, int(ChildAdded))
	}
	n.UpdateEnd(updt)
	return kid
}

func (n *Node) InsertNewChild(typ reflect.Type, at int, name string) Ki {
	updt := n.UpdateStart()
	kid := n.NewOfType(typ)
	err := n.InsertChildImpl(kid, at)
	if err == nil {
		kid.SetName(name)
		bitflag.Set(&n.Flag, int(ChildAdded))
	}
	n.UpdateEnd(updt)
	return kid
}

func (n *Node) InsertNewChildUnique(typ reflect.Type, at int, name string) Ki {
	updt := n.UpdateStart()
	kid := n.NewOfType(typ)
	err := n.InsertChildImpl(kid, at)
	if err == nil {
		kid.SetNameRaw(name)
		kid.SetUniqueName(name)
		bitflag.Set(&n.Flag, int(ChildAdded))
	}
	n.UpdateEnd(updt)
	return kid
}

func (n *Node) MoveChild(from, to int) error {
	updt := n.UpdateStart()
	err := n.Kids.Move(from, to)
	if err == nil {
		bitflag.Set(&n.Flag, int(ChildMoved))
	}
	n.UpdateEnd(updt)
	return err
}

func (n *Node) SetNChildren(trgn int, typ reflect.Type, nameStub string) (mods, updt bool) {
	mods, updt = false, false
	sz := len(n.Kids)
	if trgn == sz {
		return
	}
	for sz > trgn {
		if !mods {
			mods = true
			updt = n.UpdateStart()
		}
		sz--
		n.DeleteChildAtIndex(sz, true)
	}
	for sz < trgn {
		if !mods {
			mods = true
			updt = n.UpdateStart()
		}
		nm := fmt.Sprintf("%v%v", nameStub, sz)
		n.InsertNewChildUnique(typ, sz, nm)
		sz++
	}
	return
}

func (n *Node) ConfigChildren(config kit.TypeAndNameList, uniqNm bool) (mods, updt bool) {
	return n.Kids.Config(n.This, config, uniqNm)
}

//////////////////////////////////////////////////////////////////////////
//  Find child / parent by..

func (n *Node) ChildIndexByFunc(startIdx int, match func(ki Ki) bool) int {
	return n.Kids.IndexByFunc(startIdx, match)
}

func (n *Node) ChildIndex(kid Ki, startIdx int) int {
	return n.Kids.Index(kid, startIdx)
}

func (n *Node) ChildIndexByName(name string, startIdx int) int {
	return n.Kids.IndexByName(name, startIdx)
}

func (n *Node) ChildIndexByUniqueName(name string, startIdx int) int {
	return n.Kids.IndexByUniqueName(name, startIdx)
}

func (n *Node) ChildIndexByType(t reflect.Type, embeds bool, startIdx int) int {
	return n.Kids.IndexByType(t, embeds, startIdx)
}

func (n *Node) ChildByName(name string, startIdx int) Ki {
	idx := n.Kids.IndexByName(name, startIdx)
	if idx < 0 {
		return nil
	}
	return n.Kids[idx]
}

func (n *Node) ChildByType(t reflect.Type, embeds bool, startIdx int) Ki {
	idx := n.Kids.IndexByType(t, embeds, startIdx)
	if idx < 0 {
		return nil
	}
	return n.Kids[idx]
}

func (n *Node) ParentByName(name string) Ki {
	if n.IsRoot() {
		return nil
	}
	if n.Par.Name() == name {
		return n.Par
	}
	return n.Par.ParentByName(name)
}

func (n *Node) ParentByType(t reflect.Type, embeds bool) Ki {
	if n.IsRoot() {
		return nil
	}
	if embeds {
		if n.Par.TypeEmbeds(t) {
			return n.Par
		}
	} else {
		if n.Par.Type() == t {
			return n.Par
		}
	}
	return n.Par.ParentByType(t, embeds)
}

func (n *Node) KiFieldByName(name string) Ki {
	v := reflect.ValueOf(n.This).Elem()
	f := v.FieldByName(name)
	if !f.IsValid() {
		return nil
	}
	if !kit.EmbeddedTypeImplements(f.Type(), KiType()) {
		return nil
	}
	return kit.PtrValue(f).Interface().(Ki)
}

//////////////////////////////////////////////////////////////////////////
//  Deleting

func (n *Node) DeleteChildAtIndex(idx int, destroy bool) {
	idx, err := n.Kids.ValidIndex(idx)
	if err != nil {
		log.Print("Ki Node DeleteChildAtIndex -- attempt to delete item in empty children slice")
		return
	}
	child := n.Kids[idx]
	updt := n.UpdateStart()
	bitflag.Set(&n.Flag, int(ChildDeleted))
	if child.Parent() == n.This {
		// only deleting if we are still parent -- change parent first to
		// signal move delete is always sent live to affected node without
		// update blocking note: children of child etc will not send a signal
		// at this point -- only later at destroy -- up to this parent to
		// manage all that
		bitflag.Set(child.Flags(), int(NodeDeleted))
		child.NodeSignal().Emit(child, int64(NodeSignalDeleting), nil)
		child.SetParent(nil)
	}
	_ = n.Kids.DeleteAtIndex(idx)
	if destroy {
		DelMgr.Add(child)
	}
	child.UpdateReset() // it won't get the UpdateEnd from us anymore -- init fresh in any case
	n.UpdateEnd(updt)
}

func (n *Node) DeleteChild(child Ki, destroy bool) {
	idx := n.ChildIndex(child, 0)
	if idx < 0 {
		return
	}
	n.DeleteChildAtIndex(idx, destroy)
}

func (n *Node) DeleteChildByName(name string, destroy bool) Ki {
	idx := n.ChildIndexByName(name, 0)
	if idx < 0 {
		return nil
	}
	child := n.Kids[idx]
	n.DeleteChildAtIndex(idx, destroy)
	return child
}

func (n *Node) DeleteChildren(destroy bool) {
	updt := n.UpdateStart()
	bitflag.Set(&n.Flag, int(ChildrenDeleted))
	for _, child := range n.Kids {
		bitflag.Set(child.Flags(), int(NodeDeleted))
		child.NodeSignal().Emit(child, int64(NodeSignalDeleting), nil)
		child.SetParent(nil)
		child.UpdateReset()
	}
	if destroy {
		DelMgr.Add(n.Kids...)
	}
	n.Kids = n.Kids[:0] // preserves capacity of list
	n.UpdateEnd(updt)
}

func (n *Node) Delete(destroy bool) {
	if n.Par == nil {
		if destroy {
			n.Destroy()
		}
	} else {
		n.Par.DeleteChild(n.This, destroy)
	}
}

func (n *Node) Destroy() {
	// fmt.Printf("Destroying: %v %T %p Kids: %v\n", n.PathUnique(), n.This, n.This, len(n.Kids))
	if n.This == nil { // already dead!
		return
	}
	n.NodeSig.Emit(n.This, int64(NodeSignalDestroying), nil)
	bitflag.Set(&n.Flag, int(NodeDestroyed))
	n.DisconnectAll()
	n.DeleteChildren(true) // first delete all my children
	// and destroy all my fields
	n.FuncFields(0, nil, func(k Ki, level int, d interface{}) bool {
		k.Destroy()
		return true
	})
	DelMgr.DestroyDeleted() // then destroy all those kids
	// extra step to delete all the slices and maps -- super friendly to GC :)
	FlatFieldsValueFunc(n.This, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Map {
			fieldVal.Set(reflect.Zero(fieldVal.Type())) // set to nil
		}
		return true
	})
	n.This = nil // last gasp: lose our own sense of self..
}

//////////////////////////////////////////////////////////////////////////
//  Tree walking and state updating

func (n *Node) Fields() []uintptr {
	// we store the offsets for the fields in type properties
	tprops := kit.Types.Properties(n.Type(), true) // true = makeNew
	pnm := "__FieldOffs"
	if foff, ok := tprops[pnm]; ok {
		return foff.([]uintptr)
	}
	foff := make([]uintptr, 0)
	kitype := KiType()
	FlatFieldsValueFunc(n.This, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		if fieldVal.Kind() == reflect.Struct && kit.EmbeddedTypeImplements(field.Type, kitype) {
			foff = append(foff, field.Offset)
		}
		return true
	})
	tprops[pnm] = foff
	return foff
}

// Node version of this function from kit/embeds.go
func FlatFieldsValueFunc(stru interface{}, fun func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool) bool {
	v := kit.NonPtrValue(reflect.ValueOf(stru))
	typ := v.Type()
	if typ == KiT_Node { // this is only diff from embeds.go version -- prevent processing of any Node fields
		return true
	}
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		vf := v.Field(i)
		if !vf.CanInterface() {
			continue
		}
		vfi := vf.Interface() // todo: check for interfaceablity etc
		if vfi == nil || vfi == stru {
			continue
		}
		if f.Type.Kind() == reflect.Struct && f.Anonymous && kit.PtrType(f.Type) != KiT_Node {
			rval = FlatFieldsValueFunc(kit.PtrValue(vf).Interface(), fun)
			if !rval {
				break
			}
		} else {
			rval = fun(vfi, typ, f, vf)
			if !rval {
				break
			}
		}
	}
	return rval
}

func (n *Node) FuncFields(level int, data interface{}, fun Func) {
	op := reflect.ValueOf(n.This).Pointer()
	foffs := n.Fields()
	for _, fo := range foffs {
		fn := (*Node)(unsafe.Pointer(op + fo))
		fun(fn.This, level, data)
	}
}

func (n *Node) GoFuncFields(level int, data interface{}, fun Func) {
	op := reflect.ValueOf(n.This).Pointer()
	foffs := n.Fields()
	for _, fo := range foffs {
		fn := (*Node)(unsafe.Pointer(op + fo))
		go fun(fn.This, level, data)
	}
}

func (n *Node) FuncUp(level int, data interface{}, fun Func) bool {
	if !fun(n.This, level, data) { // false return means stop
		return false
	}
	level++
	if n.Parent() != nil && n.Parent() != n.This { // prevent loops
		return n.Parent().FuncUp(level, data, fun)
	}
	return true
}

func (n *Node) FuncUpParent(level int, data interface{}, fun Func) bool {
	if n.IsRoot() {
		return true
	}
	if !fun(n.Parent(), level, data) { // false return means stop
		return false
	}
	level++
	return n.Parent().FuncUpParent(level, data, fun)
}

func (n *Node) FuncDownMeFirst(level int, data interface{}, fun Func) bool {
	if !fun(n.This, level, data) { // false return means stop
		return false
	}
	level++
	n.FuncFields(level, data, func(k Ki, level int, d interface{}) bool {
		k.FuncDownMeFirst(level, data, fun)
		return true
	})
	for _, child := range n.Children() {
		child.FuncDownMeFirst(level, data, fun) // don't care about their return values
	}
	return true
}

func (n *Node) FuncDownDepthFirst(level int, data interface{}, doChildTestFunc Func, fun Func) {
	level++
	for _, child := range n.Children() {
		if doChildTestFunc(n.This, level, data) { // test if we should run on this child
			child.FuncDownDepthFirst(level, data, doChildTestFunc, fun)
		}
	}
	n.FuncFields(level, data, func(k Ki, level int, d interface{}) bool {
		if doChildTestFunc(k, level, data) { // test if we should run on this child
			k.FuncDownDepthFirst(level, data, doChildTestFunc, fun)
		}
		fun(k, level, data)
		return true
	})
	level--
	fun(n.This, level, data) // can't use the return value at this point
}

func (n *Node) FuncDownBreadthFirst(level int, data interface{}, fun Func) {
	dontMap := make(map[int]bool) // map of who NOT to process further -- default is false for map so reverse
	level++
	for i, child := range n.Children() {
		if !fun(child, level, data) { // false return means stop
			dontMap[i] = true
		} else {
			child.FuncFields(level+1, data, func(k Ki, level int, d interface{}) bool {
				k.FuncDownBreadthFirst(level+1, data, fun)
				fun(k, level+1, data)
				return true
			})
		}
	}
	for i, child := range n.Children() {
		if dontMap[i] {
			continue
		}
		child.FuncDownBreadthFirst(level, data, fun)
	}
}

func (n *Node) GoFuncDown(level int, data interface{}, fun Func) {
	go fun(n.This, level, data)
	level++
	n.GoFuncFields(level, data, fun)
	for _, child := range n.Children() {
		child.GoFuncDown(level, data, fun)
	}
}

func (n *Node) GoFuncDownWait(level int, data interface{}, fun Func) {
	// todo: use channel or something to wait
	go fun(n.This, level, data)
	level++
	n.GoFuncFields(level, data, fun)
	for _, child := range n.Children() {
		child.GoFuncDown(level, data, fun)
	}
}

func (n *Node) Path() string {
	if n.Par != nil {
		if n.IsField() {
			return n.Par.Path() + "." + n.Nm
		} else {
			return n.Par.Path() + "/" + n.Nm
		}
	}
	return "/" + n.Nm
}

func (n *Node) PathUnique() string {
	if n.Par != nil {
		if n.IsField() {
			return n.Par.PathUnique() + "." + n.UniqueNm
		} else {
			return n.Par.PathUnique() + "/" + n.UniqueNm
		}
	}
	return "/" + n.UniqueNm
}

func (n *Node) PathFrom(par Ki) string {
	if n.Par != nil && n.Par != par {
		if n.IsField() {
			return n.Par.PathFrom(par) + "." + n.Nm
		} else {
			return n.Par.PathFrom(par) + "/" + n.Nm
		}
	}
	return "/" + n.Nm
}

func (n *Node) PathFromUnique(par Ki) string {
	if n.Par != nil && n.Par != par {
		if n.IsField() {
			return n.Par.PathFromUnique(par) + "." + n.Nm
		} else {
			return n.Par.PathFromUnique(par) + "/" + n.Nm
		}
	}
	return "/" + n.Nm
}

func (n *Node) FindPathUnique(path string) Ki {
	curn := Ki(n)
	pels := strings.Split(strings.Trim(strings.TrimSpace(path), "\""), "/")
	for i, pe := range pels {
		if len(pe) == 0 {
			continue
		}
		if i <= 1 && curn.UniqueName() == pe {
			continue
		}
		if strings.Contains(pe, ".") { // has fields
			fels := strings.Split(pe, ".")
			// find the child first, then the fields
			idx := curn.ChildIndexByUniqueName(fels[0], 0)
			if idx < 0 {
				return nil
			}
			curn = curn.Children()[idx]
			for i := 1; i < len(fels); i++ {
				fe := fels[i]
				fk := curn.KiFieldByName(fe)
				if fk == nil {
					return nil
				}
				curn = fk
			}
		} else {
			idx := curn.ChildIndexByUniqueName(pe, 0)
			if idx < 0 {
				return nil
			}
			curn = curn.Children()[idx]
		}
	}
	return curn
}

//////////////////////////////////////////////////////////////////////////
//  State update signaling -- automatically consolidates all changes across
//   levels so there is only one update at highest level of modification
//   All modification starts with UpdateStart() and ends with UpdateEnd()

// after an UpdateEnd, DestroyDeleted is called

func (n *Node) NodeSignal() *Signal {
	return &n.NodeSig
}

func (n *Node) UpdateStart() bool {
	if n.IsUpdatingMu() {
		return false
	}
	if bitflag.Has(n.Flag, int(NodeDestroyed)) {
		return false
	}
	if n.OnlySelfUpdate() {
		n.SetFlagMu(int(Updating))
	} else {
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			if !k.IsUpdating() {
				bitflag.ClearMask(k.Flags(), int64(UpdateFlagsMask))
				k.SetFlagMu(int(Updating))
				return true // keep going down
			} else {
				return false // bail -- already updating
			}
		})
	}
	return true
}

func (n *Node) UpdateEnd(updt bool) {
	if !updt {
		return
	}
	if n.IsDestroyed() || n.IsDeleted() {
		return
	}
	if bitflag.HasAny(n.Flag, int(ChildDeleted), int(ChildrenDeleted)) {
		DelMgr.DestroyDeleted()
	}
	if n.OnlySelfUpdate() {
		n.ClearFlagMu(int(Updating))
		n.NodeSignal().Emit(n.This, int64(NodeSignalUpdated), n.Flag)
	} else {
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlagMu(int(Updating)) // todo: could check first and break here but good to ensure all clear
			return true
		})
		n.NodeSignal().Emit(n.This, int64(NodeSignalUpdated), n.Flag)
	}
}

func (n *Node) UpdateEndNoSig(updt bool) {
	if !updt {
		return
	}
	if n.IsDestroyed() || n.IsDeleted() {
		return
	}
	if bitflag.HasAny(n.Flag, int(ChildDeleted), int(ChildrenDeleted)) {
		DelMgr.DestroyDeleted()
	}
	if n.OnlySelfUpdate() {
		n.ClearFlagMu(int(Updating))
		// n.NodeSignal().Emit(n.This, int64(NodeSignalUpdated), n.Flag)
	} else {
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlagMu(int(Updating)) // todo: could check first and break here but good to ensure all clear
			return true
		})
		// n.NodeSignal().Emit(n.This, int64(NodeSignalUpdated), n.Flag)
	}
}

func (n *Node) UpdateSig() bool {
	if n.IsUpdatingMu() {
		return false
	}
	if bitflag.Has(n.Flag, int(NodeDestroyed)) {
		return false
	}
	n.NodeSignal().Emit(n.This, int64(NodeSignalUpdated), n.Flag)
	return true
}

func (n *Node) UpdateReset() {
	if n.OnlySelfUpdate() {
		n.ClearFlagMu(int(Updating))
	} else {
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlagMu(int(Updating))
			return true
		})
	}
}

func (n *Node) Disconnect() {
	n.NodeSig.DisconnectAll()
	FlatFieldsValueFunc(n.This, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		switch {
		case fieldVal.Kind() == reflect.Interface:
			if field.Name != "This" { // reserve that for last step in Destroy
				fieldVal.Set(reflect.Zero(fieldVal.Type())) // set to nil
			}
		case fieldVal.Kind() == reflect.Ptr:
			fieldVal.Set(reflect.Zero(fieldVal.Type())) // set to nil
		case fieldVal.Type() == KiT_Signal:
			if fs, ok := kit.PtrValue(fieldVal).Interface().(*Signal); ok {
				// fmt.Printf("ki.Node: %v Type: %T Disconnecting signal field: %v\n", n.Name(), n.This, field.Name)
				fs.DisconnectAll()
			}
		case fieldVal.Type() == KiT_Ptr:
			if pt, ok := kit.PtrValue(fieldVal).Interface().(*Ptr); ok {
				pt.Reset()
			}
		}
		return true
	})
}

func (n *Node) DisconnectAll() {
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		k.Disconnect()
		return true
	})
}

//////////////////////////////////////////////////////////////////////////
//  Field Value setting with notification

func (n *Node) SetField(field string, val interface{}) bool {
	fv := kit.FlatFieldValueByName(n.This, field)
	if !fv.IsValid() {
		log.Printf("ki.SetField, could not find field %v on node %v\n", field, n.PathUnique())
		return false
	}
	updt := n.UpdateStart()
	ok := kit.SetRobust(kit.PtrValue(fv).Interface(), val)
	if ok {
		bitflag.Set(n.Flags(), int(FieldUpdated))
	}
	n.UpdateEnd(updt)
	return ok
}

func (n *Node) SetFieldDown(field string, val interface{}) {
	updt := n.UpdateStart()
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		k.SetField(field, val)
		return true
	})
	n.UpdateEnd(updt)
}

func (n *Node) SetFieldUp(field string, val interface{}) {
	updt := n.UpdateStart()
	n.FuncUp(0, nil, func(k Ki, level int, d interface{}) bool {
		k.SetField(field, val)
		return true
	})
	n.UpdateEnd(updt)
}

func (n *Node) FieldByName(field string) interface{} {
	return kit.FlatFieldInterfaceByName(n.This, field)
}

func (n *Node) FieldTag(field, tag string) string {
	return kit.FlatFieldTag(n.Type(), field, tag)
}

//////////////////////////////////////////////////////////////////////////
//  Deep Copy / Clone

// note: we use the copy from direction as the receiver is modifed whereas the
// from is not and assignment is typically in same direction

func (n *Node) CopyFrom(from Ki) error {
	if from == nil {
		err := fmt.Errorf("Ki Node CopyFrom into %v -- null 'from' source\n", n.PathUnique())
		log.Print(err)
		return err
	}
	mypath := n.PathUnique()
	fmpath := from.PathUnique()
	if n.Type() != from.Type() {
		err := fmt.Errorf("Ki Node Copy to %v from %v -- must have same types, but %v != %v\n", mypath, fmpath, n.Type().Name(), from.Type().Name())
		log.Print(err)
		return err
	}
	updt := n.UpdateStart()
	bitflag.Set(&n.Flag, int(NodeCopied))
	sameTree := (n.Root() == from.Root())
	from.GetPtrPaths()
	err := n.CopyFromRaw(from)
	DelMgr.DestroyDeleted() // in case we deleted some kiddos
	if err != nil {
		n.UpdateEnd(updt)
		return err
	}
	if sameTree {
		n.UpdatePtrPaths(fmpath, mypath, true)
	}
	n.SetPtrsFmPaths()
	n.UpdateEnd(updt)
	return nil
}

func (n *Node) Clone() Ki {
	nki := n.NewOfType(n.Type())
	nki.InitName(nki, n.Nm)
	nki.CopyFrom(n.This)
	return nki
}

// use ConfigChildren to recreate source children
func (n *Node) CopyMakeChildrenFrom(from Ki) {
	sz := len(from.Children())
	if sz > 0 {
		cfg := make(kit.TypeAndNameList, sz)
		for i, kid := range from.Children() {
			cfg[i].Type = kid.Type()
			cfg[i].Name = kid.UniqueName() // use unique so guaranteed to have something
		}
		mods, updt := n.ConfigChildren(cfg, true) // use unique names -- this means name = uniquname
		for i, kid := range from.Children() {
			mkid := n.Kids[i]
			mkid.SetNameRaw(kid.Name()) // restore orig user-names
		}
		if mods {
			n.UpdateEnd(updt)
		}
	} else {
		n.DeleteChildren(true)
	}
}

// copy from primary fields of from to to, recursively following anonymous embedded structs
func (n *Node) CopyFieldsFrom(to interface{}, from interface{}) {
	kitype := KiType()
	tv := kit.NonPtrValue(reflect.ValueOf(to))
	sv := kit.NonPtrValue(reflect.ValueOf(from))
	typ := tv.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		tf := tv.Field(i)
		if !tf.CanInterface() {
			continue
		}
		ctag := f.Tag.Get("copy")
		if ctag == "-" {
			continue
		}
		sf := sv.Field(i)
		tfpi := kit.PtrValue(tf).Interface()
		sfpi := kit.PtrValue(sf).Interface()
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			n.CopyFieldsFrom(tfpi, sfpi)
		} else {
			switch {
			case sf.Kind() == reflect.Struct && kit.EmbeddedTypeImplements(sf.Type(), kitype):
				sfk := sfpi.(Ki)
				tfk := tfpi.(Ki)
				if tfk != nil && sfk != nil {
					tfk.CopyFrom(sfk)
				}
			case f.Type == KiT_Signal: // todo: don't copy signals by default
			case sf.Type().AssignableTo(tf.Type()):
				tf.Set(sf)
				// kit.PtrValue(tf).Set(sf)
			default:
				// use copier https://github.com/jinzhu/copier which handles as much as possible..
				copier.Copy(tfpi, sfpi)
			}
		}

	}
}

func (n *Node) CopyFromRaw(from Ki) error {
	n.CopyMakeChildrenFrom(from)
	n.DeleteAllProps(len(from.Properties())) // start off fresh, allocated to size of from
	n.CopyPropsFrom(from, false)             // use shallow props copy by default
	n.CopyFieldsFrom(n.This, from)
	for i, kid := range n.Kids {
		fmk := from.Children()[i]
		kid.CopyFromRaw(fmk)
	}
	return nil
}

func (n *Node) GetPtrPaths() {
	root := n.This
	n.FuncDownMeFirst(0, root, func(k Ki, level int, d interface{}) bool {
		FlatFieldsValueFunc(k, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			if fieldVal.CanInterface() {
				vfi := kit.PtrValue(fieldVal).Interface()
				switch vfv := vfi.(type) {
				case *Ptr:
					vfv.GetPath()
					// case *Signal:
					// 	vfv.GetPaths()
				}
			}
			return true
		})
		return true
	})
}

func (n *Node) SetPtrsFmPaths() {
	root := n.Root()
	n.FuncDownMeFirst(0, root, func(k Ki, level int, d interface{}) bool {
		FlatFieldsValueFunc(k, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			if fieldVal.CanInterface() {
				vfi := kit.PtrValue(fieldVal).Interface()
				switch vfv := vfi.(type) {
				case *Ptr:
					if !vfv.PtrFmPath(root) {
						log.Printf("Ki Node SetPtrsFmPaths: could not find path: %v in root obj: %v", vfv.Path, root.Name())
					}
				}
			}
			return true
		})
		return true
	})
}

func (n *Node) UpdatePtrPaths(oldPath, newPath string, startOnly bool) {
	root := n.Root()
	n.FuncDownMeFirst(0, root, func(k Ki, level int, d interface{}) bool {
		FlatFieldsValueFunc(k, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			if fieldVal.CanInterface() {
				vfi := kit.PtrValue(fieldVal).Interface()
				switch vfv := vfi.(type) {
				case *Ptr:
					vfv.UpdatePath(oldPath, newPath, startOnly)
				}
			}
			return true
		})
		return true
	})
}

//////////////////////////////////////////////////////////////////////////
//  IO Marshal / Unmarshal support -- mostly in Slice

func (n *Node) SaveJSON(indent bool) ([]byte, error) {
	if err := n.ThisCheck(); err != nil {
		return nil, err
	}
	if indent {
		return json.MarshalIndent(n.This, "", "  ")
	} else {
		return json.Marshal(n.This)
	}
}

func (n *Node) SaveJSONToFile(filename string) error {
	b, err := n.SaveJSON(true) // use indent by default
	if err != nil {
		log.Println(err)
		fmt.Println(b)
		return err
	}
	err = ioutil.WriteFile(filename, b, 0644) // todo: permissions??
	if err != nil {
		log.Println(err)
	}
	return err
}

func (n *Node) LoadJSON(b []byte) error {
	var err error
	if err = n.ThisCheck(); err != nil {
		return err
		log.Println(err)
	}
	updt := n.UpdateStart()
	err = json.Unmarshal(b, n.This) // key use of this!
	if err == nil {
		n.UnmarshalPost()
	}
	bitflag.Set(&n.Flag, int(ChildAdded)) // this might not be set..
	n.UpdateEnd(updt)
	return err
}

func (n *Node) LoadJSONFromFile(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	return n.LoadJSON(b)
}

func (n *Node) SaveXML(indent bool) ([]byte, error) {
	if err := n.ThisCheck(); err != nil {
		log.Println(err)
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
		log.Println(err)
		return err
	}
	updt := n.UpdateStart()
	err = xml.Unmarshal(b, n.This) // key use of this!
	if err == nil {
		n.UnmarshalPost()
	}
	n.UpdateEnd(updt)
	return nil
}

func (n *Node) ParentAllChildren() {
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		for _, child := range k.Children() {
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

// Deleted manages all the deleted Ki elements, that are destined to then be
// destroyed, without having an additional pointer on the Ki object
type Deleted struct {
	Dels []Ki
	Mu   sync.Mutex
}

// DelMgr is the manager of all deleted items
var DelMgr = Deleted{}

// Add the Ki elements to the deleted list
func (dm *Deleted) Add(kis ...Ki) {
	dm.Mu.Lock()
	if dm.Dels == nil {
		dm.Dels = make([]Ki, 0, 1000)
	}
	dm.Dels = append(dm.Dels, kis...)
	dm.Mu.Unlock()
}

func (dm *Deleted) DestroyDeleted() {
	dm.Mu.Lock()
	curdels := make([]Ki, len(dm.Dels))
	copy(curdels, dm.Dels)
	dm.Dels = dm.Dels[:0]
	dm.Mu.Unlock()
	for _, ki := range curdels {
		ki.Destroy() // destroy will add to the dels so we need to do this outside of lock
	}
}
