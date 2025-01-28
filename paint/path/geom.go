// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

import "cogentcore.org/core/math32"

// direction returns the direction of the path at the given index
// into Path and t in [0.0,1.0]. Path must not contain subpaths,
// and will return the path's starting direction when i points
// to a MoveTo, or the path's final direction when i points to
// a Close of zero-length.
func (p Path) direction(i int, t float32) math32.Vector2 {
	last := len(p)
	if p[last-1] == Close && EqualPoint(math32.Vec2(p[last-CmdLen(Close)-3], p[last-CmdLen(Close)-2]), math32.Vec2(p[last-3], p[last-2])) {
		// point-closed
		last -= CmdLen(Close)
	}

	if i == 0 {
		// get path's starting direction when i points to MoveTo
		i = 4
		t = 0.0
	} else if i < len(p) && i == last {
		// get path's final direction when i points to zero-length Close
		i -= CmdLen(p[i-1])
		t = 1.0
	}
	if i < 0 || len(p) <= i || last < i+CmdLen(p[i]) {
		return math32.Vector2{}
	}

	cmd := p[i]
	var start math32.Vector2
	if i == 0 {
		start = math32.Vec2(p[last-3], p[last-2])
	} else {
		start = math32.Vec2(p[i-3], p[i-2])
	}

	i += CmdLen(cmd)
	end := math32.Vec2(p[i-3], p[i-2])
	switch cmd {
	case LineTo, Close:
		return end.Sub(start).Normal()
	case QuadTo:
		cp := math32.Vec2(p[i-5], p[i-4])
		return quadraticBezierDeriv(start, cp, end, t).Normal()
	case CubeTo:
		cp1 := math32.Vec2(p[i-7], p[i-6])
		cp2 := math32.Vec2(p[i-5], p[i-4])
		return cubicBezierDeriv(start, cp1, cp2, end, t).Normal()
	case ArcTo:
		rx, ry, phi := p[i-7], p[i-6], p[i-5]
		large, sweep := toArcFlags(p[i-4])
		_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
		theta := theta0 + t*(theta1-theta0)
		return ellipseDeriv(rx, ry, phi, sweep, theta).Normal()
	}
	return math32.Vector2{}
}

// Direction returns the direction of the path at the given
// segment and t in [0.0,1.0] along that path.
// The direction is a vector of unit length.
func (p Path) Direction(seg int, t float32) math32.Vector2 {
	if len(p) <= 4 {
		return math32.Vector2{}
	}

	curSeg := 0
	iStart, iSeg, iEnd := 0, 0, 0
	for i := 0; i < len(p); {
		cmd := p[i]
		if cmd == MoveTo {
			if seg < curSeg {
				pi := p[iStart:iEnd]
				return pi.direction(iSeg-iStart, t)
			}
			iStart = i
		}
		if seg == curSeg {
			iSeg = i
		}
		i += CmdLen(cmd)
	}
	return math32.Vector2{} // if segment doesn't exist
}

// CoordDirections returns the direction of the segment start/end points.
// It will return the average direction at the intersection of two
// end points, and for an open path it will simply return the direction
// of the start and end points of the path.
func (p Path) CoordDirections() []math32.Vector2 {
	if len(p) <= 4 {
		return []math32.Vector2{{}}
	}
	last := len(p)
	if p[last-1] == Close && EqualPoint(math32.Vec2(p[last-CmdLen(Close)-3], p[last-CmdLen(Close)-2]), math32.Vec2(p[last-3], p[last-2])) {
		// point-closed
		last -= CmdLen(Close)
	}

	dirs := []math32.Vector2{}
	var closed bool
	var dirPrev math32.Vector2
	for i := 4; i < last; {
		cmd := p[i]
		dir := p.direction(i, 0.0)
		if i == 0 {
			dirs = append(dirs, dir)
		} else {
			dirs = append(dirs, dirPrev.Add(dir).Normal())
		}
		dirPrev = p.direction(i, 1.0)
		closed = cmd == Close
		i += CmdLen(cmd)
	}
	if closed {
		dirs[0] = dirs[0].Add(dirPrev).Normal()
		dirs = append(dirs, dirs[0])
	} else {
		dirs = append(dirs, dirPrev)
	}
	return dirs
}

