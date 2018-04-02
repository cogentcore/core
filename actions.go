// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"

	// "golang.org/x/image/font"
	"image"
	"log"
	// "math"
)

// actions contains the Action and Menus and Toolbars where actions live

////////////////////////////////////////////////////////////////////////////////////////
// Action -- for menu items and tool bars

// these extend NodeBase NodeFlags to hold action state
const (
	// action is in a menu -- styles and behaves differently than in a toolbar -- set by menu call
	ActionFlagMenu NodeFlags = ButtonFlagsN + iota
)

// signals that buttons can send
type ActionSignals int64

const (
	// action just sends one signal: triggered -- use ButtonSig for more detailed ones
	ActionTriggered ActionSignals = iota
	ActionSignalsN
)

//go:generate stringer -type=ActionSignals

// Action is a button widget that can display a text label and / or an icon and / or
// a keyboard shortcut -- this is what is put in menus and toolbars
// todo: need icon
type Action struct {
	ButtonBase
	ActionSig ki.Signal `json:"-" xml:"-" desc:"signal for action -- very simple -- Action triggered"`
}

func (g *Action) ActionReleased() {
	wasPressed := (g.State == ButtonDown)
	g.UpdateStart()
	g.SetButtonState(ButtonNormal)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ActionSig.Emit(g.This, int64(ActionTriggered), nil)
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
	}
	g.UpdateEnd()
}

var KiT_Action = kit.Types.AddType(&Action{}, nil)

func (g *Action) SetAsMenu() {
	bitflag.Set(&g.NodeFlags, int(ActionFlagMenu))
}

func (g *Action) SetAsButton() {
	bitflag.Clear(&g.NodeFlags, int(ActionFlagMenu))
}

func (g *Action) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Action) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Action) AsLayout2D() *Layout {
	return nil
}

func (g *Action) Init2D() {
	g.Init2DBase()
	g.ReceiveEventType(oswin.MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Action) // note: will fail for any derived classes..
		if ok {
			ab.ButtonPressed()
		}
	})
	g.ReceiveEventType(oswin.MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Action)
		if ok {
			ab.ActionReleased()
		}
	})
	g.ReceiveEventType(oswin.MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Action)
		if ok {
			ab.ButtonEnterHover()
		}
	})
	g.ReceiveEventType(oswin.MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Action)
		if ok {
			ab.ButtonExitHover()
		}
	})
	g.ReceiveEventType(oswin.KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Action)
		if ok {
			kt, ok := d.(oswin.KeyTypedEvent)
			if ok {
				// todo: register shortcuts with window, and generalize these keybindings
				kf := KeyFun(kt.Key, kt.Chord)
				if kf == KeyFunSelectItem || kt.Key == "space" {
					ab.ButtonPressed()
					// todo: brief delay??
					ab.ButtonReleased()
				}
			}
		}
	})
}

var ActionProps = []map[string]interface{}{
	{
		"border-width":  "0px",
		"border-radius": "0px",
		"border-color":  "black",
		"border-style":  "solid",
		"padding":       "2px",
		"margin":        "0px",
		// "font-family":         "Arial", // this is crashing
		"font-size":        "20pt",
		"text-align":       "center",
		"vertical-align":   "top",
		"color":            "black",
		"background-color": "#EEF",
	}, { // disabled
		"border-color":     "#BBB",
		"color":            "#AAA",
		"background-color": "#DDD",
	}, { // hover
		"background-color": "#CCF", // todo "darker"
	}, { // focus
		"border-color":     "#EEF",
		"box-shadow.color": "#BBF",
	}, { // press
		"border-color":     "#DDF",
		"color":            "white",
		"background-color": "#008",
	}, { // selected
		"border-color":     "#DDF",
		"color":            "white",
		"background-color": "#00F",
	},
}

func (g *Action) Style2D() {
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.Style.SetStyle(nil, &StyleDefault, ActionProps[ButtonNormal])
	g.Style2DWidget()
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, ActionProps[i])
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2D{})
	}
}

func (g *Action) Size2D() {
	g.InitLayout2D()
	g.Size2DFromText(g.Text)
}

func (g *Action) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *Action) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *Action) ComputeBBox2D(parBBox image.Rectangle) Vec2D {
	return g.ComputeBBox2DBase(parBBox)
}

func (g *Action) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

func (g *Action) Render2D() {
	if g.PushBounds() {
		if !g.HasChildren() {
			g.Render2DDefaultStyle()
		} else {
			// todo: manage stacked layout to select appropriate image based on state
			// return
		}
		g.Render2DChildren()
		g.PopBounds()
	}
}

// render using a default style if not otherwise styled
func (g *Action) Render2DDefaultStyle() {
	if g.PushBounds() {
		st := &g.Style
		g.RenderStdBox(st)
		g.Render2DText(g.Text)
		g.PopBounds()
	}
}

