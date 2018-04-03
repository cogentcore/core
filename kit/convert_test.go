// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"fmt"
	"testing"
)

func TestConverts(t *testing.T) {
	fv := 3.14
	iv := 10
	sv := "25"
	// bv := true

	// note: this does not work
	// reflect.ValueOf(&fv).Elem().Set(reflect.ValueOf("1.58").Convert(reflect.TypeOf(fv)))
	ok := false

	ft := "1.58"
	ok = SetRobust(&fv, ft)
	fs := fmt.Sprintf("%v", fv)
	if !ok || fs != ft {
		t.Errorf("Float convert error: %v != %v, ok: %v\n", fs, ft, ok)
	}

	it := "1"
	ok = SetRobust(&iv, true)
	is := fmt.Sprintf("%v", iv)
	if !ok || is != it {
		t.Errorf("Int convert error: %v != %v, ok: %v\n", is, it, ok)
	}

	st := "22"
	ok = SetRobust(&sv, 22)
	ss := fmt.Sprintf("%v", sv)
	if !ok || ss != st {
		t.Errorf("String convert error: %v != %v, ok: %v\n", ss, st, ok)
	}
	tc := C{}
	InitC()
	ok = SetRobust(&tc, c)
	// fmt.Printf("tc %+v\n", tc)
	if !ok || tc != c {
		t.Errorf("Struct convert error: %+v != %+v, ok: %v\n", c, tc, ok)
	}
}
