// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// Widget base type -- manages control elements and provides standard box model rendering
type WidgetBase struct {
	Node2DBase
	Parts Layout `json:"-" xml:"-" view-closed:"true" desc:"a separate tree of sub-widgets that implement discrete parts of a widget -- positions are always relative to the parent widget -- fully managed by the widget and not saved"`
}

var KiT_WidgetBase = kit.Types.AddType(&WidgetBase{}, WidgetBaseProps)

var WidgetBaseProps = ki.Props{
	"base-type": true,
}

// WidgetBase supports full Box rendering model, so Button just calls these
// methods to render -- base function needs to take a Style arg.

func (g *WidgetBase) RenderBoxImpl(pos Vec2D, sz Vec2D, rad float32) {
	pc := &g.Paint
	rs := &g.Viewport.Render
	if rad == 0.0 {
		pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
	} else {
		pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
	}
	pc.FillStrokeClear(rs)
}

// draw standard box using given style
func (g *WidgetBase) RenderStdBox(st *Style) {
	pc := &g.Paint
	rs := &g.Viewport.Render

	pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
	sz := g.LayData.AllocSize.AddVal(-2.0 * st.Layout.Margin.Dots)

	// first do any shadow
	if st.BoxShadow.HasShadow() {
		spos := pos.Add(Vec2D{st.BoxShadow.HOffset.Dots, st.BoxShadow.VOffset.Dots})
		pc.StrokeStyle.SetColor(nil)
		pc.FillStyle.Color.SetShadowGradient(st.BoxShadow.Color, "")
		// todo: this is not rendering a transparent gradient
		// pc.FillStyle.Opacity = .5
		g.RenderBoxImpl(spos, sz, st.Border.Radius.Dots)
		// pc.FillStyle.Opacity = 1.0
	}
	// then draw the box over top of that -- note: won't work well for transparent! need to set clipping to box first..
	if !st.Background.Color.IsNil() {
		pc.FillBox(rs, pos, sz, &st.Background.Color)
	}

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	// pc.FillStyle.SetColor(&st.Background.Color)
	pc.FillStyle.SetColor(nil)
	g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
}

// measure given text string using current style
func (g *WidgetBase) MeasureTextSize(txt string) (w, h float32) {
	st := &g.Style
	pc := &g.Paint
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	w, h = pc.MeasureString(txt)
	w += 4.0 // add a little buffer for text widths so things don't get cutoff
	return
}

// set our LayData.AllocSize from measured text size
func (g *WidgetBase) Size2DFromText(txt string) {
	w, h := g.MeasureTextSize(txt)
	g.Size2DFromWH(w, h)
}

// set our LayData.AllocSize from constraints
func (g *WidgetBase) Size2DFromWH(w, h float32) {
	st := &g.Style
	if st.Layout.Width.Dots > 0 {
		w = Max32(st.Layout.Width.Dots, w)
	}
	if st.Layout.Height.Dots > 0 {
		h = Max32(st.Layout.Height.Dots, h)
	}
	spc := st.BoxSpace()
	w += 2.0 * spc
	h += 2.0 * spc
	g.LayData.AllocSize = Vec2D{w, h}
}

// add space to existing AllocSize
func (g *WidgetBase) Size2DAddSpace() {
	spc := g.Style.BoxSpace()
	g.LayData.AllocSize.SetAddVal(2.0 * spc)
}

// render a text string in standard box model (e.g., label for a button, etc)
func (g *WidgetBase) Render2DText(txt string) {
	pc := &g.Paint
	rs := &g.Viewport.Render
	st := &g.Style
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	pc.StrokeStyle.SetColor(&st.Color) // ink color

	spc := st.BoxSpace()
	pos := g.LayData.AllocPos.AddVal(spc)
	sz := g.LayData.AllocSize.AddVal(-2.0 * spc)

	// automatically compensate for alignment so top and middle = same thing
	if IsAlignMiddle(st.Text.AlignV) {
		pos.Y += 0.5 * sz.Y
	}

	pc.DrawString(rs, txt, pos.X, pos.Y, sz.X)
}

///////////////////////////////////////////////////////////////////
// Standard methods to call on the Parts

// standard FunDownMeFirst etc operate automaticaly on Field structs such as
// Parts -- custom calls only needed for manually-recursive traversal in
// Layout and Render

func (g *WidgetBase) Init2DWidget() {
	g.Init2DBase()
}

func (g *WidgetBase) SizeFromParts() {
	g.LayData.AllocSize = g.Parts.LayData.Size.Pref // get from parts
	g.Size2DAddSpace()
	if Layout2DTrace {
		fmt.Printf("Size:   %v size from parts: %v, parts pref: %v\n", g.PathUnique(), g.LayData.AllocSize, g.Parts.LayData.Size.Pref)
	}
}

