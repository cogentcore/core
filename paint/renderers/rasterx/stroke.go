// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package rasterx

import (
	"cogentcore.org/core/math32"
	"golang.org/x/image/math/fixed"
)

// Stroker does everything a [Filler] does, but
// also allows for stroking and dashed stroking in addition to
// filling
type Stroker struct {
	Filler

	// Trailing cap function
	CapT CapFunc

	// Leading cpa function
	CapL CapFunc

	// When gap appears between segments, this function is called
	JoinGap GapFunc

	// Tracks progress of the stroke
	FirstP C2Point

	// Tracks progress of the stroke
	TrailPoint C2Point

	// Tracks progress of the stroke
	LeadPoint C2Point

	// last normal of intra-seg connection.
	Ln fixed.Point26_6

	// U is the half-width of the stroke.
	U fixed.Int26_6

	MLimit fixed.Int26_6

	JoinMode JoinMode

	InStroke bool
}

// NewStroker returns a ptr to a Stroker with default values.
// A Stroker has all of the capabilities of a Filler and Scanner, plus the ability
// to stroke curves with solid lines. Use SetStroke to configure with non-default
// values.
func NewStroker(width, height int, scanner Scanner) *Stroker {
	r := new(Stroker)
	r.Scanner = scanner
	r.SetBounds(width, height)
	//Defaults for stroking
	r.SetWinding(true)
	r.U = 2 << 6
	r.MLimit = 4 << 6
	r.JoinMode = MiterClip
	r.JoinGap = RoundGap
	r.CapL = RoundCap
	r.CapT = RoundCap
	r.SetStroke(1<<6, 4<<6, ButtCap, nil, FlatGap, MiterClip)
	return r
}

// CapFunc defines a function that draws caps on the ends of lines
type CapFunc func(p Adder, a, eNorm fixed.Point26_6)

// GapFunc defines a function to bridge gaps when the miter limit is
// exceeded
type GapFunc func(p Adder, a, tNorm, lNorm fixed.Point26_6)

// C2Point represents a point that connects two stroke segments
// and holds the tangent, normal and radius of curvature
// of the trailing and leading segments in fixed point values.
type C2Point struct {
	P, TTan, LTan, TNorm, LNorm fixed.Point26_6
	RT, RL                      fixed.Int26_6
}

// JoinMode type to specify how segments join.
type JoinMode int32 //enums:enum

// JoinMode constants determine how stroke segments bridge the gap at a join
// ArcClip mode is like MiterClip applied to arcs, and is not part of the SVG2.0
// standard.
const (
	Arc JoinMode = iota
	ArcClip
	Miter
	MiterClip
	Bevel
	Round
)

const (
	// Number of cubic beziers to approx half a circle
	CubicsPerHalfCircle = 8

	// 1/4 in fixed point
	EpsilonFixed = fixed.Int26_6(16)

	// fixed point t parameterization shift factor;
	// (2^this)/64 is the max length of t for fixed.Int26_6
	TStrokeShift = 14
)

// SetStroke set the parameters for stroking a line. width is the width of the line, miterlimit is the miter cutoff
// value for miter, arc, miterclip and arcClip joinModes. CapL and CapT are the capping functions for leading and trailing
// line ends. If one is nil, the other function is used at both ends. If both are nil, both ends are ButtCapped.
// gp is the gap function that determines how a gap on the convex side of two joining lines is filled. jm is the JoinMode
// for curve segments.
func (r *Stroker) SetStroke(width, miterLimit fixed.Int26_6, capL, capT CapFunc, gp GapFunc, jm JoinMode) {
	r.U = width / 2
	r.CapL = capL
	r.CapT = capT
	r.JoinMode = jm
	r.JoinGap = gp
	r.MLimit = (r.U * miterLimit) >> 6

	if r.CapT == nil {
		if r.CapL == nil {
			r.CapT = ButtCap
		} else {
			r.CapT = r.CapL
		}
	}
	if r.CapL == nil {
		r.CapL = r.CapT
	}
	if gp == nil {
		if r.JoinMode == Round {
			r.JoinGap = RoundGap
		} else {
			r.JoinGap = FlatGap
		}
	}

}

