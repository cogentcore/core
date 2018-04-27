// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"

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

// Action is a button widget that can display a text label and / or an icon and / or
// a keyboard shortcut -- this is what is put in menus and toolbars
// todo: need icon
type Action struct {
	ButtonBase
	Data      interface{} `json:"-" xml:"-" desc:"optional data that is sent with the ActionSig when it is emitted"`
	ActionSig ki.Signal   `json:"-" xml:"-" desc:"signal for action -- does not have a signal type, as there is only one type: Action triggered -- data is Data of this action"`
}

var KiT_Action = kit.Types.AddType(&Action{}, ActionProps)

func (n *Action) New() ki.Ki { return &Action{} }

var ActionProps = ki.Props{
	"border-width":     units.NewValue(0, units.Px), // todo: should be default
	"border-radius":    units.NewValue(0, units.Px),
	"border-color":     &Prefs.BorderColor,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(0, units.Px),
	"box-shadow.color": &Prefs.ShadowColor,
	"text-align":       AlignCenter,
	"vertical-align":   AlignTop,
	"background-color": &Prefs.ControlColor,
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &Prefs.IconColor,
		"stroke":  &Prefs.FontColor,
	},
	"#label": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#indicator": ki.Props{
		"width":          units.NewValue(1.5, units.Ex),
		"height":         units.NewValue(1.5, units.Ex),
		"margin":         units.NewValue(0, units.Px),
		"padding":        units.NewValue(0, units.Px),
		"vertical-align": AlignMiddle,
	},
	ButtonSelectors[ButtonActive]: ki.Props{},
	ButtonSelectors[ButtonDisabled]: ki.Props{
		"border-color": "lighter-50",
		"color":        "lighter-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "darker-10",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "lighter-40",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "lighter-90",
		"background-color": "darker-30",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": &Prefs.SelectColor,
	},
}

// ButtonWidget interface

func (g *Action) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

// trigger action signal
func (g *Action) ButtonRelease() {
	wasPressed := (g.State == ButtonDown)
	updt := g.UpdateStart()
	g.SetButtonState(ButtonActive)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ActionSig.Emit(g.This, 0, g.Data)
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), g.Data)
	}
	if g.IsMenu() && g.Viewport != nil {
		win := g.Viewport.Win
		if win != nil {
			win.ClosePopup(g.Viewport) // in case we are a menu popup -- no harm if not
		}
	}
	g.UpdateEnd(updt)
}

// set the text and update button
func (g *Action) SetText(txt string) {
	SetButtonText(g, txt)
}

// set the Icon (could be nil) and update button
func (g *Action) SetIcon(ic *Icon) {
	SetButtonIcon(g, ic)
}

func (g *Action) SetAsMenu() {
	bitflag.Set(&g.Flag, int(ActionFlagMenu))
}

func (g *Action) IsMenu() bool {
	return bitflag.Has(g.Flag, int(ActionFlagMenu))
}

func (g *Action) SetAsButton() {
	bitflag.Clear(&g.Flag, int(ActionFlagMenu))
}

func (g *Action) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
	Init2DButtonEvents(g)
}

func (g *Action) ConfigPartsButton() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, g.Text)
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(g.Icon, g.Text, icIdx, lbIdx)
	if mods {
		g.UpdateEnd(updt)
	}

}

func (g *Action) ConfigPartsMenu() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, g.Text)
	wrIdx := -1
	if len(g.Kids) > 0 { // include a right-wedge indicator for sub-menu
		config.Add(KiT_Stretch, "indic-stretch")
		wrIdx = len(config)
		config.Add(KiT_Icon, "indicator")
	}
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	if mods {
		g.SetProp("max-width", -1)
	}
	g.ConfigPartsSetIconLabel(g.Icon, g.Text, icIdx, lbIdx)
	if wrIdx >= 0 {
		ic := g.Parts.Child(wrIdx).(*Icon)
		if !ic.HasChildren() {
			ic.CopyFromIcon(IconByName("widget-wedge-right"))
			g.StylePart(ic.This)
		}
	}
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *Action) ConfigParts() {
	if g.IsMenu() {
		g.ConfigPartsMenu()
	} else {
		g.ConfigPartsButton()
	}
}

func (g *Action) ConfigPartsIfNeeded() {
	if !g.PartsNeedUpdateIconLabel(g.Icon, g.Text) {
		return
	}
	g.ConfigParts()
}

func (g *Action) Style2D() {
	bitflag.Set(&g.Flag, int(CanFocus))
	g.Style2DWidget()
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = *g.DefaultStyle2DWidget(ButtonSelectors[i], nil)
		g.StateStyles[i].SetStyle(nil, g.StyleProps(ButtonSelectors[i]))
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.ConfigParts()
}

