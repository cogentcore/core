// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"image/color"
	"testing"

	"cogentcore.org/core/glop/tolassert"
	"cogentcore.org/core/mat32"
	"github.com/stretchr/testify/assert"
)

func TestHCT(t *testing.T) {
	h := SRGBToHCT(1, 1, 1)
	// fmt.Printf("%#v\n", h)
	tolassert.EqualTol(t, 209.492, h.Hue, 0.01)
	tolassert.Equal(t, 2.869, h.Chroma)
	tolassert.Equal(t, 100, h.Tone)

	r, g, b := SolveToRGB(120, 60, 50)
	h = SRGBToHCT(r, g, b)
	tolassert.Equal(t, 120.114, h.Hue)
	tolassert.Equal(t, 52.82, h.Chroma) // can't do 60
	tolassert.Equal(t, 50, h.Tone)
	// fmt.Printf("r: %g, g %g, b %g  hr %X, hg %X, hb %X, hct: %v\n", r, g, b, int(r*255), int(g*255), int(b*255), h)

	want := HCT{134.64685, 75.31438, 80.47883, 0.5384615, 0.8675214, 0.24786325, 0.91764706}
	assert.Equal(t, want, Model.Convert(color.RGBA{126, 203, 58, 234}))

	ru, gu, bu, au := want.RGBA()
	assert.Equal(t, uint32(0x7e7e), ru)
	assert.Equal(t, uint32(0xcbcb), gu)
	assert.Equal(t, uint32(0x3a3a), bu)
	assert.Equal(t, uint32(0xeaea), au)
}

func TestHCTAll(t *testing.T) {
	hues := []float32{15, 45, 75, 105, 135, 165, 195, 225, 255, 285, 315, 345}
	chromas := []float32{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	tones := []float32{20, 30, 40, 50, 60, 70, 80}

	for _, hue := range hues {
		for _, chroma := range chromas {
			for _, tone := range tones {
				h := New(hue, chroma, tone)
				hs := h.String()
				if chroma > 0 {
					tolassert.EqualTol(t, hue, h.Hue, 4, hs)
				}
				if h.Chroma > chroma+2.5 {
					t.Errorf("expected chroma value: %g != %g with tolerance: %g for h: %s\n", chroma, h.Chroma, 2.5, hs)
				}

				// todo: add colorisonboundary

				if !(h.Hue > 209 && h.Hue < 210 && h.Chroma > 0.78) { // that value doesn't work!
					tolassert.EqualTol(t, tone, h.Tone, 0.5, hs)
				}
			}
		}
	}
}

func TestHCTSet(t *testing.T) {
	assertEqual := func(want, have HCT) {
		t.Helper()
		tolassert.EqualTol(t, want.Hue, have.Hue, 1)
		tolassert.EqualTol(t, want.Chroma, have.Chroma, 1)
		tolassert.EqualTol(t, want.Tone, have.Tone, 1)
	}
	have := New(100, 80, 60)

	have.SetHue(200)
	assertEqual(New(200, 80, 60), have)

	have.SetChroma(45)
	assertEqual(New(200, 45, 60), have)

	have.SetTone(83)
	assertEqual(New(200, 45, 83), have)

	assertEqual(New(167, 45, 83), have.WithHue(167))
	assertEqual(New(200, 61, 83), have.WithChroma(61))
	assertEqual(New(200, 45, 57), have.WithTone(57))
}

func TestGetAxis(t *testing.T) {
	assert.Equal(t, float32(-1), GetAxis(mat32.Vec3{}, 5))
}

func BenchmarkHCT(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New(120, 45, 56)
	}
}
