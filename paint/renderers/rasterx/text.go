// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // load image formats for users of the API
	_ "image/png"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/text/shaped"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/tiff" // load image formats for users of the API
)

// RenderText rasterizes the given Text
func (rs *Renderer) RenderText(txt *render.Text) {
	rs.TextLines(txt.Text, &txt.Context, txt.Position)
}

// TextLines rasterizes the given shaped.Lines.
// The text will be drawn starting at the start pixel position, which specifies the
// left baseline location of the first text item..
func (rs *Renderer) TextLines(lns *shaped.Lines, ctx *render.Context, pos math32.Vector2) {
	start := pos.Add(lns.Offset)
	// tbb := lns.Bounds.Translate(start)
	// rs.StrokeBounds(tbb, colors.Red)
	clr := colors.Uniform(lns.Color)
	for li := range lns.Lines {
		ln := &lns.Lines[li]
		rs.TextLine(ln, clr, start) // todo: start + offset
	}
}

// TextLine rasterizes the given shaped.Line.
func (rs *Renderer) TextLine(ln *shaped.Line, clr image.Image, start math32.Vector2) {
	off := start.Add(ln.Offset)
	// tbb := ln.Bounds.Translate(off)
	// rs.StrokeBounds(tbb, colors.Blue)
	for ri := range ln.Runs {
		run := &ln.Runs[ri]
		rs.TextRun(run, clr, off)
		if run.Direction.IsVertical() {
			off.Y += math32.FromFixed(run.Advance)
		} else {
			off.X += math32.FromFixed(run.Advance)
		}
	}
}

// TextRun rasterizes the given text run into the output image using the
// font face set in the shaping.
// The text will be drawn starting at the start pixel position.
func (rs *Renderer) TextRun(run *shaped.Run, clr image.Image, start math32.Vector2) {
	// todo: render decoration
	// dir := run.Direction
	if run.Background != nil {
		rs.FillBounds(run.MaxBounds.Translate(start), run.Background)
	}
	fill := clr
	if run.FillColor != nil {
		fill = run.FillColor
	}
	stroke := run.StrokeColor
	for gi := range run.Glyphs {
		g := &run.Glyphs[gi]
		pos := start.Add(math32.Vec2(math32.FromFixed(g.XOffset), -math32.FromFixed(g.YOffset)))
		// top := yPos - math32.FromFixed(g.YBearing)
		// bottom := top - math32.FromFixed(g.Height)
		// right := xPos + math32.FromFixed(g.Width)
		// rect := image.Rect(int(xPos)-4, int(top)-4, int(right)+4, int(bottom)+4) // don't cut off
		bb := math32.B2FromFixed(run.GlyphBounds(g)).Translate(start)
		// rs.StrokeBounds(bb, colors.Yellow)

		data := run.Face.GlyphData(g.GlyphID)
		switch format := data.(type) {
		case font.GlyphOutline:
			rs.GlyphOutline(run, g, format, fill, stroke, bb, pos)
		case font.GlyphBitmap:
			fmt.Println("bitmap")
			rs.GlyphBitmap(run, g, format, fill, stroke, bb, pos)
		case font.GlyphSVG:
			fmt.Println("svg", format)
			// 	_ = rs.GlyphSVG(g, format, fill, stroke, bb, pos)
		}
		start.X += math32.FromFixed(g.XAdvance)
		start.Y -= math32.FromFixed(g.YAdvance)
	}
	// todo: render strikethrough
}

