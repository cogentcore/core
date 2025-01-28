// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

//go:generate core generate

import (
	"cogentcore.org/core/math32"
)

// FillRules specifies the algorithm for which area is to be filled and which not,
// in particular when multiple subpaths overlap. The NonZero rule is the default
// and will fill any point that is being enclosed by an unequal number of paths
// winding clock-wise and counter clock-wise, otherwise it will not be filled.
// The EvenOdd rule will fill any point that is being enclosed by an uneven number
// of paths, whichever their direction. Positive fills only counter clock-wise
// oriented paths, while Negative fills only clock-wise oriented paths.
type FillRules int32 //enums:enum -transform kebab

const (
	NonZero FillRules = iota
	EvenOdd
	Positive
	Negative
)

func (fr FillRules) Fills(windings int) bool {
	switch fr {
	case NonZero:
		return windings != 0
	case EvenOdd:
		return windings%2 != 0
	case Positive:
		return 0 < windings
	case Negative:
		return windings < 0
	}
	return false
}

// todo: these need serious work:

// VectorEffects contains special effects for rendering
type VectorEffects int32 //enums:enum -trim-prefix VectorEffect -transform kebab

const (
	VectorEffectNone VectorEffects = iota

	// VectorEffectNonScalingStroke means that the stroke width is not affected by
	// transform properties
	VectorEffectNonScalingStroke
)

// Caps specifies the end-cap of a stroked line: stroke-linecap property in SVG
type Caps int32 //enums:enum -trim-prefix Cap -transform kebab

const (
	// CapButt indicates to draw no line caps; it draws a
	// line with the length of the specified length.
	CapButt Caps = iota

	// CapRound indicates to draw a semicircle on each line
	// end with a diameter of the stroke width.
	CapRound

	// CapSquare indicates to draw a rectangle on each line end
	// with a height of the stroke width and a width of half of the
	// stroke width.
	CapSquare
)

// Joins specifies the way stroked lines are joined together:
// stroke-linejoin property in SVG
type Joins int32 //enums:enum -trim-prefix Join -transform kebab

const (
	JoinMiter Joins = iota
	JoinMiterClip
	JoinRound
	JoinBevel
	JoinArcs
	// rasterx extension
	JoinArcsClip
)

// Dash patterns
var (
	Solid              = []float32{}
	Dotted             = []float32{1.0, 2.0}
	DenselyDotted      = []float32{1.0, 1.0}
	SparselyDotted     = []float32{1.0, 4.0}
	Dashed             = []float32{3.0, 3.0}
	DenselyDashed      = []float32{3.0, 1.0}
	SparselyDashed     = []float32{3.0, 6.0}
	Dashdotted         = []float32{3.0, 2.0, 1.0, 2.0}
	DenselyDashdotted  = []float32{3.0, 1.0, 1.0, 1.0}
	SparselyDashdotted = []float32{3.0, 4.0, 1.0, 4.0}
)

func ScaleDash(scale float32, offset float32, d []float32) (float32, []float32) {
	d2 := make([]float32, len(d))
	for i := range d {
		d2[i] = d[i] * scale
	}
	return offset * scale, d2
}

// NOTE: implementation inspired from github.com/golang/freetype/raster/stroke.go

// Stroke converts a path into a stroke of width w and returns a new path.
// It uses cr to cap the start and end of the path, and jr to join all path elements.
// If the path closes itself, it will use a join between the start and end instead
// of capping them. The tolerance is the maximum deviation from the original path
// when flattening Béziers and optimizing the stroke.
func (p Path) Stroke(w float32, cr Capper, jr Joiner, tolerance float32) Path {
	if cr == nil {
		cr = ButtCap
	}
	if jr == nil {
		jr = MiterJoin
	}
	q := Path{}
	halfWidth := math32.Abs(w) / 2.0
	for _, pi := range p.Split() {
		rhs, lhs := pi.offset(halfWidth, cr, jr, true, tolerance)
		if rhs == nil {
			continue
		} else if lhs == nil {
			// open path
			q = q.Append(rhs.Settle(Positive))
		} else {
			// closed path
			// inner path should go opposite direction to cancel the outer path
			if pi.CCW() {
				q = q.Append(rhs.Settle(Positive))
				q = q.Append(lhs.Settle(Positive).Reverse())
			} else {
				// outer first, then inner
				q = q.Append(lhs.Settle(Negative))
				q = q.Append(rhs.Settle(Negative).Reverse())
			}
		}
	}
	return q
}

