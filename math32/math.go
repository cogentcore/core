// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit Cogent Core functionality.

// Package math32 is a float32 based vector, matrix, and math package
// for 2D & 3D graphics.
package math32

//go:generate core generate

import (
	"cmp"
	"math"
	"strconv"

	"github.com/chewxy/math32"
)

// These are mostly just wrappers around chewxy/math32, which has
// some optimized implementations.

// Mathematical constants.
const (
	E   = math.E
	Pi  = math.Pi
	Phi = math.Phi

	Sqrt2   = math.Sqrt2
	SqrtE   = math.SqrtE
	SqrtPi  = math.SqrtPi
	SqrtPhi = math.SqrtPhi

	Ln2    = math.Ln2
	Log2E  = math.Log2E
	Ln10   = math.Ln10
	Log10E = math.Log10E
)

// Floating-point limit values.
// Max is the largest finite value representable by the type.
// SmallestNonzero is the smallest positive, non-zero value representable by the type.
const (
	MaxFloat32             = math.MaxFloat32
	SmallestNonzeroFloat32 = math.SmallestNonzeroFloat32
)

const (
	// DegToRadFactor is the number of radians per degree.
	DegToRadFactor = Pi / 180

	// RadToDegFactor is the number of degrees per radian.
	RadToDegFactor = 180 / Pi
)

// Infinity is positive infinity.
var Infinity = float32(math.Inf(1))

// DegToRad converts a number from degrees to radians
func DegToRad(degrees float32) float32 {
	return degrees * DegToRadFactor
}

// RadToDeg converts a number from radians to degrees
func RadToDeg(radians float32) float32 {
	return radians * RadToDegFactor
}

// Abs returns the absolute value of x.
//
// Special cases are:
//
//	Abs(±Inf) = +Inf
//	Abs(NaN) = NaN
func Abs(x float32) float32 {
	return math32.Abs(x)
}

// Sign returns -1 if x < 0 and 1 otherwise.
func Sign(x float32) float32 {
	if x < 0 {
		return -1
	}
	return 1
}

// Acos returns the arccosine, in radians, of x.
//
// Special case is:
//
//	Acos(x) = NaN if x < -1 or x > 1
func Acos(x float32) float32 {
	return math32.Acos(x)
}

// Acosh returns the inverse hyperbolic cosine of x.
//
// Special cases are:
//
//	Acosh(+Inf) = +Inf
//	Acosh(x) = NaN if x < 1
//	Acosh(NaN) = NaN
func Acosh(x float32) float32 {
	return math32.Acosh(x)
}

// Asin returns the arcsine, in radians, of x.
//
// Special cases are:
//
//	Asin(±0) = ±0
//	Asin(x) = NaN if x < -1 or x > 1
func Asin(x float32) float32 {
	return math32.Asin(x)
}

// Asinh returns the inverse hyperbolic sine of x.
//
// Special cases are:
//
//	Asinh(±0) = ±0
//	Asinh(±Inf) = ±Inf
//	Asinh(NaN) = NaN
func Asinh(x float32) float32 {
	return math32.Asinh(x)
}

// Atan returns the arctangent, in radians, of x.
//
// Special cases are:
//
//	Atan(±0) = ±0
//	Atan(±Inf) = ±Pi/2
func Atan(x float32) float32 {
	return math32.Atan(x)
}

// Atan2 returns the arc tangent of y/x, using the signs of the two to determine the quadrant of the return value.
// Special cases are (in order):
//
//	Atan2(y, NaN) = NaN
//	Atan2(NaN, x) = NaN
//	Atan2(+0, x>=0) = +0
//	Atan2(-0, x>=0) = -0
//	Atan2(+0, x<=-0) = +Pi
//	Atan2(-0, x<=-0) = -Pi
//	Atan2(y>0, 0) = +Pi/2
//	Atan2(y<0, 0) = -Pi/2
//	Atan2(+Inf, +Inf) = +Pi/4
//	Atan2(-Inf, +Inf) = -Pi/4
//	Atan2(+Inf, -Inf) = 3Pi/4
//	Atan2(-Inf, -Inf) = -3Pi/4
//	Atan2(y, +Inf) = 0
//	Atan2(y>0, -Inf) = +Pi
//	Atan2(y<0, -Inf) = -Pi
//	Atan2(+Inf, x) = +Pi/2
//	Atan2(-Inf, x) = -Pi/2
func Atan2(y, x float32) float32 {
	return math32.Atan2(y, x)
}

