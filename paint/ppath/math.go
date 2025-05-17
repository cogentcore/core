// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import (
	"cogentcore.org/core/math32"
)

var (
	// Tolerance is the maximum deviation from the original path in millimeters
	// when e.g. flatting. Used for flattening in the renderers, font decorations,
	// and path intersections.
	Tolerance = float32(0.01)

	// PixelTolerance is the maximum deviation of the rasterized path from
	// the original for flattening purposed in pixels.
	PixelTolerance = float32(0.1)

	//	In C, FLT_EPSILON = 1.19209e-07

	// Epsilon is the smallest number below which we assume the value to be zero.
	// This is to avoid numerical floating point issues.
	Epsilon = float32(1e-7)

	// Precision is the number of significant digits at which floating point
	// value will be printed to output formats.
	Precision = 7

	// Origin is the coordinate system's origin.
	Origin = math32.Vector2{0.0, 0.0}
)

// Equal returns true if a and b are equal within an absolute
// tolerance of Epsilon.
func Equal(a, b float32) bool {
	// avoid math32.Abs
	if a < b {
		return b-a <= Epsilon
	}
	return a-b <= Epsilon
}

func EqualPoint(a, b math32.Vector2) bool {
	return Equal(a.X, b.X) && Equal(a.Y, b.Y)
}

// AngleEqual returns true if both angles are equal.
func AngleEqual(a, b float32) bool {
	return IsAngleBetween(a, b, b) // IsAngleBetween will add Epsilon to lower and upper
}

// AngleNorm returns the angle theta in the range [0,2PI).
func AngleNorm(theta float32) float32 {
	theta = math32.Mod(theta, 2.0*math32.Pi)
	if theta < 0.0 {
		theta += 2.0 * math32.Pi
	}
	return theta
}

// IsAngleBetween is true when theta is in range [lower,upper]
// including the end points. Angles can be outside the [0,2PI) range.
func IsAngleBetween(theta, lower, upper float32) bool {
	if upper < lower {
		// sweep is false, ie direction is along negative angle (clockwise)
		lower, upper = upper, lower
	}
	theta = AngleNorm(theta - lower + Epsilon)
	upper = AngleNorm(upper - lower + 2.0*Epsilon)
	return theta <= upper
}

// Slope returns the slope between OP, i.e. y/x.
func Slope(p math32.Vector2) float32 {
	return p.Y / p.X
}

// Angle returns the angle in radians [0,2PI) between the x-axis and OP.
func Angle(p math32.Vector2) float32 {
	return AngleNorm(math32.Atan2(p.Y, p.X))
}

// todo: use this for our AngleTo

// AngleBetween returns the angle between OP and OQ.
func AngleBetween(p, q math32.Vector2) float32 {
	return math32.Atan2(p.Cross(q), p.Dot(q))
}