func (g *WidgetBase) Size2DWidget() {
	g.InitLayout2D()
	g.SizeFromParts() // get our size from parts
}

func (g *WidgetBase) Layout2DParts(parBBox image.Rectangle) {
	spc := g.Style.BoxSpace()
	g.Parts.LayData.AllocPos = g.LayData.AllocPos.AddVal(spc)
	g.Parts.LayData.AllocSize = g.LayData.AllocSize.AddVal(-2.0 * spc)
	g.Parts.Layout2D(parBBox)
}

func (g *WidgetBase) Layout2DWidget(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DParts(parBBox)
}

func (g *WidgetBase) ComputeBBox2DWidget(parBBox image.Rectangle, delta image.Point) {
	g.ComputeBBox2DBase(parBBox, delta)
	g.Parts.This.(Node2D).ComputeBBox2D(parBBox, delta)
}

func (g *WidgetBase) Move2DWidget(delta image.Point, parBBox image.Rectangle) {
	g.Move2DBase(delta, parBBox)
	g.Parts.This.(Node2D).Move2D(delta, parBBox)
}

func (g *WidgetBase) Render2DParts() {
	g.Parts.Render2DTree()
}

///////////////////////////////////////////////////////////////////
// ConfigParts building-blocks

// ConfigPartsIconLabel returns a standard config for creating parts, of icon and label left-to right in a row, based on whether items are nil or empty
func (g *WidgetBase) ConfigPartsIconLabel(icnm string, txt string) (config kit.TypeAndNameList, icIdx, lbIdx int) {
	// todo: add some styles for button layout
	config = kit.TypeAndNameList{}
	icIdx = -1
	lbIdx = -1
	if IconNameValid(icnm) {
		config.Add(KiT_Icon, "icon")
		icIdx = 0
		if txt != "" {
			config.Add(KiT_Space, "space")
		}
	}
	if txt != "" {
		lbIdx = len(config)
		config.Add(KiT_Label, "label")
	}
	return
}

// ConfigPartsSetIconLabel sets the icon and text values in parts, and get
// part style props, using given props if not set in object props
func (g *WidgetBase) ConfigPartsSetIconLabel(icnm string, txt string, icIdx, lbIdx int) {
	if icIdx >= 0 {
		ic := g.Parts.Child(icIdx).(*Icon)
		if !ic.HasChildren() || ic.UniqueNm != icnm { // can't use nm b/c config does
			ic.InitFromName(icnm)
			ic.UniqueNm = icnm
			g.StylePart(ic.This)
		}
	}
	if lbIdx >= 0 {
		lbl := g.Parts.Child(lbIdx).(*Label)
		if lbl.Text != txt {
			g.StylePart(lbl.This)
			if icIdx >= 0 {
				g.StylePart(g.Parts.Child(lbIdx - 1)) // also get the space
			}
			lbl.Text = txt
		}
	}
}

// PartsNeedUpdateIconLabel check if parts need to be updated -- for ConfigPartsIfNeeded
func (g *WidgetBase) PartsNeedUpdateIconLabel(icnm string, txt string) bool {
	if IconNameValid(icnm) {
		ick := g.Parts.ChildByName("icon", 0)
		if ick == nil {
			return true
		}
		ic := ick.(*Icon)
		if !ic.HasChildren() || ic.UniqueNm != icnm {
			return true
		}
	} else {
		ic := g.Parts.ChildByName("icon", 0)
		if ic != nil {
			return true
		}
	}
	if txt != "" {
		lbl := g.Parts.ChildByName("label", 2)
		if lbl == nil {
			return true
		}
		lbl.(*Label).Style.Color = g.Style.Color
		if lbl.(*Label).Text != txt {
			return true
		}
	} else {
		lbl := g.Parts.ChildByName("label", 2)
		if lbl != nil {
			return true
		}
	}
	return false
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D impl for WidgetBase

func (g *WidgetBase) Init2D() {
	g.Init2DWidget()
}

func (g *WidgetBase) Size2D() {
	g.Size2DWidget()
}

func (g *WidgetBase) Layout2D(parBBox image.Rectangle) {
	g.Layout2DWidget(parBBox)
	g.Layout2DChildren()
}

func (g *WidgetBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	g.ComputeBBox2DWidget(parBBox, delta)
}

func (g *WidgetBase) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

func (g *WidgetBase) Move2D(delta image.Point, parBBox image.Rectangle) {
	g.Move2DWidget(delta, parBBox)
	g.Move2DChildren(delta)
}

func (g *WidgetBase) ReRender2D() (node Node2D, layout bool) {
	node = g.This.(Node2D)
	layout = false
	return
}

func (g *WidgetBase) FocusChanged2D(gotFocus bool) {
}
