// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestNonPointerType(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[int](), NonPointerType(reflect.TypeFor[int]()))
	assert.Equal(t, reflect.TypeFor[int](), NonPointerType(reflect.TypeFor[*int]()))
	assert.Equal(t, reflect.TypeFor[int](), NonPointerType(reflect.TypeFor[**int]()))
	assert.Equal(t, reflect.TypeFor[int](), NonPointerType(reflect.TypeFor[***int]()))

	assert.Equal(t, reflect.TypeFor[any](), NonPointerType(reflect.TypeFor[any]()))
	assert.Equal(t, reflect.TypeFor[any](), NonPointerType(reflect.TypeFor[*any]()))
	assert.Equal(t, reflect.TypeFor[any](), NonPointerType(reflect.TypeFor[**any]()))
	assert.Equal(t, reflect.TypeFor[any](), NonPointerType(reflect.TypeFor[***any]()))

	assert.Equal(t, nil, NonPointerType(reflect.TypeOf(nil)))
}

func TestNonPointerValue(t *testing.T) {
	v := 1
	rv := reflect.ValueOf(v)
	assert.True(t, NonPointerValue(reflect.ValueOf(v)).Equal(rv))
	assert.True(t, NonPointerValue(reflect.ValueOf(&v)).Equal(rv))

	p := &v
	assert.True(t, NonPointerValue(reflect.ValueOf(p)).Equal(rv))
	assert.True(t, NonPointerValue(reflect.ValueOf(&p)).Equal(rv))

	a := any(v)
	assert.True(t, NonPointerValue(reflect.ValueOf(a)).Equal(rv))
	assert.Equal(t, rv.Type(), NonPointerValue(reflect.ValueOf(a)).Type())
	assert.True(t, NonPointerValue(reflect.ValueOf(&a)).Equal(rv))
	// NonPointerValue cannot go through *any to get the true type
	assert.NotEqual(t, rv.Type(), NonPointerValue(reflect.ValueOf(&a)).Type())

	n := (*int)(nil)
	assert.False(t, NonPointerValue(reflect.ValueOf(n)).IsValid())

	an := any(nil)
	assert.False(t, NonPointerValue(reflect.ValueOf(an)).IsValid())
}

func TestPointerValue(t *testing.T) {
	v := 1
	rv := reflect.ValueOf(v)
	assert.False(t, rv.CanAddr())
	assert.False(t, PointerValue(reflect.ValueOf(v)).Equal(rv))
	assert.Equal(t, reflect.TypeFor[*int](), PointerValue(reflect.ValueOf(v)).Type())

	p := &v
	rp := reflect.ValueOf(p)
	assert.True(t, PointerValue(rp).Equal(rp))
	assert.Equal(t, reflect.TypeFor[*int](), PointerValue(rp).Type())

	assert.True(t, rp.Elem().CanAddr())
	assert.True(t, PointerValue(rp.Elem()).Equal(rp))
	assert.True(t, PointerValue(rp.Elem()).Equal(rp.Elem().Addr()))

	pp := &p
	rpp := reflect.ValueOf(pp)
	assert.True(t, PointerValue(rpp).Equal(rpp))
	assert.Equal(t, reflect.TypeFor[**int](), PointerValue(rpp).Type())

	n := (*int)(nil)
	rn := reflect.ValueOf(n)
	assert.True(t, PointerValue(rn).Equal(rn))

	an := any(nil)
	ran := reflect.ValueOf(an)
	assert.False(t, PointerValue(ran).IsValid())
}

func TestOnePointerValue(t *testing.T) {
	v := 1
	rv := reflect.ValueOf(v)
	assert.False(t, rv.CanAddr())
	assert.False(t, OnePointerValue(reflect.ValueOf(v)).Equal(rv))
	assert.Equal(t, reflect.TypeFor[*int](), OnePointerValue(reflect.ValueOf(v)).Type())

	p := &v
	rp := reflect.ValueOf(p)
	assert.True(t, OnePointerValue(rp).Equal(rp))
	assert.Equal(t, reflect.TypeFor[*int](), OnePointerValue(rp).Type())

	assert.True(t, rp.Elem().CanAddr())
	assert.True(t, OnePointerValue(rp.Elem()).Equal(rp))
	assert.True(t, OnePointerValue(rp.Elem()).Equal(rp.Elem().Addr()))

	pp := &p
	rpp := reflect.ValueOf(pp)
	assert.False(t, OnePointerValue(rpp).Equal(rpp))
	assert.True(t, OnePointerValue(rpp).Equal(rp))
	assert.Equal(t, reflect.TypeFor[*int](), OnePointerValue(rpp).Type())

	n := (*int)(nil)
	rn := reflect.ValueOf(n)
	assert.True(t, OnePointerValue(rn).Equal(rn))

	an := any(nil)
	ran := reflect.ValueOf(an)
	assert.False(t, OnePointerValue(ran).IsValid())
}

func TestUnderlying(t *testing.T) {
	v := 1
	rv := reflect.ValueOf(v)
	assert.True(t, Underlying(reflect.ValueOf(v)).Equal(rv))
	assert.True(t, Underlying(reflect.ValueOf(&v)).Equal(rv))

	p := &v
	assert.True(t, Underlying(reflect.ValueOf(p)).Equal(rv))
	assert.True(t, Underlying(reflect.ValueOf(&p)).Equal(rv))

	a := any(v)
	assert.True(t, Underlying(reflect.ValueOf(a)).Equal(rv))
	assert.Equal(t, rv.Type(), Underlying(reflect.ValueOf(a)).Type())
	assert.True(t, Underlying(reflect.ValueOf(&a)).Equal(rv))
	assert.Equal(t, rv.Type(), Underlying(reflect.ValueOf(&a)).Type())

	n := (*int)(nil)
	assert.False(t, Underlying(reflect.ValueOf(n)).IsValid())

	an := any(nil)
	assert.False(t, Underlying(reflect.ValueOf(an)).IsValid())
}

