// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package htmlcanvas

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
	"cogentcore.org/core/text/shaped/shapers/shapedjs"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
)

// RenderText rasterizes the given Text
func (rs *Renderer) RenderText(txt *render.Text) {
	rs.TextLines(&txt.Context, txt.Text, txt.Position)
}

// TextLines rasterizes the given shaped.Lines.
// The text will be drawn starting at the start pixel position, which specifies the
// left baseline location of the first text item..
func (rs *Renderer) TextLines(ctx *render.Context, lns *shaped.Lines, pos math32.Vector2) {
	rs.setTransform(ctx)
	off := pos.Add(lns.Offset)
	clr := colors.Uniform(lns.Color)
	runes := lns.Source.Join()
	for li := range lns.Lines {
		ln := &lns.Lines[li]
		rs.TextLine(ctx, ln, lns, runes, clr, off)
	}
}

// TextLine rasterizes the given shaped.Line.
func (rs *Renderer) TextLine(ctx *render.Context, ln *shaped.Line, lns *shaped.Lines, runes []rune, clr image.Image, off math32.Vector2) {
	start := off.Add(ln.Offset)
	off = start
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
		rs.TextRun(ctx, run, ln, lns, runes, clr, off)
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
			sbb := run.GlyphRegionBounds(fi, li).Canon()
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
func (rs *Renderer) TextRun(ctx *render.Context, run *shapedgt.Run, ln *shaped.Line, lns *shaped.Lines, runes []rune, clr image.Image, off math32.Vector2) {
	// dir := run.Direction
	region := run.Runes()
	rbb := run.MaxBounds.Translate(off)
	fill := clr
	if run.FillColor != nil {
		fill = run.FillColor
	}
	fsz := math32.FromFixed(run.Size)
	lineW := max(fsz/16, 1) // 1 at 16, bigger if biggerr
	if run.Math.Path != nil {
		m := ctx.Transform
		ctx.Transform.X0 += off.X
		ctx.Transform.Y0 += off.Y
		rs.setTransform(ctx)
		rs.writePath(run.Math.Path)
		rs.setFill(fill)
		rs.ctx.Call("fill", "nonzero")
		ctx.Transform = m
		rs.setTransform(ctx)
		return
	}

	if run.Decoration.HasFlag(rich.Underline) || run.Decoration.HasFlag(rich.DottedUnderline) {
		dash := []float32{2, 2}
		if run.Decoration.HasFlag(rich.Underline) {
			dash = nil
		}
		if run.Direction.IsVertical() {

		} else {
			dec := off.Y + 3
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

	rs.applyTextStyle(&run.Font, &ctx.Style.Text, fill, run.StrokeColor, math32.FromFixed(run.Size), lns.LineHeight)

	raw := runes[region.Start:region.End]
	sraw := string(raw)
	if fill != nil {
		rs.ctx.Call("fillText", sraw, off.X, off.Y)
	}
	if run.StrokeColor != nil {
		rs.ctx.Call("strokeText", sraw, off.X, off.Y)
	}
	if run.Decoration.HasFlag(rich.LineThrough) {
		if run.Direction.IsVertical() {
		} else {
			dec := off.Y - 0.2*rbb.Size().Y
			rs.StrokeTextLine(ctx, math32.Vec2(rbb.Min.X, dec), math32.Vec2(rbb.Max.X, dec), lineW, fill, nil)
		}
	}
}

// applyTextStyle applies the given styles to the HTML canvas context.
func (rs *Renderer) applyTextStyle(fnt *text.Font, tsty *text.Style, fill, stroke image.Image, size, lineHeight float32) {
	shapedjs.SetFontStyle(rs.ctx, fnt, tsty, lineHeight)

	rs.ctx.Set("fillStyle", rs.imageToStyle(fill))
	rs.ctx.Set("strokeStyle", rs.imageToStyle(stroke))
	// note: text decorations not available in canvas
}

// StrokeTextLine strokes a line for text decoration.
func (rs *Renderer) StrokeTextLine(ctx *render.Context, sp, ep math32.Vector2, width float32, clr image.Image, dash []float32) {
	stroke := &styles.Stroke{}
	stroke.Defaults()
	stroke.Width.Dots = width
	stroke.Color = clr
	stroke.Dashes = dash
	rs.setStroke(stroke)
	rs.ctx.Call("beginPath")
	rs.ctx.Call("moveTo", sp.X, sp.Y)
	rs.ctx.Call("lineTo", ep.X, ep.Y)
	rs.ctx.Call("stroke")
}

// FillBounds fills a bounding box in the given color.
func (rs *Renderer) FillBounds(ctx *render.Context, bb math32.Box2, clr image.Image) {
	rs.setFill(clr)
	rs.ctx.Call("beginPath")
	rs.ctx.Call("moveTo", bb.Min.X, bb.Min.Y)
	rs.ctx.Call("lineTo", bb.Max.X, bb.Min.Y)
	rs.ctx.Call("lineTo", bb.Max.X, bb.Max.Y)
	rs.ctx.Call("lineTo", bb.Min.X, bb.Max.Y)
	rs.ctx.Call("closePath")
	rs.ctx.Call("fill")
}
