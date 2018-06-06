// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Buttons

// these extend NodeBase NodeFlags to hold button state
const (
	// button is selected
	ButtonFlagSelected NodeFlags = NodeFlagsN + iota

	// button is checkable -- enables display of check control
	ButtonFlagCheckable

	// button is checked
	ButtonFlagChecked

	// Menu flag means that the button is a menu item
	ButtonFlagMenu

	ButtonFlagsN
)

// signals that buttons can send
type ButtonSignals int64

const (
	// ButtonClicked is the main signal to check for normal button activation
	// -- button pressed down and up
	ButtonClicked ButtonSignals = iota

	// Pressed means button pushed down but not yet up
	ButtonPressed

	// Released means mose button was released - typically look at
	// ButtonClicked instead of this one
	ButtonReleased

	// Toggled means the checked / unchecked state was toggled -- only sent
	// for buttons with Checkable flag set
	ButtonToggled

	ButtonSignalsN
)

//go:generate stringer -type=ButtonSignals

// https://ux.stackexchange.com/questions/84872/what-is-the-buttons-unpressed-and-unhovered-state-called

// mutually-exclusive button states -- determines appearance
type ButtonStates int32

const (
	// normal active state -- there but not being interacted with
	ButtonActive ButtonStates = iota

	// inactive -- not pressable -- no events
	ButtonInactive

	// mouse is hovering over the button
	ButtonHover

	// button is the focus -- will respond to keyboard input
	ButtonFocus

	// button is currently being pressed down
	ButtonDown

	// button has been selected -- maintains selected state
	ButtonSelected

	// total number of button states
	ButtonStatesN
)

//go:generate stringer -type=ButtonStates

var KiT_ButtonStates = kit.Enums.AddEnumAltLower(ButtonStatesN, false, StylePropProps, "Button")

func (ev ButtonStates) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *ButtonStates) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// Style selector names for the different states: https://www.w3schools.com/cssref/css_selectors.asp
var ButtonSelectors = []string{":active", ":inactive", ":hover", ":focus", ":down", ":selected"}

// todo: autoRepeat, autoRepeatInterval, autoRepeatDelay

