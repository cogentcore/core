// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package intersect

import (
	"fmt"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
)

// see https://github.com/signavio/svg-intersections
// see https://github.com/w8r/bezier-intersect
// see https://cs.nyu.edu/exact/doc/subdiv1.pdf

// intersect for path segments a and b, starting at a0 and b0.
// Note that all intersection functions return up to two intersections.
func IntersectionSegment(zs Intersections, a0 math32.Vector2, a ppath.Path, b0 math32.Vector2, b ppath.Path) Intersections {
	n := len(zs)
	swapCurves := false
	if a[0] == ppath.LineTo || a[0] == ppath.Close {
		if b[0] == ppath.LineTo || b[0] == ppath.Close {
			zs = intersectionLineLine(zs, a0, math32.Vec2(a[1], a[2]), b0, math32.Vec2(b[1], b[2]))
		} else if b[0] == ppath.QuadTo {
			zs = intersectionLineQuad(zs, a0, math32.Vec2(a[1], a[2]), b0, math32.Vec2(b[1], b[2]), math32.Vec2(b[3], b[4]))
		} else if b[0] == ppath.CubeTo {
			zs = intersectionLineCube(zs, a0, math32.Vec2(a[1], a[2]), b0, math32.Vec2(b[1], b[2]), math32.Vec2(b[3], b[4]), math32.Vec2(b[5], b[6]))
		} else if b[0] == ppath.ArcTo {
			rx := b[1]
			ry := b[2]
			phi := b[3]
			large, sweep := ppath.ToArcFlags(b[4])
			cx, cy, theta0, theta1 := ppath.EllipseToCenter(b0.X, b0.Y, rx, ry, phi, large, sweep, b[5], b[6])
			zs = intersectionLineEllipse(zs, a0, math32.Vec2(a[1], a[2]), math32.Vector2{cx, cy}, math32.Vector2{rx, ry}, phi, theta0, theta1)
		}
	} else if a[0] == ppath.QuadTo {
		if b[0] == ppath.LineTo || b[0] == ppath.Close {
			zs = intersectionLineQuad(zs, b0, math32.Vec2(b[1], b[2]), a0, math32.Vec2(a[1], a[2]), math32.Vec2(a[3], a[4]))
			swapCurves = true
		} else if b[0] == ppath.QuadTo {
			panic("unsupported intersection for quad-quad")
		} else if b[0] == ppath.CubeTo {
			panic("unsupported intersection for quad-cube")
		} else if b[0] == ppath.ArcTo {
			panic("unsupported intersection for quad-arc")
		}
	} else if a[0] == ppath.CubeTo {
		if b[0] == ppath.LineTo || b[0] == ppath.Close {
			zs = intersectionLineCube(zs, b0, math32.Vec2(b[1], b[2]), a0, math32.Vec2(a[1], a[2]), math32.Vec2(a[3], a[4]), math32.Vec2(a[5], a[6]))
			swapCurves = true
		} else if b[0] == ppath.QuadTo {
			panic("unsupported intersection for cube-quad")
		} else if b[0] == ppath.CubeTo {
			panic("unsupported intersection for cube-cube")
		} else if b[0] == ppath.ArcTo {
			panic("unsupported intersection for cube-arc")
		}
	} else if a[0] == ppath.ArcTo {
		rx := a[1]
		ry := a[2]
		phi := a[3]
		large, sweep := ppath.ToArcFlags(a[4])
		cx, cy, theta0, theta1 := ppath.EllipseToCenter(a0.X, a0.Y, rx, ry, phi, large, sweep, a[5], a[6])
		if b[0] == ppath.LineTo || b[0] == ppath.Close {
			zs = intersectionLineEllipse(zs, b0, math32.Vec2(b[1], b[2]), math32.Vector2{cx, cy}, math32.Vector2{rx, ry}, phi, theta0, theta1)
			swapCurves = true
		} else if b[0] == ppath.QuadTo {
			panic("unsupported intersection for arc-quad")
		} else if b[0] == ppath.CubeTo {
			panic("unsupported intersection for arc-cube")
		} else if b[0] == ppath.ArcTo {
			rx2 := b[1]
			ry2 := b[2]
			phi2 := b[3]
			large2, sweep2 := ppath.ToArcFlags(b[4])
			cx2, cy2, theta20, theta21 := ppath.EllipseToCenter(b0.X, b0.Y, rx2, ry2, phi2, large2, sweep2, b[5], b[6])
			zs = intersectionEllipseEllipse(zs, math32.Vector2{cx, cy}, math32.Vector2{rx, ry}, phi, theta0, theta1, math32.Vector2{cx2, cy2}, math32.Vector2{rx2, ry2}, phi2, theta20, theta21)
		}
	}

	// swap A and B in the intersection found to match segments A and B of this function
	if swapCurves {
		for i := n; i < len(zs); i++ {
			zs[i].T[0], zs[i].T[1] = zs[i].T[1], zs[i].T[0]
			zs[i].Dir[0], zs[i].Dir[1] = zs[i].Dir[1], zs[i].Dir[0]
		}
	}
	return zs
}