// GapToCap is a utility that converts a CapFunc to GapFunc
func GapToCap(p Adder, a, eNorm fixed.Point26_6, gf GapFunc) {
	p.Start(a.Add(eNorm))
	gf(p, a, eNorm, Invert(eNorm))
	p.Line(a.Sub(eNorm))
}

var (
	// ButtCap caps lines with a straight line
	ButtCap CapFunc = func(p Adder, a, eNorm fixed.Point26_6) {
		p.Start(a.Add(eNorm))
		p.Line(a.Sub(eNorm))
	}

	// SquareCap caps lines with a square which is slightly longer than ButtCap
	SquareCap CapFunc = func(p Adder, a, eNorm fixed.Point26_6) {
		tpt := a.Add(TurnStarboard90(eNorm))
		p.Start(a.Add(eNorm))
		p.Line(tpt.Add(eNorm))
		p.Line(tpt.Sub(eNorm))
		p.Line(a.Sub(eNorm))
	}

	// RoundCap caps lines with a half-circle
	RoundCap CapFunc = func(p Adder, a, eNorm fixed.Point26_6) {
		GapToCap(p, a, eNorm, RoundGap)
	}

	// CubicCap caps lines with a cubic bezier
	CubicCap CapFunc = func(p Adder, a, eNorm fixed.Point26_6) {
		GapToCap(p, a, eNorm, CubicGap)
	}

	// QuadraticCap caps lines with a quadratic bezier
	QuadraticCap CapFunc = func(p Adder, a, eNorm fixed.Point26_6) {
		GapToCap(p, a, eNorm, QuadraticGap)
	}

	// Gap functions

	// FlatGap bridges miter-limit gaps with a straight line
	FlatGap GapFunc = func(p Adder, a, tNorm, lNorm fixed.Point26_6) {
		p.Line(a.Add(lNorm))
	}

	// RoundGap bridges miter-limit gaps with a circular arc
	RoundGap GapFunc = func(p Adder, a, tNorm, lNorm fixed.Point26_6) {
		StrokeArc(p, a, a.Add(tNorm), a.Add(lNorm), true, 0, 0, p.Line)
		p.Line(a.Add(lNorm)) // just to be sure line joins cleanly,
		// last pt in stoke arc may not be precisely s2
	}

	// CubicGap bridges miter-limit gaps with a cubic bezier
	CubicGap GapFunc = func(p Adder, a, tNorm, lNorm fixed.Point26_6) {
		p.CubeBezier(a.Add(tNorm).Add(TurnStarboard90(tNorm)), a.Add(lNorm).Add(TurnPort90(lNorm)), a.Add(lNorm))
	}

	// QuadraticGap bridges miter-limit gaps with a quadratic bezier
	QuadraticGap GapFunc = func(p Adder, a, tNorm, lNorm fixed.Point26_6) {
		c1, c2 := a.Add(tNorm).Add(TurnStarboard90(tNorm)), a.Add(lNorm).Add(TurnPort90(lNorm))
		cm := c1.Add(c2).Mul(fixed.Int26_6(1 << 5))
		p.QuadBezier(cm, a.Add(lNorm))
	}
)

