// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"math"

	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("Abs", Abs)
	tensor.AddFunc("Acos", Acos)
	tensor.AddFunc("Acosh", Acosh)
	tensor.AddFunc("Asin", Asin)
	tensor.AddFunc("Asinh", Asinh)
	tensor.AddFunc("Atan", Atan)
	tensor.AddFunc("Atanh", Atanh)
	tensor.AddFunc("Cbrt", Cbrt)
	tensor.AddFunc("Ceil", Ceil)
	tensor.AddFunc("Cos", Cos)
	tensor.AddFunc("Cosh", Cosh)
	tensor.AddFunc("Erf", Erf)
	tensor.AddFunc("Erfc", Erfc)
	tensor.AddFunc("Erfcinv", Erfcinv)
	tensor.AddFunc("Erfinv", Erfinv)
	tensor.AddFunc("Exp", Exp)
	tensor.AddFunc("Exp2", Exp2)
	tensor.AddFunc("Expm1", Expm1)
	tensor.AddFunc("Floor", Floor)
	tensor.AddFunc("Gamma", Gamma)
	tensor.AddFunc("J0", J0)
	tensor.AddFunc("J1", J1)
	tensor.AddFunc("Log", Log)
	tensor.AddFunc("Log10", Log10)
	tensor.AddFunc("Log1p", Log1p)
	tensor.AddFunc("Log2", Log2)
	tensor.AddFunc("Logb", Logb)
	tensor.AddFunc("Round", Round)
	tensor.AddFunc("RoundToEven", RoundToEven)
	tensor.AddFunc("Sin", Sin)
	tensor.AddFunc("Sinh", Sinh)
	tensor.AddFunc("Sqrt", Sqrt)
	tensor.AddFunc("Tan", Tan)
	tensor.AddFunc("Tanh", Tanh)
	tensor.AddFunc("Trunc", Trunc)
	tensor.AddFunc("Y0", Y0)
	tensor.AddFunc("Y1", Y1)
}

func Abs(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(AbsOut, in)
}

func AbsOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Abs(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Acos(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(AcosOut, in)
}

func AcosOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Acos(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Acosh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(AcoshOut, in)
}

func AcoshOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Acosh(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Asin(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(AsinOut, in)
}

func AsinOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Asin(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Asinh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(AsinhOut, in)
}

func AsinhOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Asinh(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Atan(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(AtanOut, in)
}

func AtanOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Atan(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Atanh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(AtanhOut, in)
}

func AtanhOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Atanh(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Cbrt(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(CbrtOut, in)
}

func CbrtOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Cbrt(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Ceil(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(CeilOut, in)
}

func CeilOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Ceil(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Cos(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(CosOut, in)
}

func CosOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Cos(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Cosh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(CoshOut, in)
}

func CoshOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Cosh(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Erf(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(ErfOut, in)
}

func ErfOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Erf(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Erfc(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(ErfcOut, in)
}

func ErfcOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Erfc(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Erfcinv(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(ErfcinvOut, in)
}

func ErfcinvOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Erfcinv(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Erfinv(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(ErfinvOut, in)
}

func ErfinvOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Erfinv(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Exp(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(ExpOut, in)
}

func ExpOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Exp(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Exp2(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(Exp2Out, in)
}

func Exp2Out(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Exp2(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Expm1(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(Expm1Out, in)
}

func Expm1Out(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Expm1(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Floor(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(FloorOut, in)
}

func FloorOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Floor(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Gamma(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(GammaOut, in)
}

func GammaOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Gamma(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func J0(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(J0Out, in)
}

func J0Out(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.J0(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func J1(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(J1Out, in)
}

func J1Out(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.J1(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Log(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(LogOut, in)
}

func LogOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Log(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Log10(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(Log10Out, in)
}

func Log10Out(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Log10(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Log1p(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(Log1pOut, in)
}

func Log1pOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Log1p(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Log2(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(Log2Out, in)
}

func Log2Out(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Log2(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Logb(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(LogbOut, in)
}

func LogbOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Logb(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Round(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(RoundOut, in)
}

func RoundOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Round(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func RoundToEven(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(RoundToEvenOut, in)
}

func RoundToEvenOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.RoundToEven(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Sin(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(SinOut, in)
}

func SinOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Sin(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Sinh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(SinhOut, in)
}

func SinhOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Sinh(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Sqrt(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(SqrtOut, in)
}

func SqrtOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Sqrt(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Tan(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(TanOut, in)
}

func TanOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Tan(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Tanh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(TanhOut, in)
}

func TanhOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Tanh(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Trunc(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(TruncOut, in)
}

func TruncOut(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Trunc(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Y0(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(Y0Out, in)
}

func Y0Out(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Y0(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

func Y1(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(Y1Out, in)
}

func Y1Out(in tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math.Y1(tsr[0].Float1D(idx)), idx)
	}, in, out)
	return nil
}

/*
func Atan2(y, in tensor.Tensor, out tensor.Values)
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

func Jn(n int, in tensor.Tensor, out tensor.Values)
func Yn(n int, in tensor.Tensor, out tensor.Values)

func Ldexp(frac float64, exp int) float64

func Ilogb(x float64) int
func Pow10(n int) float64

func Frexp(f float64) (frac float64, exp int)
func Modf(f float64) (int float64, frac float64)
func Lgamma(x float64) (lgamma float64, sign int)
func Sincos(x float64) (sin, cos float64)
*/
