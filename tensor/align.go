// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"slices"
)

// AlignShapes aligns the shapes of two tensors, a and b for a binary
// computation producing an output, returning the effective aligned shapes
// for a, b, and the output, all with the same number of dimensions.
// Alignment proceeds from the innermost dimension out, with 1s provided
// beyond the number of dimensions for a or b.
// The output has the max of the dimension sizes for each dimension.
// An error is returned if the rules of alignment are violated:
// each dimension size must be either the same, or one of them
// is equal to 1. This corresponds to the "broadcasting" logic of NumPy.
func AlignShapes(a, b Tensor) (as, bs, os *Shape, err error) {
	asz := a.ShapeSizes()
	bsz := b.ShapeSizes()
	an := len(asz)
	bn := len(bsz)
	n := max(an, bn)
	osizes := make([]int, n)
	asizes := make([]int, n)
	bsizes := make([]int, n)
	for d := range n {
		ai := an - 1 - d
		bi := bn - 1 - d
		oi := n - 1 - d
		ad := 1
		bd := 1
		if ai >= 0 {
			ad = asz[ai]
		}
		if bi >= 0 {
			bd = bsz[bi]
		}
		if ad != bd && !(ad == 1 || bd == 1) {
			err = fmt.Errorf("tensor.AlignShapes: output dimension %d does not align for a=%d b=%d: must be either the same or one of them is a 1", oi, ad, bd)
			return
		}
		od := max(ad, bd)
		osizes[oi] = od
		asizes[oi] = ad
		bsizes[oi] = bd
	}
	as = NewShape(asizes...)
	bs = NewShape(bsizes...)
	os = NewShape(osizes...)
	return
}

// WrapIndex1D returns the 1d flat index for given n-dimensional index
// based on given shape, where any singleton dimension sizes cause the
// resulting index value to remain at 0, effectively causing that dimension
// to wrap around, while the other tensor is presumably using the full range
// of the values along this dimension. See [AlignShapes] for more info.
func WrapIndex1D(sh *Shape, i ...int) int {
	nd := sh.NumDims()
	ai := slices.Clone(i)
	for d := range nd {
		if sh.DimSize(d) == 1 {
			ai[d] = 0
		}
	}
	return sh.IndexTo1D(ai...)
}