// curvature returns the curvature of the path at the given index
// into Path and t in [0.0,1.0]. Path must not contain subpaths,
// and will return the path's starting curvature when i points
// to a MoveTo, or the path's final curvature when i points to
// a Close of zero-length.
func (p Path) curvature(i int, t float32) float32 {
	last := len(p)
	if p[last-1] == Close && EqualPoint(math32.Vec2(p[last-CmdLen(Close)-3], p[last-CmdLen(Close)-2]), math32.Vec2(p[last-3], p[last-2])) {
		// point-closed
		last -= CmdLen(Close)
	}

	if i == 0 {
		// get path's starting direction when i points to MoveTo
		i = 4
		t = 0.0
	} else if i < len(p) && i == last {
		// get path's final direction when i points to zero-length Close
		i -= CmdLen(p[i-1])
		t = 1.0
	}
	if i < 0 || len(p) <= i || last < i+CmdLen(p[i]) {
		return 0.0
	}

	cmd := p[i]
	var start math32.Vector2
	if i == 0 {
		start = math32.Vec2(p[last-3], p[last-2])
	} else {
		start = math32.Vec2(p[i-3], p[i-2])
	}

	i += CmdLen(cmd)
	end := math32.Vec2(p[i-3], p[i-2])
	switch cmd {
	case LineTo, Close:
		return 0.0
	case QuadTo:
		cp := math32.Vec2(p[i-5], p[i-4])
		return 1.0 / quadraticBezierCurvatureRadius(start, cp, end, t)
	case CubeTo:
		cp1 := math32.Vec2(p[i-7], p[i-6])
		cp2 := math32.Vec2(p[i-5], p[i-4])
		return 1.0 / cubicBezierCurvatureRadius(start, cp1, cp2, end, t)
	case ArcTo:
		rx, ry, phi := p[i-7], p[i-6], p[i-5]
		large, sweep := toArcFlags(p[i-4])
		_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
		theta := theta0 + t*(theta1-theta0)
		return 1.0 / ellipseCurvatureRadius(rx, ry, sweep, theta)
	}
	return 0.0
}

// Curvature returns the curvature of the path at the given segment
// and t in [0.0,1.0] along that path. It is zero for straight lines
// and for non-existing segments.
func (p Path) Curvature(seg int, t float32) float32 {
	if len(p) <= 4 {
		return 0.0
	}

	curSeg := 0
	iStart, iSeg, iEnd := 0, 0, 0
	for i := 0; i < len(p); {
		cmd := p[i]
		if cmd == MoveTo {
			if seg < curSeg {
				pi := p[iStart:iEnd]
				return pi.curvature(iSeg-iStart, t)
			}
			iStart = i
		}
		if seg == curSeg {
			iSeg = i
		}
		i += CmdLen(cmd)
	}
	return 0.0 // if segment doesn't exist
}

