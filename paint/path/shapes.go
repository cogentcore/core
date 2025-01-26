// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

import (
	"cogentcore.org/core/math32"
)

// Line returns a line segment of from (0,0) to (x,y).
func Line(x, y float32) Path {
	if Equal(x, 0.0) && Equal(y, 0.0) {
		return Path{}
	}

	p := Path{}
	p.LineTo(x, y)
	return p
}

// Arc returns a circular arc with radius r and theta0 and theta1 as the angles
// in degrees of the ellipse (before rot is applies) between which the arc
// will run. If theta0 < theta1, the arc will run in a CCW direction.
// If the difference between theta0 and theta1 is bigger than 360 degrees,
// one full circle will be drawn and the remaining part of diff % 360,
// e.g. a difference of 810 degrees will draw one full circle and an arc
// over 90 degrees.
func Arc(r, theta0, theta1 float32) Path {
	return EllipticalArc(r, r, 0.0, theta0, theta1)
}

// EllipticalArc returns an elliptical arc with radii rx and ry, with rot
// the counter clockwise rotation in degrees, and theta0 and theta1 the
// angles in degrees of the ellipse (before rot is applies) between which
// the arc will run. If theta0 < theta1, the arc will run in a CCW direction.
// If the difference between theta0 and theta1 is bigger than 360 degrees,
// one full circle will be drawn and the remaining part of diff % 360,
// e.g. a difference of 810 degrees will draw one full circle and an arc
// over 90 degrees.
func EllipticalArc(rx, ry, rot, theta0, theta1 float32) Path {
	p := Path{}
	p.Arc(rx, ry, rot, theta0, theta1)
	return p
}

// Rectangle returns a rectangle of width w and height h.
func Rectangle(w, h float32) Path {
	if Equal(w, 0.0) || Equal(h, 0.0) {
		return Path{}
	}
	p := Path{}
	p.LineTo(w, 0.0)
	p.LineTo(w, h)
	p.LineTo(0.0, h)
	p.Close()
	return p
}

// RoundedRectangle returns a rectangle of width w and height h
// with rounded corners of radius r. A negative radius will cast
// the corners inwards (i.e. concave).
func RoundedRectangle(w, h, r float32) Path {
	if Equal(w, 0.0) || Equal(h, 0.0) {
		return Path{}
	} else if Equal(r, 0.0) {
		return Rectangle(w, h)
	}

	sweep := true
	if r < 0.0 {
		sweep = false
		r = -r
	}
	r = math32.Min(r, w/2.0)
	r = math32.Min(r, h/2.0)

	p := Path{}
	p.MoveTo(0.0, r)
	p.ArcTo(r, r, 0.0, false, sweep, r, 0.0)
	p.LineTo(w-r, 0.0)
	p.ArcTo(r, r, 0.0, false, sweep, w, r)
	p.LineTo(w, h-r)
	p.ArcTo(r, r, 0.0, false, sweep, w-r, h)
	p.LineTo(r, h)
	p.ArcTo(r, r, 0.0, false, sweep, 0.0, h-r)
	p.Close()
	return p
}

// BeveledRectangle returns a rectangle of width w and height h
// with beveled corners at distance r from the corner.
func BeveledRectangle(w, h, r float32) Path {
	if Equal(w, 0.0) || Equal(h, 0.0) {
		return Path{}
	} else if Equal(r, 0.0) {
		return Rectangle(w, h)
	}

	r = math32.Abs(r)
	r = math32.Min(r, w/2.0)
	r = math32.Min(r, h/2.0)

	p := Path{}
	p.MoveTo(0.0, r)
	p.LineTo(r, 0.0)
	p.LineTo(w-r, 0.0)
	p.LineTo(w, r)
	p.LineTo(w, h-r)
	p.LineTo(w-r, h)
	p.LineTo(r, h)
	p.LineTo(0.0, h-r)
	p.Close()
	return p
}

// Circle returns a circle of radius r.
func Circle(r float32) Path {
	return Ellipse(r, r)
}

// Ellipse returns an ellipse of radii rx and ry.
func Ellipse(rx, ry float32) Path {
	if Equal(rx, 0.0) || Equal(ry, 0.0) {
		return Path{}
	}

	p := Path{}
	p.MoveTo(rx, 0.0)
	p.ArcTo(rx, ry, 0.0, false, true, -rx, 0.0)
	p.ArcTo(rx, ry, 0.0, false, true, rx, 0.0)
	p.Close()
	return p
}

