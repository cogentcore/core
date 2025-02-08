// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package htmlcanvas

import (
	"fmt"
	"image"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
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
	// rs.Scanner.SetClip(ctx.Bounds.Rect.ToRect())
	clr := colors.Uniform(lns.Color)
	runes := lns.Source.Join() // TODO: bad for performance with append
	for li := range lns.Lines {
		ln := &lns.Lines[li]
		rs.TextLine(ln, lns, runes, clr, start) // todo: start + offset
	}
}

// TextLine rasterizes the given shaped.Line.
func (rs *Renderer) TextLine(ln *shaped.Line, lns *shaped.Lines, runes []rune, clr image.Image, start math32.Vector2) {
	off := start.Add(ln.Offset)
	for ri := range ln.Runs {
		run := &ln.Runs[ri]
		rs.TextRun(run, ln, lns, runes, clr, off)
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
func (rs *Renderer) TextRun(run *shaped.Run, ln *shaped.Line, lns *shaped.Lines, runes []rune, clr image.Image, start math32.Vector2) {
	// todo: render strike-through
	// dir := run.Direction
	// rbb := run.MaxBounds.Translate(start)
	if run.Background != nil {
		// rs.FillBounds(rbb, run.Background) TODO
	}
	if len(ln.Selections) > 0 {
		for _, sel := range ln.Selections {
			rsel := sel.Intersect(run.Runes())
			if rsel.Len() > 0 {
				fi := run.FirstGlyphAt(rsel.Start)
				li := run.LastGlyphAt(rsel.End - 1)
				if fi >= 0 && li >= fi {
					// sbb := run.GlyphRegionBounds(fi, li) TODO
					// rs.FillBounds(sbb.Translate(start), lns.SelectionColor) TODO
				}
			}
		}
	}

	s := &textStyle{}
	s.fill = clr
	if run.FillColor != nil {
		s.fill = run.FillColor
	}
	s.stroke = run.StrokeColor
	s.slant = rich.SlantNormal
	s.variant = "normal"
	s.weight = rich.Normal
	s.stretch = rich.StretchNormal
	s.size = math32.FromFixed(run.Size)
	s.lineHeight = 1
	s.family = rich.SansSerif
	// TODO: implement all style values
	rs.applyTextStyle(s)

	region := run.Runes()
	raw := runes[region.Start:region.End]
	rs.ctx.Call("fillText", string(raw), start.X, start.Y) // TODO: also stroke
}

type textStyle struct {
	fill, stroke image.Image
	slant        rich.Slants
	variant      string
	weight       rich.Weights
	stretch      rich.Stretch
	size         float32
	lineHeight   float32
	family       rich.Family
}

// applyTextStyle applies the given styles to the HTML canvas context.
func (rs *Renderer) applyTextStyle(s *textStyle) {
	// See https://developer.mozilla.org/en-US/docs/Web/CSS/font

	parts := []string{s.slant.String(), s.variant, s.weight.String(), s.stretch.String(), fmt.Sprintf("%gpx/%g", s.size, s.lineHeight), s.family.String()}
	fmt.Println(parts)
	rs.ctx.Set("font", strings.Join(parts, " "))

	// TODO: use caching like in RenderPath?
	rs.ctx.Set("fillStyle", rs.imageToStyle(s.fill))
	rs.ctx.Set("strokeStyle", rs.imageToStyle(s.stroke))
}
