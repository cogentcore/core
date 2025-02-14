// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package rasterx

import (
	"golang.org/x/image/math/fixed"
)

// Dasher struct extends the Stroker and can draw
// dashed lines with end capping
type Dasher struct {
	Stroker
	Dashes         []fixed.Int26_6
	DashPlace      int
	FirstDashIsGap bool
	DashIsGap      bool
	DeltaDash      fixed.Int26_6
	DashOffset     fixed.Int26_6

	// Sgm allows us to switch between dashing
	// and non-dashing rasterizers in the SetStroke function.
	Sgm Raster
}

// NewDasher returns a Dasher ptr with default values.
// A Dasher has all of the capabilities of a Stroker, Filler, and Scanner, plus the ability
// to stroke curves with solid lines. Use SetStroke to configure with non-default
// values.
func NewDasher(width, height int, scanner Scanner) *Dasher {
	r := new(Dasher)
	r.Scanner = scanner
	r.SetBounds(width, height)
	r.SetWinding(true)
	r.SetStroke(1*64, 4*64, ButtCap, nil, FlatGap, MiterClip, nil, 0)
	r.Sgm = &r.Stroker
	return r
}

// JoinF overides stroker JoinF during dashed stroking, because we need to slightly modify
// the the call as below to handle the case of the join being in a dash gap.
func (r *Dasher) JoinF() {
	if len(r.Dashes) == 0 || !r.InStroke || !r.DashIsGap {
		r.Stroker.JoinF()
	}
}

// Start starts a dashed line
func (r *Dasher) Start(a fixed.Point26_6) {
	// Advance dashPlace to the dashOffset start point and set deltaDash
	if len(r.Dashes) > 0 {
		r.DeltaDash = r.DashOffset
		r.DashIsGap = false
		r.DashPlace = 0
		for r.DeltaDash > r.Dashes[r.DashPlace] {
			r.DeltaDash -= r.Dashes[r.DashPlace]
			r.DashIsGap = !r.DashIsGap
			r.DashPlace++
			if r.DashPlace == len(r.Dashes) {
				r.DashPlace = 0
			}
		}
		r.FirstDashIsGap = r.DashIsGap
	}
	r.Stroker.Start(a)
}

// LineF overides stroker LineF to modify the the call as below
// while performing the join in a dashed stroke.
func (r *Dasher) LineF(b fixed.Point26_6) {
	var bnorm fixed.Point26_6
	a := r.A // Copy local a since r.a is going to change during stroke operation
	ba := b.Sub(a)
	segLen := Length(ba)
	var nlt fixed.Int26_6
	if b == r.LeadPoint.P { // End of segment
		bnorm = r.LeadPoint.TNorm // Use more accurate leadPoint tangent
	} else {
		bnorm = TurnPort90(ToLength(b.Sub(a), r.U)) // Intra segment normal
	}
	for segLen+r.DeltaDash > r.Dashes[r.DashPlace] {
		nl := r.Dashes[r.DashPlace] - r.DeltaDash
		nlt += nl
		r.DashLineStrokeBit(a.Add(ToLength(ba, nlt)), bnorm, false)
		r.DashIsGap = !r.DashIsGap
		segLen -= nl
		r.DeltaDash = 0
		r.DashPlace++
		if r.DashPlace == len(r.Dashes) {
			r.DashPlace = 0
		}
	}
	r.DeltaDash += segLen
	r.DashLineStrokeBit(b, bnorm, true)
}