// windings counts intersections of ray with path.
// Paths that cross downwards are negative and upwards are positive.
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
		} else {
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
func (p Path) Windings(x, y float32) (int, bool) {
	n := 0
	boundary := false
	for _, pi := range p.Split() {
		zs := pi.RayIntersections(x, y)
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
func (p Path) Crossings(x, y float32) (int, bool) {
	n := 0
	boundary := false
	for _, pi := range p.Split() {
		// Count intersections of ray with path. Count half an intersection on boundaries.
		ni := 0.0
		for _, z := range pi.RayIntersections(x, y) {
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
// This depends on the FillRules. It uses a ray from (x,y) toward (∞,y) and
// counts the number of intersections with the path.
// When the point is on the boundary it is considered to be on the path's exterior.
func (p Path) Contains(x, y float32, fillRule FillRules) bool {
	n, boundary := p.Windings(x, y)
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
func (p Path) CCW() bool {
	if len(p) <= 4 || (p[4] == LineTo || p[4] == Close) && len(p) <= 4+CmdLen(p[4]) {
		// empty path or single straight segment
		return true
	}

	p = p.XMonotone()

	// pick bottom-right-most coordinate of subpath, as we know its left-hand side is filling
	k, kMax := 4, len(p)
	if p[kMax-1] == Close {
		kMax -= CmdLen(Close)
	}
	for i := 4; i < len(p); {
		cmd := p[i]
		if cmd == MoveTo {
			// only handle first subpath
			kMax = i
			break
		}
		i += CmdLen(cmd)
		if x, y := p[i-3], p[i-2]; p[k-3] < x || Equal(p[k-3], x) && y < p[k-2] {
			k = i
		}
	}

	// get coordinates of previous and next segments
	var kPrev int
	if k == 4 {
		kPrev = kMax
	} else {
		kPrev = k - CmdLen(p[k-1])
	}

	var angleNext float32
	anglePrev := angleNorm(Angle(p.direction(kPrev, 1.0)) + math32.Pi)
	if k == kMax {
		// use implicit close command
		angleNext = Angle(math32.Vec2(p[1], p[2]).Sub(math32.Vec2(p[k-3], p[k-2])))
	} else {
		angleNext = Angle(p.direction(k, 0.0))
	}
	if Equal(anglePrev, angleNext) {
		// segments have the same direction at their right-most point
		// one or both are not straight lines, check if curvature is different
		var curvNext float32
		curvPrev := -p.curvature(kPrev, 1.0)
		if k == kMax {
			// use implicit close command
			curvNext = 0.0
		} else {
			curvNext = p.curvature(k, 0.0)
		}
		if !Equal(curvPrev, curvNext) {
			// ccw if curvNext is smaller than curvPrev
			return curvNext < curvPrev
		}
	}
	return (angleNext - anglePrev) < 0.0
}

// Filling returns whether each subpath gets filled or not.
// Whether a path is filled depends on the FillRules and whether it
// negates another path. If a subpath is not closed, it is implicitly
// assumed to be closed.
func (p Path) Filling(fillRule FillRules) []bool {
	ps := p.Split()
	filling := make([]bool, len(ps))
	for i, pi := range ps {
		// get current subpath's winding
		n := 0
		if pi.CCW() {
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
			zs := pj.RayIntersections(pos.X, pos.Y)
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
func (p Path) Length() float32 {
	d := float32(0.0)
	var start, end math32.Vector2
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			end = math32.Vec2(p[i+1], p[i+2])
		case LineTo, Close:
			end = math32.Vec2(p[i+1], p[i+2])
			d += end.Sub(start).Length()
		case QuadTo:
			cp := math32.Vec2(p[i+1], p[i+2])
			end = math32.Vec2(p[i+3], p[i+4])
			d += quadraticBezierLength(start, cp, end)
		case CubeTo:
			cp1 := math32.Vec2(p[i+1], p[i+2])
			cp2 := math32.Vec2(p[i+3], p[i+4])
			end = math32.Vec2(p[i+5], p[i+6])
			d += cubicBezierLength(start, cp1, cp2, end)
		case ArcTo:
			var rx, ry, phi float32
			var large, sweep bool
			rx, ry, phi, large, sweep, end = p.ArcToPoints(i)
			_, _, theta1, theta2 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			d += ellipseLength(rx, ry, theta1, theta2)
		}
		i += CmdLen(cmd)
		start = end
	}
	return d
}

// IsFlat returns true if the path consists of solely line segments,
// that is only MoveTo, LineTo and Close commands.
func (p Path) IsFlat() bool {
	for i := 0; i < len(p); {
		cmd := p[i]
		if cmd != MoveTo && cmd != LineTo && cmd != Close {
			return false
		}
		i += CmdLen(cmd)
	}
	return true
}
