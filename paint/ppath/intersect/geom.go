// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package intersect

import (
	"slices"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
)

// curvature returns the curvature of the path at the given index
// into ppath.Path and t in [0.0,1.0]. ppath.Path must not contain subpaths,
// and will return the path's starting curvature when i points
// to a MoveTo, or the path's final curvature when i points to
// a Close of zero-length.
func curvature(p ppath.Path, i int, t float32) float32 {
	last := len(p)
	if p[last-1] == ppath.Close && ppath.EqualPoint(math32.Vec2(p[last-ppath.CmdLen(ppath.Close)-3], p[last-ppath.CmdLen(ppath.Close)-2]), math32.Vec2(p[last-3], p[last-2])) {
		// point-closed
		last -= ppath.CmdLen(ppath.Close)
	}

	if i == 0 {
		// get path's starting direction when i points to MoveTo
		i = 4
		t = 0.0
	} else if i < len(p) && i == last {
		// get path's final direction when i points to zero-length Close
		i -= ppath.CmdLen(p[i-1])
		t = 1.0
	}
	if i < 0 || len(p) <= i || last < i+ppath.CmdLen(p[i]) {
		return 0.0
	}

	cmd := p[i]
	var start math32.Vector2
	if i == 0 {
		start = math32.Vec2(p[last-3], p[last-2])
	} else {
		start = math32.Vec2(p[i-3], p[i-2])
	}

	i += ppath.CmdLen(cmd)
	end := math32.Vec2(p[i-3], p[i-2])
	switch cmd {
	case ppath.LineTo, ppath.Close:
		return 0.0
	case ppath.QuadTo:
		cp := math32.Vec2(p[i-5], p[i-4])
		return 1.0 / quadraticBezierCurvatureRadius(start, cp, end, t)
	case ppath.CubeTo:
		cp1 := math32.Vec2(p[i-7], p[i-6])
		cp2 := math32.Vec2(p[i-5], p[i-4])
		return 1.0 / CubicBezierCurvatureRadius(start, cp1, cp2, end, t)
	case ppath.ArcTo:
		rx, ry, phi := p[i-7], p[i-6], p[i-5]
		large, sweep := ppath.ToArcFlags(p[i-4])
		_, _, theta0, theta1 := ppath.EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
		theta := theta0 + t*(theta1-theta0)
		return 1.0 / EllipseCurvatureRadius(rx, ry, sweep, theta)
	}
	return 0.0
}

// Curvature returns the curvature of the path at the given segment
// and t in [0.0,1.0] along that path. It is zero for straight lines
// and for non-existing segments.
func Curvature(p ppath.Path, seg int, t float32) float32 {
	if len(p) <= 4 {
		return 0.0
	}

	curSeg := 0
	iStart, iSeg, iEnd := 0, 0, 0
	for i := 0; i < len(p); {
		cmd := p[i]
		if cmd == ppath.MoveTo {
			if seg < curSeg {
				pi := p[iStart:iEnd]
				return curvature(pi, iSeg-iStart, t)
			}
			iStart = i
		}
		if seg == curSeg {
			iSeg = i
		}
		i += ppath.CmdLen(cmd)
	}
	return 0.0 // if segment doesn't exist
}