func (g *Action) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded()
	g.Layout2DWidget(parBBox)
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *Action) Render2D() {
	if g.PushBounds() {
		g.Style = g.StateStyles[g.State] // get current styles
		g.ConfigPartsIfNeeded()
		if !g.HasChildren() {
			g.Render2DDefaultStyle()
		} else {
			g.Render2DChildren()
		}
		g.PopBounds()
	}
}

// render using a default style if no children
func (g *Action) Render2DDefaultStyle() {
	st := &g.Style
	g.RenderStdBox(st)
	g.Render2DParts()
}

func (g *Action) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonActive)
	}
	g.UpdateSig()
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

var KiT_Separator = kit.Types.AddType(&Separator{}, SeparatorProps)

func (n *Separator) New() ki.Ki { return &Separator{} }

var SeparatorProps = ki.Props{
	"padding":      units.NewValue(2, units.Px),
	"margin":       units.NewValue(2, units.Px),
	"align-vert":   AlignCenter,
	"align-horiz":  AlignCenter,
	"stroke-width": units.NewValue(2, units.Px),
	// todo: dotted
}

func (g *Separator) Style2D() {
	g.Style2DWidget()
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

// check for interface implementation
var _ Node2D = &Separator{}

////////////////////////////////////////////////////////////////////////////////////////
//  Menus

// a menu is a list of Node2D actions, which can contain sub-actions (though
// it can contain anything -- it is just added to a column layout and
// displayed in a popup) -- don't use stretchy sizes in general for these items!
type Menu []Node2D

var MenuProps = ki.Props{
	"#frame": ki.Props{
		"border-width":        units.NewValue(0, units.Px),
		"border-color":        "none",
		"margin":              units.NewValue(4, units.Px),
		"padding":             units.NewValue(2, units.Px),
		"box-shadow.h-offset": units.NewValue(2, units.Px),
		"box-shadow.v-offset": units.NewValue(2, units.Px),
		"box-shadow.blur":     units.NewValue(2, units.Px),
		"box-shadow.color":    &Prefs.ShadowColor,
	},
}

// menu just pops up a viewport with a layout that draws the supplied actions
// positions are relative to given viewport -- name is relevant base name to
// which Menu is appended
func PopupMenu(menu Menu, x, y int, win *Window, name string) *Viewport2D {
	vp := win.Viewport
	if len(menu) == 0 {
		log.Printf("GoGi PopupMenu: empty menu given\n")
		return nil
	}
	pvp := Viewport2D{}
	pvp.InitName(&pvp, name+"Menu")
	pvp.Win = win
	updt := pvp.UpdateStart()
	pvp.Fill = true
	bitflag.Set(&pvp.Flag, int(VpFlagPopup))
	bitflag.Set(&pvp.Flag, int(VpFlagMenu))
	pvp.ViewBox.Min = image.Point{x, y}
	// note: not setting VpFlagPopopDestroyAll -- we keep the menu list intact
	frame := pvp.AddNewChild(KiT_Frame, "Frame").(*Frame)
	frame.Lay = LayoutCol
	// todo: need this case!
	// frame.StylePart(frame.This, MenuProps)
	for _, ac := range menu {
		acn := ac.AsNode2D()
		frame.AddChild(acn.This)
	}
	frame.Init2DTree()
	frame.Style2DTree()                            // sufficient to get sizes
	frame.LayData.AllocSize = vp.LayData.AllocSize // give it the whole vp initially
	frame.Size2DTree()                             // collect sizes
	pvp.Win = nil
	vpsz := frame.LayData.Size.Pref.Min(vp.LayData.AllocSize).ToPoint()
	x = kit.MinInt(x, vp.ViewBox.Size.X-vpsz.X) // fit
	y = kit.MinInt(y, vp.ViewBox.Size.Y-vpsz.Y) // fit
	pvp.Resize(vpsz.X, vpsz.Y)
	pvp.ViewBox.Min = image.Point{x, y}
	pvp.UpdateEndNoSig(updt)

	win.PushPopup(pvp.This)
	return &pvp
}

////////////////////////////////////////////////////////////////////////////////////////
// MenuButton pops up a menu

// MenuButtonMakeMenuFunc is a callback for making the menu on demand
type MenuButtonMakeMenuFunc func(mb *MenuButton)

type MenuButton struct {
	ButtonBase
	Menu         Menu                   `desc:"the menu items for this menu"`
	MakeMenuFunc MenuButtonMakeMenuFunc `desc:"set this to make the menu on demand"`
}

var KiT_MenuButton = kit.Types.AddType(&MenuButton{}, MenuButtonProps)

func (n *MenuButton) New() ki.Ki { return &MenuButton{} }

var MenuButtonProps = ki.Props{
	"border-width":     units.NewValue(1, units.Px),
	"border-radius":    units.NewValue(4, units.Px),
	"border-color":     &Prefs.BorderColor,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(4, units.Px),
	"margin":           units.NewValue(4, units.Px),
	"box-shadow.color": &Prefs.ShadowColor,
	"text-align":       AlignCenter,
	"vertical-align":   AlignMiddle,
	"background-color": &Prefs.ControlColor,
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &Prefs.IconColor,
		"stroke":  &Prefs.FontColor,
	},
	"#label": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#indicator": ki.Props{
		"width":          units.NewValue(1.5, units.Ex),
		"height":         units.NewValue(1.5, units.Ex),
		"margin":         units.NewValue(0, units.Px),
		"padding":        units.NewValue(0, units.Px),
		"vertical-align": AlignBottom,
		"fill":           &Prefs.IconColor,
		"stroke":         &Prefs.FontColor,
	},
	ButtonSelectors[ButtonActive]: ki.Props{},
	ButtonSelectors[ButtonDisabled]: ki.Props{
		"border-color": "lighter-50",
		"color":        "lighter-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "darker-10",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "lighter-40",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "lighter-90",
		"background-color": "darker-30",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": &Prefs.SelectColor,
	},
}