// StrokeArc strokes a circular arc by approximation with bezier curves
func StrokeArc(p Adder, a, s1, s2 fixed.Point26_6, clockwise bool, trimStart,
	trimEnd fixed.Int26_6, firstPoint func(p fixed.Point26_6)) (ps1, ds1, ps2, ds2 fixed.Point26_6) {
	// Approximate the circular arc using a set of cubic bezier curves by the method of
	// L. Maisonobe, "Drawing an elliptical arc using polylines, quadratic
	// or cubic Bezier curves", 2003
	// https://www.spaceroots.org/documents/elllipse/elliptical-arc.pdf
	// The method was simplified for circles.
	theta1 := math32.Atan2(float32(s1.Y-a.Y), float32(s1.X-a.X))
	theta2 := math32.Atan2(float32(s2.Y-a.Y), float32(s2.X-a.X))
	if !clockwise {
		for theta1 < theta2 {
			theta1 += math32.Pi * 2
		}
	} else {
		for theta2 < theta1 {
			theta2 += math32.Pi * 2
		}
	}
	deltaTheta := theta2 - theta1
	if trimStart > 0 {
		ds := (deltaTheta * float32(trimStart)) / float32(1<<TStrokeShift)
		deltaTheta -= ds
		theta1 += ds
	}
	if trimEnd > 0 {
		ds := (deltaTheta * float32(trimEnd)) / float32(1<<TStrokeShift)
		deltaTheta -= ds
	}

	segs := int(math32.Abs(deltaTheta)/(math32.Pi/CubicsPerHalfCircle)) + 1
	dTheta := deltaTheta / float32(segs)
	tde := math32.Tan(dTheta / 2)
	alpha := fixed.Int26_6(math32.Sin(dTheta) * (math32.Sqrt(4+3*tde*tde) - 1) * (64.0 / 3.0)) // math32 is fun!
	r := float32(Length(s1.Sub(a)))                                                            // Note r is *64
	ldp := fixed.Point26_6{X: -fixed.Int26_6(r * math32.Sin(theta1)), Y: fixed.Int26_6(r * math32.Cos(theta1))}
	ds1 = ldp
	ps1 = fixed.Point26_6{X: a.X + ldp.Y, Y: a.Y - ldp.X}
	firstPoint(ps1)
	s1 = ps1
	for i := 1; i <= segs; i++ {
		eta := theta1 + dTheta*float32(i)
		ds2 = fixed.Point26_6{X: -fixed.Int26_6(r * math32.Sin(eta)), Y: fixed.Int26_6(r * math32.Cos(eta))}
		ps2 = fixed.Point26_6{X: a.X + ds2.Y, Y: a.Y - ds2.X} // Using deriviative to calc new pt, because circle
		p1 := s1.Add(ldp.Mul(alpha))
		p2 := ps2.Sub(ds2.Mul(alpha))
		p.CubeBezier(p1, p2, ps2)
		s1, ldp = ps2, ds2
	}
	return
}

// Joiner is called when two segments of a stroke are joined. it is exposed
// so that if can be wrapped to generate callbacks for the join points.
func (r *Stroker) Joiner(p C2Point) {
	if p.P.X < 0 || p.P.Y < 0 {
		return
	}
	crossProd := p.LNorm.X*p.TNorm.Y - p.TNorm.X*p.LNorm.Y
	// stroke bottom edge, with the reverse of p
	r.StrokeEdge(C2Point{P: p.P, TNorm: Invert(p.LNorm), LNorm: Invert(p.TNorm),
		TTan: Invert(p.LTan), LTan: Invert(p.TTan), RT: -p.RL, RL: -p.RT}, -crossProd)
	// stroke top edge
	r.StrokeEdge(p, crossProd)
}

