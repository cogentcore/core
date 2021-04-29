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

// Text renders SVG text, handling both text and tspan elements.
// tspan is nested under a parent text -- text has empty Text string.
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
	LastPos      mat32.Vec2 `xml:"-" json:"-" desc:"last text render position -- lower-left baseline of start"`
	LastBBox     mat32.Box2 `xml:"-" json:"-" desc:"last actual bounding box in display units (dots)"`
}

var KiT_Text = kit.Types.AddType(&Text{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewText adds a new text to given parent node, with given name, pos and text.
func AddNewText(parent ki.Ki, name string, x, y float32, text string) *Text {
	g := parent.AddNewChild(KiT_Text, name).(*Text)
	g.Pos.Set(x, y)
	g.Text = text
	return g
}

func (g *Text) SVGName() string {
	if len(g.Text) == 0 {
		return "text"
	}
	return "tspan"
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

// IsParText returns true if this element serves as a parent text element
// to tspan elements within it.  This is true if NumChildren() > 0 and
// Text == ""
func (g *Text) IsParText() bool {
	return g.NumChildren() > 0 && g.Text == ""
}

func (g *Text) SetPos(pos mat32.Vec2) {
	g.Pos = pos
	for _, kii := range g.Kids {
		kt := kii.(*Text)
		kt.Pos = g.Pnt.XForm.MulVec2AsPt(pos)
	}
}

func (g *Text) SetSize(sz mat32.Vec2) {
	g.Width = sz.X
	scx, _ := g.Pnt.XForm.ExtractScale()
	for _, kii := range g.Kids {
		kt := kii.(*Text)
		kt.Width = g.Width * scx
	}
}

func (g *Text) BBox2D() image.Rectangle {
	if g.IsParText() {
		return BBoxFromChildren(g)
	} else {
		return image.Rectangle{Min: g.LastBBox.Min.ToPointFloor(), Max: g.LastBBox.Max.ToPointCeil()}
	}
}

// TextBBox returns the bounding box in local coordinates
func (g *Text) TextBBox() mat32.Box2 {
	if g.Text == "" {
		return mat32.Box2{}
	}
	pc := &g.Pnt
	girl.OpenFont(&pc.FontStyle, &pc.UnContext) // use original size font
	g.TextRender.SetString(g.Text, &pc.FontStyle, &pc.UnContext, &pc.TextStyle, true, 0, 1)
	sr := &(g.TextRender.Spans[0])
	sr.Render[0].Face = pc.FontStyle.Face.Face // upscale

	pos := g.Pos

	if gist.IsAlignMiddle(pc.TextStyle.Align) || pc.TextStyle.Anchor == gist.AnchorMiddle {
		pos.X -= g.TextRender.Size.X * .5
	} else if gist.IsAlignEnd(pc.TextStyle.Align) || pc.TextStyle.Anchor == gist.AnchorEnd {
		pos.X -= g.TextRender.Size.X
	}
	if len(g.CharPosX) > 0 {
		mx := ints.MinInt(len(g.CharPosX), len(sr.Render))
		for i := 0; i < mx; i++ {
			sr.Render[i].RelPos.X = g.CharPosX[i]
		}
	}
	if len(g.CharPosY) > 0 {
		mx := ints.MinInt(len(g.CharPosY), len(sr.Render))
		for i := 0; i < mx; i++ {
			sr.Render[i].RelPos.Y = g.CharPosY[i]
		}
	}
	if len(g.CharPosDX) > 0 {
		mx := ints.MinInt(len(g.CharPosDX), len(sr.Render))
		for i := 0; i < mx; i++ {
			if i > 0 {
				sr.Render[i].RelPos.X = sr.Render[i-1].RelPos.X + g.CharPosDX[i]
			} else {
				sr.Render[i].RelPos.X = g.CharPosDX[i] // todo: not sure this is right
			}
		}
	}
	if len(g.CharPosDY) > 0 {
		mx := ints.MinInt(len(g.CharPosDY), len(sr.Render))
		for i := 0; i < mx; i++ {
			if i > 0 {
				sr.Render[i].RelPos.Y = sr.Render[i-1].RelPos.Y + g.CharPosDY[i]
			} else {
				sr.Render[i].RelPos.Y = g.CharPosDY[i] // todo: not sure this is right
			}
		}
	}
	// todo: TextLength, AdjustGlyphs -- also svg2 at least supports word wrapping!

	// accumulate final bbox
	sz := mat32.Vec2{}
	maxh := float32(0)
	for i := range sr.Render {
		mxp := sr.Render[i].RelPos.Add(sr.Render[i].Size)
		sz.SetMax(mxp)
		maxh = mat32.Max(maxh, sr.Render[i].Size.Y)
	}
	bb := mat32.Box2{}
	bb.Min = pos
	bb.Min.Y -= maxh * .8 // baseline adjust
	bb.Max = bb.Min.Add(g.TextRender.Size)
	return bb
}

// RenderText renders the text in full coords
func (g *Text) RenderText() {
	pc := &g.Pnt
	rs := g.Render()
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
	pc.FontStyle.Size = units.Value{orgsz.Val * scy, orgsz.Un, orgsz.Dots * scy} // rescale by y
	girl.OpenFont(&pc.FontStyle, &pc.UnContext)
	sr := &(g.TextRender.Spans[0])
	sr.Render[0].Face = pc.FontStyle.Face.Face // upscale
	g.TextRender.Size = g.TextRender.Size.Mul(mat32.Vec2{scx, scy})

	// todo: align styling only affects multi-line text and is about how tspan is arranged within
	// the overall text block.

	if gist.IsAlignMiddle(pc.TextStyle.Align) || pc.TextStyle.Anchor == gist.AnchorMiddle {
		pos.X -= g.TextRender.Size.X * .5
	} else if gist.IsAlignEnd(pc.TextStyle.Align) || pc.TextStyle.Anchor == gist.AnchorEnd {
		pos.X -= g.TextRender.Size.X
	}
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

	// accumulate final bbox
	sz := mat32.Vec2{}
	maxh := float32(0)
	for i := range sr.Render {
		mxp := sr.Render[i].RelPos.Add(sr.Render[i].Size)
		sz.SetMax(mxp)
		maxh = mat32.Max(maxh, sr.Render[i].Size.Y)
	}
	g.TextRender.Size = sz
	g.LastPos = pos
	g.LastBBox.Min = pos
	g.LastBBox.Min.Y -= maxh * .8 // baseline adjust
	g.LastBBox.Max = g.LastBBox.Min.Add(g.TextRender.Size)
	g.TextRender.Render(rs, pos)
	g.ComputeBBoxSVG()
}

func (g *Text) SVGLocalBBox() mat32.Box2 {
	return g.TextBBox()
}

func (g *Text) Render2D() {
	if g.IsParText() {
		if g.Viewport == nil {
			g.This().(gi.Node2D).Init2D()
		}
		pc := &g.Pnt
		rs := g.Render()
		if rs == nil {
			return
		}
		rs.PushXFormLock(pc.XForm)

		g.Render2DChildren()
		g.ComputeBBoxSVG() // must come after render

		rs.PopXFormLock()
	} else {
		vis, rs := g.PushXForm()
		if !vis {
			return
		}
		if len(g.Text) > 0 {
			g.RenderText()
		}
		g.Render2DChildren()
		if g.IsParText() {
			g.ComputeBBoxSVG() // after kids have rendered
		}
		rs.PopXFormLock()
	}
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Text) ApplyXForm(xf mat32.Mat2) {
	rot := xf.ExtractRot()
	if rot != 0 || !g.Pnt.XForm.IsIdentity() {
		g.Pnt.XForm = g.Pnt.XForm.Mul(xf)
		g.SetProp("transform", g.Pnt.XForm.String())
		g.GradientApplyXForm(xf)
	} else {
		g.Pos = xf.MulVec2AsPt(g.Pos)
		scx, _ := xf.ExtractScale()
		g.Width *= scx
	}
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Text) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	if rot != 0 {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, false) // exclude self
		mat := g.Pnt.XForm.MulCtr(xf, lpt)
		g.Pnt.XForm = mat
		g.SetProp("transform", g.Pnt.XForm.String())
	} else {
		if g.IsParText() {
			// translation transform
			xft, lptt := g.DeltaXForm(trans, scale, rot, pt, true) // include self when not a parent
			// transform transform
			xf, lpt := g.DeltaXForm(trans, scale, rot, pt, false)
			xf.X0 = 0 // negate translation effects
			xf.Y0 = 0
			mat := g.Pnt.XForm.MulCtr(xf, lpt)
			g.Pnt.XForm = mat
			g.SetProp("transform", g.Pnt.XForm.String())

			g.Pos = xft.MulVec2AsPtCtr(g.Pos, lptt)
			scx, _ := xft.ExtractScale()
			g.Width *= scx
			for _, kii := range g.Kids {
				kt := kii.(*Text)
				kt.Pos = xft.MulVec2AsPtCtr(kt.Pos, lptt)
				kt.Width *= scx
			}
		} else {
			xf, lpt := g.DeltaXForm(trans, scale, rot, pt, true) // include self when not a parent
			g.Pos = xf.MulVec2AsPtCtr(g.Pos, lpt)
			scx, _ := xf.ExtractScale()
			g.Width *= scx
		}
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Text) WriteGeom(dat *[]float32) {
	if g.IsParText() {
		npt := 9 + g.NumChildren()*3
		SetFloat32SliceLen(dat, npt)
		(*dat)[0] = g.Pos.X
		(*dat)[1] = g.Pos.Y
		(*dat)[2] = g.Width
		g.WriteXForm(*dat, 3)
		for i, kii := range g.Kids {
			kt := kii.(*Text)
			off := 9 + i*3
			(*dat)[off+0] = kt.Pos.X
			(*dat)[off+1] = kt.Pos.Y
			(*dat)[off+2] = kt.Width
		}
	} else {
		SetFloat32SliceLen(dat, 3+6)
		(*dat)[0] = g.Pos.X
		(*dat)[1] = g.Pos.Y
		(*dat)[2] = g.Width
		g.WriteXForm(*dat, 3)
	}
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Text) ReadGeom(dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Width = dat[2]
	g.ReadXForm(dat, 3)
	if g.IsParText() {
		for i, kii := range g.Kids {
			kt := kii.(*Text)
			off := 9 + i*3
			kt.Pos.X = dat[off+0]
			kt.Pos.Y = dat[off+1]
			kt.Width = dat[off+2]
		}
	}
}
