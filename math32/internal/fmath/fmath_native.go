// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript

package fmath

import "github.com/chewxy/math32"

func Abs(x float32) float32 { return math32.Abs(x) }

func Acos(x float32) float32 { return math32.Acos(x) }

func Acosh(x float32) float32 { return math32.Acosh(x) }

func Asin(x float32) float32 { return math32.Asin(x) }

func Asinh(x float32) float32 { return math32.Asinh(x) }

func Atan(x float32) float32 { return math32.Atan(x) }

func Atan2(y, x float32) float32 { return math32.Atan2(y, x) }

func Atanh(x float32) float32 { return math32.Atanh(x) }

func Cbrt(x float32) float32 { return math32.Cbrt(x) }

func Ceil(x float32) float32 { return math32.Ceil(x) }

func Copysign(x, y float32) float32 { return math32.Copysign(x, y) }

func Cos(x float32) float32 { return math32.Cos(x) }

func Cosh(x float32) float32 { return math32.Cosh(x) }

func Dim(x, y float32) float32 { return math32.Dim(x, y) }

func Erf(x float32) float32 { return math32.Erf(x) }

func Erfc(x float32) float32 { return math32.Erfc(x) }

func Exp(x float32) float32 { return math32.Exp(x) }

func Exp2(x float32) float32 { return math32.Exp2(x) }

func Expm1(x float32) float32 { return math32.Expm1(x) }

func Floor(x float32) float32 { return math32.Floor(x) }

func Frexp(f float32) (float32, int) { return math32.Frexp(f) }

func Gamma(x float32) float32 { return math32.Gamma(x) }

func Hypot(p, q float32) float32 { return math32.Hypot(p, q) }

func Ilogb(x float32) int { return math32.Ilogb(x) }

func Inf(sign int) float32 { return math32.Inf(sign) }

func IsInf(f float32, sign int) bool { return math32.IsInf(f, sign) }

func IsNaN(f float32) bool { return math32.IsNaN(f) }

func J0(x float32) float32 { return math32.J0(x) }

func J1(x float32) float32 { return math32.J1(x) }

func Jn(n int, x float32) float32 { return math32.Jn(n, x) }

func Ldexp(frac float32, exp int) float32 { return math32.Ldexp(frac, exp) }

func Lgamma(x float32) (float32, int) { return math32.Lgamma(x) }

func Log(x float32) float32 { return math32.Log(x) }

func Log10(x float32) float32 { return math32.Log10(x) }

func Log1p(x float32) float32 { return math32.Log1p(x) }

func Log2(x float32) float32 { return math32.Log2(x) }

func Logb(x float32) float32 { return math32.Logb(x) }

func Max(x, y float32) float32 { return math32.Max(x, y) }

func Min(x, y float32) float32 { return math32.Min(x, y) }

func Mod(x, y float32) float32 { return math32.Mod(x, y) }

func Modf(f float32) (float32, float32) { return math32.Modf(f) }

func NaN() float32 { return math32.NaN() }

func Nextafter(x, y float32) float32 { return math32.Nextafter(x, y) }

func Pow(x, y float32) float32 { return math32.Pow(x, y) }

func Pow10(e int) float32 { return math32.Pow10(e) }

func Remainder(x, y float32) float32 { return math32.Remainder(x, y) }

func Round(x float32) float32 { return math32.Round(x) }

func Signbit(x float32) bool { return math32.Signbit(x) }

func Sin(x float32) float32 { return math32.Sin(x) }

func Sincos(x float32) (float32, float32) { return math32.Sincos(x) }

func Sinh(x float32) float32 { return math32.Sinh(x) }

func Sqrt(x float32) float32 { return math32.Sqrt(x) }

func Tan(x float32) float32 { return math32.Tan(x) }

func Tanh(x float32) float32 { return math32.Tanh(x) }

func Trunc(x float32) float32 { return math32.Trunc(x) }

func Y0(x float32) float32 { return math32.Y0(x) }

func Y1(x float32) float32 { return math32.Y1(x) }

func Yn(n int, x float32) float32 { return math32.Yn(n, x) }
