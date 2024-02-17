// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"cogentcore.org/core/gti"
	"fmt"
)

// Slice is just a slice of ki elements: []Ki, providing methods for accessing
// elements in the slice, and JSON marshal / unmarshal with encoding of
// underlying types
type Slice []Ki

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
func (sl *Slice) IsValidIndex(index int) error {
	if index >= 0 && index < len(*sl) {
		return nil
	}
	return fmt.Errorf("ki.Slice: invalid index: %v -- len = %v", index, len(*sl))
}

// Elem returns element at index -- panics if index is invalid
func (sl *Slice) Elem(index int) Ki {
	return (*sl)[index]
}

// ElemTry returns element at index -- Try version returns error if index is invalid.
func (sl *Slice) ElemTry(index int) (Ki, error) {
	if err := sl.IsValidIndex(index); err != nil {
		return nil, err
	}
	return (*sl)[index], nil
}

// ElemFromEnd returns element at index from end of slice (0 = last element,
// 1 = 2nd to last, etc).  Panics if invalid index.
func (sl *Slice) ElemFromEnd(index int) Ki {
	return (*sl)[len(*sl)-1-index]
}

// ElemFromEndTry returns element at index from end of slice (0 = last element,
// 1 = 2nd to last, etc). Try version returns error on invalid index.
func (sl *Slice) ElemFromEndTry(index int) (Ki, error) {
	return sl.ElemTry(len(*sl) - 1 - index)
}

