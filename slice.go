// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/json-iterator/go"
	"reflect"
	"strconv"
)

// Slice provides JSON marshal / unmarshal with encoding of underlying types
type Slice []Ki

// return a valid index given length of slice -- also supports access from the back of the slice using negative numbers -- -1 = last item, -2 = second to last, etc
func (k *Slice) ValidIndex(idx int) (int, error) {
	kl := len(*k)
	if kl == 0 {
		return 0, errors.New("ki.Slice is empty -- no valid index")
	}
	if idx < 0 {
		idx = kl + idx
	}
	if idx < 0 { // still?
		idx = 0
	}
	if idx >= kl {
		idx = kl - 1
	}
	return idx, nil
}

func (k *Slice) InsertKi(ki Ki, idx int) {
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
		copy((*k)[idx+1:], (*k)[idx:])
	}
	(*k)[idx] = ki
}

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

// find index of item based on match function (true for find, false for not) -- start_idx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
func (k *Slice) FindIndexByFun(start_idx int, match func(ki Ki) bool) int {
	sz := len(*k)
	if sz == 0 {
		return -1
	}
	if start_idx == 0 {
		for idx, child := range *k {
			if match(child) {
				return idx
			}
		}
	} else {
		if start_idx >= sz {
			start_idx = sz - 1
		}
		upi := start_idx + 1
		dni := start_idx
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

func (k *Slice) FindIndex(kid Ki, start_idx int) int {
	return k.FindIndexByFun(start_idx, func(ch Ki) bool { return ch == kid })
}

func (k *Slice) FindIndexByName(name string, start_idx int) int {
	return k.FindIndexByFun(start_idx, func(ch Ki) bool { return ch.KiName() == name })
}

func (k *Slice) FindIndexByUniqueName(name string, start_idx int) int {
	return k.FindIndexByFun(start_idx, func(ch Ki) bool { return ch.KiUniqueName() == name })
}

func (k *Slice) FindIndexByType(t ...reflect.Type) int {
	return k.FindIndexByFun(0, func(ch Ki) bool { return ch.IsType(t...) })
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
		b = append(b, []byte("\"type\":\"")...)
		knm := reflect.TypeOf(kid).Elem().Name()
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
		typ := Types.FindType(tn)
		if typ == nil {
			return fmt.Errorf("ki.Slice UnmarshalJSON: Types type name not found: %v", tn)
		}
		nkid := reflect.New(typ).Interface()
		// fmt.Printf("nkid is new obj of type %T val: %+v\n", nkid, nkid)
		kid, ok := nkid.(Ki)
		if !ok {
			return fmt.Errorf("ki.Slice UnmarshalJSON: New child of type %v cannot convert to Ki", tn)
		}
		kid.SetThis(kid)
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
