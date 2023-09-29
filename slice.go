// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"

	"goki.dev/gti"
)

// Slice is just a slice of ki elements: []Ki, providing methods for accessing
// elements in the slice, and JSON marshal / unmarshal with encoding of
// underlying types
type Slice []Ki

// StartMiddle indicates to start searching
// in the middle for slice search functions.
const StartMiddle int = -1

// NOTE: we have to define Slice* functions operating on a generic *[]Ki
// element as the first (not receiver) argument, to be able to use these
// functions in any other types that are based on ki.Slice or are other forms
// of []Ki.  It doesn't seem like it would have been THAT hard to just grab
// all the methods on Slice when you "inherit" from it -- unlike with structs,
// where there are issues with the underlying representation, a simple "type A
// B" kind of expression could easily have inherited the exact same code
// because, underneath, it IS the same type.  Only for the receiver methods --
// it does seem reasonable that other uses of different types should
// differentiate them.  But there you still be able to directly cast!

// SliceIsValidIndex checks whether the given index is a valid index into slice,
// within range of 0..len-1.  Returns error if not.
func SliceIsValidIndex(sl *[]Ki, idx int) error {
	if idx >= 0 && idx < len(*sl) {
		return nil
	}
	return fmt.Errorf("ki.Slice: invalid index: %v -- len = %v", idx, len(*sl))
}

// IsValidIndex checks whether the given index is a valid index into slice,
// within range of 0..len-1.  Returns error if not.
func (sl *Slice) IsValidIndex(idx int) error {
	if idx >= 0 && idx < len(*sl) {
		return nil
	}
	return fmt.Errorf("ki.Slice: invalid index: %v -- len = %v", idx, len(*sl))
}

// Elem returns element at index -- panics if index is invalid
func (sl *Slice) Elem(idx int) Ki {
	return (*sl)[idx]
}

// ElemTry returns element at index -- Try version returns error if index is invalid.
func (sl *Slice) ElemTry(idx int) (Ki, error) {
	if err := sl.IsValidIndex(idx); err != nil {
		return nil, err
	}
	return (*sl)[idx], nil
}

// ElemFromEnd returns element at index from end of slice (0 = last element,
// 1 = 2nd to last, etc).  Panics if invalid index.
func (sl *Slice) ElemFromEnd(idx int) Ki {
	return (*sl)[len(*sl)-1-idx]
}

// ElemFromEndTry returns element at index from end of slice (0 = last element,
// 1 = 2nd to last, etc). Try version returns error on invalid index.
func (sl *Slice) ElemFromEndTry(idx int) (Ki, error) {
	return sl.ElemTry(len(*sl) - 1 - idx)
}

// SliceIndexByFunc finds index of item based on match function (which must
// return true for a find match, false for not).  Returns false if not found.
// startIdx arg allows for optimized bidirectional find if you have an idea
// where it might be -- can be key speedup for large lists -- pass [ki.StartMiddle] to start
// in the middle (good default)
func SliceIndexByFunc(sl *[]Ki, startIdx int, match func(k Ki) bool) (int, bool) {
	sz := len(*sl)
	if sz == 0 {
		return -1, false
	}
	if startIdx < 0 {
		startIdx = sz / 2
	}
	if startIdx == 0 {
		for idx, child := range *sl {
			if match(child) {
				return idx, true
			}
		}
	} else {
		if startIdx >= sz {
			startIdx = sz - 1
		}
		upi := startIdx + 1
		dni := startIdx
		upo := false
		for {
			if !upo && upi < sz {
				if match((*sl)[upi]) {
					return upi, true
				}
				upi++
			} else {
				upo = true
			}
			if dni >= 0 {
				if match((*sl)[dni]) {
					return dni, true
				}
				dni--
			} else if upo {
				break
			}
		}
	}
	return -1, false
}

// IndexByFunc finds index of item based on match function (which must return
// true for a find match, false for not).  Returns false if not found.
// startIdx arg allows for optimized bidirectional find if you have an idea
// where it might be -- can be key speedup for large lists -- pass [ki.StartMiddle] to start
// in the middle (good default).
func (sl *Slice) IndexByFunc(startIdx int, match func(k Ki) bool) (int, bool) {
	return SliceIndexByFunc((*[]Ki)(sl), startIdx, match)
}

// SliceIndexOf returns index of element in list, false if not there.  startIdx arg
// allows for optimized bidirectional find if you have an idea where it might
// be -- can be key speedup for large lists -- pass [ki.StartMiddle] to start in the middle
// (good default).
func SliceIndexOf(sl *[]Ki, kid Ki, startIdx int) (int, bool) {
	return SliceIndexByFunc(sl, startIdx, func(ch Ki) bool { return ch == kid })
}

