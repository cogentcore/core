// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("tmath.Assign", Assign)
	tensor.AddFunc("tmath.AddAssign", AddAssign)
	tensor.AddFunc("tmath.SubAssign", SubAssign)
	tensor.AddFunc("tmath.MulAssign", MulAssign)
	tensor.AddFunc("tmath.DivAssign", DivAssign)

	tensor.AddFunc("tmath.Inc", Inc)
	tensor.AddFunc("tmath.Dec", Dec)

	tensor.AddFunc("tmath.Add", Add)
	tensor.AddFunc("tmath.Sub", Sub)
	tensor.AddFunc("tmath.Mul", Mul)
	tensor.AddFunc("tmath.Div", Div)
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
func Add(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2(AddOut, a, b)
}

// AddOut adds two tensors into output.
func AddOut(a, b tensor.Tensor, out tensor.Values) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	out.SetShapeSizes(os.Sizes...)
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

// Sub subtracts tensors into output.
func Sub(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2(SubOut, a, b)
}

// SubOut subtracts two tensors into output.
func SubOut(a, b tensor.Tensor, out tensor.Values) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	out.SetShapeSizes(os.Sizes...)
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

// Mul multiplies tensors into output.
func Mul(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2(MulOut, a, b)
}

// MulOut multiplies two tensors into output.
func MulOut(a, b tensor.Tensor, out tensor.Values) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	out.SetShapeSizes(os.Sizes...)
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

// Div divides tensors into output.
func Div(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2(DivOut, a, b)
}

// DivOut divides two tensors into output.
func DivOut(a, b tensor.Tensor, out tensor.Values) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	out.SetShapeSizes(os.Sizes...)
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

// Greater stores in the output the bool value a > b.
func Greater(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2Bool(GreaterOut, a, b)
}

// GreaterOut stores in the output the bool value a > b.
func GreaterOut(a, b tensor.Tensor, out *tensor.Bool) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	out.SetShapeSizes(os.Sizes...)
	olen := os.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return olen
	},
		func(idx int, tsr ...tensor.Tensor) {
			oi := os.IndexFrom1D(idx)
			ai := tensor.WrapIndex1D(as, oi...)
			bi := tensor.WrapIndex1D(bs, oi...)
			out.SetBool1D(tsr[0].Float1D(ai) > tsr[1].Float1D(bi), idx)
		}, a, b, out)
	return nil
}

// Less stores in the output the bool value a > b.
func Less(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2Bool(LessOut, a, b)
}

// LessOut stores in the output the bool value a > b.
func LessOut(a, b tensor.Tensor, out *tensor.Bool) error {
	as, bs, os, err := tensor.AlignShapes(a, b)
	if err != nil {
		return err
	}
	out.SetShapeSizes(os.Sizes...)
	olen := os.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return olen
	},
		func(idx int, tsr ...tensor.Tensor) {
			oi := os.IndexFrom1D(idx)
			ai := tensor.WrapIndex1D(as, oi...)
			bi := tensor.WrapIndex1D(bs, oi...)
			out.SetBool1D(tsr[0].Float1D(ai) < tsr[1].Float1D(bi), idx)
		}, a, b, out)
	return nil
}
