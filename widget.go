// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
	"image"
	"math"
	// "reflect"
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
//  Standard methods to call on the Parts

func (g *WidgetBase) Init2DParts() {
	if g.Parts.This == nil {
		g.Parts.SetThisName(&g.Parts, "Parts")
		g.Parts.SetParent(g.This)
		g.Parts.Init2DTree()
	}
	bitflag.Set(&g.Parts.NodeFlags, int(IsStructField)) // key for e.g., not adding parent pos
}

func (g *WidgetBase) Init2DWidget() {
	g.Init2DBase()
	g.Init2DParts()
}

func (g *WidgetBase) Style2DParts() {
	g.Parts.Style2DTree()
}

func (g *WidgetBase) Size2DParts(getSize bool) {
	g.Parts.Size2DTree()
	if getSize {
		g.LayData.AllocSize = g.Parts.LayData.Size.Pref // get from parts
		g.Size2DAddSpace()
	}
}

func (g *WidgetBase) Size2DWidget() {
	g.InitLayout2D()
	g.Size2DParts(true) // get our size from parts
}

func (g *WidgetBase) Layout2DParts(parBBox image.Rectangle) {
	spc := g.Style.BoxSpace()
	g.Parts.LayData = g.LayData
	g.Parts.LayData.AllocPos.SetAddVal(spc)
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
	g.Parts.This.(Node2D).ComputeBBox2D(parBBox)
	return psize
}

func (g *WidgetBase) Render2DParts() {
	g.Parts.Render2DTree()
}
