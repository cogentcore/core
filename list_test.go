// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"fmt"
	"math"
	"testing"

	"goki.dev/cam/hct"
	"goki.dev/mat32/v2"
)

func TestList(t *testing.T) {
	type data struct {
		n      int
		chroma float32
		tone   float32
	}
	tests := []data{
		{0, 48, 40},
		{1, 52, 40},
		{2, 30, 80},
		{6, 12, 18},
	}

	almostEqual := func(x, y float32) bool {
		return mat32.Abs(x-y) < 2.5 || x > 359 && y < 1 || y > 359 && x < 1
	}

	for i, test := range tests {
		list := List(test.n, test.chroma, test.tone)
		if len(list) != test.n {
			t.Errorf("expected length %d but got %d", test.n, len(list))
		}
		if test.n == 0 {
			continue
		}
		inc := 360 / float32(test.n)
		for j, l := range list {
			h := hct.FromColor(l)
			if !almostEqual(h.Chroma, test.chroma) {
				t.Errorf("%d.%d: expected chroma %g but got %g", i, j, test.chroma, h.Chroma)
			}
			if !almostEqual(h.Tone, test.tone) {
				t.Errorf("%d.%d: expected tone %g but got %g", i, j, test.tone, h.Tone)
			}
			ehue := float32(j) * inc
			if !almostEqual(h.Hue, ehue) {
				t.Errorf("%d.%d: expected hue %g but got %g", i, j, ehue, h.Hue)
			}
		}
	}
}

func TestListExpPow(t *testing.T) {
	ndiv := 2.0
	for v := 0.0; v < 100; v++ {
		// nary := []int{}
		nb := int(math.Ceil(math.Log(v) / math.Log(ndiv)))
		rv := 0.0
		for i := 0; i <= nb; i++ {
			pbase := math.Pow(ndiv, float64(i))
			base := math.Pow(ndiv, float64(i+1))
			dv := math.Floor((math.Mod(v, base)) / pbase)
			// nary = append(nary, int(dv))
			iv := dv * (1.0 / base)
			rv += iv
			fmt.Println("v: ", v, "i: ", i, "base: ", base, "pbase: ", pbase, "iv: ", iv, "dv: ", dv)
		}
		fmt.Printf("v: %g  rv: %7.4g\n########\n", v, rv)
	}
}

func TestListExpBin(t *testing.T) {
	for v := 0; v < 100; v++ {
		nb := int(mat32.Ceil(mat32.Log(float32(v)) / mat32.Log(2)))
		rv := float32(0)
		for i := 0; i <= nb; i++ {
			pbase := 1 << i
			base := 1 << (i + 1)
			dv := (v % base) / pbase
			iv := float32(dv) * (1 / float32(base))
			rv += iv
			fmt.Println("v: ", v, "i: ", i, "base: ", base, "pbase: ", pbase, "iv: ", iv, "dv: ", dv)
		}
		fmt.Printf("v: %d  rv: %7.4g\n########\n", v, rv)
	}
}