// StrokeEdge reduces code redundancy in the Joiner function by 2x since it handles
// the top and bottom edges. This function encodes most of the logic of how to
// handle joins between the given C2Point point p, and the end of the line.
func (r *Stroker) StrokeEdge(p C2Point, crossProd fixed.Int26_6) {
	ra := &r.Filler
	s1, s2 := p.P.Add(p.TNorm), p.P.Add(p.LNorm) // Bevel points for top leading and trailing
	ra.Start(s1)
	if crossProd > -EpsilonFixed*EpsilonFixed { // Almost co-linear or convex
		ra.Line(s2)
		return // No need to fill any gaps
	}

	var ct, cl fixed.Point26_6 // Center of curvature trailing, leading
	var rt, rl fixed.Int26_6   // Radius of curvature trailing, leading

	// Adjust radiuses for stroke width
	if r.JoinMode == Arc || r.JoinMode == ArcClip {
		// Find centers of radius of curvature and adjust the radius to be drawn
		// by half the stroke width.
		if p.RT != 0 {
			if p.RT > 0 {
				ct = p.P.Add(ToLength(TurnPort90(p.TTan), p.RT))
				rt = p.RT - r.U
			} else {
				ct = p.P.Sub(ToLength(TurnPort90(p.TTan), -p.RT))
				rt = -p.RT + r.U
			}
			if rt < 0 {
				rt = 0
			}
		}
		if p.RL != 0 {
			if p.RL > 0 {
				cl = p.P.Add(ToLength(TurnPort90(p.LTan), p.RL))
				rl = p.RL - r.U
			} else {
				cl = p.P.Sub(ToLength(TurnPort90(p.LTan), -p.RL))
				rl = -p.RL + r.U
			}
			if rl < 0 {
				rl = 0
			}
		}
	}

	if r.JoinMode == MiterClip || r.JoinMode == Miter ||
		// Arc or ArcClip with 0 tRadCurve and 0 lRadCurve is treated the same as a
		// Miter or MiterClip join, resp.
		((r.JoinMode == Arc || r.JoinMode == ArcClip) && (rt == 0 && rl == 0)) {
		xt := CalcIntersect(s1.Sub(p.TTan), s1, s2, s2.Sub(p.LTan))
		xa := xt.Sub(p.P)
		if Length(xa) < r.MLimit { // within miter limit
			ra.Line(xt)
			ra.Line(s2)
			return
		}
		if r.JoinMode == MiterClip || (r.JoinMode == ArcClip) {
			//Projection of tNorm onto xa
			tProjP := xa.Mul(fixed.Int26_6((DotProd(xa, p.TNorm) << 6) / DotProd(xa, xa)))
			projLen := Length(tProjP)
			if r.MLimit > projLen { // the miter limit line is past the bevel point
				// t is the fraction shifted by tStrokeShift to scale the vectors from the bevel point
				// to the line intersection, so that they abbut the miter limit line.
				tiLength := Length(xa)
				sx1, sx2 := xt.Sub(s1), xt.Sub(s2)
				t := (r.MLimit - projLen) << TStrokeShift / (tiLength - projLen)
				tx := ToLength(sx1, t*Length(sx1)>>TStrokeShift)
				lx := ToLength(sx2, t*Length(sx2)>>TStrokeShift)
				vx := ToLength(xa, t*Length(xa)>>TStrokeShift)
				s1p, _, ap := s1.Add(tx), s2.Add(lx), p.P.Add(vx)
				gLen := Length(ap.Sub(s1p))
				ra.Line(s1p)
				r.JoinGap(ra, ap, ToLength(TurnPort90(p.TTan), gLen), ToLength(TurnPort90(p.LTan), gLen))
				ra.Line(s2)
				return
			}
		} // Fallthrough
	} else if r.JoinMode == Arc || r.JoinMode == ArcClip {
		// Test for cases of a bezier meeting line, an line meeting a bezier,
		// or a bezier meeting a bezier. (Line meeting line is handled above.)
		switch {
		case rt == 0: // rl != 0, because one must be non-zero as checked above
			xt, intersect := RayCircleIntersection(s1.Add(p.TTan), s1, cl, rl)
			if intersect {
				ray1, ray2 := xt.Sub(cl), s2.Sub(cl)
				clockwise := (ray1.X*ray2.Y > ray1.Y*ray2.X) // Sign of xprod
				if Length(p.P.Sub(xt)) < r.MLimit {          // within miter limit
					StrokeArc(ra, cl, xt, s2, clockwise, 0, 0, ra.Line)
					ra.Line(s2)
					return
				}
				// Not within miter limit line
				if r.JoinMode == ArcClip { // Scale bevel points towards xt, and call gap func
					xa := xt.Sub(p.P)
					//Projection of tNorm onto xa
					tProjP := xa.Mul(fixed.Int26_6((DotProd(xa, p.TNorm) << 6) / DotProd(xa, xa)))
					projLen := Length(tProjP)
					if r.MLimit > projLen { // the miter limit line is past the bevel point
						// t is the fraction shifted by tStrokeShift to scale the line or arc from the bevel point
						// to the line intersection, so that they abbut the miter limit line.
						sx1 := xt.Sub(s1) //, xt.Sub(s2)
						t := fixed.Int26_6(1<<TStrokeShift) - ((r.MLimit - projLen) << TStrokeShift / (Length(xa) - projLen))
						tx := ToLength(sx1, t*Length(sx1)>>TStrokeShift)
						s1p := xt.Sub(tx)
						ra.Line(s1p)
						sp1, ds1, ps2, _ := StrokeArc(ra, cl, xt, s2, clockwise, t, 0, ra.Start)
						ra.Start(s1p)
						// calc gap center as pt where -tnorm and line perp to midcoord
						midP := sp1.Add(s1p).Mul(fixed.Int26_6(1 << 5)) // midpoint
						midLine := TurnPort90(midP.Sub(sp1))
						if midLine.X*midLine.X+midLine.Y*midLine.Y > EpsilonFixed { // if midline is zero, CalcIntersect is invalid
							ap := CalcIntersect(s1p, s1p.Sub(p.TNorm), midLine.Add(midP), midP)
							gLen := Length(ap.Sub(s1p))
							if clockwise {
								ds1 = Invert(ds1)
							}
							r.JoinGap(ra, ap, ToLength(TurnPort90(p.TTan), gLen), ToLength(TurnStarboard90(ds1), gLen))
						}
						ra.Line(sp1)
						ra.Start(ps2)
						ra.Line(s2)
						return
					}
					//Bevel points not past miter limit: fallthrough
				}
			}
		case rl == 0: // rt != 0, because one must be non-zero as checked above
			xt, intersect := RayCircleIntersection(s2.Sub(p.LTan), s2, ct, rt)
			if intersect {
				ray1, ray2 := s1.Sub(ct), xt.Sub(ct)
				clockwise := ray1.X*ray2.Y > ray1.Y*ray2.X
				if Length(p.P.Sub(xt)) < r.MLimit { // within miter limit
					StrokeArc(ra, ct, s1, xt, clockwise, 0, 0, ra.Line)
					ra.Line(s2)
					return
				}
				// Not within miter limit line
				if r.JoinMode == ArcClip { // Scale bevel points towards xt, and call gap func
					xa := xt.Sub(p.P)
					//Projection of lNorm onto xa
					lProjP := xa.Mul(fixed.Int26_6((DotProd(xa, p.LNorm) << 6) / DotProd(xa, xa)))
					projLen := Length(lProjP)
					if r.MLimit > projLen { // The miter limit line is past the bevel point,
						// t is the fraction to scale the line or arc from the bevel point
						// to the line intersection, so that they abbut the miter limit line.
						sx2 := xt.Sub(s2)
						t := fixed.Int26_6(1<<TStrokeShift) - ((r.MLimit - projLen) << TStrokeShift / (Length(xa) - projLen))
						lx := ToLength(sx2, t*Length(sx2)>>TStrokeShift)
						s2p := xt.Sub(lx)
						_, _, ps2, ds2 := StrokeArc(ra, ct, s1, xt, clockwise, 0, t, ra.Line)
						// calc gap center as pt where -lnorm and line perp to midcoord
						midP := s2p.Add(ps2).Mul(fixed.Int26_6(1 << 5)) // midpoint
						midLine := TurnStarboard90(midP.Sub(ps2))
						if midLine.X*midLine.X+midLine.Y*midLine.Y > EpsilonFixed { // if midline is zero, CalcIntersect is invalid
							ap := CalcIntersect(midP, midLine.Add(midP), s2p, s2p.Sub(p.LNorm))
							gLen := Length(ap.Sub(ps2))
							if clockwise {
								ds2 = Invert(ds2)
							}
							r.JoinGap(ra, ap, ToLength(TurnStarboard90(ds2), gLen), ToLength(TurnPort90(p.LTan), gLen))
						}
						ra.Line(s2)
						return
					}
					//Bevel points not past miter limit: fallthrough
				}
			}
		default: // Both rl != 0 and rt != 0 as checked above
			xt1, xt2, gIntersect := CircleCircleIntersection(ct, cl, rt, rl)
			xt, intersect := ClosestPortside(s1, s2, xt1, xt2, gIntersect)
			if intersect {
				ray1, ray2 := s1.Sub(ct), xt.Sub(ct)
				clockwiseT := (ray1.X*ray2.Y > ray1.Y*ray2.X)
				ray1, ray2 = xt.Sub(cl), s2.Sub(cl)
				clockwiseL := ray1.X*ray2.Y > ray1.Y*ray2.X

				if Length(p.P.Sub(xt)) < r.MLimit { // within miter limit
					StrokeArc(ra, ct, s1, xt, clockwiseT, 0, 0, ra.Line)
					StrokeArc(ra, cl, xt, s2, clockwiseL, 0, 0, ra.Line)
					ra.Line(s2)
					return
				}

				if r.JoinMode == ArcClip { // Scale bevel points towards xt, and call gap func
					xa := xt.Sub(p.P)
					//Projection of lNorm onto xa
					lProjP := xa.Mul(fixed.Int26_6((DotProd(xa, p.LNorm) << 6) / DotProd(xa, xa)))
					projLen := Length(lProjP)
					if r.MLimit > projLen { // The miter limit line is past the bevel point,
						// t is the fraction to scale the line or arc from the bevel point
						// to the line intersection, so that they abbut the miter limit line.
						t := fixed.Int26_6(1<<TStrokeShift) - ((r.MLimit - projLen) << TStrokeShift / (Length(xa) - projLen))
						_, _, ps1, ds1 := StrokeArc(ra, ct, s1, xt, clockwiseT, 0, t, r.Filler.Line)
						ps2, ds2, fs2, _ := StrokeArc(ra, cl, xt, s2, clockwiseL, t, 0, ra.Start)
						midP := ps1.Add(ps2).Mul(fixed.Int26_6(1 << 5)) // midpoint
						midLine := TurnStarboard90(midP.Sub(ps1))
						ra.Start(ps1)
						if midLine.X*midLine.X+midLine.Y*midLine.Y > EpsilonFixed { // if midline is zero, CalcIntersect is invalid
							if clockwiseT {
								ds1 = Invert(ds1)
							}
							if clockwiseL {
								ds2 = Invert(ds2)
							}
							ap := CalcIntersect(midP, midLine.Add(midP), ps2, ps2.Sub(TurnStarboard90(ds2)))
							gLen := Length(ap.Sub(ps2))
							r.JoinGap(ra, ap, ToLength(TurnStarboard90(ds1), gLen), ToLength(TurnStarboard90(ds2), gLen))
						}
						ra.Line(ps2)
						ra.Start(fs2)
						ra.Line(s2)
						return
					}
				}
			}
			// fallthrough to final JoinGap
		}
	}
	r.JoinGap(ra, p.P, p.TNorm, p.LNorm)
	ra.Line(s2)
}