// SliceIndexByFunc finds index of item based on match function (which must
// return true for a find match, false for not).  Returns false if not found.
// startIdx arg allows for optimized bidirectional find if you have an idea
// where it might be, which can be key speedup for large lists. If no value
// is specified for startIdx, it starts in the middle, which is a good default.
func SliceIndexByFunc(sl *[]Ki, match func(k Ki) bool, startIdx ...int) (int, bool) {
	sz := len(*sl)
	if sz == 0 {
		return -1, false
	}
	si := -1
	if len(startIdx) > 0 {
		si = startIdx[0]
	}
	if si < 0 {
		si = sz / 2
	}
	if si == 0 {
		for idx, child := range *sl {
			if match(child) {
				return idx, true
			}
		}
	} else {
		if si >= sz {
			si = sz - 1
		}
		upi := si + 1
		dni := si
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
// where it might be, which can be key speedup for large lists. If no value
// is specified for startIdx, it starts in the middle, which is a good default.
func (sl *Slice) IndexByFunc(match func(k Ki) bool, startIdx ...int) (int, bool) {
	return SliceIndexByFunc((*[]Ki)(sl), match, startIdx...)
}

// SliceIndexOf returns index of element in list, false if not there.  startIdx arg
// allows for optimized bidirectional find if you have an idea where it might
// be, which can be key speedup for large lists. If no value is specified for startIdx,
// it starts in the middle, which is a good default.
func SliceIndexOf(sl *[]Ki, kid Ki, startIdx ...int) (int, bool) {
	return SliceIndexByFunc(sl, func(ch Ki) bool { return ch == kid }, startIdx...)
}

// IndexOf returns index of element in list, false if not there.  startIdx arg
// allows for optimized bidirectional find if you have an idea where it might
// be, which can be key speedup for large lists. If no value is specified for
// startIdx, it starts in the middle, which is a good default.
func (sl *Slice) IndexOf(kid Ki, startIdx ...int) (int, bool) {
	return sl.IndexByFunc(func(ch Ki) bool { return ch == kid }, startIdx...)
}

// SliceIndexByName returns index of first element that has given name, false if
// not found. See [Slice.IndexOf] for info on startIdx.
func SliceIndexByName(sl *[]Ki, name string, startIdx ...int) (int, bool) {
	return SliceIndexByFunc(sl, func(ch Ki) bool { return ch.Name() == name }, startIdx...)
}

// IndexByName returns index of first element that has given name, false if
// not found. See [Slice.IndexOf] for info on startIdx.
func (sl *Slice) IndexByName(name string, startIdx ...int) (int, bool) {
	return sl.IndexByFunc(func(ch Ki) bool { return ch.Name() == name }, startIdx...)
}

// IndexByType returns index of element that either is that type or embeds
// that type, false if not found. See [Slice.IndexOf] for info on startIdx.
func (sl *Slice) IndexByType(t *gti.Type, embeds bool, startIdx ...int) (int, bool) {
	if embeds {
		return sl.IndexByFunc(func(ch Ki) bool { return ch.KiType().HasEmbed(t) }, startIdx...)
	}
	return sl.IndexByFunc(func(ch Ki) bool { return ch.KiType() == t }, startIdx...)
}

// ElemByName returns first element that has given name, nil if not found.
// See [Slice.IndexOf] for info on startIdx.
func (sl *Slice) ElemByName(name string, startIdx ...int) Ki {
	idx, ok := sl.IndexByName(name, startIdx...)
	if !ok {
		return nil
	}
	return (*sl)[idx]
}

// ElemByNameTry returns first element that has given name, error if not found.
// See [Slice.IndexOf] for info on startIdx.
func (sl *Slice) ElemByNameTry(name string, startIdx ...int) (Ki, error) {
	idx, ok := sl.IndexByName(name, startIdx...)
	if !ok {
		return nil, fmt.Errorf("ki.Slice: element named: %v not found", name)
	}
	return (*sl)[idx], nil
}

// ElemByType returns index of element that either is that type or embeds
// that type, nil if not found. See [Slice.IndexOf] for info on startIdx.
func (sl *Slice) ElemByType(t *gti.Type, embeds bool, startIdx ...int) Ki {
	idx, ok := sl.IndexByType(t, embeds, startIdx...)
	if !ok {
		return nil
	}
	return (*sl)[idx]
}

// ElemByTypeTry returns index of element that either is that type or embeds
// that type, error if not found. See [Slice.IndexOf] for info on startIdx.
func (sl *Slice) ElemByTypeTry(t *gti.Type, embeds bool, startIdx ...int) (Ki, error) {
	idx, ok := sl.IndexByType(t, embeds, startIdx...)
	if !ok {
		return nil, fmt.Errorf("ki.Slice: element of type: %v not found", t)
	}
	return (*sl)[idx], nil
}

// SliceInsert item at index; does not do any parent updating etc;
// use the [Ki] or [Node] method unless you know what you are doing.
func SliceInsert(sl *[]Ki, k Ki, index int) {
	kl := len(*sl)
	//if index > kl { // last position allowed for insert
	//	index = kl
	//}
	//slices.Insert(*sl, index, k)  // this will found some  nil point
	//return
	if index < 0 {
		index = kl + index
	}
	if index < 0 { // still?
		index = 0
	}
	if index > kl { // last position allowed for insert
		index = kl
	}
	// this avoids extra garbage collection
	*sl = append(*sl, nil)
	if index < kl {
		copy((*sl)[index+1:], (*sl)[index:kl])
	}
	(*sl)[index] = k
}

// Insert item at index; does not do any parent updating etc; use
// the [Ki] or [Node] method unless you know what you are doing.
func (sl *Slice) Insert(k Ki, idx int) {
	SliceInsert((*[]Ki)(sl), k, idx)
}

// SliceDeleteAtIndex deletes item at index; does not do any further management of
// deleted item. It is an optimized version for avoiding memory leaks. It returns
// an error if the index is invalid.
func SliceDeleteAtIndex(sl *[]Ki, idx int) error {
	if err := SliceIsValidIndex(sl, idx); err != nil {
		return err
	}
	// this copy makes sure there are no memory leaks
	sz := len(*sl)
	copy((*sl)[idx:], (*sl)[idx+1:])
	(*sl)[sz-1] = nil
	*sl = (*sl)[:sz-1]
	return nil
}

// DeleteAtIndex deletes item at index; does not do any further management of
// deleted item. It is an optimized version for avoiding memory leaks. It returns
// an error if the index is invalid.
func (sl *Slice) DeleteAtIndex(index int) error {
	return SliceDeleteAtIndex((*[]Ki)(sl), index)
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
