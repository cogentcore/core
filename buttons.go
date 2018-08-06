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

// todo: autoRepeat, autoRepeatInterval, autoRepeatDelay

// ButtonBase has common button functionality for all buttons, including
// Button, Action, MenuButton, CheckBox, etc
type ButtonBase struct {
	PartsWidgetBase
	Text         string               `xml:"text" desc:"label for the button -- if blank then no label is presented"`
	Icon         IconName             `xml:"icon" view:"show-name" desc:"optional icon for the button -- different buttons can configure this in different ways relative to the text if both are present"`
	Indicator    IconName             `xml:"indicator" view:"show-name" desc:"name of the menu indicator icon to present, or blank or 'nil' or 'none' -- shown automatically when there are Menu elements present unless 'none' is set"`
	Shortcut     string               `xml:"shortcut" desc:"keyboard shortcut -- todo: need to figure out ctrl, alt etc"`
	StateStyles  [ButtonStatesN]Style `json:"-" xml:"-" desc:"styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State        ButtonStates         `json:"-" xml:"-" desc:"current state of the button based on gui interaction"`
	ButtonSig    ki.Signal            `json:"-" xml:"-" desc:"signal for button -- see ButtonSignals for the types"`
	Menu         Menu                 `desc:"the menu items for this menu -- typically add Action elements for menus, along with separators"`
	MakeMenuFunc MakeMenuFunc         `json:"-" xml:"-" view:"-" desc:"set this to make a menu on demand -- if set then this button acts like a menu button"`
}

var KiT_ButtonBase = kit.Types.AddType(&ButtonBase{}, ButtonBaseProps)

var ButtonBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

// these extend NodeBase NodeFlags to hold button state
const (
	// button is checkable -- enables display of check control
	ButtonFlagCheckable NodeFlags = NodeFlagsN + iota

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

	// button has been selected -- selection is a general widget property used
	// by views and other complex widgets -- checked state is independent of this
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

// see menus.go for MakeMenuFunc, etc

// IsCheckable returns if is this button checkable -- the Checked state is
// independent of the generic widget selection state
func (g *ButtonBase) IsCheckable() bool {
	return bitflag.Has(g.Flag, int(ButtonFlagCheckable))
}

// SetCheckable sets whether this button is checkable -- emits ButtonToggled
// signals if so -- the Checked state is independent of the generic widget
// selection state
func (g *ButtonBase) SetCheckable(checkable bool) {
	bitflag.SetState(&g.Flag, checkable, int(ButtonFlagCheckable))
}

// IsChecked checks if button is checked
func (g *ButtonBase) IsChecked() bool {
	return bitflag.Has(g.Flag, int(ButtonFlagChecked))
}

// SetChecked sets the checked state of this button -- does not emit signal or
// update
func (g *ButtonBase) SetChecked(chk bool) {
	bitflag.SetState(&g.Flag, chk, int(ButtonFlagChecked))
}

// ToggleChecked toggles the checked state of this button -- does not emit
// signal or update
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

// SetIcon sets the Icon to given icon name (could be empty or 'none') and
// updates the button
func (g *ButtonBase) SetIcon(iconName string) {
	SetButtonIcon(g, iconName)
}

// SetButtonState sets the button state
func (g *ButtonBase) SetButtonState(state ButtonStates) {
	if g.IsInactive() {
		if g.IsSelected() {
			state = ButtonSelected
		} else {
			state = ButtonInactive
		}
	} else {
		if state == ButtonActive && g.IsSelected() {
			state = ButtonSelected
		} else if state == ButtonActive && g.HasFocus() {
			state = ButtonFocus
		}
	}
	g.State = state
	g.Sty = g.StateStyles[state]
}

// UpdateButtonStyle sets the button style based on current state info
func (g *ButtonBase) UpdateButtonStyle() {
	if g.IsInactive() {
		if g.IsSelected() {
			g.State = ButtonSelected
		} else {
			g.State = ButtonInactive
		}
	} else {
		if g.State == ButtonSelected && !g.IsSelected() {
			g.State = ButtonActive
		} else if g.State == ButtonActive && g.IsSelected() {
			g.State = ButtonSelected
		} else if g.State == ButtonActive && g.HasFocus() {
			g.State = ButtonFocus
		}
	}
	g.Sty = g.StateStyles[g.State]
}

// ButtonPressed sets the button in the down state -- mouse clicked down but
// not yet up -- emits ButtonPressed signal AND WidgetSig Selected signal --
// ButtonClicked is down and up
func (g *ButtonBase) ButtonPressed() {
	updt := g.UpdateStart()
	if g.IsInactive() {
		g.SetSelectedState(!g.IsSelected())
		g.EmitSelectedSignal()
		g.UpdateSig()
	} else {
		g.SetButtonState(ButtonDown)
		g.ButtonSig.Emit(g.This, int64(ButtonPressed), nil)
	}
	g.UpdateEnd(updt)
}

// ButtonReleased action: the button has just been released -- sends a released
// signal and returns state to normal, and emits clicked signal if if it was
// previously in pressed state
func (g *ButtonBase) ButtonReleased() {
	if g.IsInactive() {
		return
	}
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
		g.MakeMenuFunc(&g.Menu)
	}
	pos := g.WinBBox.Max
	indic, ok := g.Parts.ChildByName("indicator", 3)
	if ok {
		pos = KiToNode2DBase(indic).WinBBox.Min
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

// ResetMenu removes all items in the menu
func (g *ButtonBase) ResetMenu() {
	g.Menu = make(Menu, 0, 10)
}

// ConfigPartsAddIndicator adds a menu indicator if there is a menu present,
// and the Indicator field is not "none" -- defOn = true means default to
// adding the indicator even if no menu is yet present -- returns the index in
// Parts of the indicator object, which is named "indicator" -- an
// "indic-stretch" is added as well to put on the right by default
func (g *ButtonBase) ConfigPartsAddIndicator(config *kit.TypeAndNameList, defOn bool) int {
	needInd := (g.HasMenu() || defOn) && g.Indicator != "none"
	if !needInd {
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
	ic := g.Parts.KnownChild(indIdx).(*Icon)
	icnm := string(g.Indicator)
	if IconName(icnm).IsNil() {
		icnm = "widget-wedge-down"
	}
	if set, _ := IconName(icnm).SetIcon(ic); set {
		g.StylePart(Node2D(ic))
	}
}

// ButtonEnterHover called when button starting hover
func (g *ButtonBase) ButtonEnterHover() {
	if g.State != ButtonHover {
		updt := g.UpdateStart()
		g.SetButtonState(ButtonHover)
		g.UpdateEnd(updt)
	}
}

// ButtonExitHover called when button exiting hover
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

// SetButtonText set the text and update button
func SetButtonText(bw ButtonWidget, txt string) {
	g := bw.ButtonAsBase()
	updt := g.UpdateStart()
	g.Text = txt
	g.This.(ButtonWidget).ConfigParts()
	g.UpdateEnd(updt)
}

// SetButtonIcon sets the Icon (looked up by name) (could be empty or 'nil' or
// 'none') and updates button
func SetButtonIcon(bw ButtonWidget, iconName string) {
	g := bw.ButtonAsBase()
	updt := g.UpdateStart()
	if g.Icon != IconName(iconName) {
		g.SetFullReRender()
	}
	g.Icon = IconName(iconName)
	g.This.(ButtonWidget).ConfigParts()
	g.UpdateEnd(updt)
}

// ButtonEvents handles all the basic button events
func ButtonEvents(bw ButtonWidget) {
	g := bw.ButtonAsBase()
	g.HoverTooltipEvent()
	g.ConnectEventType(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		ab := recv.(ButtonWidget)
		bb := ab.ButtonAsBase()
		if me.Button == mouse.Left {
			switch me.Action {
			case mouse.DoubleClick: // we just count as a regular click
				fallthrough
			case mouse.Press:
				me.SetProcessed()
				bb.ButtonPressed()
			case mouse.Release:
				me.SetProcessed()
				ab.ButtonRelease()
			}
		}
	})
	g.ConnectEventType(oswin.MouseFocusEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := recv.(ButtonWidget)
		bb := ab.ButtonAsBase()
		if bb.IsInactive() {
			return
		}
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
		if me.Action == mouse.Enter {
			bb.ButtonEnterHover()
		} else {
			bb.ButtonExitHover()
		}
	})
	g.ConnectEventType(oswin.KeyChordEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := recv.(ButtonWidget)
		bb := ab.ButtonAsBase()
		if bb.IsInactive() {
			return
		}
		kt := d.(*key.ChordEvent)
		kf := KeyFun(kt.ChordString())
		if kf == KeyFunSelectItem || kf == KeyFunAccept || kt.Rune == ' ' {
			if !(kt.Rune == ' ' && bb.Viewport.IsCompleter()) {
				kt.SetProcessed()
				bb.ButtonPressed()
				ab.ButtonRelease()
			}
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
	g.Parts.Lay = LayoutHoriz
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(string(g.Icon), g.Text)
	indIdx := g.ConfigPartsAddIndicator(&config, false) // default off
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(string(g.Icon), g.Text, icIdx, lbIdx)
	g.ConfigPartsIndicator(indIdx)
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *ButtonBase) ConfigPartsIfNeeded() {
	if !g.PartsNeedUpdateIconLabel(string(g.Icon), g.Text) {
		return
	}
	g.This.(ButtonWidget).ConfigParts()
}

func (g *ButtonBase) Style2DWidget() {
	g.WidgetBase.Style2DWidget()
	ButtonBaseFields.Style(g, nil, g.Props)
	ButtonBaseFields.ToDots(g, &g.Sty.UnContext)
}

func (g *ButtonBase) Style2D() {
	g.SetCanFocusIfActive()
	g.Style2DWidget()
	pst := g.Par.(Styler).Style()
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyFrom(&g.Sty)
		g.StateStyles[i].SetStyleProps(pst, g.StyleProps(ButtonSelectors[i]))
		g.StateStyles[i].CopyUnitContext(&g.Sty.UnContext)
	}
	g.This.(ButtonWidget).ConfigParts()
	g.SetButtonState(ButtonActive) // initial default
}

func (g *ButtonBase) Layout2D(parBBox image.Rectangle) {
	g.This.(ButtonWidget).ConfigPartsIfNeeded()
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DParts(parBBox)
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Sty.UnContext)
	}
	g.Layout2DChildren()
}

func (g *ButtonBase) Render2D() {
	if g.FullReRenderIfNeeded() {
		return
	}
	if g.PushBounds() {
		ButtonEvents(g)
		g.UpdateButtonStyle()
		g.This.(ButtonWidget).ConfigPartsIfNeeded()
		st := &g.Sty
		g.RenderStdBox(st)
		g.Render2DParts()
		g.Render2DChildren()
		g.PopBounds()
	} else {
		g.DisconnectAllEvents(RegPri)
	}
}

func (g *ButtonBase) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.SetButtonState(ButtonFocus)
		g.EmitFocusedSignal()
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
	"border-width":  units.NewValue(1, units.Px),
	"border-radius": units.NewValue(4, units.Px),
	"border-color":  &Prefs.BorderColor,
	"border-style":  BorderSolid,
	"padding":       units.NewValue(4, units.Px),
	"margin":        units.NewValue(4, units.Px),
	// "box-shadow.h-offset": units.NewValue(10, units.Px),
	// "box-shadow.v-offset": units.NewValue(10, units.Px),
	// "box-shadow.blur":     units.NewValue(4, units.Px),
	"box-shadow.color": &Prefs.ShadowColor,
	"text-align":       AlignCenter,
	"background-color": &Prefs.ControlColor,
	"color":            &Prefs.FontColor,
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
	ButtonSelectors[ButtonActive]: ki.Props{
		"background-color": "linear-gradient(lighter-0, highlight-10)",
	},
	ButtonSelectors[ButtonInactive]: ki.Props{
		"border-color": "lighter-50",
		"color":        "lighter-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "linear-gradient(highlight-10, highlight-10)",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "linear-gradient(samelight-50, highlight-10)",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "lighter-90",
		"background-color": "linear-gradient(highlight-30, highlight-10)",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": "linear-gradient(pref(SelectColor), highlight-10)",
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
	IconOff IconName `desc:"icon to use for the off, unchecked state of the icon -- plain Icon holds the On state"`
}

var KiT_CheckBox = kit.Types.AddType(&CheckBox{}, CheckBoxProps)

var CheckBoxProps = ki.Props{
	"text-align":       AlignLeft,
	"color":            &Prefs.FontColor,
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
	ButtonSelectors[ButtonActive]: ki.Props{
		"background-color": "lighter-0",
	},
	ButtonSelectors[ButtonInactive]: ki.Props{
		"border-color": "highlight-50",
		"color":        "highlight-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "highlight-10",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "samelight-50",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "highlight-90",
		"background-color": "highlight-30",
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

// SetIcons sets the Icons (by name) for the On (checked) and Off (unchecked)
// states, and updates button
func (g *CheckBox) SetIcons(icOn, icOff string) {
	updt := g.UpdateStart()
	g.Icon = IconName(icOn)
	g.IconOff = IconName(icOff)
	g.This.(ButtonWidget).ConfigParts()
	g.UpdateEnd(updt)
}

func (g *CheckBox) Init2D() {
	g.SetCheckable(true)
	g.Init2DWidget()
	g.This.(ButtonWidget).ConfigParts()
}

func (g *CheckBox) ConfigParts() {
	g.SetCheckable(true)
	if !g.Icon.IsValid() { // todo: just use style
		g.Icon = "widget-checked-box"
	}
	if !g.IconOff.IsValid() {
		g.IconOff = "widget-unchecked-box"
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
	ist := g.Parts.KnownChild(icIdx).(*Layout)
	if mods {
		ist.Lay = LayoutStacked
		ist.SetNChildren(2, KiT_Icon, "icon") // covered by above config update
		icon := ist.KnownChild(0).(*Icon)
		if set, _ := g.Icon.SetIcon(icon); set {
			g.StylePart(Node2D(icon))
		}
		icoff := ist.KnownChild(1).(*Icon)
		if set, _ := g.IconOff.SetIcon(icoff); set {
			g.StylePart(Node2D(icoff))
		}
	}
	if g.IsChecked() {
		ist.StackTop = 0
	} else {
		ist.StackTop = 1
	}
	if lbIdx >= 0 {
		lbl := g.Parts.KnownChild(lbIdx).(*Label)
		if lbl.Text != g.Text {
			g.StylePart(g.Parts.KnownChild(lbIdx - 1).(Node2D)) // also get the space
			g.StylePart(Node2D(lbl))
			lbl.SetText(g.Text)
		}
	}
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *CheckBox) ConfigPartsIfNeeded() {
	if !g.Parts.HasChildren() {
		g.This.(ButtonWidget).ConfigParts()
	}
	icIdx := 0 // always there
	ist := g.Parts.KnownChild(icIdx).(*Layout)
	if g.IsChecked() {
		ist.StackTop = 0
	} else {
		ist.StackTop = 1
	}
}
