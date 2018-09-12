// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"
	"strings"

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
	Shortcut     key.Chord            `xml:"shortcut" desc:"optional shortcut keyboard chord to trigger this action -- always window-wide in scope, and should generally not conflict other shortcuts (a log message will be emitted if so).  Shortcuts are processed after all other processing of keyboard input.  Use Command for Control / Meta (Mac Command key) per platform"`
	StateStyles  [ButtonStatesN]Style `json:"-" xml:"-" desc:"styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State        ButtonStates         `json:"-" xml:"-" desc:"current state of the button based on gui interaction"`
	ButtonSig    ki.Signal            `json:"-" xml:"-" view:"-" desc:"signal for button -- see ButtonSignals for the types"`
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
func (bb *ButtonBase) IsCheckable() bool {
	return bitflag.Has(bb.Flag, int(ButtonFlagCheckable))
}

// SetCheckable sets whether this button is checkable -- emits ButtonToggled
// signals if so -- the Checked state is independent of the generic widget
// selection state
func (bb *ButtonBase) SetCheckable(checkable bool) {
	bitflag.SetState(&bb.Flag, checkable, int(ButtonFlagCheckable))
}

// IsChecked checks if button is checked
func (bb *ButtonBase) IsChecked() bool {
	return bitflag.Has(bb.Flag, int(ButtonFlagChecked))
}

// SetChecked sets the checked state of this button -- does not emit signal or
// update
func (bb *ButtonBase) SetChecked(chk bool) {
	bitflag.SetState(&bb.Flag, chk, int(ButtonFlagChecked))
}

// ToggleChecked toggles the checked state of this button -- does not emit
// signal or update
func (bb *ButtonBase) ToggleChecked() {
	bb.SetChecked(!bb.IsChecked())
}

// SetAsMenu ensures that this functions as a menu even before menu items are added
func (bb *ButtonBase) SetAsMenu() {
	bitflag.Set(&bb.Flag, int(ButtonFlagMenu))
}

// SetAsButton clears the explicit ButtonFlagMenu -- if there are menu items
// or a menu function then it will still behave as a menu
func (bb *ButtonBase) SetAsButton() {
	bitflag.Clear(&bb.Flag, int(ButtonFlagMenu))
}

// SetText sets the text and updates the button
func (bb *ButtonBase) SetText(txt string) {
	updt := bb.UpdateStart()
	if bb.Sty.Font.Size.Val == 0 { // not yet styled
		bb.StyleButton()
	}
	bb.Text = txt
	bb.This.(ButtonWidget).ConfigParts()
	bb.UpdateEnd(updt)
}

// Label returns the display label for this node, satisfying the Labeler interface
func (bb *ButtonBase) Label() string {
	if bb.Text != "" {
		return bb.Text
	}
	return bb.Nm
}

// SetIcon sets the Icon to given icon name (could be empty or 'none') and
// updates the button
func (bb *ButtonBase) SetIcon(iconName string) {
	updt := bb.UpdateStart()
	if bb.Sty.Font.Size.Val == 0 { // not yet styled
		bb.StyleButton()
	}
	if bb.Icon != IconName(iconName) {
		bb.SetFullReRender()
	}
	bb.Icon = IconName(iconName)
	bb.This.(ButtonWidget).ConfigParts()
	bb.UpdateEnd(updt)
}

// SetButtonState sets the button state -- returns true if state changed
func (bb *ButtonBase) SetButtonState(state ButtonStates) bool {
	prev := bb.State
	if bb.IsInactive() {
		if bb.IsSelected() {
			state = ButtonSelected
		} else {
			state = ButtonInactive
		}
	} else {
		if state == ButtonActive && bb.IsSelected() {
			state = ButtonSelected
		} else if state == ButtonActive && bb.HasFocus() {
			state = ButtonFocus
		}
	}
	bb.State = state
	bb.Sty = bb.StateStyles[state]
	if prev != bb.State {
		bb.SetFullReRenderIconLabel() // needs full rerender to update text, icon
		return true
	}
	return false
}

// UpdateButtonStyle sets the button style based on current state info --
// returns true if changed -- restyles parts if so
func (bb *ButtonBase) UpdateButtonStyle() bool {
	prev := bb.State
	if bb.IsInactive() {
		if bb.IsSelected() {
			bb.State = ButtonSelected
		} else {
			bb.State = ButtonInactive
		}
	} else {
		if bb.State == ButtonSelected && !bb.IsSelected() {
			bb.State = ButtonActive
		} else if bb.State == ButtonActive && bb.IsSelected() {
			bb.State = ButtonSelected
		} else if bb.State == ButtonActive && bb.HasFocus() {
			bb.State = ButtonFocus
		} else if bb.State == ButtonInactive {
			bb.State = ButtonActive
		}
	}
	bb.Sty = bb.StateStyles[bb.State]
	bb.This.(ButtonWidget).ConfigPartsIfNeeded()
	if prev != bb.State {
		bb.SetFullReRenderIconLabel() // needs full rerender
		return true
	}
	// fmt.Printf("but style updt: %v to %v\n", bb.PathUnique(), bb.State)
	return false
}

// ButtonPressed sets the button in the down state -- mouse clicked down but
// not yet up -- emits ButtonPressed signal AND WidgetSig Selected signal --
// ButtonClicked is down and up
func (bb *ButtonBase) ButtonPressed() {
	updt := bb.UpdateStart()
	if bb.IsInactive() {
		if !strings.HasSuffix(bb.Class, "-action") { // not for menu-action, bar-action
			bb.SetSelectedState(!bb.IsSelected())
			bb.EmitSelectedSignal()
			bb.UpdateSig()
		}
	} else {
		bb.SetButtonState(ButtonDown)
		bb.ButtonSig.Emit(bb.This, int64(ButtonPressed), nil)
	}
	bb.UpdateEnd(updt)
}

// ButtonReleased action: the button has just been released -- sends a released
// signal and returns state to normal, and emits clicked signal if if it was
// previously in pressed state
func (bb *ButtonBase) ButtonReleased() {
	if bb.IsInactive() {
		return
	}
	wasPressed := (bb.State == ButtonDown)
	updt := bb.UpdateStart()
	bb.SetButtonState(ButtonActive)
	bb.ButtonSig.Emit(bb.This, int64(ButtonReleased), nil)
	if wasPressed {
		bb.ButtonSig.Emit(bb.This, int64(ButtonClicked), nil)
		bb.OpenMenu()

		if bb.IsCheckable() {
			bb.ToggleChecked()
			bb.ButtonSig.Emit(bb.This, int64(ButtonToggled), nil)
		}
	}
	bb.UpdateEnd(updt)
}

// IsMenu returns true this button is on a menu -- it is a menu item
func (bb *ButtonBase) IsMenu() bool {
	return bitflag.Has(bb.Flag, int(ButtonFlagMenu))
}

// HasMenu returns true if there is a menu or menu-making function set, or the
// explicit ButtonFlagMenu has been set
func (bb *ButtonBase) HasMenu() bool {
	return bb.MakeMenuFunc != nil || len(bb.Menu) > 0
}

// OpenMenu will open any menu associated with this element -- returns true if
// menu opened, false if not
func (bb *ButtonBase) OpenMenu() bool {
	if !bb.HasMenu() {
		return false
	}
	if bb.MakeMenuFunc != nil {
		bb.MakeMenuFunc(bb.This, &bb.Menu)
	}
	pos := bb.WinBBox.Max
	if pos.X == 0 && pos.Y == 0 { // offscreen
		pos = bb.ObjBBox.Max
	}
	indic, ok := bb.Parts.ChildByName("indicator", 3)
	if ok {
		pos = KiToNode2DBase(indic).WinBBox.Min
		if pos.X == 0 && pos.Y == 0 {
			pos = KiToNode2DBase(indic).ObjBBox.Min
		}
	} else {
		pos.X = bb.WinBBox.Min.X
		if pos.X == 0 {
			pos.X = bb.ObjBBox.Min.X
		}
	}
	if bb.Viewport != nil {
		PopupMenu(bb.Menu, pos.X, pos.Y, bb.Viewport, bb.Text)
		return true
	}
	return false
}

// ResetMenu removes all items in the menu
func (bb *ButtonBase) ResetMenu() {
	bb.Menu = make(Menu, 0, 10)
}

// ConfigPartsAddIndicator adds a menu indicator if there is a menu present,
// and the Indicator field is not "none" -- defOn = true means default to
// adding the indicator even if no menu is yet present -- returns the index in
// Parts of the indicator object, which is named "indicator" -- an
// "ind-stretch" is added as well to put on the right by default.
func (bb *ButtonBase) ConfigPartsAddIndicator(config *kit.TypeAndNameList, defOn bool) int {
	needInd := (bb.HasMenu() || defOn) && bb.Indicator != "none"
	if !needInd {
		return -1
	}
	indIdx := -1
	config.Add(KiT_Stretch, "ind-stretch")
	indIdx = len(*config)
	config.Add(KiT_Icon, "indicator")
	return indIdx
}

func (bb *ButtonBase) ConfigPartsIndicator(indIdx int) {
	if indIdx < 0 {
		return
	}
	ic := bb.Parts.KnownChild(indIdx).(*Icon)
	icnm := string(bb.Indicator)
	if IconName(icnm).IsNil() {
		icnm = "widget-wedge-down"
	}
	if set, _ := IconName(icnm).SetIcon(ic); set {
		bb.StylePart(bb.Parts.KnownChild(indIdx - 1).(Node2D)) // also get the stretch
		bb.StylePart(Node2D(ic))
	}
}

// ButtonEnterHover called when button starting hover
func (bb *ButtonBase) ButtonEnterHover() {
	if bb.State != ButtonHover {
		updt := bb.UpdateStart()
		bb.SetButtonState(ButtonHover)
		bb.UpdateEnd(updt)
	}
}

// ButtonExitHover called when button exiting hover
func (bb *ButtonBase) ButtonExitHover() {
	if bb.State == ButtonHover {
		updt := bb.UpdateStart()
		bb.SetButtonState(ButtonActive)
		bb.UpdateEnd(updt)
	}
}

// MouseEvents handles button MouseEvent
func (bb *ButtonBase) MouseEvent() {
	bb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		bw := recv.(ButtonWidget)
		bbb := bw.ButtonAsBase()
		if me.Button == mouse.Left {
			switch me.Action {
			case mouse.DoubleClick: // we just count as a regular click
				fallthrough
			case mouse.Press:
				me.SetProcessed()
				bbb.ButtonPressed()
			case mouse.Release:
				me.SetProcessed()
				bw.ButtonRelease()
			}
		}
	})
}

// MouseFocusEvents handles button MouseFocusEvent
func (bb *ButtonBase) MouseFocusEvent() {
	bb.ConnectEvent(oswin.MouseFocusEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		bw := recv.(ButtonWidget)
		bbb := bw.ButtonAsBase()
		if bbb.IsInactive() {
			return
		}
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
		if me.Action == mouse.Enter {
			bbb.ButtonEnterHover()
		} else {
			bbb.ButtonExitHover()
		}
	})
}

// KeyChordEvent handles button KeyChord events
func (bb *ButtonBase) KeyChordEvent() {
	bb.ConnectEvent(oswin.KeyChordEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		bw := recv.(ButtonWidget)
		bbb := bw.ButtonAsBase()
		if bbb.IsInactive() {
			return
		}
		kt := d.(*key.ChordEvent)
		kf := KeyFun(kt.Chord())
		if kf == KeyFunSelectItem || kf == KeyFunAccept || kt.Rune == ' ' {
			if !(kt.Rune == ' ' && bbb.Viewport.IsCompleter()) {
				kt.SetProcessed()
				bbb.ButtonPressed()
				bw.ButtonRelease()
			}
		}
	})
}

func (bb *ButtonBase) ButtonEvents() {
	bb.HoverTooltipEvent()
	bb.MouseEvent()
	bb.MouseFocusEvent()
	bb.KeyChordEvent()
}

///////////////////////////////////////////////////////////
//   ButtonWidget

// ButtonWidget is an interface for button widgets allowing ButtonBase
// defaults to handle most cases.
type ButtonWidget interface {
	// ButtonAsBase gets the button base for most basic functions -- reduces
	// interface size.
	ButtonAsBase() *ButtonBase

	// ButtonRelease is called for release of button -- this is where buttons
	// actually differ in functionality.
	ButtonRelease()

	// StyleParts is called during Style2D to handle stying associated with
	// parts -- icons mainly.
	StyleParts()

	// ConfigParts configures the parts of the button -- called during init
	// and style.
	ConfigParts()

	// ConfigPartsIfNeeded configures the parts of the button, only if needed
	// -- called during layout and render
	ConfigPartsIfNeeded()
}

///////////////////////////////////////////////////////////
// ButtonBase Node2D and ButtonwWidget interface

func (bb *ButtonBase) ButtonAsBase() *ButtonBase {
	return bb
}

func (bb *ButtonBase) Init2D() {
	bb.Init2DWidget()
	bb.This.(ButtonWidget).ConfigParts()
}

func (bb *ButtonBase) ButtonRelease() {
	bb.ButtonReleased() // do base
}

func (bb *ButtonBase) StyleParts() {
	if pv, ok := bb.PropInherit("indicator", false, true); ok { // no inh, yes type
		pvs := kit.ToString(pv)
		bb.Indicator = IconName(pvs)
	}
	if pv, ok := bb.PropInherit("icon", false, true); ok { // no inh, yes type
		pvs := kit.ToString(pv)
		bb.Icon = IconName(pvs)
	}
}

func (bb *ButtonBase) ConfigParts() {
	bb.Parts.Lay = LayoutHoriz
	config, icIdx, lbIdx := bb.ConfigPartsIconLabel(string(bb.Icon), bb.Text)
	indIdx := bb.ConfigPartsAddIndicator(&config, false) // default off
	mods, updt := bb.Parts.ConfigChildren(config, false) // not unique names
	bb.ConfigPartsSetIconLabel(string(bb.Icon), bb.Text, icIdx, lbIdx)
	bb.ConfigPartsIndicator(indIdx)
	if mods {
		bb.UpdateEnd(updt)
	}
}

func (bb *ButtonBase) ConfigPartsIfNeeded() {
	if !bb.PartsNeedUpdateIconLabel(string(bb.Icon), bb.Text) {
		return
	}
	bb.This.(ButtonWidget).ConfigParts()
}

func (bb *ButtonBase) StyleButton() {
	bb.Style2DWidget()
	bb.This.(ButtonWidget).StyleParts()
	if nf, ok := bb.Prop("no-focus"); ok {
		bitflag.SetState(&bb.Flag, !bb.IsInactive() && !nf.(bool), int(CanFocus))
	} else {
		bitflag.SetState(&bb.Flag, !bb.IsInactive(), int(CanFocus))
	}
	pst := bb.ParentStyle()
	clsty := "." + bb.Class
	var clsp ki.Props
	if clspi, ok := bb.PropInherit(clsty, false, true); ok {
		clsp, ok = clspi.(ki.Props)
	}
	for i := 0; i < int(ButtonStatesN); i++ {
		bb.StateStyles[i].CopyFrom(&bb.Sty)
		bb.StateStyles[i].SetStyleProps(pst, bb.StyleProps(ButtonSelectors[i]))
		if clsp != nil {
			if stclsp, ok := ki.SubProps(clsp, ButtonSelectors[i]); ok {
				bb.StateStyles[i].SetStyleProps(pst, stclsp)
			}
		}
		bb.StateStyles[i].CopyUnitContext(&bb.Sty.UnContext)
	}
}

func (bb *ButtonBase) Style2D() {
	bb.StyleButton()
	bb.LayData.SetFromStyle(&bb.Sty.Layout) // also does reset
	bb.This.(ButtonWidget).ConfigParts()
	bb.SetButtonState(ButtonActive) // initial default
	if bb.Menu != nil {
		bb.Menu.SetShortcuts(bb.ParentWindow())
	}
}

func (bb *ButtonBase) Layout2D(parBBox image.Rectangle, iter int) bool {
	bb.This.(ButtonWidget).ConfigPartsIfNeeded()
	bb.Layout2DBase(parBBox, true, iter) // init style
	bb.Layout2DParts(parBBox, iter)
	for i := 0; i < int(ButtonStatesN); i++ {
		bb.StateStyles[i].CopyUnitContext(&bb.Sty.UnContext)
	}
	return bb.Layout2DChildren(iter)
}

func (bb *ButtonBase) Render2D() {
	if bb.FullReRenderIfNeeded() {
		return
	}
	if bb.PushBounds() {
		bb.This.(Node2D).ConnectEvents2D()
		bb.UpdateButtonStyle()
		st := &bb.Sty
		bb.RenderStdBox(st)
		bb.Render2DParts()
		bb.Render2DChildren()
		bb.PopBounds()
	} else {
		bb.DisconnectAllEvents(RegPri)
	}
}

func (bb *ButtonBase) ConnectEvents2D() {
	bb.ButtonEvents()
}

func (bb *ButtonBase) FocusChanged2D(change FocusChanges) {
	switch change {
	case FocusLost:
		bb.SetButtonState(ButtonActive) // lose any hover state but whatever..
		bb.UpdateSig()
	case FocusGot:
		bb.ScrollToMe()
		bb.SetButtonState(ButtonFocus)
		bb.EmitFocusedSignal()
		bb.UpdateSig()
	case FocusInactive: // don't care..
	case FocusActive:
	}
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
	"border-color":  &Prefs.Colors.Border,
	"border-style":  BorderSolid,
	"padding":       units.NewValue(4, units.Px),
	"margin":        units.NewValue(2, units.Px),
	// "box-shadow.h-offset": units.NewValue(10, units.Px),
	// "box-shadow.v-offset": units.NewValue(10, units.Px),
	// "box-shadow.blur":     units.NewValue(4, units.Px),
	"box-shadow.color": &Prefs.Colors.Shadow,
	"text-align":       AlignCenter,
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &Prefs.Colors.Icon,
		"stroke":  &Prefs.Colors.Font,
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
		"fill":           &Prefs.Colors.Icon,
		"stroke":         &Prefs.Colors.Font,
	},
	"#ind-stretch": ki.Props{
		"width": units.NewValue(1, units.Em),
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
		"background-color": "linear-gradient(pref(Select), highlight-10)",
	},
}

// ButtonWidget interface

func (bb *Button) ButtonAsBase() *ButtonBase {
	return &(bb.ButtonBase)
}

///////////////////////////////////////////////////////////
// CheckBox

// CheckBox toggles between a checked and unchecked state
type CheckBox struct {
	ButtonBase
	IconOff IconName `xml:"icon-off" view:"show-name" desc:"icon to use for the off, unchecked state of the icon -- plain Icon holds the On state -- can be set with icon-off property"`
}

var KiT_CheckBox = kit.Types.AddType(&CheckBox{}, CheckBoxProps)

var CheckBoxProps = ki.Props{
	"icon":             "widget-checked-box",
	"icon-off":         "widget-unchecked-box",
	"text-align":       AlignLeft,
	"color":            &Prefs.Colors.Font,
	"background-color": &Prefs.Colors.Control,
	"margin":           units.NewValue(1, units.Px),
	"padding":          units.NewValue(1, units.Px),
	"border-width":     units.NewValue(0, units.Px),
	"#icon0": ki.Props{
		"width":            units.NewValue(1, units.Em),
		"height":           units.NewValue(1, units.Em),
		"margin":           units.NewValue(0, units.Px),
		"padding":          units.NewValue(0, units.Px),
		"background-color": color.Transparent,
		"fill":             &Prefs.Colors.Control,
		"stroke":           &Prefs.Colors.Font,
	},
	"#icon1": ki.Props{
		"width":            units.NewValue(1, units.Em),
		"height":           units.NewValue(1, units.Em),
		"margin":           units.NewValue(0, units.Px),
		"padding":          units.NewValue(0, units.Px),
		"background-color": color.Transparent,
		"fill":             &Prefs.Colors.Control,
		"stroke":           &Prefs.Colors.Font,
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
		"background-color": &Prefs.Colors.Select,
	},
}

// CheckBoxWidget interface

func (cb *CheckBox) ButtonAsBase() *ButtonBase {
	return &(cb.ButtonBase)
}

func (cb *CheckBox) ButtonRelease() {
	cb.ButtonReleased()
}

// SetIcons sets the Icons (by name) for the On (checked) and Off (unchecked)
// states, and updates button
func (cb *CheckBox) SetIcons(icOn, icOff string) {
	updt := cb.UpdateStart()
	cb.Icon = IconName(icOn)
	cb.IconOff = IconName(icOff)
	cb.This.(ButtonWidget).ConfigParts()
	cb.UpdateEnd(updt)
}

// SetIconProps sets the icon properties from given property list -- parent
// types can use this to set different icon properties
func (cb *CheckBox) SetIconProps(props ki.Props) {
	if icp, has := props["icon"]; has {
		cb.SetProp("icon", icp)
	}
	if icp, has := props["icon-off"]; has {
		cb.SetProp("icon-off", icp)
	}
}

func (cb *CheckBox) Init2D() {
	cb.SetCheckable(true)
	cb.Init2DWidget()
	cb.This.(ButtonWidget).ConfigParts()
}

func (cb *CheckBox) StyleParts() {
	cb.ButtonBase.StyleParts()
	if pv, ok := cb.PropInherit("icon-off", false, true); ok { // no inh, yes type
		pvs := kit.ToString(pv)
		cb.IconOff = IconName(pvs)
	}
}

func (cb *CheckBox) ConfigParts() {
	cb.SetCheckable(true)
	if !cb.Icon.IsValid() {
		cb.Icon = "widget-checked-box" // fallback
	}
	if !cb.IconOff.IsValid() {
		cb.IconOff = "widget-unchecked-box"
	}
	config := kit.TypeAndNameList{}
	icIdx := 0 // always there
	lbIdx := -1
	config.Add(KiT_Layout, "stack")
	if cb.Text != "" {
		config.Add(KiT_Space, "space")
		lbIdx = len(config)
		config.Add(KiT_Label, "label")
	}
	mods, updt := cb.Parts.ConfigChildren(config, false) // not unique names
	ist := cb.Parts.KnownChild(icIdx).(*Layout)
	if mods {
		ist.Lay = LayoutStacked
		ist.SetNChildren(2, KiT_Icon, "icon") // covered by above config update
		icon := ist.KnownChild(0).(*Icon)
		if set, _ := cb.Icon.SetIcon(icon); set {
			cb.StylePart(Node2D(icon))
		}
		icoff := ist.KnownChild(1).(*Icon)
		if set, _ := cb.IconOff.SetIcon(icoff); set {
			cb.StylePart(Node2D(icoff))
		}
	}
	if cb.IsChecked() {
		ist.StackTop = 0
	} else {
		ist.StackTop = 1
	}
	if lbIdx >= 0 {
		lbl := cb.Parts.KnownChild(lbIdx).(*Label)
		if lbl.Text != cb.Text {
			cb.StylePart(cb.Parts.KnownChild(lbIdx - 1).(Node2D)) // also get the space
			cb.StylePart(Node2D(lbl))
			lbl.SetText(cb.Text)
		}
	}
	if mods {
		cb.UpdateEnd(updt)
	}
}

func (cb *CheckBox) ConfigPartsIfNeeded() {
	if !cb.Parts.HasChildren() {
		cb.This.(ButtonWidget).ConfigParts()
	}
	icIdx := 0 // always there
	ist := cb.Parts.KnownChild(icIdx).(*Layout)
	if cb.Icon.IsValid() {
		icon := ist.KnownChild(0).(*Icon)
		if !icon.HasChildren() || icon.UniqueNm != string(cb.Icon) || cb.NeedsFullReRender() {
			if set, _ := cb.Icon.SetIcon(icon); set {
				cb.StylePart(Node2D(icon))
			}
		}
	}
	if cb.IconOff.IsValid() {
		icoff := ist.KnownChild(1).(*Icon)
		if !icoff.HasChildren() || icoff.UniqueNm != string(cb.IconOff) || cb.NeedsFullReRender() {
			if set, _ := cb.IconOff.SetIcon(icoff); set {
				cb.StylePart(Node2D(icoff))
			}
		}
	}
	if cb.IsChecked() {
		ist.StackTop = 0
	} else {
		ist.StackTop = 1
	}
}
