// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("Assign", Assign, 0)
	tensor.AddFunc("AddAssign", AddAssign, 0)
	tensor.AddFunc("SubAssign", SubAssign, 0)
	tensor.AddFunc("MulAssign", MulAssign, 0)
	tensor.AddFunc("DivAssign", DivAssign, 0)

	tensor.AddFunc("Inc", Inc, 0)
	tensor.AddFunc("Dec", Dec, 0)

	tensor.AddFunc("Add", Add, 1)
	tensor.AddFunc("Sub", Sub, 1)
	tensor.AddFunc("Mul", Mul, 1)
	tensor.AddFunc("Div", Div, 1)
}

// Assign assigns values from b into a.
func Assign(a, b tensor.Tensor) error {
	as, bs, err := tensor.AlignForAssign(a, b)
	if err != nil {
		return err
	}
	alen := as.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return alen
	},
		func(idx int, tsr ...tensor.Tensor) {
			ai := as.IndexFrom1D(idx)
			bi := tensor.WrapIndex1D(bs, ai...)
			tsr[0].SetFloat1D(tsr[1].Float1D(bi), idx)
		}, a, b)
	return nil
}

// AddAssign does += add assign values from b into a.
func AddAssign(a, b tensor.Tensor) error {
	as, bs, err := tensor.AlignForAssign(a, b)
	if err != nil {
		return err
	}
	alen := as.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return alen
	},
		func(idx int, tsr ...tensor.Tensor) {
			ai := as.IndexFrom1D(idx)
			bi := tensor.WrapIndex1D(bs, ai...)
			tsr[0].SetFloat1D(tsr[0].Float1D(idx)+tsr[1].Float1D(bi), idx)
		}, a, b)
	return nil
}

// SubAssign does -= sub assign values from b into a.
func SubAssign(a, b tensor.Tensor) error {
	as, bs, err := tensor.AlignForAssign(a, b)
	if err != nil {
		return err
	}
	alen := as.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return alen
	},
		func(idx int, tsr ...tensor.Tensor) {
			ai := as.IndexFrom1D(idx)
			bi := tensor.WrapIndex1D(bs, ai...)
			tsr[0].SetFloat1D(tsr[0].Float1D(idx)-tsr[1].Float1D(bi), idx)
		}, a, b)
	return nil
}

// MulAssign does *= mul assign values from b into a.
func MulAssign(a, b tensor.Tensor) error {
	as, bs, err := tensor.AlignForAssign(a, b)
	if err != nil {
		return err
	}
	alen := as.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return alen
	},
		func(idx int, tsr ...tensor.Tensor) {
			ai := as.IndexFrom1D(idx)
			bi := tensor.WrapIndex1D(bs, ai...)
			tsr[0].SetFloat1D(tsr[0].Float1D(idx)*tsr[1].Float1D(bi), idx)
		}, a, b)
	return nil
}

// DivAssign does /= divide assign values from b into a.
func DivAssign(a, b tensor.Tensor) error {
	as, bs, err := tensor.AlignForAssign(a, b)
	if err != nil {
		return err
	}
	alen := as.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return alen
	},
		func(idx int, tsr ...tensor.Tensor) {
			ai := as.IndexFrom1D(idx)
			bi := tensor.WrapIndex1D(bs, ai...)
			tsr[0].SetFloat1D(tsr[0].Float1D(idx)/tsr[1].Float1D(bi), idx)
		}, a, b)
	return nil
}

// Inc increments values in given tensor by 1.
func Inc(a tensor.Tensor) error {
	alen := a.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return alen
	},
		func(idx int, tsr ...tensor.Tensor) {
			tsr[0].SetFloat1D(tsr[0].Float1D(idx)+1.0, idx)
		}, a)
	return nil
}

// Dec decrements values in given tensor by 1.
func Dec(a tensor.Tensor) error {
	alen := a.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return alen
	},
		func(idx int, tsr ...tensor.Tensor) {
			tsr[0].SetFloat1D(tsr[0].Float1D(idx)-1.0, idx)
		}, a)
	return nil
}

// Add adds two tensors into output.
func Add(a, b, out tensor.Tensor) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	if err := tensor.SetShapeSizesMustBeValues(out, os.Sizes...); err != nil {
		return err
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
	return nil
}

// Sub subtracts two tensors into output.
func Sub(a, b, out tensor.Tensor) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	if err := tensor.SetShapeSizesMustBeValues(out, os.Sizes...); err != nil {
		return err
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
	return nil
}

// Mul multiplies two tensors into output.
func Mul(a, b, out tensor.Tensor) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	if err := tensor.SetShapeSizesMustBeValues(out, os.Sizes...); err != nil {
		return err
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
	return nil
}

// Div divides two tensors into output.
func Div(a, b, out tensor.Tensor) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	if err := tensor.SetShapeSizesMustBeValues(out, os.Sizes...); err != nil {
		return err
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
	return nil
}
