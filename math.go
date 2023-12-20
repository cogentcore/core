// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

import (
	"math"
	"strconv"

	"github.com/chewxy/math32"
)

// These are mostly just wrappers around chewxy/math32, which has
// some optimized implementations.

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

const (
	MaxFloat32             = math.MaxFloat32
	SmallestNonzeroFloat32 = math.SmallestNonzeroFloat32
)

const (
	DegToRadFactor = Pi / 180
	RadToDegFactor = 180 / Pi
)

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

func Acosh(x float32) float32 {
	return math32.Acosh(x)
}

func Asin(x float32) float32 {
	return math32.Asin(x)
}

func Asinh(x float32) float32 {
	return math32.Asinh(x)
}

func Atan(x float32) float32 {
	return math32.Atan(x)
}

func Atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}

func Atanh(x float32) float32 {
	return math32.Atanh(x)
}

func Cbrt(x float32) float32 {
	return math32.Cbrt(x)
}

func Ceil(x float32) float32 {
	return math32.Ceil(x)
}

func Copysign(x, y float32) float32 {
	return float32(math.Copysign(float64(x), float64(y)))
}

func Cos(x float32) float32 {
	return math32.Cos(x)
}

func Cosh(x float32) float32 {
	return math32.Cosh(x)
}

func Dim(x, y float32) float32 {
	return float32(math.Dim(float64(x), float64(y)))
}

func Erf(x float32) float32 {
	return math32.Erf(x)
}

func Erfc(x float32) float32 {
	return math32.Erfc(x)
}

func Erfcinv(x float32) float32 {
	return float32(math.Erfcinv(float64(x)))
}

func Erfinv(x float32) float32 {
	return float32(math.Erfinv(float64(x)))
}

func Exp(x float32) float32 {
	return math32.Exp(x)
}

func Exp2(x float32) float32 {
	return math32.Exp2(x)
}

func Expm1(x float32) float32 {
	return math32.Expm1(x)
}

func FMA(x, y, z float32) float32 {
	return float32(math.FMA(float64(x), float64(y), float64(z)))
}

func Floor(x float32) float32 {
	return math32.Floor(x)
}

func Frexp(f float32) (frac float32, exp int) {
	var fc float64
	fc, exp = math.Frexp(float64(f))
	frac = float32(fc)
	return
}

func Gamma(x float32) float32 {
	return math32.Gamma(x)
}

func Hypot(p, q float32) float32 {
	return float32(math.Hypot(float64(p), float64(q)))
}

func Ilogb(x float32) float32 {
	return float32(math32.Ilogb(x))
}

func Inf(sign int) float32 {
	return float32(math.Inf(sign))
}

func IsInf(x float32, sign int) bool {
	return math.IsInf(float64(x), sign)
}

func IsNaN(x float32) bool {
	return math.IsNaN(float64(x))
}

func J0(x float32) float32 {
	return math32.J0(x)
}

func J1(x float32) float32 {
	return math32.J1(x)
}

func Jn(n int, x float32) float32 {
	return float32(math.Jn(n, float64(x)))
}

func Ldexp(frac float32, exp int) float32 {
	return float32(math.Ldexp(float64(frac), exp))
}

// Lerp is linear interpolation between start and stop in proportion to amount
func Lerp(start, stop, amount float32) float32 {
	return (1-amount)*start + amount*stop
}

func Lgamma(x float32) (lgamma float32, sign int) {
	var l float64
	l, sign = math.Lgamma(float64(x))
	lgamma = float32(l)
	return
}

func Log(x float32) float32 {
	return math32.Log(x)
}

func Log10(x float32) float32 {
	return math32.Log10(x)
}

func Log1p(x float32) float32 {
	return math32.Log1p(x)
}

func Log2(x float32) float32 {
	return math32.Log2(x)
}

func Logb(x float32) float32 {
	return math32.Logb(x)
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

func Modf(f float32) (it float32, frac float32) {
	var ii, ff float64
	ii, ff = math.Modf(float64(f))
	it = float32(ii)
	frac = float32(ff)
	return
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

func Pow10(n int) float32 {
	return float32(math.Pow10(n))
}

func Remainder(x, y float32) float32 {
	return float32(math.Remainder(float64(x), float64(y)))
}

func Round(x float32) float32 {
	return math32.Round(x)
}

func RoundToEven(x float32) float32 {
	return float32(math.RoundToEven(float64(x)))
}

func Signbit(x float32) bool {
	return math.Signbit(float64(x))
}

func Sin(x float32) float32 {
	return math32.Sin(x)
}

func Sincos(x float32) (sin, cos float32) {
	s, c := math.Sincos(float64(x))
	sin = float32(s)
	cos = float32(c)
	return
}

func Sinh(x float32) float32 {
	return math32.Sinh(x)
}

func Sqrt(x float32) float32 {
	return math32.Sqrt(x)
}

func Tan(x float32) float32 {
	return math32.Tan(x)
}

func Tanh(x float32) float32 {
	return math32.Tanh(x)
}

func Trunc(x float32) float32 {
	return math32.Trunc(x)
}

func Y0(x float32) float32 {
	return math32.Y0(x)
}

func Y1(x float32) float32 {
	return math32.Y1(x)
}

func Yn(n int, x float32) float32 {
	return float32(math.Yn(n, float64(x)))
}

//////////////////////////////////////////////////////////////
// Special additions to math. functions

// Clamp clamps x to the provided closed interval [a, b]
func Clamp(x, a, b float32) float32 {
	if x < a {
		return a
	}
	if x > b {
		return b
	}
	return x
}

// ClampInt clamps x to the provided closed interval [a, b]
func ClampInt(x, a, b int) int {
	if x < a {
		return a
	}
	if x > b {
		return b
	}
	return x
}

// SetMax sets a to Max(a,b)
func SetMax(a *float32, b float32) {
	*a = Max(*a, b)
}

// SetMin sets a to Min(a,b)
func SetMin(a *float32, b float32) {
	*a = Min(*a, b)
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
// int(Round(val / mod)) * mod
func IntMultiple(val, mod float32) float32 {
	return float32(int(math.Round(float64(val)/float64(mod)))) * mod
}

// IntMultipleGE returns the interger multiple of mod >= given value:
// int(Ceil(val / mod)) * mod
func IntMultipleGE(val, mod float32) float32 {
	return float32(int(math.Ceil(float64(val)/float64(mod)))) * mod
}

// IntMultiple64 returns the interger multiple of mod closest to given value:
// int(Round(val / mod)) * mod
func IntMultiple64(val, mod float64) float64 {
	return float64(int(math.Round(float64(val)/float64(mod)))) * mod
}

// Truncate64 truncates a floating point number to given level of precision
// -- slow.. uses string conversion
func Truncate64(val float64, prec int) float64 {
	frep := strconv.FormatFloat(val, 'g', prec, 64)
	val, _ = strconv.ParseFloat(frep, 64)
	return val
}

// Truncate truncates a floating point number to given level of precision
// -- slow.. uses string conversion
func Truncate(val float32, prec int) float32 {
	frep := strconv.FormatFloat(float64(val), 'g', prec, 32)
	tval, _ := strconv.ParseFloat(frep, 32)
	return float32(tval)
}