func (rs *Renderer) GlyphOutline(run *shaped.Run, g *shaping.Glyph, bitmap font.GlyphOutline, fill, stroke image.Image, bb math32.Box2, pos math32.Vector2) {
	scale := math32.FromFixed(run.Size) / float32(run.Face.Upem())
	x := pos.X
	y := pos.Y

	rs.Path.Clear()
	for _, s := range bitmap.Segments {
		switch s.Op {
		case opentype.SegmentOpMoveTo:
			rs.Path.Start(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)})
		case opentype.SegmentOpLineTo:
			rs.Path.Line(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)})
		case opentype.SegmentOpQuadTo:
			rs.Path.QuadBezier(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)},
				fixed.Point26_6{X: math32.ToFixed(s.Args[1].X*scale + x), Y: math32.ToFixed(-s.Args[1].Y*scale + y)})
		case opentype.SegmentOpCubeTo:
			rs.Path.CubeBezier(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)},
				fixed.Point26_6{X: math32.ToFixed(s.Args[1].X*scale + x), Y: math32.ToFixed(-s.Args[1].Y*scale + y)},
				fixed.Point26_6{X: math32.ToFixed(s.Args[2].X*scale + x), Y: math32.ToFixed(-s.Args[2].Y*scale + y)})
		}
	}
	rs.Path.Stop(true)
	rf := &rs.Raster.Filler
	rf.SetWinding(true)
	rf.SetColor(fill)
	rs.Path.AddTo(rf)
	rf.Draw()
	rf.Clear()

	if stroke != nil {
		sw := math32.FromFixed(run.Size) / 32.0 // scale with font size
		rs.Raster.SetStroke(
			math32.ToFixed(sw),
			math32.ToFixed(10),
			ButtCap, nil, nil, Miter, nil, 0)
		rs.Path.AddTo(rs.Raster)
		rs.Raster.SetColor(stroke)
		rs.Raster.Draw()
		rs.Raster.Clear()
	}
	rs.Path.Clear()
}

func (rs *Renderer) GlyphBitmap(run *shaped.Run, g *shaping.Glyph, bitmap font.GlyphBitmap, fill, stroke image.Image, bb math32.Box2, pos math32.Vector2) error {
	// scaled glyph rect content
	x := pos.X
	y := pos.Y
	top := y - math32.FromFixed(g.YBearing)
	switch bitmap.Format {
	case font.BlackAndWhite:
		rec := image.Rect(0, 0, bitmap.Width, bitmap.Height)
		sub := image.NewPaletted(rec, color.Palette{color.Transparent, colors.ToUniform(fill)})

		for i := range sub.Pix {
			sub.Pix[i] = bitAt(bitmap.Data, i)
		}
		// todo: does it need scale? presumably not
		// scale.NearestNeighbor.Scale(img, bb, sub, sub.Bounds(), int(top)}, draw.Over, nil)
		draw.Draw(rs.image, sub.Bounds(), sub, image.Point{int(x), int(top)}, draw.Over)
	case font.JPG, font.PNG, font.TIFF:
		fmt.Println("img")
		// todo: how often?
		pix, _, err := image.Decode(bytes.NewReader(bitmap.Data))
		if err != nil {
			return err
		}
		// scale.BiLinear.Scale(img, bb, pix, pix.Bounds(), draw.Over, nil)
		draw.Draw(rs.image, pix.Bounds(), pix, image.Point{int(x), int(top)}, draw.Over)
	}

	if bitmap.Outline != nil {
		rs.GlyphOutline(run, g, *bitmap.Outline, fill, stroke, bb, pos)
	}
	return nil
}

// bitAt returns the bit at the given index in the byte slice.
func bitAt(b []byte, i int) byte {
	return (b[i/8] >> (7 - i%8)) & 1
}

// StrokeBounds strokes a bounding box in the given color. Useful for debugging.
func (rs *Renderer) StrokeBounds(bb math32.Box2, clr color.Color) {
	rs.Raster.Clear()
	rs.Raster.SetStroke(
		math32.ToFixed(1),
		math32.ToFixed(4),
		ButtCap, nil, nil, Miter,
		nil, 0)
	rs.Raster.SetColor(colors.Uniform(clr))
	AddRect(bb.Min.X, bb.Min.Y, bb.Max.X, bb.Max.Y, 0, rs.Raster)
	rs.Raster.Draw()
	rs.Raster.Clear()
}

// FillBounds fills a bounding box in the given color.
func (rs *Renderer) FillBounds(bb math32.Box2, clr image.Image) {
	rf := &rs.Raster.Filler
	rf.Clear()
	rf.SetColor(clr)
	AddRect(bb.Min.X, bb.Min.Y, bb.Max.X, bb.Max.Y, 0, rf)
	rf.Draw()
	rf.Clear()
}