func TestUnderlyingPointer(t *testing.T) {
	v := 1
	rv := reflect.ValueOf(v)
	assert.False(t, rv.CanAddr())
	assert.False(t, UnderlyingPointer(reflect.ValueOf(v)).Equal(rv))
	assert.Equal(t, reflect.TypeFor[*int](), UnderlyingPointer(reflect.ValueOf(v)).Type())

	p := &v
	rp := reflect.ValueOf(p)
	assert.True(t, UnderlyingPointer(rp).Equal(rp))
	assert.Equal(t, reflect.TypeFor[*int](), UnderlyingPointer(rp).Type())

	assert.True(t, rp.Elem().CanAddr())
	assert.True(t, UnderlyingPointer(rp.Elem()).Equal(rp))
	assert.True(t, UnderlyingPointer(rp.Elem()).Equal(rp.Elem().Addr()))

	pp := &p
	rpp := reflect.ValueOf(pp)
	assert.False(t, UnderlyingPointer(rpp).Equal(rpp))
	assert.True(t, UnderlyingPointer(rpp).Equal(rp))
	assert.Equal(t, reflect.TypeFor[*int](), UnderlyingPointer(rpp).Type())

	a := any(v)
	ap := &a
	rap := reflect.ValueOf(ap)
	// Different pointer, same type
	assert.False(t, UnderlyingPointer(rap).Equal(rp))
	assert.Equal(t, rp.Type(), UnderlyingPointer(rap).Type())
	assert.Equal(t, reflect.TypeFor[*int](), UnderlyingPointer(rap).Type())

	n := (*int)(nil)
	rn := reflect.ValueOf(n)
	assert.True(t, UnderlyingPointer(rn).Equal(rn))

	an := any(nil)
	ran := reflect.ValueOf(an)
	assert.False(t, UnderlyingPointer(ran).IsValid())
}

func TestNonNilNew(t *testing.T) {
	n0 := NonNilNew(reflect.TypeFor[int]())
	assert.Equal(t, reflect.TypeFor[*int](), n0.Type())
	assert.False(t, n0.IsNil())
	assert.Equal(t, 0, n0.Elem().Interface())

	n1 := NonNilNew(reflect.TypeFor[*int]())
	assert.Equal(t, reflect.TypeFor[**int](), n1.Type())
	assert.False(t, n1.IsNil())
	assert.False(t, n1.Elem().IsNil())
	assert.Equal(t, 0, n1.Elem().Elem().Interface())

	n2 := NonNilNew(reflect.TypeFor[**int]())
	assert.Equal(t, reflect.TypeFor[***int](), n2.Type())
	assert.False(t, n2.IsNil())
	assert.False(t, n2.Elem().IsNil())
	assert.False(t, n2.Elem().Elem().IsNil())
	assert.Equal(t, 0, n2.Elem().Elem().Elem().Interface())
}

type PointerTestSub struct {
	Mbr1 string
	Mbr2 int
}

type PointerTest struct {
	Mbr1     string
	Mbr2     int
	SubField PointerTestSub
}

var pt = PointerTest{}

func InitPointerTest() {
	pt.Mbr1 = "mbr1 string"
	pt.Mbr2 = 2
}

func FieldValue(obj any, fld reflect.StructField) reflect.Value {
	ov := reflect.ValueOf(obj)
	f := unsafe.Pointer(ov.Pointer() + fld.Offset)
	nw := reflect.NewAt(fld.Type, f)
	return nw
}

func SubFieldValue(obj any, fld reflect.StructField, sub reflect.StructField) reflect.Value {
	ov := reflect.ValueOf(obj)
	f := unsafe.Pointer(ov.Pointer() + fld.Offset + sub.Offset)
	nw := reflect.NewAt(sub.Type, f)
	return nw
}

// test ability to create an addressable pointer value to fields of a struct
func TestNewAt(t *testing.T) {
	InitPointerTest()
	typ := reflect.TypeOf(pt)
	fld, _ := typ.FieldByName("Mbr2")
	vf := FieldValue(&pt, fld)

	// fmt.Printf("Fld: %v Typ: %v vf: %v vfi: %v vfT: %v vfp: %v canaddr: %v canset: %v caninterface: %v\n", fld.Name, vf.Type().String(), vf.String(), vf.Interface(), vf.Interface(), vf.Interface(), vf.CanAddr(), vf.CanSet(), vf.CanInterface())

	vf.Elem().Set(reflect.ValueOf(int(10)))

	if pt.Mbr2 != 10 {
		t.Errorf("Mbr2 should be 10, is: %v\n", pt.Mbr2)
	}

	fld, _ = typ.FieldByName("Mbr1")
	vf = FieldValue(&pt, fld)

	// fmt.Printf("Fld: %v Typ: %v vf: %v vfi: %v vfT: %v vfp: %v canaddr: %v canset: %v caninterface: %v\n", fld.Name, vf.Type().String(), vf.String(), vf.Interface(), vf.Interface(), vf.Interface(), vf.CanAddr(), vf.CanSet(), vf.CanInterface())

	vf.Elem().Set(reflect.ValueOf("this is a new string"))

	if pt.Mbr1 != "this is a new string" {
		t.Errorf("Mbr1 should be 'this is a new string': %v\n", pt.Mbr1)
	}
}
