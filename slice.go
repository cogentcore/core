// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
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

	"github.com/json-iterator/go"
	"github.com/rcoreilly/goki/ki/kit"
)

// Slice provides JSON marshal / unmarshal with encoding of underlying types
type Slice []Ki

// ValidIndex returns a valid index given length of slice -- also supports access from the back of the slice using negative numbers -- -1 = last item, -2 = second to last, etc
func (k Slice) ValidIndex(idx int) (int, error) {
	kl := len(k)
	if kl == 0 {
		return 0, errors.New("ki.Slice is empty -- no valid index")
	}
	if idx < 0 {
		idx = kl + idx
	}
	if idx < 0 { // still?
		return 0, fmt.Errorf("ki.Slice negative index: %v from back of list of children went past start of list, length: %v\n", idx, kl)
	}
	if idx >= kl {
		return 0, fmt.Errorf("ki.Slice index: %v exceeds length of list: %v\n", idx, kl)
	}
	return idx, nil
}

// IsValidIndex checks whether the given index is a valid index into slice, within range of 0..len-1 -- see ValidIndex for version that transforms negative numbers into indicies from end of slice, and has explicit error messages
func (k Slice) IsValidIndex(idx int) bool {
	return idx >= 0 && idx < len(k)
}

// Elem returns element at index, using ValidIndex supporting negative indexing from back of list -- returns nil if index is invalid -- use ValidIndex or IsValidIndex directly to test if unsure
func (k *Slice) Elem(idx int) Ki {
	idx, err := k.ValidIndex(idx)
	if err != nil {
		return nil
	}
	return (*k)[idx]
}

// Insert item at index
func (k *Slice) Insert(ki Ki, idx int) {
	kl := len(*k)
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
	*k = append(*k, nil)
	if idx < kl {
		copy((*k)[idx+1:], (*k)[idx:kl])
	}
	(*k)[idx] = ki
}

// Delete item at index -- does not do any further management deleted item -- optimized version for avoiding memory leaks
func (k *Slice) DeleteAtIndex(idx int) error {
	idx, err := k.ValidIndex(idx)
	if err != nil {
		return err
	}
	// this copy makes sure there are no memory leaks
	sz := len(*k)
	copy((*k)[idx:], (*k)[idx+1:])
	(*k)[sz-1] = nil
	(*k) = (*k)[:sz-1]
	return nil
}

// Move element from one position to another
func (k *Slice) Move(from, to int) error {
	var err error
	from, err = k.ValidIndex(from)
	if err != nil {
		return err
	}
	to, err = k.ValidIndex(to)
	if err != nil {
		return err
	}
	if from == to {
		return nil
	}
	ki := (*k)[from]
	k.DeleteAtIndex(from)
	k.Insert(ki, to)
	return nil
}

