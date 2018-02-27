// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English
package goki

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

func (node *Node) SetParent(parent *Node) {
	if node.Parent != nil {
		node.Parent.RemoveChild(node, false)
	}
	node.Parent = parent
}

func (node *Node) AddChildren(kids ...*Node) {
	for _, kid := range kids {
		kid.SetParent(node)
	}
	node.Children = append(node.Children, kids...)
}

// find index of child -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
func (node *Node) FindChildIndex(kid *Node, start_idx int) int {
	if start_idx == 0 {
		for idx, child := range node.Children {
			if child == kid {
				return idx
			}
		}
	} else {
		upi := start_idx + 1
		dni := start_idx
		upo := false
		sz := len(node.Children)
		for {
			if !upo && upi < sz {
				if node.Children[upi] == kid {
					return upi
				}
				upi++
			} else {
				upo = true
			}
			if dni >= 0 {
				if node.Children[dni] == kid {
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

// find index of child -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
func (node *Node) FindChildNameIndex(name string, start_idx int) int {
	if start_idx == 0 {
		for idx, child := range node.Children {
			if child.Name == name {
				return idx
			}
		}
	} else {
		upi := start_idx + 1
		dni := start_idx
		upo := false
		sz := len(node.Children)
		for {
			if !upo && upi < sz {
				if node.Children[upi].Name == name {
					return upi
				}
				upi++
			} else {
				upo = true
			}
			if dni >= 0 {
				if node.Children[dni].Name == name {
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

func (node *Node) RemoveChildIndex(idx int, destroy bool) {
	child := node.Children[idx]
	// this copy makes sure there are no memory leaks
	copy(node.Children[idx:], node.Children[idx+1:])
	node.Children[len(node.Children)-1] = nil
	node.Children = node.Children[:len(node.Children)-1]
	child.SetParent(nil)
	if destroy {
		node.deleted = append(node.deleted, child)
	}
}

// Remove child node -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
func (node *Node) RemoveChild(child *Node, destroy bool) {
	idx := node.FindChildIndex(child, 0)
	if idx < 0 {
		return
	}
	node.RemoveChildIndex(idx, destroy)
}

// Remove child node by name -- returns child -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
func (node *Node) RemoveChildName(name string, destroy bool) *Node {
	idx := node.FindChildNameIndex(name, 0)
	if idx < 0 {
		return nil
	}
	child := node.Children[idx]
	node.RemoveChildIndex(idx, destroy)
	return child
}

// Remove all children nodes -- destroy will add removed children to deleted list, to be destroyed later -- otherwise children remain intact but parent is nil -- could be inserted elsewhere, but you better have kept a slice of them before calling this
func (node *Node) RemoveAllChildren(destroy bool) {
	for _, child := range node.Children {
		child.SetParent(nil)
	}
	if destroy {
		node.deleted = append(node.deleted, node.Children...)
	}
	node.Children = node.Children[:0]
}

// second-pass actually delete all previously-removed children: causes them to remove all their children and then destroy them
func (node *Node) DestroyDeleted() {
	for _, child := range node.deleted {
		child.DestroyNode()
	}
	node.deleted = node.deleted[:0]
}

// remove all children and their childrens-children, etc
func (node *Node) DestroyNode() {
	for _, child := range node.Children {
		child.DestroyNode()
	}
	node.RemoveAllChildren(true)
	node.DestroyDeleted()
}

// todo: paths, notifications
// github.com/tucnak/meta has signal / slot impl -- doesn't use reflect though
