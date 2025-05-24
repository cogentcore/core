// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/htmltext"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
)

// Text renders SVG text, handling both text and tspan elements.
// tspan is nested under a parent text, where text has empty Text string.
type Text struct {
	NodeBase

	// position of the left, baseline of the text
	Pos math32.Vector2 `xml:"{x,y}"`

	// width of text to render if using word-wrapping
	Width float32 `xml:"width"`

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

func (g *Text) SetNodeSize(sz math32.Vector2) {
	g.Width = sz.X
	scx, _ := g.Paint.Transform.ExtractScale()
	for _, kii := range g.Children {
		kt := kii.(*Text)
		kt.Width = g.Width * scx
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
	sz := math32.Vec2(10000, 10000)
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
	if rot != 0 || !g.Paint.Transform.IsIdentity() {
		g.Paint.Transform.SetMul(xf)
		g.SetProperty("transform", g.Paint.Transform.String())
	} else {
		if g.IsParText() {
			for _, kii := range g.Children {
				kt := kii.(*Text)
				kt.ApplyTransform(sv, xf)
			}
		} else {
			g.Pos = xf.MulVector2AsPoint(g.Pos)
			scx, _ := xf.ExtractScale()
			g.Width *= scx
			g.GradientApplyTransform(sv, xf)
		}
	}
}

// ApplyDeltaTransform applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Text) ApplyDeltaTransform(sv *SVG, trans math32.Vector2, scale math32.Vector2, rot float32, pt math32.Vector2) {
	crot := g.Paint.Transform.ExtractRot()
	if rot != 0 || crot != 0 {
		xf, lpt := g.DeltaTransform(trans, scale, rot, pt, false) // exclude self
		g.Paint.Transform.SetMulCenter(xf, lpt)
		g.SetProperty("transform", g.Paint.Transform.String())
	} else {
		if g.IsParText() {
			// translation transform
			xft, lptt := g.DeltaTransform(trans, scale, rot, pt, true) // include self when not a parent
			// transform transform
			xf, lpt := g.DeltaTransform(trans, scale, rot, pt, false)
			xf.X0 = 0 // negate translation effects
			xf.Y0 = 0
			g.Paint.Transform.SetMulCenter(xf, lpt)
			g.SetProperty("transform", g.Paint.Transform.String())

			g.Pos = xft.MulVector2AsPointCenter(g.Pos, lptt)
			scx, _ := xft.ExtractScale()
			g.Width *= scx
			for _, kii := range g.Children {
				kt := kii.(*Text)
				kt.Pos = xft.MulVector2AsPointCenter(kt.Pos, lptt)
				kt.Width *= scx
			}
		} else {
			xf, lpt := g.DeltaTransform(trans, scale, rot, pt, true) // include self when not a parent
			g.Pos = xf.MulVector2AsPointCenter(g.Pos, lpt)
			scx, _ := xf.ExtractScale()
			g.Width *= scx
		}
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Text) WriteGeom(sv *SVG, dat *[]float32) {
	if g.IsParText() {
		npt := 9 + g.NumChildren()*3
		*dat = slicesx.SetLength(*dat, npt)
		(*dat)[0] = g.Pos.X
		(*dat)[1] = g.Pos.Y
		(*dat)[2] = g.Width
		g.WriteTransform(*dat, 3)
		for i, kii := range g.Children {
			kt := kii.(*Text)
			off := 9 + i*3
			(*dat)[off+0] = kt.Pos.X
			(*dat)[off+1] = kt.Pos.Y
			(*dat)[off+2] = kt.Width
		}
	} else {
		*dat = slicesx.SetLength(*dat, 3+6)
		(*dat)[0] = g.Pos.X
		(*dat)[1] = g.Pos.Y
		(*dat)[2] = g.Width
		g.WriteTransform(*dat, 3)
	}
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Text) ReadGeom(sv *SVG, dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Width = dat[2]
	g.ReadTransform(dat, 3)
	if g.IsParText() {
		for i, kii := range g.Children {
			kt := kii.(*Text)
			off := 9 + i*3
			kt.Pos.X = dat[off+0]
			kt.Pos.Y = dat[off+1]
			kt.Width = dat[off+2]
		}
	}
}
