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
	// rs.DrawBounds(tbb, colors.Red)
	for li := range lns.Lines {
		ln := &lns.Lines[li]
		rs.TextLine(ln, lns.Color, start) // todo: start + offset
	}
}

// TextLine rasterizes the given shaped.Line.
func (rs *Renderer) TextLine(ln *shaped.Line, clr color.Color, start math32.Vector2) {
	off := start.Add(ln.Offset)
	// tbb := ln.Bounds.Translate(off)
	// rs.DrawBounds(tbb, colors.Blue)
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
func (rs *Renderer) TextRun(run *shaping.Output, clr color.Color, start math32.Vector2) {
	x := start.X
	y := start.Y
	// todo: render bg, render decoration
	tbb := math32.B2FromFixed(shaped.OutputBounds(run)).Translate(start)
	rs.DrawBounds(tbb, colors.Red)

	for gi := range run.Glyphs {
		g := &run.Glyphs[gi]
		xPos := x + math32.FromFixed(g.XOffset)
		yPos := y - math32.FromFixed(g.YOffset)
		top := yPos - math32.FromFixed(g.YBearing)
		bottom := top - math32.FromFixed(g.Height)
		right := xPos + math32.FromFixed(g.Width)
		rect := image.Rect(int(xPos)-4, int(top)-4, int(right)+4, int(bottom)+4) // don't cut off
		data := run.Face.GlyphData(g.GlyphID)
		switch format := data.(type) {
		case font.GlyphOutline:
			rs.GlyphOutline(run, g, format, clr, rect, xPos, yPos)
		case font.GlyphBitmap:
			fmt.Println("bitmap")
			rs.GlyphBitmap(run, g, format, clr, rect, xPos, yPos)
		case font.GlyphSVG:
			fmt.Println("svg", format)
			// 	_ = rs.GlyphSVG(g, format, clr, xPos, yPos)
		}
		x += math32.FromFixed(g.XAdvance)
		y -= math32.FromFixed(g.YAdvance)
	}
	// todo: render strikethrough
}

func (rs *Renderer) GlyphOutline(run *shaping.Output, g *shaping.Glyph, bitmap font.GlyphOutline, clr color.Color, rect image.Rectangle, x, y float32) {
	rs.Raster.SetColor(colors.Uniform(clr))
	rf := &rs.Raster.Filler

	scale := math32.FromFixed(run.Size) / float32(run.Face.Upem())
	// rs.Scanner.SetClip(rect) // todo: not good -- cuts off japanese!
	rf.SetWinding(true)

	// todo: use stroke vs. fill color
	for _, s := range bitmap.Segments {
		switch s.Op {
		case opentype.SegmentOpMoveTo:
			rf.Start(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)})
		case opentype.SegmentOpLineTo:
			rf.Line(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)})
		case opentype.SegmentOpQuadTo:
			rf.QuadBezier(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)},
				fixed.Point26_6{X: math32.ToFixed(s.Args[1].X*scale + x), Y: math32.ToFixed(-s.Args[1].Y*scale + y)})
		case opentype.SegmentOpCubeTo:
			rf.CubeBezier(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)},
				fixed.Point26_6{X: math32.ToFixed(s.Args[1].X*scale + x), Y: math32.ToFixed(-s.Args[1].Y*scale + y)},
				fixed.Point26_6{X: math32.ToFixed(s.Args[2].X*scale + x), Y: math32.ToFixed(-s.Args[2].Y*scale + y)})
		}
	}
	rf.Stop(true)
	rf.Draw()
	rf.Clear()
}

func (rs *Renderer) GlyphBitmap(run *shaping.Output, g *shaping.Glyph, bitmap font.GlyphBitmap, clr color.Color, rect image.Rectangle, x, y float32) error {
	// scaled glyph rect content
	top := y - math32.FromFixed(g.YBearing)
	switch bitmap.Format {
	case font.BlackAndWhite:
		rec := image.Rect(0, 0, bitmap.Width, bitmap.Height)
		sub := image.NewPaletted(rec, color.Palette{color.Transparent, clr})

		for i := range sub.Pix {
			sub.Pix[i] = bitAt(bitmap.Data, i)
		}
		// todo: does it need scale? presumably not
		// scale.NearestNeighbor.Scale(img, rect, sub, sub.Bounds(), int(top)}, draw.Over, nil)
		draw.Draw(rs.image, sub.Bounds(), sub, image.Point{int(x), int(top)}, draw.Over)
	case font.JPG, font.PNG, font.TIFF:
		fmt.Println("img")
		// todo: how often?
		pix, _, err := image.Decode(bytes.NewReader(bitmap.Data))
		if err != nil {
			return err
		}
		// scale.BiLinear.Scale(img, rect, pix, pix.Bounds(), draw.Over, nil)
		draw.Draw(rs.image, pix.Bounds(), pix, image.Point{int(x), int(top)}, draw.Over)
	}

	if bitmap.Outline != nil {
		rs.GlyphOutline(run, g, *bitmap.Outline, clr, rect, x, y)
	}
	return nil
}

// bitAt returns the bit at the given index in the byte slice.
func bitAt(b []byte, i int) byte {
	return (b[i/8] >> (7 - i%8)) & 1
}

// DrawBounds draws a bounding box in the given color. Useful for debugging.
func (rs *Renderer) DrawBounds(bb math32.Box2, clr color.Color) {
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