func (g *Action) CanReRender2D() bool {
	return true
}

func (g *Action) FocusChanged2D(gotFocus bool) {
	g.UpdateStart()
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonNormal) // lose any hover state but whatever..
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &Action{}

////////////////////////////////////////////////////////////////////////////////////////
// Separator

// Separator draws a vertical or horizontal line
type Separator struct {
	WidgetBase
	Horiz bool `xml:"horiz" desc:"is this a horizontal separator -- otherwise vertical"`
}

var KiT_Separator = kit.Types.AddType(&Separator{}, nil)

func (g *Separator) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Separator) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Separator) AsLayout2D() *Layout {
	return nil
}

func (g *Separator) Init2D() {
	g.Init2DBase()
}

var SeparatorProps = map[string]interface{}{
	"padding":      "2px",
	"margin":       "2px",
	"font-size":    "24pt",
	"align-vert":   "center",
	"align-horiz":  "center",
	"stroke-width": "2px",
	// todo: dotted
}

func (g *Separator) Style2D() {
	g.Style.SetStyle(nil, &StyleDefault, SeparatorProps)
	g.Style2DWidget()
}

func (g *Separator) Size2D() {
	g.InitLayout2D()
}

func (g *Separator) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DChildren()
}

func (g *Separator) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *Separator) ComputeBBox2D(parBBox image.Rectangle) Vec2D {
	return g.ComputeBBox2DBase(parBBox)
}

func (g *Separator) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

func (g *Separator) Render2D() {
	if g.PushBounds() {
		pc := &g.Paint
		rs := &g.Viewport.Render
		st := &g.Style
		pc.FontStyle = st.Font
		pc.TextStyle = st.Text
		g.RenderStdBox(st)
		pc.StrokeStyle.SetColor(&st.Color) // ink color

		spc := st.BoxSpace()
		pos := g.LayData.AllocPos.AddVal(spc)
		sz := g.LayData.AllocSize.AddVal(-2.0 * spc)

		if g.Horiz {
			pc.DrawLine(rs, pos.X, pos.Y+0.5*sz.Y, pos.X+sz.X, pos.Y+0.5*sz.Y)
		} else {
			pc.DrawLine(rs, pos.X+0.5*sz.X, pos.Y, pos.X+0.5*sz.X, pos.Y+sz.Y)
		}
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *Separator) CanReRender2D() bool {
	return true
}

func (g *Separator) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Separator{}

////////////////////////////////////////////////////////////////////////////////////////
//  Menus

// a menu is a list of Node2D actions, which can contain sub-actions (though
// it can contain anything -- it is just added to a column layout and
// displayed in a popup) -- don't use stretchy sizes in general for these items!
type Menu []Node2D

// menu just pops up a viewport with a layout that draws the supplied actions
// positions are relative to given viewport -- name is relevant base name to
// which Menu is appended
func PopupMenu(menu Menu, x, y int, vp *Viewport2D, name string) *Viewport2D {
	if len(menu) == 0 {
		log.Printf("GoGi PopupMenu: empty menu given\n")
		return nil
	}
	lay := Layout{}
	lay.InitName(&lay, name+"Menu")
	lay.Lay = LayoutCol
	for _, ac := range menu {
		acn := ac.AsNode2D()
		lay.AddChild(acn.This)
	}
	lay.Init2DTree()
	lay.Style2DTree()                            // sufficient to get sizes
	lay.LayData.AllocSize = vp.LayData.AllocSize // give it the whole vp initially
	lay.Size2DTree()                             // collect sizes
	vpsz := lay.LayData.Size.Pref.Min(vp.LayData.AllocSize).ToPoint()
	x = kit.MinInt(x, vp.ViewBox.Size.X-vpsz.X) // fit
	y = kit.MinInt(y, vp.ViewBox.Size.Y-vpsz.Y) // fit
	pvp := NewViewport2D(vpsz.X, vpsz.Y)
	pvp.InitName(pvp, name+"PopupVP")
	pvp.Fill = true
	bitflag.Set(&pvp.NodeFlags, int(VpFlagPopup))
	bitflag.Set(&pvp.NodeFlags, int(VpFlagMenu))
	pvp.ViewBox.Min = image.Point{x, y}
	// note: not setting VpFlagPopopDestroyAll -- we keep the menu list intact
	pvp.Init2D() // todo: these are here for later smarter updates -- redundant now
	pvp.Style2D()
	pvp.AddChild(lay.This)
	vp.PushPopup(pvp)
	return pvp
}

///////////////////////////////////////////////////////////

// MenuButton is a standard command button -- PushMenuButton in Qt Widgets, and MenuButton in Qt Quick
type MenuButton struct {
	Menu Menu
	ButtonBase
}

var KiT_MenuButton = kit.Types.AddType(&MenuButton{}, nil)

// add an action to the menu -- todo: shortcuts
func (g *MenuButton) AddMenuText(txt string, sigTo ki.Ki, fun ki.RecvFun) *Action {
	if g.Menu == nil {
		g.Menu = make(Menu, 0, 10)
	}
	ac := Action{}
	ac.InitName(&ac, txt)
	ac.Text = txt
	ac.SetAsMenu()
	g.Menu = append(g.Menu, ac.This.(Node2D))
	if sigTo != nil && fun != nil {
		ac.ActionSig.Connect(sigTo, fun)
	}
	return &ac
}

func (g *MenuButton) ButtonReleased(where image.Point) {
	wasPressed := (g.State == ButtonDown)
	g.UpdateStart()
	g.SetButtonState(ButtonNormal)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
	}
	g.UpdateEnd()
	PopupMenu(g.Menu, where.X, where.Y, g.Viewport, g.Text)
}

