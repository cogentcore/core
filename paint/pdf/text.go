// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package pdf

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
)

// Text renders text to the canvas using a transformation matrix,
// (the translation component specifies the starting offset)
func (r *PDF) Text(style *styles.Paint, m math32.Matrix2, pos math32.Vector2, lns *shaped.Lines) {
	mt := m.Mul(math32.Translate2D(pos.X, pos.Y))
	r.w.PushTransform(mt)
	off := lns.Offset
	clr := colors.Uniform(lns.Color)
	runes := lns.Source.Join()
	for li := range lns.Lines {
		ln := &lns.Lines[li]
		r.textLine(style, m, ln, lns, runes, clr, off)
	}
	r.links(lns, m, pos)
	r.w.PopStack()
}

// TextLine rasterizes the given shaped.Line.
func (r *PDF) textLine(style *styles.Paint, m math32.Matrix2, ln *shaped.Line, lns *shaped.Lines, runes []rune, clr image.Image, off math32.Vector2) {
	start := off.Add(ln.Offset)
	off = start
	for ri := range ln.Runs {
		run := ln.Runs[ri].(*shapedgt.Run)
		r.textRunRegions(m, run, ln, lns, off)
		if run.Direction.IsVertical() {
			off.Y += run.Advance()
		} else {
			off.X += run.Advance()
		}
	}
	off = start
	for ri := range ln.Runs {
		run := ln.Runs[ri].(*shapedgt.Run)
		r.textRun(style, m, run, ln, lns, runes, clr, off)
		if run.Direction.IsVertical() {
			off.Y += run.Advance()
		} else {
			off.X += run.Advance()
		}
	}
}

// textRegionFill fills given regions within run with given fill color.
func (r *PDF) textRegionFill(m math32.Matrix2, run *shapedgt.Run, off math32.Vector2, fill image.Image, ranges []textpos.Range) {
	if fill == nil {
		return
	}
	idm := math32.Identity2()
	for _, sel := range ranges {
		rsel := sel.Intersect(run.Runes())
		if rsel.Len() == 0 {
			continue
		}
		fi := run.FirstGlyphAt(rsel.Start)
		li := run.LastGlyphAt(rsel.End - 1)
		if fi >= 0 && li >= fi {
			sbb := run.GlyphRegionBounds(fi, li).Canon()
			r.FillBox(idm, sbb.Translate(off), fill)
		}
	}
}

// textRunRegions draws region fills for given run.
func (r *PDF) textRunRegions(m math32.Matrix2, run *shapedgt.Run, ln *shaped.Line, lns *shaped.Lines, off math32.Vector2) {
	idm := math32.Identity2()

	// dir := run.Direction
	rbb := run.MaxBounds.Translate(off)
	if run.Background != nil {
		r.FillBox(idm, rbb, run.Background)
	}
	r.textRegionFill(m, run, off, lns.SelectionColor, ln.Selections)
	r.textRegionFill(m, run, off, lns.HighlightColor, ln.Highlights)
}

