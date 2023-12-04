// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"fmt"
	"testing"
)

func TestToDots(t *testing.T) {
	tests := map[Units]float32{
		UnitPx: 100,
	}
	var uc Context
	uc.Defaults()
	for unit, want := range tests {
		v := New(100, unit)
		have := v.ToDots(&uc)
		if want != have {
			t.Errorf("expected %g for %v, but got %g", want, unit, have)
		}
	}
}

func TestValCvt(t *testing.T) {
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
}
