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
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Abs(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Acos(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Acos(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Acosh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Acosh(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Asin(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Asin(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Asinh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Asinh(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Atan(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Atan(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Atanh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Atanh(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Cbrt(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Cbrt(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Ceil(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Ceil(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Cos(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Cos(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Cosh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Cosh(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Erf(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Erf(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Erfc(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Erfc(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Erfcinv(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Erfcinv(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Erfinv(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Erfinv(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Exp(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Exp(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Exp2(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Exp2(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Expm1(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Expm1(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Floor(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Floor(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Gamma(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Gamma(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func J0(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.J0(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func J1(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.J1(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Log(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Log(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Log10(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Log10(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Log1p(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Log1p(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Log2(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Log2(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Logb(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Logb(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Round(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Round(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func RoundToEven(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.RoundToEven(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Sin(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Sin(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Sinh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Sinh(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Sqrt(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Sqrt(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Tan(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Tan(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Tanh(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Tanh(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Trunc(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Trunc(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Y0(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Y0(tsr[0].Tensor.Float1D(i1d)))
	}, in, out)
}

func Y1(in, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i1d, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].Tensor.SetFloat1D(i1d, math.Y1(tsr[0].Tensor.Float1D(i1d)))
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