// IndexOf returns index of element in list, false if not there.  startIdx arg
// allows for optimized bidirectional find if you have an idea where it might
// be -- can be key speedup for large lists -- pass [ki.StartMiddle] to start in the middle
// (good default).
func (sl *Slice) IndexOf(kid Ki, startIdx int) (int, bool) {
	return sl.IndexByFunc(startIdx, func(ch Ki) bool { return ch == kid })
}

// SliceIndexByName returns index of first element that has given name, false if
// not found. See IndexOf for info on startIdx.
func SliceIndexByName(sl *[]Ki, name string, startIdx int) (int, bool) {
	return SliceIndexByFunc(sl, startIdx, func(ch Ki) bool { return ch.Name() == name })
}

// IndexByName returns index of first element that has given name, false if
// not found. See IndexOf for info on startIdx
func (sl *Slice) IndexByName(name string, startIdx int) (int, bool) {
	return sl.IndexByFunc(startIdx, func(ch Ki) bool { return ch.Name() == name })
}

// IndexByType returns index of element that either is that type or embeds
// that type, false if not found. See IndexOf for info on startIdx.
func (sl *Slice) IndexByType(t *gti.Type, embeds bool, startIdx int) (int, bool) {
	if embeds {
		return sl.IndexByFunc(startIdx, func(ch Ki) bool { return ch.KiType().HasEmbed(t) })
	}
	return sl.IndexByFunc(startIdx, func(ch Ki) bool { return ch.KiType() == t })
}

// ElemByName returns first element that has given name, nil if not found.
// See IndexOf for info on startIdx.
func (sl *Slice) ElemByName(name string, startIdx int) Ki {
	idx, ok := sl.IndexByName(name, startIdx)
	if !ok {
		return nil
	}
	return (*sl)[idx]
}

// ElemByNameTry returns first element that has given name, error if not found.
// See IndexOf for info on startIdx.
func (sl *Slice) ElemByNameTry(name string, startIdx int) (Ki, error) {
	idx, ok := sl.IndexByName(name, startIdx)
	if !ok {
		return nil, fmt.Errorf("ki.Slice: element named: %v not found", name)
	}
	return (*sl)[idx], nil
}

// ElemByType returns index of element that either is that type or embeds
// that type, nil if not found. See IndexOf for info on startIdx.
func (sl *Slice) ElemByType(t *gti.Type, embeds bool, startIdx int) Ki {
	idx, ok := sl.IndexByType(t, embeds, startIdx)
	if !ok {
		return nil
	}
	return (*sl)[idx]
}

// ElemByTypeTry returns index of element that either is that type or embeds
// that type, error if not found. See IndexOf for info on startIdx.
func (sl *Slice) ElemByTypeTry(t *gti.Type, embeds bool, startIdx int) (Ki, error) {
	idx, ok := sl.IndexByType(t, embeds, startIdx)
	if !ok {
		return nil, fmt.Errorf("ki.Slice: element of type: %v not found", t)
	}
	return (*sl)[idx], nil
}

// SliceInsert item at index -- does not do any parent updating etc -- use Ki/Node
// method unless you know what you are doing.
func SliceInsert(sl *[]Ki, k Ki, idx int) {
	kl := len(*sl)
	if idx < 0 {
		idx = kl + idx
	}
	if idx < 0 { // still?
		idx = 0
	}
	if idx > kl { // last position allowed for insert
		idx = kl
	}
	// this avoids extra garbage collection
	*sl = append(*sl, nil)
	if idx < kl {
		copy((*sl)[idx+1:], (*sl)[idx:kl])
	}
	(*sl)[idx] = k
}

// Insert item at index -- does not do any parent updating etc -- use Ki/Node
// method unless you know what you are doing.
func (sl *Slice) Insert(k Ki, idx int) {
	SliceInsert((*[]Ki)(sl), k, idx)
}

// SliceDeleteAtIndex deletes item at index -- does not do any further management
// deleted item -- optimized version for avoiding memory leaks.  returns error
// if index is invalid.
func SliceDeleteAtIndex(sl *[]Ki, idx int) error {
	if err := SliceIsValidIndex(sl, idx); err != nil {
		return err
	}
	// this copy makes sure there are no memory leaks
	sz := len(*sl)
	copy((*sl)[idx:], (*sl)[idx+1:])
	(*sl)[sz-1] = nil
	(*sl) = (*sl)[:sz-1]
	return nil
}

