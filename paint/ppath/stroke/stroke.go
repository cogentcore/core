// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package stroke

//go:generate core generate

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/ppath/intersect"
)

// ellipseNormal returns the normal to the right at angle theta of the ellipse, given rotation phi.
func ellipseNormal(rx, ry, phi float32, sweep bool, theta, d float32) math32.Vector2 {
	return ppath.EllipseDeriv(rx, ry, phi, sweep, theta).Rot90CW().Normal().MulScalar(d)
}

// NOTE: implementation inspired from github.com/golang/freetype/raster/stroke.go

// Stroke converts a path into a stroke of width w and returns a new path.
// It uses cr to cap the start and end of the path, and jr to join all path elements.
// If the path closes itself, it will use a join between the start and end instead
// of capping them. The tolerance is the maximum deviation from the original path
// when flattening Béziers and optimizing the stroke.
func Stroke(p ppath.Path, w float32, cr Capper, jr Joiner, tolerance float32) ppath.Path {
	if cr == nil {
		cr = ButtCap
	}
	if jr == nil {
		jr = MiterJoin
	}
	q := ppath.Path{}
	halfWidth := math32.Abs(w) / 2.0
	for _, pi := range p.Split() {
		rhs, lhs := offset(pi, halfWidth, cr, jr, true, tolerance)
		if rhs == nil {
			continue
		} else if lhs == nil {
			// open path
			q = q.Append(intersect.Settle(rhs, ppath.Positive))
		} else {
			// closed path
			// inner path should go opposite direction to cancel the outer path
			if intersect.CCW(pi) {
				q = q.Append(intersect.Settle(rhs, ppath.Positive))
				q = q.Append(intersect.Settle(lhs, ppath.Positive).Reverse())
			} else {
				// outer first, then inner
				q = q.Append(intersect.Settle(lhs, ppath.Negative))
				q = q.Append(intersect.Settle(rhs, ppath.Negative).Reverse())
			}
		}
	}
	return q
}

func CapFromStyle(st ppath.Caps) Capper {
	switch st {
	case ppath.CapButt:
		return ButtCap
	case ppath.CapRound:
		return RoundCap
	case ppath.CapSquare:
		return SquareCap
	}
	return ButtCap
}

func JoinFromStyle(st ppath.Joins) Joiner {
	switch st {
	case ppath.JoinMiter:
		return MiterJoin
	case ppath.JoinMiterClip:
		return MiterClipJoin
	case ppath.JoinRound:
		return RoundJoin
	case ppath.JoinBevel:
		return BevelJoin
	case ppath.JoinArcs:
		return ArcsJoin
	case ppath.JoinArcsClip:
		return ArcsClipJoin
	}
	return MiterJoin
}

// Capper implements Cap, with rhs the path to append to,
// halfWidth the half width of the stroke, pivot the pivot point around
// which to construct a cap, and n0 the normal at the start of the path.
// The length of n0 is equal to the halfWidth.
type Capper interface {
	Cap(*ppath.Path, float32, math32.Vector2, math32.Vector2)
}

// RoundCap caps the start or end of a path by a round cap.
var RoundCap Capper = RoundCapper{}

// RoundCapper is a round capper.
type RoundCapper struct{}

// Cap adds a cap to path p of width 2*halfWidth,
// at a pivot point and initial normal direction of n0.
func (RoundCapper) Cap(p *ppath.Path, halfWidth float32, pivot, n0 math32.Vector2) {
	end := pivot.Sub(n0)
	p.ArcTo(halfWidth, halfWidth, 0, false, true, end.X, end.Y)
}

func (RoundCapper) String() string {
	return "Round"
}

// ButtCap caps the start or end of a path by a butt cap.
var ButtCap Capper = ButtCapper{}

// ButtCapper is a butt capper.
type ButtCapper struct{}

// Cap adds a cap to path p of width 2*halfWidth,
// at a pivot point and initial normal direction of n0.
func (ButtCapper) Cap(p *ppath.Path, halfWidth float32, pivot, n0 math32.Vector2) {
	end := pivot.Sub(n0)
	p.LineTo(end.X, end.Y)
}

