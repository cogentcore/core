// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English
package ki

import (
	// "encoding/json"
	"errors"
	"fmt"
	"github.com/cznic/mathutil"
	"reflect"
	"strconv"
	"strings"
)

/*
The Node implements the Ki interface and provides the core functionality for the GoKi Tree functionality -- insipred by Qt QObject in specific and every other Tree everywhere in general -- provides core functionality:
* Parent / Child Tree structure -- each Node can ONLY have one parent
* Paths for locating Nodes within the hierarchy -- key for many use-cases, including IO for pointers
* Generalized I/O -- can Save and Load the Tree as JSON, XML, etc
* Event sending and receiving between Nodes (simlar to Qt Signals / Slots)
*/

type Node struct {
	Name       string
	UniqueName string
	Properties []string
	Parent     Ki `json:"-"`
	Children   []Ki
	ChildType  reflect.Type

	// keep track of deleted items until truly done with them
	deleted []Ki
}

func NewNode() *Node {
	return &Node{}
}

// Kier interface
func (n *Node) Ki() Ki {
	return n
}

// Return a pointer to the supplied Node struct via Ki -- from https://groups.google.com/forum/#!msg/Golang-Nuts/KB3_Yj3Ny4c/Ai8tz-nkBwAJ -- see InterfaceToStructPtr in ki.go for more generic version
// func KiToNodePtr(ki Ki) (*Node, error) {
// 	vp := reflect.New(reflect.TypeOf(ki))
// 	vp.Elem().Set(reflect.ValueOf(ki))
// 	rval, ok := vp.Interface().(*Node)
// 	if !ok {
// 		rval, ok := vp.Interface().(**Node) // maybe we got a double-pointer
// 		if !ok {
// 			return nil, fmt.Errorf("KiToNodePtr: Ki was not a Node type, is: %T", ki)
// 		}
// 		return *rval, nil
// 	}
// 	return rval, nil
// }

// above is unnec -- just convert directly..
func KiToNodePtr(ki Ki) (*Node, error) {
	rval, ok := ki.(*Node)
	if !ok {
		// rval, ok := ki.(**Node) // maybe we got a double-pointer
		// if !ok {
		return nil, fmt.Errorf("KiToNodePtr: Ki was not a Node type, is: %T", ki)
		// }
		// return *rval, nil
	}
	return rval, nil
}

// Return a pointer to the supplied Node struct via Ki -- from https://groups.google.com/forum/#!msg/Golang-Nuts/KB3_Yj3Ny4c/Ai8tz-nkBwAJ -- see InterfaceToStructPtr in ki.go for more generic version
func ObjToNodePtr(obj interface{}) (*Node, error) {
	vp := reflect.New(reflect.TypeOf(obj))
	vp.Elem().Set(reflect.ValueOf(obj))
	rval, ok := vp.Interface().(*Node)
	if !ok {
		// rval, ok := vp.Interface().(**Node) // maybe we got a double-pointer
		// if !ok {
		return nil, fmt.Errorf("KiToNodePtr: Obj was not a Node type, is: %T", obj)
		// }
		// return *rval, nil
	}
	return rval, nil
}

//////////////////////////////////////////////////////////////////////////
//  Basic Ki properties

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

func (n *Node) KiChildren() []Ki {
	kids := make([]Ki, len(n.Children))
	for i, child := range n.Children {
		kids[i] = child
	}
	return kids
}

func (n *Node) KiName() string {
	return n.Name
}

func (n *Node) KiUniqueName() string {
	return n.UniqueName
}

func (n *Node) KiProperties() []string {
	return n.Properties
}

//////////////////////////////////////////////////////////////////////////
//  Parent / Child Functionality

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

// set parent of node -- if parent is already set, then removes from that parent first -- nodes can ONLY have one parent -- only for true Tree structures, not DAG's or other such graphs that do not enforce a strict single-parent relationship
func (n *Node) SetParent(parent Ki) {
	if n.Parent != nil {
		n.Parent.RemoveChild(n, false)
	}
	n.Parent = parent
}

func (n *Node) SetChildType(t reflect.Type) error {
	// var tst Ki = &Node{}
	// if !t.Implements(reflect.TypeOf(tst)) {
	// 	return fmt.Errorf("Node SetChildType: type does not implement the Ki interface -- must -- type passed is: %v", t.Name())
	// }
	n.ChildType = t
	return nil
}