// Intersection is an intersection between two path segments, e.g. Line x Line. Note that an
// intersection is tangent also when it is at one of the endpoints, in which case it may be tangent
// for this segment but may or may not cross the path depending on the adjacent segment.
// Notabene: for quad/cube/ellipse aligned angles at the endpoint for non-overlapping curves are deviated slightly to correctly calculate the value for Into, and will thus not be aligned
type Intersection struct {
	math32.Vector2            // coordinate of intersection
	T              [2]float32 // position along segment [0,1]
	Dir            [2]float32 // direction at intersection [0,2*pi)
	Tangent        bool       // intersection is tangent (touches) instead of secant (crosses)
	Same           bool       // intersection is of two overlapping segments (tangent is also true)
}

// Into returns true if first path goes into the left-hand side of the second path,
// i.e. the second path goes to the right-hand side of the first path.
func (z Intersection) Into() bool {
	return angleBetweenExclusive(z.Dir[1]-z.Dir[0], math32.Pi, 2.0*math32.Pi)
}

func (z Intersection) Equals(o Intersection) bool {
	return ppath.EqualPoint(z.Vector2, o.Vector2) && ppath.Equal(z.T[0], o.T[0]) && ppath.Equal(z.T[1], o.T[1]) && ppath.AngleEqual(z.Dir[0], o.Dir[0]) && ppath.AngleEqual(z.Dir[1], o.Dir[1]) && z.Tangent == o.Tangent && z.Same == o.Same
}

func (z Intersection) String() string {
	var extra string
	if z.Tangent {
		extra = " Tangent"
	}
	if z.Same {
		extra = " Same"
	}
	return fmt.Sprintf("({%v,%v} t={%v,%v} dir={%v°,%v°}%v)", numEps(z.Vector2.X), numEps(z.Vector2.Y), numEps(z.T[0]), numEps(z.T[1]), numEps(math32.RadToDeg(ppath.AngleNorm(z.Dir[0]))), numEps(math32.RadToDeg(ppath.AngleNorm(z.Dir[1]))), extra)
}

type Intersections []Intersection

// Has returns true if there are secant/tangent intersections.
func (zs Intersections) Has() bool {
	return 0 < len(zs)
}

// HasSecant returns true when there are secant intersections, i.e. the curves intersect and cross (they cut).
func (zs Intersections) HasSecant() bool {
	for _, z := range zs {
		if !z.Tangent {
			return true
		}
	}
	return false
}

// HasTangent returns true when there are tangent intersections, i.e. the curves intersect but don't cross (they touch).
func (zs Intersections) HasTangent() bool {
	for _, z := range zs {
		if z.Tangent {
			return true
		}
	}
	return false
}

func (zs Intersections) add(pos math32.Vector2, ta, tb, dira, dirb float32, tangent, same bool) Intersections {
	// normalise T values between [0,1]
	if ta < 0.0 { // || ppath.Equal(ta, 0.0) {
		ta = 0.0
	} else if 1.0 <= ta { // || ppath.Equal(ta, 1.0) {
		ta = 1.0
	}
	if tb < 0.0 { // || ppath.Equal(tb, 0.0) {
		tb = 0.0
	} else if 1.0 < tb { // || ppath.Equal(tb, 1.0) {
		tb = 1.0
	}
	return append(zs, Intersection{pos, [2]float32{ta, tb}, [2]float32{dira, dirb}, tangent, same})
}

func correctIntersection(z, aMin, aMax, bMin, bMax math32.Vector2) math32.Vector2 {
	if z.X < aMin.X {
		//fmt.Println("CORRECT 1:", a0, a1, "--", b0, b1)
		z.X = aMin.X
	} else if aMax.X < z.X {
		//fmt.Println("CORRECT 2:", a0, a1, "--", b0, b1)
		z.X = aMax.X
	}
	if z.X < bMin.X {
		//fmt.Println("CORRECT 3:", a0, a1, "--", b0, b1)
		z.X = bMin.X
	} else if bMax.X < z.X {
		//fmt.Println("CORRECT 4:", a0, a1, "--", b0, b1)
		z.X = bMax.X
	}
	if z.Y < aMin.Y {
		//fmt.Println("CORRECT 5:", a0, a1, "--", b0, b1)
		z.Y = aMin.Y
	} else if aMax.Y < z.Y {
		//fmt.Println("CORRECT 6:", a0, a1, "--", b0, b1)
		z.Y = aMax.Y
	}
	if z.Y < bMin.Y {
		//fmt.Println("CORRECT 7:", a0, a1, "--", b0, b1)
		z.Y = bMin.Y
	} else if bMax.Y < z.Y {
		//fmt.Println("CORRECT 8:", a0, a1, "--", b0, b1)
		z.Y = bMax.Y
	}
	return z
}

