// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"
)

func TestV3AngleTo(t *testing.T) {
	ref := V3(1, 0, 0)
	for ang := float32(0); ang < Pi*2; ang += Pi / 10 {
		cos := Cos(ang)
		sin := Sin(ang)
		v := V3(cos, sin, 0)
		vang := v.AngleTo(ref)
		vcos := Cos(vang)
		vsin := Sin(vang)
		// cross := v.Cross(ref)
		// fmt.Printf("ang: %v  cross: %v\n", ang, cross)
		if Abs(vcos-cos) > 1.0e-6 {
			t.Errorf("Vec3.AngleTo: Cos: %v != orig Cos: %v\n", vcos, cos)
		}
		if Abs(vsin-sin) > 1.0e-6 {
			t.Errorf("Vec3.AngleTo: Sin: %v != orig Sin: %v\n", vsin, sin)
		}
	}
	ref = V3(0, 1, 0)
	for ang := float32(0); ang < Pi*2; ang += Pi / 10 {
		cos := Cos(ang)
		sin := Sin(ang)
		v := V3(0, cos, sin)
		vang := v.AngleTo(ref)
		vcos := Cos(vang)
		vsin := Sin(vang)
		// cross := v.Cross(ref)
		// fmt.Printf("ang: %v  cross: %v\n", ang, cross)
		if Abs(vcos-cos) > 1.0e-6 {
			t.Errorf("Vec3.AngleTo: Cos: %v != orig Cos: %v\n", vcos, cos)
		}
		if Abs(vsin-sin) > 1.0e-6 {
			t.Errorf("Vec3.AngleTo: Sin: %v != orig Sin: %v\n", vsin, sin)
		}
	}
	ref = V3(0, 0, 1)
	for ang := float32(0); ang < Pi*2; ang += Pi / 10 {
		cos := Cos(ang)
		sin := Sin(ang)
		v := V3(sin, 0, cos)
		vang := v.AngleTo(ref)
		vcos := Cos(vang)
		vsin := Sin(vang)
		// cross := v.Cross(ref)
		// fmt.Printf("ang: %v  cross: %v\n", ang, cross)
		if Abs(vcos-cos) > 1.0e-6 {
			t.Errorf("Vec3.AngleTo: Cos: %v != orig Cos: %v\n", vcos, cos)
		}
		if Abs(vsin-sin) > 1.0e-6 {
			t.Errorf("Vec3.AngleTo: Sin: %v != orig Sin: %v\n", vsin, sin)
		}
	}
}

func TestV2AngleTo(t *testing.T) {
	ref := V2(1, 0)
	for ang := float32(0); ang < Pi*2; ang += Pi / 10 {
		cos := Cos(ang)
		sin := Sin(ang)
		v := V2(cos, sin)
		vang := v.AngleTo(ref)
		vcos := Cos(vang)
		vsin := Sin(vang)
		// cross := v.Cross(ref)
		// fmt.Printf("ang: %v  cross: %v\n", ang, cross)
		if Abs(vcos-cos) > 1.0e-6 {
			t.Errorf("Vec2.AngleTo: Cos: %v != orig Cos: %v\n", vcos, cos)
		}
		if Abs(vsin-sin) > 1.0e-6 {
			t.Errorf("Vec2.AngleTo: Sin: %v != orig Sin: %v\n", vsin, sin)
		}
	}
}
