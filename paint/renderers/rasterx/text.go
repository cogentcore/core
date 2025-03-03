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
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/shaped/shapedgt"
	"cogentcore.org/core/text/textpos"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/shaping"
	_ "golang.org/x/image/tiff" // load image formats for users of the API
)

// RenderText rasterizes the given Text
func (rs *Renderer) RenderText(txt *render.Text) {
	// pr := profile.Start("RenderText")
	rs.TextLines(&txt.Context, txt.Text, txt.Position)
	// pr.End()
}

// TextLines rasterizes the given shaped.Lines.
// The text will be drawn starting at the start pixel position, which specifies the
// left baseline location of the first text item..
func (rs *Renderer) TextLines(ctx *render.Context, lns *shaped.Lines, pos math32.Vector2) {
	m := ctx.Transform
	identity := m == math32.Identity2()
	off := pos.Add(lns.Offset)
	rs.Scanner.SetClip(ctx.Bounds.Rect.ToRect())
	// tbb := lns.Bounds.Translate(off)
	// rs.StrokeBounds(ctx, tbb, colors.Red)
	clr := colors.Uniform(lns.Color)
	for li := range lns.Lines {
		ln := &lns.Lines[li]
		rs.TextLine(ctx, ln, lns, clr, off, identity)
	}
}

// TextLine rasterizes the given shaped.Line.
func (rs *Renderer) TextLine(ctx *render.Context, ln *shaped.Line, lns *shaped.Lines, clr image.Image, off math32.Vector2, identity bool) {
	start := off.Add(ln.Offset)
	off = start
	// tbb := ln.Bounds.Translate(off)
	// rs.StrokeBounds(ctx, tbb, colors.Blue)
	for ri := range ln.Runs {
		run := ln.Runs[ri].(*shapedgt.Run)
		rs.TextRunRegions(ctx, run, ln, lns, off)
		if run.Direction.IsVertical() {
			off.Y += run.Advance()
		} else {
			off.X += run.Advance()
		}
	}
	off = start
	for ri := range ln.Runs {
		run := ln.Runs[ri].(*shapedgt.Run)
		rs.TextRun(ctx, run, ln, lns, clr, off, identity)
		if run.Direction.IsVertical() {
			off.Y += run.Advance()
		} else {
			off.X += run.Advance()
		}
	}
}

// TextRegionFill fills given regions within run with given fill color.
func (rs *Renderer) TextRegionFill(ctx *render.Context, run *shapedgt.Run, off math32.Vector2, fill image.Image, ranges []textpos.Range) {
	if fill == nil {
		return
	}
	for _, sel := range ranges {
		rsel := sel.Intersect(run.Runes())
		if rsel.Len() == 0 {
			continue
		}
		fi := run.FirstGlyphAt(rsel.Start)
		li := run.LastGlyphAt(rsel.End - 1)
		if fi >= 0 && li >= fi {
			sbb := run.GlyphRegionBounds(fi, li)
			rs.FillBounds(ctx, sbb.Translate(off), fill)
		}
	}
}

// TextRunRegions draws region fills for given run.
func (rs *Renderer) TextRunRegions(ctx *render.Context, run *shapedgt.Run, ln *shaped.Line, lns *shaped.Lines, off math32.Vector2) {
	// dir := run.Direction
	rbb := run.MaxBounds.Translate(off)
	if run.Background != nil {
		rs.FillBounds(ctx, rbb, run.Background)
	}
	rs.TextRegionFill(ctx, run, off, lns.SelectionColor, ln.Selections)
	rs.TextRegionFill(ctx, run, off, lns.HighlightColor, ln.Highlights)
}