// Triangle returns a triangle of radius r pointing upwards.
func Triangle(r float32) Path {
	return RegularPolygon(3, r, true)
}

// RegularPolygon returns a regular polygon with radius r.
// It uses n vertices/edges, so when n approaches infinity
// this will return a path that approximates a circle.
// n must be 3 or more. The up boolean defines whether
// the first point will point upwards or downwards.
func RegularPolygon(n int, r float32, up bool) Path {
	return RegularStarPolygon(n, 1, r, up)
}

// RegularStarPolygon returns a regular star polygon with radius r.
// It uses n vertices of density d. This will result in a
// self-intersection star in counter clockwise direction.
// If n/2 < d the star will be clockwise and if n and d are not coprime
// a regular polygon will be obtained, possible with multiple windings.
// n must be 3 or more and d 2 or more. The up boolean defines whether
// the first point will point upwards or downwards.
func RegularStarPolygon(n, d int, r float32, up bool) Path {
	if n < 3 || d < 1 || n == d*2 || Equal(r, 0.0) {
		return Path{}
	}

	dtheta := 2.0 * math32.Pi / float32(n)
	theta0 := float32(0.5 * math32.Pi)
	if !up {
		theta0 += dtheta / 2.0
	}

	p := Path{}
	for i := 0; i == 0 || i%n != 0; i += d {
		theta := theta0 + float32(i)*dtheta
		sintheta, costheta := math32.Sincos(theta)
		if i == 0 {
			p.MoveTo(r*costheta, r*sintheta)
		} else {
			p.LineTo(r*costheta, r*sintheta)
		}
	}
	p.Close()
	return p
}

// StarPolygon returns a star polygon of n points with alternating
// radius R and r. The up boolean defines whether the first point
// will be point upwards or downwards.
func StarPolygon(n int, R, r float32, up bool) Path {
	if n < 3 || Equal(R, 0.0) || Equal(r, 0.0) {
		return Path{}
	}

	n *= 2
	dtheta := 2.0 * math32.Pi / float32(n)
	theta0 := float32(0.5 * math32.Pi)
	if !up {
		theta0 += dtheta
	}

	p := Path{}
	for i := 0; i < n; i++ {
		theta := theta0 + float32(i)*dtheta
		sintheta, costheta := math32.Sincos(theta)
		if i == 0 {
			p.MoveTo(R*costheta, R*sintheta)
		} else if i%2 == 0 {
			p.LineTo(R*costheta, R*sintheta)
		} else {
			p.LineTo(r*costheta, r*sintheta)
		}
	}
	p.Close()
	return p
}

// Grid returns a stroked grid of width w and height h,
// with grid line thickness r, and the number of cells horizontally
// and vertically as nx and ny respectively.
func Grid(w, h float32, nx, ny int, r float32) Path {
	if nx < 1 || ny < 1 || w <= float32(nx+1)*r || h <= float32(ny+1)*r {
		return Path{}
	}

	p := Rectangle(w, h)
	dx, dy := (w-float32(nx+1)*r)/float32(nx), (h-float32(ny+1)*r)/float32(ny)
	cell := Rectangle(dx, dy).Reverse()
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			x := r + float32(i)*(r+dx)
			y := r + float32(j)*(r+dy)
			p = p.Append(cell.Translate(x, y))
		}
	}
	return p
}

// EllipsePos returns the position on the ellipse at angle theta.
func EllipsePos(rx, ry, phi, cx, cy, theta float32) math32.Vector2 {
	sintheta, costheta := math32.Sincos(theta)
	sinphi, cosphi := math32.Sincos(phi)
	x := cx + rx*costheta*cosphi - ry*sintheta*sinphi
	y := cy + rx*costheta*sinphi + ry*sintheta*cosphi
	return math32.Vector2{x, y}
}

func arcToQuad(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) Path {
	p := Path{}
	p.MoveTo(start.X, start.Y)
	for _, bezier := range ellipseToQuadraticBeziers(start, rx, ry, phi, large, sweep, end) {
		p.QuadTo(bezier[1].X, bezier[1].Y, bezier[2].X, bezier[2].Y)
	}
	return p
}

func arcToCube(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) Path {
	p := Path{}
	p.MoveTo(start.X, start.Y)
	for _, bezier := range ellipseToCubicBeziers(start, rx, ry, phi, large, sweep, end) {
		p.CubeTo(bezier[1].X, bezier[1].Y, bezier[2].X, bezier[2].Y, bezier[3].X, bezier[3].Y)
	}
	return p
}
