// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package pdf

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"golang.org/x/text/encoding/charmap"
)

// Text renders text to the canvas using a transformation matrix,
// (the translation component specifies the starting offset)
func (r *PDF) Text(style *styles.Paint, m math32.Matrix2, pos math32.Vector2, lns *shaped.Lines) {
	if lns.Anchor != "" {
		anc := lns.Anchor
		hidx := strings.Index(anc, ";Header ")
		if hidx >= 0 {
			if hidx > 0 {
				r.w.AddAnchor(anc[:hidx], pos)
				anc = anc[hidx:]
			}
			anc = anc[8:]
			level := errors.Log1(strconv.Atoi(anc[0:1]))
			anc = anc[2:]
			r.w.AddOutline(anc, level, pos.Y)
		} else {
			r.w.AddAnchor(lns.Anchor, pos)
		}
	}
	mt := m.Mul(math32.Translate2D(pos.X, pos.Y))
	r.w.PushTransform(mt)
	off := lns.Offset
	clr := colors.Uniform(lns.Color)
	runes := lns.Source.Join()
	for li := range lns.Lines {
		ln := &lns.Lines[li]
		r.textLine(style, m, li, ln, lns, runes, clr, off)
	}
	r.links(lns, m, pos)
	r.w.PopStack()
}

// TextLine rasterizes the given shaped.Line.
func (r *PDF) textLine(style *styles.Paint, m math32.Matrix2, li int, ln *shaped.Line, lns *shaped.Lines, runes []rune, clr image.Image, off math32.Vector2) {
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
		psty := *style
		psty.Stroke.Color = run.StrokeColor
		psty.Fill.Color = fill
		r.Path(*run.Math.Path, &psty, math32.B2FromFixed(run.Bounds()), math32.Identity2(), math32.Identity2())
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

func (r *PDF) setTextStrokeColor(clr image.Image) {
	sc := r.w.style().Stroke
	sc.Color = clr
	r.w.SetStroke(&sc)
}

func (r *PDF) setTextFillColor(clr image.Image) {
	fc := r.w.style().Fill
	fc.Color = clr
	r.w.SetFill(&fc, math32.Box2{}, math32.Identity2())
}

// setTextStyle applies the given styles.
func (r *PDF) setTextStyle(fnt *text.Font, style *styles.Paint, fill, stroke image.Image, size, lineHeight float32) {
	tsty := &style.Text
	sty := fnt.Style(tsty)
	r.w.SetFont(sty, tsty)
	mode := 0
	if stroke != nil {
		r.setTextStrokeColor(stroke)
	}
	if fill != nil {
		r.setTextFillColor(fill)
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
	r.Path(*p, sty, math32.Box2{}, m, m)
}

// FillBox fills a box in the given color.
func (r *PDF) FillBox(m math32.Matrix2, bb math32.Box2, clr image.Image) {
	sty := styles.NewPaint()
	sty.Stroke.Color = nil
	sty.Fill.Color = clr
	sz := bb.Size()
	p := ppath.New().Rectangle(bb.Min.X, bb.Min.Y, sz.X, sz.Y)
	r.Path(*p, sty, bb, m, m)
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
		r.w.AddLink(lk.URL, rb)
	}
}

// SetFont sets the font.
func (w *pdfPage) SetFont(sty *rich.Style, tsty *text.Style) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	size := tsty.FontHeight(sty) // * w.pdf.globalScale
	ref := w.pdf.getFont(sty, tsty)
	if _, ok := w.resources["Font"]; !ok {
		w.resources["Font"] = pdfDict{}
	} else {
		for name, fontRef := range w.resources["Font"].(pdfDict) {
			if ref == fontRef {
				fmt.Fprintf(w, " /%v %v Tf", name, dec(size))
				return nil
			}
		}
	}

	name := pdfName(fmt.Sprintf("F%d", len(w.resources["Font"].(pdfDict))))
	w.resources["Font"].(pdfDict)[name] = ref
	fmt.Fprintf(w, " /%v %v Tf", name, dec(size))
	return nil
}

// SetTextPosition sets the text offset position.
func (w *pdfPage) SetTextPosition(off math32.Vector2) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	do := off.Sub(w.textPosition)
	// and finally apply an offset from there, in reverse for Y
	fmt.Fprintf(w, " %v %v Td", dec(do.X), dec(-do.Y))
	w.textPosition = off
	return nil
}

// SetTextRenderMode sets the text rendering mode.
// 0 = fill text, 1 = stroke text, 2 = fill, then stroke.
// higher numbers support clip path.
func (w *pdfPage) SetTextRenderMode(mode int) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	fmt.Fprintf(w, " %d Tr", mode)
	w.textRenderMode = mode
	return nil
}

// SetTextCharSpace sets the text character spacing.
func (w *pdfPage) SetTextCharSpace(space float32) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	fmt.Fprintf(w, " %v Tc", dec(space))
	w.textCharSpace = space
	return nil
}

// StartTextObject starts a text object, adding to the graphics
// CTM transform matrix as given by the arg, and setting an inverting
// text transform, so text is rendered upright.
func (w *pdfPage) StartTextObject(m math32.Matrix2) error {
	if w.inTextObject {
		return errors.Log(errors.New("pdfWriter: already in text object"))
	}
	// set the graphics transform to m first
	w.PushTransform(m)
	fmt.Fprintf(w, " BT")
	// then apply an inversion text matrix
	tm := math32.Scale2D(1, -1)
	fmt.Fprintf(w, " %s Tm", mat2(tm))
	w.inTextObject = true
	w.textPosition = math32.Vector2{}
	return nil
}

// EndTextObject ends a text object.
func (w *pdfPage) EndTextObject() error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	fmt.Fprintf(w, " ET")
	w.PopStack()
	w.inTextObject = false
	return nil
}

// WriteText writes text using current text style.
func (w *pdfPage) WriteText(tx string) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	if len(tx) == 0 {
		return nil
	}

	first := true
	write := func(s string) {
		if first {
			fmt.Fprintf(w, "(")
			first = false
		} else {
			fmt.Fprintf(w, " (")
		}
		rs := []rune(s)
		for _, r := range rs {
			c, ok := charmap.Windows1252.EncodeRune(r)
			if !ok {
				if '\u2000' <= r && r <= '\u200A' {
					c = ' '
				}
			}
			switch c {
			case '\n':
				w.WriteByte('\\')
				w.WriteByte('n')
			case '\r':
				w.WriteByte('\\')
				w.WriteByte('r')
			case '\t':
				w.WriteByte('\\')
				w.WriteByte('t')
			case '\b':
				w.WriteByte('\\')
				w.WriteByte('b')
			case '\f':
				w.WriteByte('\\')
				w.WriteByte('f')
			case '\\', '(', ')':
				w.WriteByte('\\')
				w.WriteByte(c)
			default:
				w.WriteByte(c)
			}
		}
		fmt.Fprintf(w, ")")
	}

	fmt.Fprintf(w, "[")
	write(tx)
	fmt.Fprintf(w, "]TJ")
	return nil
}