// TextRun rasterizes the given text run into the output image using the
// font face set in the shaping.
// The text will be drawn starting at the start pixel position.
func (rs *Renderer) TextRun(ctx *render.Context, run *shapedgt.Run, ln *shaped.Line, lns *shaped.Lines, clr image.Image, off math32.Vector2, identity bool) {
	// dir := run.Direction
	rbb := run.MaxBounds.Translate(off)
	fill := clr
	if run.FillColor != nil {
		fill = run.FillColor
	}
	stroke := run.StrokeColor
	fsz := math32.FromFixed(run.Size)
	lineW := max(fsz/16, 1) // 1 at 16, bigger if biggerr

	if run.Decoration.HasFlag(rich.Underline) || run.Decoration.HasFlag(rich.DottedUnderline) {
		dash := []float32{2, 2}
		if run.Decoration.HasFlag(rich.Underline) {
			dash = nil
		}
		if run.Direction.IsVertical() {

		} else {
			dec := off.Y + 2
			rs.StrokeTextLine(ctx, math32.Vec2(rbb.Min.X, dec), math32.Vec2(rbb.Max.X, dec), lineW, fill, dash)
		}
	}
	if run.Decoration.HasFlag(rich.Overline) {
		if run.Direction.IsVertical() {
		} else {
			dec := off.Y - 0.7*rbb.Size().Y
			rs.StrokeTextLine(ctx, math32.Vec2(rbb.Min.X, dec), math32.Vec2(rbb.Max.X, dec), lineW, fill, nil)
		}
	}

	for gi := range run.Glyphs {
		g := &run.Glyphs[gi]
		pos := off.Add(math32.Vec2(math32.FromFixed(g.XOffset), -math32.FromFixed(g.YOffset)))
		// top := yPos - math32.FromFixed(g.YBearing)
		// bottom := top - math32.FromFixed(g.Height)
		// right := xPos + math32.FromFixed(g.Width)
		// rect := image.Rect(int(xPos)-4, int(top)-4, int(right)+4, int(bottom)+4) // don't cut off
		bb := run.GlyphBoundsBox(g).Translate(off)
		// rs.StrokeBounds(ctx, bb, colors.Yellow)

		data := run.Face.GlyphData(g.GlyphID)
		switch format := data.(type) {
		case font.GlyphOutline:
			rs.GlyphOutline(ctx, run, g, format, fill, stroke, bb, pos, identity)
		case font.GlyphBitmap:
			fmt.Println("bitmap")
			rs.GlyphBitmap(ctx, run, g, format, fill, stroke, bb, pos, identity)
		case font.GlyphSVG:
			fmt.Println("svg", format)
			// 	_ = rs.GlyphSVG(ctx, g, format, fill, stroke, bb, pos)
		}
		off.X += math32.FromFixed(g.XAdvance)
		off.Y -= math32.FromFixed(g.YAdvance)
	}

	if run.Decoration.HasFlag(rich.LineThrough) {
		if run.Direction.IsVertical() {
		} else {
			dec := off.Y - 0.2*rbb.Size().Y
			rs.StrokeTextLine(ctx, math32.Vec2(rbb.Min.X, dec), math32.Vec2(rbb.Max.X, dec), lineW, fill, nil)
		}
	}
}

func (rs *Renderer) GlyphOutline(ctx *render.Context, run *shapedgt.Run, g *shaping.Glyph, outline font.GlyphOutline, fill, stroke image.Image, bb math32.Box2, pos math32.Vector2, identity bool) {
	scale := math32.FromFixed(run.Size) / float32(run.Face.Upem())
	x := pos.X
	y := pos.Y
	if len(outline.Segments) == 0 {
		// fmt.Println("nil path:", g.GlyphID)
		return
	}

	if UseGlyphCache && identity && stroke == nil {
		mask, pi := theGlyphCache.Glyph(run.Face, g, outline, scale, pos)
		if mask != nil {
			rs.GlyphMask(ctx, run, g, fill, stroke, bb, pi, mask)
			return
		}
	}

	rs.Path.Clear()
	m := ctx.Transform
	for _, s := range outline.Segments {
		p0 := m.MulVector2AsPoint(math32.Vec2(s.Args[0].X*scale+x, -s.Args[0].Y*scale+y))
		switch s.Op {
		case opentype.SegmentOpMoveTo:
			rs.Path.Start(p0.ToFixed())
		case opentype.SegmentOpLineTo:
			rs.Path.Line(p0.ToFixed())
		case opentype.SegmentOpQuadTo:
			p1 := m.MulVector2AsPoint(math32.Vec2(s.Args[1].X*scale+x, -s.Args[1].Y*scale+y))
			rs.Path.QuadBezier(p0.ToFixed(), p1.ToFixed())
		case opentype.SegmentOpCubeTo:
			p1 := m.MulVector2AsPoint(math32.Vec2(s.Args[1].X*scale+x, -s.Args[1].Y*scale+y))
			p2 := m.MulVector2AsPoint(math32.Vec2(s.Args[2].X*scale+x, -s.Args[2].Y*scale+y))
			rs.Path.CubeBezier(p0.ToFixed(), p1.ToFixed(), p2.ToFixed())
		}
	}
	rs.Path.Stop(true)
	if fill != nil {
		rf := &rs.Raster.Filler
		rf.SetWinding(true)
		rf.SetColor(fill)
		rs.Path.AddTo(rf)
		rf.Draw()
		rf.Clear()
	}

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

func (rs *Renderer) GlyphMask(ctx *render.Context, run *shapedgt.Run, g *shaping.Glyph, fill, stroke image.Image, bb math32.Box2, pos image.Point, mask *image.Alpha) error {
	if g.XOffset != 0 || g.YOffset != 0 {
		fmt.Println("mask offset:", math32.FromFixed(g.XOffset), math32.FromFixed(g.YOffset), "bearing:", math32.FromFixed(g.XBearing), math32.FromFixed(g.YBearing))
	}
	pos.X += g.XOffset.Round()
	pos.Y += g.YOffset.Round()
	mbb := mask.Bounds()
	dbb := mbb.Add(pos)
	ibb := dbb.Intersect(ctx.Bounds.Rect.ToRect())
	if ibb == (image.Rectangle{}) {
		return nil
	}
	mp := ibb.Min.Sub(dbb.Min)
	draw.DrawMask(rs.image, ibb, fill, image.Point{}, mask, mp, draw.Over)
	return nil
}

func (rs *Renderer) GlyphBitmap(ctx *render.Context, run *shapedgt.Run, g *shaping.Glyph, bitmap font.GlyphBitmap, fill, stroke image.Image, bb math32.Box2, pos math32.Vector2, identity bool) error {
	// scaled glyph rect content
	// todo: this needs serious work to function with transforms etc.
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
		rs.GlyphOutline(ctx, run, g, *bitmap.Outline, fill, stroke, bb, pos, identity)
	}
	return nil
}