// Atanh returns the inverse hyperbolic tangent of x.
//
// Special cases are:
//
//	Atanh(1) = +Inf
//	Atanh(±0) = ±0
//	Atanh(-1) = -Inf
//	Atanh(x) = NaN if x < -1 or x > 1
//	Atanh(NaN) = NaN
func Atanh(x float32) float32 {
	return math32.Atanh(x)
}

// Cbrt returns the cube root of x.
//
// Special cases are:
//
//	Cbrt(±0) = ±0
//	Cbrt(±Inf) = ±Inf
//	Cbrt(NaN) = NaN
func Cbrt(x float32) float32 {
	return math32.Cbrt(x)
}

// Ceil returns the least integer value greater than or equal to x.
//
// Special cases are:
//
//	Ceil(±0) = ±0
//	Ceil(±Inf) = ±Inf
//	Ceil(NaN) = NaN
func Ceil(x float32) float32 {
	return math32.Ceil(x)
}

// Copysign returns a value with the magnitude of f
// and the sign of sign.
func Copysign(f, sign float32) float32 {
	return math32.Copysign(f, sign)
}

// Cos returns the cosine of the radian argument x.
//
// Special cases are:
//
//	Cos(±Inf) = NaN
//	Cos(NaN) = NaN
func Cos(x float32) float32 {
	return math32.Cos(x)
}

// Cosh returns the hyperbolic cosine of x.
//
// Special cases are:
//
//	Cosh(±0) = 1
//	Cosh(±Inf) = +Inf
//	Cosh(NaN) = NaN
func Cosh(x float32) float32 {
	return math32.Cosh(x)
}

// Dim returns the maximum of x-y or 0.
//
// Special cases are:
//
//	Dim(+Inf, +Inf) = NaN
//	Dim(-Inf, -Inf) = NaN
//	Dim(x, NaN) = Dim(NaN, x) = NaN
func Dim(x, y float32) float32 {
	return math32.Dim(x, y)
}

// Erf returns the error function of x.
//
// Special cases are:
//
//	Erf(+Inf) = 1
//	Erf(-Inf) = -1
//	Erf(NaN) = NaN
func Erf(x float32) float32 {
	return math32.Erf(x)
}

// Erfc returns the complementary error function of x.
//
// Special cases are:
//
//	Erfc(+Inf) = 0
//	Erfc(-Inf) = 2
//	Erfc(NaN) = NaN
func Erfc(x float32) float32 {
	return math32.Erfc(x)
}

// Erfcinv returns the inverse of Erfc(x).
//
// Special cases are:
//
//	Erfcinv(0) = +Inf
//	Erfcinv(2) = -Inf
//	Erfcinv(x) = NaN if x < 0 or x > 2
//	Erfcinv(NaN) = NaN
func Erfcinv(x float32) float32 {
	return float32(math.Erfcinv(float64(x)))
}

// Erfinv returns the inverse error function of x.
//
// Special cases are:
//
//	Erfinv(1) = +Inf
//	Erfinv(-1) = -Inf
//	Erfinv(x) = NaN if x < -1 or x > 1
//	Erfinv(NaN) = NaN
func Erfinv(x float32) float32 {
	return float32(math.Erfinv(float64(x)))
}

// Exp returns e**x, the base-e exponential of x.
//
// Special cases are:
//
//	Exp(+Inf) = +Inf
//	Exp(NaN) = NaN
//
// Very large values overflow to 0 or +Inf.
// Very small values underflow to 1.
func Exp(x float32) float32 {
	return math32.Exp(x)
}

// Exp2 returns 2**x, the base-2 exponential of x.
//
// Special cases are the same as Exp.
func Exp2(x float32) float32 {
	return math32.Exp2(x)
}

