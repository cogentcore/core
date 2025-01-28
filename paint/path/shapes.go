// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/sides"
)

// Line returns a line segment of from (x1,y1) to (x2,y2).
func Line(x1, y1, x2, y2 float32) Path {
	if Equal(x1, x2) && Equal(y1, y2) {
		return Path{}
	}

	p := Path{}
	p.MoveTo(x1, y1)
	p.LineTo(x2, y2)
	return p
}

// Arc returns a circular arc with radius r and theta0 and theta1 as the angles
// in degrees of the ellipse (before rot is applied) between which the arc
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
	p.ArcDeg(rx, ry, rot, theta0, theta1)
	return p
}

// Rectangle returns a rectangle of width w and height h.
func Rectangle(x, y, w, h float32) Path {
	if Equal(w, 0.0) || Equal(h, 0.0) {
		return Path{}
	}
	p := Path{}
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
	return p
}

// RoundedRectangle returns a rectangle of width w and height h
// with rounded corners of radius r. A negative radius will cast
// the corners inwards (i.e. concave).
func RoundedRectangle(x, y, w, h, r float32) Path {
	if Equal(w, 0.0) || Equal(h, 0.0) {
		return Path{}
	} else if Equal(r, 0.0) {
		return Rectangle(x, y, w, h)
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

// RoundedRectangleSides draws a standard rounded rectangle
// with a consistent border and with the given x and y position,
// width and height, and border radius for each corner.
func RoundedRectangleSides(x, y, w, h float32, r sides.Floats) Path {
	// clamp border radius values
	min := math32.Min(w/2, h/2)
	r.Top = math32.Clamp(r.Top, 0, min)
	r.Right = math32.Clamp(r.Right, 0, min)
	r.Bottom = math32.Clamp(r.Bottom, 0, min)
	r.Left = math32.Clamp(r.Left, 0, min)

	// position values; some variables are missing because they are unused
	var (
		xtl, ytl   = x, y                 // top left
		xtli, ytli = x + r.Top, y + r.Top // top left inset

		ytr        = y                            // top right
		xtri, ytri = x + w - r.Right, y + r.Right // top right inset

		xbr        = x + w                              // bottom right
		xbri, ybri = x + w - r.Bottom, y + h - r.Bottom // bottom right inset

		ybl        = y + h                      // bottom left
		xbli, ybli = x + r.Left, y + h - r.Left // bottom left inset
	)

	p := Path{}
	p.MoveTo(xtl, ytli)
	p.ArcTo(r.Top, r.Top, 0, false, true, xtli, ytl)
	p.LineTo(xtri, ytr)
	p.ArcTo(r.Right, r.Right, 0, false, true, xbr, ytri)
	p.LineTo(xbr, ybri)
	p.ArcTo(r.Bottom, r.Bottom, 0, false, true, xbri, ybl)
	p.LineTo(xbli, ybl)
	p.ArcTo(r.Left, r.Left, 0, false, true, xtl, ybli)
	p.Close()
	return p
}

// BeveledRectangle returns a rectangle of width w and height h
// with beveled corners at distance r from the corner.
func BeveledRectangle(x, y, w, h, r float32) Path {
	if Equal(w, 0.0) || Equal(h, 0.0) {
		return Path{}
	} else if Equal(r, 0.0) {
		return Rectangle(x, y, w, h)
	}

	r = math32.Abs(r)
	r = math32.Min(r, w/2.0)
	r = math32.Min(r, h/2.0)

	p := Path{}
	p.MoveTo(x, y+r)
	p.LineTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.LineTo(x+w, y+r)
	p.LineTo(x+w, y+h-r)
	p.LineTo(x+w-r, y+h)
	p.LineTo(x+r, y+h)
	p.LineTo(x, y+h-r)
	p.Close()
	return p
}

// Circle returns a circle of radius r.
func Circle(x, y, r float32) Path {
	return Ellipse(x, y, r, r)
}

// Ellipse returns an ellipse of radii rx and ry.
func Ellipse(x, y, rx, ry float32) Path {
	if Equal(rx, 0.0) || Equal(ry, 0.0) {
		return Path{}
	}

	p := Path{}
	p.MoveTo(x+rx, y)
	p.ArcTo(rx, ry, 0.0, false, true, x-rx, y)
	p.ArcTo(rx, ry, 0.0, false, true, x+rx, y)
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

	p := Rectangle(0, 0, w, h)
	dx, dy := (w-float32(nx+1)*r)/float32(nx), (h-float32(ny+1)*r)/float32(ny)
	cell := Rectangle(0, 0, dx, dy).Reverse()
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
