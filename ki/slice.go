// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/goki/ki/kit"
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
// where it might be -- can be key speedup for large lists -- pass -1 to start
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
// where it might be -- can be key speedup for large lists -- pass -1 to start
// in the middle (good default).
func (sl *Slice) IndexByFunc(startIdx int, match func(k Ki) bool) (int, bool) {
	return SliceIndexByFunc((*[]Ki)(sl), startIdx, match)
}

// SliceIndexOf returns index of element in list, false if not there.  startIdx arg
// allows for optimized bidirectional find if you have an idea where it might
// be -- can be key speedup for large lists -- pass -1 to start in the middle
// (good default).
func SliceIndexOf(sl *[]Ki, kid Ki, startIdx int) (int, bool) {
	return SliceIndexByFunc(sl, startIdx, func(ch Ki) bool { return ch == kid })
}

// IndexOf returns index of element in list, false if not there.  startIdx arg
// allows for optimized bidirectional find if you have an idea where it might
// be -- can be key speedup for large lists -- pass -1 to start in the middle
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

// SliceIndexByUniqueName returns index of first element that has given unique
// name, false if not found. See IndexOf for info on startIdx.
func SliceIndexByUniqueName(sl *[]Ki, name string, startIdx int) (int, bool) {
	return SliceIndexByFunc(sl, startIdx, func(ch Ki) bool { return ch.UniqueName() == name })
}

// IndexByUniqueName returns index of first element that has given unique
// name, false if not found. See IndexOf for info on startIdx.
func (sl *Slice) IndexByUniqueName(name string, startIdx int) (int, bool) {
	return sl.IndexByFunc(startIdx, func(ch Ki) bool { return ch.UniqueName() == name })
}

// SliceIndexByType returns index of element that either is that type or embeds
// that type, false if not found. See IndexOf for info on startIdx.
func SliceIndexByType(sl *[]Ki, t reflect.Type, embeds bool, startIdx int) (int, bool) {
	if embeds {
		return SliceIndexByFunc(sl, startIdx, func(ch Ki) bool { return ch.TypeEmbeds(t) })
	}
	return SliceIndexByFunc(sl, startIdx, func(ch Ki) bool { return ch.Type() == t })
}

