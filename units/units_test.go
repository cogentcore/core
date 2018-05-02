// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"fmt"
	"testing"
)

func TestValCvt(t *testing.T) {
	var ctxt Context
	ctxt.Defaults()
	for un := Pct; un <= Dp; un++ {
		v1 := NewValue(1.0, un)
		s1 := fmt.Sprintf("%v = %v dots", v1, v1.ToDots(&ctxt))
		v2 := StringToValue("1.0" + UnitNames[un])
		s2 := fmt.Sprintf("%v = %v dots", v2, v2.ToDots(&ctxt))
		if s1 != s2 {
			t.Errorf("strings don't match: %v != %v\n", s1, s2)
			// } else {
			// 	fmt.Printf("%v = %v\n", s1, s2)
		}
	}
	v1 := NewValue(1.0, In)
	v2 := v1.Convert(Px, &ctxt)
	s1 := fmt.Sprintf("%v dots\n", v1.ToDots(&ctxt))
	s2 := fmt.Sprintf("%v dots\n", v2.ToDots(&ctxt))
	if s1 != s2 {
		t.Errorf("strings don't match: %v != %v\n", s1, s2)
	}
}
