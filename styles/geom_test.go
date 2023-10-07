// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import "testing"

func TestFitIn(t *testing.T) {
	p, s := FitGeomInWindow(100, 100, 0, 200)
	if p != 100 || s != 100 {
		t.Errorf("100, 100, 0, 200")
	}
	p, s = FitGeomInWindow(150, 100, 0, 200)
	if p != 100 || s != 100 {
		t.Errorf("150, 100, 0, 200")
	}
	p, s = FitGeomInWindow(150, 200, 0, 200)
	if p != 0 || s != 200 {
		t.Errorf("150, 200, 0, 200")
	}
	p, s = FitGeomInWindow(-150, 100, 0, 200)
	if p != 0 || s != 100 {
		t.Errorf("-150, 100, 0, 200")
	}
	p, s = FitGeomInWindow(150, 300, 0, 200)
	if p != 0 || s != 200 {
		t.Errorf("150, 300, 0, 200")
	}
	p, s = FitGeomInWindow(150, 300, 50, 200)
	if p != 50 || s != 150 {
		t.Errorf("150, 300, 50, 200")
	}
}
