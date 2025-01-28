// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package rasterizer

import (
	"fmt"
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/path"
	"golang.org/x/image/vector"
)

func (r *Renderer) RenderPath(pt *paint.Path) {
	pc := &pt.Context
	sty := &pc.Style
	var fill, stroke path.Path
	var bounds math32.Box2
	if sty.HasFill() {
		fill = pt.Path.Clone() // .Transform(pc.Transform)
		bounds = fill.FastBounds()
	}
	if sty.HasStroke() {
		tolerance := path.PixelTolerance
		stroke = pt.Path
		if 0 < len(sty.Stroke.Dashes) {
			dashOffset, dashes := path.ScaleDash(sty.Stroke.Width.Dots, sty.Stroke.DashOffset, sty.Stroke.Dashes)
			stroke = stroke.Dash(dashOffset, dashes...)
		}
		stroke = stroke.Stroke(sty.Stroke.Width.Dots, path.CapFromStyle(sty.Stroke.Cap), path.JoinFromStyle(sty.Stroke.Join), tolerance)
		// stroke = stroke.Transform(pc.Transform)
		if sty.HasFill() {
			bounds = bounds.Union(stroke.FastBounds())
		} else {
			bounds = stroke.FastBounds()
		}
	}

	dx, dy := 0, 0
	origin := pc.Bounds.Rect.Min
	size := pc.Bounds.Rect.Size()
	isz := size.ToPoint()
	w := isz.X
	h := isz.Y
	x := int(origin.X)
	y := int(origin.Y)

	if sty.HasFill() {
		// if sty.Fill.IsPattern() {
		// 	if hatch, ok := sty.Fill.Pattern.(*canvas.HatchPattern); ok {
		// 		sty.Fill = hatch.Fill
		// 		fill = hatch.Tile(fill)
		// 	}
		// }

		r.ras.Reset(w, h)
		// todo: clip here:
		ToRasterizer(fill, r.ras)
		r.ras.Draw(r.Image, image.Rect(x, y, x+w, y+h), sty.Fill.Color, image.Point{dx, dy})
	}
	if sty.HasStroke() {
		// if sty.Stroke.IsPattern() {
		// 	if hatch, ok := sty.Stroke.Pattern.(*canvas.HatchPattern); ok {
		// 		sty.Stroke = hatch.Fill
		// 		stroke = hatch.Tile(stroke)
		// 	}
		// }

		r.ras.Reset(w, h)
		// todo: clip here
		fmt.Println(stroke)
		ToRasterizer(stroke, r.ras)
		r.ras.Draw(r.Image, image.Rect(x, y, x+w, y+h), sty.Stroke.Color, image.Point{dx, dy})
	}
}

// ToRasterizer rasterizes the path using the given rasterizer and resolution.
func ToRasterizer(p path.Path, ras *vector.Rasterizer) {
	// TODO: smoothen path using Ramer-...

	tolerance := path.PixelTolerance
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case path.MoveTo:
			ras.MoveTo(p[i+1], p[i+2])
		case path.LineTo:
			ras.LineTo(p[i+1], p[i+2])
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
				ras.LineTo(q[j+1], q[j+2])
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

// RenderText renders a text object to the canvas using a transformation matrix.
// func (r *Rasterizer) RenderText(text *canvas.Text, m canvas.Matrix) {
// 	text.RenderAsPath(r, m, r.resolution)
// }

// RenderImage renders an image to the canvas using a transformation matrix.
// func (r *Rasterizer) RenderImage(img image.Image, m canvas.Matrix) {
// 	// add transparent margin to image for smooth borders when rotating
// 	// TODO: optimize when transformation is only translation or stretch (if optimizing, dont overwrite original img when gamma correcting)
// 	margin := 0
// 	if (m[0][1] != 0.0 || m[1][0] != 0.0) && (m[0][0] != 0.0 || m[1][1] == 0.0) {
// 		// only add margin for shear transformation or rotations that are not 90/180/270 degrees
// 		margin = 4
// 		size := img.Bounds().Size()
// 		sp := img.Bounds().Min // starting point
// 		img2 := image.NewRGBA(image.Rect(0, 0, size.X+margin*2, size.Y+margin*2))
// 		draw.Draw(img2, image.Rect(margin, margin, size.X+margin, size.Y+margin), img, sp, draw.Over)
// 		img = img2
// 	}
//
// 	if _, ok := r.colorSpace.(canvas.LinearColorSpace); !ok {
// 		// gamma decompress
// 		changeColorSpace(img.(draw.Image), img, r.colorSpace.ToLinear)
// 	}
//
// 	// draw to destination image
// 	// note that we need to correct for the added margin in origin and m
// 	dpmm := r.resolution.DPMM()
// 	origin := m.Dot(canvas.Point{-float64(margin), float64(img.Bounds().Size().Y - margin)}).Mul(dpmm)
// 	m = m.Scale(dpmm, dpmm)
//
// 	h := float64(r.Bounds().Size().Y)
// 	aff3 := f64.Aff3{m[0][0], -m[0][1], origin.X, -m[1][0], m[1][1], h - origin.Y}
// 	draw.CatmullRom.Transform(r, aff3, img, img.Bounds(), draw.Over, nil)
// }
