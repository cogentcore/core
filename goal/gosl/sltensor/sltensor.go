// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sltensor

import (
	"math"
	"reflect"

	"cogentcore.org/core/base/num"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/tensor"
)

// SetShapeSizes sets the shape of a [tensor.Number] tensor
// for use as a GPU data variable, with strides encoded as uint32
// values in the first NumDims Header values. Must use this instead
// of the SetShapeSizes method on the tensor.
func SetShapeSizes[T num.Number](tsr *tensor.Number[T], sizes ...int) {
	tsr.Shape().SetShapeSizes(sizes...)
	tsr.Shape().Header = tsr.Shape().NumDims()
	nln := tsr.Shape().Header + tsr.Shape().Len()
	tsr.Values = slicesx.SetLength(tsr.Values, nln)
	SetStrides(tsr)
}

// SetStrides sets the strides of a [tensor.Number] tensor
// for use as a GPU data variable, with strides encoded as uint32
// values in the first NumDims Header values. Must have already
// used [SetShapeSizes] to reserve header space.
func SetStrides[T num.Number](tsr *tensor.Number[T]) {
	switch {
	case reflectx.KindIsInt(tsr.DataType()):
		for i, d := range tsr.Shape().Strides {
			tsr.Values[i] = T(d)
		}
	case tsr.DataType() == reflect.Float32:
		for i, d := range tsr.Shape().Strides {
			tsr.Values[i] = T(math.Float32frombits(uint32(d)))
		}
	case tsr.DataType() == reflect.Float64:
		for i, d := range tsr.Shape().Strides {
			tsr.Values[i] = T(math.Float64frombits(uint64(d)))
		}
	}
}