// Stop a stroked line. The line will close
// is isClosed is true. Otherwise end caps will
// be drawn at both ends.
func (r *Stroker) Stop(isClosed bool) {
	if !r.InStroke {
		return
	}
	rf := &r.Filler
	if isClosed {
		if r.FirstP.P != rf.A {
			r.Line(r.FirstP.P)
		}
		a := rf.A
		r.FirstP.TNorm = r.LeadPoint.TNorm
		r.FirstP.RT = r.LeadPoint.RT
		r.FirstP.TTan = r.LeadPoint.TTan

		rf.Start(r.FirstP.P.Sub(r.FirstP.TNorm))
		rf.Line(a.Sub(r.Ln))
		rf.Start(a.Add(r.Ln))
		rf.Line(r.FirstP.P.Add(r.FirstP.TNorm))
		r.Joiner(r.FirstP)
		r.FirstP.BlackWidowMark(rf)
	} else {
		a := rf.A
		rf.Start(r.LeadPoint.P.Sub(r.LeadPoint.TNorm))
		rf.Line(a.Sub(r.Ln))
		rf.Start(a.Add(r.Ln))
		rf.Line(r.LeadPoint.P.Add(r.LeadPoint.TNorm))
		r.CapL(rf, r.LeadPoint.P, r.LeadPoint.TNorm)
		r.CapT(rf, r.FirstP.P, Invert(r.FirstP.LNorm))
	}
	r.InStroke = false
}