// IndexByType returns index of element that either is that type or embeds
// that type, false if not found. See IndexOf for info on startIdx.
func (sl *Slice) IndexByType(t reflect.Type, embeds bool, startIdx int) (int, bool) {
	if embeds {
		return sl.IndexByFunc(startIdx, func(ch Ki) bool { return ch.TypeEmbeds(t) })
	}
	return sl.IndexByFunc(startIdx, func(ch Ki) bool { return ch.Type() == t })
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

// ElemByUniqueName returns index of first element that has given unique
// name, nil if not found. See IndexOf for info on startIdx.
func (sl *Slice) ElemByUniqueName(name string, startIdx int) Ki {
	idx, ok := sl.IndexByUniqueName(name, startIdx)
	if !ok {
		return nil
	}
	return (*sl)[idx]
}

// ElemByUniqueNameTry returns index of first element that has given unique
// name, error if not found. See IndexOf for info on startIdx.
func (sl *Slice) ElemByUniqueNameTry(name string, startIdx int) (Ki, error) {
	idx, ok := sl.IndexByUniqueName(name, startIdx)
	if !ok {
		return nil, fmt.Errorf("ki.Slice: element with unique name: %v not found", name)
	}
	return (*sl)[idx], nil
}

// ElemByType returns index of element that either is that type or embeds
// that type, nil if not found. See IndexOf for info on startIdx.
func (sl *Slice) ElemByType(t reflect.Type, embeds bool, startIdx int) Ki {
	idx, ok := sl.IndexByType(t, embeds, startIdx)
	if !ok {
		return nil
	}
	return (*sl)[idx]
}

// ElemByTypeTry returns index of element that either is that type or embeds
// that type, error if not found. See IndexOf for info on startIdx.
func (sl *Slice) ElemByTypeTry(t reflect.Type, embeds bool, startIdx int) (Ki, error) {
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

// TypeAndNames returns a kit.TypeAndNameList of elements in the slice --
// useful for Ki ConfigChildren.
func (sl *Slice) TypeAndNames() kit.TypeAndNameList {
	if len(*sl) == 0 {
		return nil
	}
	tn := make(kit.TypeAndNameList, len(*sl))
	for _, kid := range *sl {
		tn.Add(kid.Type(), kid.Name())
	}
	return tn
}

// TypeAndUniqueNames returns a kit.TypeAndNameList of elements in the slice
// using UniqueNames -- useful for Ki ConfigChildren.
func (sl *Slice) TypeAndUniqueNames() kit.TypeAndNameList {
	if len(*sl) == 0 {
		return nil
	}
	tn := make(kit.TypeAndNameList, len(*sl))
	for _, kid := range *sl {
		tn.Add(kid.Type(), kid.UniqueName())
	}
	return tn
}

// NameToIndexMap returns a Name to Index map for faster lookup when needing to
// do a lot of name lookups on same fixed slice.
func (sl *Slice) NameToIndexMap() map[string]int {
	if len(*sl) == 0 {
		return nil
	}
	nim := make(map[string]int, len(*sl))
	for i, kid := range *sl {
		nim[kid.Name()] = i
	}
	return nim
}

// UniqueNameToIndexMap returns a UniqueName to Index map for faster lookup
// when needing to do a lot of name lookups on same fixed slice.
func (sl *Slice) UniqueNameToIndexMap() map[string]int {
	if len(*sl) == 0 {
		return nil
	}
	nim := make(map[string]int, len(*sl))
	for i, kid := range *sl {
		nim[kid.UniqueName()] = i
	}
	return nim
}

///////////////////////////////////////////////////////////////////////////
// Config

// Config is a major work-horse routine for minimally-destructive reshaping of
// a tree structure to fit a target configuration, specified in terms of a
// type-and-name list.  If the node is != nil, then it has UpdateStart / End
// logic applied to it, only if necessary, as indicated by mods, updt return
// values.
func (sl *Slice) Config(n Ki, config kit.TypeAndNameList, uniqNm bool) (mods, updt bool) {
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
		var knm string
		if uniqNm {
			knm = kid.UniqueName()
		} else {
			knm = kid.Name()
		}
		ti, ok := nm[knm]
		if !ok {
			sl.configDeleteKid(kid, i, n, &mods, &updt)
		} else if kid.Type() != config[ti].Type {
			sl.configDeleteKid(kid, i, n, &mods, &updt)
		}
	}
	// next add and move items as needed -- in order so guaranteed
	for i, tn := range config {
		var kidx int
		var ok bool
		if uniqNm {
			kidx, ok = sl.IndexByUniqueName(tn.Name, i)
		} else {
			kidx, ok = sl.IndexByName(tn.Name, i)
		}
		if !ok {
			setMods(n, &mods, &updt)
			nkid := NewOfType(tn.Type)
			nkid.Init(nkid)
			sl.Insert(nkid, i)
			if n != nil {
				nkid.SetParent(n)
				n.SetFlag(int(ChildAdded))
			}
			if uniqNm {
				nkid.SetNameRaw(tn.Name)
				nkid.SetUniqueName(tn.Name)
			} else {
				nkid.SetName(tn.Name) // triggers uniquify -- slow!
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
			n.SetFlag(int(ChildDeleted))
		}
	}
	kid.SetFlag(int(NodeDeleted))
	kid.NodeSignal().Emit(kid, int64(NodeSignalDeleting), nil)
	kid.SetParent(nil)
	DelMgr.Add(kid)
	sl.DeleteAtIndex(i)
	kid.UpdateReset() // it won't get the UpdateEnd from us anymore -- init fresh in any case
}

// CopyFrom another Slice.  It is efficient by using the Config method
// which attempts to preserve any existing nodes in the destination
// if they have the same name and type -- so a copy from a source to
// a target that only differ minimally will be minimally destructive.
func (sl *Slice) CopyFrom(frm Slice) {
	sl.ConfigCopy(nil, frm)
	for i, kid := range *sl {
		fmk := frm[i]
		kid.CopyFrom(fmk)
	}
}

// ConfigCopy uses Config method to copy name / type config of Slice from source
// If n is != nil then Update etc is called properly.
func (sl *Slice) ConfigCopy(n Ki, frm Slice) {
	sz := len(frm)
	if sz > 0 || n == nil {
		cfg := make(kit.TypeAndNameList, sz)
		for i, kid := range frm {
			cfg[i].Type = kid.Type()
			cfg[i].Name = kid.UniqueName() // use unique so guaranteed to have something
		}
		mods, updt := sl.Config(n, cfg, true) // use unique names -- this means name = uniquname
		for i, kid := range frm {
			mkid := (*sl)[i]
			mkid.SetNameRaw(kid.Name()) // restore orig user-names
		}
		if mods && n != nil {
			n.UpdateEnd(updt)
		}
	} else {
		n.DeleteChildren(true)
	}
}

// MarshalJSON saves the length and type, name information for each object in a
// slice, as a separate struct-like record at the start, followed by the
// structs for each element in the slice -- this allows the Unmarshal to first
// create all the elements and then load them
func (sl Slice) MarshalJSON() ([]byte, error) {
	nk := len(sl)
	b := make([]byte, 0, nk*100+20)
	if nk == 0 {
		b = append(b, []byte("null")...)
		return b, nil
	}
	nstr := fmt.Sprintf("[{\"n\":%d,", nk)
	b = append(b, []byte(nstr)...)
	for i, kid := range sl {
		// fmt.Printf("json out of %v\n", kid.PathUnique())
		knm := kit.Types.TypeName(reflect.TypeOf(kid).Elem())
		tstr := fmt.Sprintf("\"type\":\"%v\", \"name\": \"%v\"", knm, kid.UniqueName()) // todo: escape names!
		b = append(b, []byte(tstr)...)
		if i < nk-1 {
			b = append(b, []byte(",")...)
		}
	}
	b = append(b, []byte("},")...)
	for i, kid := range sl {
		var err error
		var kb []byte
		kb, err = json.Marshal(kid)
		if err == nil {
			b = append(b, []byte("{")...)
			b = append(b, kb[1:len(kb)-1]...)
			b = append(b, []byte("}")...)
			if i < nk-1 {
				b = append(b, []byte(",")...)
			}
		} else {
			fmt.Printf("error doing json.Marshall from kid: %v\n", kid.PathUnique())
			log.Println(err)
			fmt.Printf("output to point of error: %v\n", string(b))
		}
	}
	b = append(b, []byte("]")...)
	// fmt.Printf("json out: %v\n", string(b))
	return b, nil
}

///////////////////////////////////////////////////////////////////////////
// JSON

// UnmarshalJSON parses the length and type information for each object in the
// slice, creates the new slice with those elements, and then loads based on
// the remaining bytes which represent each element
func (sl *Slice) UnmarshalJSON(b []byte) error {
	// fmt.Printf("json in: %v\n", string(b))
	if bytes.Equal(b, []byte("null")) {
		*sl = nil
		return nil
	}
	lb := bytes.IndexRune(b, '{')
	rb := bytes.IndexRune(b, '}')
	if lb < 0 || rb < 0 { // probably null
		return nil
	}
	// todo: if name contains "," this won't work..
	flds := bytes.Split(b[lb+1:rb], []byte(","))
	if len(flds) == 0 {
		return errors.New("Slice UnmarshalJSON: no child data found")
	}
	// fmt.Printf("flds[0]:\n%v\n", string(flds[0]))
	ns := bytes.Index(flds[0], []byte("\"n\":"))
	bn := bytes.TrimSpace(flds[0][ns+4:])

	n64, err := strconv.ParseInt(string(bn), 10, 64)
	if err != nil {
		return err
	}
	n := int(n64)
	if n == 0 {
		return nil
	}
	// fmt.Printf("n parsed: %d from %v\n", n, string(bn))

	tnl := make(kit.TypeAndNameList, n)

	for i := 0; i < n; i++ {
		fld := flds[2*i+1]
		// fmt.Printf("fld:\n%v\n", string(fld))
		ti := bytes.Index(fld, []byte("\"type\":"))
		tn := string(bytes.Trim(bytes.TrimSpace(fld[ti+7:]), "\""))
		fld = flds[2*i+2]
		ni := bytes.Index(fld, []byte("\"name\":"))
		nm := string(bytes.Trim(bytes.TrimSpace(fld[ni+7:]), "\""))
		// fmt.Printf("making type: %v", tn)
		typ := kit.Types.Type(tn)
		if typ == nil {
			return fmt.Errorf("ki.Slice UnmarshalJSON: kit.Types type name not found: %v", tn)
		}
		tnl[i].Type = typ
		tnl[i].Name = nm
	}

	sl.Config(nil, tnl, true) // true = uniq names

	nwk := make([]Ki, n) // allocate new slice containing *pointers* to kids

	for i, kid := range *sl {
		nwk[i] = kid
	}

	cb := make([]byte, 0, 1+len(b)-rb)
	cb = append(cb, []byte("[")...)
	cb = append(cb, b[rb+2:]...)

	// fmt.Printf("loading:\n%v", string(cb))

	err = json.Unmarshal(cb, &nwk)
	if err != nil {
		return err
	}
	return nil
}

// todo: save N as an attr instead of a full element

// MarshalXML saves the length and type information for each object in a
// slice, as a separate struct-like record at the start, followed by the
// structs for each element in the slice -- this allows the Unmarshal to first
// create all the elements and then load them
func (sl Slice) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	nk := len(sl)
	nt := xml.StartElement{Name: xml.Name{Space: "", Local: "N"}}
	tokens = append(tokens, nt, xml.CharData(fmt.Sprintf("%d", nk)), xml.EndElement{Name: nt.Name})
	for _, kid := range sl {
		knm := kit.Types.TypeName(reflect.TypeOf(kid).Elem())
		t := xml.StartElement{Name: xml.Name{Space: "", Local: "Type"}}
		tokens = append(tokens, t, xml.CharData(knm), xml.EndElement{Name: t.Name})
	}
	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}
	err := e.Flush()
	if err != nil {
		return err
	}
	for _, kid := range sl {
		knm := reflect.TypeOf(kid).Elem().Name()
		ct := xml.StartElement{Name: xml.Name{Space: "", Local: knm}}
		err := e.EncodeElement(kid, ct)
		if err != nil {
			return err
		}
	}
	err = e.EncodeToken(xml.EndElement{Name: start.Name})
	if err != nil {
		return err
	}
	err = e.Flush()
	if err != nil {
		return err
	}
	return nil
}

