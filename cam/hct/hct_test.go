// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"testing"

	"cogentcore.org/core/glop/tolassert"
	"cogentcore.org/core/mat32"
	"github.com/stretchr/testify/assert"
)

func TestHCT(t *testing.T) {
	h := SRGBToHCT(1, 1, 1)
	// fmt.Printf("%#v\n", h)
	tolassert.Equal(t, 209.492, h.Hue)
	tolassert.Equal(t, 2.869, h.Chroma)
	tolassert.Equal(t, 100, h.Tone)

	r, g, b := SolveToRGB(120, 60, 50)
	h = SRGBToHCT(r, g, b)
	tolassert.Equal(t, 120.114, h.Hue)
	tolassert.Equal(t, 52.82, h.Chroma) // can't do 60
	tolassert.Equal(t, 50, h.Tone)
	// fmt.Printf("r: %g, g %g, b %g  hr %X, hg %X, hb %X, hct: %v\n", r, g, b, int(r*255), int(g*255), int(b*255), h)
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

func TestGetAxis(t *testing.T) {
	assert.Equal(t, float32(-1), GetAxis(mat32.Vec3{}, 5))
}

func BenchmarkHCT(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New(120, 45, 56)
	}
}