// bitAt returns the bit at the given index in the byte slice.
func bitAt(b []byte, i int) byte {
	return (b[i/8] >> (7 - i%8)) & 1
}

// StrokeBounds strokes a bounding box in the given color. Useful for debugging.
func (rs *Renderer) StrokeBounds(ctx *render.Context, bb math32.Box2, clr color.Color) {
	rs.Raster.SetStroke(
		math32.ToFixed(1),
		math32.ToFixed(10),
		ButtCap, nil, nil, Miter,
		nil, 0)
	rs.Raster.SetColor(colors.Uniform(clr))
	m := ctx.Transform
	rs.Raster.Start(m.MulVector2AsPoint(math32.Vec2(bb.Min.X, bb.Min.Y)).ToFixed())
	rs.Raster.Line(m.MulVector2AsPoint(math32.Vec2(bb.Max.X, bb.Min.Y)).ToFixed())
	rs.Raster.Line(m.MulVector2AsPoint(math32.Vec2(bb.Max.X, bb.Max.Y)).ToFixed())
	rs.Raster.Line(m.MulVector2AsPoint(math32.Vec2(bb.Min.X, bb.Max.Y)).ToFixed())
	rs.Raster.Stop(true)
	rs.Raster.Draw()
	rs.Raster.Clear()
}

// StrokeTextLine strokes a line for text decoration.
func (rs *Renderer) StrokeTextLine(ctx *render.Context, sp, ep math32.Vector2, width float32, clr image.Image, dash []float32) {
	m := ctx.Transform
	sp = m.MulVector2AsPoint(sp)
	ep = m.MulVector2AsPoint(ep)
	width *= MeanScale(m)
	rs.Raster.SetStroke(
		math32.ToFixed(width),
		math32.ToFixed(10),
		ButtCap, nil, nil, Miter,
		dash, 0)
	rs.Raster.SetColor(clr)
	rs.Raster.Start(sp.ToFixed())
	rs.Raster.Line(ep.ToFixed())
	rs.Raster.Stop(false)
	rs.Raster.Draw()
	rs.Raster.Clear()
}

// FillBounds fills a bounding box in the given color.
func (rs *Renderer) FillBounds(ctx *render.Context, bb math32.Box2, clr image.Image) {
	rf := &rs.Raster.Filler
	rf.SetColor(clr)
	m := ctx.Transform
	rf.Start(m.MulVector2AsPoint(math32.Vec2(bb.Min.X, bb.Min.Y)).ToFixed())
	rf.Line(m.MulVector2AsPoint(math32.Vec2(bb.Max.X, bb.Min.Y)).ToFixed())
	rf.Line(m.MulVector2AsPoint(math32.Vec2(bb.Max.X, bb.Max.Y)).ToFixed())
	rf.Line(m.MulVector2AsPoint(math32.Vec2(bb.Min.X, bb.Max.Y)).ToFixed())
	rf.Stop(true)
	rf.Draw()
	rf.Clear()
}
