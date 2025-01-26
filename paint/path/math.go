// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

import (
	"fmt"
	"math"
	"strings"

	"cogentcore.org/core/math32"
	"github.com/tdewolff/minify/v2"
)

// Epsilon is the smallest number below which we assume the value to be zero.
// This is to avoid numerical floating point issues.
var Epsilon = float32(1e-10)

// Precision is the number of significant digits at which floating point
// value will be printed to output formats.
var Precision = 8

// Origin is the coordinate system's origin.
var Origin = math32.Vector2{0.0, 0.0}

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

// InInterval returns true if f is in closed interval
// [lower-Epsilon,upper+Epsilon] where lower and upper can be interchanged.
func InInterval(f, lower, upper float32) bool {
	if upper < lower {
		lower, upper = upper, lower
	}
	return lower-Epsilon <= f && f <= upper+Epsilon
}

// InIntervalExclusive returns true if f is in open interval
// [lower+Epsilon,upper-Epsilon] where lower and upper can be interchanged.
func InIntervalExclusive(f, lower, upper float32) bool {
	if upper < lower {
		lower, upper = upper, lower
	}
	return lower+Epsilon < f && f < upper-Epsilon
}

// TouchesPoint returns true if the rectangle touches a point (within +-Epsilon).
func TouchesPoint(r math32.Box2, p math32.Vector2) bool {
	return InInterval(p.X, r.Min.X, r.Max.X) && InInterval(p.Y, r.Min.Y, r.Max.Y)
}

// Touches returns true if both rectangles touch (or overlap).
func Touches(r, q math32.Box2) bool {
	if q.Max.X+Epsilon < r.Min.X || r.Max.X < q.Min.X-Epsilon {
		// left or right
		return false
	} else if q.Max.Y+Epsilon < r.Min.Y || r.Max.Y < q.Min.Y-Epsilon {
		// below or above
		return false
	}
	return true
}

// angleEqual returns true if both angles are equal.
func angleEqual(a, b float32) bool {
	return angleBetween(a, b, b) // angleBetween will add Epsilon to lower and upper
}

// angleNorm returns the angle theta in the range [0,2PI).
func angleNorm(theta float32) float32 {
	theta = math32.Mod(theta, 2.0*math32.Pi)
	if theta < 0.0 {
		theta += 2.0 * math32.Pi
	}
	return theta
}

// angleTime returns the time [0.0,1.0] of theta between
// [lower,upper]. When outside of [lower,upper], the result will also be outside of [0.0,1.0].
func angleTime(theta, lower, upper float32) float32 {
	sweep := true
	if upper < lower {
		// sweep is false, ie direction is along negative angle (clockwise)
		lower, upper = upper, lower
		sweep = false
	}
	theta = angleNorm(theta - lower + Epsilon)
	upper = angleNorm(upper - lower)

	t := (theta - Epsilon) / upper
	if !sweep {
		t = 1.0 - t
	}
	if Equal(t, 0.0) {
		return 0.0
	} else if Equal(t, 1.0) {
		return 1.0
	}
	return t
}

// angleBetween is true when theta is in range [lower,upper]
// including the end points. Angles can be outside the [0,2PI) range.
func angleBetween(theta, lower, upper float32) bool {
	if upper < lower {
		// sweep is false, ie direction is along negative angle (clockwise)
		lower, upper = upper, lower
	}
	theta = angleNorm(theta - lower + Epsilon)
	upper = angleNorm(upper - lower + 2.0*Epsilon)
	return theta <= upper
}

// angleBetweenExclusive is true when theta is in range (lower,upper)
// excluding the end points. Angles can be outside the [0,2PI) range.
func angleBetweenExclusive(theta, lower, upper float32) bool {
	if upper < lower {
		// sweep is false, ie direction is along negative angle (clockwise)
		lower, upper = upper, lower
	}
	theta = angleNorm(theta - lower)
	upper = angleNorm(upper - lower)
	if 0.0 < theta && theta < upper {
		return true
	}
	return false
}

// Slope returns the slope between OP, i.e. y/x.
func Slope(p math32.Vector2) float32 {
	return p.Y / p.X
}

// Angle returns the angle in radians [0,2PI) between the x-axis and OP.
func Angle(p math32.Vector2) float32 {
	return angleNorm(math32.Atan2(p.Y, p.X))
}

// todo: use this for our AngleTo

// AngleBetween returns the angle between OP and OQ.
func AngleBetween(p, q math32.Vector2) float32 {
	return math32.Atan2(p.Cross(q), p.Dot(q))
}

// snap "gridsnaps" the floating point to a grid of the given spacing
func snap(val, spacing float32) float32 {
	return math32.Round(val/spacing) * spacing
}

