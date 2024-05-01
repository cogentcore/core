// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
)

func TestVector3AngleTo(t *testing.T) {
	ref := Vec3(1, 0, 0)
	for ang := float32(0); ang < Pi*2; ang += Pi / 10 {
		cos := Cos(ang)
		sin := Sin(ang)
		v := Vec3(cos, sin, 0)
		vang := v.AngleTo(ref)
		vcos := Cos(vang)
		vsin := Sin(vang)
		tolassert.Equal(t, cos, vcos)
		tolassert.Equal(t, sin, vsin)
	}
	ref = Vec3(0, 1, 0)
	for ang := float32(0); ang < Pi*2; ang += Pi / 10 {
		cos := Cos(ang)
		sin := Sin(ang)
		v := Vec3(0, cos, sin)
		vang := v.AngleTo(ref)
		vcos := Cos(vang)
		vsin := Sin(vang)
		tolassert.Equal(t, cos, vcos)
		tolassert.Equal(t, sin, vsin)
	}
	ref = Vec3(0, 0, 1)
	for ang := float32(0); ang < Pi*2; ang += Pi / 10 {
		cos := Cos(ang)
		sin := Sin(ang)
		v := Vec3(sin, 0, cos)
		vang := v.AngleTo(ref)
		vcos := Cos(vang)
		vsin := Sin(vang)
		tolassert.Equal(t, cos, vcos)
		tolassert.Equal(t, sin, vsin)
	}
}

func TestVector2AngleTo(t *testing.T) {
	ref := Vec2(1, 0)
	for ang := float32(0); ang < Pi*2; ang += Pi / 10 {
		cos := Cos(ang)
		sin := Sin(ang)
		v := Vec2(cos, sin)
		vang := v.AngleTo(ref)
		vcos := Cos(vang)
		vsin := Sin(vang)
		tolassert.Equal(t, cos, vcos)
		tolassert.Equal(t, sin, vsin)
	}
}
