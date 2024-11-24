// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"fmt"
	"reflect"
	"testing"

	"cogentcore.org/core/base/reflectx/testdata"
	"github.com/stretchr/testify/assert"
)

type A struct {
	Mbr1 string
	Mbr2 int
}

type B struct {
	A
	Mbr3 string
	Mbr4 int
}

type C struct {
	B
	Mbr5 string
	Mbr6 int
}

var a = A{}
var c = C{}

func InitC() {
	c.Mbr1 = "mbr1 string"
	c.Mbr2 = 2
	c.Mbr3 = "mbr3 string"
	c.Mbr4 = 4
	c.Mbr5 = "mbr5 string"
	c.Mbr6 = 6
}

func AFun(aa any) bool {
	return IsNil(reflect.ValueOf(aa))
}

func TestAnyIsNil(t *testing.T) {
	ai := any(a)

	assert.False(t, IsNil(reflect.ValueOf(ai)))

	var ap *A
	api := any(ap)

	assert.True(t, IsNil(reflect.ValueOf(api)))
	assert.True(t, AFun(ap))
	assert.False(t, AFun(&a))
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

func TestPointerSetRobust(t *testing.T) {
	a := A{}
	aptr := &a
	b := A{}
	bptr := &b
	err := SetRobust(&aptr, bptr)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, aptr, bptr)
}

func BenchmarkFloatToFloat(b *testing.B) {
	sum := 0.0
	for range b.N {
		v, _ := ToFloat(1.0)
		sum += v
	}
	b.Log(sum)
}

func BenchmarkUint64ToFloat(b *testing.B) {
	sum := 0.0
	for range b.N {
		v, _ := ToFloat(uint64(1))
		sum += v
	}
	b.Log(sum)
}

func BenchmarkStringToFloat(b *testing.B) {
	sum := 0.0
	for range b.N {
		v, _ := ToFloat("1")
		sum += v
	}
	b.Log(sum)
}

func TestSliceSetRobust(t *testing.T) {
	a := &[]int{1, 2, 3}
	aa := any(a)

	var b any
	ba := any(&b)

	assert.NoError(t, SetRobust(ba, aa))
	assert.Equal(t, fmt.Sprintf("%p", a), fmt.Sprintf("%p", b))

	assert.NoError(t, SetRobust(aa, ba))
	assert.Equal(t, fmt.Sprintf("%p", a), fmt.Sprintf("%p", b))
}
