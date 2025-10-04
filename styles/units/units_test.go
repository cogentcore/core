// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"fmt"
	"testing"

	"cogentcore.org/core/base/tolassert"
)

func TestToDots(t *testing.T) {
	tests := map[Units]float32{
		UnitPx:   83.33333,
		UnitDp:   50,
		UnitEw:   200,
		UnitEh:   250,
		UnitPw:   450,
		UnitPh:   350,
		UnitRem:  800,
		UnitEm:   800,
		UnitEx:   400,
		UnitCh:   400,
		UnitVw:   960,
		UnitVh:   540,
		UnitVmin: 540,
		UnitVmax: 960,
		UnitCm:   3149.6064,
		UnitMm:   314.96063,
		UnitQ:    78.74016,
		UnitIn:   8000,
		UnitPc:   1333.3333,
		UnitPt:   111.111115,
		UnitDot:  50,
	}
	var uc Context
	uc.Defaults()
	uc.Vpw = 1920
	uc.Vph = 1080
	uc.Elw = 400
	uc.Elh = 500
	uc.Paw = 900
	uc.Pah = 700
	for unit, want := range tests {
		v := New(50, unit)
		have := v.ToDots(&uc)
		tolassert.Equal(t, want, have, unit)
	}
}

func TestValueConvert(t *testing.T) {
	var ctxt Context
	ctxt.Defaults()
	for _, un := range UnitsValues() {
		v1 := New(1.0, un)
		s1 := fmt.Sprintf("%v = %v dots", v1, v1.ToDots(&ctxt))
		v2 := StringToValue("1.0" + un.String())
		s2 := fmt.Sprintf("%v = %v dots", v2, v2.ToDots(&ctxt))
		if s1 != s2 {
			t.Errorf("strings don't match: %v != %v\n", s1, s2)
			// } else {
			// 	fmt.Printf("%v = %v\n", s1, s2)
		}
	}
	v1 := In(1)
	v2 := v1.Convert(UnitPx, &ctxt)
	s1 := fmt.Sprintf("%v dots\n", v1.ToDots(&ctxt))
	s2 := fmt.Sprintf("%v dots\n", v2.ToDots(&ctxt))
	if s1 != s2 {
		t.Errorf("strings don't match: %v != %v\n", s1, s2)
	}

	tolassert.Equal(t, 72, ctxt.Convert(1, UnitIn, UnitPt))
	tolassert.Equal(t, 25.4/72.0, ctxt.Convert(1, UnitPt, UnitMm))
}