// Expm1 returns e**x - 1, the base-e exponential of x minus 1.
// It is more accurate than Exp(x) - 1 when x is near zero.
//
// Special cases are:
//
//	Expm1(+Inf) = +Inf
//	Expm1(-Inf) = -1
//	Expm1(NaN) = NaN
//
// Very large values overflow to -1 or +Inf.
func Expm1(x float32) float32 {
	return math32.Expm1(x)
}

// FMA returns x * y + z, computed with only one rounding.
// (That is, FMA returns the fused multiply add of x, y, and z.)
func FMA(x, y, z float32) float32 {
	return float32(math.FMA(float64(x), float64(y), float64(z)))
}

// Floor returns the greatest integer value less than or equal to x.
//
// Special cases are:
//
//	Floor(±0) = ±0
//	Floor(±Inf) = ±Inf
//	Floor(NaN) = NaN
func Floor(x float32) float32 {
	return math32.Floor(x)
}

// Frexp breaks f into a normalized fraction
// and an integral power of two.
// It returns frac and exp satisfying f == frac × 2**exp,
// with the absolute value of frac in the interval [½, 1).
//
// Special cases are:
//
//	Frexp(±0) = ±0, 0
//	Frexp(±Inf) = ±Inf, 0
//	Frexp(NaN) = NaN, 0
func Frexp(f float32) (frac float32, exp int) {
	return math32.Frexp(f)
}

// Gamma returns the Gamma function of x.
//
// Special cases are:
//
//	Gamma(+Inf) = +Inf
//	Gamma(+0) = +Inf
//	Gamma(-0) = -Inf
//	Gamma(x) = NaN for integer x < 0
//	Gamma(-Inf) = NaN
//	Gamma(NaN) = NaN
func Gamma(x float32) float32 {
	return math32.Gamma(x)
}

// Hypot returns Sqrt(p*p + q*q), taking care to avoid
// unnecessary overflow and underflow.
//
// Special cases are:
//
//	Hypot(±Inf, q) = +Inf
//	Hypot(p, ±Inf) = +Inf
//	Hypot(NaN, q) = NaN
//	Hypot(p, NaN) = NaN
func Hypot(p, q float32) float32 {
	return math32.Hypot(p, q)
}

// Ilogb returns the binary exponent of x as an integer.
//
// Special cases are:
//
//	Ilogb(±Inf) = MaxInt32
//	Ilogb(0) = MinInt32
//	Ilogb(NaN) = MaxInt32
func Ilogb(x float32) float32 {
	return float32(math32.Ilogb(x))
}

// Inf returns positive infinity if sign >= 0, negative infinity if sign < 0.
func Inf(sign int) float32 {
	return math32.Inf(sign)
}

// IsInf reports whether f is an infinity, according to sign.
// If sign > 0, IsInf reports whether f is positive infinity.
// If sign < 0, IsInf reports whether f is negative infinity.
// If sign == 0, IsInf reports whether f is either infinity.
func IsInf(x float32, sign int) bool {
	return math32.IsInf(x, sign)
}

// IsNaN reports whether f is an IEEE 754 “not-a-number” value.
func IsNaN(x float32) bool {
	return math32.IsNaN(x)
}

// J0 returns the order-zero Bessel function of the first kind.
//
// Special cases are:
//
//	J0(±Inf) = 0
//	J0(0) = 1
//	J0(NaN) = NaN
func J0(x float32) float32 {
	return math32.J0(x)
}

// J1 returns the order-one Bessel function of the first kind.
//
// Special cases are:
//
//	J1(±Inf) = 0
//	J1(NaN) = NaN
func J1(x float32) float32 {
	return math32.J1(x)
}

// Jn returns the order-n Bessel function of the first kind.
//
// Special cases are:
//
//	Jn(n, ±Inf) = 0
//	Jn(n, NaN) = NaN
func Jn(n int, x float32) float32 {
	return math32.Jn(n, x)
}

