// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"fmt"
	"testing"
)

func TestValCvt(t *testing.T) {
	var ctxt UnitContext
	for un := Pct; un <= Dp; un++ {
		v1 := Value{1.0, un}
		s1 := fmt.Sprintf("%v = %v dots\n", v1, v1.ToDots(&ctxt))
		v2 := StringToValue("1.0" + UnitNames[un])
		s2 := fmt.Sprintf("%v = %v dots\n", v2, v2.ToDots(&ctxt))
		if s1 != s2 {
			t.Errorf("strings don't match: %v != %v\n", s1, s2)
		}
	}
}
