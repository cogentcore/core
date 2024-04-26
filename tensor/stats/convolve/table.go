// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package convolve

import (
	"reflect"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// SmoothTable returns a cloned table with each of the floating-point
// columns in the source table smoothed over rows.
// khalf is the half-width of the Gaussian smoothing kernel,
// where larger values produce more smoothing.  A sigma of .5
// is used for the kernel.
func SmoothTable(src *table.Table, khalf int) *table.Table {
	k64 := GaussianKernel64(khalf, .5)
	k32 := GaussianKernel32(khalf, .5)
	dest := src.Clone()
	for ci, sci := range src.Columns {
		switch sci.DataType() {
		case reflect.Float32:
			sc := sci.(*tensor.Float32)
			dc := dest.Columns[ci].(*tensor.Float32)
			Slice32(&dc.Values, sc.Values, k32)
		case reflect.Float64:
			sc := sci.(*tensor.Float64)
			dc := dest.Columns[ci].(*tensor.Float64)
			Slice64(&dc.Values, sc.Values, k64)
		}
	}
	return dest
}
