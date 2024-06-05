// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"fmt"
	"slices"

	"cogentcore.org/core/base/findfast"
	"cogentcore.org/core/types"
)

// Slice is just a slice of tree nodes: []Node, providing methods for accessing
// elements in the slice, and JSON marshal / unmarshal with encoding of
// underlying types
type Slice []Node

// IsValidIndex checks whether the given index is a valid index into slice,
// within range of 0..len-1.  Returns error if not.
func (sl *Slice) IsValidIndex(idx int) error {
	if idx >= 0 && idx < len(*sl) {
		return nil
	}
	return fmt.Errorf("tree.Slice: invalid index: %v with len = %v", idx, len(*sl))
}

// todo: remove the bool?

// IndexByFunc finds index of item based on match function (which must return
// true for a find match, false for not).  Returns false if not found.
// startIndex arg allows for optimized bidirectional find if you have an idea
// where it might be, which can be key speedup for large lists. If no value
// is specified for startIndex, it starts in the middle, which is a good default.
func (sl *Slice) IndexByFunc(match func(k Node) bool, startIndex ...int) (int, bool) {
	idx := findfast.FindFunc(*sl, match, startIndex...)
	return idx, idx >= 0
}

// IndexOf returns index of element in list, false if not there.  startIndex arg
// allows for optimized bidirectional find if you have an idea where it might
// be, which can be key speedup for large lists. If no value is specified for
// startIndex, it starts in the middle, which is a good default.
func (sl *Slice) IndexOf(kid Node, startIndex ...int) (int, bool) {
	return sl.IndexByFunc(func(ch Node) bool { return ch == kid }, startIndex...)
}

// IndexByName returns index of first element that has given name, false if
// not found. See [Slice.IndexOf] for info on startIndex.
func (sl *Slice) IndexByName(name string, startIndex ...int) (int, bool) {
	return sl.IndexByFunc(func(ch Node) bool { return ch.Name() == name }, startIndex...)
}

// IndexByType returns index of element that either is that type or embeds
// that type, false if not found. See [Slice.IndexOf] for info on startIndex.
func (sl *Slice) IndexByType(t *types.Type, embeds bool, startIndex ...int) (int, bool) {
	if embeds {
		return sl.IndexByFunc(func(ch Node) bool { return ch.NodeType().HasEmbed(t) }, startIndex...)
	}
	return sl.IndexByFunc(func(ch Node) bool { return ch.NodeType() == t }, startIndex...)
}

// ElemByName returns first element that has given name, nil if not found.
// See [Slice.IndexOf] for info on startIndex.
func (sl *Slice) ElemByName(name string, startIndex ...int) Node {
	idx, ok := sl.IndexByName(name, startIndex...)
	if !ok {
		return nil
	}
	return (*sl)[idx]
}

// ElemByNameTry returns first element that has given name, error if not found.
// See [Slice.IndexOf] for info on startIndex.
func (sl *Slice) ElemByNameTry(name string, startIndex ...int) (Node, error) {
	idx, ok := sl.IndexByName(name, startIndex...)
	if !ok {
		return nil, fmt.Errorf("tree.Slice: element named: %v not found", name)
	}
	return (*sl)[idx], nil
}

// ElemByType returns index of element that either is that type or embeds
// that type, nil if not found. See [Slice.IndexOf] for info on startIndex.
func (sl *Slice) ElemByType(t *types.Type, embeds bool, startIndex ...int) Node {
	idx, ok := sl.IndexByType(t, embeds, startIndex...)
	if !ok {
		return nil
	}
	return (*sl)[idx]
}

// ElemByTypeTry returns index of element that either is that type or embeds
// that type, error if not found. See [Slice.IndexOf] for info on startIndex.
func (sl *Slice) ElemByTypeTry(t *types.Type, embeds bool, startIndex ...int) (Node, error) {
	idx, ok := sl.IndexByType(t, embeds, startIndex...)
	if !ok {
		return nil, fmt.Errorf("tree.Slice: element of type: %v not found", t)
	}
	return (*sl)[idx], nil
}

// Insert item at index; does not do any parent updating etc; use
// the [Node] or [NodeBase] method unless you know what you are doing.
func (sl *Slice) Insert(k Node, i int) {
	*sl = slices.Insert(*sl, i, k)
}

// DeleteAtIndex deletes item at index; does not do any further management of
// deleted item. It is an optimized version for avoiding memory leaks. It returns
// an error if the index is invalid.
func (sl *Slice) DeleteAtIndex(i int) error {
	if err := sl.IsValidIndex(i); err != nil {
		return err
	}
	*sl = slices.Delete(*sl, i, i+1)
	return nil
}

// Move element from one position to another.  Returns error if either index
// is invalid.
func (sl *Slice) Move(frm, to int) error {
	if err := sl.IsValidIndex(frm); err != nil {
		return err
	}
	if err := sl.IsValidIndex(to); err != nil {
		return err
	}
	if frm == to {
		return nil
	}
	tmp := (*sl)[frm]
	sl.DeleteAtIndex(frm)
	*sl = slices.Insert(*sl, to, tmp)
	return nil
}

// Swap elements between positions.  Returns error if either index is invalid
func (sl *Slice) Swap(i, j int) error {
	if err := sl.IsValidIndex(i); err != nil {
		return err
	}
	if err := sl.IsValidIndex(j); err != nil {
		return err
	}
	if i == j {
		return nil
	}
	(*sl)[j], (*sl)[i] = (*sl)[i], (*sl)[j]
	return nil
}

// CopyFrom another Slice.  It is efficient by using the Config method
// which attempts to preserve any existing nodes in the destination
// if they have the same name and type -- so a copy from a source to
// a target that only differ minimally will be minimally destructive.
// it is essential that child names are unique.
func (sl *Slice) CopyFrom(frm Slice) {
	sl.ConfigCopy(nil, frm)
	for i, kid := range *sl {
		fmk := frm[i]
		kid.CopyFrom(fmk)
	}
}

// ConfigCopy uses Config method to copy name / type config of Slice from source
// If n is != nil then Update etc is called properly.
// it is essential that child names are unique.
func (sl *Slice) ConfigCopy(n Node, frm Slice) {
	sz := len(frm)
	if sz > 0 || n == nil {
		p := make(TypePlan, sz)
		for i, kid := range frm {
			p[i].Type = kid.NodeType()
			p[i].Name = kid.Name()
		}
		UpdateSlice(sl, n, p)
	} else {
		n.AsTree().DeleteChildren()
	}
}