// F. Antonio, "Faster Line Segment Intersection", Graphics Gems III, 1992
func intersectionLineLineBentleyOttmann(zs []math32.Vector2, a0, a1, b0, b1 math32.Vector2) []math32.Vector2 {
	// fast line-line intersection code, with additional constraints for the BentleyOttmann code:
	// - a0 is to the left and/or bottom of a1, same for b0 and b1
	// - an intersection z must keep the above property between (a0,z), (z,a1), (b0,z), and (z,b1)
	// note that an exception is made for (z,a1) and (z,b1) to allow them to become vertical, this
	// is because there isn't always "space" between a0.X and a1.X, eg. when a1.X = nextafter(a0.X)
	if a1.X < b0.X || b1.X < a0.X {
		return zs
	}

	aMin, aMax, bMin, bMax := a0, a1, b0, b1
	if a1.Y < a0.Y {
		aMin.Y, aMax.Y = aMax.Y, aMin.Y
	}
	if b1.Y < b0.Y {
		bMin.Y, bMax.Y = bMax.Y, bMin.Y
	}
	if aMax.Y < bMin.Y || bMax.Y < aMin.Y {
		return zs
	} else if (aMax.X == bMin.X || bMax.X == aMin.X) && (aMax.Y == bMin.Y || bMax.Y == aMin.Y) {
		return zs
	}

	// only the position and T values are valid for each intersection
	A := a1.Sub(a0)
	B := b0.Sub(b1)
	C := a0.Sub(b0)
	denom := B.Cross(A)
	// divide by length^2 since the perpdot between very small segments may be below Epsilon
	if denom == 0.0 {
		// colinear
		if C.Cross(B) == 0.0 {
			// overlap, rotate to x-axis
			a, b, c, d := a0.X, a1.X, b0.X, b1.X
			if math32.Abs(A.X) < math32.Abs(A.Y) {
				// mostly vertical
				a, b, c, d = a0.Y, a1.Y, b0.Y, b1.Y
			}
			if c < b && a < d {
				if a < c {
					zs = append(zs, b0)
				} else if c < a {
					zs = append(zs, a0)
				}
				if d < b {
					zs = append(zs, b1)
				} else if b < d {
					zs = append(zs, a1)
				}
			}
		}
		return zs
	}

	// find intersections within +-Epsilon to avoid missing near intersections
	ta := C.Cross(B) / denom
	if ta < -ppath.Epsilon || 1.0+ppath.Epsilon < ta {
		return zs
	}

	tb := A.Cross(C) / denom
	if tb < -ppath.Epsilon || 1.0+ppath.Epsilon < tb {
		return zs
	}

	// ta is snapped to 0.0 or 1.0 if very close
	if ta <= ppath.Epsilon {
		ta = 0.0
	} else if 1.0-ppath.Epsilon <= ta {
		ta = 1.0
	}

	z := a0.Lerp(a1, ta)
	z = correctIntersection(z, aMin, aMax, bMin, bMax)
	if z != a0 && z != a1 || z != b0 && z != b1 {
		// not at endpoints for both
		if a0 != b0 && z != a0 && z != b0 && b0.Sub(z).Cross(z.Sub(a0)) == 0.0 {
			a, c, m := a0.X, b0.X, z.X
			if math32.Abs(z.Sub(a0).X) < math32.Abs(z.Sub(a0).Y) {
				// mostly vertical
				a, c, m = a0.Y, b0.Y, z.Y
			}

			if a != c && (a < m) == (c < m) {
				if a < m && a < c || m < a && c < a {
					zs = append(zs, b0)
				} else {
					zs = append(zs, a0)
				}
			}
			zs = append(zs, z)
		} else if a1 != b1 && z != a1 && z != b1 && z.Sub(b1).Cross(a1.Sub(z)) == 0.0 {
			b, d, m := a1.X, b1.X, z.X
			if math32.Abs(z.Sub(a1).X) < math32.Abs(z.Sub(a1).Y) {
				// mostly vertical
				b, d, m = a1.Y, b1.Y, z.Y
			}

			if b != d && (b < m) == (d < m) {
				if b < m && b < d || m < b && d < b {
					zs = append(zs, b1)
				} else {
					zs = append(zs, a1)
				}
			}
		} else {
			zs = append(zs, z)
		}
	}
	return zs
}

