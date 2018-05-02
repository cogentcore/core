// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"reflect"
	"testing"
	"unsafe"
)

// structs are in embeds_test.go

type PtrTstSub struct {
	Mbr1 string
	Mbr2 int
	Enum TestFlags
}

type PtrTst struct {
	Mbr1     string
	Mbr2     int
	Enum     TestFlags
	SubField PtrTstSub
}

var pt = PtrTst{}

func InitPtrTst() {
	pt.Mbr1 = "mbr1 string"
	pt.Mbr2 = 2
	pt.Enum = TestFlag2
}

func FieldValue(obj interface{}, fld reflect.StructField) reflect.Value {
	ov := reflect.ValueOf(obj)
	f := unsafe.Pointer(ov.Pointer() + fld.Offset)
	nw := reflect.NewAt(fld.Type, f)
	return UnhideIfaceValue(nw).Elem()
}

func SubFieldValue(obj interface{}, fld reflect.StructField, sub reflect.StructField) reflect.Value {
	ov := reflect.ValueOf(obj)
	f := unsafe.Pointer(ov.Pointer() + fld.Offset + sub.Offset)
	nw := reflect.NewAt(sub.Type, f)
	return UnhideIfaceValue(nw).Elem()
}

// test ability to create an addressable pointer value to fields of a struct
func TestNewAt(t *testing.T) {
	InitPtrTst()
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

	fld, _ = typ.FieldByName("Enum")
	vf = FieldValue(&pt, fld)

	// fmt.Printf("Fld: %v Typ: %v vf: %v vfi: %v vfT: %v vfp: %v canaddr: %v canset: %v caninterface: %v\n", fld.Name, vf.Type().String(), vf.String(), vf.Interface(), vf.Interface(), vf.Interface(), vf.CanAddr(), vf.CanSet(), vf.CanInterface())

	vf.Elem().Set(reflect.ValueOf(TestFlag1))

	if pt.Enum != TestFlag1 {
		t.Errorf("Enum should be TestFlag1: %v\n", pt.Enum)
	}

	err := SetEnumValueFromString(vf, "TestFlag2")
	if err != nil {
		t.Errorf("%v", err)
	}

	if pt.Enum != TestFlag2 {
		t.Errorf("Enum should be TestFlag2: %v\n", pt.Enum)
	}

	err = Enums.SetEnumValueFromAltString(vf, "flag1")
	if err != nil {
		t.Errorf("%v", err)
	}

	if pt.Enum != TestFlag1 {
		t.Errorf("Enum should be TestFlag1: %v\n", pt.Enum)
	}
}

func TestNewAtSub(t *testing.T) {
	InitPtrTst()
	typ := reflect.TypeOf(pt)
	subtyp := reflect.TypeOf(pt.SubField)

	fld, _ := typ.FieldByName("SubField")
	sub, _ := subtyp.FieldByName("Enum")
	vf := SubFieldValue(&pt, fld, sub)

	// fmt.Printf("Fld: %v Typ: %v vf: %v vfi: %v vfT: %v vfp: %v canaddr: %v canset: %v caninterface: %v\n", fld.Name, vf.Type().String(), vf.String(), vf.Interface(), vf.Interface(), vf.Interface(), vf.CanAddr(), vf.CanSet(), vf.CanInterface())

	pt.SubField.Enum = TestFlag2
	vf.Elem().Set(reflect.ValueOf(TestFlag1))

	if pt.SubField.Enum != TestFlag1 {
		t.Errorf("Enum should be TestFlag1: %v\n", pt.SubField.Enum)
	}

	err := SetEnumValueFromString(vf, "TestFlag2")
	if err != nil {
		t.Errorf("%v", err)
	}

	if pt.SubField.Enum != TestFlag2 {
		t.Errorf("Enum should be TestFlag2: %v\n", pt.SubField.Enum)
	}

	err = Enums.SetEnumValueFromAltString(vf, "flag1")
	if err != nil {
		t.Errorf("%v", err)
	}

	if pt.SubField.Enum != TestFlag1 {
		t.Errorf("Enum should be TestFlag1: %v\n", pt.SubField.Enum)
	}

}
