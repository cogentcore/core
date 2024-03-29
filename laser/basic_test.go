// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"fmt"
	"testing"

	"cogentcore.org/core/laser/testdata"
)

func AFun(aa any) bool {
	return AnyIsNil(aa)
}

func TestAnyIsNil(t *testing.T) {
	ai := any(a)

	if AnyIsNil(ai) != false {
		t.Errorf("should be non-nil: %v\n", ai)
	}

	var ap *A
	api := any(ap)

	if AnyIsNil(api) != true {
		t.Errorf("should be nil: %v\n", api)
	}

	if AFun(ap) != true {
		t.Errorf("should be nil: %v\n", ap)
	}

	if AFun(&a) != false {
		t.Errorf("should be non-nil: %v\n", &a)
	}

}

func TestConverts(t *testing.T) {
	fv := 3.14
	iv := 10
	sv := "25"
	// bv := true

	// note: this does not work
	// reflect.ValueOf(&fv).Elem().Set(reflect.ValueOf("1.58").Convert(reflect.TypeOf(fv)))

	ft := "1.58"
	err := SetRobust(&fv, ft)
	fs := fmt.Sprintf("%v", fv)
	if err != nil || fs != ft {
		t.Errorf("Float convert error: %v != %v, err: %v", fs, ft, err)
	}

	it := "1"
	err = SetRobust(&iv, true)
	is := fmt.Sprintf("%v", iv)
	if err != nil || is != it {
		t.Errorf("Int convert error: %v != %v, err: %v", is, it, err)
	}

	st := "22"
	err = SetRobust(&sv, 22)
	ss := fmt.Sprintf("%v", sv)
	if err != nil || ss != st {
		t.Errorf("String convert error: %v != %v, err: %v\n", ss, st, err)
	}
	tc := C{}
	InitC()
	err = SetRobust(&tc, c)
	// fmt.Printf("tc %+v\n", tc)
	if err != nil || tc != c {
		t.Errorf("Struct convert error: %+v != %+v, err: %v\n", c, tc, err)
	}
}

func TestSetRobustFomString(t *testing.T) {
	ta := A{}
	ta.Mbr1 = "av"
	ta.Mbr2 = 22
	SetRobust(&ta, `{"Mbr1":"aa", "Mbr2":14}`) // note: only uses "
	if ta.Mbr1 != "aa" || ta.Mbr2 != 14 {
		t.Errorf("SetRobust: fields from struct string failed")
	}

	flts := []float32{1, 2, 3}
	SetRobust(&flts, `[3, 4, 5]`)
	if flts[1] != 4 {
		t.Errorf("SetRobust: slice from string failed")
	}

	mp := map[string]float32{"a": 1, "b": 2, "c": 3}
	SetRobust(&mp, `{"d":3,"e":4,"f":5}`)
	if mp["e"] != 4 {
		t.Errorf("SetRobust: map from string failed")
	}
	if mp["a"] != 1 {
		t.Errorf("SetRobust: map from string reset prior values")
	}

	fruit := testdata.Peach
	SetRobust(&fruit, "Strawberry")
	if fruit != testdata.Strawberry {
		t.Errorf("SetRobust: enum from string failed")
	}

	if ToString(fruit) != "Strawberry" {
		t.Errorf("ToString: failed to use Stringer on enum")
	}
}
