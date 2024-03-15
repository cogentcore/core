// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"slices"

	"cogentcore.org/core/gti"
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

// Insert item at index; does not do any parent updating etc; use
// the [Ki] or [Node] method unless you know what you are doing.
func (sl *Slice) Insert(k Ki, i int) {
	*sl = slices.Insert(*sl, i, k)
}

// SliceDeleteAtIndex deletes item at index; does not do any further management of
// deleted item. It is an optimized version for avoiding memory leaks. It returns
// an error if the index is invalid.
func SliceDeleteAtIndex(sl *[]Ki, i int) error {
	if err := SliceIsValidIndex(sl, i); err != nil {
		return err
	}
	*sl = slices.Delete(*sl, i, i+1) // this copy makes sure there are no memory leaks
	return nil
}

// DeleteAtIndex deletes item at index; does not do any further management of
// deleted item. It is an optimized version for avoiding memory leaks. It returns
// an error if the index is invalid.
func (sl *Slice) DeleteAtIndex(idx int) error {
	return SliceDeleteAtIndex((*[]Ki)(sl), idx)
}

// SliceMove moves element from one position to another.  Returns error if
// either index is invalid.
func SliceMove(sl *[]Ki, frm, i int) error {
	if err := SliceIsValidIndex(sl, frm); err != nil {
		return err
	}
	if err := SliceIsValidIndex(sl, i); err != nil {
		return err
	}
	if frm == i {
		return nil
	}
	tmp := (*sl)[frm]
	SliceDeleteAtIndex(sl, frm)
	*sl = slices.Insert(*sl, i, tmp)
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

// Config is a major work-horse routine for minimally destructive reshaping of
// a tree structure to fit a target configuration, specified in terms of a
// type-and-name list. It returns whether any changes were made to the slice.
func (sl *Slice) Config(n Ki, config Config) bool {
	mods := false
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
			sl.configDeleteKid(kid, i, &mods)
		} else if kid.KiType() != config[ti].Type {
			sl.configDeleteKid(kid, i, &mods)
		}
	}
	// next add and move items as needed -- in order so guaranteed
	for i, tn := range config {
		kidx, ok := sl.IndexByName(tn.Name, i)
		if !ok {
			mods = true
			nkid := NewOfType(tn.Type)
			nkid.SetName(tn.Name)
			InitNode(nkid)
			sl.Insert(nkid, i)
			if n != nil {
				SetParent(nkid, n)
			}
		} else {
			if kidx != i {
				mods = true
				sl.Move(kidx, i)
			}
		}
	}
	return mods
}

func (sl *Slice) configDeleteKid(kid Ki, i int, mods *bool) {
	*mods = true
	kid.Destroy()
	sl.DeleteAtIndex(i)
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
		sl.Config(n, cfg)
	} else {
		n.DeleteChildren()
	}
}