// find index of item based on match function (true for find, false for not) -- startIdx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
func (k *Slice) IndexByFunc(startIdx int, match func(ki Ki) bool) int {
	sz := len(*k)
	if sz == 0 {
		return -1
	}
	// todo: benchmark setting startIdx = sz / 2 here..
	if startIdx == 0 {
		for idx, child := range *k {
			if match(child) {
				return idx
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
				if match((*k)[upi]) {
					return upi
				}
				upi++
			} else {
				upo = true
			}
			if dni >= 0 {
				if match((*k)[dni]) {
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

// Index returns index of element in list or -1 if not there
func (k *Slice) Index(kid Ki, startIdx int) int {
	return k.IndexByFunc(startIdx, func(ch Ki) bool { return ch == kid })
}

// IndexByName returns index of first element that has given name -- startIdx arg allows for optimized bidirectional search if you have an idea where it might be -- can be key speedup for large lists
func (k *Slice) IndexByName(name string, startIdx int) int {
	return k.IndexByFunc(startIdx, func(ch Ki) bool { return ch.Name() == name })
}

// IndexByUniqueName returns index of first element that has given unique name -- startIdx arg allows for optimized bidirectional search if you have an idea where it might be -- can be key speedup for large lists
func (k *Slice) IndexByUniqueName(name string, startIdx int) int {
	return k.IndexByFunc(startIdx, func(ch Ki) bool { return ch.UniqueName() == name })
}

// IndexByType returns index of element that either is that type or embeds that type -- startIdx arg allows for optimized bidirectional search if you have an idea where it might be -- can be key speedup for large lists
func (k *Slice) IndexByType(t reflect.Type, embeds bool, startIdx int) int {
	if embeds {
		return k.IndexByFunc(startIdx, func(ch Ki) bool { return ch.TypeEmbeds(t) })
	} else {
		return k.IndexByFunc(startIdx, func(ch Ki) bool { return ch.Type() == t })
	}
}

// return a kit.TypeAndNameList of elements in the slice -- useful for Ki ConfigChildren
func (k *Slice) TypeAndNames() kit.TypeAndNameList {
	if len(*k) == 0 {
		return nil
	}
	tn := make(kit.TypeAndNameList, len(*k))
	for _, kid := range *k {
		tn.Add(kid.Type(), kid.Name())
	}
	return tn
}

// return a kit.TypeAndNameList of elements in the slice using UniqueNames -- useful for Ki ConfigChildren
func (k *Slice) TypeAndUniqueNames() kit.TypeAndNameList {
	if len(*k) == 0 {
		return nil
	}
	tn := make(kit.TypeAndNameList, len(*k))
	for _, kid := range *k {
		tn.Add(kid.Type(), kid.UniqueName())
	}
	return tn
}

// return a Name to Index map for faster lookup when needing to do a lot of name lookups on same fixed slice
func (k *Slice) NameToIndexMap() map[string]int {
	if len(*k) == 0 {
		return nil
	}
	nim := make(map[string]int, len(*k))
	for i, kid := range *k {
		nim[kid.Name()] = i
	}
	return nim
}

// return a UniqueName to Index map for faster lookup when needing to do a lot of name lookups on same fixed slice
func (k *Slice) UniqueNameToIndexMap() map[string]int {
	if len(*k) == 0 {
		return nil
	}
	nim := make(map[string]int, len(*k))
	for i, kid := range *k {
		nim[kid.UniqueName()] = i
	}
	return nim
}

// MarshalJSON saves the length and type information for each object in a slice, as a separate struct-like record at the start, followed by the structs for each element in the slice -- this allows the Unmarshal to first create all the elements and then load them
func (k Slice) MarshalJSON() ([]byte, error) {
	nk := len(k)
	b := make([]byte, 0, nk*100+20)
	if nk == 0 {
		b = append(b, []byte("null")...)
		return b, nil
	}
	b = append(b, []byte("[{\"n\":")...)
	b = append(b, []byte(fmt.Sprintf("%d", nk))...)
	b = append(b, []byte(",")...)
	for i, kid := range k {
		// fmt.Printf("json out of %v\n", kid.PathUnique())
		b = append(b, []byte("\"type\":\"")...)
		knm := kit.FullTypeName(reflect.TypeOf(kid).Elem())
		b = append(b, []byte(knm)...)
		b = append(b, []byte("\"")...)
		if i < nk-1 {
			b = append(b, []byte(",")...)
		}
	}
	b = append(b, []byte("},")...)
	for i, kid := range k {
		var err error
		var kb []byte
		if UseJsonIter {
			kb, err = jsoniter.Marshal(kid)
		} else {
			kb, err = json.Marshal(kid)
		}
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

// UnmarshalJSON parses the length and type information for each object in the slice, creates the new slice with those elements, and then loads based on the remaining bytes which represent each element
func (k *Slice) UnmarshalJSON(b []byte) error {
	// fmt.Printf("json in: %v\n", string(b))
	if bytes.Equal(b, []byte("null")) {
		*k = nil
		return nil
	}
	lb := bytes.IndexRune(b, '{')
	rb := bytes.IndexRune(b, '}')
	if lb < 0 || rb < 0 { // probably null
		return nil
	}
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

	nwk := make([]Ki, 0, n) // allocate new slice

	for i := 0; i < n; i++ {
		fld := flds[i+1]
		// fmt.Printf("fld:\n%v\n", string(fld))
		ti := bytes.Index(fld, []byte("\"type\":"))
		tn := string(bytes.Trim(bytes.TrimSpace(fld[ti+7:]), "\""))
		// fmt.Printf("making type: %v", tn)
		typ := kit.Types.Type(tn)
		if typ == nil {
			return fmt.Errorf("ki.Slice UnmarshalJSON: kit.Types type name not found: %v", tn)
		}
		nkid := reflect.New(typ).Interface()
		// fmt.Printf("nkid is new obj of type %T val: %+v\n", nkid, nkid)
		kid, ok := nkid.(Ki)
		if !ok {
			return fmt.Errorf("ki.Slice UnmarshalJSON: New child of type %v cannot convert to Ki", tn)
		}
		kid.Init(kid)
		nwk = append(nwk, kid)
	}

	cb := make([]byte, 0, 1+len(b)-rb)
	cb = append(cb, []byte("[")...)
	cb = append(cb, b[rb+2:]...)

	// fmt.Printf("loading:\n%v", string(cb))

	if UseJsonIter {
		err = jsoniter.Unmarshal(cb, &nwk)
	} else {
		err = json.Unmarshal(cb, &nwk)
	}
	if err != nil {
		return err
	}
	*k = append(*k, nwk...)
	return nil
}

// todo: save N as an attr instead of a full element

// MarshalXML saves the length and type information for each object in a slice, as a separate struct-like record at the start, followed by the structs for each element in the slice -- this allows the Unmarshal to first create all the elements and then load them
func (k Slice) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	nk := len(k)
	nt := xml.StartElement{Name: xml.Name{"", "N"}}
	tokens = append(tokens, nt, xml.CharData(fmt.Sprintf("%d", nk)), xml.EndElement{nt.Name})
	for _, kid := range k {
		knm := kit.FullTypeName(reflect.TypeOf(kid).Elem())
		t := xml.StartElement{Name: xml.Name{"", "Type"}}
		tokens = append(tokens, t, xml.CharData(knm), xml.EndElement{t.Name})
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
	for _, kid := range k {
		knm := reflect.TypeOf(kid).Elem().Name()
		ct := xml.StartElement{Name: xml.Name{"", knm}}
		err := e.EncodeElement(kid, ct)
		if err != nil {
			return err
		}
	}
	err = e.EncodeToken(xml.EndElement{start.Name})
	if err != nil {
		return err
	}
	err = e.Flush()
	if err != nil {
		return err
	}
	return nil
}

// read a start element token
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
			err = fmt.Errorf("ki.DecodeXMLStartEl: got unexpected EndElement\n")
			log.Printf("%v", err)
			return
		default:
			continue
		}
	}
}

// read an end element
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
				log.Printf("%v", err)
				return err
			}
			return nil
		case xml.CharData: // actually passes the spaces and everything through here
			continue
		case xml.StartElement:
			err = fmt.Errorf("ki.DecodeXMLEndEl: got unexpected StartElement: %v\n", tv.Name)
			log.Printf("%v", err)
			return err
		default:
			continue
		}
	}
}

// read char data..
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
			err = fmt.Errorf("ki.DecodeXMLCharData: got unexpected StartElement: %v\n", tv.Name)
			log.Printf("%v", err)
			return
		case xml.EndElement:
			err = fmt.Errorf("ki.DecodeXMLCharData: got unexpected EndElement: %v\n", tv.Name)
			log.Printf("%v", err)
			return
		}
	}
}

// read a start / chardata / end sequence of 3 elements, returning name, val
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

// UnmarshalJSON parses the length and type information for each object in the slice, creates the new slice with those elements, and then loads based on the remaining bytes which represent each element
func (k *Slice) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
		*k = append(*k, nwk...)
		// } else {
	}
	// todo: in theory we could just parse a list of type names as tags, but for the "dump" format
	// this is more robust.
	return DecodeXMLEndEl(d, start) // final end
}
