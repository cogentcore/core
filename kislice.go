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

// KiSlice provides JSON marshal / unmarshal with encoding of underlying types
type KiSlice []Ki

// MarshalJSON saves the length and type information for each object in a slice, as a separate struct-like record at the start, followed by the structs for each element in the slice -- this allows the Unmarshal to first create all the elements and then load them
func (k KiSlice) MarshalJSON() ([]byte, error) {
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
func (k *KiSlice) UnmarshalJSON(b []byte) error {
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
		return errors.New("KiSlice UnmarshalJSON: no child data found")
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
		typ := KiTypes.GetType(tn)
		if typ == nil {
			return fmt.Errorf("KiSlice UnmarshalJSON: KiTypes type name not found: %v", tn)
		}
		nkid := reflect.New(typ).Interface()
		// fmt.Printf("nkid is new obj of type %T val: %+v\n", nkid, nkid)
		kid, ok := nkid.(Ki)
		if !ok {
			return fmt.Errorf("KiSlice UnmarshalJSON: New child of type %v cannot convert to Ki", tn)
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
