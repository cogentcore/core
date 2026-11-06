// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"strings"
	"unicode"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/htmltext"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
)

// todo: needs some work to get up to spec:
// https://developer.mozilla.org/en-US/docs/Web/SVG/Reference/Element/tspan
// dx, dy need to be specifiable as relative offsets from parent text, not just glyph
// relative offsets. Also, the literal parsing of example in link of an inline <tspan>
// element within a broader text shows that a different kind of parsing
// structure is required.

// Text renders SVG text, handling both text and tspan elements.
// tspan is nested under a parent text, where text has empty Text string.
// There is no line wrapping on SVG Text: every span is a separate line.
type Text struct {
	NodeBase

	// position of the left, baseline of the text
	Pos math32.Vector2 `xml:"{x,y}"`

	// text string to render
	Text string `xml:"text"`

	// render version of text
	TextShaped *shaped.Lines `xml:"-" json:"-" copier:"-"`

	// character positions along X axis, if specified
	CharPosX []float32

	// character positions along Y axis, if specified
	CharPosY []float32

	// character delta-positions along X axis, if specified
	CharPosDX []float32

	// character delta-positions along Y axis, if specified
	CharPosDY []float32

	// character rotations, if specified
	CharRots []float32

	// author's computed text length, if specified -- we attempt to match
	TextLength float32

	// in attempting to match TextLength, should we adjust glyphs in addition to spacing?
	AdjustGlyphs bool

	// hasX and hasY are whether the x and y of the text are specified,
	// which for a span (tspan) is an absolute position (absPos), which
	// otherwise follows the span before it.
	hasX, hasY bool

	// absPos is the absolute position of a span, for hasX and hasY.
	absPos math32.Vector2

	// laidOut is whether the span is laid out by its parent text
	// (which aligns its spans by its text-anchor).
	laidOut bool
}

// collapseSpace returns the given text with its runs of white space
// collapsed to a single space, as SVG text is by default.
func collapseSpace(s string) string {
	var sb strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if space {
			sb.WriteByte(' ')
			space = false
		}
		sb.WriteRune(r)
	}
	if space {
		sb.WriteByte(' ')
	}
	return sb.String()
}

// trimSpans trims the space at the start and end of the text (of the
// first and last of its spans, if it has them), as SVG text does.
func (g *Text) trimSpans() {
	if g.NumChildren() == 0 {
		g.Text = strings.TrimSpace(g.Text)
		return
	}
	if first, ok := g.Child(0).(*Text); ok {
		first.Text = strings.TrimLeft(first.Text, " ")
	}
	if last, ok := g.Child(g.NumChildren() - 1).(*Text); ok {
		last.Text = strings.TrimRight(last.Text, " ")
	}
}

// layoutSpans positions the spans of a parent text: each after the one
// before it (from the position of the text), unless it has an absolute
// position, with its dx and dy offsets, and the spans of each chunk (from
// an absolute position) aligned as a whole by the text-anchor of the span
// that starts it.
func (g *Text) layoutSpans(sv *SVG) {
	pen := g.Pos
	var chunk []*Text
	chunkStart := pen.X
	flush := func() {
		if len(chunk) == 0 {
			return
		}
		w := pen.X - chunkStart
		shift := float32(0)
		switch chunk[0].Paint.Text.Align { // that of the span that starts the chunk

		case text.Center:
			shift = -w / 2
		case text.End:
			shift = -w
		}
		for _, kt := range chunk {
			kt.Pos.X += shift
		}
		chunk = chunk[:0]
	}
	for _, k := range g.Children {
		kt, ok := k.(*Text)
		if !ok {
			continue
		}
		if kt.hasX {
			flush()
			pen.X = kt.absPos.X
			chunkStart = pen.X
		}
		if kt.hasY {
			pen.Y = kt.absPos.Y
		}
		if len(kt.CharPosDX) > 0 {
			pen.X += kt.CharPosDX[0]
		}
		if len(kt.CharPosDY) > 0 {
			pen.Y += kt.CharPosDY[0]
		}
		kt.Pos = pen
		kt.laidOut = true
		kt.LocalBBox(sv)
		if kt.TextShaped != nil {
			pen.X += kt.TextShaped.Bounds.Size().X
		}
		chunk = append(chunk, kt)
	}
	flush()
}

func (g *Text) SVGName() string {
	if len(g.Text) == 0 {
		return "text"
	}
	return "tspan"
}

// IsParText returns true if this element serves as a parent text element
// to tspan elements within it.  This is true if NumChildren() > 0 and
// Text == ""
func (g *Text) IsParText() bool {
	return g.NumChildren() > 0 && g.Text == ""
}

func (g *Text) SetNodePos(pos math32.Vector2) {
	g.Pos = pos
	for _, kii := range g.Children {
		kt := kii.(*Text)
		kt.Pos = g.Paint.Transform.MulPoint(pos)
	}
}