// QuadBezier starts a stroked quadratic bezier.
func (r *Stroker) QuadBezier(b, c fixed.Point26_6) {
	r.QuadBezierf(r, b, c)
}

// CubeBezier starts a stroked quadratic bezier.
func (r *Stroker) CubeBezier(b, c, d fixed.Point26_6) {
	r.CubeBezierf(r, b, c, d)
}

// QuadBezierf calcs end curvature of beziers
func (r *Stroker) QuadBezierf(s Raster, b, c fixed.Point26_6) {
	r.TrailPoint = r.LeadPoint
	r.CalcEndCurvature(r.A, b, c, c, b, r.A, fixed.Int52_12(2<<12), DoCalcCurvature(s))
	r.QuadBezierF(s, b, c)
	r.A = c
}

// DoCalcCurvature determines if calculation of the end curvature is required
// depending on the raster type and JoinMode
func DoCalcCurvature(r Raster) bool {
	switch q := r.(type) {
	case *Filler:
		return false // never for filler
	case *Stroker:
		return (q.JoinMode == Arc || q.JoinMode == ArcClip)
	case *Dasher:
		return (q.JoinMode == Arc || q.JoinMode == ArcClip)
	default:
		return true // Better safe than sorry if another raster type is used
	}
}

func (r *Stroker) CubeBezierf(sgm Raster, b, c, d fixed.Point26_6) {
	if (r.A == b && c == d) || (r.A == b && b == c) || (c == b && d == c) {
		sgm.Line(d)
		return
	}
	r.TrailPoint = r.LeadPoint
	// Only calculate curvature if stroking or and using arc or arc-clip
	doCalcCurve := DoCalcCurvature(sgm)
	const dm = fixed.Int52_12((3 << 12) / 2)
	switch {
	// b != c, and c != d see above
	case r.A == b:
		r.CalcEndCurvature(b, c, d, d, c, b, dm, doCalcCurve)
	// b != a,  and b != c, see above
	case c == d:
		r.CalcEndCurvature(r.A, b, c, c, b, r.A, dm, doCalcCurve)
	default:
		r.CalcEndCurvature(r.A, b, c, d, c, b, dm, doCalcCurve)
	}
	r.CubeBezierF(sgm, b, c, d)
	r.A = d
}