// DeleteAtIndex deletes item at index -- does not do any further management
// deleted item -- optimized version for avoiding memory leaks.  returns error
// if index is invalid.
func (sl *Slice) DeleteAtIndex(idx int) error {
	return SliceDeleteAtIndex((*[]Ki)(sl), idx)
}

// SliceMove moves element from one position to another.  Returns error if
// either index is invalid.
func SliceMove(sl *[]Ki, frm, to int) error {
	if err := SliceIsValidIndex(sl, frm); err != nil {
		return err
	}
	if err := SliceIsValidIndex(sl, to); err != nil {
		return err
	}
	if frm == to {
		return nil
	}
	tmp := (*sl)[frm]
	SliceDeleteAtIndex(sl, frm)
	SliceInsert(sl, tmp, to)
	return nil
}

// Move element from one position to another.  Returns error if either index
// is invalid.
func (sl *Slice) Move(frm, to int) error {
	return SliceMove((*[]Ki)(sl), frm, to)
}

// SliceSwap swaps elements between positions.  Returns error if either index is invalid
func SliceSwap(sl *[]Ki, i, j int) error {
	if err := SliceIsValidIndex(sl, i); err != nil {
		return err
	}
	if err := SliceIsValidIndex(sl, j); err != nil {
		return err
	}
	if i == j {
		return nil
	}
	(*sl)[j], (*sl)[i] = (*sl)[i], (*sl)[j]
	return nil
}

// Swap elements between positions.  Returns error if either index is invalid
func (sl *Slice) Swap(i, j int) error {
	return SliceSwap((*[]Ki)(sl), i, j)
}

///////////////////////////////////////////////////////////////////////////
// Config

// Config is a major work-horse routine for minimally-destructive reshaping of
// a tree structure to fit a target configuration, specified in terms of a
// type-and-name list.  If the node is != nil, then it has UpdateStart / End
// logic applied to it, only if necessary, as indicated by mods, updt return
// values.
func (sl *Slice) Config(n Ki, config Config) (mods, updt bool) {
	mods, updt = false, false
	// first make a map for looking up the indexes of the names
	nm := make(map[string]int)
	for i, tn := range config {
		nm[tn.Name] = i
	}
	// first remove any children not in the config
	sz := len(*sl)
	for i := sz - 1; i >= 0; i-- {
		kid := (*sl)[i]
		knm := kid.Name()
		ti, ok := nm[knm]
		if !ok {
			sl.configDeleteKid(kid, i, n, &mods, &updt)
		} else if kid.KiType() != config[ti].Type {
			sl.configDeleteKid(kid, i, n, &mods, &updt)
		}
	}
	// next add and move items as needed -- in order so guaranteed
	for i, tn := range config {
		kidx, ok := sl.IndexByName(tn.Name, i)
		if !ok {
			setMods(n, &mods, &updt)
			nkid := NewOfType(tn.Type)
			nkid.SetName(tn.Name)
			InitNode(nkid)
			sl.Insert(nkid, i)
			if n != nil {
				SetParent(nkid, n)
				n.SetChildAdded()
			}
		} else {
			if kidx != i {
				setMods(n, &mods, &updt)
				sl.Move(kidx, i)
			}
		}
	}
	DelMgr.DestroyDeleted()
	return
}

func setMods(n Ki, mods *bool, updt *bool) {
	if !*mods {
		*mods = true
		if n != nil {
			*updt = n.UpdateStart()
		}
	}
}

func (sl *Slice) configDeleteKid(kid Ki, i int, n Ki, mods, updt *bool) {
	if !*mods {
		*mods = true
		if n != nil {
			*updt = n.UpdateStart()
			n.SetFlag(true, ChildDeleted)
		}
	}
	DeleteFromParent(kid)
	DelMgr.Add(kid)
	sl.DeleteAtIndex(i)
	UpdateReset(kid) // it won't get the UpdateEnd from us anymore -- init fresh in any case
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
func (sl *Slice) ConfigCopy(n Ki, frm Slice) {
	sz := len(frm)
	if sz > 0 || n == nil {
		cfg := make(Config, sz)
		for i, kid := range frm {
			cfg[i].Type = kid.KiType()
			cfg[i].Name = kid.Name()
		}
		mods, updt := sl.Config(n, cfg)
		if mods && n != nil {
			n.UpdateEnd(updt)
		}
	} else {
		n.DeleteChildren(true)
	}
}