// LocalBBox does full text layout, but no transforms
func (g *Text) LocalBBox(sv *SVG) math32.Box2 {
	if g.Text == "" {
		return math32.Box2{}
	}
	pc := &g.Paint
	fs := pc.Font
	if pc.Fill.Color != nil {
		fs.SetFillColor(colors.ApplyOpacity(colors.ToUniform(pc.Fill.Color), pc.Fill.Opacity))
	}
	tx, _ := htmltext.HTMLToRich([]byte(g.Text), &fs, nil)
	// fmt.Println(tx)
	sz := math32.Vec2(10000, 10000) // no wrapping!!
	g.TextShaped = sv.TextShaper.WrapLines(tx, &fs, &pc.Text, sz)
	if len(g.TextShaped.Lines) > 0 {
		baseOff := g.TextShaped.Lines[0].Offset
		g.TextShaped.StartAtBaseline() // remove top-left offset
		return g.TextShaped.Bounds.Translate(g.Pos.Sub(baseOff))
	}
	return g.TextShaped.Bounds

	// fmt.Println("baseoff:", baseOff)
	// fmt.Println(pc.Text.FontSize, pc.Text.FontSize.Dots)

	// todo: align styling only affects multi-line text and is about how tspan is arranged within
	// the overall text block.

	/*
		if len(g.CharPosX) > 0 {
			mx := min(len(g.CharPosX), len(sr.Render))
			for i := 0; i < mx; i++ {
				sr.Render[i].RelPos.X = g.CharPosX[i]
			}
		}
		if len(g.CharPosY) > 0 {
			mx := min(len(g.CharPosY), len(sr.Render))
			for i := 0; i < mx; i++ {
				sr.Render[i].RelPos.Y = g.CharPosY[i]
			}
		}
		if len(g.CharPosDX) > 0 {
			mx := min(len(g.CharPosDX), len(sr.Render))
			for i := 0; i < mx; i++ {
				if i > 0 {
					sr.Render[i].RelPos.X = sr.Render[i-1].RelPos.X + g.CharPosDX[i]
				} else {
					sr.Render[i].RelPos.X = g.CharPosDX[i] // todo: not sure this is right
				}
			}
		}
		if len(g.CharPosDY) > 0 {
			mx := min(len(g.CharPosDY), len(sr.Render))
			for i := 0; i < mx; i++ {
				if i > 0 {
					sr.Render[i].RelPos.Y = sr.Render[i-1].RelPos.Y + g.CharPosDY[i]
				} else {
					sr.Render[i].RelPos.Y = g.CharPosDY[i] // todo: not sure this is right
				}
			}
		}
	*/
	// todo: TextLength, AdjustGlyphs -- also svg2 at least supports word wrapping!
	// g.TextShaped.UpdateBBox()
}

func (g *Text) BBoxes(sv *SVG, parTransform math32.Matrix2) {
	if g.IsParText() {
		g.layoutSpans(sv)
		g.BBoxesFromChildren(sv, parTransform)
		return
	}
	xf := parTransform.Mul(g.Paint.Transform)
	ni := g.This.(Node)
	lbb := ni.LocalBBox(sv)
	g.BBox = lbb.MulMatrix2(xf)
	g.VisBBox = sv.Geom.Box2().Intersect(g.BBox)
}

func (g *Text) Render(sv *SVG) {
	if g.IsParText() {
		if !g.PushContext(sv) {
			return
		}
		g.setLayer(sv, math32.Identity2())
		g.preparePatterns(sv, math32.Identity2())
		pc := g.Painter(sv)
		g.RenderChildren(sv)
		pc.PopContext()
		return
	}
	if !g.IsVisible(sv) {
		return
	}
	if len(g.Text) > 0 {
		g.RenderText(sv)
	}
}

func (g *Text) RenderText(sv *SVG) {
	// note: transform is managed entirely in the render side function!
	pc := g.Painter(sv)
	pos := g.Pos
	bsz := g.TextShaped.Bounds.Size()
	if g.laidOut { // aligned by the parent text
	} else if pc.Text.Align == text.Center {
		pos.X -= bsz.X * .5
	} else if pc.Text.Align == text.End {
		pos.X -= bsz.X
	}
	pc.DrawText(g.TextShaped, pos)
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Text) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	rot := xf.ExtractRot()
	scx, scy := xf.ExtractScale()
	if rot != 0 || scx != 1 || scy != 1 || g.IsParText() {
		// note: par text requires transform b/c not saving children pos
		g.Paint.Transform.SetMul(xf)
		g.SetTransformProperty()
	} else {
		g.Pos = xf.MulPoint(g.Pos)
		g.GradientApplyTransform(sv, xf)
	}
}