// Ldexp is the inverse of Frexp.
// It returns frac × 2**exp.
//
// Special cases are:
//
//	Ldexp(±0, exp) = ±0
//	Ldexp(±Inf, exp) = ±Inf
//	Ldexp(NaN, exp) = NaN
func Ldexp(frac float32, exp int) float32 {
	return math32.Ldexp(frac, exp)
}

// Lerp returns the linear interpolation between start and stop in proportion to amount
func Lerp(start, stop, amount float32) float32 {
	return (1-amount)*start + amount*stop
}

// Lgamma returns the natural logarithm and sign (-1 or +1) of Gamma(x).
//
// Special cases are:
//
//	Lgamma(+Inf) = +Inf
//	Lgamma(0) = +Inf
//	Lgamma(-integer) = +Inf
//	Lgamma(-Inf) = -Inf
//	Lgamma(NaN) = NaN
func Lgamma(x float32) (lgamma float32, sign int) {
	return math32.Lgamma(x)
}

// Log returns the natural logarithm of x.
//
// Special cases are:
//
//	Log(+Inf) = +Inf
//	Log(0) = -Inf
//	Log(x < 0) = NaN
//	Log(NaN) = NaN
func Log(x float32) float32 {
	return math32.Log(x)
}

// Log10 returns the decimal logarithm of x.
// The special cases are the same as for Log.
func Log10(x float32) float32 {
	return math32.Log10(x)
}

// Log1p returns the natural logarithm of 1 plus its argument x.
// It is more accurate than Log(1 + x) when x is near zero.
//
// Special cases are:
//
//	Log1p(+Inf) = +Inf
//	Log1p(±0) = ±0
//	Log1p(-1) = -Inf
//	Log1p(x < -1) = NaN
//	Log1p(NaN) = NaN
func Log1p(x float32) float32 {
	return math32.Log1p(x)
}

// Log2 returns the binary logarithm of x.
// The special cases are the same as for Log.
func Log2(x float32) float32 {
	return math32.Log2(x)
}

// Logb returns the binary exponent of x.
//
// Special cases are:
//
//	Logb(±Inf) = +Inf
//	Logb(0) = -Inf
//	Logb(NaN) = NaN
func Logb(x float32) float32 {
	return math32.Logb(x)
}

// TODO(kai): should we use builtin max and min?

// Max returns the larger of x or y.
//
// Special cases are:
//
//	Max(x, +Inf) = Max(+Inf, x) = +Inf
//	Max(x, NaN) = Max(NaN, x) = NaN
//	Max(+0, ±0) = Max(±0, +0) = +0
//	Max(-0, -0) = -0
//
// Note that this differs from the built-in function max when called
// with NaN and +Inf.
func Max(x, y float32) float32 {
	return math32.Max(x, y)
}

// Min returns the smaller of x or y.
//
// Special cases are:
//
//	Min(x, -Inf) = Min(-Inf, x) = -Inf
//	Min(x, NaN) = Min(NaN, x) = NaN
//	Min(-0, ±0) = Min(±0, -0) = -0
//
// Note that this differs from the built-in function min when called
// with NaN and -Inf.
func Min(x, y float32) float32 {
	return math32.Min(x, y)
}

// Mod returns the floating-point remainder of x/y.
// The magnitude of the result is less than y and its
// sign agrees with that of x.
//
// Special cases are:
//
//	Mod(±Inf, y) = NaN
//	Mod(NaN, y) = NaN
//	Mod(x, 0) = NaN
//	Mod(x, ±Inf) = x
//	Mod(x, NaN) = NaN
func Mod(x, y float32) float32 {
	return math32.Mod(x, y)
}

// Modf returns integer and fractional floating-point numbers
// that sum to f. Both values have the same sign as f.
//
// Special cases are:
//
//	Modf(±Inf) = ±Inf, NaN
//	Modf(NaN) = NaN, NaN
func Modf(f float32) (it float32, frac float32) {
	return math32.Modf(f)
}

// NaN returns an IEEE 754 “not-a-number” value.
func NaN() float32 {
	return math32.NaN()
}

// Nextafter returns the next representable float32 value after x towards y.
//
// Special cases are:
//
//	Nextafter32(x, x)   = x
//	Nextafter32(NaN, y) = NaN
//	Nextafter32(x, NaN) = NaN
func Nextafter(x, y float32) float32 {
	return math32.Nextafter(x, y)
}

