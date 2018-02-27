// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English
package ki

import ()

// Default implementation of a Ki node -- use this as embedded element in other structs
type Node struct {
	Parent     *Node
	Children   []*Node
	Name       string
	UniqueName string
	Properties []string // todo: map

	// keep track of deleted items until truly done with them
	deleted []*Node
}

// not really clear what use interface is at this point

// func (n Node) KiParent() Ki {
// 	return n.Parent;
// }

// func (n Node) KiChildren() []Ki {
// 	return n.Parent;
// }

// func (n Node) KiName() string {
// 	return n.Name;
// }

// func (n Node) KiUniqueName() string {
// 	return n.UniqueName;
// }

// func (n Node) KiProperties() []string {
// 	return n.Properties;
// }

func NewNode() *Node {
	return &Node{}
}

func (n *Node) SetParent(parent *Node) {
	if n.Parent != nil {
		n.Parent.RemoveChild(n, false)
	}
	n.Parent = parent
}

func (n *Node) AddChildren(kids ...*Node) {
	for _, kid := range kids {
		kid.SetParent(n)
	}
	n.Children = append(n.Children, kids...)
}

// find index of child -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
func (n *Node) FindChildIndex(kid *Node, start_idx int) int {
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
			if child.Name == name {
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
				if n.Children[upi].Name == name {
					return upi
				}
				upi++
			} else {
				upo = true
			}
			if dni >= 0 {
				if n.Children[dni].Name == name {
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
func (n *Node) FindChildName(name string, start_idx int) *Node {
	idx := n.FindChildNameIndex(name, start_idx)
	if idx < 0 {
		return nil
	}
	return n.Children[idx]
}

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
func (n *Node) RemoveChild(child *Node, destroy bool) {
	idx := n.FindChildIndex(child, 0)
	if idx < 0 {
		return
	}
	n.RemoveChildIndex(idx, destroy)
}

// Remove child node by name -- returns child -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
func (n *Node) RemoveChildName(name string, destroy bool) *Node {
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
		child.DestroyNode()
	}
	n.deleted = n.deleted[:0]
}

// remove all children and their childrens-children, etc
func (n *Node) DestroyNode() {
	for _, child := range n.Children {
		child.DestroyNode()
	}
	n.RemoveAllChildren(true)
	n.DestroyDeleted()
}

// todo: paths, notifications
// github.com/tucnak/meta has signal / slot impl -- doesn't use reflect though