// Line adds a line segment to the rasterizer
func (r *Stroker) Line(b fixed.Point26_6) {
	r.LineSeg(r, b)
}

// LineSeg is called by both the Stroker and Dasher
func (r *Stroker) LineSeg(sgm Raster, b fixed.Point26_6) {
	r.TrailPoint = r.LeadPoint
	ba := b.Sub(r.A)
	if ba.X == 0 && ba.Y == 0 { // a == b, line is degenerate
		if r.TrailPoint.TTan.X != 0 || r.TrailPoint.TTan.Y != 0 {
			ba = r.TrailPoint.TTan // Use last tangent for seg tangent
		} else { // Must be on top of last moveto; set ba to X axis unit vector
			ba = fixed.Point26_6{X: 1 << 6, Y: 0}
		}
	}
	bnorm := TurnPort90(ToLength(ba, r.U))
	r.TrailPoint.LTan = ba
	r.LeadPoint.TTan = ba
	r.TrailPoint.LNorm = bnorm
	r.LeadPoint.TNorm = bnorm
	r.TrailPoint.RL = 0.0
	r.LeadPoint.RT = 0.0
	r.TrailPoint.P = r.A
	r.LeadPoint.P = b

	sgm.JoinF()
	sgm.LineF(b)
	r.A = b
}