// DecodeXMLStartEl reads a start element token
func DecodeXMLStartEl(d *xml.Decoder) (start xml.StartElement, err error) {
	for {
		var t xml.Token
		t, err = d.Token()
		if err != nil {
			log.Printf("ki.DecodeXMLStartEl err %v\n", err)
			return
		}
		switch tv := t.(type) {
		case xml.StartElement:
			start = tv
			return
		case xml.CharData: // actually passes the spaces and everything through here
			continue
		case xml.EndElement:
			err = fmt.Errorf("ki.DecodeXMLStartEl: got unexpected EndElement")
			log.Println(err)
			return
		default:
			continue
		}
	}
}

// DecodeXMLEndEl reads an end element
func DecodeXMLEndEl(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err != nil {
			log.Printf("ki.DecodeXMLEndEl err %v\n", err)
			return err
		}
		switch tv := t.(type) {
		case xml.EndElement:
			if tv.Name != start.Name {
				err = fmt.Errorf("ki.DecodeXMLEndEl: EndElement: %v does not match StartElement: %v", tv.Name, start.Name)
				log.Println(err)
				return err
			}
			return nil
		case xml.CharData: // actually passes the spaces and everything through here
			continue
		case xml.StartElement:
			err = fmt.Errorf("ki.DecodeXMLEndEl: got unexpected StartElement: %v", tv.Name)
			log.Println(err)
			return err
		default:
			continue
		}
	}
}

