// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/oswin/key"
	"github.com/rcoreilly/goki/gi/oswin/mouse"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
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
	ButtonFlagsN
)

// signals that buttons can send
type ButtonSignals int64

const (
	// ButtonClicked is the main signal to check for normal button activation -- button pressed down and up
	ButtonClicked ButtonSignals = iota
	// button pushed down but not yet up
	ButtonPressed
	// a mouse up event occurred -- typically look at ButtonClicked instead of this one
	ButtonReleased
	// toggled means the checked / unchecked state was toggled -- only sent for buttons with Checkable flag set
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
	// disabled -- not pressable
	ButtonDisabled
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

// Style selector names for the different states: https://www.w3schools.com/cssref/css_selectors.asp
var ButtonSelectors = []string{":active", ":disabled", ":hover", ":focus", ":down", ":selected"}

// ButtonBase has common button functionality -- properties: checkable, checked, autoRepeat, autoRepeatInterval, autoRepeatDelay
type ButtonBase struct {
	WidgetBase
	Text        string               `xml:"text" desc:"label for the button -- if blank then no label is presented"`
	Icon        *Icon                `json:"-" xml:"-" desc:"optional icon for the button -- different buttons can configure this in different ways relative to the text if both are present"`
	Shortcut    string               `xml:"shortcut" desc:"keyboard shortcut -- todo: need to figure out ctrl, alt etc"`
	StateStyles [ButtonStatesN]Style `json:"-" xml:"-" desc:"styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State       ButtonStates         `json:"-" xml:"-" desc:"current state of the button based on gui interaction"`
	ButtonSig   ki.Signal            `json:"-" xml:"-" desc:"signal for button -- see ButtonSignals for the types"`
}

var KiT_ButtonBase = kit.Types.AddType(&ButtonBase{}, ButtonBaseProps)

var ButtonBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

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

// set the button state to target
func (g *ButtonBase) SetButtonState(state ButtonStates) {
	if g.IsReadOnly() {
		state = ButtonDisabled
	} else {
		if state == ButtonActive && g.IsSelected() {
			state = ButtonSelected
		} else if state == ButtonActive && g.HasFocus() {
			state = ButtonFocus
		}
	}
	g.State = state
	g.Style = g.StateStyles[state] // get relevant styles
	// g.Parts.ReStyle2DTree()
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
	if g.IsReadOnly() {
		g.SetButtonState(ButtonActive)
		return
	}
	wasPressed := (g.State == ButtonDown)
	updt := g.UpdateStart()
	g.SetButtonState(ButtonActive)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
		if g.IsCheckable() {
			g.ToggleChecked()
			g.ButtonSig.Emit(g.This, int64(ButtonToggled), nil)
		}
	}
	g.UpdateEnd(updt)
}

// button starting hover-- todo: keep track of time and popup a tooltip -- signal?
func (g *ButtonBase) ButtonEnterHover() {
	if g.IsReadOnly() {
		return
	}
	if g.State != ButtonHover {
		updt := g.UpdateStart()
		g.SetButtonState(ButtonHover)
		g.UpdateEnd(updt)
	}
}

// button exiting hover
func (g *ButtonBase) ButtonExitHover() {
	if g.IsReadOnly() {
		return
	}
	if g.State == ButtonHover {
		updt := g.UpdateStart()
		g.SetButtonState(ButtonActive)
		g.UpdateEnd(updt)
	}
}

// interface for button widgets -- can extend as needed
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

// handles all the basic button events
func Init2DButtonEvents(bw ButtonWidget) {
	g := bw.ButtonAsBase()
	// if g.IsReadOnly() {
	// 	return
	// }
	g.ReceiveEventType(oswin.MouseEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		me.SetProcessed()
		ab := recv.(ButtonWidget)
		if me.Action == mouse.Press {
			ab.ButtonAsBase().ButtonPressed()
		} else {
			ab.ButtonRelease() // special one
		}
	})
	g.ReceiveEventType(oswin.MouseFocusEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
		ab := recv.(ButtonWidget)
		if me.Action == mouse.Enter {
			ab.ButtonAsBase().ButtonEnterHover()
		} else {
			ab.ButtonAsBase().ButtonExitHover()
		}
	})
	g.ReceiveEventType(oswin.KeyChordEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		kt := d.(*key.ChordEvent)
		ab := recv.(ButtonWidget)
		bb := ab.ButtonAsBase()
		// todo: register shortcuts with window, and generalize these keybindings
		kf := KeyFun(kt.ChordString())
		if kf == KeyFunSelectItem || kf == KeyFunAccept || kt.Rune == ' ' {
			kt.SetProcessed()
			bb.ButtonPressed()
			// todo: brief delay??
			ab.ButtonRelease() // special one
		}
	})
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
	ButtonSelectors[ButtonActive]: ki.Props{
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
	},
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

func (g *Button) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

func (g *Button) ButtonRelease() {
	g.ButtonReleased() // do base
}

// set the text and update button
func (g *Button) SetText(txt string) {
	SetButtonText(g, txt)
}

// set the Icon (could be nil) and update button
func (g *Button) SetIcon(ic *Icon) {
	SetButtonIcon(g, ic)
}

func (g *Button) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
	Init2DButtonEvents(g)
}

func (g *Button) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, g.Text)
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(g.Icon, g.Text, icIdx, lbIdx, g.StyleProps(ButtonSelectors[ButtonActive]))
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *Button) ConfigPartsIfNeeded() {
	if !g.PartsNeedUpdateIconLabel(g.Icon, g.Text) {
		return
	}
	g.ConfigParts()
}

func (g *Button) Style2D() {
	bitflag.Set(&g.Flag, int(CanFocus))
	g.Style2DWidget(g.StyleProps(ButtonSelectors[ButtonActive]))
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, g.StyleProps(ButtonSelectors[i]))
		}
	}
	g.ConfigParts()
}

func (g *Button) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded()
	g.Layout2DWidget(parBBox) // lays out parts
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

// todo: need color brigher / darker functions

func (g *Button) Render2D() {
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
func (g *Button) Render2DDefaultStyle() {
	st := &g.Style
	g.RenderStdBox(st)
	g.Render2DParts()
}

func (g *Button) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonActive) // lose any hover state but whatever..
	}
	g.UpdateSig()
}

// check for interface implementation
var _ Node2D = &Button{}

///////////////////////////////////////////////////////////
// CheckBox

// CheckBox toggles between a checked and unchecked state
type CheckBox struct {
	ButtonBase
	IconOff *Icon `json:"-" xml:"-" desc:"icon to use for the off, unchecked state of the icon -- plain Icon holds the On state"`
}

var KiT_CheckBox = kit.Types.AddType(&CheckBox{}, CheckBoxProps)

var CheckBoxProps = ki.Props{
	ButtonSelectors[ButtonActive]: ki.Props{
		"text-align":       AlignLeft,
		"background-color": &Prefs.ControlColor,
		"#icon0": ki.Props{
			"width":   units.NewValue(1, units.Em),
			"height":  units.NewValue(1, units.Em),
			"margin":  units.NewValue(0, units.Px),
			"padding": units.NewValue(0, units.Px),
			"fill":    &Prefs.ControlColor,
			"stroke":  &Prefs.FontColor,
		},
		"#icon1": ki.Props{
			"width":   units.NewValue(1, units.Em),
			"height":  units.NewValue(1, units.Em),
			"margin":  units.NewValue(0, units.Px),
			"padding": units.NewValue(0, units.Px),
			"fill":    &Prefs.ControlColor,
			"stroke":  &Prefs.FontColor,
		},
		"#space": ki.Props{
			"width": units.NewValue(1, units.Ex),
		},
		"#label": ki.Props{
			"margin":  units.NewValue(0, units.Px),
			"padding": units.NewValue(0, units.Px),
		},
	},
	ButtonSelectors[ButtonDisabled]: ki.Props{
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

// set the text and update button
func (g *CheckBox) SetText(txt string) {
	SetButtonText(g, txt)
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
	Init2DButtonEvents(g)
}

func (g *CheckBox) ConfigParts() {
	g.SetCheckable(true)
	if g.Icon == nil { // todo: just use style
		g.Icon = IconByName("widget-checked-box")
	}
	if g.IconOff == nil {
		g.IconOff = IconByName("widget-unchecked-box")
	}
	props := g.StyleProps(ButtonSelectors[ButtonActive])
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
			g.StylePart(icon.This, props)
		}
		icoff := ist.Child(1).(*Icon)
		if !icoff.HasChildren() || icoff.UniqueNm != g.IconOff.UniqueNm { // can't use nm b/c config does
			icoff.CopyFromIcon(g.IconOff)
			icoff.UniqueNm = g.IconOff.UniqueNm
			g.StylePart(icoff.This, props)
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
			g.StylePart(g.Parts.Child(lbIdx-1), props) // also get the space
			g.StylePart(lbl.This, props)
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

func (g *CheckBox) Style2D() {
	bitflag.Set(&g.Flag, int(CanFocus))
	props := g.StyleProps(ButtonSelectors[ButtonActive])
	g.Style2DWidget(props)
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, g.StyleProps(ButtonSelectors[i]))
		}
	}
	g.ConfigParts()
}

func (g *CheckBox) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded()
	g.Layout2DWidget(parBBox) // lays out parts
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *CheckBox) Render2D() {
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
func (g *CheckBox) Render2DDefaultStyle() {
	st := &g.Style
	g.RenderStdBox(st)
	g.Render2DParts()
}

func (g *CheckBox) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonActive) // lose any hover state but whatever..
	}
	g.UpdateSig()
}

// check for interface implementation
var _ Node2D = &CheckBox{}
