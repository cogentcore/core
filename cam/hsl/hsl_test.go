// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hsl

import (
	"image/color"
	"testing"

	"cogentcore.org/core/glop/tolassert"
	"github.com/stretchr/testify/assert"
)

func TestHSL(t *testing.T) {
	assert.Equal(t, HSL{100, 0.87, 0.56, 1}, New(100, 0.87, 0.56))

	want := HSL{20.583939, 0.6372093, 0.5576132, 0.9529412}
	assert.Equal(t, want, Model.Convert(want))
	have := Model.Convert(color.RGBA{204, 114, 67, 243}).(HSL)
	tolassert.Equal(t, want.H, have.H)
	tolassert.Equal(t, want.S, have.S)
	tolassert.Equal(t, want.L, have.L)
	tolassert.Equal(t, want.A, have.A)

	r, g, b, a := want.RGBA()
	assert.Equal(t, uint32(0xcccc), r)
	assert.Equal(t, uint32(0x7272), g)
	assert.Equal(t, uint32(0x4343), b)
	assert.Equal(t, uint32(0xf3f3), a)

	assert.Equal(t, color.RGBA{204, 114, 67, 243}, want.AsRGBA())

	have = HSL{}
	have.SetUint32(r, g, b, a)
	tolassert.Equal(t, want.H, have.H)
	tolassert.Equal(t, want.S, have.S)
	tolassert.Equal(t, want.L, have.L)
	tolassert.Equal(t, want.A, have.A)

	have = HSL{}
	have.SetColor(want)
	tolassert.Equal(t, want.H, have.H)
	tolassert.Equal(t, want.S, have.S)
	tolassert.Equal(t, want.L, have.L)
	tolassert.Equal(t, want.A, have.A)

	assert.Equal(t, "hsl(86, 0.54, 0.97)", New(86, 0.54, 0.97).String())
}