// windings counts intersections of ray with path.
// ppath.Paths that cross downwards are negative and upwards are positive.
// It returns the windings excluding the start position and the
// windings of the start position itself. If the windings of the
// start position is not zero, the start position is on a boundary.
func windings(zs []Intersection) (int, bool) {
	// There are four particular situations to be aware of. Whenever the path is horizontal it
	// will be parallel to the ray, and usually overlapping. Either we have:
	// - a starting point to the left of the overlapping section: ignore the overlapping
	//   intersections so that it appears as a regular intersection, albeit at the endpoints
	//   of two segments, which may either cancel out to zero (top or bottom edge) or add up to
	//   1 or -1 if the path goes upwards or downwards respectively before/after the overlap.
	// - a starting point on the left-hand corner of an overlapping section: ignore if either
	//   intersection of an endpoint pair (t=0,t=1) is overlapping, but count for nb upon
	//   leaving the overlap.
	// - a starting point in the middle of an overlapping section: same as above
	// - a starting point on the right-hand corner of an overlapping section: intersections are
	//   tangent and thus already ignored for n, but for nb we should ignore the intersection with
	//   a 0/180 degree direction, and count the other

	n := 0
	boundary := false
	for i := 0; i < len(zs); i++ {
		z := zs[i]
		if z.T[0] == 0.0 {
			boundary = true
			continue
		}

		d := 1
		if z.Into() {
			d = -1 // downwards
		}
		if z.T[1] != 0.0 && z.T[1] != 1.0 {
			if !z.Same {
				n += d
			}
		} else if i+1 < len(zs) {
			same := z.Same || (len(zs) > i+1 && zs[i+1].Same)
			if !same && len(zs) > i+1 {
				if z.Into() == zs[i+1].Into() {
					n += d
				}
			}
			i++
		}
	}
	return n, boundary
}

// Windings returns the number of windings at the given point,
// i.e. the sum of windings for each time a ray from (x,y)
// towards (∞,y) intersects the path. Counter clock-wise
// intersections count as positive, while clock-wise intersections
// count as negative. Additionally, it returns whether the point
// is on a path's boundary (which counts as being on the exterior).
func Windings(p ppath.Path, x, y float32) (int, bool) {
	n := 0
	boundary := false
	for _, pi := range p.Split() {
		zs := RayIntersections(pi, x, y)
		if ni, boundaryi := windings(zs); boundaryi {
			boundary = true
		} else {
			n += ni
		}
	}
	return n, boundary
}

// Crossings returns the number of crossings with the path from the
// given point outwards, i.e. the number of times a ray from (x,y)
// towards (∞,y) intersects the path. Additionally, it returns whether
// the point is on a path's boundary (which does not count towards
// the number of crossings).
func Crossings(p ppath.Path, x, y float32) (int, bool) {
	n := 0
	boundary := false
	for _, pi := range p.Split() {
		// Count intersections of ray with path. Count half an intersection on boundaries.
		ni := 0.0
		for _, z := range RayIntersections(pi, x, y) {
			if z.T[0] == 0.0 {
				boundary = true
			} else if !z.Same {
				if z.T[1] == 0.0 || z.T[1] == 1.0 {
					ni += 0.5
				} else {
					ni += 1.0
				}
			} else if z.T[1] == 0.0 || z.T[1] == 1.0 {
				ni -= 0.5
			}
		}
		n += int(ni)
	}
	return n, boundary
}

// Contains returns whether the point (x,y) is contained/filled by the path.
// This depends on the ppath.FillRules. It uses a ray from (x,y) toward (∞,y) and
// counts the number of intersections with the path.
// When the point is on the boundary it is considered to be on the path's exterior.
func Contains(p ppath.Path, x, y float32, fillRule ppath.FillRules) bool {
	n, boundary := Windings(p, x, y)
	if boundary {
		return true
	}
	return fillRule.Fills(n)
}

