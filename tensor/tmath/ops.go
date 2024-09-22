// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("Add", Add, 1)
	tensor.AddFunc("Sub", Sub, 1)
	tensor.AddFunc("Mul", Mul, 1)
	tensor.AddFunc("Div", Div, 1)
}

// Add adds two tensors into output.
func Add(a, b, out tensor.Tensor) {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if errors.Log(err) != nil {
		return
	}
	if err := tensor.SetShapeMustBeValues(out, os); errors.Log(err) != nil {
		return
	}
	olen := os.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return olen
	},
		func(idx int, tsr ...tensor.Tensor) {
			oi := os.IndexFrom1D(idx)
			ai := tensor.WrapIndex1D(as, oi...)
			bi := tensor.WrapIndex1D(bs, oi...)
			out.SetFloat1D(tsr[0].Float1D(ai)+tsr[1].Float1D(bi), idx)
		}, a, b, out)
}

// Sub subtracts two tensors into output.
func Sub(a, b, out tensor.Tensor) {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if errors.Log(err) != nil {
		return
	}
	if err := tensor.SetShapeMustBeValues(out, os); errors.Log(err) != nil {
		return
	}
	olen := os.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return olen
	},
		func(idx int, tsr ...tensor.Tensor) {
			oi := os.IndexFrom1D(idx)
			ai := tensor.WrapIndex1D(as, oi...)
			bi := tensor.WrapIndex1D(bs, oi...)
			out.SetFloat1D(tsr[0].Float1D(ai)-tsr[1].Float1D(bi), idx)
		}, a, b, out)
}

// Mul multiplies two tensors into output.
func Mul(a, b, out tensor.Tensor) {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if errors.Log(err) != nil {
		return
	}
	if err := tensor.SetShapeMustBeValues(out, os); errors.Log(err) != nil {
		return
	}
	olen := os.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return olen
	},
		func(idx int, tsr ...tensor.Tensor) {
			oi := os.IndexFrom1D(idx)
			ai := tensor.WrapIndex1D(as, oi...)
			bi := tensor.WrapIndex1D(bs, oi...)
			out.SetFloat1D(tsr[0].Float1D(ai)*tsr[1].Float1D(bi), idx)
		}, a, b, out)
}

// Div divides two tensors into output.
func Div(a, b, out tensor.Tensor) {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if errors.Log(err) != nil {
		return
	}
	if err := tensor.SetShapeMustBeValues(out, os); errors.Log(err) != nil {
		return
	}
	olen := os.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return olen
	},
		func(idx int, tsr ...tensor.Tensor) {
			oi := os.IndexFrom1D(idx)
			ai := tensor.WrapIndex1D(as, oi...)
			bi := tensor.WrapIndex1D(bs, oi...)
			out.SetFloat1D(tsr[0].Float1D(ai)/tsr[1].Float1D(bi), idx)
		}, a, b, out)
}
