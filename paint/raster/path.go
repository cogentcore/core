// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package raster

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/path"
	"golang.org/x/image/vector"
)

// todo: resolution, use scan instead of vector

// ToRasterizer rasterizes the path using the given rasterizer and resolution.
func ToRasterizer(p path.Path, ras *vector.Rasterizer, resolution float32) {
	// TODO: smoothen path using Ramer-...

	dpmm := float32(96.0 / 25.4)            // todo: resolution.DPMM()
	tolerance := path.PixelTolerance / dpmm // tolerance of 1/10 of a pixel
	dy := float32(ras.Bounds().Size().Y)
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case path.MoveTo:
			ras.MoveTo(p[i+1]*dpmm, dy-p[i+2]*dpmm)
		case path.LineTo:
			ras.LineTo(p[i+1]*dpmm, dy-p[i+2]*dpmm)
		case path.QuadTo, path.CubeTo, path.ArcTo:
			// flatten
			var q path.Path
			var start math32.Vector2
			if 0 < i {
				start = math32.Vec2(p[i-3], p[i-2])
			}
			if cmd == path.QuadTo {
				cp := math32.Vec2(p[i+1], p[i+2])
				end := math32.Vec2(p[i+3], p[i+4])
				q = path.FlattenQuadraticBezier(start, cp, end, tolerance)
			} else if cmd == path.CubeTo {
				cp1 := math32.Vec2(p[i+1], p[i+2])
				cp2 := math32.Vec2(p[i+3], p[i+4])
				end := math32.Vec2(p[i+5], p[i+6])
				q = path.FlattenCubicBezier(start, cp1, cp2, end, tolerance)
			} else {
				rx, ry, phi, large, sweep, end := p.ArcToPoints(i)
				q = path.FlattenEllipticArc(start, rx, ry, phi, large, sweep, end, tolerance)
			}
			for j := 4; j < len(q); j += 4 {
				ras.LineTo(q[j+1]*dpmm, dy-q[j+2]*dpmm)
			}
		case path.Close:
			ras.ClosePath()
		default:
			panic("quadratic and cubic Béziers and arcs should have been replaced")
		}
		i += path.CmdLen(cmd)
	}
	if !p.Closed() {
		// implicitly close path
		ras.ClosePath()
	}
}