// ButtonWidget interface

func (g *MenuButton) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

func (g *MenuButton) ButtonRelease() {
	wasPressed := (g.State == ButtonDown)
	updt := g.UpdateStart()
	g.SetButtonState(ButtonActive)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
		if g.MakeMenuFunc != nil {
			g.MakeMenuFunc(g)
		}
		g.UpdateEnd(updt)
		pos := g.WinBBox.Max
		_, indic := KiToNode2D(g.Parts.ChildByName("indicator", 3))
		if indic != nil {
			pos = indic.WinBBox.Min
		} else {
			pos.Y -= 10
			pos.X -= 10
		}
		if g.Viewport != nil {
			PopupMenu(g.Menu, pos.X, pos.Y, g.Viewport.Win, g.Text)
		}
	}
}

// set the text and update button
func (g *MenuButton) SetText(txt string) {
	SetButtonText(g, txt)
}

// set the Icon (could be nil) and update button
func (g *MenuButton) SetIcon(ic *Icon) {
	SetButtonIcon(g, ic)
}

// add an action to the menu -- todo: shortcuts
func (g *MenuButton) AddMenuText(txt string, sigTo ki.Ki, data interface{}, fun ki.RecvFunc) *Action {
	if g.Menu == nil {
		g.Menu = make(Menu, 0, 10)
	}
	ac := Action{}
	ac.InitName(&ac, txt)
	ac.Text = txt
	ac.Data = data
	ac.SetAsMenu()
	g.Menu = append(g.Menu, ac.This.(Node2D))
	if sigTo != nil && fun != nil {
		ac.ActionSig.Connect(sigTo, fun)
	}
	return &ac
}

// remove all items in the menu
func (g *MenuButton) ResetMenu() {
	g.Menu = make(Menu, 0, 10)
}

func (g *MenuButton) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
	Init2DButtonEvents(g)
}

func (g *MenuButton) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, g.Text)
	wrIdx := -1
	icnm := kit.ToString(g.Prop("indicator", false, false))
	if icnm == "" || icnm == "nil" {
		icnm = "widget-wedge-down"
	}
	if icnm != "none" {
		config.Add(KiT_Space, "indic-stretch")
		wrIdx = len(config)
		config.Add(KiT_Icon, "indicator")
	}
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(g.Icon, g.Text, icIdx, lbIdx)
	if wrIdx >= 0 {
		ic := g.Parts.Child(wrIdx).(*Icon)
		if !ic.HasChildren() || ic.UniqueNm != icnm {
			ic.CopyFromIcon(IconByName(icnm))
			ic.UniqueNm = icnm
			g.StylePart(ic.This)
		}
	}
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *MenuButton) ConfigPartsIfNeeded() {
	if !g.PartsNeedUpdateIconLabel(g.Icon, g.Text) {
		return
	}
	g.ConfigParts()
}

func (g *MenuButton) Style2D() {
	bitflag.Set(&g.Flag, int(CanFocus))
	g.Style2DWidget()
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = *g.DefaultStyle2DWidget(ButtonSelectors[i], nil)
		g.StateStyles[i].SetStyle(nil, g.StyleProps(ButtonSelectors[i]))
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.ConfigParts()
}

func (g *MenuButton) Layout2D(parBBox image.Rectangle) {
	g.ConfigParts()
	g.Layout2DWidget(parBBox)
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *MenuButton) Render2D() {
	if g.PushBounds() {
		g.Style = g.StateStyles[g.State] // get current styles
		g.ConfigPartsIfNeeded()
		if !g.HasChildren() {
			g.Render2DDefaultStyle()
		} else {
			g.Render2DChildren()
		}
		g.PopBounds()
	}
}

// render using a default style if not otherwise styled
func (g *MenuButton) Render2DDefaultStyle() {
	st := &g.Style
	g.RenderStdBox(st)
	g.Render2DParts()
}

func (g *MenuButton) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonActive) // lose any hover state but whatever..
	}
	g.UpdateSig()
}

// check for interface implementation
var _ Node2D = &MenuButton{}