// Gridsnap snaps point to a grid with the given spacing.
func Gridsnap(p math32.Vector2, spacing float32) math32.Vector2 {
	return math32.Vector2{snap(p.X, spacing), snap(p.Y, spacing)}
}

func cohenSutherlandOutcode(rect math32.Box2, p math32.Vector2, eps float32) int {
	code := 0b0000
	if p.X < rect.Min.X-eps {
		code |= 0b0001 // left
	} else if rect.Max.X+eps < p.X {
		code |= 0b0010 // right
	}
	if p.Y < rect.Min.Y-eps {
		code |= 0b0100 // bottom
	} else if rect.Max.Y+eps < p.Y {
		code |= 0b1000 // top
	}
	return code
}

// return whether line is inside the rectangle, either entirely or partially.
func cohenSutherlandLineClip(rect math32.Box2, a, b math32.Vector2, eps float32) (math32.Vector2, math32.Vector2, bool, bool) {
	outcode0 := cohenSutherlandOutcode(rect, a, eps)
	outcode1 := cohenSutherlandOutcode(rect, b, eps)
	if outcode0 == 0 && outcode1 == 0 {
		return a, b, true, false
	}
	for {
		if (outcode0 | outcode1) == 0 {
			// both inside
			return a, b, true, true
		} else if (outcode0 & outcode1) != 0 {
			// both in same region outside
			return a, b, false, false
		}

		// pick point outside
		outcodeOut := outcode0
		if outcode0 < outcode1 {
			outcodeOut = outcode1
		}

		// intersect with rectangle
		var c math32.Vector2
		if (outcodeOut & 0b1000) != 0 {
			// above
			c.X = a.X + (b.X-a.X)*(rect.Max.Y-a.Y)/(b.Y-a.Y)
			c.Y = rect.Max.Y
		} else if (outcodeOut & 0b0100) != 0 {
			// below
			c.X = a.X + (b.X-a.X)*(rect.Min.Y-a.Y)/(b.Y-a.Y)
			c.Y = rect.Min.Y
		} else if (outcodeOut & 0b0010) != 0 {
			// right
			c.X = rect.Max.X
			c.Y = a.Y + (b.Y-a.Y)*(rect.Max.X-a.X)/(b.X-a.X)
		} else if (outcodeOut & 0b0001) != 0 {
			// left
			c.X = rect.Min.X
			c.Y = a.Y + (b.Y-a.Y)*(rect.Min.X-a.X)/(b.X-a.X)
		}

		// prepare next pass
		if outcodeOut == outcode0 {
			outcode0 = cohenSutherlandOutcode(rect, c, eps)
			a = c
		} else {
			outcode1 = cohenSutherlandOutcode(rect, c, eps)
			b = c
		}
	}
}

// Numerically stable quadratic formula, lowest root is returned first, see https://math32.stackexchange.com/a/2007723
func solveQuadraticFormula(a, b, c float32) (float32, float32) {
	if Equal(a, 0.0) {
		if Equal(b, 0.0) {
			if Equal(c, 0.0) {
				// all terms disappear, all x satisfy the solution
				return 0.0, math32.NaN()
			}
			// linear term disappears, no solutions
			return math32.NaN(), math32.NaN()
		}
		// quadratic term disappears, solve linear equation
		return -c / b, math32.NaN()
	}

	if Equal(c, 0.0) {
		// no constant term, one solution at zero and one from solving linearly
		if Equal(b, 0.0) {
			return 0.0, math32.NaN()
		}
		return 0.0, -b / a
	}

	discriminant := b*b - 4.0*a*c
	if discriminant < 0.0 {
		return math32.NaN(), math32.NaN()
	} else if Equal(discriminant, 0.0) {
		return -b / (2.0 * a), math32.NaN()
	}

	// Avoid catastrophic cancellation, which occurs when we subtract two nearly equal numbers and causes a large error. This can be the case when 4*a*c is small so that sqrt(discriminant) -> b, and the sign of b and in front of the radical are the same. Instead, we calculate x where b and the radical have different signs, and then use this result in the analytical equivalent of the formula, called the Citardauq Formula.
	q := math32.Sqrt(discriminant)
	if b < 0.0 {
		// apply sign of b
		q = -q
	}
	x1 := -(b + q) / (2.0 * a)
	x2 := c / (a * x1)
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	return x1, x2
}

