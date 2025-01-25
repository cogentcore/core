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

// MaxDx is the Maximum radians a cubic splice is allowed to span
// in ellipse parametric when approximating an off-axis ellipse.
const MaxDx float32 = math32.Pi / 8

// ToFixedP converts two floats to a fixed point.
func ToFixedP(x, y float32) (p fixed.Point26_6) {
	p.X = fixed.Int26_6(x * 64)
	p.Y = fixed.Int26_6(y * 64)
	return
}

// AddCircle adds a circle to the Adder p
func AddCircle(cx, cy, r float32, p Adder) {
	AddEllipse(cx, cy, r, r, 0, p)
}

// AddEllipse adds an elipse with center at cx,cy, with the indicated
// x and y radius, (rx, ry), rotated around the center by rot degrees.
func AddEllipse(cx, cy, rx, ry, rot float32, p Adder) {
	rotRads := rot * math32.Pi / 180
	pt := math32.Identity2().Translate(cx, cy).Rotate(rotRads).Translate(-cx, -cy).MulVector2AsPoint(math32.Vec2(cx+rx, cy))
	points := []float32{rx, ry, rot, 1.0, 0.0, pt.X, pt.Y}
	p.Start(pt.ToFixed())
	AddArc(points, cx, cy, pt.X, pt.Y, p)
	p.Stop(true)
}

// AddRect adds a rectangle of the indicated size, rotated
// around the center by rot degrees.
func AddRect(minX, minY, maxX, maxY, rot float32, p Adder) {
	rot *= math32.Pi / 180
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	m := math32.Identity2().Translate(cx, cy).Rotate(rot).Translate(-cx, -cy)
	q := &MatrixAdder{M: m, Adder: p}
	q.Start(ToFixedP(minX, minY))
	q.Line(ToFixedP(maxX, minY))
	q.Line(ToFixedP(maxX, maxY))
	q.Line(ToFixedP(minX, maxY))
	q.Stop(true)
}

// AddRoundRect adds a rectangle of the indicated size, rotated
// around the center by rot degrees with rounded corners of radius
// rx in the x axis and ry in the y axis. gf specifes the shape of the
// filleting function. Valid values are RoundGap, QuadraticGap, CubicGap,
// FlatGap, or nil which defaults to a flat gap.
func AddRoundRect(minX, minY, maxX, maxY, rx, ry, rot float32, gf GapFunc, p Adder) {
	if rx <= 0 || ry <= 0 {
		AddRect(minX, minY, maxX, maxY, rot, p)
		return
	}
	rot *= math32.Pi / 180
	if gf == nil {
		gf = FlatGap
	}
	w := maxX - minX
	if w < rx*2 {
		rx = w / 2
	}
	h := maxY - minY
	if h < ry*2 {
		ry = h / 2
	}
	stretch := rx / ry
	midY := minY + h/2
	m := math32.Identity2().Translate(minX+w/2, midY).Rotate(rot).Scale(1, 1/stretch).Translate(-minX-w/2, -minY-h/2)
	maxY = midY + h/2*stretch
	minY = midY - h/2*stretch

	q := &MatrixAdder{M: m, Adder: p}

	q.Start(ToFixedP(minX+rx, minY))
	q.Line(ToFixedP(maxX-rx, minY))
	gf(q, ToFixedP(maxX-rx, minY+rx), ToFixedP(0, -rx), ToFixedP(rx, 0))
	q.Line(ToFixedP(maxX, maxY-rx))
	gf(q, ToFixedP(maxX-rx, maxY-rx), ToFixedP(rx, 0), ToFixedP(0, rx))
	q.Line(ToFixedP(minX+rx, maxY))
	gf(q, ToFixedP(minX+rx, maxY-rx), ToFixedP(0, rx), ToFixedP(-rx, 0))
	q.Line(ToFixedP(minX, minY+rx))
	gf(q, ToFixedP(minX+rx, minY+rx), ToFixedP(-rx, 0), ToFixedP(0, -rx))
	q.Stop(true)
}