// Pow returns x**y, the base-x exponential of y.
//
// Special cases are (in order):
//
//	Pow(x, ±0) = 1 for any x
//	Pow(1, y) = 1 for any y
//	Pow(x, 1) = x for any x
//	Pow(NaN, y) = NaN
//	Pow(x, NaN) = NaN
//	Pow(±0, y) = ±Inf for y an odd integer < 0
//	Pow(±0, -Inf) = +Inf
//	Pow(±0, +Inf) = +0
//	Pow(±0, y) = +Inf for finite y < 0 and not an odd integer
//	Pow(±0, y) = ±0 for y an odd integer > 0
//	Pow(±0, y) = +0 for finite y > 0 and not an odd integer
//	Pow(-1, ±Inf) = 1
//	Pow(x, +Inf) = +Inf for |x| > 1
//	Pow(x, -Inf) = +0 for |x| > 1
//	Pow(x, +Inf) = +0 for |x| < 1
//	Pow(x, -Inf) = +Inf for |x| < 1
//	Pow(+Inf, y) = +Inf for y > 0
//	Pow(+Inf, y) = +0 for y < 0
//	Pow(-Inf, y) = Pow(-0, -y)
//	Pow(x, y) = NaN for finite x < 0 and finite non-integer y
func Pow(x, y float32) float32 {
	return math32.Pow(x, y)
}

// Pow10 returns 10**n, the base-10 exponential of n.
//
// Special cases are:
//
//	Pow10(n) =    0 for n < -323
//	Pow10(n) = +Inf for n > 308
func Pow10(n int) float32 {
	return math32.Pow10(n)
}

// Remainder returns the IEEE 754 floating-point remainder of x/y.
//
// Special cases are:
//
//	Remainder(±Inf, y) = NaN
//	Remainder(NaN, y) = NaN
//	Remainder(x, 0) = NaN
//	Remainder(x, ±Inf) = x
//	Remainder(x, NaN) = NaN
func Remainder(x, y float32) float32 {
	return math32.Remainder(x, y)
}

// Round returns the nearest integer, rounding half away from zero.
//
// Special cases are:
//
//	Round(±0) = ±0
//	Round(±Inf) = ±Inf
//	Round(NaN) = NaN
func Round(x float32) float32 {
	return math32.Round(x)
}

// RoundToEven returns the nearest integer, rounding ties to even.
//
// Special cases are:
//
//	RoundToEven(±0) = ±0
//	RoundToEven(±Inf) = ±Inf
//	RoundToEven(NaN) = NaN
func RoundToEven(x float32) float32 {
	return float32(math.RoundToEven(float64(x)))
}

// Signbit returns true if x is negative or negative zero.
func Signbit(x float32) bool {
	return math32.Signbit(x)
}

// Sin returns the sine of the radian argument x.
//
// Special cases are:
//
//	Sin(±0) = ±0
//	Sin(±Inf) = NaN
//	Sin(NaN) = NaN
func Sin(x float32) float32 {
	return math32.Sin(x)
}

// Sincos returns Sin(x), Cos(x).
//
// Special cases are:
//
//	Sincos(±0) = ±0, 1
//	Sincos(±Inf) = NaN, NaN
//	Sincos(NaN) = NaN, NaN
func Sincos(x float32) (sin, cos float32) {
	return math32.Sincos(x)
}

// Sinh returns the hyperbolic sine of x.
//
// Special cases are:
//
//	Sinh(±0) = ±0
//	Sinh(±Inf) = ±Inf
//	Sinh(NaN) = NaN
func Sinh(x float32) float32 {
	return math32.Sinh(x)
}

// Sqrt returns the square root of x.
//
// Special cases are:
//
//	Sqrt(+Inf) = +Inf
//	Sqrt(±0) = ±0
//	Sqrt(x < 0) = NaN
//	Sqrt(NaN) = NaN
func Sqrt(x float32) float32 {
	return math32.Sqrt(x)
}