// textRun rasterizes the given text run into the output image using the
// font face set in the shaping.
// The text will be drawn starting at the start pixel position.
func (r *PDF) textRun(style *styles.Paint, m math32.Matrix2, run *shapedgt.Run, ln *shaped.Line, lns *shaped.Lines, runes []rune, clr image.Image, off math32.Vector2) {
	// dir := run.Direction
	region := run.Runes()
	offTrans := math32.Translate2D(off.X, off.Y)
	rbb := run.MaxBounds.Translate(off)
	fill := clr
	if run.FillColor != nil {
		fill = run.FillColor
	}
	fsz := math32.FromFixed(run.Size)
	lineW := max(fsz/16, 1) // 1 at 16, bigger if biggerr
	if run.Math.Path != nil {
		r.w.PushTransform(offTrans)
		r.Path(*run.Math.Path, style, math32.Identity2())
		r.w.PopStack()
		return
	}

	idm := math32.Identity2()
	if run.Decoration.HasFlag(rich.Underline) || run.Decoration.HasFlag(rich.DottedUnderline) {
		dash := []float32{2, 2}
		if run.Decoration.HasFlag(rich.Underline) {
			dash = nil
		}
		if run.Direction.IsVertical() {
		} else {
			dec := off.Y + 3
			r.strokeTextLine(idm, math32.Vec2(rbb.Min.X, dec), math32.Vec2(rbb.Max.X, dec), lineW, fill, dash)
		}
	}
	if run.Decoration.HasFlag(rich.Overline) {
		if run.Direction.IsVertical() {
		} else {
			dec := off.Y - 0.7*rbb.Size().Y
			r.strokeTextLine(idm, math32.Vec2(rbb.Min.X, dec), math32.Vec2(rbb.Max.X, dec), lineW, fill, nil)
		}
	}

	r.w.StartTextObject(offTrans)
	r.setTextStyle(&run.Font, style, fill, run.StrokeColor, math32.FromFixed(run.Size), lns.LineHeight)
	raw := string(runes[region.Start:region.End])
	r.w.WriteText(raw)
	r.w.EndTextObject()

	if run.Decoration.HasFlag(rich.LineThrough) {
		if run.Direction.IsVertical() {
		} else {
			dec := off.Y - 0.2*rbb.Size().Y
			r.strokeTextLine(idm, math32.Vec2(rbb.Min.X, dec), math32.Vec2(rbb.Max.X, dec), lineW, fill, nil)
		}
	}
}

// setTextStyle applies the given styles.
func (r *PDF) setTextStyle(fnt *text.Font, style *styles.Paint, fill, stroke image.Image, size, lineHeight float32) {
	tsty := &style.Text
	sty := fnt.Style(tsty)
	r.w.SetFont(sty, tsty)
	mode := 0
	if stroke != nil {
		sc := styles.Stroke{}
		sc.Defaults()
		sc.Color = stroke
		r.w.SetStroke(&sc)
	}
	if fill != nil {
		fc := styles.Fill{}
		fc.Defaults()
		fc.Color = fill
		fc.Opacity = 1
		r.w.SetFill(&fc)
		if stroke != nil {
			mode = 2
		}
	} else {
		if stroke != nil {
			mode = 1
		}
	}
	r.w.SetTextRenderMode(mode)
}

// strokeTextLine strokes a line for text decoration.
func (r *PDF) strokeTextLine(m math32.Matrix2, sp, ep math32.Vector2, width float32, clr image.Image, dash []float32) {
	sty := styles.NewPaint()
	sty.Fill.Color = nil
	sty.Stroke.Width.Dots = width
	sty.Stroke.Color = clr
	sty.Stroke.Dashes = dash
	p := ppath.New().Line(sp.X, sp.Y, ep.X, ep.Y)
	r.Path(*p, sty, m)
}

// FillBox fills a box in the given color.
func (r *PDF) FillBox(m math32.Matrix2, bb math32.Box2, clr image.Image) {
	sty := styles.NewPaint()
	sty.Stroke.Color = nil
	sty.Fill.Color = clr
	sz := bb.Size()
	p := ppath.New().Rectangle(bb.Min.X, bb.Min.Y, sz.X, sz.Y)
	r.Path(*p, sty, m)
}

func (r *PDF) links(lns *shaped.Lines, m math32.Matrix2, pos math32.Vector2) {
	lks := lns.GetLinks()
	for _, lk := range lks {
		// note: link coordinates are in default user space, not current transform.
		srb := lns.RuneBounds(lk.Range.Start)
		erb := lns.RuneBounds(lk.Range.End)
		if erb.Max.X > srb.Max.X {
			srb.Max.X = erb.Max.X
		}
		rb := srb.Translate(pos)
		rb = rb.MulMatrix2(m)
		r.w.AddURIAction(lk.URL, rb)
	}
}
