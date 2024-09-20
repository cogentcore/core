// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"cogentcore.org/core/tensor"
	"gonum.org/v1/gonum/mat"
)

// CopyDense copies a gonum mat.Dense matrix into given Tensor
// using standard Float64 interface
func CopyDense(to tensor.Tensor, dm *mat.Dense) {
	nr, nc := dm.Dims()
	to.SetShapeInts(nr, nc)
	idx := 0
	for ri := 0; ri < nr; ri++ {
		for ci := 0; ci < nc; ci++ {
			v := dm.At(ri, ci)
			to.SetFloat1D(v, idx)
			idx++
		}
	}
}
