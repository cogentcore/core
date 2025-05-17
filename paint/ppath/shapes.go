// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/sides"
)

// Line adds a line segment of from (x1,y1) to (x2,y2).
func (p *Path) Line(x1, y1, x2, y2 float32) *Path {
	if Equal(x1, x2) && Equal(y1, y2) {
		return p
	}
	p.MoveTo(x1, y1)
	p.LineTo(x2, y2)
	return p
}

// Polyline adds multiple connected lines, with no final Close.
func (p *Path) Polyline(points ...math32.Vector2) *Path {
	sz := len(points)
	if sz < 2 {
		return p
	}
	p.MoveTo(points[0].X, points[0].Y)
	for i := 1; i < sz; i++ {
		p.LineTo(points[i].X, points[i].Y)
	}
	return p
}

// Polygon adds multiple connected lines with a final Close.
func (p *Path) Polygon(points ...math32.Vector2) *Path {
	p.Polyline(points...)
	p.Close()
	return p
}

// Rectangle adds a rectangle of width w and height h.
func (p *Path) Rectangle(x, y, w, h float32) *Path {
	if Equal(w, 0.0) || Equal(h, 0.0) {
		return p
	}
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
	return p
}

// RoundedRectangle adds a rectangle of width w and height h
// with rounded corners of radius r. A negative radius will cast
// the corners inwards (i.e. concave).
func (p *Path) RoundedRectangle(x, y, w, h, r float32) *Path {
	if Equal(w, 0.0) || Equal(h, 0.0) {
		return p
	} else if Equal(r, 0.0) {
		return p.Rectangle(x, y, w, h)
	}

	sweep := true
	if r < 0.0 {
		sweep = false
		r = -r
	}
	r = math32.Min(r, w/2.0)
	r = math32.Min(r, h/2.0)

	p.MoveTo(x, y+r)
	p.ArcTo(r, r, 0.0, false, sweep, x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(r, r, 0.0, false, sweep, x+w, y+r)
	p.LineTo(x+w, y+h-r)
	p.ArcTo(r, r, 0.0, false, sweep, x+w-r, y+h)
	p.LineTo(x+r, y+h)
	p.ArcTo(r, r, 0.0, false, sweep, x, y+h-r)
	p.Close()
	return p
}

// RoundedRectangleSides draws a standard rounded rectangle
// with a consistent border and with the given x and y position,
// width and height, and border radius for each corner.
// This version uses the Arc elliptical arc function.
func (p *Path) RoundedRectangleSides(x, y, w, h float32, r sides.Floats) *Path {
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

	p.MoveTo(xtl, ytli)
	if r.Top != 0 {
		p.ArcTo(r.Top, r.Top, 0, false, true, xtli, ytl)
	}
	p.LineTo(xtri, ytr)
	if r.Right != 0 {
		p.ArcTo(r.Right, r.Right, 0, false, true, xbr, ytri)
	}
	p.LineTo(xbr, ybri)
	if r.Bottom != 0 {
		p.ArcTo(r.Bottom, r.Bottom, 0, false, true, xbri, ybl)
	}
	p.LineTo(xbli, ybl)
	if r.Left != 0 {
		p.ArcTo(r.Left, r.Left, 0, false, true, xtl, ybli)
	}
	p.Close()
	return p
}

// BeveledRectangle adds a rectangle of width w and height h
// with beveled corners at distance r from the corner.
func (p *Path) BeveledRectangle(x, y, w, h, r float32) *Path {
	if Equal(w, 0.0) || Equal(h, 0.0) {
		return p
	} else if Equal(r, 0.0) {
		return p.Rectangle(x, y, w, h)
	}

	r = math32.Abs(r)
	r = math32.Min(r, w/2.0)
	r = math32.Min(r, h/2.0)

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

// Circle adds a circle at given center coordinates of radius r.
func (p *Path) Circle(cx, cy, r float32) *Path {
	return p.Ellipse(cx, cy, r, r)
}

// Ellipse adds an ellipse at given center coordinates of radii rx and ry.
func (p *Path) Ellipse(cx, cy, rx, ry float32) *Path {
	if Equal(rx, 0.0) || Equal(ry, 0.0) {
		return p
	}

	p.MoveTo(cx+rx, cy+(ry*0.001))
	p.ArcTo(rx, ry, 0.0, false, true, cx-rx, cy)
	p.ArcTo(rx, ry, 0.0, false, true, cx+rx, cy)
	p.Close()
	return p
}

// CircularArc adds a circular arc centered at given coordinates with radius r
// and theta0 and theta1 as the angles in degrees of the ellipse
// (before rot is applied) between which the arc will run.
// If theta0 < theta1, the arc will run in a CCW direction.
// If the difference between theta0 and theta1 is bigger than 360 degrees,
// one full circle will be drawn and the remaining part of diff % 360,
// e.g. a difference of 810 degrees will draw one full circle and an arc
// over 90 degrees.
func (p *Path) CircularArc(x, y, r, theta0, theta1 float32) *Path {
	return p.EllipticalArc(x, y, r, r, 0, theta0, theta1)
}

// EllipticalArc adds an elliptical arc centered at given coordinates with
// radii rx and ry, with rot the counter clockwise rotation in radians,
// and theta0 and theta1 the angles in radians of the ellipse
// (before rot is applied) between which the arc will run.
// If theta0 < theta1, the arc will run in a CCW direction.
// If the difference between theta0 and theta1 is bigger than 360 degrees,
// one full circle will be drawn and the remaining part of diff % 360,
// e.g. a difference of 810 degrees will draw one full circle and an arc
// over 90 degrees.
func (p *Path) EllipticalArc(x, y, rx, ry, rot, theta0, theta1 float32) *Path {
	sins, coss := math32.Sincos(theta0)
	sx := rx * coss
	sy := ry * sins
	p.MoveTo(x+sx, y+sy)
	p.Arc(rx, ry, rot, theta0, theta1)
	return p
}

// Triangle adds a triangle of radius r pointing upwards.
func (p *Path) Triangle(r float32) *Path {
	return p.RegularPolygon(3, r, true)
}

// RegularPolygon adds a regular polygon with radius r.
// It uses n vertices/edges, so when n approaches infinity
// this will return a path that approximates a circle.
// n must be 3 or more. The up boolean defines whether
// the first point will point upwards or downwards.
func (p *Path) RegularPolygon(n int, r float32, up bool) *Path {
	return p.RegularStarPolygon(n, 1, r, up)
}

// RegularStarPolygon adds a regular star polygon with radius r.
// It uses n vertices of density d. This will result in a
// self-intersection star in counter clockwise direction.
// If n/2 < d the star will be clockwise and if n and d are not coprime
// a regular polygon will be obtained, possible with multiple windings.
// n must be 3 or more and d 2 or more. The up boolean defines whether
// the first point will point upwards or downwards.
func (p *Path) RegularStarPolygon(n, d int, r float32, up bool) *Path {
	if n < 3 || d < 1 || n == d*2 || Equal(r, 0.0) {
		return p
	}

	dtheta := 2.0 * math32.Pi / float32(n)
	theta0 := float32(0.5 * math32.Pi)
	if !up {
		theta0 += dtheta / 2.0
	}

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

// StarPolygon adds a star polygon of n points with alternating
// radius R and r. The up boolean defines whether the first point
// will be point upwards or downwards.
func (p *Path) StarPolygon(n int, R, r float32, up bool) *Path {
	if n < 3 || Equal(R, 0.0) || Equal(r, 0.0) {
		return p
	}

	n *= 2
	dtheta := 2.0 * math32.Pi / float32(n)
	theta0 := float32(0.5 * math32.Pi)
	if !up {
		theta0 += dtheta
	}

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

// Grid adds a stroked grid of width w and height h,
// with grid line thickness r, and the number of cells horizontally
// and vertically as nx and ny respectively.
func (p *Path) Grid(w, h float32, nx, ny int, r float32) *Path {
	if nx < 1 || ny < 1 || w <= float32(nx+1)*r || h <= float32(ny+1)*r {
		return p
	}

	p.Rectangle(0, 0, w, h)
	dx, dy := (w-float32(nx+1)*r)/float32(nx), (h-float32(ny+1)*r)/float32(ny)
	cell := New().Rectangle(0, 0, dx, dy).Reverse()
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			x := r + float32(i)*(r+dx)
			y := r + float32(j)*(r+dy)
			*p = p.Append(cell.Translate(x, y))
		}
	}
	return p
}

func ArcToQuad(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) Path {
	p := Path{}
	p.MoveTo(start.X, start.Y)
	for _, bezier := range ellipseToQuadraticBeziers(start, rx, ry, phi, large, sweep, end) {
		p.QuadTo(bezier[1].X, bezier[1].Y, bezier[2].X, bezier[2].Y)
	}
	return p
}

func ArcToCube(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) Path {
	p := Path{}
	p.MoveTo(start.X, start.Y)
	for _, bezier := range ellipseToCubicBeziers(start, rx, ry, phi, large, sweep, end) {
		p.CubeTo(bezier[1].X, bezier[1].Y, bezier[2].X, bezier[2].Y, bezier[3].X, bezier[3].Y)
	}
	return p
}