// see https://www.geometrictools.com/Documentation/LowDegreePolynomialRoots.pdf
// see https://github.com/thelonious/kld-polynomial/blob/development/lib/Polynomial.js
func solveCubicFormula(a, b, c, d float32) (float32, float32, float32) {
	var x1, x2, x3 float32
	x2, x3 = math32.NaN(), math32.NaN() // x1 is always set to a number below
	if Equal(a, 0.0) {
		x1, x2 = solveQuadraticFormula(b, c, d)
	} else {
		// obtain monic polynomial: x^3 + f.x^2 + g.x + h = 0
		b /= a
		c /= a
		d /= a

		// obtain depressed polynomial: x^3 + c1.x + c0
		bthird := b / 3.0
		c0 := d - bthird*(c-2.0*bthird*bthird)
		c1 := c - b*bthird
		if Equal(c0, 0.0) {
			if c1 < 0.0 {
				tmp := math32.Sqrt(-c1)
				x1 = -tmp - bthird
				x2 = tmp - bthird
				x3 = 0.0 - bthird
			} else {
				x1 = 0.0 - bthird
			}
		} else if Equal(c1, 0.0) {
			if 0.0 < c0 {
				x1 = -math32.Cbrt(c0) - bthird
			} else {
				x1 = math32.Cbrt(-c0) - bthird
			}
		} else {
			delta := -(4.0*c1*c1*c1 + 27.0*c0*c0)
			if Equal(delta, 0.0) {
				delta = 0.0
			}

			if delta < 0.0 {
				betaRe := -c0 / 2.0
				betaIm := math32.Sqrt(-delta / 108.0)
				tmp := betaRe - betaIm
				if 0.0 <= tmp {
					x1 = math32.Cbrt(tmp)
				} else {
					x1 = -math32.Cbrt(-tmp)
				}
				tmp = betaRe + betaIm
				if 0.0 <= tmp {
					x1 += math32.Cbrt(tmp)
				} else {
					x1 -= math32.Cbrt(-tmp)
				}
				x1 -= bthird
			} else if 0.0 < delta {
				betaRe := -c0 / 2.0
				betaIm := math32.Sqrt(delta / 108.0)
				theta := math32.Atan2(betaIm, betaRe) / 3.0
				sintheta, costheta := math32.Sincos(theta)
				distance := math32.Sqrt(-c1 / 3.0) // same as rhoPowThird
				tmp := distance * sintheta * math32.Sqrt(3.0)
				x1 = 2.0*distance*costheta - bthird
				x2 = -distance*costheta - tmp - bthird
				x3 = -distance*costheta + tmp - bthird
			} else {
				tmp := -3.0 * c0 / (2.0 * c1)
				x1 = tmp - bthird
				x2 = -2.0*tmp - bthird
			}
		}
	}

	// sort
	if x3 < x2 || math32.IsNaN(x2) {
		x2, x3 = x3, x2
	}
	if x2 < x1 || math32.IsNaN(x1) {
		x1, x2 = x2, x1
	}
	if x3 < x2 || math32.IsNaN(x2) {
		x2, x3 = x3, x2
	}
	return x1, x2, x3
}

type gaussLegendreFunc func(func(float32) float32, float32, float32) float32

// Gauss-Legendre quadrature integration from a to b with n=3, see https://pomax.github.io/bezierinfo/legendre-gauss.html for more values
func gaussLegendre3(f func(float32) float32, a, b float32) float32 {
	c := (b - a) / 2.0
	d := (a + b) / 2.0
	Qd1 := f(-0.774596669*c + d)
	Qd2 := f(d)
	Qd3 := f(0.774596669*c + d)
	return c * ((5.0/9.0)*(Qd1+Qd3) + (8.0/9.0)*Qd2)
}

// Gauss-Legendre quadrature integration from a to b with n=5
func gaussLegendre5(f func(float32) float32, a, b float32) float32 {
	c := (b - a) / 2.0
	d := (a + b) / 2.0
	Qd1 := f(-0.90618*c + d)
	Qd2 := f(-0.538469*c + d)
	Qd3 := f(d)
	Qd4 := f(0.538469*c + d)
	Qd5 := f(0.90618*c + d)
	return c * (0.236927*(Qd1+Qd5) + 0.478629*(Qd2+Qd4) + 0.568889*Qd3)
}

// Gauss-Legendre quadrature integration from a to b with n=7
func gaussLegendre7(f func(float32) float32, a, b float32) float32 {
	c := (b - a) / 2.0
	d := (a + b) / 2.0
	Qd1 := f(-0.949108*c + d)
	Qd2 := f(-0.741531*c + d)
	Qd3 := f(-0.405845*c + d)
	Qd4 := f(d)
	Qd5 := f(0.405845*c + d)
	Qd6 := f(0.741531*c + d)
	Qd7 := f(0.949108*c + d)
	return c * (0.129485*(Qd1+Qd7) + 0.279705*(Qd2+Qd6) + 0.381830*(Qd3+Qd5) + 0.417959*Qd4)
}

