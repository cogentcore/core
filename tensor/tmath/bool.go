// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("tmath.Equal", Equal)
	tensor.AddFunc("tmath.Less", Less)
	tensor.AddFunc("tmath.Greater", Greater)
	tensor.AddFunc("tmath.NotEqual", NotEqual)
	tensor.AddFunc("tmath.LessEqual", LessEqual)
	tensor.AddFunc("tmath.GreaterEqual", GreaterEqual)
}

// Equal stores in the output the bool value a == b.
func Equal(a, b tensor.Tensor) tensor.Tensor {
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
func Less(a, b tensor.Tensor) tensor.Tensor {
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
func Greater(a, b tensor.Tensor) tensor.Tensor {
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
func NotEqual(a, b tensor.Tensor) tensor.Tensor {
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
func LessEqual(a, b tensor.Tensor) tensor.Tensor {
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
func GreaterEqual(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2Bool(GreaterEqualOut, a, b)
}

// GreaterEqualOut stores in the output the bool value a >= b.
func GreaterEqualOut(a, b tensor.Tensor, out *tensor.Bool) error {
	if a.IsString() {
		return tensor.BoolStringsFuncOut(func(a, b string) bool { return a >= b }, a, b, out)
	}
	return tensor.BoolFloatsFuncOut(func(a, b float64) bool { return a >= b }, a, b, out)
}