func intersectionLineLine(zs Intersections, a0, a1, b0, b1 math32.Vector2) Intersections {
	if ppath.EqualPoint(a0, a1) || ppath.EqualPoint(b0, b1) {
		return zs // zero-length Close
	}

	da := a1.Sub(a0)
	db := b1.Sub(b0)
	anglea := ppath.Angle(da)
	angleb := ppath.Angle(db)
	div := da.Cross(db)

	// divide by length^2 since otherwise the perpdot between very small segments may be
	// below Epsilon
	if length := da.Length() * db.Length(); ppath.Equal(div/length, 0.0) {
		// parallel
		if ppath.Equal(b0.Sub(a0).Cross(db), 0.0) {
			// overlap, rotate to x-axis
			a := a0.Rot(-anglea, math32.Vector2{}).X
			b := a1.Rot(-anglea, math32.Vector2{}).X
			c := b0.Rot(-anglea, math32.Vector2{}).X
			d := b1.Rot(-anglea, math32.Vector2{}).X
			if inInterval(a, c, d) && inInterval(b, c, d) {
				// a-b in c-d or a-b == c-d
				zs = zs.add(a0, 0.0, (a-c)/(d-c), anglea, angleb, true, true)
				zs = zs.add(a1, 1.0, (b-c)/(d-c), anglea, angleb, true, true)
			} else if inInterval(c, a, b) && inInterval(d, a, b) {
				// c-d in a-b
				zs = zs.add(b0, (c-a)/(b-a), 0.0, anglea, angleb, true, true)
				zs = zs.add(b1, (d-a)/(b-a), 1.0, anglea, angleb, true, true)
			} else if inInterval(a, c, d) {
				// a in c-d
				same := a < d-ppath.Epsilon || a < c-ppath.Epsilon
				zs = zs.add(a0, 0.0, (a-c)/(d-c), anglea, angleb, true, same)
				if a < d-ppath.Epsilon {
					zs = zs.add(b1, (d-a)/(b-a), 1.0, anglea, angleb, true, true)
				} else if a < c-ppath.Epsilon {
					zs = zs.add(b0, (c-a)/(b-a), 0.0, anglea, angleb, true, true)
				}
			} else if inInterval(b, c, d) {
				// b in c-d
				same := c < b-ppath.Epsilon || d < b-ppath.Epsilon
				if c < b-ppath.Epsilon {
					zs = zs.add(b0, (c-a)/(b-a), 0.0, anglea, angleb, true, true)
				} else if d < b-ppath.Epsilon {
					zs = zs.add(b1, (d-a)/(b-a), 1.0, anglea, angleb, true, true)
				}
				zs = zs.add(a1, 1.0, (b-c)/(d-c), anglea, angleb, true, same)
			}
		}
		return zs
	} else if ppath.EqualPoint(a1, b0) {
		// handle common cases with endpoints to avoid numerical issues
		zs = zs.add(a1, 1.0, 0.0, anglea, angleb, true, false)
		return zs
	} else if ppath.EqualPoint(a0, b1) {
		// handle common cases with endpoints to avoid numerical issues
		zs = zs.add(a0, 0.0, 1.0, anglea, angleb, true, false)
		return zs
	}

	ta := db.Cross(a0.Sub(b0)) / div
	tb := da.Cross(a0.Sub(b0)) / div
	if inInterval(ta, 0.0, 1.0) && inInterval(tb, 0.0, 1.0) {
		tangent := ppath.Equal(ta, 0.0) || ppath.Equal(ta, 1.0) || ppath.Equal(tb, 0.0) || ppath.Equal(tb, 1.0)
		zs = zs.add(a0.Lerp(a1, ta), ta, tb, anglea, angleb, tangent, false)
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func intersectionLineQuad(zs Intersections, l0, l1, p0, p1, p2 math32.Vector2) Intersections {
	if ppath.EqualPoint(l0, l1) {
		return zs // zero-length Close
	}

	// write line as A.X = bias
	A := math32.Vector2{l1.Y - l0.Y, l0.X - l1.X}
	bias := l0.Dot(A)

	a := A.Dot(p0.Sub(p1.MulScalar(2.0)).Add(p2))
	b := A.Dot(p1.Sub(p0).MulScalar(2.0))
	c := A.Dot(p0) - bias

	roots := []float32{}
	r0, r1 := solveQuadraticFormula(a, b, c)
	if !math32.IsNaN(r0) {
		roots = append(roots, r0)
		if !math32.IsNaN(r1) {
			roots = append(roots, r1)
		}
	}

	dira := ppath.Angle(l1.Sub(l0))
	horizontal := math32.Abs(l1.Y-l0.Y) <= math32.Abs(l1.X-l0.X)
	for _, root := range roots {
		if inInterval(root, 0.0, 1.0) {
			var s float32
			pos := quadraticBezierPos(p0, p1, p2, root)
			if horizontal {
				s = (pos.X - l0.X) / (l1.X - l0.X)
			} else {
				s = (pos.Y - l0.Y) / (l1.Y - l0.Y)
			}
			if inInterval(s, 0.0, 1.0) {
				deriv := ppath.QuadraticBezierDeriv(p0, p1, p2, root)
				dirb := ppath.Angle(deriv)
				endpoint := ppath.Equal(root, 0.0) || ppath.Equal(root, 1.0) || ppath.Equal(s, 0.0) || ppath.Equal(s, 1.0)
				if endpoint {
					// deviate angle slightly at endpoint when aligned to properly set Into
					deriv2 := quadraticBezierDeriv2(p0, p1, p2)
					if (0.0 <= deriv.Cross(deriv2)) == (ppath.Equal(root, 0.0) || !ppath.Equal(root, 1.0) && ppath.Equal(s, 0.0)) {
						dirb += ppath.Epsilon * 2.0 // t=0 and CCW, or t=1 and CW
					} else {
						dirb -= ppath.Epsilon * 2.0 // t=0 and CW, or t=1 and CCW
					}
					dirb = ppath.AngleNorm(dirb)
				}
				zs = zs.add(pos, s, root, dira, dirb, endpoint || ppath.Equal(A.Dot(deriv), 0.0), false)
			}
		}
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func intersectionLineCube(zs Intersections, l0, l1, p0, p1, p2, p3 math32.Vector2) Intersections {
	if ppath.EqualPoint(l0, l1) {
		return zs // zero-length Close
	}

	// write line as A.X = bias
	A := math32.Vector2{l1.Y - l0.Y, l0.X - l1.X}
	bias := l0.Dot(A)

	a := A.Dot(p3.Sub(p0).Add(p1.MulScalar(3.0)).Sub(p2.MulScalar(3.0)))
	b := A.Dot(p0.MulScalar(3.0).Sub(p1.MulScalar(6.0)).Add(p2.MulScalar(3.0)))
	c := A.Dot(p1.MulScalar(3.0).Sub(p0.MulScalar(3.0)))
	d := A.Dot(p0) - bias

	roots := []float32{}
	r0, r1, r2 := solveCubicFormula(a, b, c, d)
	if !math32.IsNaN(r0) {
		roots = append(roots, r0)
		if !math32.IsNaN(r1) {
			roots = append(roots, r1)
			if !math32.IsNaN(r2) {
				roots = append(roots, r2)
			}
		}
	}

	dira := ppath.Angle(l1.Sub(l0))
	horizontal := math32.Abs(l1.Y-l0.Y) <= math32.Abs(l1.X-l0.X)
	for _, root := range roots {
		if inInterval(root, 0.0, 1.0) {
			var s float32
			pos := cubicBezierPos(p0, p1, p2, p3, root)
			if horizontal {
				s = (pos.X - l0.X) / (l1.X - l0.X)
			} else {
				s = (pos.Y - l0.Y) / (l1.Y - l0.Y)
			}
			if inInterval(s, 0.0, 1.0) {
				deriv := ppath.CubicBezierDeriv(p0, p1, p2, p3, root)
				dirb := ppath.Angle(deriv)
				tangent := ppath.Equal(A.Dot(deriv), 0.0)
				endpoint := ppath.Equal(root, 0.0) || ppath.Equal(root, 1.0) || ppath.Equal(s, 0.0) || ppath.Equal(s, 1.0)
				if endpoint {
					// deviate angle slightly at endpoint when aligned to properly set Into
					deriv2 := cubicBezierDeriv2(p0, p1, p2, p3, root)
					if (0.0 <= deriv.Cross(deriv2)) == (ppath.Equal(root, 0.0) || !ppath.Equal(root, 1.0) && ppath.Equal(s, 0.0)) {
						dirb += ppath.Epsilon * 2.0 // t=0 and CCW, or t=1 and CW
					} else {
						dirb -= ppath.Epsilon * 2.0 // t=0 and CW, or t=1 and CCW
					}
				} else if ppath.AngleEqual(dira, dirb) || ppath.AngleEqual(dira, dirb+math32.Pi) {
					// directions are parallel but the paths do cross (inflection point)
					// TODO: test better
					deriv2 := cubicBezierDeriv2(p0, p1, p2, p3, root)
					if ppath.Equal(deriv2.X, 0.0) && ppath.Equal(deriv2.Y, 0.0) {
						deriv3 := cubicBezierDeriv3(p0, p1, p2, p3, root)
						if 0.0 < deriv.Cross(deriv3) {
							dirb += ppath.Epsilon * 2.0
						} else {
							dirb -= ppath.Epsilon * 2.0
						}
						dirb = ppath.AngleNorm(dirb)
						tangent = false
					}
				}
				zs = zs.add(pos, s, root, dira, dirb, endpoint || tangent, false)
			}
		}
	}
	return zs
}

// handle line-arc intersections and their peculiarities regarding angles
func addLineArcIntersection(zs Intersections, pos math32.Vector2, dira, dirb, t, t0, t1, angle, theta0, theta1 float32, tangent bool) Intersections {
	if theta0 <= theta1 {
		angle = theta0 - ppath.Epsilon + ppath.AngleNorm(angle-theta0+ppath.Epsilon)
	} else {
		angle = theta1 - ppath.Epsilon + ppath.AngleNorm(angle-theta1+ppath.Epsilon)
	}
	endpoint := ppath.Equal(t, t0) || ppath.Equal(t, t1) || ppath.Equal(angle, theta0) || ppath.Equal(angle, theta1)
	if endpoint {
		// deviate angle slightly at endpoint when aligned to properly set Into
		if (theta0 <= theta1) == (ppath.Equal(angle, theta0) || !ppath.Equal(angle, theta1) && ppath.Equal(t, t0)) {
			dirb += ppath.Epsilon * 2.0 // t=0 and CCW, or t=1 and CW
		} else {
			dirb -= ppath.Epsilon * 2.0 // t=0 and CW, or t=1 and CCW
		}
		dirb = ppath.AngleNorm(dirb)
	}

	// snap segment parameters to 0.0 and 1.0 to avoid numerical issues
	var s float32
	if ppath.Equal(t, t0) {
		t = 0.0
	} else if ppath.Equal(t, t1) {
		t = 1.0
	} else {
		t = (t - t0) / (t1 - t0)
	}
	if ppath.Equal(angle, theta0) {
		s = 0.0
	} else if ppath.Equal(angle, theta1) {
		s = 1.0
	} else {
		s = (angle - theta0) / (theta1 - theta0)
	}
	return zs.add(pos, t, s, dira, dirb, endpoint || tangent, false)
}

// https://www.geometrictools.com/GTE/Mathematics/IntrLine2Circle2.h
func intersectionLineCircle(zs Intersections, l0, l1, center math32.Vector2, radius, theta0, theta1 float32) Intersections {
	if ppath.EqualPoint(l0, l1) {
		return zs // zero-length Close
	}

	// solve l0 + t*(l1-l0) = P + t*D = X  (line equation)
	// and |X - center| = |X - C| = R = radius  (circle equation)
	// by substitution and squaring: |P + t*D - C|^2 = R^2
	// giving: D^2 t^2 + 2D(P-C) t + (P-C)^2-R^2 = 0
	dir := l1.Sub(l0)
	diff := l0.Sub(center) // P-C
	length := dir.Length()
	D := dir.DivScalar(length)

	// we normalise D to be of length 1, so that the roots are in [0,length]
	a := float32(1.0)
	b := 2.0 * D.Dot(diff)
	c := diff.Dot(diff) - radius*radius

	// find solutions for t ∈ [0,1], the parameter along the line's path
	roots := []float32{}
	r0, r1 := solveQuadraticFormula(a, b, c)
	if !math32.IsNaN(r0) {
		roots = append(roots, r0)
		if !math32.IsNaN(r1) && !ppath.Equal(r0, r1) {
			roots = append(roots, r1)
		}
	}

	// handle common cases with endpoints to avoid numerical issues
	// snap closest root to path's start or end
	if 0 < len(roots) {
		if pos := l0.Sub(center); ppath.Equal(pos.Length(), radius) {
			if len(roots) == 1 || math32.Abs(roots[0]) < math32.Abs(roots[1]) {
				roots[0] = 0.0
			} else {
				roots[1] = 0.0
			}
		}
		if pos := l1.Sub(center); ppath.Equal(pos.Length(), radius) {
			if len(roots) == 1 || math32.Abs(roots[0]-length) < math32.Abs(roots[1]-length) {
				roots[0] = length
			} else {
				roots[1] = length
			}
		}
	}

	// add intersections
	dira := ppath.Angle(dir)
	tangent := len(roots) == 1
	for _, root := range roots {
		pos := diff.Add(dir.MulScalar(root / length))
		angle := math32.Atan2(pos.Y*radius, pos.X*radius)
		if inInterval(root, 0.0, length) && ppath.IsAngleBetween(angle, theta0, theta1) {
			pos = center.Add(pos)
			dirb := ppath.Angle(ppath.EllipseDeriv(radius, radius, 0.0, theta0 <= theta1, angle))
			zs = addLineArcIntersection(zs, pos, dira, dirb, root, 0.0, length, angle, theta0, theta1, tangent)
		}
	}
	return zs
}

func intersectionLineEllipse(zs Intersections, l0, l1, center, radius math32.Vector2, phi, theta0, theta1 float32) Intersections {
	if ppath.Equal(radius.X, radius.Y) {
		return intersectionLineCircle(zs, l0, l1, center, radius.X, theta0, theta1)
	} else if ppath.EqualPoint(l0, l1) {
		return zs // zero-length Close
	}

	// TODO: needs more testing
	// TODO: intersection inconsistency due to numerical stability in finding tangent collisions for subsequent paht segments (line -> ellipse), or due to the endpoint of a line not touching with another arc, but the subsequent segment does touch with its starting point
	dira := ppath.Angle(l1.Sub(l0))

	// we take the ellipse center as the origin and counter-rotate by phi
	l0 = l0.Sub(center).Rot(-phi, ppath.Origin)
	l1 = l1.Sub(center).Rot(-phi, ppath.Origin)

	// line: cx + dy + e = 0
	c := l0.Y - l1.Y
	d := l1.X - l0.X
	e := l0.Cross(l1)

	// follow different code paths when line is mostly horizontal or vertical
	horizontal := math32.Abs(c) <= math32.Abs(d)

	// ellipse: x^2/a + y^2/b = 1
	a := radius.X * radius.X
	b := radius.Y * radius.Y

	// rewrite as a polynomial by substituting x or y to obtain:
	// At^2 + Bt + C = 0, with t either x (horizontal) or y (!horizontal)
	var A, B, C float32
	A = a*c*c + b*d*d
	if horizontal {
		B = 2.0 * a * c * e
		C = a*e*e - a*b*d*d
	} else {
		B = 2.0 * b * d * e
		C = b*e*e - a*b*c*c
	}

	// find solutions
	roots := []float32{}
	r0, r1 := solveQuadraticFormula(A, B, C)
	if !math32.IsNaN(r0) {
		roots = append(roots, r0)
		if !math32.IsNaN(r1) && !ppath.Equal(r0, r1) {
			roots = append(roots, r1)
		}
	}

	for _, root := range roots {
		// get intersection position with center as origin
		var x, y, t0, t1 float32
		if horizontal {
			x = root
			y = -e/d - c*root/d
			t0 = l0.X
			t1 = l1.X
		} else {
			x = -e/c - d*root/c
			y = root
			t0 = l0.Y
			t1 = l1.Y
		}

		tangent := ppath.Equal(root, 0.0)
		angle := math32.Atan2(y*radius.X, x*radius.Y)
		if inInterval(root, t0, t1) && ppath.IsAngleBetween(angle, theta0, theta1) {
			pos := math32.Vector2{x, y}.Rot(phi, ppath.Origin).Add(center)
			dirb := ppath.Angle(ppath.EllipseDeriv(radius.X, radius.Y, phi, theta0 <= theta1, angle))
			zs = addLineArcIntersection(zs, pos, dira, dirb, root, t0, t1, angle, theta0, theta1, tangent)
		}
	}
	return zs
}

func intersectionEllipseEllipse(zs Intersections, c0, r0 math32.Vector2, phi0, thetaStart0, thetaEnd0 float32, c1, r1 math32.Vector2, phi1, thetaStart1, thetaEnd1 float32) Intersections {
	// TODO: needs more testing
	if !ppath.Equal(r0.X, r0.Y) || !ppath.Equal(r1.X, r1.Y) {
		panic("not handled") // ellipses
	}

	arcAngle := func(theta float32, sweep bool) float32 {
		theta += math32.Pi / 2.0
		if !sweep {
			theta -= math32.Pi
		}
		return ppath.AngleNorm(theta)
	}

	dtheta0 := thetaEnd0 - thetaStart0
	thetaStart0 = ppath.AngleNorm(thetaStart0 + phi0)
	thetaEnd0 = thetaStart0 + dtheta0

	dtheta1 := thetaEnd1 - thetaStart1
	thetaStart1 = ppath.AngleNorm(thetaStart1 + phi1)
	thetaEnd1 = thetaStart1 + dtheta1

	if ppath.EqualPoint(c0, c1) && ppath.EqualPoint(r0, r1) {
		// parallel
		tOffset1 := float32(0.0)
		dirOffset1 := float32(0.0)
		if (0.0 <= dtheta0) != (0.0 <= dtheta1) {
			thetaStart1, thetaEnd1 = thetaEnd1, thetaStart1 // keep order on first arc
			dirOffset1 = math32.Pi
			tOffset1 = 1.0
		}

		// will add either 1 (when touching) or 2 (when overlapping) intersections
		if t := angleTime(thetaStart0, thetaStart1, thetaEnd1); inInterval(t, 0.0, 1.0) {
			// ellipse0 starts within/on border of ellipse1
			dir := arcAngle(thetaStart0, 0.0 <= dtheta0)
			pos := ppath.EllipsePos(r0.X, r0.Y, 0.0, c0.X, c0.Y, thetaStart0)
			zs = zs.add(pos, 0.0, math32.Abs(t-tOffset1), dir, ppath.AngleNorm(dir+dirOffset1), true, true)
		}
		if t := angleTime(thetaStart1, thetaStart0, thetaEnd0); inIntervalExclusive(t, 0.0, 1.0) {
			// ellipse1 starts within ellipse0
			dir := arcAngle(thetaStart1, 0.0 <= dtheta0)
			pos := ppath.EllipsePos(r0.X, r0.Y, 0.0, c0.X, c0.Y, thetaStart1)
			zs = zs.add(pos, t, tOffset1, dir, ppath.AngleNorm(dir+dirOffset1), true, true)
		}
		if t := angleTime(thetaEnd1, thetaStart0, thetaEnd0); inIntervalExclusive(t, 0.0, 1.0) {
			// ellipse1 ends within ellipse0
			dir := arcAngle(thetaEnd1, 0.0 <= dtheta0)
			pos := ppath.EllipsePos(r0.X, r0.Y, 0.0, c0.X, c0.Y, thetaEnd1)
			zs = zs.add(pos, t, 1.0-tOffset1, dir, ppath.AngleNorm(dir+dirOffset1), true, true)
		}
		if t := angleTime(thetaEnd0, thetaStart1, thetaEnd1); inInterval(t, 0.0, 1.0) {
			// ellipse0 ends within/on border of ellipse1
			dir := arcAngle(thetaEnd0, 0.0 <= dtheta0)
			pos := ppath.EllipsePos(r0.X, r0.Y, 0.0, c0.X, c0.Y, thetaEnd0)
			zs = zs.add(pos, 1.0, math32.Abs(t-tOffset1), dir, ppath.AngleNorm(dir+dirOffset1), true, true)
		}
		return zs
	}

	// https://math32.stackexchange.com/questions/256100/how-can-i-find-the-points-at-which-two-circles-intersect
	// https://gist.github.com/jupdike/bfe5eb23d1c395d8a0a1a4ddd94882ac
	R := c0.Sub(c1).Length()
	if R < math32.Abs(r0.X-r1.X) || r0.X+r1.X < R {
		return zs
	}
	R2 := R * R

	k := r0.X*r0.X - r1.X*r1.X
	a := float32(0.5)
	b := 0.5 * k / R2
	c := 0.5 * math32.Sqrt(2.0*(r0.X*r0.X+r1.X*r1.X)/R2-k*k/(R2*R2)-1.0)

	mid := c1.Sub(c0).MulScalar(a + b)
	dev := math32.Vector2{c1.Y - c0.Y, c0.X - c1.X}.MulScalar(c)

	tangent := ppath.EqualPoint(dev, math32.Vector2{})
	anglea0 := ppath.Angle(mid.Add(dev))
	anglea1 := ppath.Angle(c0.Sub(c1).Add(mid).Add(dev))
	ta0 := angleTime(anglea0, thetaStart0, thetaEnd0)
	ta1 := angleTime(anglea1, thetaStart1, thetaEnd1)
	if inInterval(ta0, 0.0, 1.0) && inInterval(ta1, 0.0, 1.0) {
		dir0 := arcAngle(anglea0, 0.0 <= dtheta0)
		dir1 := arcAngle(anglea1, 0.0 <= dtheta1)
		endpoint := ppath.Equal(ta0, 0.0) || ppath.Equal(ta0, 1.0) || ppath.Equal(ta1, 0.0) || ppath.Equal(ta1, 1.0)
		zs = zs.add(c0.Add(mid).Add(dev), ta0, ta1, dir0, dir1, tangent || endpoint, false)
	}

	if !tangent {
		angleb0 := ppath.Angle(mid.Sub(dev))
		angleb1 := ppath.Angle(c0.Sub(c1).Add(mid).Sub(dev))
		tb0 := angleTime(angleb0, thetaStart0, thetaEnd0)
		tb1 := angleTime(angleb1, thetaStart1, thetaEnd1)
		if inInterval(tb0, 0.0, 1.0) && inInterval(tb1, 0.0, 1.0) {
			dir0 := arcAngle(angleb0, 0.0 <= dtheta0)
			dir1 := arcAngle(angleb1, 0.0 <= dtheta1)
			endpoint := ppath.Equal(tb0, 0.0) || ppath.Equal(tb0, 1.0) || ppath.Equal(tb1, 0.0) || ppath.Equal(tb1, 1.0)
			zs = zs.add(c0.Add(mid).Sub(dev), tb0, tb1, dir0, dir1, endpoint, false)
		}
	}
	return zs
}

// TODO: bezier-bezier intersection
// TODO: bezier-ellipse intersection

// For Bézier-Bézier intersections:
// see T.W. Sederberg, "Computer Aided Geometric Design", 2012
// see T.W. Sederberg and T. Nishita, "Curve intersection using Bézier clipping", 1990
// see T.W. Sederberg and S.R. Parry, "Comparison of three curve intersection algorithms", 1986

func IntersectionRayLine(a0, a1, b0, b1 math32.Vector2) (math32.Vector2, bool) {
	da := a1.Sub(a0)
	db := b1.Sub(b0)
	div := da.Cross(db)
	if ppath.Equal(div, 0.0) {
		// parallel
		return math32.Vector2{}, false
	}

	tb := da.Cross(a0.Sub(b0)) / div
	if inInterval(tb, 0.0, 1.0) {
		return b0.Lerp(b1, tb), true
	}
	return math32.Vector2{}, false
}

// https://mathworld.wolfram.com/Circle-LineIntersection.html
func IntersectionRayCircle(l0, l1, c math32.Vector2, r float32) (math32.Vector2, math32.Vector2, bool) {
	d := l1.Sub(l0).Normal() // along line direction, anchored in l0, its length is 1
	D := l0.Sub(c).Cross(d)
	discriminant := r*r - D*D
	if discriminant < 0 {
		return math32.Vector2{}, math32.Vector2{}, false
	}
	discriminant = math32.Sqrt(discriminant)

	ax := D * d.Y
	bx := d.X * discriminant
	if d.Y < 0.0 {
		bx = -bx
	}
	ay := -D * d.X
	by := math32.Abs(d.Y) * discriminant
	return c.Add(math32.Vector2{ax + bx, ay + by}), c.Add(math32.Vector2{ax - bx, ay - by}), true
}

// https://math32.stackexchange.com/questions/256100/how-can-i-find-the-points-at-which-two-circles-intersect
// https://gist.github.com/jupdike/bfe5eb23d1c395d8a0a1a4ddd94882ac
func IntersectionCircleCircle(c0 math32.Vector2, r0 float32, c1 math32.Vector2, r1 float32) (math32.Vector2, math32.Vector2, bool) {
	R := c0.Sub(c1).Length()
	if R < math32.Abs(r0-r1) || r0+r1 < R || ppath.EqualPoint(c0, c1) {
		return math32.Vector2{}, math32.Vector2{}, false
	}
	R2 := R * R

	k := r0*r0 - r1*r1
	a := float32(0.5)
	b := 0.5 * k / R2
	c := 0.5 * math32.Sqrt(2.0*(r0*r0+r1*r1)/R2-k*k/(R2*R2)-1.0)

	i0 := c0.Add(c1).MulScalar(a)
	i1 := c1.Sub(c0).MulScalar(b)
	i2 := math32.Vector2{c1.Y - c0.Y, c0.X - c1.X}.MulScalar(c)
	return i0.Add(i1).Add(i2), i0.Add(i1).Sub(i2), true
}