// Tan returns the tangent of the radian argument x.
//
// Special cases are:
//
//	Tan(±0) = ±0
//	Tan(±Inf) = NaN
//	Tan(NaN) = NaN
func Tan(x float32) float32 {
	return math32.Tan(x)
}

// Tanh returns the hyperbolic tangent of x.
//
// Special cases are:
//
//	Tanh(±0) = ±0
//	Tanh(±Inf) = ±1
//	Tanh(NaN) = NaN
func Tanh(x float32) float32 {
	return math32.Tanh(x)
}

// Trunc returns the integer value of x.
//
// Special cases are:
//
//	Trunc(±0) = ±0
//	Trunc(±Inf) = ±Inf
//	Trunc(NaN) = NaN
func Trunc(x float32) float32 {
	return math32.Trunc(x)
}

// Y0 returns the order-zero Bessel function of the second kind.
//
// Special cases are:
//
//	Y0(+Inf) = 0
//	Y0(0) = -Inf
//	Y0(x < 0) = NaN
//	Y0(NaN) = NaN
func Y0(x float32) float32 {
	return math32.Y0(x)
}

// Y1 returns the order-one Bessel function of the second kind.
//
// Special cases are:
//
//	Y1(+Inf) = 0
//	Y1(0) = -Inf
//	Y1(x < 0) = NaN
//	Y1(NaN) = NaN
func Y1(x float32) float32 {
	return math32.Y1(x)
}

// Yn returns the order-n Bessel function of the second kind.
//
// Special cases are:
//
//	Yn(n, +Inf) = 0
//	Yn(n ≥ 0, 0) = -Inf
//	Yn(n < 0, 0) = +Inf if n is odd, -Inf if n is even
//	Yn(n, x < 0) = NaN
//	Yn(n, NaN) = NaN
func Yn(n int, x float32) float32 {
	return math32.Yn(n, x)
}

//////////////////////////////////////////////////////////////
// Special additions to math. functions

// Clamp clamps x to the provided closed interval [a, b]
func Clamp[T cmp.Ordered](x, a, b T) T {
	if x < a {
		return a
	}
	if x > b {
		return b
	}
	return x
}

// MinPos returns the minimum of the two values, excluding any that are <= 0
func MinPos(a, b float32) float32 {
	if a > 0 && b > 0 {
		return Min(a, b)
	} else if a > 0 {
		return a
	} else if b > 0 {
		return b
	}
	return a
}

// MaxPos returns the minimum of the two values, excluding any that are <= 0
func MaxPos(a, b float32) float32 {
	if a > 0 && b > 0 {
		return Max(a, b)
	} else if a > 0 {
		return a
	} else if b > 0 {
		return b
	}
	return a
}

// IntMultiple returns the interger multiple of mod closest to given value:
// Round(val / mod) * mod
func IntMultiple(val, mod float32) float32 {
	return Round(val/mod) * mod
}

// IntMultipleGE returns the interger multiple of mod >= given value:
// Ceil(val / mod) * mod
func IntMultipleGE(val, mod float32) float32 {
	return Ceil(val/mod) * mod
}

// TODO: maybe make these functions faster at some point

// Truncate rounds a float32 number to the given level of precision,
// which the number of significant digits to include in the result.
func Truncate(val float32, prec int) float32 {
	frep := strconv.FormatFloat(float64(val), 'g', prec, 32)
	tval, _ := strconv.ParseFloat(frep, 32)
	return float32(tval)
	// note: this unfortunately does not work.  also Pow(prec) is not likely to be that much faster ;)
	// pow := Pow(10, float32(prec))
	// return Round(val*pow) / pow
}

// Truncate64 rounds a float64 number to the given level of precision,
// which the number of significant digits to include in the result.
func Truncate64(val float64, prec int) float64 {
	frep := strconv.FormatFloat(val, 'g', prec, 64)
	val, _ = strconv.ParseFloat(frep, 64)
	return val
	// note: this unfortunately does not work.  also Pow(prec) is not likely to be that much faster ;)
	// pow := math.Pow(10, float64(prec))
	// return math.Round(val*pow) / pow
}
