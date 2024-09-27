// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"math"

	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("tmath.Assign", Assign)
	tensor.AddFunc("tmath.AddAssign", AddAssign)
	tensor.AddFunc("tmath.SubAssign", SubAssign)
	tensor.AddFunc("tmath.MulAssign", MulAssign)
	tensor.AddFunc("tmath.DivAssign", DivAssign)
	tensor.AddFunc("tmath.ModAssign", ModAssign)

	tensor.AddFunc("tmath.Inc", Inc)
	tensor.AddFunc("tmath.Dec", Dec)

	tensor.AddFunc("tmath.Add", Add)
	tensor.AddFunc("tmath.Sub", Sub)
	tensor.AddFunc("tmath.Mul", Mul)
	tensor.AddFunc("tmath.Div", Div)
	tensor.AddFunc("tmath.Mod", Mod)

	tensor.AddFunc("tmath.Negate", Negate)
}

// Assign assigns values from b into a.
func Assign(a, b tensor.Tensor) error {
	return tensor.FloatAssignFunc(func(a, b float64) float64 { return b }, a, b)
}

// AddAssign does += add assign values from b into a.
func AddAssign(a, b tensor.Tensor) error {
	if a.IsString() {
		return tensor.StringAssignFunc(func(a, b string) string { return a + b }, a, b)
	}
	return tensor.FloatAssignFunc(func(a, b float64) float64 { return a + b }, a, b)
}

// SubAssign does -= sub assign values from b into a.
func SubAssign(a, b tensor.Tensor) error {
	return tensor.FloatAssignFunc(func(a, b float64) float64 { return a - b }, a, b)
}

// MulAssign does *= mul assign values from b into a.
func MulAssign(a, b tensor.Tensor) error {
	return tensor.FloatAssignFunc(func(a, b float64) float64 { return a * b }, a, b)
}

// DivAssign does /= divide assign values from b into a.
func DivAssign(a, b tensor.Tensor) error {
	return tensor.FloatAssignFunc(func(a, b float64) float64 { return a / b }, a, b)
}

// ModAssign does %= modulus assign values from b into a.
func ModAssign(a, b tensor.Tensor) error {
	return tensor.FloatAssignFunc(func(a, b float64) float64 { return math.Mod(a, b) }, a, b)
}

// Inc increments values in given tensor by 1.
func Inc(a tensor.Tensor) error {
	alen := a.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int { return alen },
		func(idx int, tsr ...tensor.Tensor) {
			tsr[0].SetFloat1D(tsr[0].Float1D(idx)+1.0, idx)
		}, a)
	return nil
}

// Dec decrements values in given tensor by 1.
func Dec(a tensor.Tensor) error {
	alen := a.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int { return alen },
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
	if a.IsString() {
		return tensor.StringBinaryFuncOut(func(a, b string) string { return a + b }, a, b, out)
	}
	return tensor.FloatBinaryFuncOut(func(a, b float64) float64 { return a + b }, a, b, out)
}

// Sub subtracts tensors into output.
func Sub(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2(SubOut, a, b)
}

// SubOut subtracts two tensors into output.
func SubOut(a, b tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(func(a, b float64) float64 { return a - b }, a, b, out)
}

// Mul multiplies tensors into output.
func Mul(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2(MulOut, a, b)
}

// MulOut multiplies two tensors into output.
func MulOut(a, b tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(func(a, b float64) float64 { return a * b }, a, b, out)
}

// Div divides tensors into output.
func Div(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2(DivOut, a, b)
}

// DivOut divides two tensors into output.
func DivOut(a, b tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(func(a, b float64) float64 { return a / b }, a, b, out)
}

// Mod performs modulus a%b on tensors into output.
func Mod(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut2(ModOut, a, b)
}

// ModOut performs modulus a%b on tensors into output.
func ModOut(a, b tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(func(a, b float64) float64 { return math.Mod(a, b) }, a, b, out)
}

// Negate stores in the output the bool value -a.
func Negate(a tensor.Tensor) tensor.Tensor {
	return tensor.CallOut1(NegateOut, a)
}

// NegateOut stores in the output the bool value -a.
func NegateOut(a tensor.Tensor, out tensor.Values) error {
	out.SetShapeSizes(a.Shape().Sizes...)
	alen := a.Len()
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int { return alen },
		func(idx int, tsr ...tensor.Tensor) {
			out.SetFloat1D(-tsr[0].Float1D(idx), idx)
		}, a, out)
	return nil
}