//func lookupMin(f func(float64) float64, xmin, xmax float64) float64 {
//	const MaxIterations = 1000
//	min := math32.Inf(1)
//	for i := 0; i <= MaxIterations; i++ {
//		t := float64(i) / float64(MaxIterations)
//		x := xmin + t*(xmax-xmin)
//		y := f(x)
//		if y < min {
//			min = y
//		}
//	}
//	return min
//}
//
//func gradientDescent(f func(float64) float64, xmin, xmax float64) float64 {
//	const MaxIterations = 100
//	const Delta = 0.0001
//	const Rate = 0.01
//
//	x := (xmin + xmax) / 2.0
//	for i := 0; i < MaxIterations; i++ {
//		dydx := (f(x+Delta) - f(x-Delta)) / 2.0 / Delta
//		x -= Rate * dydx
//	}
//	return x
//}

// find value x for which f(x) = y in the interval x in [xmin, xmax] using the bisection method
func bisectionMethod(f func(float32) float32, y, xmin, xmax float32) float32 {
	const MaxIterations = 100
	const Tolerance = 0.001 // 0.1%

	n := 0
	toleranceX := math32.Abs(xmax-xmin) * Tolerance
	toleranceY := math32.Abs(f(xmax)-f(xmin)) * Tolerance

	var x float32
	for {
		x = (xmin + xmax) / 2.0
		if n >= MaxIterations {
			return x
		}

		dy := f(x) - y
		if math32.Abs(dy) < toleranceY || math32.Abs(xmax-xmin)/2.0 < toleranceX {
			return x
		} else if dy > 0.0 {
			xmax = x
		} else {
			xmin = x
		}
		n++
	}
}

func invSpeedPolynomialChebyshevApprox(N int, gaussLegendre gaussLegendreFunc, fp func(float32) float32, tmin, tmax float32) (func(float32) float32, float32) {
	// TODO: find better way to determine N. For Arc 10 seems fine, for some Quads 10 is too low, for Cube depending on inflection points is maybe not the best indicator
	// TODO: track efficiency, how many times is fp called? Does a look-up table make more sense?
	fLength := func(t float32) float32 {
		return math32.Abs(gaussLegendre(fp, tmin, t))
	}
	totalLength := fLength(tmax)
	t := func(L float32) float32 {
		return bisectionMethod(fLength, L, tmin, tmax)
	}
	return polynomialChebyshevApprox(N, t, 0.0, totalLength, tmin, tmax), totalLength
}

func polynomialChebyshevApprox(N int, f func(float32) float32, xmin, xmax, ymin, ymax float32) func(float32) float32 {
	fs := make([]float32, N)
	for k := 0; k < N; k++ {
		u := math32.Cos(math32.Pi * (float32(k+1) - 0.5) / float32(N))
		fs[k] = f(xmin + (xmax-xmin)*(u+1.0)/2.0)
	}

	c := make([]float32, N)
	for j := 0; j < N; j++ {
		a := float32(0.0)
		for k := 0; k < N; k++ {
			a += fs[k] * math32.Cos(float32(j)*math32.Pi*(float32(k+1)-0.5)/float32(N))
		}
		c[j] = (2.0 / float32(N)) * a
	}

	if ymax < ymin {
		ymin, ymax = ymax, ymin
	}
	return func(x float32) float32 {
		x = math32.Min(xmax, math32.Max(xmin, x))
		u := (x-xmin)/(xmax-xmin)*2.0 - 1.0
		a := float32(0.0)
		for j := 0; j < N; j++ {
			a += c[j] * math32.Cos(float32(j)*math32.Acos(u))
		}
		y := -0.5*c[0] + a
		if !math32.IsNaN(ymin) && !math32.IsNaN(ymax) {
			y = math32.Min(ymax, math32.Max(ymin, y))
		}
		return y
	}
}

type numEps float32

func (f numEps) String() string {
	s := fmt.Sprintf("%.*g", int(math32.Ceil(-math32.Log10(Epsilon))), f)
	if dot := strings.IndexByte(s, '.'); dot != -1 {
		for dot < len(s) && s[len(s)-1] == '0' {
			s = s[:len(s)-1]
		}
		if dot < len(s) && s[len(s)-1] == '.' {
			s = s[:len(s)-1]
		}
	}
	return s
}

type num float32

func (f num) String() string {
	s := fmt.Sprintf("%.*g", Precision, f)
	if num(math.MaxInt32) < f || f < num(math.MinInt32) {
		if i := strings.IndexAny(s, ".eE"); i == -1 {
			s += ".0"
		}
	}
	return string(minify.Number([]byte(s), Precision))
}

type dec float32

func (f dec) String() string {
	s := fmt.Sprintf("%.*f", Precision, f)
	s = string(minify.Decimal([]byte(s), Precision))
	if dec(math.MaxInt32) < f || f < dec(math.MinInt32) {
		if i := strings.IndexByte(s, '.'); i == -1 {
			s += ".0"
		}
	}
	return s
}