func (ButtCapper) String() string {
	return "Butt"
}

// SquareCap caps the start or end of a path by a square cap.
var SquareCap Capper = SquareCapper{}

// SquareCapper is a square capper.
type SquareCapper struct{}

// Cap adds a cap to path p of width 2*halfWidth,
// at a pivot point and initial normal direction of n0.
func (SquareCapper) Cap(p *ppath.Path, halfWidth float32, pivot, n0 math32.Vector2) {
	e := n0.Rot90CCW()
	corner1 := pivot.Add(e).Add(n0)
	corner2 := pivot.Add(e).Sub(n0)
	end := pivot.Sub(n0)
	p.LineTo(corner1.X, corner1.Y)
	p.LineTo(corner2.X, corner2.Y)
	p.LineTo(end.X, end.Y)
}

func (SquareCapper) String() string {
	return "Square"
}

////////

// Joiner implements Join, with rhs the right path and lhs the left path
// to append to, pivot the intersection of both path elements, n0 and n1
// the normals at the start and end of the path respectively.
// The length of n0 and n1 are equal to the halfWidth.
type Joiner interface {
	Join(*ppath.Path, *ppath.Path, float32, math32.Vector2, math32.Vector2, math32.Vector2, float32, float32)
}

// BevelJoin connects two path elements by a linear join.
var BevelJoin Joiner = BevelJoiner{}

// BevelJoiner is a bevel joiner.
type BevelJoiner struct{}

