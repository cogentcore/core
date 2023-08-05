// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	package floats provides a standard Floater interface and all the std math functions

defined on Floater types.  Furthermore, fully generic slice sort and
conversion methods in the kit type kit package attempt to use this interface,
before falling back on reflection.  If you have a struct that can be converted
into an float64, then this is the only way to allow it to be sorted using those
generic functions, as the reflect.Kind fallback will fail.
*/
package floats

import "math"

// Floater converts a type from a float64, used in kit.ToFloat function and in
// sorting comparisons -- tried first in sorting
type Floater interface {
	Float() float64
}

// FloatSetter is Floater that also supports setting the value from a float64.
// Satisfying this interface requires a pointer to the underlying type.
type FloatSetter interface {
	Floater
	FromFloat(val float64)
}

///////////////////////////////////////////////////////
//   math wrappers

func Abs(x Floater) float64 {
	return math.Abs(x.Float())
}
func Acos(x Floater) float64 {
	return math.Acos(x.Float())
}
func Acosh(x Floater) float64 {
	return math.Acosh(x.Float())
}
func Asin(x Floater) float64 {
	return math.Asin(x.Float())
}
func Asinh(x Floater) float64 {
	return math.Asinh(x.Float())
}
func Atan(x Floater) float64 {
	return math.Atan(x.Float())
}
func Atan2(y, x Floater) float64 {
	return math.Atan2(x.Float(), y.Float())
}
func Atanh(x Floater) float64 {
	return math.Atanh(x.Float())
}
func Cbrt(x Floater) float64 {
	return math.Cbrt(x.Float())
}
func Ceil(x Floater) float64 {
	return math.Ceil(x.Float())
}
func Copysign(x, y Floater) float64 {
	return math.Copysign(x.Float(), y.Float())
}
func Cos(x Floater) float64 {
	return math.Cos(x.Float())
}
func Cosh(x Floater) float64 {
	return math.Cosh(x.Float())
}
func Dim(x, y Floater) float64 {
	return math.Dim(x.Float(), y.Float())
}
func Erf(x Floater) float64 {
	return math.Erf(x.Float())
}
func Erfc(x Floater) float64 {
	return math.Erfc(x.Float())
}
func Erfcinv(x Floater) float64 {
	return math.Erfcinv(x.Float())
}
func Erfinv(x Floater) float64 {
	return math.Erfinv(x.Float())
}
func Exp(x Floater) float64 {
	return math.Exp(x.Float())
}
func Exp2(x Floater) float64 {
	return math.Exp2(x.Float())
}
func Expm1(x Floater) float64 {
	return math.Expm1(x.Float())
}
func Floor(x Floater) float64 {
	return math.Floor(x.Float())
}
func Frexp(f Floater) (frac float64, exp int) {
	return math.Frexp(f.Float())
}
func Gamma(x Floater) float64 {
	return math.Gamma(x.Float())
}
func Hypot(p, q Floater) float64 {
	return math.Hypot(p.Float(), q.Float())
}
func Ilogb(x Floater) int {
	return math.Ilogb(x.Float())
}
func IsInf(f Floater, sign int) bool {
	return math.IsInf(f.Float(), sign)
}
func IsNaN(f Floater) (is bool) {
	return math.IsNaN(f.Float())
}
func J0(x Floater) float64 {
	return math.J0(x.Float())
}
func J1(x Floater) float64 {
	return math.J1(x.Float())
}
func Jn(n int, x Floater) float64 {
	return math.Jn(n, x.Float())
}
func Ldexp(frac Floater, exp int) float64 {
	return math.Ldexp(frac.Float(), exp)
}
func Lgamma(x Floater) (lgamma float64, sign int) {
	return math.Lgamma(x.Float())
}
func Log(x Floater) float64 {
	return math.Log(x.Float())
}
func Log10(x Floater) float64 {
	return math.Log10(x.Float())
}
func Log1p(x Floater) float64 {
	return math.Log1p(x.Float())
}
func Log2(x Floater) float64 {
	return math.Log2(x.Float())
}
func Logb(x Floater) float64 {
	return math.Logb(x.Float())
}
func Max(x, y Floater) float64 {
	return math.Max(x.Float(), y.Float())
}
func Min(x, y Floater) float64 {
	return math.Min(x.Float(), y.Float())
}
func Mod(x, y Floater) float64 {
	return math.Mod(x.Float(), y.Float())
}
func Modf(f Floater) (int float64, frac float64) {
	return math.Modf(f.Float())
}
func Nextafter(x, y Floater) (r float64) {
	return math.Nextafter(x.Float(), y.Float())
}
func Pow(x, y Floater) float64 {
	return math.Pow(x.Float(), y.Float())
}
func Remainder(x, y Floater) float64 {
	return math.Remainder(x.Float(), y.Float())
}
func Round(x Floater) float64 {
	return math.Round(x.Float())
}
func RoundToEven(x Floater) float64 {
	return math.RoundToEven(x.Float())
}
func Signbit(x Floater) bool {
	return math.Signbit(x.Float())
}
func Sin(x Floater) float64 {
	return math.Sin(x.Float())
}
func Sincos(x Floater) (sin, cos float64) {
	return math.Sincos(x.Float())
}
func Sinh(x Floater) float64 {
	return math.Sinh(x.Float())
}
func Sqrt(x Floater) float64 {
	return math.Sqrt(x.Float())
}
func Tan(x Floater) float64 {
	return math.Tan(x.Float())
}
func Tanh(x Floater) float64 {
	return math.Tanh(x.Float())
}
func Trunc(x Floater) float64 {
	return math.Trunc(x.Float())
}
func Y0(x Floater) float64 {
	return math.Y0(x.Float())
}
func Y1(x Floater) float64 {
	return math.Y1(x.Float())
}
func Yn(n int, x Floater) float64 {
	return math.Yn(n, x.Float())
}
