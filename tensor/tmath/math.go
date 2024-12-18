// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"math"

	"cogentcore.org/core/tensor"
)

func Abs(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(AbsOut, in)
}

func AbsOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Abs(a) }, in, out)
}

func Acos(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(AcosOut, in)
}

func AcosOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Acos(a) }, in, out)
}

func Acosh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(AcoshOut, in)
}

func AcoshOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Acosh(a) }, in, out)
}

func Asin(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(AsinOut, in)
}

func AsinOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Asin(a) }, in, out)
}

func Asinh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(AsinhOut, in)
}

func AsinhOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Asinh(a) }, in, out)
}

func Atan(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(AtanOut, in)
}

func AtanOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Atan(a) }, in, out)
}

func Atanh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(AtanhOut, in)
}

func AtanhOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Atanh(a) }, in, out)
}

func Cbrt(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(CbrtOut, in)
}

func CbrtOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Cbrt(a) }, in, out)
}

func Ceil(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(CeilOut, in)
}

func CeilOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Ceil(a) }, in, out)
}

func Cos(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(CosOut, in)
}

func CosOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Cos(a) }, in, out)
}

func Cosh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(CoshOut, in)
}

func CoshOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Cosh(a) }, in, out)
}

func Erf(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(ErfOut, in)
}

func ErfOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Erf(a) }, in, out)
}

func Erfc(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(ErfcOut, in)
}

func ErfcOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Erfc(a) }, in, out)
}

func Erfcinv(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(ErfcinvOut, in)
}

func ErfcinvOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Erfcinv(a) }, in, out)
}

func Erfinv(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(ErfinvOut, in)
}

func ErfinvOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Erfinv(a) }, in, out)
}

func Exp(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(ExpOut, in)
}

func ExpOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Exp(a) }, in, out)
}

func Exp2(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(Exp2Out, in)
}

func Exp2Out(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Exp2(a) }, in, out)
}

func Expm1(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(Expm1Out, in)
}

func Expm1Out(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Expm1(a) }, in, out)
}

func Floor(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(FloorOut, in)
}

func FloorOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Floor(a) }, in, out)
}

func Gamma(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(GammaOut, in)
}

func GammaOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Gamma(a) }, in, out)
}

func J0(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(J0Out, in)
}

func J0Out(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.J0(a) }, in, out)
}

func J1(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(J1Out, in)
}

func J1Out(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.J1(a) }, in, out)
}

func Log(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(LogOut, in)
}

func LogOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Log(a) }, in, out)
}

func Log10(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(Log10Out, in)
}

func Log10Out(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Log10(a) }, in, out)
}

func Log1p(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(Log1pOut, in)
}

func Log1pOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Log1p(a) }, in, out)
}

func Log2(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(Log2Out, in)
}

func Log2Out(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Log2(a) }, in, out)
}

func Logb(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(LogbOut, in)
}

func LogbOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Logb(a) }, in, out)
}

func Round(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(RoundOut, in)
}

func RoundOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Round(a) }, in, out)
}

func RoundToEven(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(RoundToEvenOut, in)
}

func RoundToEvenOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.RoundToEven(a) }, in, out)
}

func Sin(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(SinOut, in)
}

func SinOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Sin(a) }, in, out)
}

func Sinh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(SinhOut, in)
}

func SinhOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Sinh(a) }, in, out)
}

func Sqrt(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(SqrtOut, in)
}

func SqrtOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Sqrt(a) }, in, out)
}

func Tan(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(TanOut, in)
}

func TanOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Tan(a) }, in, out)
}

func Tanh(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(TanhOut, in)
}

func TanhOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Tanh(a) }, in, out)
}

func Trunc(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(TruncOut, in)
}

func TruncOut(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Trunc(a) }, in, out)
}

func Y0(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(Y0Out, in)
}

func Y0Out(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Y0(a) }, in, out)
}

func Y1(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Float64(Y1Out, in)
}

func Y1Out(in tensor.Tensor, out tensor.Values) error {
	return tensor.FloatFuncOut(1, func(a float64) float64 { return math.Y1(a) }, in, out)
}

//////// Binary

func Atan2(y, x tensor.Tensor) tensor.Values {
	return tensor.CallOut2(Atan2Out, y, x)
}

func Atan2Out(y, x tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(1, func(a, b float64) float64 { return math.Atan2(a, b) }, y, x, out)
}

func Copysign(x, y tensor.Tensor) tensor.Values {
	return tensor.CallOut2(CopysignOut, x, y)
}

func CopysignOut(x, y tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(1, func(a, b float64) float64 { return math.Copysign(a, b) }, x, y, out)
}

func Dim(x, y tensor.Tensor) tensor.Values {
	return tensor.CallOut2(DimOut, x, y)
}

func DimOut(x, y tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(1, func(a, b float64) float64 { return math.Dim(a, b) }, x, y, out)
}

func Hypot(x, y tensor.Tensor) tensor.Values {
	return tensor.CallOut2(HypotOut, x, y)
}

func HypotOut(x, y tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(1, func(a, b float64) float64 { return math.Hypot(a, b) }, x, y, out)
}

func Max(x, y tensor.Tensor) tensor.Values {
	return tensor.CallOut2(MaxOut, x, y)
}

func MaxOut(x, y tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(1, func(a, b float64) float64 { return math.Max(a, b) }, x, y, out)
}

func Min(x, y tensor.Tensor) tensor.Values {
	return tensor.CallOut2(MinOut, x, y)
}

func MinOut(x, y tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(1, func(a, b float64) float64 { return math.Min(a, b) }, x, y, out)
}

func Nextafter(x, y tensor.Tensor) tensor.Values {
	return tensor.CallOut2(NextafterOut, x, y)
}

func NextafterOut(x, y tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(1, func(a, b float64) float64 { return math.Nextafter(a, b) }, x, y, out)
}

func Pow(x, y tensor.Tensor) tensor.Values {
	return tensor.CallOut2(PowOut, x, y)
}

func PowOut(x, y tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(1, func(a, b float64) float64 { return math.Pow(a, b) }, x, y, out)
}

func Remainder(x, y tensor.Tensor) tensor.Values {
	return tensor.CallOut2(RemainderOut, x, y)
}

func RemainderOut(x, y tensor.Tensor, out tensor.Values) error {
	return tensor.FloatBinaryFuncOut(1, func(a, b float64) float64 { return math.Remainder(a, b) }, x, y, out)
}

/*
func Nextafter32(x, y float32) (r float32)

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
