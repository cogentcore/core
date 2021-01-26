// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Text renders SVG text-- it handles both text and tspan elements (a tspan is
// just nested under a parent text)
type Text struct {
	NodeBase
	Pos          mat32.Vec2 `xml:"{x,y}" desc:"position of the left, baseline of the text"`
	Width        float32    `xml:"width" desc:"width of text to render if using word-wrapping"`
	Text         string     `xml:"text" desc:"text string to render"`
	TextRender   girl.Text  `xml:"-" json:"-" desc:"render version of text"`
	CharPosX     []float32  `desc:"character positions along X axis, if specified"`
	CharPosY     []float32  `desc:"character positions along Y axis, if specified"`
	CharPosDX    []float32  `desc:"character delta-positions along X axis, if specified"`
	CharPosDY    []float32  `desc:"character delta-positions along Y axis, if specified"`
	CharRots     []float32  `desc:"character rotations, if specified"`
	TextLength   float32    `desc:"author's computed text length, if specified -- we attempt to match"`
	AdjustGlyphs bool       `desc:"in attempting to match TextLength, should we adjust glyphs in addition to spacing?"`
}

var KiT_Text = kit.Types.AddType(&Text{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewText adds a new text to given parent node, with given name, pos and text.
func AddNewText(parent ki.Ki, name string, x, y float32, text string) *Text {
	g := parent.AddNewChild(KiT_Text, name).(*Text)
	g.Pos.Set(x, y)
	g.Text = text
	return g
}

func (g *Text) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Text)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Width = fr.Width
	g.Text = fr.Text
	mat32.CopyFloat32s(&g.CharPosX, fr.CharPosX)
	mat32.CopyFloat32s(&g.CharPosY, fr.CharPosY)
	mat32.CopyFloat32s(&g.CharPosDX, fr.CharPosDX)
	mat32.CopyFloat32s(&g.CharPosDY, fr.CharPosDY)
	mat32.CopyFloat32s(&g.CharRots, fr.CharRots)
	g.TextLength = fr.TextLength
	g.AdjustGlyphs = fr.AdjustGlyphs
}

func (g *Text) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	// todo: could be much more accurate..
	return g.Pnt.BoundingBox(rs, g.Pos.X, g.Pos.Y, g.Pos.X+g.TextRender.Size.X, g.Pos.Y+g.TextRender.Size.Y)
}

func (g *Text) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := g.Render()
	rs.PushXForm(pc.XForm)
	if len(g.Text) > 0 {
		orgsz := pc.FontStyle.Size
		pos := rs.XForm.MulVec2AsPt(mat32.Vec2{g.Pos.X, g.Pos.Y})
		rot := rs.XForm.ExtractRot()
		scx, scy := rs.XForm.ExtractScale()
		scalex := scx / scy
		if scalex == 1 {
			scalex = 0
		}
		girl.OpenFont(&pc.FontStyle, &pc.UnContext) // use original size font
		if !pc.FillStyle.Color.IsNil() {
			pc.FontStyle.Color = pc.FillStyle.Color.Color
		}
		g.TextRender.SetString(g.Text, &pc.FontStyle, &pc.UnContext, &pc.TextStyle, true, rot, scalex)
		g.TextRender.Size = g.TextRender.Size.Mul(mat32.Vec2{scx, scy})
		if gist.IsAlignMiddle(pc.TextStyle.Align) || pc.TextStyle.Anchor == gist.AnchorMiddle {
			pos.X -= g.TextRender.Size.X * .5
		} else if gist.IsAlignEnd(pc.TextStyle.Align) || pc.TextStyle.Anchor == gist.AnchorEnd {
			pos.X -= g.TextRender.Size.X
		}
		pc.FontStyle.Size = units.Value{orgsz.Val * scy, orgsz.Un, orgsz.Dots * scy} // rescale by y
		girl.OpenFont(&pc.FontStyle, &pc.UnContext)
		sr := &(g.TextRender.Spans[0])
		sr.Render[0].Face = pc.FontStyle.Face.Face // upscale
		for i := range sr.Render {
			sr.Render[i].RelPos = rs.XForm.MulVec2AsVec(sr.Render[i].RelPos)
			sr.Render[i].Size.Y *= scy
			sr.Render[i].Size.X *= scx
		}
		pc.FontStyle.Size = orgsz
		if len(g.CharPosX) > 0 {
			mx := ints.MinInt(len(g.CharPosX), len(sr.Render))
			for i := 0; i < mx; i++ {
				// todo: this may not be fully correct, given relativity constraints
				cpx := rs.XForm.MulVec2AsVec(mat32.Vec2{g.CharPosX[i], 0})
				sr.Render[i].RelPos.X = cpx.X
			}
		}
		if len(g.CharPosY) > 0 {
			mx := ints.MinInt(len(g.CharPosY), len(sr.Render))
			for i := 0; i < mx; i++ {
				cpy := rs.XForm.MulVec2AsPt(mat32.Vec2{g.CharPosY[i], 0})
				sr.Render[i].RelPos.Y = cpy.Y
			}
		}
		if len(g.CharPosDX) > 0 {
			mx := ints.MinInt(len(g.CharPosDX), len(sr.Render))
			for i := 0; i < mx; i++ {
				dx := rs.XForm.MulVec2AsVec(mat32.Vec2{g.CharPosDX[i], 0})
				if i > 0 {
					sr.Render[i].RelPos.X = sr.Render[i-1].RelPos.X + dx.X
				} else {
					sr.Render[i].RelPos.X = dx.X // todo: not sure this is right
				}
			}
		}
		if len(g.CharPosDY) > 0 {
			mx := ints.MinInt(len(g.CharPosDY), len(sr.Render))
			for i := 0; i < mx; i++ {
				dy := rs.XForm.MulVec2AsVec(mat32.Vec2{g.CharPosDY[i], 0})
				if i > 0 {
					sr.Render[i].RelPos.Y = sr.Render[i-1].RelPos.Y + dy.Y
				} else {
					sr.Render[i].RelPos.Y = dy.Y // todo: not sure this is right
				}
			}
		}
		// todo: TextLength, AdjustGlyphs -- also svg2 at least supports word wrapping!
		g.TextRender.Render(rs, pos)
		g.ComputeBBoxSVG()
	}
	g.Render2DChildren()
	rs.PopXForm()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Text) ApplyXForm(xf mat32.Mat2) {
	g.Pos = xf.MulVec2AsPt(g.Pos)
	scx, _ := xf.ExtractScale()
	g.Width *= scx
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Text) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	xf, lpt := g.DeltaXForm(trans, scale, rot, pt)
	g.Pos = xf.MulVec2AsPtCtr(g.Pos, lpt)
	scx, _ := xf.ExtractScale()
	g.Width *= scx
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Text) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 3)
	(*dat)[0] = g.Pos.X
	(*dat)[1] = g.Pos.Y
	(*dat)[2] = g.Width
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Text) ReadGeom(dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Width = dat[2]
}
