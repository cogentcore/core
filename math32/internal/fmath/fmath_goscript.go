// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build goscript

package fmath

import "math"

func Abs(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

func Acos(x float32) float32 {
	return float32(math.Acos(float64(x)))
}

func Acosh(x float32) float32 {
	return float32(math.Acosh(float64(x)))
}

func Asin(x float32) float32 {
	return float32(math.Asin(float64(x)))
}

func Asinh(x float32) float32 {
	return float32(math.Asinh(float64(x)))
}

func Atan(x float32) float32 {
	return float32(math.Atan(float64(x)))
}

func Atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}

func Atanh(x float32) float32 {
	return float32(math.Atanh(float64(x)))
}

func Cbrt(x float32) float32 {
	return float32(math.Cbrt(float64(x)))
}

func Ceil(x float32) float32 {
	return float32(math.Ceil(float64(x)))
}

func Copysign(x, y float32) float32 {
	return float32(math.Copysign(float64(x), float64(y)))
}

func Cos(x float32) float32 {
	return float32(math.Cos(float64(x)))
}

func Cosh(x float32) float32 {
	return float32(math.Cosh(float64(x)))
}

func Dim(x, y float32) float32 {
	return float32(math.Dim(float64(x), float64(y)))
}

func Erf(x float32) float32 {
	return float32(math.Erf(float64(x)))
}

func Erfc(x float32) float32 {
	return float32(math.Erfc(float64(x)))
}

func Exp(x float32) float32 {
	return float32(math.Exp(float64(x)))
}

func Exp2(x float32) float32 {
	return float32(math.Exp2(float64(x)))
}

func Expm1(x float32) float32 {
	return float32(math.Expm1(float64(x)))
}

func Floor(x float32) float32 {
	return float32(math.Floor(float64(x)))
}

func Frexp(f float32) (float32, int) {
	fr, e := math.Frexp(float64(f))
	return float32(fr), e
}

func Gamma(x float32) float32 {
	return float32(math.Gamma(float64(x)))
}

func Hypot(p, q float32) float32 {
	return float32(math.Hypot(float64(p), float64(q)))
}

func Ilogb(x float32) int {
	return math.Ilogb(float64(x))
}

func Inf(sign int) float32 {
	return float32(math.Inf(sign))
}

func IsInf(f float32, sign int) bool {
	return math.IsInf(float64(f), sign)
}

func IsNaN(f float32) bool {
	return f != f
}

func J0(x float32) float32 {
	return float32(math.J0(float64(x)))
}

func J1(x float32) float32 {
	return float32(math.J1(float64(x)))
}

func Jn(n int, x float32) float32 {
	return float32(math.Jn(n, float64(x)))
}

func Ldexp(frac float32, exp int) float32 {
	return float32(math.Ldexp(float64(frac), exp))
}

func Lgamma(x float32) (float32, int) {
	lg, s := math.Lgamma(float64(x))
	return float32(lg), s
}

func Log(x float32) float32 {
	return float32(math.Log(float64(x)))
}

func Log10(x float32) float32 {
	return float32(math.Log10(float64(x)))
}

func Log1p(x float32) float32 {
	return float32(math.Log1p(float64(x)))
}

func Log2(x float32) float32 {
	return float32(math.Log2(float64(x)))
}

func Logb(x float32) float32 {
	return float32(math.Logb(float64(x)))
}

func Max(x, y float32) float32 {
	return float32(math.Max(float64(x), float64(y)))
}

func Min(x, y float32) float32 {
	return float32(math.Min(float64(x), float64(y)))
}

func Mod(x, y float32) float32 {
	return float32(math.Mod(float64(x), float64(y)))
}

func Modf(f float32) (float32, float32) {
	i, fr := math.Modf(float64(f))
	return float32(i), float32(fr)
}

func NaN() float32 {
	return float32(math.NaN())
}

func Nextafter(x, y float32) float32 {
	return math.Nextafter32(x, y)
}

func Pow(x, y float32) float32 {
	return float32(math.Pow(float64(x), float64(y)))
}

func Pow10(e int) float32 {
	return float32(math.Pow10(e))
}

func Remainder(x, y float32) float32 {
	return float32(math.Remainder(float64(x), float64(y)))
}

func Round(x float32) float32 {
	return float32(math.Round(float64(x)))
}

func Signbit(x float32) bool {
	return math.Signbit(float64(x))
}

func Sin(x float32) float32 {
	return float32(math.Sin(float64(x)))
}

func Sincos(x float32) (float32, float32) {
	s, c := math.Sincos(float64(x))
	return float32(s), float32(c)
}

func Sinh(x float32) float32 {
	return float32(math.Sinh(float64(x)))
}

func Sqrt(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

func Tan(x float32) float32 {
	return float32(math.Tan(float64(x)))
}

func Tanh(x float32) float32 {
	return float32(math.Tanh(float64(x)))
}

func Trunc(x float32) float32 {
	return float32(math.Trunc(float64(x)))
}

func Y0(x float32) float32 {
	return float32(math.Y0(float64(x)))
}

func Y1(x float32) float32 {
	return float32(math.Y1(float64(x)))
}

func Yn(n int, x float32) float32 {
	return float32(math.Yn(n, float64(x)))
}