func CapFromStyle(st Caps) Capper {
	switch st {
	case CapButt:
		return ButtCap
	case CapRound:
		return RoundCap
	case CapSquare:
		return SquareCap
	}
	return ButtCap
}

func JoinFromStyle(st Joins) Joiner {
	switch st {
	case JoinMiter:
		return MiterJoin
	case JoinMiterClip:
		return MiterClipJoin
	case JoinRound:
		return RoundJoin
	case JoinBevel:
		return BevelJoin
	case JoinArcs:
		return ArcsJoin
	case JoinArcsClip:
		return ArcsClipJoin
	}
	return MiterJoin
}

// Capper implements Cap, with rhs the path to append to,
// halfWidth the half width of the stroke, pivot the pivot point around
// which to construct a cap, and n0 the normal at the start of the path.
// The length of n0 is equal to the halfWidth.
type Capper interface {
	Cap(*Path, float32, math32.Vector2, math32.Vector2)
}

// RoundCap caps the start or end of a path by a round cap.
var RoundCap Capper = RoundCapper{}

// RoundCapper is a round capper.
type RoundCapper struct{}

// Cap adds a cap to path p of width 2*halfWidth,
// at a pivot point and initial normal direction of n0.
func (RoundCapper) Cap(p *Path, halfWidth float32, pivot, n0 math32.Vector2) {
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
func (ButtCapper) Cap(p *Path, halfWidth float32, pivot, n0 math32.Vector2) {
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
func (SquareCapper) Cap(p *Path, halfWidth float32, pivot, n0 math32.Vector2) {
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
	Join(*Path, *Path, float32, math32.Vector2, math32.Vector2, math32.Vector2, float32, float32)
}

// BevelJoin connects two path elements by a linear join.
var BevelJoin Joiner = BevelJoiner{}

// BevelJoiner is a bevel joiner.
type BevelJoiner struct{}

// Join adds a join to a right-hand-side and left-hand-side path,
// of width 2*halfWidth, around a pivot point with starting and
// ending normals of n0 and n1, and radius of curvatures of the
// previous and next segments.
func (BevelJoiner) Join(rhs, lhs *Path, halfWidth float32, pivot, n0, n1 math32.Vector2, r0, r1 float32) {
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

func (RoundJoiner) Join(rhs, lhs *Path, halfWidth float32, pivot, n0, n1 math32.Vector2, r0, r1 float32) {
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

func (j MiterJoiner) Join(rhs, lhs *Path, halfWidth float32, pivot, n0, n1 math32.Vector2, r0, r1 float32) {
	if EqualPoint(n0, n1.Negate()) {
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
	theta := AngleBetween(n0, n1) / 2.0 // half the angle between normals
	d := hw / math32.Cos(theta)         // half the miter length
	limit := math32.Max(j.Limit, 1.001) // otherwise nearly linear joins will also get clipped
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
	thetaPivot := Angle(pivot.Sub(c))
	dtheta0 := Angle(i0.Sub(c)) - thetaPivot
	dtheta1 := Angle(i1.Sub(c)) - thetaPivot
	if cw { // arc runs clockwise, so look the other way around
		dtheta0 = -dtheta0
		dtheta1 = -dtheta1
	}
	if angleNorm(dtheta1) < angleNorm(dtheta0) {
		return i1
	}
	return i0
}

func (j ArcsJoiner) Join(rhs, lhs *Path, halfWidth float32, pivot, n0, n1 math32.Vector2, r0, r1 float32) {
	if EqualPoint(n0, n1.Negate()) {
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
		i0, i1, ok = intersectionRayCircle(line, line.Add(n0.Rot90CCW()), c1, R1)
	} else if math32.IsNaN(r1) {
		line := pivot.Add(n1)
		if cw {
			line = pivot.Sub(n1)
		}
		i0, i1, ok = intersectionRayCircle(line, line.Add(n1.Rot90CCW()), c0, R0)
	} else {
		i0, i1, ok = intersectionCircleCircle(c0, R0, c1, R1)
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
			cx, cy, a0, _ := ellipseToCenter(pivot.X, pivot.Y, RMid, RMid, 0.0, false, sweep, mid.X, mid.Y)
			cMid := math32.Vector2{cx, cy}
			dtheta := limit * halfWidth / rMid

			clipMid = EllipsePos(RMid, RMid, 0.0, cMid.X, cMid.Y, a0+dtheta)
			clipNormal = ellipseNormal(RMid, RMid, 0.0, sweep, a0+dtheta, 1.0)
		}

		if math32.IsNaN(r1) {
			i0, ok = intersectionRayLine(clipMid, clipMid.Add(clipNormal), mid, end)
			if !ok {
				// not sure when this occurs
				BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
				return
			}
			mid2 = i0
		} else {
			i0, i1, ok = intersectionRayCircle(clipMid, clipMid.Add(clipNormal), c1, R1)
			if !ok {
				// not sure when this occurs
				BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
				return
			}
			mid2 = closestArcIntersection(c1, 0.0 <= r1, pivot, i0, i1)
		}

		if math32.IsNaN(r0) {
			i0, ok = intersectionRayLine(clipMid, clipMid.Add(clipNormal), start, mid)
			if !ok {
				// not sure when this occurs
				BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
				return
			}
			mid = i0
		} else {
			i0, i1, ok = intersectionRayCircle(clipMid, clipMid.Add(clipNormal), c0, R0)
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

func (p Path) optimizeInnerBend(i int) {
	// i is the index of the line segment in the inner bend connecting both edges
	ai := i - CmdLen(p[i-1])
	bi := i + CmdLen(p[i])
	if ai == 0 {
		return
	}

	a0 := math32.Vector2{p[ai-3], p[ai-2]}
	b0 := math32.Vector2{p[bi-3], p[bi-2]}
	if bi == len(p) {
		// inner bend is at the path's start
		bi = 4
	}

	// TODO: implement other segment combinations
	zs_ := [2]Intersection{}
	zs := zs_[:]
	if (p[ai] == LineTo || p[ai] == Close) && (p[bi] == LineTo || p[bi] == Close) {
		zs = intersectionSegment(zs[:0], a0, p[ai:ai+4], b0, p[bi:bi+4])
		// TODO: check conditions for pathological cases
		if len(zs) == 1 && zs[0].T[0] != 0.0 && zs[0].T[0] != 1.0 && zs[0].T[1] != 0.0 && zs[0].T[1] != 1.0 {
			p[ai+1] = zs[0].X
			p[ai+2] = zs[0].Y
			if bi == 4 {
				// inner bend is at the path's start
				if p[i] == Close {
					if p[ai] == LineTo {
						p[ai] = Close
						p[ai+3] = Close
					} else {
						p = append(p, Close, zs[0].X, zs[1].Y, Close)
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
func (p Path) offset(halfWidth float32, cr Capper, jr Joiner, strokeOpen bool, tolerance float32) (Path, Path) {
	// only non-empty paths are evaluated
	closed := false
	states := []pathStrokeState{}
	var start, end math32.Vector2
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			end = math32.Vector2{p[i+1], p[i+2]}
		case LineTo:
			end = math32.Vector2{p[i+1], p[i+2]}
			n := end.Sub(start).Rot90CW().Normal().MulScalar(halfWidth)
			states = append(states, pathStrokeState{
				cmd: LineTo,
				p0:  start,
				p1:  end,
				n0:  n,
				n1:  n,
				r0:  math32.NaN(),
				r1:  math32.NaN(),
			})
		case QuadTo, CubeTo:
			var cp1, cp2 math32.Vector2
			if cmd == QuadTo {
				cp := math32.Vector2{p[i+1], p[i+2]}
				end = math32.Vector2{p[i+3], p[i+4]}
				cp1, cp2 = quadraticToCubicBezier(start, cp, end)
			} else {
				cp1 = math32.Vector2{p[i+1], p[i+2]}
				cp2 = math32.Vector2{p[i+3], p[i+4]}
				end = math32.Vector2{p[i+5], p[i+6]}
			}
			n0 := cubicBezierNormal(start, cp1, cp2, end, 0.0, halfWidth)
			n1 := cubicBezierNormal(start, cp1, cp2, end, 1.0, halfWidth)
			r0 := cubicBezierCurvatureRadius(start, cp1, cp2, end, 0.0)
			r1 := cubicBezierCurvatureRadius(start, cp1, cp2, end, 1.0)
			states = append(states, pathStrokeState{
				cmd: CubeTo,
				p0:  start,
				p1:  end,
				n0:  n0,
				n1:  n1,
				r0:  r0,
				r1:  r1,
				cp1: cp1,
				cp2: cp2,
			})
		case ArcTo:
			rx, ry, phi := p[i+1], p[i+2], p[i+3]
			large, sweep := toArcFlags(p[i+4])
			end = math32.Vector2{p[i+5], p[i+6]}
			_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			n0 := ellipseNormal(rx, ry, phi, sweep, theta0, halfWidth)
			n1 := ellipseNormal(rx, ry, phi, sweep, theta1, halfWidth)
			r0 := ellipseCurvatureRadius(rx, ry, sweep, theta0)
			r1 := ellipseCurvatureRadius(rx, ry, sweep, theta1)
			states = append(states, pathStrokeState{
				cmd:    ArcTo,
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
		case Close:
			end = math32.Vector2{p[i+1], p[i+2]}
			if !Equal(start.X, end.X) || !Equal(start.Y, end.Y) {
				n := end.Sub(start).Rot90CW().Normal().MulScalar(halfWidth)
				states = append(states, pathStrokeState{
					cmd: LineTo,
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
		i += CmdLen(cmd)
	}
	if len(states) == 0 {
		return nil, nil
	}

	rhs, lhs := Path{}, Path{}
	rStart := states[0].p0.Add(states[0].n0)
	lStart := states[0].p0.Sub(states[0].n0)
	rhs.MoveTo(rStart.X, rStart.Y)
	lhs.MoveTo(lStart.X, lStart.Y)
	rhsJoinIndex, lhsJoinIndex := -1, -1
	for i, cur := range states {
		switch cur.cmd {
		case LineTo:
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			rhs.LineTo(rEnd.X, rEnd.Y)
			lhs.LineTo(lEnd.X, lEnd.Y)
		case CubeTo:
			rhs = rhs.Join(strokeCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, halfWidth, tolerance))
			lhs = lhs.Join(strokeCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, -halfWidth, tolerance))
		case ArcTo:
			rStart := cur.p0.Add(cur.n0)
			lStart := cur.p0.Sub(cur.n0)
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			dr := halfWidth
			if !cur.sweep { // bend to the right, ie. CW
				dr = -dr
			}

			rLambda := ellipseRadiiCorrection(rStart, cur.rx+dr, cur.ry+dr, cur.rot*math32.Pi/180.0, rEnd)
			lLambda := ellipseRadiiCorrection(lStart, cur.rx-dr, cur.ry-dr, cur.rot*math32.Pi/180.0, lEnd)
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
				rhs.optimizeInnerBend(rhsJoinIndex)
			} else if !cw && lhsJoinIndex != -1 {
				lhs.optimizeInnerBend(lhsJoinIndex)
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
			if !EqualPoint(cur.n1, next.n0) {
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
				rhs.optimizeInnerBend(rhsJoinIndex)
			} else if !cw && lhsJoinIndex != -1 {
				lhs.optimizeInnerBend(lhsJoinIndex)
			}
		}

		rhs.optimizeClose()
		lhs.optimizeClose()
	} else if strokeOpen {
		lhs = lhs.Reverse()
		cr.Cap(&rhs, halfWidth, states[len(states)-1].p1, states[len(states)-1].n1)
		rhs = rhs.Join(lhs)
		cr.Cap(&rhs, halfWidth, states[0].p0, states[0].n0.Negate())
		lhs = nil

		rhs.Close()
		rhs.optimizeClose()
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
func (p Path) Offset(w float32, tolerance float32) Path {
	if Equal(w, 0.0) {
		return p
	}

	positive := 0.0 < w
	w = math32.Abs(w)

	q := Path{}
	for _, pi := range p.Split() {
		r := Path{}
		rhs, lhs := pi.offset(w, ButtCap, RoundJoin, false, tolerance)
		if rhs == nil {
			continue
		} else if positive {
			r = rhs
		} else {
			r = lhs
		}
		if pi.Closed() {
			if pi.CCW() {
				r = r.Settle(Positive)
			} else {
				r = r.Settle(Negative).Reverse()
			}
		}
		q = q.Append(r)
	}
	return q
}