// ButtonBase has common button functionality for all buttons, including
// Button, Action, MenuButton, CheckBox, etc
type ButtonBase struct {
	WidgetBase
	Text         string               `xml:"text" desc:"label for the button -- if blank then no label is presented"`
	Icon         *Icon                `json:"-" xml:"-" desc:"optional icon for the button -- different buttons can configure this in different ways relative to the text if both are present"`
	Indicator    string               `xml:"indicator" desc:"name of the menu indicator icon to present, or 'none' -- shown automatically when there are Menu elements present unless 'none' is set"`
	Shortcut     string               `xml:"shortcut" desc:"keyboard shortcut -- todo: need to figure out ctrl, alt etc"`
	StateStyles  [ButtonStatesN]Style `json:"-" xml:"-" desc:"styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State        ButtonStates         `json:"-" xml:"-" desc:"current state of the button based on gui interaction"`
	ButtonSig    ki.Signal            `json:"-" xml:"-" desc:"signal for button -- see ButtonSignals for the types"`
	Menu         ki.Slice             `desc:"the menu items for this menu -- typically add Action elements for menus, along with separators"`
	MakeMenuFunc MakeMenuFunc         `json:"-" xml:"-" desc:"set this to make a menu on demand -- if set then this button acts like a menu button"`
}

var KiT_ButtonBase = kit.Types.AddType(&ButtonBase{}, ButtonBaseProps)

var ButtonBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

// see menus.go for MakeMenuFunc, etc

// is this button selected?
func (g *ButtonBase) IsSelected() bool {
	return bitflag.Has(g.Flag, int(ButtonFlagSelected))
}

// is this button checkable
func (g *ButtonBase) IsCheckable() bool {
	return bitflag.Has(g.Flag, int(ButtonFlagCheckable))
}

// SetCheckable sets whether this button is checkable -- emits ButtonToggled signals if so
func (g *ButtonBase) SetCheckable(checkable bool) {
	bitflag.SetState(&g.Flag, checkable, int(ButtonFlagCheckable))
}

// is this button checked
func (g *ButtonBase) IsChecked() bool {
	return bitflag.Has(g.Flag, int(ButtonFlagChecked))
}

// set the selected state of this button -- does not emit signal or update
func (g *ButtonBase) SetSelected(sel bool) {
	bitflag.SetState(&g.Flag, sel, int(ButtonFlagSelected))
	g.SetButtonState(ButtonActive) // update style
}

// set the checked state of this button -- does not emit signal or update
func (g *ButtonBase) SetChecked(chk bool) {
	bitflag.SetState(&g.Flag, chk, int(ButtonFlagChecked))
}

// ToggleChecked toggles the checked state of this button -- does not emit signal or update
func (g *ButtonBase) ToggleChecked() {
	g.SetChecked(!g.IsChecked())
}

// SetAsMenu ensures that this functions as a menu even before menu items are added
func (g *ButtonBase) SetAsMenu() {
	bitflag.Set(&g.Flag, int(ButtonFlagMenu))
}

// SetAsButton clears the explicit ButtonFlagMenu -- if there are menu items
// or a menu function then it will still behave as a menu
func (g *ButtonBase) SetAsButton() {
	bitflag.Clear(&g.Flag, int(ButtonFlagMenu))
}

// SetText sets the text and updates the button
func (g *ButtonBase) SetText(txt string) {
	SetButtonText(g, txt)
}

// SetIcon sets the Icon (could be nil) and updates the button
func (g *ButtonBase) SetIcon(ic *Icon) {
	SetButtonIcon(g, ic)
}

// set the button state to target
func (g *ButtonBase) SetButtonState(state ButtonStates) {
	if g.IsInactive() {
		state = ButtonInactive
	} else {
		if state == ButtonActive && g.IsSelected() {
			state = ButtonSelected
		} else if state == ButtonActive && g.HasFocus() {
			state = ButtonFocus
		}
	}
	g.State = state
	g.Style = g.StateStyles[state] // get relevant styles
}

// set the button in the down state -- mouse clicked down but not yet up --
// emits ButtonPressed signal -- ButtonClicked is down and up
func (g *ButtonBase) ButtonPressed() {
	updt := g.UpdateStart()
	g.SetButtonState(ButtonDown)
	g.ButtonSig.Emit(g.This, int64(ButtonPressed), nil)
	g.UpdateEnd(updt)
}

// the button has just been released -- sends a released signal and returns
// state to normal, and emits clicked signal if if it was previously in pressed state
func (g *ButtonBase) ButtonReleased() {
	wasPressed := (g.State == ButtonDown)
	updt := g.UpdateStart()
	g.SetButtonState(ButtonActive)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
		g.OpenMenu()

		if g.IsCheckable() {
			g.ToggleChecked()
			g.ButtonSig.Emit(g.This, int64(ButtonToggled), nil)
		}
	}
	g.UpdateEnd(updt)
}

// IsMenu returns true this button is on a menu -- it is a menu item
func (g *ButtonBase) IsMenu() bool {
	return bitflag.Has(g.Flag, int(ButtonFlagMenu))
}

// HasMenu returns true if there is a menu or menu-making function set, or the
// explicit ButtonFlagMenu has been set
func (g *ButtonBase) HasMenu() bool {
	return g.MakeMenuFunc != nil || len(g.Menu) > 0
}

// OpenMenu will open any menu associated with this element -- returns true if
// menu opened, false if not
func (g *ButtonBase) OpenMenu() bool {
	if !g.HasMenu() {
		return false
	}
	if g.MakeMenuFunc != nil {
		g.MakeMenuFunc(g)
	}
	pos := g.WinBBox.Max
	_, indic := KiToNode2D(g.Parts.ChildByName("indicator", 3))
	if indic != nil {
		pos = indic.WinBBox.Min
	} else {
		pos.Y -= 10
		pos.X -= 10
	}
	if g.Viewport != nil {
		PopupMenu(g.Menu, pos.X, pos.Y, g.Viewport, g.Text)
		return true
	}
	return false
}

// AddMenuText adds an action to the menu with a text label -- todo: shortcuts
func (g *ButtonBase) AddMenuText(txt string, sigTo ki.Ki, data interface{}, fun ki.RecvFunc) *Action {
	if g.Menu == nil {
		g.Menu = make(ki.Slice, 0, 10)
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

// AddSeparator adds a separator at the next point in the menu
func (g *ButtonBase) AddSeparator(name string) *Separator {
	if g.Menu == nil {
		g.Menu = make(ki.Slice, 0, 10)
	}
	sp := Separator{}
	if name == "" {
		name = "sep"
	}
	sp.InitName(&sp, name)
	sp.SetProp("min-height", units.NewValue(0.5, units.Em))
	sp.SetProp("max-width", -1)
	sp.Horiz = true
	g.Menu = append(g.Menu, sp.This.(Node2D))
	return &sp
}

// ResetMenu removes all items in the menu
func (g *ButtonBase) ResetMenu() {
	g.Menu = make(ki.Slice, 0, 10)
}

// ConfigPartsAddIndicator adds a menu indicator if there is a menu present,
// and the Indicator field is not "none" -- defOn = true means default to
// adding the indicator even if no menu is yet present -- returns the index in
// Parts of the indicator object, which is named "indicator" -- an
// "indic-stretch" is added as well to put on the right by default
func (g *ButtonBase) ConfigPartsAddIndicator(config *kit.TypeAndNameList, defOn bool) int {
	if !g.HasMenu() && !defOn || g.Indicator == "none" {
		return -1
	}
	indIdx := -1
	config.Add(KiT_Space, "indic-stretch")
	indIdx = len(*config)
	config.Add(KiT_Icon, "indicator")
	return indIdx
}

func (g *ButtonBase) ConfigPartsIndicator(indIdx int) {
	if indIdx < 0 {
		return
	}
	ic := g.Parts.Child(indIdx).(*Icon)
	icnm := g.Indicator
	if icnm == "" || icnm == "nil" {
		icnm = "widget-wedge-down"
	}
	if !ic.HasChildren() || ic.UniqueNm != icnm {
		ic.CopyFromIcon(IconByName(icnm))
		ic.UniqueNm = icnm
		g.StylePart(ic.This)
	}
}

// button starting hover-- todo: keep track of time and popup a tooltip -- signal?
func (g *ButtonBase) ButtonEnterHover() {
	if g.State != ButtonHover {
		updt := g.UpdateStart()
		g.SetButtonState(ButtonHover)
		g.UpdateEnd(updt)
	}
}

// button exiting hover
func (g *ButtonBase) ButtonExitHover() {
	if g.State == ButtonHover {
		updt := g.UpdateStart()
		g.SetButtonState(ButtonActive)
		g.UpdateEnd(updt)
	}
}

// ButtonWidget is an interface for button widgets allowing ButtonBase
// defaults to handle most cases
type ButtonWidget interface {
	// get the button base for most basic functions -- reduces interface size
	ButtonAsBase() *ButtonBase
	// called for release of button -- this is where buttons actually differ in functionality
	ButtonRelease()
	// configure the parts of the button -- called during init and style
	ConfigParts()
	// configure the parts of the button, only if needed -- called during layout and render
	ConfigPartsIfNeeded()
}

// set the text and update button
func SetButtonText(bw ButtonWidget, txt string) {
	g := bw.ButtonAsBase()
	updt := g.UpdateStart()
	g.Text = txt
	bw.ConfigParts()
	g.UpdateEnd(updt)
}

// set the Icon (could be nil) and update button
func SetButtonIcon(bw ButtonWidget, ic *Icon) {
	g := bw.ButtonAsBase()
	updt := g.UpdateStart()
	g.Icon = ic // this is jut the pointer
	bw.ConfigParts()
	g.UpdateEnd(updt)
}

// ButtonEvents handles all the basic button events
func ButtonEvents(bw ButtonWidget) {
	g := bw.ButtonAsBase()
	g.ConnectEventType(oswin.MouseEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		me.SetProcessed()
		ab := recv.(ButtonWidget)
		bb := ab.ButtonAsBase()
		if me.Action == mouse.DoubleClick { // we just count as a regular click
			bb.ButtonPressed()
		} else if me.Action == mouse.Press {
			bb.ButtonPressed()
		} else {
			ab.ButtonRelease() // special one
		}
	})
	g.ConnectEventType(oswin.MouseFocusEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
		ab := recv.(ButtonWidget)
		bb := ab.ButtonAsBase()
		if me.Action == mouse.Enter {
			bb.ButtonEnterHover()
		} else {
			bb.ButtonExitHover()
		}
	})
	g.ConnectEventType(oswin.KeyChordEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		kt := d.(*key.ChordEvent)
		ab := recv.(ButtonWidget)
		bb := ab.ButtonAsBase()
		kf := KeyFun(kt.ChordString())
		if kf == KeyFunSelectItem || kf == KeyFunAccept || kt.Rune == ' ' {
			kt.SetProcessed()
			bb.ButtonPressed()
			// todo: brief delay??
			ab.ButtonRelease() // special one
		}
	})
}

// ButtonBaseDefault is default obj that can be used when property specifies "default"
var ButtonBaseDefault ButtonBase

// ButtonBaseFields contain the StyledFields for ButtonBase type
var ButtonBaseFields = initButtonBase()

func initButtonBase() *StyledFields {
	ButtonBaseDefault = ButtonBase{}
	sf := &StyledFields{}
	sf.Default = &ButtonBaseDefault
	sf.AddField(&ButtonBaseDefault, "Indicator")
	return sf
}

///////////////////////////////////////////////////////////
// ButtonBase Node2D and ButtonwWidget interface

func (g *ButtonBase) ButtonAsBase() *ButtonBase {
	return g
}

func (g *ButtonBase) Init2D() {
	g.Init2DWidget()
	g.This.(ButtonWidget).ConfigParts()
}

func (g *ButtonBase) ButtonRelease() {
	g.ButtonReleased() // do base
}

func (g *ButtonBase) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, g.Text)
	indIdx := g.ConfigPartsAddIndicator(&config, false) // default off
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(g.Icon, g.Text, icIdx, lbIdx)
	g.ConfigPartsIndicator(indIdx)
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *ButtonBase) ConfigPartsIfNeeded() {
	if !g.PartsNeedUpdateIconLabel(g.Icon, g.Text) {
		return
	}
	g.This.(ButtonWidget).ConfigParts()
}

func (g *ButtonBase) Style2DWidget() {
	g.WidgetBase.Style2DWidget()
	ButtonBaseFields.Style(g, nil, g.Props)
	ButtonBaseFields.ToDots(g, &g.Style.UnContext)
}

func (g *ButtonBase) Style2D() {
	g.SetCanFocusIfActive()
	g.Style2DWidget()
	var pst *Style
	_, pg := KiToNode2D(g.Par)
	if pg != nil {
		pst = &pg.Style
	}
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyFrom(&g.Style)
		g.StateStyles[i].SetStyle(pst, g.StyleProps(ButtonSelectors[i]))
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.This.(ButtonWidget).ConfigParts()
	g.SetButtonState(ButtonActive) // initial default
}

func (g *ButtonBase) Layout2D(parBBox image.Rectangle) {
	g.This.(ButtonWidget).ConfigPartsIfNeeded()
	g.Layout2DWidget(parBBox) // lays out parts
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *ButtonBase) Render2D() {
	if g.PushBounds() {
		ButtonEvents(g)
		g.Style = g.StateStyles[g.State] // get current styles
		g.This.(ButtonWidget).ConfigPartsIfNeeded()
		if !g.HasChildren() {
			g.Render2DDefaultStyle()
		} else {
			g.Render2DChildren()
		}
		g.PopBounds()
	} else {
		g.DisconnectAllEvents()
	}
}

func (g *ButtonBase) Render2DDefaultStyle() {
	st := &g.Style
	g.RenderStdBox(st)
	g.Render2DParts()
}

func (g *ButtonBase) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonActive) // lose any hover state but whatever..
	}
	g.UpdateSig()
}

///////////////////////////////////////////////////////////
// Button

// Button is a standard command button -- PushButton in Qt Widgets, and Button
// in Qt Quick -- by default it puts the icon to the left and the text to the
// right
type Button struct {
	ButtonBase
}

var KiT_Button = kit.Types.AddType(&Button{}, ButtonProps)

var ButtonProps = ki.Props{
	"border-width":        units.NewValue(1, units.Px),
	"border-radius":       units.NewValue(4, units.Px),
	"border-color":        &Prefs.BorderColor,
	"border-style":        BorderSolid,
	"padding":             units.NewValue(4, units.Px),
	"margin":              units.NewValue(4, units.Px),
	"box-shadow.h-offset": units.NewValue(4, units.Px),
	"box-shadow.v-offset": units.NewValue(4, units.Px),
	"box-shadow.blur":     units.NewValue(4, units.Px),
	"box-shadow.color":    &Prefs.ShadowColor,
	"text-align":          AlignCenter,
	"vertical-align":      AlignTop,
	"background-color":    &Prefs.ControlColor,
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
	ButtonSelectors[ButtonInactive]: ki.Props{
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

func (g *Button) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

///////////////////////////////////////////////////////////
// CheckBox

// CheckBox toggles between a checked and unchecked state
type CheckBox struct {
	ButtonBase
	IconOff *Icon `json:"-" xml:"-" desc:"icon to use for the off, unchecked state of the icon -- plain Icon holds the On state"`
}

var KiT_CheckBox = kit.Types.AddType(&CheckBox{}, CheckBoxProps)

var CheckBoxProps = ki.Props{
	"text-align":       AlignLeft,
	"background-color": &Prefs.ControlColor,
	"#icon0": ki.Props{
		"width":            units.NewValue(1, units.Em),
		"height":           units.NewValue(1, units.Em),
		"margin":           units.NewValue(0, units.Px),
		"padding":          units.NewValue(0, units.Px),
		"background-color": color.Transparent,
		"fill":             &Prefs.ControlColor,
		"stroke":           &Prefs.FontColor,
	},
	"#icon1": ki.Props{
		"width":            units.NewValue(1, units.Em),
		"height":           units.NewValue(1, units.Em),
		"margin":           units.NewValue(0, units.Px),
		"padding":          units.NewValue(0, units.Px),
		"background-color": color.Transparent,
		"fill":             &Prefs.ControlColor,
		"stroke":           &Prefs.FontColor,
	},
	"#space": ki.Props{
		"width": units.NewValue(1, units.Ex),
	},
	"#label": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	ButtonSelectors[ButtonActive]: ki.Props{},
	ButtonSelectors[ButtonInactive]: ki.Props{
		"border-color": "lighter-50",
		"color":        "lighter-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "darker-10",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "lighter-20",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "lighter-90",
		"background-color": "darker-30",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": &Prefs.SelectColor,
	},
}

// CheckBoxWidget interface

func (g *CheckBox) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

func (g *CheckBox) ButtonRelease() {
	g.ButtonReleased()
}

// set the Icons for the On (checked) and Off (unchecked) states, and updates button
func (g *CheckBox) SetIcons(icOn, icOff *Icon) {
	updt := g.UpdateStart()
	g.Icon = icOn
	g.IconOff = icOff
	g.ConfigParts()
	g.UpdateEnd(updt)
}

func (g *CheckBox) Init2D() {
	g.SetCheckable(true)
	g.Init2DWidget()
	g.ConfigParts()
}

func (g *CheckBox) ConfigParts() {
	g.SetCheckable(true)
	if g.Icon == nil { // todo: just use style
		g.Icon = IconByName("widget-checked-box")
	}
	if g.IconOff == nil {
		g.IconOff = IconByName("widget-unchecked-box")
	}
	config := kit.TypeAndNameList{}
	icIdx := 0 // always there
	lbIdx := -1
	config.Add(KiT_Layout, "stack")
	if g.Text != "" {
		config.Add(KiT_Space, "space")
		lbIdx = len(config)
		config.Add(KiT_Label, "label")
	}
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	ist := g.Parts.Child(icIdx).(*Layout)
	if mods {
		ist.Lay = LayoutStacked
		ist.SetNChildren(2, KiT_Icon, "icon") // covered by above config update
		icon := ist.Child(0).(*Icon)
		if !icon.HasChildren() || icon.UniqueNm != g.Icon.UniqueNm { // can't use nm b/c config does
			icon.CopyFromIcon(g.Icon)
			icon.UniqueNm = g.Icon.UniqueNm
			g.StylePart(icon.This)
		}
		icoff := ist.Child(1).(*Icon)
		if !icoff.HasChildren() || icoff.UniqueNm != g.IconOff.UniqueNm { // can't use nm b/c config does
			icoff.CopyFromIcon(g.IconOff)
			icoff.UniqueNm = g.IconOff.UniqueNm
			g.StylePart(icoff.This)
		}
	}
	if g.IsChecked() {
		ist.ShowChildAtIndex(0)
	} else {
		ist.ShowChildAtIndex(1)
	}
	if lbIdx >= 0 {
		lbl := g.Parts.Child(lbIdx).(*Label)
		if lbl.Text != g.Text {
			g.StylePart(g.Parts.Child(lbIdx - 1)) // also get the space
			g.StylePart(lbl.This)
			lbl.Text = g.Text
		}
	}
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *CheckBox) ConfigPartsIfNeeded() {
	if !g.Parts.HasChildren() {
		g.ConfigParts()
	}
	icIdx := 0 // always there
	ist := g.Parts.Child(icIdx).(*Layout)
	if g.IsChecked() {
		ist.ShowChildAtIndex(0)
	} else {
		ist.ShowChildAtIndex(1)
	}
}