// LineF is for intra-curve lines. It is required for the Rasterizer interface
// so that if the line is being stroked or dash stroked, different actions can be
// taken.
func (r *Stroker) LineF(b fixed.Point26_6) {
	// b is either an intra-segment value, or
	// the end of the segment.
	var bnorm fixed.Point26_6
	a := r.A                // Hold a since r.a is going to change during stroke operation
	if b == r.LeadPoint.P { // End of segment
		bnorm = r.LeadPoint.TNorm // Use more accurate leadPoint tangent
	} else {
		bnorm = TurnPort90(ToLength(b.Sub(a), r.U)) // Intra segment normal
	}
	ra := &r.Filler
	ra.Start(b.Sub(bnorm))
	ra.Line(a.Sub(r.Ln))
	ra.Start(a.Add(r.Ln))
	ra.Line(b.Add(bnorm))
	r.A = b
	r.Ln = bnorm
}

// Start iniitates a stroked path
func (r *Stroker) Start(a fixed.Point26_6) {
	r.InStroke = false
	r.Filler.Start(a)
}

// CalcEndCurvature calculates the radius of curvature given the control points
// of a bezier curve.
// It is a low level function exposed for the purposes of callbacks
// and debugging.
func (r *Stroker) CalcEndCurvature(p0, p1, p2, q0, q1, q2 fixed.Point26_6,
	dm fixed.Int52_12, calcRadCuve bool) {
	r.TrailPoint.P = p0
	r.LeadPoint.P = q0
	r.TrailPoint.LTan = p1.Sub(p0)
	r.LeadPoint.TTan = q0.Sub(q1)
	r.TrailPoint.LNorm = TurnPort90(ToLength(r.TrailPoint.LTan, r.U))
	r.LeadPoint.TNorm = TurnPort90(ToLength(r.LeadPoint.TTan, r.U))
	if calcRadCuve {
		r.TrailPoint.RL = RadCurvature(p0, p1, p2, dm)
		r.LeadPoint.RT = -RadCurvature(q0, q1, q2, dm)
	} else {
		r.TrailPoint.RL = 0
		r.LeadPoint.RT = 0
	}
}

func (r *Stroker) JoinF() {
	if !r.InStroke {
		r.InStroke = true
		r.FirstP = r.TrailPoint
	} else {
		ra := &r.Filler
		tl := r.TrailPoint.P.Sub(r.TrailPoint.TNorm)
		th := r.TrailPoint.P.Add(r.TrailPoint.TNorm)
		if r.A != r.TrailPoint.P || r.Ln != r.TrailPoint.TNorm {
			a := r.A
			ra.Start(tl)
			ra.Line(a.Sub(r.Ln))
			ra.Start(a.Add(r.Ln))
			ra.Line(th)
		}
		r.Joiner(r.TrailPoint)
		r.TrailPoint.BlackWidowMark(ra)
	}
	r.Ln = r.TrailPoint.LNorm
	r.A = r.TrailPoint.P
}

// BlackWidowMark handles a gap in a stroke that can occur when a line end is too close
// to a segment to segment join point. Although it is only required in those cases,
// at this point, no code has been written to properly detect when it is needed,
// so for now it just draws by default.
func (jp *C2Point) BlackWidowMark(ra Adder) {
	xprod := jp.TNorm.X*jp.LNorm.Y - jp.TNorm.Y*jp.LNorm.X
	if xprod > EpsilonFixed*EpsilonFixed {
		tl := jp.P.Sub(jp.TNorm)
		ll := jp.P.Sub(jp.LNorm)
		ra.Start(jp.P)
		ra.Line(tl)
		ra.Line(ll)
		ra.Line(jp.P)
	} else if xprod < -EpsilonFixed*EpsilonFixed {
		th := jp.P.Add(jp.TNorm)
		lh := jp.P.Add(jp.LNorm)
		ra.Start(jp.P)
		ra.Line(lh)
		ra.Line(th)
		ra.Line(jp.P)
	}
}
