// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("tmath.Equal", Equal)
	tensor.AddFunc("tmath.Less", Less)
	tensor.AddFunc("tmath.Greater", Greater)
	tensor.AddFunc("tmath.NotEqual", NotEqual)
	tensor.AddFunc("tmath.LessEqual", LessEqual)
	tensor.AddFunc("tmath.GreaterEqual", GreaterEqual)
	tensor.AddFunc("tmath.Or", Or)
	tensor.AddFunc("tmath.And", And)
	tensor.AddFunc("tmath.Not", Not)
}

// Equal stores in the output the bool value a == b.
func Equal(a, b tensor.Tensor) *tensor.Bool {
	return tensor.CallOut2Bool(EqualOut, a, b)
}

// EqualOut stores in the output the bool value a == b.
func EqualOut(a, b tensor.Tensor, out *tensor.Bool) error {
	if a.IsString() {
		return tensor.BoolStringsFuncOut(func(a, b string) bool { return a == b }, a, b, out)
	}
	return tensor.BoolFloatsFuncOut(func(a, b float64) bool { return a == b }, a, b, out)
}

// Less stores in the output the bool value a < b.
func Less(a, b tensor.Tensor) *tensor.Bool {
	return tensor.CallOut2Bool(LessOut, a, b)
}

// LessOut stores in the output the bool value a < b.
func LessOut(a, b tensor.Tensor, out *tensor.Bool) error {
	if a.IsString() {
		return tensor.BoolStringsFuncOut(func(a, b string) bool { return a < b }, a, b, out)
	}
	return tensor.BoolFloatsFuncOut(func(a, b float64) bool { return a < b }, a, b, out)
}

// Greater stores in the output the bool value a > b.
func Greater(a, b tensor.Tensor) *tensor.Bool {
	return tensor.CallOut2Bool(GreaterOut, a, b)
}

// GreaterOut stores in the output the bool value a > b.
func GreaterOut(a, b tensor.Tensor, out *tensor.Bool) error {
	if a.IsString() {
		return tensor.BoolStringsFuncOut(func(a, b string) bool { return a > b }, a, b, out)
	}
	return tensor.BoolFloatsFuncOut(func(a, b float64) bool { return a > b }, a, b, out)
}

// NotEqual stores in the output the bool value a != b.
func NotEqual(a, b tensor.Tensor) *tensor.Bool {
	return tensor.CallOut2Bool(NotEqualOut, a, b)
}

// NotEqualOut stores in the output the bool value a != b.
func NotEqualOut(a, b tensor.Tensor, out *tensor.Bool) error {
	if a.IsString() {
		return tensor.BoolStringsFuncOut(func(a, b string) bool { return a != b }, a, b, out)
	}
	return tensor.BoolFloatsFuncOut(func(a, b float64) bool { return a != b }, a, b, out)
}

// LessEqual stores in the output the bool value a <= b.
func LessEqual(a, b tensor.Tensor) *tensor.Bool {
	return tensor.CallOut2Bool(LessEqualOut, a, b)
}

// LessEqualOut stores in the output the bool value a <= b.
func LessEqualOut(a, b tensor.Tensor, out *tensor.Bool) error {
	if a.IsString() {
		return tensor.BoolStringsFuncOut(func(a, b string) bool { return a <= b }, a, b, out)
	}
	return tensor.BoolFloatsFuncOut(func(a, b float64) bool { return a <= b }, a, b, out)
}

// GreaterEqual stores in the output the bool value a >= b.
func GreaterEqual(a, b tensor.Tensor) *tensor.Bool {
	return tensor.CallOut2Bool(GreaterEqualOut, a, b)
}

// GreaterEqualOut stores in the output the bool value a >= b.
func GreaterEqualOut(a, b tensor.Tensor, out *tensor.Bool) error {
	if a.IsString() {
		return tensor.BoolStringsFuncOut(func(a, b string) bool { return a >= b }, a, b, out)
	}
	return tensor.BoolFloatsFuncOut(func(a, b float64) bool { return a >= b }, a, b, out)
}

// Or stores in the output the bool value a || b.
func Or(a, b tensor.Tensor) *tensor.Bool {
	return tensor.CallOut2Bool(OrOut, a, b)
}

// OrOut stores in the output the bool value a || b.
func OrOut(a, b tensor.Tensor, out *tensor.Bool) error {
	return tensor.BoolIntsFuncOut(func(a, b int) bool { return a > 0 || b > 0 }, a, b, out)
}

// And stores in the output the bool value a || b.
func And(a, b tensor.Tensor) *tensor.Bool {
	return tensor.CallOut2Bool(AndOut, a, b)
}

// AndOut stores in the output the bool value a || b.
func AndOut(a, b tensor.Tensor, out *tensor.Bool) error {
	return tensor.BoolIntsFuncOut(func(a, b int) bool { return a > 0 && b > 0 }, a, b, out)
}

// Not stores in the output the bool value !a.
func Not(a tensor.Tensor) *tensor.Bool {
	out := tensor.NewBool()
	errors.Log(NotOut(a, out))
	return out
}

// NotOut stores in the output the bool value !a.
func NotOut(a tensor.Tensor, out *tensor.Bool) error {
	out.SetShapeSizes(a.Shape().Sizes...)
	alen := a.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int { return alen },
		func(idx int, tsr ...tensor.Tensor) {
			out.SetBool1D(tsr[0].Int1D(idx) == 0, idx)
		}, a, out)
	return nil
}
