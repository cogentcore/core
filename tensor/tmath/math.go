// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"math"

	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("Abs", Abs, 1)
	tensor.AddFunc("Acos", Acos, 1)
	tensor.AddFunc("Acosh", Acosh, 1)
	tensor.AddFunc("Asin", Asin, 1)
	tensor.AddFunc("Asinh", Asinh, 1)
	tensor.AddFunc("Atan", Atan, 1)
	tensor.AddFunc("Atanh", Atanh, 1)
	tensor.AddFunc("Cbrt", Cbrt, 1)
	tensor.AddFunc("Ceil", Ceil, 1)
	tensor.AddFunc("Cos", Cos, 1)
	tensor.AddFunc("Cosh", Cosh, 1)
	tensor.AddFunc("Erf", Erf, 1)
	tensor.AddFunc("Erfc", Erfc, 1)
	tensor.AddFunc("Erfcinv", Erfcinv, 1)
	tensor.AddFunc("Erfinv", Erfinv, 1)
	tensor.AddFunc("Exp", Exp, 1)
	tensor.AddFunc("Exp2", Exp2, 1)
	tensor.AddFunc("Expm1", Expm1, 1)
	tensor.AddFunc("Floor", Floor, 1)
	tensor.AddFunc("Gamma", Gamma, 1)
	tensor.AddFunc("J0", J0, 1)
	tensor.AddFunc("J1", J1, 1)
	tensor.AddFunc("Log", Log, 1)
	tensor.AddFunc("Log10", Log10, 1)
	tensor.AddFunc("Log1p", Log1p, 1)
	tensor.AddFunc("Log2", Log2, 1)
	tensor.AddFunc("Logb", Logb, 1)
	tensor.AddFunc("Round", Round, 1)
	tensor.AddFunc("RoundToEven", RoundToEven, 1)
	tensor.AddFunc("Sin", Sin, 1)
	tensor.AddFunc("Sinh", Sinh, 1)
	tensor.AddFunc("Sqrt", Sqrt, 1)
	tensor.AddFunc("Tan", Tan, 1)
	tensor.AddFunc("Tanh", Tanh, 1)
	tensor.AddFunc("Trunc", Trunc, 1)
	tensor.AddFunc("Y0", Y0, 1)
	tensor.AddFunc("Y1", Y1, 1)
}

func Abs(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Abs(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Acos(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Acos(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Acosh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Acosh(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Asin(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Asin(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Asinh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Asinh(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Atan(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Atan(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Atanh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Atanh(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Cbrt(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Cbrt(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Ceil(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Ceil(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Cos(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Cos(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Cosh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Cosh(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Erf(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Erf(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Erfc(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Erfc(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Erfcinv(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Erfcinv(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Erfinv(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Erfinv(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Exp(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Exp(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Exp2(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Exp2(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Expm1(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Expm1(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Floor(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Floor(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Gamma(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Gamma(tsr[0].Float1D(i)), i)
	}, in, out)
}

func J0(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.J0(tsr[0].Float1D(i)), i)
	}, in, out)
}

func J1(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.J1(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Log(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Log(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Log10(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Log10(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Log1p(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Log1p(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Log2(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Log2(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Logb(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Logb(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Round(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Round(tsr[0].Float1D(i)), i)
	}, in, out)
}

func RoundToEven(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.RoundToEven(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Sin(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Sin(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Sinh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Sinh(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Sqrt(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Sqrt(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Tan(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Tan(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Tanh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Tanh(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Trunc(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Trunc(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Y0(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Y0(tsr[0].Float1D(i)), i)
	}, in, out)
}

func Y1(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math.Y1(tsr[0].Float1D(i)), i)
	}, in, out)
}

/*
func Atan2(y, in, out *tensor.Indexed)
func Copysign(f, sign float64) float64
func Dim(x, y float64) float64
func Hypot(p, q float64) float64
func Max(x, y float64) float64
func Min(x, y float64) float64
func Mod(x, y float64) float64
func Nextafter(x, y float64) (r float64)
func Nextafter32(x, y float32) (r float32)
func Pow(x, y float64) float64
func Remainder(x, y float64) float64

func Inf(sign int) float64
func IsInf(f float64, sign int) bool
func IsNaN(f float64) (is bool)
func NaN() float64
func Signbit(x float64) bool

func Float32bits(f float32) uint32
func Float32frombits(b uint32) float32
func Float64bits(f float64) uint64
func Float64frombits(b uint64) float64

func FMA(x, y, z float64) float64

func Jn(n int, in, out *tensor.Indexed)
func Yn(n int, in, out *tensor.Indexed)

func Ldexp(frac float64, exp int) float64

func Ilogb(x float64) int
func Pow10(n int) float64

func Frexp(f float64) (frac float64, exp int)
func Modf(f float64) (int float64, frac float64)
func Lgamma(x float64) (lgamma float64, sign int)
func Sincos(x float64) (sin, cos float64)
*/