// CCW returns true when the path is counter clockwise oriented at its
// bottom-right-most coordinate. It is most useful when knowing that
// the path does not self-intersect as it will tell you if the entire
// path is CCW or not. It will only return the result for the first subpath.
// It will return true for an empty path or a straight line.
// It may not return a valid value when the right-most point happens to be a
// (self-)overlapping segment.
func CCW(p ppath.Path) bool {
	if len(p) <= 4 || (p[4] == ppath.LineTo || p[4] == ppath.Close) && len(p) <= 4+ppath.CmdLen(p[4]) {
		// empty path or single straight segment
		return true
	}

	p = XMonotone(p)

	// pick bottom-right-most coordinate of subpath, as we know its left-hand side is filling
	k, kMax := 4, len(p)
	if p[kMax-1] == ppath.Close {
		kMax -= ppath.CmdLen(ppath.Close)
	}
	for i := 4; i < len(p); {
		cmd := p[i]
		if cmd == ppath.MoveTo {
			// only handle first subpath
			kMax = i
			break
		}
		i += ppath.CmdLen(cmd)
		if x, y := p[i-3], p[i-2]; p[k-3] < x || ppath.Equal(p[k-3], x) && y < p[k-2] {
			k = i
		}
	}

	// get coordinates of previous and next segments
	var kPrev int
	if k == 4 {
		kPrev = kMax
	} else {
		kPrev = k - ppath.CmdLen(p[k-1])
	}

	var angleNext float32
	anglePrev := ppath.AngleNorm(ppath.Angle(ppath.DirectionIndex(p, kPrev, 1.0)) + math32.Pi)
	if k == kMax {
		// use implicit close command
		angleNext = ppath.Angle(math32.Vec2(p[1], p[2]).Sub(math32.Vec2(p[k-3], p[k-2])))
	} else {
		angleNext = ppath.Angle(ppath.DirectionIndex(p, k, 0.0))
	}
	if ppath.Equal(anglePrev, angleNext) {
		// segments have the same direction at their right-most point
		// one or both are not straight lines, check if curvature is different
		var curvNext float32
		curvPrev := -curvature(p, kPrev, 1.0)
		if k == kMax {
			// use implicit close command
			curvNext = 0.0
		} else {
			curvNext = curvature(p, k, 0.0)
		}
		if !ppath.Equal(curvPrev, curvNext) {
			// ccw if curvNext is smaller than curvPrev
			return curvNext < curvPrev
		}
	}
	return (angleNext - anglePrev) < 0.0
}

// Filling returns whether each subpath gets filled or not.
// Whether a path is filled depends on the ppath.FillRules and whether it
// negates another path. If a subpath is not closed, it is implicitly
// assumed to be closed.
func Filling(p ppath.Path, fillRule ppath.FillRules) []bool {
	ps := p.Split()
	filling := make([]bool, len(ps))
	for i, pi := range ps {
		// get current subpath's winding
		n := 0
		if CCW(pi) {
			n++
		} else {
			n--
		}

		// sum windings from other subpaths
		pos := math32.Vec2(pi[1], pi[2])
		for j, pj := range ps {
			if i == j {
				continue
			}
			zs := RayIntersections(pj, pos.X, pos.Y)
			if ni, boundaryi := windings(zs); !boundaryi {
				n += ni
			} else {
				// on the boundary, check if around the interior or exterior of pos
			}
		}
		filling[i] = fillRule.Fills(n)
	}
	return filling
}

// Length returns the length of the path in millimeters.
// The length is approximated for cubic Béziers.
func Length(p ppath.Path) float32 {
	d := float32(0.0)
	var start, end math32.Vector2
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case ppath.MoveTo:
			end = math32.Vec2(p[i+1], p[i+2])
		case ppath.LineTo, ppath.Close:
			end = math32.Vec2(p[i+1], p[i+2])
			d += end.Sub(start).Length()
		case ppath.QuadTo:
			cp := math32.Vec2(p[i+1], p[i+2])
			end = math32.Vec2(p[i+3], p[i+4])
			d += quadraticBezierLength(start, cp, end)
		case ppath.CubeTo:
			cp1 := math32.Vec2(p[i+1], p[i+2])
			cp2 := math32.Vec2(p[i+3], p[i+4])
			end = math32.Vec2(p[i+5], p[i+6])
			d += cubicBezierLength(start, cp1, cp2, end)
		case ppath.ArcTo:
			var rx, ry, phi float32
			var large, sweep bool
			rx, ry, phi, large, sweep, end = p.ArcToPoints(i)
			_, _, theta1, theta2 := ppath.EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			d += ellipseLength(rx, ry, theta1, theta2)
		}
		i += ppath.CmdLen(cmd)
		start = end
	}
	return d
}

// IsFlat returns true if the path consists of solely line segments,
// that is only MoveTo, ppath.LineTo and Close commands.
func IsFlat(p ppath.Path) bool {
	for i := 0; i < len(p); {
		cmd := p[i]
		if cmd != ppath.MoveTo && cmd != ppath.LineTo && cmd != ppath.Close {
			return false
		}
		i += ppath.CmdLen(cmd)
	}
	return true
}

