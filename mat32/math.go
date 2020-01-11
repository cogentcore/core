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
)

const Pi = math.Pi
const DegToRadFactor = math.Pi / 180
const RadToDegFactor = 180.0 / math.Pi

var Infinity = float32(math.Inf(1))

// DegToRad converts a number from degrees to radians
func DegToRad(degrees float32) float32 {
	return degrees * DegToRadFactor
}

// RadToDeg converts a number from radians to degrees
func RadToDeg(radians float32) float32 {
	return radians * RadToDegFactor
}

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

func Abs(v float32) float32 {
	return float32(math.Abs(float64(v)))
}

func Sign(v float32) float32 {
	if v < 0 {
		return -1
	}
	return 1
}

func Acos(v float32) float32 {
	return float32(math.Acos(float64(v)))
}

func Asin(v float32) float32 {
	return float32(math.Asin(float64(v)))
}

func Atan(v float32) float32 {
	return float32(math.Atan(float64(v)))
}

func Atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}

func Ceil(v float32) float32 {
	return float32(math.Ceil(float64(v)))
}

func Cos(v float32) float32 {
	return float32(math.Cos(float64(v)))
}

func Floor(v float32) float32 {
	return float32(math.Floor(float64(v)))
}

func Inf(sign int) float32 {
	return float32(math.Inf(sign))
}

func Round(v float32) float32 {
	return float32(math.Round(float64(v)))
}

func IsNaN(v float32) bool {
	return math.IsNaN(float64(v))
}

func Sin(v float32) float32 {
	return float32(math.Sin(float64(v)))
}

func Sqrt(v float32) float32 {
	return float32(math.Sqrt(float64(v)))
}

// note: it's surprisingly complicated..

func Max(a, b float32) float32 {
	return float32(math.Max(float64(a), float64(b)))
}

func Min(a, b float32) float32 {
	return float32(math.Min(float64(a), float64(b)))
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

func Mod(a, b float32) float32 {
	return float32(math.Mod(float64(a), float64(b)))
}

// IntMultiple returns the interger multiple of mod closest to given value:
// int(Round(val / mod)) * mod
func IntMultiple(val, mod float32) float32 {
	return float32(int(math.Round(float64(val/mod)))) * mod
}

// IntMultiple64 returns the interger multiple of mod closest to given value:
// int(Round(val / mod)) * mod
func IntMultiple64(val, mod float64) float64 {
	return float64(int(math.Round(float64(val/mod)))) * mod
}

func NaN() float32 {
	return float32(math.NaN())
}

func Pow(a, b float32) float32 {
	return float32(math.Pow(float64(a), float64(b)))
}

func Tan(v float32) float32 {
	return float32(math.Tan(float64(v)))
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
