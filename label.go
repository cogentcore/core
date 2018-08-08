// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"
	"reflect"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Labeler Interface and ToLabel method

// the labeler interface provides a GUI-appropriate label (todo: rich text
// html tags!?) for an item -- use ToLabel converter to attempt to use this
// interface and then fall back on Stringer via kit.ToString conversion
// function
type Labeler interface {
	Label() string
}

// ToLabel returns the gui-appropriate label for an item, using the Labeler
// interface if it is defined, and falling back on kit.ToString converter
// otherwise -- also contains label impls for basic interface types for which
// we cannot easily define the Labeler interface
func ToLabel(it interface{}) string {
	lbler, ok := it.(Labeler)
	if !ok {
		// typ := reflect.TypeOf(it)
		// if kit.EmbeddedTypeImplements(typ, reflect.TypeOf((*reflect.Type)(nil)).Elem()) {
		// 	to, ok :=
		// }
		switch v := it.(type) {
		case reflect.Type:
			return v.Name()
		}
		return kit.ToString(it)
	}
	return lbler.Label()
}

////////////////////////////////////////////////////////////////////////////////////////
// Label

// Label is a widget for rendering text labels -- supports full widget model
// including box rendering
type Label struct {
	WidgetBase
	Text        string              `xml:"text" desc:"label to display"`
	Selectable  bool                `desc:"is this label selectable? if so, it will change background color in response to selection events and update selection state on mouse clicks"`
	StateStyles [LabelStatesN]Style `json:"-" xml:"-" desc:"styles for different states of label"`
	Render      TextRender          `xml:"-" json:"-" desc:"render data for text label"`
	CurBgColor  Color               `xml:"-" json:"-" desc:"current background color -- grabbed when rendering for first time, and used when toggling off of selected mode, to wipe out bg"`
}

var KiT_Label = kit.Types.AddType(&Label{}, LabelProps)

var LabelProps = ki.Props{
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"vertical-align":   AlignTop,
	"color":            &Prefs.FontColor,
	"background-color": color.Transparent,
	LabelSelectors[LabelActive]: ki.Props{
		"background-color": color.Transparent,
	},
	LabelSelectors[LabelInactive]: ki.Props{
		"color": "lighter-50",
	},
	LabelSelectors[LabelSelected]: ki.Props{
		"background-color": &Prefs.SelectColor,
	},
}

// LabelStates are mutually-exclusive label states -- determines appearance
type LabelStates int32

const (
	// normal active state
	LabelActive LabelStates = iota

	// inactive -- font is dimmed
	LabelInactive

	// selected -- background is selected color
	LabelSelected

	// total number of button states
	LabelStatesN
)

//go:generate stringer -type=LabelStates

var KiT_LabelStates = kit.Enums.AddEnumAltLower(LabelStatesN, false, StylePropProps, "Label")

func (ev LabelStates) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *LabelStates) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// LabelSelectors are Style selector names for the different states:
var LabelSelectors = []string{":active", ":inactive", ":selected"}

// SetText sets the text and updates the rendered version
func (g *Label) SetText(txt string) {
	g.Text = txt
	g.Render.SetHTML(g.Text, &(g.Sty.Font), &(g.Sty.UnContext), g.CSSAgg)
	spc := g.Sty.BoxSpace()
	sz := g.LayData.AllocSize
	if sz.IsZero() {
		sz = g.LayData.SizePrefOrMax()
	}
	if !sz.IsZero() {
		sz.SetSubVal(2 * spc)
	}
	g.Render.LayoutStdLR(&(g.Sty.Text), &(g.Sty.Font), &(g.Sty.UnContext), sz)
}

// SetTextAction sets the text and triggers an update action
func (g *Label) SetTextAction(txt string) {
	g.SetText(txt)
	g.UpdateSig()
}

// SetStateStyle sets the style based on the inactive, selected flags
func (g *Label) SetStateStyle() {
	if g.IsInactive() {
		g.Sty = g.StateStyles[LabelInactive]
	} else if g.IsSelected() {
		g.Sty = g.StateStyles[LabelSelected]
	} else {
		g.Sty = g.StateStyles[LabelActive]
		if g.Selectable && !g.CurBgColor.IsNil() {
			g.Sty.Font.BgColor.SetColor(g.CurBgColor)
		}
	}
}

func (g *Label) Style2D() {
	g.Style2DWidget()
	pst := g.ParentStyle()
	for i := 0; i < int(LabelStatesN); i++ {
		g.StateStyles[i].CopyFrom(&g.Sty)
		g.StateStyles[i].SetStyleProps(pst, g.StyleProps(LabelSelectors[i]))
		g.StateStyles[i].CopyUnitContext(&g.Sty.UnContext)
	}
	g.Render.SetHTML(g.Text, &(g.Sty.Font), &(g.Sty.UnContext), g.CSSAgg)
	spc := g.Sty.BoxSpace()
	sz := g.LayData.SizePrefOrMax()
	if !sz.IsZero() {
		sz.SetSubVal(2 * spc)
	}
	g.Render.LayoutStdLR(&(g.Sty.Text), &(g.Sty.Font), &(g.Sty.UnContext), sz)
}

func (g *Label) Size2D() {
	g.InitLayout2D()
	g.Size2DFromWH(g.Render.Size.X, g.Render.Size.Y)
}

func (g *Label) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true)
	g.Layout2DChildren()
	sz := g.Size2DSubSpace()
	g.Render.LayoutStdLR(&(g.Sty.Text), &(g.Sty.Font), &(g.Sty.UnContext), sz)
}

func (g *Label) LabelEvents() {
	g.HoverTooltipEvent()
	g.WidgetMouseEvents(g.Selectable, true)
}

func (g *Label) GrabCurBgColor() {
	if g.Viewport == nil || g.IsSelected() || !g.CurBgColor.IsNil() {
		return
	}
	pos := g.ContextMenuPos()
	clr := g.Viewport.Pixels.At(pos.X, pos.Y)
	g.CurBgColor.SetColor(clr)
}

func (g *Label) Render2D() {
	if g.FullReRenderIfNeeded() {
		return
	}
	if g.PushBounds() {
		g.LabelEvents()
		g.GrabCurBgColor()
		g.SetStateStyle()
		st := &g.Sty
		rs := &g.Viewport.Render
		g.RenderStdBox(st)
		pos := g.LayData.AllocPos.AddVal(st.BoxSpace())
		g.Render.Render(rs, pos)
		g.Render2DChildren()
		g.PopBounds()
	} else {
		g.DisconnectAllEvents(RegPri)
	}
}