// AddArc adds an arc to the adder p
func AddArc(points []float32, cx, cy, px, py float32, p Adder) (lx, ly float32) {
	rotX := points[2] * math32.Pi / 180 // Convert degress to radians
	largeArc := points[3] != 0
	sweep := points[4] != 0
	startAngle := math32.Atan2(py-cy, px-cx) - rotX
	endAngle := math32.Atan2(points[6]-cy, points[5]-cx) - rotX
	deltaTheta := endAngle - startAngle
	arcBig := math32.Abs(deltaTheta) > math32.Pi

	// Approximate ellipse using cubic bezeir splines
	etaStart := math32.Atan2(math32.Sin(startAngle)/points[1], math32.Cos(startAngle)/points[0])
	etaEnd := math32.Atan2(math32.Sin(endAngle)/points[1], math32.Cos(endAngle)/points[0])
	deltaEta := etaEnd - etaStart
	if (arcBig && !largeArc) || (!arcBig && largeArc) { // Go has no boolean XOR
		if deltaEta < 0 {
			deltaEta += math32.Pi * 2
		} else {
			deltaEta -= math32.Pi * 2
		}
	}
	// This check might be needed if the center point of the ellipse is
	// at the midpoint of the start and end lines.
	if deltaEta < 0 && sweep {
		deltaEta += math32.Pi * 2
	} else if deltaEta >= 0 && !sweep {
		deltaEta -= math32.Pi * 2
	}

	// Round up to determine number of cubic splines to approximate bezier curve
	segs := int(math32.Abs(deltaEta)/MaxDx) + 1
	dEta := deltaEta / float32(segs) // span of each segment
	// Approximate the ellipse using a set of cubic bezier curves by the method of
	// L. Maisonobe, "Drawing an elliptical arc using polylines, quadratic
	// or cubic Bezier curves", 2003
	// https://www.spaceroots.org/documents/elllipse/elliptical-arc.pdf
	tde := math32.Tan(dEta / 2)
	alpha := math32.Sin(dEta) * (math32.Sqrt(4+3*tde*tde) - 1) / 3 // math32 is fun!
	lx, ly = px, py
	sinTheta, cosTheta := math32.Sin(rotX), math32.Cos(rotX)
	ldx, ldy := EllipsePrime(points[0], points[1], sinTheta, cosTheta, etaStart, cx, cy)
	for i := 1; i <= segs; i++ {
		eta := etaStart + dEta*float32(i)
		var px, py float32
		if i == segs {
			px, py = points[5], points[6] // Just makes the end point exact; no roundoff error
		} else {
			px, py = EllipsePointAt(points[0], points[1], sinTheta, cosTheta, eta, cx, cy)
		}
		dx, dy := EllipsePrime(points[0], points[1], sinTheta, cosTheta, eta, cx, cy)
		p.CubeBezier(ToFixedP(lx+alpha*ldx, ly+alpha*ldy),
			ToFixedP(px-alpha*dx, py-alpha*dy), ToFixedP(px, py))
		lx, ly, ldx, ldy = px, py, dx, dy
	}
	return lx, ly
}

// EllipsePrime gives tangent vectors for parameterized ellipse; a, b, radii, eta parameter, center cx, cy
func EllipsePrime(a, b, sinTheta, cosTheta, eta, cx, cy float32) (px, py float32) {
	bCosEta := b * math32.Cos(eta)
	aSinEta := a * math32.Sin(eta)
	px = -aSinEta*cosTheta - bCosEta*sinTheta
	py = -aSinEta*sinTheta + bCosEta*cosTheta
	return
}

// EllipsePointAt gives points for parameterized ellipse; a, b, radii, eta parameter, center cx, cy
func EllipsePointAt(a, b, sinTheta, cosTheta, eta, cx, cy float32) (px, py float32) {
	aCosEta := a * math32.Cos(eta)
	bSinEta := b * math32.Sin(eta)
	px = cx + aCosEta*cosTheta - bSinEta*sinTheta
	py = cy + aCosEta*sinTheta + bSinEta*cosTheta
	return
}

// FindEllipseCenter locates the center of the Ellipse if it exists. If it does not exist,
// the radius values will be increased minimally for a solution to be possible
// while preserving the ra to rb ratio.  ra and rb arguments are pointers that can be
// checked after the call to see if the values changed. This method uses coordinate transformations
// to reduce the problem to finding the center of a circle that includes the origin
// and an arbitrary point. The center of the circle is then transformed
// back to the original coordinates and returned.
func FindEllipseCenter(ra, rb *float32, rotX, startX, startY, endX, endY float32, sweep, smallArc bool) (cx, cy float32) {
	cos, sin := math32.Cos(rotX), math32.Sin(rotX)

	// Move origin to start point
	nx, ny := endX-startX, endY-startY

	// Rotate ellipse x-axis to coordinate x-axis
	nx, ny = nx*cos+ny*sin, -nx*sin+ny*cos
	// Scale X dimension so that ra = rb
	nx *= *rb / *ra // Now the ellipse is a circle radius rb; therefore foci and center coincide

	midX, midY := nx/2, ny/2
	midlenSq := midX*midX + midY*midY

	var hr float32
	if *rb**rb < midlenSq {
		// Requested ellipse does not exist; scale ra, rb to fit. Length of
		// span is greater than max width of ellipse, must scale *ra, *rb
		nrb := math32.Sqrt(midlenSq)
		if *ra == *rb {
			*ra = nrb // prevents roundoff
		} else {
			*ra = *ra * nrb / *rb
		}
		*rb = nrb
	} else {
		hr = math32.Sqrt(*rb**rb-midlenSq) / math32.Sqrt(midlenSq)
	}
	// Notice that if hr is zero, both answers are the same.
	if (sweep && smallArc) || (!sweep && !smallArc) {
		cx = midX + midY*hr
		cy = midY - midX*hr
	} else {
		cx = midX - midY*hr
		cy = midY + midX*hr
	}

	// reverse scale
	cx *= *ra / *rb
	//Reverse rotate and translate back to original coordinates
	return cx*cos - cy*sin + startX, cx*sin + cy*cos + startY
}
