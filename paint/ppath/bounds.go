// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import "cogentcore.org/core/math32"

// FastBounds returns the maximum bounding box rectangle of the path.
// It is quicker than Bounds but less accurate.
func (p Path) FastBounds() math32.Box2 {
	if len(p) < 4 {
		return math32.Box2{}
	}

	// first command is MoveTo
	start, end := math32.Vec2(p[1], p[2]), math32.Vector2{}
	xmin, xmax := start.X, start.X
	ymin, ymax := start.Y, start.Y
	for i := 4; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo, LineTo, Close:
			end = math32.Vec2(p[i+1], p[i+2])
			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
		case QuadTo:
			cp := math32.Vec2(p[i+1], p[i+2])
			end = math32.Vec2(p[i+3], p[i+4])
			xmin = math32.Min(xmin, math32.Min(cp.X, end.X))
			xmax = math32.Max(xmax, math32.Max(cp.X, end.X))
			ymin = math32.Min(ymin, math32.Min(cp.Y, end.Y))
			ymax = math32.Max(ymax, math32.Max(cp.Y, end.Y))
		case CubeTo:
			cp1 := math32.Vec2(p[i+1], p[i+2])
			cp2 := math32.Vec2(p[i+3], p[i+4])
			end = math32.Vec2(p[i+5], p[i+6])
			xmin = math32.Min(xmin, math32.Min(cp1.X, math32.Min(cp2.X, end.X)))
			xmax = math32.Max(xmax, math32.Max(cp1.X, math32.Min(cp2.X, end.X)))
			ymin = math32.Min(ymin, math32.Min(cp1.Y, math32.Min(cp2.Y, end.Y)))
			ymax = math32.Max(ymax, math32.Max(cp1.Y, math32.Min(cp2.Y, end.Y)))
		case ArcTo:
			var rx, ry, phi float32
			var large, sweep bool
			rx, ry, phi, large, sweep, end = p.ArcToPoints(i)
			cx, cy, _, _ := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			r := math32.Max(rx, ry)
			xmin = math32.Min(xmin, cx-r)
			xmax = math32.Max(xmax, cx+r)
			ymin = math32.Min(ymin, cy-r)
			ymax = math32.Max(ymax, cy+r)

		}
		i += CmdLen(cmd)
		start = end
	}
	return math32.B2(xmin, ymin, xmax, ymax)
}

// Bounds returns the exact bounding box rectangle of the path.
func (p Path) Bounds() math32.Box2 {
	if len(p) < 4 {
		return math32.Box2{}
	}

	// first command is MoveTo
	start, end := math32.Vec2(p[1], p[2]), math32.Vector2{}
	xmin, xmax := start.X, start.X
	ymin, ymax := start.Y, start.Y
	for i := 4; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo, LineTo, Close:
			end = math32.Vec2(p[i+1], p[i+2])
			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
		case QuadTo:
			cp := math32.Vec2(p[i+1], p[i+2])
			end = math32.Vec2(p[i+3], p[i+4])

			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			if tdenom := (start.X - 2*cp.X + end.X); !Equal(tdenom, 0.0) {
				if t := (start.X - cp.X) / tdenom; InIntervalExclusive(t, 0.0, 1.0) {
					x := quadraticBezierPos(start, cp, end, t)
					xmin = math32.Min(xmin, x.X)
					xmax = math32.Max(xmax, x.X)
				}
			}

			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
			if tdenom := (start.Y - 2*cp.Y + end.Y); !Equal(tdenom, 0.0) {
				if t := (start.Y - cp.Y) / tdenom; InIntervalExclusive(t, 0.0, 1.0) {
					y := quadraticBezierPos(start, cp, end, t)
					ymin = math32.Min(ymin, y.Y)
					ymax = math32.Max(ymax, y.Y)
				}
			}
		case CubeTo:
			cp1 := math32.Vec2(p[i+1], p[i+2])
			cp2 := math32.Vec2(p[i+3], p[i+4])
			end = math32.Vec2(p[i+5], p[i+6])

			a := -start.X + 3*cp1.X - 3*cp2.X + end.X
			b := 2*start.X - 4*cp1.X + 2*cp2.X
			c := -start.X + cp1.X
			t1, t2 := solveQuadraticFormula(a, b, c)

			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			if !math32.IsNaN(t1) && InIntervalExclusive(t1, 0.0, 1.0) {
				x1 := cubicBezierPos(start, cp1, cp2, end, t1)
				xmin = math32.Min(xmin, x1.X)
				xmax = math32.Max(xmax, x1.X)
			}
			if !math32.IsNaN(t2) && InIntervalExclusive(t2, 0.0, 1.0) {
				x2 := cubicBezierPos(start, cp1, cp2, end, t2)
				xmin = math32.Min(xmin, x2.X)
				xmax = math32.Max(xmax, x2.X)
			}

			a = -start.Y + 3*cp1.Y - 3*cp2.Y + end.Y
			b = 2*start.Y - 4*cp1.Y + 2*cp2.Y
			c = -start.Y + cp1.Y
			t1, t2 = solveQuadraticFormula(a, b, c)

			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
			if !math32.IsNaN(t1) && InIntervalExclusive(t1, 0.0, 1.0) {
				y1 := cubicBezierPos(start, cp1, cp2, end, t1)
				ymin = math32.Min(ymin, y1.Y)
				ymax = math32.Max(ymax, y1.Y)
			}
			if !math32.IsNaN(t2) && InIntervalExclusive(t2, 0.0, 1.0) {
				y2 := cubicBezierPos(start, cp1, cp2, end, t2)
				ymin = math32.Min(ymin, y2.Y)
				ymax = math32.Max(ymax, y2.Y)
			}
		case ArcTo:
			var rx, ry, phi float32
			var large, sweep bool
			rx, ry, phi, large, sweep, end = p.ArcToPoints(i)
			cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

			// find the four extremes (top, bottom, left, right) and apply those who are between theta1 and theta2
			// x(theta) = cx + rx*cos(theta)*cos(phi) - ry*sin(theta)*sin(phi)
			// y(theta) = cy + rx*cos(theta)*sin(phi) + ry*sin(theta)*cos(phi)
			// be aware that positive rotation appears clockwise in SVGs (non-Cartesian coordinate system)
			// we can now find the angles of the extremes

			sinphi, cosphi := math32.Sincos(phi)
			thetaRight := math32.Atan2(-ry*sinphi, rx*cosphi)
			thetaTop := math32.Atan2(rx*cosphi, ry*sinphi)
			thetaLeft := thetaRight + math32.Pi
			thetaBottom := thetaTop + math32.Pi

			dx := math32.Sqrt(rx*rx*cosphi*cosphi + ry*ry*sinphi*sinphi)
			dy := math32.Sqrt(rx*rx*sinphi*sinphi + ry*ry*cosphi*cosphi)
			if angleBetween(thetaLeft, theta0, theta1) {
				xmin = math32.Min(xmin, cx-dx)
			}
			if angleBetween(thetaRight, theta0, theta1) {
				xmax = math32.Max(xmax, cx+dx)
			}
			if angleBetween(thetaBottom, theta0, theta1) {
				ymin = math32.Min(ymin, cy-dy)
			}
			if angleBetween(thetaTop, theta0, theta1) {
				ymax = math32.Max(ymax, cy+dy)
			}
			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
		}
		i += CmdLen(cmd)
		start = end
	}
	return math32.B2(xmin, ymin, xmax, ymax)
}
