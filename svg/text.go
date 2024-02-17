// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// Text renders SVG text, handling both text and tspan elements.
// tspan is nested under a parent text -- text has empty Text string.
type Text struct {
	NodeBase

	// position of the left, baseline of the text
	Pos mat32.Vec2 `xml:"{x,y}" set:"-"`

	// width of text to render if using word-wrapping
	Width float32 `xml:"width"`

	// text string to render
	Text string `xml:"text"`

	// render version of text
	TextRender paint.Text `xml:"-" json:"-" copier:"-"`

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

	// last text render position -- lower-left baseline of start
	LastPos mat32.Vec2 `xml:"-" json:"-" copier:"-"`

	// last actual bounding box in display units (dots)
	LastBBox mat32.Box2 `xml:"-" json:"-" copier:"-"`
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

func (g *Text) SetPos(pos mat32.Vec2) *Text {
	g.Pos = pos
	for _, kii := range g.Kids {
		kt := kii.(*Text)
		kt.Pos = g.Paint.Transform.MulVec2AsPt(pos)
	}
	return g
}

func (g *Text) SetSize(sz mat32.Vec2) *Text {
	g.Width = sz.X
	scx, _ := g.Paint.Transform.ExtractScale()
	for _, kii := range g.Kids {
		kt := kii.(*Text)
		kt.Width = g.Width * scx
	}
	return g
}

func (g *Text) NodeBBox(sv *SVG) image.Rectangle {
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
	pc := &g.Paint
	pc.FontStyle.Font = paint.OpenFont(&pc.FontStyle, &pc.UnContext) // use original size font
	g.TextRender.SetString(g.Text, &pc.FontStyle, &pc.UnContext, &pc.TextStyle, true, 0, 1)
	sr := &(g.TextRender.Spans[0])
	sr.Render[0].Face = pc.FontStyle.Face.Face // upscale

	pos := g.Pos

	if pc.TextStyle.Align == styles.Center || pc.TextStyle.Anchor == styles.AnchorMiddle {
		pos.X -= g.TextRender.Size.X * .5
	} else if pc.TextStyle.Align == styles.End || pc.TextStyle.Anchor == styles.AnchorEnd {
		pos.X -= g.TextRender.Size.X
	}
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
func (g *Text) RenderText(sv *SVG) {
	pc := &paint.Context{State: &sv.RenderState, Paint: &g.Paint}
	orgsz := pc.FontStyle.Size
	pos := pc.CurTransform.MulVec2AsPt(mat32.V2(g.Pos.X, g.Pos.Y))
	rot := pc.CurTransform.ExtractRot()
	scx, scy := pc.CurTransform.ExtractScale()
	scalex := scx / scy
	if scalex == 1 {
		scalex = 0
	}
	pc.FontStyle.Font = paint.OpenFont(&pc.FontStyle, &pc.UnContext) // use original size font
	if pc.FillStyle.Color != nil {
		pc.FontStyle.Color = colors.ToUniform(pc.FillStyle.Color)
	}
	g.TextRender.SetString(g.Text, &pc.FontStyle, &pc.UnContext, &pc.TextStyle, true, rot, scalex)
	pc.FontStyle.Size = units.Value{Val: orgsz.Val * scy, Un: orgsz.Un, Dots: orgsz.Dots * scy} // rescale by y
	pc.FontStyle.Font = paint.OpenFont(&pc.FontStyle, &pc.UnContext)
	sr := &(g.TextRender.Spans[0])
	sr.Render[0].Face = pc.FontStyle.Face.Face // upscale
	g.TextRender.Size = g.TextRender.Size.Mul(mat32.V2(scx, scy))

	// todo: align styling only affects multi-line text and is about how tspan is arranged within
	// the overall text block.

	if pc.TextStyle.Align == styles.Center || pc.TextStyle.Anchor == styles.AnchorMiddle {
		pos.X -= g.TextRender.Size.X * .5
	} else if pc.TextStyle.Align == styles.End || pc.TextStyle.Anchor == styles.AnchorEnd {
		pos.X -= g.TextRender.Size.X
	}
	for i := range sr.Render {
		sr.Render[i].RelPos = pc.CurTransform.MulVec2AsVec(sr.Render[i].RelPos)
		sr.Render[i].Size.Y *= scy
		sr.Render[i].Size.X *= scx
	}
	pc.FontStyle.Size = orgsz
	if len(g.CharPosX) > 0 {
		mx := min(len(g.CharPosX), len(sr.Render))
		for i := 0; i < mx; i++ {
			// todo: this may not be fully correct, given relativity constraints
			cpx := pc.CurTransform.MulVec2AsVec(mat32.V2(g.CharPosX[i], 0))
			sr.Render[i].RelPos.X = cpx.X
		}
	}
	if len(g.CharPosY) > 0 {
		mx := min(len(g.CharPosY), len(sr.Render))
		for i := 0; i < mx; i++ {
			cpy := pc.CurTransform.MulVec2AsPt(mat32.V2(g.CharPosY[i], 0))
			sr.Render[i].RelPos.Y = cpy.Y
		}
	}
	if len(g.CharPosDX) > 0 {
		mx := min(len(g.CharPosDX), len(sr.Render))
		for i := 0; i < mx; i++ {
			dx := pc.CurTransform.MulVec2AsVec(mat32.V2(g.CharPosDX[i], 0))
			if i > 0 {
				sr.Render[i].RelPos.X = sr.Render[i-1].RelPos.X + dx.X
			} else {
				sr.Render[i].RelPos.X = dx.X // todo: not sure this is right
			}
		}
	}
	if len(g.CharPosDY) > 0 {
		mx := min(len(g.CharPosDY), len(sr.Render))
		for i := 0; i < mx; i++ {
			dy := pc.CurTransform.MulVec2AsVec(mat32.V2(g.CharPosDY[i], 0))
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
	g.TextRender.Render(pc, pos)
	g.BBoxes(sv)
}

func (g *Text) LocalBBox() mat32.Box2 {
	return g.TextBBox()
}

func (g *Text) Render(sv *SVG) {
	if g.IsParText() {
		pc := &g.Paint
		rs := &sv.RenderState
		rs.PushTransformLock(pc.Transform)

		g.RenderChildren(sv)
		g.BBoxes(sv) // must come after render

		rs.PopTransformLock()
	} else {
		vis, rs := g.PushTransform(sv)
		if !vis {
			return
		}
		if len(g.Text) > 0 {
			rs.Lock()
			g.RenderText(sv)
			rs.Unlock()
		}
		g.RenderChildren(sv)
		if g.IsParText() {
			g.BBoxes(sv) // after kids have rendered
		}
		rs.PopTransformLock()
	}
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Text) ApplyTransform(sv *SVG, xf mat32.Mat2) {
	rot := xf.ExtractRot()
	if rot != 0 || !g.Paint.Transform.IsIdentity() {
		g.Paint.Transform.SetMul(xf)
		g.SetProp("transform", g.Paint.Transform.String())
	} else {
		if g.IsParText() {
			for _, kii := range g.Kids {
				kt := kii.(*Text)
				kt.ApplyTransform(sv, xf)
			}
		} else {
			g.Pos = xf.MulVec2AsPt(g.Pos)
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
func (g *Text) ApplyDeltaTransform(sv *SVG, trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	crot := g.Paint.Transform.ExtractRot()
	if rot != 0 || crot != 0 {
		xf, lpt := g.DeltaTransform(trans, scale, rot, pt, false) // exclude self
		g.Paint.Transform.SetMulCtr(xf, lpt)
		g.SetProp("transform", g.Paint.Transform.String())
	} else {
		if g.IsParText() {
			// translation transform
			xft, lptt := g.DeltaTransform(trans, scale, rot, pt, true) // include self when not a parent
			// transform transform
			xf, lpt := g.DeltaTransform(trans, scale, rot, pt, false)
			xf.X0 = 0 // negate translation effects
			xf.Y0 = 0
			g.Paint.Transform.SetMulCtr(xf, lpt)
			g.SetProp("transform", g.Paint.Transform.String())

			g.Pos = xft.MulVec2AsPtCtr(g.Pos, lptt)
			scx, _ := xft.ExtractScale()
			g.Width *= scx
			for _, kii := range g.Kids {
				kt := kii.(*Text)
				kt.Pos = xft.MulVec2AsPtCtr(kt.Pos, lptt)
				kt.Width *= scx
			}
		} else {
			xf, lpt := g.DeltaTransform(trans, scale, rot, pt, true) // include self when not a parent
			g.Pos = xf.MulVec2AsPtCtr(g.Pos, lpt)
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
		SetFloat32SliceLen(dat, npt)
		(*dat)[0] = g.Pos.X
		(*dat)[1] = g.Pos.Y
		(*dat)[2] = g.Width
		g.WriteTransform(*dat, 3)
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
		for i, kii := range g.Kids {
			kt := kii.(*Text)
			off := 9 + i*3
			kt.Pos.X = dat[off+0]
			kt.Pos.Y = dat[off+1]
			kt.Width = dat[off+2]
		}
	}
}
