// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package raster

import (
	"cogentcore.org/core/math32"
	"golang.org/x/image/math/fixed"
)

// Filler is a filler that implements [Raster].
type Filler struct {
	Scanner
	A     fixed.Point26_6
	First fixed.Point26_6
}

// NewFiller returns a Filler ptr with default values.
// A Filler in addition to rasterizing lines like a Scann,
// can also rasterize quadratic and cubic bezier curves.
// If Scanner is nil default scanner ScannerGV is used
func NewFiller(width, height int, scanner Scanner) *Filler {
	r := new(Filler)
	r.Scanner = scanner
	r.SetBounds(width, height)
	r.SetWinding(true)
	return r
}

// Start starts a new path at the given point.
func (r *Filler) Start(a fixed.Point26_6) {
	r.A = a
	r.First = a
	r.Scanner.Start(a)
}

// Stop sends a path at the given point.
func (r *Filler) Stop(isClosed bool) {
	if r.First != r.A {
		r.Line(r.First)
	}
}

// QuadBezier adds a quadratic segment to the current curve.
func (r *Filler) QuadBezier(b, c fixed.Point26_6) {
	r.QuadBezierF(r, b, c)
}

// QuadTo flattens the quadratic Bezier curve into lines through the LineTo func
// This functions is adapted from the version found in
// golang.org/x/image/vector
func QuadTo(ax, ay, bx, by, cx, cy float32, LineTo func(dx, dy float32)) {
	devsq := DevSquared(ax, ay, bx, by, cx, cy)
	if devsq >= 0.333 {
		const tol = 3
		n := 1 + int(math32.Sqrt(math32.Sqrt(tol*float32(devsq))))
		t, nInv := float32(0), 1/float32(n)
		for i := 0; i < n-1; i++ {
			t += nInv

			mt := 1 - t
			t1 := mt * mt
			t2 := mt * t * 2
			t3 := t * t
			LineTo(
				ax*t1+bx*t2+cx*t3,
				ay*t1+by*t2+cy*t3)
		}
	}
	LineTo(cx, cy)
}

// CubeTo flattens the cubic Bezier curve into lines through the LineTo func
// This functions is adapted from the version found in
// golang.org/x/image/vector
func CubeTo(ax, ay, bx, by, cx, cy, dx, dy float32, LineTo func(ex, ey float32)) {
	devsq := DevSquared(ax, ay, bx, by, dx, dy)
	if devsqAlt := DevSquared(ax, ay, cx, cy, dx, dy); devsq < devsqAlt {
		devsq = devsqAlt
	}
	if devsq >= 0.333 {
		const tol = 3
		n := 1 + int(math32.Sqrt(math32.Sqrt(tol*float32(devsq))))
		t, nInv := float32(0), 1/float32(n)
		for i := 0; i < n-1; i++ {
			t += nInv

			tsq := t * t
			mt := 1 - t
			mtsq := mt * mt
			t1 := mtsq * mt
			t2 := mtsq * t * 3
			t3 := mt * tsq * 3
			t4 := tsq * t
			LineTo(
				ax*t1+bx*t2+cx*t3+dx*t4,
				ay*t1+by*t2+cy*t3+dy*t4)
		}
	}
	LineTo(dx, dy)
}

// DevSquared returns a measure of how curvy the sequence (ax, ay) to (bx, by)
// to (cx, cy) is. It determines how many line segments will approximate a
// Bézier curve segment. This functions is copied from the version found in
// golang.org/x/image/vector as are the below comments.
//
// http://lists.nongnu.org/archive/html/freetype-devel/2016-08/msg00080.html
// gives the rationale for this evenly spaced heuristic instead of a recursive
// de Casteljau approach:
//
// The reason for the subdivision by n is that I expect the "flatness"
// computation to be semi-expensive (it's done once rather than on each
// potential subdivision) and also because you'll often get fewer subdivisions.
// Taking a circular arc as a simplifying assumption (ie a spherical cow),
// where I get n, a recursive approach would get 2^⌈lg n⌉, which, if I haven't
// made any horrible mistakes, is expected to be 33% more in the limit.
func DevSquared(ax, ay, bx, by, cx, cy float32) float32 {
	devx := ax - 2*bx + cx
	devy := ay - 2*by + cy
	return devx*devx + devy*devy
}

// QuadBezierF adds a quadratic segment to the sgm Rasterizer.
func (r *Filler) QuadBezierF(sgm Raster, b, c fixed.Point26_6) {
	// check for degenerate bezier
	if r.A == b || b == c {
		sgm.Line(c)
		return
	}
	sgm.JoinF()
	QuadTo(float32(r.A.X), float32(r.A.Y), // Pts are x64, but does not matter.
		float32(b.X), float32(b.Y),
		float32(c.X), float32(c.Y),
		func(dx, dy float32) {
			sgm.LineF(fixed.Point26_6{X: fixed.Int26_6(dx), Y: fixed.Int26_6(dy)})
		})

}

// CubeBezier adds a cubic bezier to the curve
func (r *Filler) CubeBezier(b, c, d fixed.Point26_6) {
	r.CubeBezierF(r, b, c, d)
}

// JoinF is a no-op for a filling rasterizer. This is used in stroking and dashed
// stroking
func (r *Filler) JoinF() {

}

// Line for a filling rasterizer is just the line call in scan
func (r *Filler) Line(b fixed.Point26_6) {
	r.LineF(b)
}

// LineF for a filling rasterizer is just the line call in scan
func (r *Filler) LineF(b fixed.Point26_6) {
	r.Scanner.Line(b)
	r.A = b
}

// CubeBezierF adds a cubic bezier to the curve. sending the line calls the the
// sgm Rasterizer
func (r *Filler) CubeBezierF(sgm Raster, b, c, d fixed.Point26_6) {
	if (r.A == b && c == d) || (r.A == b && b == c) || (c == b && d == c) {
		sgm.Line(d)
		return
	}
	sgm.JoinF()
	CubeTo(float32(r.A.X), float32(r.A.Y),
		float32(b.X), float32(b.Y),
		float32(c.X), float32(c.Y),
		float32(d.X), float32(d.Y),
		func(ex, ey float32) {
			sgm.LineF(fixed.Point26_6{X: fixed.Int26_6(ex), Y: fixed.Int26_6(ey)})
		})
}

// Clear resets the filler
func (r *Filler) Clear() {
	r.A = fixed.Point26_6{}
	r.First = r.A
	r.Scanner.Clear()
}

// SetBounds sets the maximum width and height of the rasterized image and
// calls Clear. The width and height are in pixels, not fixed.Int26_6 units.
func (r *Filler) SetBounds(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	r.Scanner.SetBounds(width, height)
	r.Clear()
}
