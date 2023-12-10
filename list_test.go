// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

/*
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
*/
