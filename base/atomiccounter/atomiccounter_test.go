// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package atomiccounter

import "testing"

func TestCtr(t *testing.T) {
	var a Counter
	if a.Value() != 0 {
		t.Error("Ctr.Value() != 0")
	}
	if a.Inc() != 1 {
		t.Error("Ctr.Inc() != 1")
	}
	if a.Value() != 1 {
		t.Error("Ctr.Value() != 1")
	}
	if a.Add(5) != 6 {
		t.Error("Ctr.Add(5) != 6")
	}
	if a.Value() != 6 {
		t.Error("Ctr.Value() != 6")
	}
	if a.Sub(2) != 4 {
		t.Error("Ctr.Sub(2) != 4")
	}
	if a.Value() != 4 {
		t.Error("Ctr.Value() != 4")
	}
	if a.Dec() != 3 {
		t.Error("Ctr.Dec() != 3")
	}
	if a.Value() != 3 {
		t.Error("Ctr.Value() != 3")
	}
	if a.Swap(7) != 3 {
		t.Error("Ctr.Swap(7) != 3")
	}
	if a.Value() != 7 {
		t.Error("Ctr.Value() != 7")
	}
	if a.Set(9); a.Value() != 9 {
		t.Error("Ctr.Set(9); Ctr.Value() != 9")
	}
}
