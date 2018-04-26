// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"reflect"
	"testing"

	"github.com/rcoreilly/goki/ki/bitflag"
)

func TestEnums(t *testing.T) {

	et := TestFlag1

	i := EnumToInt64(et)
	if i != int64(et) {
		t.Errorf("EnumToInt64 failed %v != %v", i, int64(et))
	}

	err := EnumFromInt64(&et, 2, KiT_TestFlags)
	if err != nil {
		t.Errorf("%v", err)
	}
	if et != TestFlag2 {
		t.Errorf("EnumFromInt64 failed %v != %v", et, TestFlag2)
	}

	ei := EnumIfaceFromInt64(1, KiT_TestFlags)
	if ei == nil {
		t.Errorf("EnumIfaceFromInt64 nil: %v", ei)
	}
	eiv, ok := ToInt(ei)
	if !ok {
		t.Errorf("EnumIfaceFromInt64 ToInt failed\n")
	}
	if eiv != int64(TestFlag1) {
		t.Errorf("EnumIfaceFromInt64 failed %v != %v", eiv, TestFlag1)
	}
	eit := ei.(TestFlags)
	if eit != TestFlag1 {
		t.Errorf("EnumIfaceFromInt64 failed %v != %v", eit, TestFlag1)
	}

	es := EnumInt64ToString(2, KiT_TestFlags)
	if es != "TestFlag2" {
		t.Errorf("EnumInt64ToString failed %v != %v", es, TestFlag1)
	}

	et = TestFlag2
	es = EnumToString(et)
	if es != "TestFlag2" {
		t.Errorf("EnumToString failed %v != %v", es, TestFlag2)
	}

	es = Enums.EnumToAltString(et)
	if es != "flag2" {
		t.Errorf("EnumToAltString failed %v != %v", es, "flag2")
	}

	bf := int64(0)
	bitflag.Set(&bf, int(TestFlag1), int(TestFlag2))
	es = BitFlagsToString(bf, TestFlagsN)
	if es != "TestFlag1|TestFlag2" {
		t.Errorf("EnumToString failed %v != %v", es, "TestFlag1|TestFlag2")
	}

	err = SetEnumValueFromString(reflect.ValueOf(&et), "TestFlag1")
	if err != nil {
		t.Errorf("%v", err)
	}
	if et != TestFlag1 {
		t.Errorf("SetEnumValueFromString failed %v != %v", et, TestFlag1)
	}

	err = Enums.SetEnumValueFromAltString(reflect.ValueOf(&et), "flag2")
	if err != nil {
		t.Errorf("%v", err)
	}
	if et != TestFlag2 {
		t.Errorf("SetEnumValueFromAltString failed %v != %v", et, TestFlag2)
	}

	err = Enums.SetEnumValueFromStringAltFirst(reflect.ValueOf(&et), "flag1")
	if err != nil {
		t.Errorf("%v", err)
	}
	if et != TestFlag1 {
		t.Errorf("SetEnumValueFromStringAltFirst failed %v != %v", et, TestFlag1)
	}

	err = Enums.SetEnumValueFromStringAltFirst(reflect.ValueOf(&et), "TestFlag2")
	if err != nil {
		t.Errorf("%v", err)
	}
	if et != TestFlag2 {
		t.Errorf("SetEnumValueFromStringAltFirst failed %v != %v", et, TestFlag2)
	}

	err = Enums.SetEnumValueFromInt64(reflect.ValueOf(&et), 1)
	if err != nil {
		t.Errorf("%v", err)
	}
	if et != TestFlag1 {
		t.Errorf("SetEnumValueFromInt64 failed %v != %v", et, TestFlag1)
	}

}