// SplitAt splits the path into separate paths at the specified
// intervals (given in millimeters) along the path.
func SplitAt(p ppath.Path, ts ...float32) []ppath.Path {
	if len(ts) == 0 {
		return []ppath.Path{p}
	}

	slices.Sort(ts)
	if ts[0] == 0.0 {
		ts = ts[1:]
	}

	j := 0            // index into ts
	T := float32(0.0) // current position along curve

	qs := []ppath.Path{}
	q := ppath.Path{}
	push := func() {
		qs = append(qs, q)
		q = ppath.Path{}
	}

	if 0 < len(p) && p[0] == ppath.MoveTo {
		q.MoveTo(p[1], p[2])
	}
	for _, ps := range p.Split() {
		var start, end math32.Vector2
		for i := 0; i < len(ps); {
			cmd := ps[i]
			switch cmd {
			case ppath.MoveTo:
				end = math32.Vec2(p[i+1], p[i+2])
			case ppath.LineTo, ppath.Close:
				end = math32.Vec2(p[i+1], p[i+2])

				if j == len(ts) {
					q.LineTo(end.X, end.Y)
				} else {
					dT := end.Sub(start).Length()
					Tcurve := T
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						tpos := (ts[j] - T) / dT
						pos := start.Lerp(end, tpos)
						Tcurve = ts[j]

						q.LineTo(pos.X, pos.Y)
						push()
						q.MoveTo(pos.X, pos.Y)
						j++
					}
					if Tcurve < T+dT {
						q.LineTo(end.X, end.Y)
					}
					T += dT
				}
			case ppath.QuadTo:
				cp := math32.Vec2(p[i+1], p[i+2])
				end = math32.Vec2(p[i+3], p[i+4])

				if j == len(ts) {
					q.QuadTo(cp.X, cp.Y, end.X, end.Y)
				} else {
					speed := func(t float32) float32 {
						return ppath.QuadraticBezierDeriv(start, cp, end, t).Length()
					}
					invL, dT := invSpeedPolynomialChebyshevApprox(20, gaussLegendre7, speed, 0.0, 1.0)

					t0 := float32(0.0)
					r0, r1, r2 := start, cp, end
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						t := invL(ts[j] - T)
						tsub := (t - t0) / (1.0 - t0)
						t0 = t

						var q1 math32.Vector2
						_, q1, _, r0, r1, r2 = quadraticBezierSplit(r0, r1, r2, tsub)

						q.QuadTo(q1.X, q1.Y, r0.X, r0.Y)
						push()
						q.MoveTo(r0.X, r0.Y)
						j++
					}
					if !ppath.Equal(t0, 1.0) {
						q.QuadTo(r1.X, r1.Y, r2.X, r2.Y)
					}
					T += dT
				}
			case ppath.CubeTo:
				cp1 := math32.Vec2(p[i+1], p[i+2])
				cp2 := math32.Vec2(p[i+3], p[i+4])
				end = math32.Vec2(p[i+5], p[i+6])

				if j == len(ts) {
					q.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, end.X, end.Y)
				} else {
					speed := func(t float32) float32 {
						// splitting on inflection points does not improve output
						return ppath.CubicBezierDeriv(start, cp1, cp2, end, t).Length()
					}
					N := 20 + 20*cubicBezierNumInflections(start, cp1, cp2, end) // TODO: needs better N
					invL, dT := invSpeedPolynomialChebyshevApprox(N, gaussLegendre7, speed, 0.0, 1.0)

					t0 := float32(0.0)
					r0, r1, r2, r3 := start, cp1, cp2, end
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						t := invL(ts[j] - T)
						tsub := (t - t0) / (1.0 - t0)
						t0 = t

						var q1, q2 math32.Vector2
						_, q1, q2, _, r0, r1, r2, r3 = cubicBezierSplit(r0, r1, r2, r3, tsub)

						q.CubeTo(q1.X, q1.Y, q2.X, q2.Y, r0.X, r0.Y)
						push()
						q.MoveTo(r0.X, r0.Y)
						j++
					}
					if !ppath.Equal(t0, 1.0) {
						q.CubeTo(r1.X, r1.Y, r2.X, r2.Y, r3.X, r3.Y)
					}
					T += dT
				}
			case ppath.ArcTo:
				var rx, ry, phi float32
				var large, sweep bool
				rx, ry, phi, large, sweep, end = p.ArcToPoints(i)
				cx, cy, theta1, theta2 := ppath.EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

				if j == len(ts) {
					q.ArcTo(rx, ry, phi, large, sweep, end.X, end.Y)
				} else {
					speed := func(theta float32) float32 {
						return ppath.EllipseDeriv(rx, ry, 0.0, true, theta).Length()
					}
					invL, dT := invSpeedPolynomialChebyshevApprox(10, gaussLegendre7, speed, theta1, theta2)

					startTheta := theta1
					nextLarge := large
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						theta := invL(ts[j] - T)
						mid, large1, large2, ok := ellipseSplit(rx, ry, phi, cx, cy, startTheta, theta2, theta)
						if !ok {
							panic("theta not in elliptic arc range for splitting")
						}

						q.ArcTo(rx, ry, phi, large1, sweep, mid.X, mid.Y)
						push()
						q.MoveTo(mid.X, mid.Y)
						startTheta = theta
						nextLarge = large2
						j++
					}
					if !ppath.Equal(startTheta, theta2) {
						q.ArcTo(rx, ry, phi*180.0/math32.Pi, nextLarge, sweep, end.X, end.Y)
					}
					T += dT
				}
			}
			i += ppath.CmdLen(cmd)
			start = end
		}
	}
	if ppath.CmdLen(ppath.MoveTo) < len(q) {
		push()
	}
	return qs
}

