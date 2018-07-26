// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki/kit"
)

// Text renders SVG text-- it handles both text and tspan elements (a tspan is
// just nested under a parent text)
type Text struct {
	SVGNodeBase
	Pos          gi.Vec2D      `xml:"{x,y}" desc:"position of the left, baseline of the text"`
	Width        float32       `xml:"width" desc:"width of text to render if using word-wrapping"`
	Text         string        `xml:"text" desc:"text string to render"`
	Render       gi.TextRender `xml:"-" json:"-" desc:"render version of text"`
	CharPosX     []float32     `desc:"character positions along X axis, if specified"`
	CharPosY     []float32     `desc:"character positions along Y axis, if specified"`
	CharPosDX    []float32     `desc:"character delta-positions along X axis, if specified"`
	CharPosDY    []float32     `desc:"character delta-positions along Y axis, if specified"`
	CharRots     []float32     `desc:"character rotations, if specified"`
	TextLength   float32       `desc:"author's computed text length, if specified -- we attempt to match"`
	AdjustGlyphs bool          `desc:"in attempting to match TextLength, should we adjust glyphs in addition to spacing?"`
}

var KiT_Text = kit.Types.AddType(&Text{}, nil)

func (g *Text) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	// todo: could be much more accurate..
	return g.Pnt.BoundingBox(rs, g.Pos.X, g.Pos.Y, g.Pos.X+g.Render.Size.X, g.Pos.Y+g.Render.Size.Y)
}

func (g *Text) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	if len(g.Text) > 0 {
		orgsz := pc.FontStyle.Size
		pos := rs.XForm.TransformPointVec2D(gi.Vec2D{g.Pos.X, g.Pos.Y})
		rot := rs.XForm.ExtractRot()
		scx, scy := rs.XForm.ExtractScale()
		scalex := scx / scy
		if scalex == 1 {
			scalex = 0
		}
		pc.FontStyle.LoadFont(&pc.UnContext, "") // use original size font
		if !pc.FillStyle.Color.IsNil() {
			pc.FontStyle.Color = pc.FillStyle.Color.Color
		}
		g.Render.SetString(g.Text, &pc.FontStyle, &pc.TextStyle, true, rot, scalex)
		g.Render.Size = g.Render.Size.Mul(gi.Vec2D{scx, scy})
		if gi.IsAlignMiddle(pc.TextStyle.Align) || pc.TextStyle.Anchor == gi.AnchorMiddle {
			pos.X -= g.Render.Size.X * .5
		} else if gi.IsAlignEnd(pc.TextStyle.Align) || pc.TextStyle.Anchor == gi.AnchorEnd {
			pos.X -= g.Render.Size.X
		}
		pc.FontStyle.Size = units.Value{orgsz.Val * scy, orgsz.Un, orgsz.Dots * scy} // rescale by y
		pc.FontStyle.LoadFont(&pc.UnContext, "")
		sr := &(g.Render.Spans[0])
		sr.Render[0].Face = pc.FontStyle.Face // upscale
		for i := range sr.Render {
			sr.Render[i].RelPos = rs.XForm.TransformVectorVec2D(sr.Render[i].RelPos)
			sr.Render[i].Size.Y *= scy
			sr.Render[i].Size.X *= scx
		}
		pc.FontStyle.Size = orgsz
		if len(g.CharPosX) > 0 {
			mx := kit.MinInt(len(g.CharPosX), len(sr.Render))
			for i := 0; i < mx; i++ {
				// todo: this may not be fully correct, given relativity constraints
				sr.Render[i].RelPos.X, _ = rs.XForm.TransformVector(g.CharPosX[i], 0)
			}
		}
		if len(g.CharPosY) > 0 {
			mx := kit.MinInt(len(g.CharPosY), len(sr.Render))
			for i := 0; i < mx; i++ {
				_, sr.Render[i].RelPos.Y = rs.XForm.TransformPoint(g.CharPosY[i], 0)
			}
		}
		if len(g.CharPosDX) > 0 {
			mx := kit.MinInt(len(g.CharPosDX), len(sr.Render))
			for i := 0; i < mx; i++ {
				dx, _ := rs.XForm.TransformVector(g.CharPosDX[i], 0)
				if i > 0 {
					sr.Render[i].RelPos.X = sr.Render[i-1].RelPos.X + dx
				} else {
					sr.Render[i].RelPos.X = dx // todo: not sure this is right
				}
			}
		}
		if len(g.CharPosDY) > 0 {
			mx := kit.MinInt(len(g.CharPosDY), len(sr.Render))
			for i := 0; i < mx; i++ {
				dy, _ := rs.XForm.TransformVector(g.CharPosDY[i], 0)
				if i > 0 {
					sr.Render[i].RelPos.Y = sr.Render[i-1].RelPos.Y + dy
				} else {
					sr.Render[i].RelPos.Y = dy // todo: not sure this is right
				}
			}
		}
		// todo: TextLength, AdjustGlyphs -- also svg2 at least supports word wrapping!
		g.Render.Render(rs, pos)
		g.ComputeBBoxSVG()
	}
	g.Render2DChildren()
	rs.PopXForm()
}