// Join adds a join to a right-hand-side and left-hand-side path,
// of width 2*halfWidth, around a pivot point with starting and
// ending normals of n0 and n1, and radius of curvatures of the
// previous and next segments.
func (BevelJoiner) Join(rhs, lhs *ppath.Path, halfWidth float32, pivot, n0, n1 math32.Vector2, r0, r1 float32) {
	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

func (BevelJoiner) String() string {
	return "Bevel"
}

// RoundJoin connects two path elements by a round join.
var RoundJoin Joiner = RoundJoiner{}

// RoundJoiner is a round joiner.
type RoundJoiner struct{}

func (RoundJoiner) Join(rhs, lhs *ppath.Path, halfWidth float32, pivot, n0, n1 math32.Vector2, r0, r1 float32) {
	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	cw := 0.0 <= n0.Rot90CW().Dot(n1)
	if cw { // bend to the right, ie. CW (or 180 degree turn)
		rhs.LineTo(rEnd.X, rEnd.Y)
		lhs.ArcTo(halfWidth, halfWidth, 0.0, false, false, lEnd.X, lEnd.Y)
	} else { // bend to the left, ie. CCW
		rhs.ArcTo(halfWidth, halfWidth, 0.0, false, true, rEnd.X, rEnd.Y)
		lhs.LineTo(lEnd.X, lEnd.Y)
	}
}

func (RoundJoiner) String() string {
	return "Round"
}

// MiterJoin connects two path elements by extending the ends
// of the paths as lines until they meet.
// If this point is further than the limit, this will result in a bevel
// join (MiterJoin) or they will meet at the limit (MiterClipJoin).
var MiterJoin Joiner = MiterJoiner{BevelJoin, 4.0}
var MiterClipJoin Joiner = MiterJoiner{nil, 4.0} // TODO: should extend limit*halfwidth before bevel

// MiterJoiner is a miter joiner.
type MiterJoiner struct {
	GapJoiner Joiner
	Limit     float32
}

func (j MiterJoiner) Join(rhs, lhs *ppath.Path, halfWidth float32, pivot, n0, n1 math32.Vector2, r0, r1 float32) {
	if ppath.EqualPoint(n0, n1.Negate()) {
		BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}

	cw := 0.0 <= n0.Rot90CW().Dot(n1)
	hw := halfWidth
	if cw {
		hw = -hw // used to calculate |R|, when running CW then n0 and n1 point the other way, so the sign of r0 and r1 is negated
	}

	// note that cos(theta) below refers to sin(theta/2) in the documentation of stroke-miterlimit
	// in https://developer.mozilla.org/en-US/docs/Web/SVG/Attribute/stroke-miterlimit
	theta := ppath.AngleBetween(n0, n1) / 2.0 // half the angle between normals
	d := hw / math32.Cos(theta)               // half the miter length
	limit := math32.Max(j.Limit, 1.001)       // otherwise nearly linear joins will also get clipped
	clip := !math32.IsNaN(limit) && limit*halfWidth < math32.Abs(d)
	if clip && j.GapJoiner != nil {
		j.GapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	mid := pivot.Add(n0.Add(n1).Normal().MulScalar(d))
	if clip {
		// miter-clip
		t := math32.Abs(limit * halfWidth / d)
		if cw { // bend to the right, ie. CW
			mid0 := lhs.Pos().Lerp(mid, t)
			mid1 := lEnd.Lerp(mid, t)
			lhs.LineTo(mid0.X, mid0.Y)
			lhs.LineTo(mid1.X, mid1.Y)
		} else {
			mid0 := rhs.Pos().Lerp(mid, t)
			mid1 := rEnd.Lerp(mid, t)
			rhs.LineTo(mid0.X, mid0.Y)
			rhs.LineTo(mid1.X, mid1.Y)
		}
	} else {
		if cw { // bend to the right, ie. CW
			lhs.LineTo(mid.X, mid.Y)
		} else {
			rhs.LineTo(mid.X, mid.Y)
		}
	}
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

func (j MiterJoiner) String() string {
	if j.GapJoiner == nil {
		return "MiterClip"
	}
	return "Miter"
}

// ArcsJoin connects two path elements by extending the ends
// of the paths as circle arcs until they meet.
// If this point is further than the limit, this will result
// in a bevel join (ArcsJoin) or they will meet at the limit (ArcsClipJoin).
var ArcsJoin Joiner = ArcsJoiner{BevelJoin, 4.0}
var ArcsClipJoin Joiner = ArcsJoiner{nil, 4.0}

// ArcsJoiner is an arcs joiner.
type ArcsJoiner struct {
	GapJoiner Joiner
	Limit     float32
}

func closestArcIntersection(c math32.Vector2, cw bool, pivot, i0, i1 math32.Vector2) math32.Vector2 {
	thetaPivot := ppath.Angle(pivot.Sub(c))
	dtheta0 := ppath.Angle(i0.Sub(c)) - thetaPivot
	dtheta1 := ppath.Angle(i1.Sub(c)) - thetaPivot
	if cw { // arc runs clockwise, so look the other way around
		dtheta0 = -dtheta0
		dtheta1 = -dtheta1
	}
	if ppath.AngleNorm(dtheta1) < ppath.AngleNorm(dtheta0) {
		return i1
	}
	return i0
}

func (j ArcsJoiner) Join(rhs, lhs *ppath.Path, halfWidth float32, pivot, n0, n1 math32.Vector2, r0, r1 float32) {
	if ppath.EqualPoint(n0, n1.Negate()) {
		BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	} else if math32.IsNaN(r0) && math32.IsNaN(r1) {
		MiterJoiner(j).Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}
	limit := math32.Max(j.Limit, 1.001) // 1.001 so that nearly linear joins will not get clipped

	cw := 0.0 <= n0.Rot90CW().Dot(n1)
	hw := halfWidth
	if cw {
		hw = -hw // used to calculate |R|, when running CW then n0 and n1 point the other way, so the sign of r0 and r1 is negated
	}

	// r is the radius of the original curve, R the radius of the stroke curve, c are the centers of the circles
	c0 := pivot.Add(n0.Normal().MulScalar(-r0))
	c1 := pivot.Add(n1.Normal().MulScalar(-r1))
	R0, R1 := math32.Abs(r0+hw), math32.Abs(r1+hw)

	// TODO: can simplify if intersection returns angles too?
	var i0, i1 math32.Vector2
	var ok bool
	if math32.IsNaN(r0) {
		line := pivot.Add(n0)
		if cw {
			line = pivot.Sub(n0)
		}
		i0, i1, ok = intersect.IntersectionRayCircle(line, line.Add(n0.Rot90CCW()), c1, R1)
	} else if math32.IsNaN(r1) {
		line := pivot.Add(n1)
		if cw {
			line = pivot.Sub(n1)
		}
		i0, i1, ok = intersect.IntersectionRayCircle(line, line.Add(n1.Rot90CCW()), c0, R0)
	} else {
		i0, i1, ok = intersect.IntersectionCircleCircle(c0, R0, c1, R1)
	}
	if !ok {
		// no intersection
		BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}

	// find the closest intersection when following the arc (using either arc r0 or r1 with center c0 or c1 respectively)
	var mid math32.Vector2
	if !math32.IsNaN(r0) {
		mid = closestArcIntersection(c0, r0 < 0.0, pivot, i0, i1)
	} else {
		mid = closestArcIntersection(c1, 0.0 <= r1, pivot, i0, i1)
	}

	// check arc limit
	d := mid.Sub(pivot).Length()
	clip := !math32.IsNaN(limit) && limit*halfWidth < d
	if clip && j.GapJoiner != nil {
		j.GapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}

	mid2 := mid
	if clip {
		// arcs-clip
		start, end := pivot.Add(n0), pivot.Add(n1)
		if cw {
			start, end = pivot.Sub(n0), pivot.Sub(n1)
		}

		var clipMid, clipNormal math32.Vector2
		if !math32.IsNaN(r0) && !math32.IsNaN(r1) && (0.0 < r0) == (0.0 < r1) {
			// circle have opposite direction/sweep
			// NOTE: this may cause the bevel to be imperfectly oriented
			clipMid = mid.Sub(pivot).Normal().MulScalar(limit * halfWidth)
			clipNormal = clipMid.Rot90CCW()
		} else {
			// circle in between both stroke edges
			rMid := (r0 - r1) / 2.0
			if math32.IsNaN(r0) {
				rMid = -(r1 + hw) * 2.0
			} else if math32.IsNaN(r1) {
				rMid = (r0 + hw) * 2.0
			}

			sweep := 0.0 < rMid
			RMid := math32.Abs(rMid)
			cx, cy, a0, _ := ppath.EllipseToCenter(pivot.X, pivot.Y, RMid, RMid, 0.0, false, sweep, mid.X, mid.Y)
			cMid := math32.Vector2{cx, cy}
			dtheta := limit * halfWidth / rMid

			clipMid = ppath.EllipsePos(RMid, RMid, 0.0, cMid.X, cMid.Y, a0+dtheta)
			clipNormal = ellipseNormal(RMid, RMid, 0.0, sweep, a0+dtheta, 1.0)
		}

		if math32.IsNaN(r1) {
			i0, ok = intersect.IntersectionRayLine(clipMid, clipMid.Add(clipNormal), mid, end)
			if !ok {
				// not sure when this occurs
				BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
				return
			}
			mid2 = i0
		} else {
			i0, i1, ok = intersect.IntersectionRayCircle(clipMid, clipMid.Add(clipNormal), c1, R1)
			if !ok {
				// not sure when this occurs
				BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
				return
			}
			mid2 = closestArcIntersection(c1, 0.0 <= r1, pivot, i0, i1)
		}

		if math32.IsNaN(r0) {
			i0, ok = intersect.IntersectionRayLine(clipMid, clipMid.Add(clipNormal), start, mid)
			if !ok {
				// not sure when this occurs
				BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
				return
			}
			mid = i0
		} else {
			i0, i1, ok = intersect.IntersectionRayCircle(clipMid, clipMid.Add(clipNormal), c0, R0)
			if !ok {
				// not sure when this occurs
				BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
				return
			}
			mid = closestArcIntersection(c0, r0 < 0.0, pivot, i0, i1)
		}
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	if cw { // bend to the right, ie. CW
		rhs.LineTo(rEnd.X, rEnd.Y)
		if math32.IsNaN(r0) {
			lhs.LineTo(mid.X, mid.Y)
		} else {
			lhs.ArcTo(R0, R0, 0.0, false, 0.0 < r0, mid.X, mid.Y)
		}
		if clip {
			lhs.LineTo(mid2.X, mid2.Y)
		}
		if math32.IsNaN(r1) {
			lhs.LineTo(lEnd.X, lEnd.Y)
		} else {
			lhs.ArcTo(R1, R1, 0.0, false, 0.0 < r1, lEnd.X, lEnd.Y)
		}
	} else { // bend to the left, ie. CCW
		if math32.IsNaN(r0) {
			rhs.LineTo(mid.X, mid.Y)
		} else {
			rhs.ArcTo(R0, R0, 0.0, false, 0.0 < r0, mid.X, mid.Y)
		}
		if clip {
			rhs.LineTo(mid2.X, mid2.Y)
		}
		if math32.IsNaN(r1) {
			rhs.LineTo(rEnd.X, rEnd.Y)
		} else {
			rhs.ArcTo(R1, R1, 0.0, false, 0.0 < r1, rEnd.X, rEnd.Y)
		}
		lhs.LineTo(lEnd.X, lEnd.Y)
	}
}

func (j ArcsJoiner) String() string {
	if j.GapJoiner == nil {
		return "ArcsClip"
	}
	return "Arcs"
}

// optimizeClose removes a superfluous first line segment in-place
// of a subpath. If both the first and last segment are line segments
// and are colinear, move the start of the path forward one segment
func optimizeClose(p *ppath.Path) {
	if len(*p) == 0 || (*p)[len(*p)-1] != ppath.Close {
		return
	}

	// find last MoveTo
	end := math32.Vector2{}
	iMoveTo := len(*p)
	for 0 < iMoveTo {
		cmd := (*p)[iMoveTo-1]
		iMoveTo -= ppath.CmdLen(cmd)
		if cmd == ppath.MoveTo {
			end = math32.Vec2((*p)[iMoveTo+1], (*p)[iMoveTo+2])
			break
		}
	}

	if (*p)[iMoveTo] == ppath.MoveTo && (*p)[iMoveTo+ppath.CmdLen(ppath.MoveTo)] == ppath.LineTo && iMoveTo+ppath.CmdLen(ppath.MoveTo)+ppath.CmdLen(ppath.LineTo) < len(*p)-ppath.CmdLen(ppath.Close) {
		// replace Close + MoveTo + LineTo by Close + MoveTo if equidirectional
		// move Close and MoveTo forward along the path
		start := math32.Vec2((*p)[len(*p)-ppath.CmdLen(ppath.Close)-3], (*p)[len(*p)-ppath.CmdLen(ppath.Close)-2])
		nextEnd := math32.Vec2((*p)[iMoveTo+ppath.CmdLen(ppath.MoveTo)+ppath.CmdLen(ppath.LineTo)-3], (*p)[iMoveTo+ppath.CmdLen(ppath.MoveTo)+ppath.CmdLen(ppath.LineTo)-2])
		if ppath.Equal(ppath.AngleBetween(end.Sub(start), nextEnd.Sub(end)), 0.0) {
			// update Close
			(*p)[len(*p)-3] = nextEnd.X
			(*p)[len(*p)-2] = nextEnd.Y

			// update MoveTo
			(*p)[iMoveTo+1] = nextEnd.X
			(*p)[iMoveTo+2] = nextEnd.Y

			// remove LineTo
			*p = append((*p)[:iMoveTo+ppath.CmdLen(ppath.MoveTo)], (*p)[iMoveTo+ppath.CmdLen(ppath.MoveTo)+ppath.CmdLen(ppath.LineTo):]...)
		}
	}
}

func optimizeInnerBend(p ppath.Path, i int) {
	// i is the index of the line segment in the inner bend connecting both edges
	ai := i - ppath.CmdLen(p[i-1])
	if ai == 0 {
		return
	}
	if i >= len(p) {
		return
	}
	bi := i + ppath.CmdLen(p[i])

	a0 := math32.Vector2{p[ai-3], p[ai-2]}
	b0 := math32.Vector2{p[bi-3], p[bi-2]}
	if bi == len(p) {
		// inner bend is at the path's start
		bi = 4
	}

	// TODO: implement other segment combinations
	zs_ := [2]intersect.Intersection{}
	zs := zs_[:]
	if (p[ai] == ppath.LineTo || p[ai] == ppath.Close) && (p[bi] == ppath.LineTo || p[bi] == ppath.Close) {
		zs = intersect.IntersectionSegment(zs[:0], a0, p[ai:ai+4], b0, p[bi:bi+4])
		// TODO: check conditions for pathological cases
		if len(zs) == 1 && zs[0].T[0] != 0.0 && zs[0].T[0] != 1.0 && zs[0].T[1] != 0.0 && zs[0].T[1] != 1.0 {
			p[ai+1] = zs[0].X
			p[ai+2] = zs[0].Y
			if bi == 4 {
				// inner bend is at the path's start
				if p[i] == ppath.Close {
					if p[ai] == ppath.LineTo {
						p[ai] = ppath.Close
						p[ai+3] = ppath.Close
					} else {
						p = append(p, ppath.Close, zs[0].X, zs[1].Y, ppath.Close)
					}
				}
				p = p[:i]
				p[1] = zs[0].X
				p[2] = zs[0].Y
			} else {
				p = append(p[:i], p[bi:]...)
			}
		}
	}
}

type pathStrokeState struct {
	cmd    float32
	p0, p1 math32.Vector2 // position of start and end
	n0, n1 math32.Vector2 // normal of start and end (points right when walking the path)
	r0, r1 float32        // radius of start and end

	cp1, cp2                    math32.Vector2 // Béziers
	rx, ry, rot, theta0, theta1 float32        // arcs
	large, sweep                bool           // arcs
}

// offset returns the rhs and lhs paths from offsetting a path
// (must not have subpaths). It closes rhs and lhs when p is closed as well.
func offset(p ppath.Path, halfWidth float32, cr Capper, jr Joiner, strokeOpen bool, tolerance float32) (ppath.Path, ppath.Path) {
	// only non-empty paths are evaluated
	closed := false
	states := []pathStrokeState{}
	var start, end math32.Vector2
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case ppath.MoveTo:
			end = math32.Vector2{p[i+1], p[i+2]}
		case ppath.LineTo:
			end = math32.Vector2{p[i+1], p[i+2]}
			n := end.Sub(start).Rot90CW().Normal().MulScalar(halfWidth)
			states = append(states, pathStrokeState{
				cmd: ppath.LineTo,
				p0:  start,
				p1:  end,
				n0:  n,
				n1:  n,
				r0:  math32.NaN(),
				r1:  math32.NaN(),
			})
		case ppath.QuadTo, ppath.CubeTo:
			var cp1, cp2 math32.Vector2
			if cmd == ppath.QuadTo {
				cp := math32.Vector2{p[i+1], p[i+2]}
				end = math32.Vector2{p[i+3], p[i+4]}
				cp1, cp2 = ppath.QuadraticToCubicBezier(start, cp, end)
			} else {
				cp1 = math32.Vector2{p[i+1], p[i+2]}
				cp2 = math32.Vector2{p[i+3], p[i+4]}
				end = math32.Vector2{p[i+5], p[i+6]}
			}
			n0 := intersect.CubicBezierNormal(start, cp1, cp2, end, 0.0, halfWidth)
			n1 := intersect.CubicBezierNormal(start, cp1, cp2, end, 1.0, halfWidth)
			r0 := intersect.CubicBezierCurvatureRadius(start, cp1, cp2, end, 0.0)
			r1 := intersect.CubicBezierCurvatureRadius(start, cp1, cp2, end, 1.0)
			states = append(states, pathStrokeState{
				cmd: ppath.CubeTo,
				p0:  start,
				p1:  end,
				n0:  n0,
				n1:  n1,
				r0:  r0,
				r1:  r1,
				cp1: cp1,
				cp2: cp2,
			})
		case ppath.ArcTo:
			rx, ry, phi := p[i+1], p[i+2], p[i+3]
			large, sweep := ppath.ToArcFlags(p[i+4])
			end = math32.Vector2{p[i+5], p[i+6]}
			_, _, theta0, theta1 := ppath.EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			n0 := ellipseNormal(rx, ry, phi, sweep, theta0, halfWidth)
			n1 := ellipseNormal(rx, ry, phi, sweep, theta1, halfWidth)
			r0 := intersect.EllipseCurvatureRadius(rx, ry, sweep, theta0)
			r1 := intersect.EllipseCurvatureRadius(rx, ry, sweep, theta1)
			states = append(states, pathStrokeState{
				cmd:    ppath.ArcTo,
				p0:     start,
				p1:     end,
				n0:     n0,
				n1:     n1,
				r0:     r0,
				r1:     r1,
				rx:     rx,
				ry:     ry,
				rot:    phi * 180.0 / math32.Pi,
				theta0: theta0,
				theta1: theta1,
				large:  large,
				sweep:  sweep,
			})
		case ppath.Close:
			end = math32.Vector2{p[i+1], p[i+2]}
			if !ppath.Equal(start.X, end.X) || !ppath.Equal(start.Y, end.Y) {
				n := end.Sub(start).Rot90CW().Normal().MulScalar(halfWidth)
				states = append(states, pathStrokeState{
					cmd: ppath.LineTo,
					p0:  start,
					p1:  end,
					n0:  n,
					n1:  n,
					r0:  math32.NaN(),
					r1:  math32.NaN(),
				})
			}
			closed = true
		}
		start = end
		i += ppath.CmdLen(cmd)
	}
	if len(states) == 0 {
		return nil, nil
	}

	rhs, lhs := ppath.Path{}, ppath.Path{}
	rStart := states[0].p0.Add(states[0].n0)
	lStart := states[0].p0.Sub(states[0].n0)
	rhs.MoveTo(rStart.X, rStart.Y)
	lhs.MoveTo(lStart.X, lStart.Y)
	rhsJoinIndex, lhsJoinIndex := -1, -1
	for i, cur := range states {
		switch cur.cmd {
		case ppath.LineTo:
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			rhs.LineTo(rEnd.X, rEnd.Y)
			lhs.LineTo(lEnd.X, lEnd.Y)
		case ppath.CubeTo:
			rhs = rhs.Join(intersect.FlattenCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, halfWidth, tolerance))
			lhs = lhs.Join(intersect.FlattenCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, -halfWidth, tolerance))
		case ppath.ArcTo:
			rStart := cur.p0.Add(cur.n0)
			lStart := cur.p0.Sub(cur.n0)
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			dr := halfWidth
			if !cur.sweep { // bend to the right, ie. CW
				dr = -dr
			}

			rLambda := ppath.EllipseRadiiCorrection(rStart, cur.rx+dr, cur.ry+dr, cur.rot*math32.Pi/180.0, rEnd)
			lLambda := ppath.EllipseRadiiCorrection(lStart, cur.rx-dr, cur.ry-dr, cur.rot*math32.Pi/180.0, lEnd)
			if rLambda <= 1.0 && lLambda <= 1.0 {
				rLambda, lLambda = 1.0, 1.0
			}
			rhs.ArcTo(rLambda*(cur.rx+dr), rLambda*(cur.ry+dr), cur.rot, cur.large, cur.sweep, rEnd.X, rEnd.Y)
			lhs.ArcTo(lLambda*(cur.rx-dr), lLambda*(cur.ry-dr), cur.rot, cur.large, cur.sweep, lEnd.X, lEnd.Y)
		}

		// optimize inner bend
		if 0 < i {
			prev := states[i-1]
			cw := 0.0 <= prev.n1.Rot90CW().Dot(cur.n0)
			if cw && rhsJoinIndex != -1 {
				optimizeInnerBend(rhs, rhsJoinIndex)
			} else if !cw && lhsJoinIndex != -1 {
				optimizeInnerBend(lhs, lhsJoinIndex)
			}
		}
		rhsJoinIndex = -1
		lhsJoinIndex = -1

		// join the cur and next path segments
		if i+1 < len(states) || closed {
			next := states[0]
			if i+1 < len(states) {
				next = states[i+1]
			}
			if !ppath.EqualPoint(cur.n1, next.n0) {
				rhsJoinIndex = len(rhs)
				lhsJoinIndex = len(lhs)
				jr.Join(&rhs, &lhs, halfWidth, cur.p1, cur.n1, next.n0, cur.r1, next.r0)
			}
		}
	}

	if closed {
		rhs.Close()
		lhs.Close()

		// optimize inner bend
		if 1 < len(states) {
			cw := 0.0 <= states[len(states)-1].n1.Rot90CW().Dot(states[0].n0)
			if cw && rhsJoinIndex != -1 {
				optimizeInnerBend(rhs, rhsJoinIndex)
			} else if !cw && lhsJoinIndex != -1 {
				optimizeInnerBend(lhs, lhsJoinIndex)
			}
		}

		optimizeClose(&rhs)
		optimizeClose(&lhs)
	} else if strokeOpen {
		lhs = lhs.Reverse()
		cr.Cap(&rhs, halfWidth, states[len(states)-1].p1, states[len(states)-1].n1)
		rhs = rhs.Join(lhs)
		cr.Cap(&rhs, halfWidth, states[0].p0, states[0].n0.Negate())
		lhs = nil

		rhs.Close()
		optimizeClose(&rhs)
	}
	return rhs, lhs
}

