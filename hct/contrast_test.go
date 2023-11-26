// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"testing"
)

func TestToneContrastRatio(t *testing.T) {
	type data struct {
		a    float32
		b    float32
		want float32
	}
	tests := []data{
		{0, 100, 21},
		{100, 0, 21},
		{50, 50, 1},
	}
	for i, test := range tests {
		res := ToneContrastRatio(test.a, test.b)
		if res != test.want {
			t.Errorf("%d: expected %g but got %g", i, test.want, res)
		}
	}
}
