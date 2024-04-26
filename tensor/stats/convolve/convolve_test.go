// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package convolve

import (
	"testing"

	"cogentcore.org/core/math32"
)

func TestConv32(t *testing.T) {
	kern := GaussianKernel32(2, .5)
	// fmt.Printf("kern: %v\n", kern)
	sz := 20
	src := make([]float32, sz)
	for i := range src {
		src[i] = float32(i)
	}
	var dest []float32
	Slice32(&dest, src, kern)
	khalf := (len(kern) - 1) / 2
	for i := range src {
		if i >= khalf && i < (sz-1-khalf) {
			err := math32.Abs(src[i] - float32(i))
			if err > 1.0e-7 {
				t.Errorf("error: %d:\t%g\t->\t%g\n", i, src[i], dest[i])
			}
		}
		// fmt.Printf("%d:\t%g\t->\t%g\n", i, src[i], dest[i])
	}
}