// Offset offsets the path by w and returns a new path.
// A positive w will offset the path to the right-hand side, that is,
// it expands CCW oriented contours and contracts CW oriented contours.
// If you don't know the orientation you can use `Path.CCW` to find out,
// but if there may be self-intersection you should use `Path.Settle`
// to remove them and orient all filling contours CCW.
// The tolerance is the maximum deviation from the actual offset when
// flattening Béziers and optimizing the path.
func Offset(p ppath.Path, w float32, tolerance float32) ppath.Path {
	if ppath.Equal(w, 0.0) {
		return p
	}

	positive := 0.0 < w
	w = math32.Abs(w)

	q := ppath.Path{}
	for _, pi := range p.Split() {
		r := ppath.Path{}
		rhs, lhs := offset(pi, w, ButtCap, RoundJoin, false, tolerance)
		if rhs == nil {
			continue
		} else if positive {
			r = rhs
		} else {
			r = lhs
		}
		if pi.Closed() {
			if intersect.CCW(pi) {
				r = intersect.Settle(r, ppath.Positive)
			} else {
				r = intersect.Settle(r, ppath.Negative).Reverse()
			}
		}
		q = q.Append(r)
	}
	return q
}

// Markers returns an array of start, mid and end marker paths along
// the path at the coordinates between commands.
// Align will align the markers with the path direction so that
// the markers orient towards the path's left.
func Markers(p ppath.Path, first, mid, last ppath.Path, align bool) []ppath.Path {
	markers := []ppath.Path{}
	coordPos := p.Coords()
	coordDir := p.CoordDirections()
	for i := range coordPos {
		q := mid
		if i == 0 {
			q = first
		} else if i == len(coordPos)-1 {
			q = last
		}

		if q != nil {
			pos, dir := coordPos[i], coordDir[i]
			m := math32.Identity2().Translate(pos.X, pos.Y)
			if align {
				m = m.Rotate(ppath.Angle(dir))
			}
			markers = append(markers, q.Clone().Transform(m))
		}
	}
	return markers
}