func (g *MenuButton) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *MenuButton) AsViewport2D() *Viewport2D {
	return nil
}

func (g *MenuButton) AsLayout2D() *Layout {
	return nil
}

func (g *MenuButton) Init2D() {
	g.Init2DBase()
	g.ReceiveEventType(oswin.MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*MenuButton) // note: will fail for any derived classes..
		if ok {
			ab.ButtonPressed()
		}
	})
	g.ReceiveEventType(oswin.MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*MenuButton)
		if ok {
			ab.ButtonReleased(d.(oswin.MouseUpEvent).Where)
		}
	})
	g.ReceiveEventType(oswin.MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*MenuButton)
		if ok {
			ab.ButtonEnterHover()
		}
	})
	g.ReceiveEventType(oswin.MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*MenuButton)
		if ok {
			ab.ButtonExitHover()
		}
	})
	g.ReceiveEventType(oswin.KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*MenuButton)
		if ok {
			kt, ok := d.(oswin.KeyTypedEvent)
			if ok {
				// todo: register shortcuts with window, and generalize these keybindings
				kf := KeyFun(kt.Key, kt.Chord)
				if kf == KeyFunSelectItem || kt.Key == "space" {
					ab.ButtonPressed()
					// todo: brief delay??
					ab.ButtonReleased(image.ZP)
				}
			}
		}
	})
}

var MenuButtonProps = []map[string]interface{}{
	{
		"border-width":        units.NewValue(1, units.Px),
		"border-radius":       units.NewValue(4, units.Px),
		"border-color":        "black",
		"border-style":        "solid",
		"padding":             units.NewValue(4, units.Px),
		"margin":              units.NewValue(4, units.Px),
		"box-shadow.h-offset": units.NewValue(4, units.Px),
		"box-shadow.v-offset": units.NewValue(4, units.Px),
		"box-shadow.blur":     units.NewValue(4, units.Px),
		"box-shadow.color":    "#CCC",
		// "font-family":         "Arial", // this is crashing
		"font-size":        units.NewValue(24, units.Pt),
		"text-align":       AlignCenter,
		"vertical-align":   AlignTop,
		"color":            "black",
		"background-color": "#EEF",
	}, { // disabled
		"border-color":     "#BBB",
		"color":            "#AAA",
		"background-color": "#DDD",
	}, { // hover
		"background-color": "#CCF", // todo "darker"
	}, { // focus
		"border-color":     "#EEF",
		"box-shadow.color": "#BBF",
	}, { // press
		"border-color":     "#DDF",
		"color":            "white",
		"background-color": "#008",
	}, { // selected
		"border-color":     "#DDF",
		"color":            "white",
		"background-color": "#00F",
	},
}

func (g *MenuButton) Style2D() {
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.Style.SetStyle(nil, &StyleDefault, MenuButtonProps[ButtonNormal])
	g.Style2DWidget()
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, MenuButtonProps[i])
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *MenuButton) Size2D() {
	g.InitLayout2D()
	g.Size2DFromText(g.Text)
}

func (g *MenuButton) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *MenuButton) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *MenuButton) ComputeBBox2D(parBBox image.Rectangle) Vec2D {
	return g.ComputeBBox2DBase(parBBox)
}

func (g *MenuButton) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

// todo: need color brigher / darker functions

func (g *MenuButton) Render2D() {
	if g.PushBounds() {
		if !g.HasChildren() {
			g.Render2DDefaultStyle()
		} else {
			// todo: manage stacked layout to select appropriate image based on state
			// return
		}
		g.Render2DChildren()
		g.PopBounds()
	}
}

// render using a default style if not otherwise styled
func (g *MenuButton) Render2DDefaultStyle() {
	st := &g.Style
	g.RenderStdBox(st)
	g.Render2DText(g.Text)

}

func (g *MenuButton) CanReRender2D() bool {
	return true
}

func (g *MenuButton) FocusChanged2D(gotFocus bool) {
	g.UpdateStart()
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonNormal) // lose any hover state but whatever..
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &MenuButton{}
