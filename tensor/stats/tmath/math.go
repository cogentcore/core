// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"math"

	"cogentcore.org/core/tensor"
)

func Abs(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := in.RowCellIndex(idx)
		out.Tensor.SetFloat1D(i1d, math.Abs(in.Tensor.Float1D(i1d)))
	}, in, out)
}

func Acos(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := in.RowCellIndex(idx)
		out.Tensor.SetFloat1D(i1d, math.Acos(in.Tensor.Float1D(i1d)))
	}, in, out)
}

func Acosh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := in.RowCellIndex(idx)
		out.Tensor.SetFloat1D(i1d, math.Acosh(in.Tensor.Float1D(i1d)))
	}, in, out)
}

/*
func Asin(in, out *tensor.Indexed)
func Asinh(in, out *tensor.Indexed)
func Atan(in, out *tensor.Indexed)
func Atan2(y, in, out *tensor.Indexed)
func Atanh(in, out *tensor.Indexed)
func Cbrt(in, out *tensor.Indexed)
func Ceil(in, out *tensor.Indexed)
func Copysign(f, sign float64) float64
func Cos(in, out *tensor.Indexed)
func Cosh(in, out *tensor.Indexed)
func Dim(x, y float64) float64
func Erf(in, out *tensor.Indexed)
func Erfc(in, out *tensor.Indexed)
func Erfcinv(in, out *tensor.Indexed)
func Erfinv(in, out *tensor.Indexed)
func Exp(in, out *tensor.Indexed)
func Exp2(in, out *tensor.Indexed)
func Expm1(in, out *tensor.Indexed)
func FMA(x, y, z float64) float64
func Float32bits(f float32) uint32
func Float32frombits(b uint32) float32
func Float64bits(f float64) uint64
func Float64frombits(b uint64) float64
func Floor(in, out *tensor.Indexed)
func Frexp(f float64) (frac float64, exp int)
func Gamma(in, out *tensor.Indexed)
func Hypot(p, q float64) float64
func Ilogb(x float64) int
func Inf(sign int) float64
func IsInf(f float64, sign int) bool
func IsNaN(f float64) (is bool)
func J0(in, out *tensor.Indexed)
func J1(in, out *tensor.Indexed)
func Jn(n int, in, out *tensor.Indexed)
func Ldexp(frac float64, exp int) float64
func Lgamma(x float64) (lgamma float64, sign int)
func Log(in, out *tensor.Indexed)
func Log10(in, out *tensor.Indexed)
func Log1p(in, out *tensor.Indexed)
func Log2(in, out *tensor.Indexed)
func Logb(in, out *tensor.Indexed)
func Max(x, y float64) float64
func Min(x, y float64) float64
func Mod(x, y float64) float64
func Modf(f float64) (int float64, frac float64)
func NaN() float64
func Nextafter(x, y float64) (r float64)
func Nextafter32(x, y float32) (r float32)
func Pow(x, y float64) float64
func Pow10(n int) float64
func Remainder(x, y float64) float64
func Round(in, out *tensor.Indexed)
func RoundToEven(in, out *tensor.Indexed)
func Signbit(x float64) bool
func Sin(in, out *tensor.Indexed)
func Sincos(x float64) (sin, cos float64)
func Sinh(in, out *tensor.Indexed)
func Sqrt(in, out *tensor.Indexed)
func Tan(in, out *tensor.Indexed)
func Tanh(in, out *tensor.Indexed)
func Trunc(in, out *tensor.Indexed)
func Y0(in, out *tensor.Indexed)
func Y1(in, out *tensor.Indexed)
func Yn(n int, in, out *tensor.Indexed)
*/
