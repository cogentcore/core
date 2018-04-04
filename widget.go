// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"math"
	"strings"

	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
)

// Widget base type -- manages control elements and provides standard box model rendering
type WidgetBase struct {
	Node2DBase
	Parts Layout `desc:"a separate tree of sub-widgets that implement discrete parts of a widget -- positions are always relative to the parent widget"`
}

var KiT_WidgetBase = kit.Types.AddType(&WidgetBase{}, nil)

// Styling notes:
// simple elemental widgets (buttons etc) have a DefaultRender method that renders based on
// Style, with full css styling support -- code has built-in initial defaults for a default
// style based on fusion style parameters on QML Qt Quick Controls

// Alternatively they support custom svg code for rendering each state as appropriate in a Stack
// more complex widgets such as a TreeView automatically render and don't support custom svg

// WidgetBase supports full Box rendering model, so Button just calls these methods to render
// -- base function needs to take a Style arg.

func (g *WidgetBase) RenderBoxImpl(pos Vec2D, sz Vec2D, rad float64) {
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
	// rs := &g.Viewport.Render

	pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
	sz := g.LayData.AllocSize.AddVal(-2.0 * st.Layout.Margin.Dots)

	// first do any shadow
	if st.BoxShadow.HasShadow() {
		spos := pos.Add(Vec2D{st.BoxShadow.HOffset.Dots, st.BoxShadow.VOffset.Dots})
		pc.StrokeStyle.SetColor(nil)
		pc.FillStyle.SetColor(&st.BoxShadow.Color)
		g.RenderBoxImpl(spos, sz, st.Border.Radius.Dots)
	}
	// then draw the box over top of that -- note: won't work well for transparent! need to set clipping to box first..
	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColor(&st.Background.Color)
	g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
}

// measure given text string using current style
func (g *WidgetBase) MeasureTextSize(txt string) (w, h float64) {
	st := &g.Style
	pc := &g.Paint
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	w, h = pc.MeasureString(txt)
	return
}

// set our LayData.AllocSize from measured text size
func (g *WidgetBase) Size2DFromText(txt string) {
	st := &g.Style
	w, h := g.MeasureTextSize(txt)
	if st.Layout.Width.Dots > 0 {
		w = math.Max(st.Layout.Width.Dots, w)
	}
	if st.Layout.Height.Dots > 0 {
		h = math.Max(st.Layout.Height.Dots, h)
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
}

func (g *WidgetBase) Size2DWidget() {
	g.InitLayout2D()
	g.SizeFromParts() // get our size from parts
}

func (g *WidgetBase) Layout2DParts(parBBox image.Rectangle) {
	spc := g.Style.BoxSpace()
	g.Parts.LayData = g.LayData
	g.Parts.LayData.AllocPos.SetAddVal(spc)
	g.Parts.LayData.AllocSize.SetAddVal(-2.0 * spc)
	g.Parts.Layout2DTree(parBBox)

}

func (g *WidgetBase) Layout2DWidget(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DParts(parBBox)
}

func (g *WidgetBase) ComputeBBox2DWidget(parBBox image.Rectangle) Vec2D {
	psize := g.ComputeBBox2DBase(parBBox)
	spc := g.Style.BoxSpace()
	g.Parts.LayData.AllocPos = g.LayData.AllocPos.AddVal(spc)
	g.Parts.LayData.AllocSize = g.LayData.AllocSize.AddVal(-2.0 * spc)
	g.Parts.This.(Node2D).ComputeBBox2D(parBBox)
	return psize
}

func (g *WidgetBase) Render2DParts() {
	g.Parts.Render2DTree()
}

///////////////////////////////////////////////////////////////////
// ConfigParts building-blocks

// get the style properties for a child in parts based on its name -- only
// call this when new parts were created -- name of properties is #partname
// (lower cased) and it should contain a map[string]interface{} which is then
// added to the part's props
func (g *WidgetBase) PartStyleProps(part ki.Ki, props map[string]interface{}) {
	stynm := "#" + strings.ToLower(part.Name())
	pp := g.Prop(stynm, false, false)
	if pp == nil {
		if props != nil {
			ok := false
			pp, ok = props[stynm]
			if !ok {
				return
			}
		} else {
			return
		}
	}
	pmap, ok := pp.(map[string]interface{})
	if !ok {
		return
	}
	for key, val := range pmap {
		part.SetProp(key, val)
	}
}

// standard config of icon and label left-to right in a row, based on whether items are nil or empty
func (g *WidgetBase) ConfigPartsIconLabel(icn *Icon, txt string) (config kit.TypeAndNameList, icIdx, lbIdx int) {
	// todo: add some styles for button layout
	g.Parts.Lay = LayoutRow
	config = kit.TypeAndNameList{} // note: slice is already a pointer
	icIdx = -1
	lbIdx = -1
	if icn != nil {
		config.Add(KiT_Icon, "Icon")
		icIdx = 0
		if txt != "" {
			config.Add(KiT_Space, "Space")
		}
	}
	if txt != "" {
		lbIdx = len(config)
		config.Add(KiT_Label, "Label")
	}
	return
}

// set the icon and text values in parts, and get part style props, using given props if not set in object props
func (g *WidgetBase) ConfigPartsSetIconLabel(icn *Icon, txt string, icIdx, lbIdx int, props map[string]interface{}) {
	if icIdx >= 0 {
		kc, _ := g.Parts.Child(icIdx)
		ic := kc.(*Icon)
		if !ic.HasChildren() || ic.UniqueNm != icn.UniqueNm { // can't use nm b/c config does
			ic.CopyFrom(icn.This)
			ic.UniqueNm = icn.UniqueNm
			g.PartStyleProps(ic.This, props)
		}
	}
	if lbIdx >= 0 {
		kc, _ := g.Parts.Child(lbIdx)
		lbl := kc.(*Label)
		if lbl.Text != txt {
			g.PartStyleProps(lbl.This, props)
			lbl.Text = txt
		}
	}
}
