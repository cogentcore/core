// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package num

import "testing"

func TestConvert(t *testing.T) {
	f32 := Convert[float32](int(2))
	if f32 != 2 {
		t.Errorf("failed: %g != 2", f32)
	}
	SetNumber(&f32, uint8(5))
	if f32 != 5 {
		t.Errorf("failed: %g != 5", f32)
	}
}

func TestBool(t *testing.T) {
	b := ToBool(1)
	if !b {
		t.Errorf("failed: %v != true", b)
	}
	b = ToBool(0.0)
	if b {
		t.Errorf("failed: %v != false", b)
	}
	f32 := FromBool[float32](true)
	if f32 != 1 {
		t.Errorf("failed: %g != 1", f32)
	}
	SetFromBool(&f32, false)
	if f32 != 0 {
		t.Errorf("failed: %g != 0", f32)
	}
}

func TestAbs(t *testing.T) {
	i := Abs(-22)
	if i != 22 {
		t.Errorf("failed: %d != 22", i)
	}
	// this correctly does not compile:
	// ib := Abs(uint8(5))
	f := Abs(-4.31)
	if f != 4.31 {
		t.Errorf("failed: %g != 4.31", f)
	}
}