// SetStroke set the parameters for stroking a line. width is the width of the line, miterlimit is the miter cutoff
// value for miter, arc, miterclip and arcClip joinModes. CapL and CapT are the capping functions for leading and trailing
// line ends. If one is nil, the other function is used at both ends. gp is the gap function that determines how a
// gap on the convex side of two lines joining is filled. jm is the JoinMode for curve segments. Dashes is the values for
// the dash pattern. Pass in nil or an empty slice for no dashes. dashoffset is the starting offset into the dash array.
func (r *Dasher) SetStroke(width, miterLimit fixed.Int26_6, capL, capT CapFunc, gp GapFunc, jm JoinMode, dashes []float32, dashOffset float32) {
	r.Stroker.SetStroke(width, miterLimit, capL, capT, gp, jm)

	r.Dashes = r.Dashes[:0] // clear the dash array
	if len(dashes) == 0 {
		r.Sgm = &r.Stroker // This is just plain stroking
		return
	}
	// Dashed Stroke
	// Convert the float dash array and offset to fixed point and attach to the Filler
	oneIsPos := false // Check to see if at least one dash is > 0
	for _, v := range dashes {
		fv := fixed.Int26_6(v * 64)
		if fv <= 0 { // Negatives are considered 0s.
			fv = 0
		} else {
			oneIsPos = true
		}
		r.Dashes = append(r.Dashes, fv)
	}
	if !oneIsPos {
		r.Dashes = r.Dashes[:0]
		r.Sgm = &r.Stroker // This is just plain stroking
		return
	}
	r.DashOffset = fixed.Int26_6(dashOffset * 64)
	r.Sgm = r // Use the full dasher
}

// Stop terminates a dashed line
func (r *Dasher) Stop(isClosed bool) {
	if len(r.Dashes) == 0 {
		r.Stroker.Stop(isClosed)
		return
	}
	if !r.InStroke {
		return
	}
	if isClosed && r.A != r.FirstP.P {
		r.LineSeg(r.Sgm, r.FirstP.P)
	}
	ra := &r.Filler
	if isClosed && !r.FirstDashIsGap && !r.DashIsGap { // closed connect w/o caps
		a := r.A
		r.FirstP.TNorm = r.LeadPoint.TNorm
		r.FirstP.RT = r.LeadPoint.RT
		r.FirstP.TTan = r.LeadPoint.TTan
		ra.Start(r.FirstP.P.Sub(r.FirstP.TNorm))
		ra.Line(a.Sub(r.Ln))
		ra.Start(a.Add(r.Ln))
		ra.Line(r.FirstP.P.Add(r.FirstP.TNorm))
		r.Joiner(r.FirstP)
		r.FirstP.BlackWidowMark(ra)
	} else { // Cap open ends
		if !r.DashIsGap {
			r.CapL(ra, r.LeadPoint.P, r.LeadPoint.TNorm)
		}
		if !r.FirstDashIsGap {
			r.CapT(ra, r.FirstP.P, Invert(r.FirstP.LNorm))
		}
	}
	r.InStroke = false
}

// DashLineStrokeBit is a helper function that reduces code redundancy in the
// LineF function.
func (r *Dasher) DashLineStrokeBit(b, bnorm fixed.Point26_6, dontClose bool) {
	if !r.DashIsGap { // Moving from dash to gap
		a := r.A
		ra := &r.Filler
		ra.Start(b.Sub(bnorm))
		ra.Line(a.Sub(r.Ln))
		ra.Start(a.Add(r.Ln))
		ra.Line(b.Add(bnorm))
		if !dontClose {
			r.CapL(ra, b, bnorm)
		}
	} else { // Moving from gap to dash
		if !dontClose {
			ra := &r.Filler
			r.CapT(ra, b, Invert(bnorm))
		}
	}
	r.A = b
	r.Ln = bnorm
}

// Line for Dasher is here to pass the dasher sgm to LineP
func (r *Dasher) Line(b fixed.Point26_6) {
	r.LineSeg(r.Sgm, b)
}

// QuadBezier for dashing
func (r *Dasher) QuadBezier(b, c fixed.Point26_6) {
	r.QuadBezierf(r.Sgm, b, c)
}

// CubeBezier starts a stroked cubic bezier.
// It is a low level function exposed for the purposes of callbacks
// and debugging.
func (r *Dasher) CubeBezier(b, c, d fixed.Point26_6) {
	r.CubeBezierf(r.Sgm, b, c, d)
}