// DecodeXMLCharData reads char data..
func DecodeXMLCharData(d *xml.Decoder) (val string, err error) {
	for {
		var t xml.Token
		t, err = d.Token()
		if err != nil {
			log.Printf("ki.DecodeXMLCharData err %v\n", err)
			return
		}
		switch tv := t.(type) {
		case xml.CharData:
			val = string([]byte(tv))
			return
		case xml.StartElement:
			err = fmt.Errorf("ki.DecodeXMLCharData: got unexpected StartElement: %v", tv.Name)
			log.Println(err)
			return
		case xml.EndElement:
			err = fmt.Errorf("ki.DecodeXMLCharData: got unexpected EndElement: %v", tv.Name)
			log.Println(err)
			return
		}
	}
}

// DecodeXMLCharEl reads a start / chardata / end sequence of 3 elements, returning name, val
func DecodeXMLCharEl(d *xml.Decoder) (name, val string, err error) {
	var st xml.StartElement
	st, err = DecodeXMLStartEl(d)
	if err != nil {
		return
	}
	name = st.Name.Local
	val, err = DecodeXMLCharData(d)
	if err != nil {
		return
	}
	err = DecodeXMLEndEl(d, st)
	if err != nil {
		return
	}
	return
}

// UnmarshalXML parses the length and type information for each object in the
// slice, creates the new slice with those elements, and then loads based on
// the remaining bytes which represent each element
func (sl *Slice) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// for _, attr := range start.Attr {
	// 	// todo: need to set the props from name / value -- don't have parent though!
	// }
	name, val, err := DecodeXMLCharEl(d)
	if err != nil {
		return err
	}
	if name == "N" {
		n64, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			return err
		}
		n := int(n64)
		if n == 0 {
			return DecodeXMLEndEl(d, start)
		}
		// fmt.Printf("n parsed: %d from %v\n", n, string(val))
		nwk := make([]Ki, 0, n) // allocate new slice

		for i := 0; i < n; i++ {
			name, val, err = DecodeXMLCharEl(d)
			if name == "Type" {
				tn := strings.TrimSpace(val)
				// fmt.Printf("making type: %v\n", tn)
				typ := kit.Types.Type(tn)
				if typ == nil {
					return fmt.Errorf("ki.Slice UnmarshalXML: kit.Types type name not found: %v", tn)
				}
				nkid := reflect.New(typ).Interface()
				// fmt.Printf("nkid is new obj of type %T val: %+v\n", nkid, nkid)
				kid, ok := nkid.(Ki)
				if !ok {
					return fmt.Errorf("ki.Slice UnmarshalXML: New child of type %v cannot convert to Ki", tn)
				}
				kid.Init(kid)
				nwk = append(nwk, kid)
			}
		}

		for i := 0; i < n; i++ {
			st, err := DecodeXMLStartEl(d)
			if err != nil {
				return err
			}
			// todo: could double-check st
			err = d.DecodeElement(nwk[i], &st)
			if err != nil {
				log.Printf("%v", err)
				return err
			}
		}
		*sl = append(*sl, nwk...)
		// } else {
	}
	// todo: in theory we could just parse a list of type names as tags, but for the "dump" format
	// this is more robust.
	return DecodeXMLEndEl(d, start) // final end
}
