// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/htmltext"
	"cogentcore.org/core/text/rich"
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
		kt.Pos = g.Paint.Transform.MulVector2AsPoint(pos)
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
		fs.SetFillColor(colors.ToUniform(pc.Fill.Color))
	}
	tx, _ := htmltext.HTMLToRich([]byte(g.Text), &fs, nil)
	// fmt.Println(tx)
	sz := math32.Vec2(10000, 10000) // no wrapping!!
	g.TextShaped = sv.TextShaper.WrapLines(tx, &fs, &pc.Text, &rich.DefaultSettings, sz)
	baseOff := g.TextShaped.Lines[0].Offset
	g.TextShaped.StartAtBaseline() // remove top-left offset
	return g.TextShaped.Bounds.Translate(g.Pos.Sub(baseOff))

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
	if pc.Text.Align == text.Center {
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
		g.Pos = xf.MulVector2AsPoint(g.Pos)
		g.GradientApplyTransform(sv, xf)
	}
}
