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
			cx, cy, _, _ := EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
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