func (n *Node) AddChild(kid Ki) {
	n.Children = append(n.Children, kid)
	kid.SetParent(n)
}

func (n *Node) InsertChild(kid Ki, at int) {
	at = mathutil.Min(at, len(n.Children))
	// this avoids extra garbage collection
	n.Children = append(n.Children, nil)
	copy(n.Children[at+1:], n.Children[at:])
	n.Children[at] = kid
	kid.SetParent(n)
}

func (n *Node) AddChildNamed(kid Ki, name string) {
	n.AddChild(kid)
	kid.SetName(name)
}

func (n *Node) InsertChildNamed(kid Ki, at int, name string) {
	n.InsertChild(kid, at)
	kid.SetName(name)
}

func (n *Node) AddNewChild() (Ki, error) {
	if n.ChildType == nil {
		return nil, errors.New("Node AddNewChild: ChildType not set -- must set first")
	}
	nkid := reflect.New(n.ChildType).Interface()
	// fmt.Printf("nkid is new obj of type %T val: %+v\n", nkid, nkid)
	kid, ok := nkid.(Ki)
	if !ok {
		return nil, errors.New("Node AddNewChild: ChildType cannot convert to Ki")
	}
	// fmt.Printf("kid is new obj of type %T val: %+v\n", kid, kid)
	n.AddChild(kid)
	return kid, nil
}

func (n *Node) InsertNewChild(at int) (Ki, error) {
	if n.ChildType == nil {
		return nil, errors.New("Node InsertNewChild: ChildType not set -- must set first")
	}
	nkid := reflect.New(n.ChildType).Interface()
	// fmt.Printf("nkid is new obj of type %T val: %+v\n", nkid, nkid)
	kid, ok := nkid.(Ki)
	if !ok {
		return nil, errors.New("Node AddNewChild: ChildType cannot convert to Ki")
	}
	n.InsertChild(kid, at)
	return kid, nil
}

func (n *Node) AddNewChildNamed(name string) (Ki, error) {
	kid, err := n.AddNewChild()
	if err != nil {
		return nil, err
	}
	kid.SetName(name)
	return kid, err
}

func (n *Node) InsertNewChildNamed(at int, name string) (Ki, error) {
	kid, err := n.InsertNewChild(at)
	if err != nil {
		return nil, err
	}
	kid.SetName(name)
	return kid, err
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

// Remove all children nodes -- destroy will add removed children to deleted list, to be destroyed later -- otherwise children remain intact but parent is nil -- could be inserted elsewhere, but you better have kept a slice of them before calling this
func (n *Node) RemoveAllChildren(destroy bool) {
	for _, child := range n.Children {
		child.SetParent(nil)
	}
	if destroy {
		n.deleted = append(n.deleted, n.Children...)
	}
	n.Children = n.Children[:0]
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

func (n *Node) IsTop() bool {
	return n.Parent == nil
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

///////////////////////////////////////////////////
//  JSON marshaling

// json tags allow parent to be omitted!

// for example code on running unmarshall:
// http://gregtrowbridge.com/golang-json-serialization-with-interfaces/

// func (n *Node) MarshalJSON() ([]byte, error) {
// 	// must exclude parent and deal with children!
// 	totlen := 0
// 	fi := 0
// 	t := reflect.TypeOf(n)
// 	b := make([]byte, 0, 250)
// 	for i := 0; i < t.NumField(); i++ {
// 		f := t.Field(i)
// 		// fmt.Printf("%d: %s %s = %v\n", i,
// 		// 	f.Name, f.Type(), f.Interface())
// 		if f.Name != "Parent" {
// 			fb, err := json.Marshal(f.Type.Value.Interface())
// 			if fi == 0 {
// 				b = append(b, []byte("\"")...)
// 			} else {
// 				b = append(b, []byte(",\"")...)
// 			}
// 			b = append(b, []byte(f.Name)...)
// 			b = append(b, []byte("\":")...)
// 			b = append(b, fb...)
// 		}
// 	}
// 	return b, err
// }

//////////////////////////////////////////////////////////////////////////
//  Signal / Slot Functionality

// todo: look at QObject interface, also qtquick 2 scenegraph methods

// todo: paths, notifications
// github.com/tucnak/meta has signal / slot impl -- doesn't use reflect though