// XMonotone replaces all Bézier and arc segments to be x-monotone
// and returns a new path, that is each path segment is either increasing
// or decreasing with X while moving across the segment.
// This is always true for line segments.
func XMonotone(p ppath.Path) ppath.Path {
	quad := func(p0, p1, p2 math32.Vector2) ppath.Path {
		return xmonotoneQuadraticBezier(p0, p1, p2)
	}
	cube := func(p0, p1, p2, p3 math32.Vector2) ppath.Path {
		return xmonotoneCubicBezier(p0, p1, p2, p3)
	}
	arc := func(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) ppath.Path {
		return xmonotoneEllipticArc(start, rx, ry, phi, large, sweep, end)
	}
	return p.Replace(nil, quad, cube, arc)
}

// Bounds returns the exact bounding box rectangle of the path.
func Bounds(p ppath.Path) math32.Box2 {
	if len(p) < 4 {
		return math32.Box2{}
	}

	// first command is MoveTo
	start, end := math32.Vec2(p[1], p[2]), math32.Vector2{}
	xmin, xmax := start.X, start.X
	ymin, ymax := start.Y, start.Y
	for i := 4; i < len(p); {
		cmd := p[i]
		switch cmd {
		case ppath.MoveTo, ppath.LineTo, ppath.Close:
			end = math32.Vec2(p[i+1], p[i+2])
			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
		case ppath.QuadTo:
			cp := math32.Vec2(p[i+1], p[i+2])
			end = math32.Vec2(p[i+3], p[i+4])

			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			if tdenom := (start.X - 2*cp.X + end.X); !ppath.Equal(tdenom, 0.0) {
				if t := (start.X - cp.X) / tdenom; inIntervalExclusive(t, 0.0, 1.0) {
					x := quadraticBezierPos(start, cp, end, t)
					xmin = math32.Min(xmin, x.X)
					xmax = math32.Max(xmax, x.X)
				}
			}

			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
			if tdenom := (start.Y - 2*cp.Y + end.Y); !ppath.Equal(tdenom, 0.0) {
				if t := (start.Y - cp.Y) / tdenom; inIntervalExclusive(t, 0.0, 1.0) {
					y := quadraticBezierPos(start, cp, end, t)
					ymin = math32.Min(ymin, y.Y)
					ymax = math32.Max(ymax, y.Y)
				}
			}
		case ppath.CubeTo:
			cp1 := math32.Vec2(p[i+1], p[i+2])
			cp2 := math32.Vec2(p[i+3], p[i+4])
			end = math32.Vec2(p[i+5], p[i+6])

			a := -start.X + 3*cp1.X - 3*cp2.X + end.X
			b := 2*start.X - 4*cp1.X + 2*cp2.X
			c := -start.X + cp1.X
			t1, t2 := solveQuadraticFormula(a, b, c)

			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			if !math32.IsNaN(t1) && inIntervalExclusive(t1, 0.0, 1.0) {
				x1 := cubicBezierPos(start, cp1, cp2, end, t1)
				xmin = math32.Min(xmin, x1.X)
				xmax = math32.Max(xmax, x1.X)
			}
			if !math32.IsNaN(t2) && inIntervalExclusive(t2, 0.0, 1.0) {
				x2 := cubicBezierPos(start, cp1, cp2, end, t2)
				xmin = math32.Min(xmin, x2.X)
				xmax = math32.Max(xmax, x2.X)
			}

			a = -start.Y + 3*cp1.Y - 3*cp2.Y + end.Y
			b = 2*start.Y - 4*cp1.Y + 2*cp2.Y
			c = -start.Y + cp1.Y
			t1, t2 = solveQuadraticFormula(a, b, c)

			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
			if !math32.IsNaN(t1) && inIntervalExclusive(t1, 0.0, 1.0) {
				y1 := cubicBezierPos(start, cp1, cp2, end, t1)
				ymin = math32.Min(ymin, y1.Y)
				ymax = math32.Max(ymax, y1.Y)
			}
			if !math32.IsNaN(t2) && inIntervalExclusive(t2, 0.0, 1.0) {
				y2 := cubicBezierPos(start, cp1, cp2, end, t2)
				ymin = math32.Min(ymin, y2.Y)
				ymax = math32.Max(ymax, y2.Y)
			}
		case ppath.ArcTo:
			var rx, ry, phi float32
			var large, sweep bool
			rx, ry, phi, large, sweep, end = p.ArcToPoints(i)
			cx, cy, theta0, theta1 := ppath.EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

			// find the four extremes (top, bottom, left, right) and apply those who are between theta1 and theta2
			// x(theta) = cx + rx*cos(theta)*cos(phi) - ry*sin(theta)*sin(phi)
			// y(theta) = cy + rx*cos(theta)*sin(phi) + ry*sin(theta)*cos(phi)
			// be aware that positive rotation appears clockwise in SVGs (non-Cartesian coordinate system)
			// we can now find the angles of the extremes

			sinphi, cosphi := math32.Sincos(phi)
			thetaRight := math32.Atan2(-ry*sinphi, rx*cosphi)
			thetaTop := math32.Atan2(rx*cosphi, ry*sinphi)
			thetaLeft := thetaRight + math32.Pi
			thetaBottom := thetaTop + math32.Pi

			dx := math32.Sqrt(rx*rx*cosphi*cosphi + ry*ry*sinphi*sinphi)
			dy := math32.Sqrt(rx*rx*sinphi*sinphi + ry*ry*cosphi*cosphi)
			if ppath.IsAngleBetween(thetaLeft, theta0, theta1) {
				xmin = math32.Min(xmin, cx-dx)
			}
			if ppath.IsAngleBetween(thetaRight, theta0, theta1) {
				xmax = math32.Max(xmax, cx+dx)
			}
			if ppath.IsAngleBetween(thetaBottom, theta0, theta1) {
				ymin = math32.Min(ymin, cy-dy)
			}
			if ppath.IsAngleBetween(thetaTop, theta0, theta1) {
				ymax = math32.Max(ymax, cy+dy)
			}
			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
		}
		i += ppath.CmdLen(cmd)
		start = end
	}
	return math32.B2(xmin, ymin, xmax, ymax)
}
