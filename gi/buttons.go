// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strings"
	"sync"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/gist/colors"
	"github.com/goki/gi/icons"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// todo: autoRepeat, autoRepeatInterval, autoRepeatDelay

// ButtonBase has common button functionality for all buttons, including
// Button, Action, MenuButton, CheckBox, etc
type ButtonBase struct {
	PartsWidgetBase

	// label for the button -- if blank then no label is presented
	Text string `xml:"text" desc:"label for the button -- if blank then no label is presented"`

	// [view: show-name] optional icon for the button -- different buttons can configure this in different ways relative to the text if both are present
	Icon icons.Icon `xml:"icon" view:"show-name" desc:"optional icon for the button -- different buttons can configure this in different ways relative to the text if both are present"`

	// [view: show-name] name of the menu indicator icon to present, or blank or 'nil' or 'none' -- shown automatically when there are Menu elements present unless 'none' is set
	Indicator icons.Icon `xml:"indicator" view:"show-name" desc:"name of the menu indicator icon to present, or blank or 'nil' or 'none' -- shown automatically when there are Menu elements present unless 'none' is set"`

	// optional shortcut keyboard chord to trigger this action -- always window-wide in scope, and should generally not conflict other shortcuts (a log message will be emitted if so).  Shortcuts are processed after all other processing of keyboard input.  Use Command for Control / Meta (Mac Command key) per platform.  These are only set automatically for Menu items, NOT for items in ToolBar or buttons somewhere, but the tooltip for buttons will show the shortcut if set.
	Shortcut key.Chord `xml:"shortcut" desc:"optional shortcut keyboard chord to trigger this action -- always window-wide in scope, and should generally not conflict other shortcuts (a log message will be emitted if so).  Shortcuts are processed after all other processing of keyboard input.  Use Command for Control / Meta (Mac Command key) per platform.  These are only set automatically for Menu items, NOT for items in ToolBar or buttons somewhere, but the tooltip for buttons will show the shortcut if set."`

	// styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that
	StateStyles [ButtonStatesN]gist.Style `copy:"-" json:"-" xml:"-" desc:"styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`

	// current state of the button based on gui interaction
	State ButtonStates `copy:"-" json:"-" xml:"-" desc:"current state of the button based on gui interaction"`

	// [view: -] signal for button -- see ButtonSignals for the types
	ButtonSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for button -- see ButtonSignals for the types"`

	// the menu items for this menu -- typically add Action elements for menus, along with separators
	Menu Menu `desc:"the menu items for this menu -- typically add Action elements for menus, along with separators"`

	// [view: -] set this to make a menu on demand -- if set then this button acts like a menu button
	MakeMenuFunc MakeMenuFunc `copy:"-" json:"-" xml:"-" view:"-" desc:"set this to make a menu on demand -- if set then this button acts like a menu button"`

	// [view: -] button state mutex
	ButStateMu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"button state mutex"`
}

var TypeButtonBase = kit.Types.AddType(&ButtonBase{}, ButtonBaseProps)

var ButtonBaseProps = ki.Props{
	"base-type":     true, // excludes type from user selections
	ki.EnumTypeFlag: TypeButtonFlags,
}

func (bb *ButtonBase) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*ButtonBase)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier ButtonBase one\n", ki.Type(bb).Name())
		ki.GenCopyFieldsFrom(bb.This(), frm)
		return
	}
	bb.PartsWidgetBase.CopyFieldsFrom(&fr.PartsWidgetBase)
	bb.Text = fr.Text
	bb.Icon = fr.Icon
	bb.Indicator = fr.Indicator
	bb.Shortcut = fr.Shortcut
	bb.Menu = fr.Menu
}

func (bb *ButtonBase) Disconnect() {
	bb.PartsWidgetBase.Disconnect()
	bb.ButtonSig.DisconnectAll()
}

// ButtonFlags extend NodeBase NodeFlags to hold button state
type ButtonFlags int

var TypeButtonFlags = kit.Enums.AddEnumExt(TypeNodeFlags, ButtonFlagsN, kit.BitFlag, nil)

const (
	// button is checkable -- enables display of check control
	ButtonFlagCheckable ButtonFlags = ButtonFlags(NodeFlagsN) + iota

	// button is checked
	ButtonFlagChecked

	// Menu flag means that the button is a menu item
	ButtonFlagMenu

	ButtonFlagsN
)

// ButtonSignals are signals that buttons can send
type ButtonSignals int64

const (
	// ButtonClicked is the main signal to check for normal button activation
	// -- button pressed down and up
	ButtonClicked ButtonSignals = iota

	// Pressed means button pushed down but not yet up
	ButtonPressed

	// Released means mouse button was released - typically look at
	// ButtonClicked instead of this one
	ButtonReleased

	// Toggled means the checked / unchecked state was toggled -- only sent
	// for buttons with Checkable flag set
	ButtonToggled

	ButtonSignalsN
)

// https://ux.stackexchange.com/questions/84872/what-is-the-buttons-unpressed-and-unhovered-state-called

// mutually-exclusive button states -- determines appearance
type ButtonStates int32

var TypeButtonStates = kit.Enums.AddEnumAltLower(ButtonStatesN, kit.NotBitFlag, gist.StylePropProps, "Button")

func (ev ButtonStates) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *ButtonStates) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

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

// Style selector names for the different states: https://www.w3schools.com/cssref/css_selectors.asp
var ButtonSelectors = []string{":active", ":inactive", ":hover", ":focus", ":down", ":selected"}

// see menus.go for MakeMenuFunc, etc

// IsCheckable returns if is this button checkable -- the Checked state is
// independent of the generic widget selection state
func (bb *ButtonBase) IsCheckable() bool {
	return bb.HasFlag(int(ButtonFlagCheckable))
}

// SetCheckable sets whether this button is checkable -- emits ButtonToggled
// signals if so -- the Checked state is independent of the generic widget
// selection state
func (bb *ButtonBase) SetCheckable(checkable bool) {
	bb.SetFlagState(checkable, int(ButtonFlagCheckable))
}

// IsChecked checks if button is checked
func (bb *ButtonBase) IsChecked() bool {
	return bb.HasFlag(int(ButtonFlagChecked))
}

// SetChecked sets the checked state of this button -- does not emit signal or
// update
func (bb *ButtonBase) SetChecked(chk bool) {
	bb.SetFlagState(chk, int(ButtonFlagChecked))
}

// ToggleChecked toggles the checked state of this button -- does not emit
// signal or update
func (bb *ButtonBase) ToggleChecked() {
	bb.SetChecked(!bb.IsChecked())
}

// SetAsMenu ensures that this functions as a menu even before menu items are added
func (bb *ButtonBase) SetAsMenu() {
	bb.SetFlag(int(ButtonFlagMenu))
}

// SetAsButton clears the explicit ButtonFlagMenu -- if there are menu items
// or a menu function then it will still behave as a menu
func (bb *ButtonBase) SetAsButton() {
	bb.ClearFlag(int(ButtonFlagMenu))
}

// SetText sets the text and updates the button
func (bb *ButtonBase) SetText(txt string) {
	if bb.This() == nil {
		return
	}
	updt := bb.UpdateStart()
	bb.StyMu.RLock()
	needSty := bb.Style.Font.Size.Val == 0
	bb.StyMu.RUnlock()
	if needSty {
		bb.StyleButton()
	}
	if bb.Text != txt {
		bb.SetFullReRender() // needed for resize
		bb.Text = txt
	}
	bb.This().(ButtonWidget).ConfigParts()
	bb.UpdateEnd(updt)
}

// SetIcon sets the Icon to given icon name (could be empty or 'none') and
// updates the button
func (bb *ButtonBase) SetIcon(iconName icons.Icon) {
	updt := bb.UpdateStart()
	defer bb.UpdateEnd(updt)
	if !bb.IsVisible() {
		bb.Icon = iconName
		return
	}
	bb.StyMu.RLock()
	needSty := bb.Style.Font.Size.Val == 0
	bb.StyMu.RUnlock()
	if needSty {
		bb.StyleButton()
	}
	if bb.Icon != iconName {
		bb.SetFullReRender()
	}
	bb.Icon = iconName
	bb.This().(ButtonWidget).ConfigParts()
}

// SetButtonState sets the button state -- returns true if state changed
func (bb *ButtonBase) SetButtonState(state ButtonStates) bool {
	bb.ButStateMu.Lock()
	defer bb.ButStateMu.Unlock()
	prev := bb.State
	if bb.IsDisabled() {
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
	bb.StyMu.Lock()
	bb.Style = bb.StateStyles[state]
	bb.StyMu.Unlock()
	if prev != bb.State {
		bb.SetFullReRenderIconLabel() // needs full rerender to update text, icon
		return true
	}
	return false
}

// UpdateButtonStyle sets the button style based on current state info --
// returns true if changed -- restyles parts if so
func (bb *ButtonBase) UpdateButtonStyle() bool {
	bb.ButStateMu.Lock()
	defer bb.ButStateMu.Unlock()
	prev := bb.State
	if bb.IsDisabled() {
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
	bb.Style = bb.StateStyles[bb.State]
	bb.This().(ButtonWidget).ConfigPartsIfNeeded()
	bb.StyMu.Lock()
	bb.Style2DWidget()
	bb.StyMu.Unlock()
	if prev != bb.State {
		bb.SetFullReRenderIconLabel() // needs full rerender
		return true
	}
	// fmt.Printf("but style updt: %v to %v\n", bb.Path(), bb.State)
	return false
}

// OnClicked calls the given function when the button is clicked,
// which is the default / standard way of activating the button
func (bb *ButtonBase) OnClicked(fun func()) {
	bb.ButtonSig.Connect(bb.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(ButtonClicked) {
			fun()
		}
	})
}

// ButtonPress sets the button in the down state -- mouse clicked down but
// not yet up -- emits ButtonPressed signal AND WidgetSig Selected signal --
// ButtonClicked is down and up
func (bb *ButtonBase) ButtonPress() {
	updt := bb.UpdateStart()
	if bb.IsDisabled() {
		if !strings.HasSuffix(bb.Class, "-action") { // not for menu-action, bar-action
			bb.SetSelectedState(!bb.IsSelected())
			bb.EmitSelectedSignal()
			bb.UpdateSig()
		}
	} else {
		bb.SetButtonState(ButtonDown)
		bb.ButtonSig.Emit(bb.This(), int64(ButtonPressed), nil)
	}
	bb.UpdateEnd(updt)
}

// BaseButtonRelease action: the button has just been released -- sends a released
// signal and returns state to normal, and emits clicked signal if if it was
// previously in pressed state
func (bb *ButtonBase) BaseButtonRelease() {
	if bb.IsDisabled() {
		return
	}
	wasPressed := (bb.State == ButtonDown)
	updt := bb.UpdateStart()

	bb.SetButtonState(ButtonHover)
	bb.ButtonSig.Emit(bb.This(), int64(ButtonReleased), nil)
	if wasPressed {
		bb.ButtonSig.Emit(bb.This(), int64(ButtonClicked), nil)
		bb.OpenMenu()

		if bb.IsCheckable() {
			bb.ToggleChecked()
			bb.ButtonSig.Emit(bb.This(), int64(ButtonToggled), nil)
		}
	}
	bb.UpdateEnd(updt)
}

// IsMenu returns true this button is on a menu -- it is a menu item
func (bb *ButtonBase) IsMenu() bool {
	return bb.HasFlag(int(ButtonFlagMenu))
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
		bb.MakeMenuFunc(bb.This(), &bb.Menu)
	}
	bb.BBoxMu.RLock()
	pos := bb.WinBBox.Max
	if pos.X == 0 && pos.Y == 0 { // offscreen
		pos = bb.ObjBBox.Max
	}
	indic := bb.Parts.ChildByName("indicator", 3)
	if indic != nil {
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
	bb.BBoxMu.RUnlock()
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

// ConfigPartsAddIndicator adds a menu indicator if the Indicator field is set to an icon;
// if defOn is true, an indicator is added even if the Indicator field is unset
// (as long as it is not explicitly set to [icons.None]);
// returns the index in Parts of the indicator object, which is named "indicator";
// an "ind-stretch" is added as well to put on the right by default.
func (bb *ButtonBase) ConfigPartsAddIndicator(config *kit.TypeAndNameList, defOn bool) int {
	needInd := !bb.Indicator.IsNil() || (defOn && bb.Indicator != icons.None)
	if !needInd {
		return -1
	}
	indIdx := -1
	config.Add(TypeStretch, "ind-stretch")
	indIdx = len(*config)
	config.Add(TypeIcon, "indicator")
	return indIdx
}

func (bb *ButtonBase) ConfigPartsIndicator(indIdx int) {
	if indIdx < 0 {
		return
	}
	ic := bb.Parts.Child(indIdx).(*Icon)
	icnm := bb.Indicator
	if icnm.IsNil() {
		icnm = icons.KeyboardArrowDown
	}
	if set, _ := ic.SetIcon(icnm); set {
		bb.StylePart(bb.Parts.Child(indIdx - 1).(Node2D)) // also get the stretch
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
	bb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		bw := recv.(ButtonWidget)
		bbb := bw.AsButtonBase()
		bbb.Node2DOnMouseEvent(me)
		if me.Button == mouse.Left {
			switch me.Action {
			case mouse.DoubleClick: // we just count as a regular click
				fallthrough
			case mouse.Press:
				me.SetProcessed()
				bbb.ButtonPress()
			case mouse.Release:
				me.SetProcessed()
				bw.ButtonRelease()
			}
		}
	})
}

// MouseFocusEvents handles button MouseFocusEvent
func (bb *ButtonBase) MouseFocusEvent() {
	bb.ConnectEvent(oswin.MouseFocusEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		bw := recv.(ButtonWidget)
		bbb := bw.AsButtonBase()
		if bbb.IsDisabled() {
			return
		}
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
		bbb.Node2DOnMouseFocusEvent(me)
		if me.Action == mouse.Enter {
			if EventTrace {
				fmt.Printf("bb focus enter: %v\n", bbb.Name())
			}
			bbb.ButtonEnterHover()
		} else {
			if EventTrace {
				fmt.Printf("bb focus exit: %v\n", bbb.Name())
			}
			bbb.ButtonExitHover()
		}
	})
}

// KeyChordEvent handles button KeyChord events
func (bb *ButtonBase) KeyChordEvent() {
	bb.ConnectEvent(oswin.KeyChordEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		bw := recv.(ButtonWidget)
		bbb := bw.AsButtonBase()
		if bbb.IsDisabled() {
			return
		}
		kt := d.(*key.ChordEvent)
		if KeyEventTrace {
			fmt.Printf("Button KeyChordEvent: %v\n", bbb.Path())
		}
		kf := KeyFun(kt.Chord())
		if kf == KeyFunEnter || kt.Rune == ' ' {
			if !(kt.Rune == ' ' && bbb.Viewport.IsCompleter()) {
				kt.SetProcessed()
				bbb.ButtonPress()
				bw.ButtonRelease()
			}
		}
	})
}

func (bb *ButtonBase) HoverTooltipEvent() {
	bb.ConnectEvent(oswin.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		wbb := recv.Embed(TypeButtonBase).(*ButtonBase)
		tt := wbb.Tooltip
		if wbb.Shortcut != "" {
			tt = "[ " + wbb.Shortcut.Shortcut() + " ]: " + tt
		}
		if tt != "" {
			me.SetProcessed()
			bb.BBoxMu.RLock()
			pos := wbb.WinBBox.Max
			bb.BBoxMu.RUnlock()
			pos.X -= 20
			PopupTooltip(tt, pos.X, pos.Y, wbb.Viewport, wbb.Nm)
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
	// AsButtonBase gets the button base for most basic functions -- reduces
	// interface size.
	AsButtonBase() *ButtonBase

	// ButtonRelease is called for release of button -- this is where buttons
	// actually differ in functionality.
	ButtonRelease()

	// StyleParts is called during Style2D to handle styling associated with
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

func (bb *ButtonBase) AsButtonBase() *ButtonBase {
	return bb
}

func (bb *ButtonBase) Init2D() {
	bb.Init2DWidget()
	// bb.State = ButtonActive
	bb.This().(ButtonWidget).ConfigParts()
}

func (bb *ButtonBase) ButtonRelease() {
	bb.BaseButtonRelease() // do base
}

func (bb *ButtonBase) StyleParts() {
	if pv, ok := bb.PropInherit("indicator", ki.NoInherit, ki.TypeProps); ok {
		pvs := kit.ToString(pv)
		bb.Indicator = icons.Icon(pvs)
	}
	if pv, ok := bb.PropInherit("icon", ki.NoInherit, ki.TypeProps); ok {
		pvs := kit.ToString(pv)
		bb.Icon = icons.Icon(pvs)
	}
}

func (bb *ButtonBase) ConfigParts() {
	bb.Parts.Lay = LayoutHoriz
	config := kit.TypeAndNameList{}
	icIdx, lbIdx := bb.ConfigPartsIconLabel(&config, bb.Icon, bb.Text)
	indIdx := bb.ConfigPartsAddIndicator(&config, false) // default off
	mods, updt := bb.Parts.ConfigChildren(config)
	bb.ConfigPartsSetIconLabel(bb.Icon, bb.Text, icIdx, lbIdx)
	bb.ConfigPartsIndicator(indIdx)
	if mods {
		bb.UpdateEnd(updt)
	}
}

func (bb *ButtonBase) ConfigPartsIfNeeded() {
	if !bb.PartsNeedUpdateIconLabel(bb.Icon, bb.Text) {
		return
	}
	bb.This().(ButtonWidget).ConfigParts()
}

// StyleButton does button styling -- it sets the StyMu Lock
func (bb *ButtonBase) StyleButton() {
	bb.StyMu.Lock()
	defer bb.StyMu.Unlock()

	hasTempl, saveTempl := bb.Style.FromTemplate()
	if !hasTempl || saveTempl {
		bb.Style2DWidget()
	}
	bb.This().(ButtonWidget).StyleParts()
	if nf, err := bb.PropTry("no-focus"); err == nil {
		bb.SetFlagState(!bb.IsDisabled() && !nf.(bool), int(CanFocus))
	} else {
		bb.SetFlagState(!bb.IsDisabled(), int(CanFocus))
	}
	parSty := bb.ParentActiveStyle()
	clsty := "." + bb.Class
	var clsp ki.Props
	if clspi, ok := bb.PropInherit(clsty, ki.NoInherit, ki.TypeProps); ok {
		clsp, ok = clspi.(ki.Props)
	}
	if hasTempl && saveTempl {
		bb.Style.SaveTemplate()
	}
	if hasTempl && !saveTempl {
		for i := 0; i < int(ButtonStatesN); i++ {
			bb.StateStyles[i].Template = bb.Style.Template + ButtonSelectors[i]
			bb.StateStyles[i].FromTemplate()
		}
	} else {
		for i := 0; i < int(ButtonStatesN); i++ {
			bb.StateStyles[i].CopyFrom(&bb.Style)
			bb.StateStyles[i].SetStyleProps(parSty, bb.StyleProps(ButtonSelectors[i]), bb.Viewport)
			if clsp != nil {
				if stclsp, ok := ki.SubProps(clsp, ButtonSelectors[i]); ok {
					bb.StateStyles[i].SetStyleProps(parSty, stclsp, bb.Viewport)
				}
			}
			bb.StateStyles[i].CopyUnitContext(&bb.Style.UnContext)
		}
	}
	if hasTempl && saveTempl {
		for i := 0; i < int(ButtonStatesN); i++ {
			bb.StateStyles[i].Template = bb.Style.Template + ButtonSelectors[i]
			bb.StateStyles[i].SaveTemplate()
		}
	}
	bb.ParentStyleRUnlock()
}

func (bb *ButtonBase) Style2D() {
	bb.StyleButton()

	bb.StyMu.Lock()
	bb.LayState.SetFromStyle(&bb.Style) // also does reset
	bb.StyMu.Unlock()
	bb.This().(ButtonWidget).ConfigParts()
	if bb.Menu != nil {
		bb.Menu.SetShortcuts(bb.ParentWindow())
	}
}

func (bb *ButtonBase) Layout2D(parBBox image.Rectangle, iter int) bool {
	bb.This().(ButtonWidget).ConfigPartsIfNeeded()
	bb.Layout2DBase(parBBox, true, iter) // init style
	bb.Layout2DParts(parBBox, iter)
	bb.StyMu.Lock()
	for i := 0; i < int(ButtonStatesN); i++ {
		bb.StateStyles[i].CopyUnitContext(&bb.Style.UnContext)
	}
	bb.StyMu.Unlock()
	return bb.Layout2DChildren(iter)
}

func (bb *ButtonBase) RenderButton() {
	rs, _, st := bb.RenderLock()
	bb.RenderStdBox(st)
	bb.RenderUnlock(rs)
}

func (bb *ButtonBase) Render2D() {
	if bb.FullReRenderIfNeeded() {
		return
	}
	if bb.PushBounds() {
		bb.This().(Node2D).ConnectEvents2D()
		bb.UpdateButtonStyle()
		bb.RenderButton()
		bb.Render2DParts()
		bb.Render2DChildren()
		bb.PopBounds()
	} else {
		bb.DisconnectAllEvents(RegPri)
	}
}

func (bb *ButtonBase) ConnectEvents2D() {
	bb.WidgetEvents()
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

func (bb *ButtonBase) Destroy() {
	if bb.Menu != nil {
		bb.Menu.DeleteShortcuts(bb.ParentWindow())
	}
}

///////////////////////////////////////////////////////////
// Button

// Button is a standard command button -- PushButton in Qt Widgets, and Button
// in Qt Quick -- by default it puts the icon to the left and the text to the
// right
type Button struct {
	ButtonBase

	// the type of button (default, primary, secondary, etc)
	Type ButtonTypes `desc:"the type of button (default, primary, secondary, etc)"`
}

var TypeButton = kit.Types.AddType(&Button{}, ButtonProps)

var ButtonProps = ki.Props{
	ki.EnumTypeFlag: TypeButtonFlags,
}

// ButtonTypes is an enum containing the
// different possible types of buttons
type ButtonTypes int

const (
	// ButtonFilled is a filled button with a
	// contrasting background color. It should be
	// used for prominent actions, typically those
	// that are the final in a sequence. It is equivalent
	// to Material Design's filled button.
	ButtonFilled ButtonTypes = iota
	// ButtonTonal is a filled button, similar
	// to [ButtonFilled]. It is used for the same purposes,
	// but it has a lighter background color and less emphasis.
	// It is equivalent to Material Design's filled tonal button.
	ButtonTonal
	// ButtonElevated is an elevated button with
	// a light background color and a shadow.
	// It is equivalent to Material Design's elevated button.
	ButtonElevated
	// ButtonOutlined is an outlined button that is
	// used for secondary actions that are still important.
	// It is equivalent to Material Design's outlined button.
	ButtonOutlined
	// ButtonText is a low-importance button with only
	// text and/or an icon and no border, background color,
	// or shadow. They should only be used for low emphasis
	// actions, and you must ensure they stand out from the
	// surrounding context sufficiently. It is equivalent
	// to Material Design's text and icon buttons.
	ButtonText

	ButtonTypesN
)

var TypeButtonTypes = kit.Enums.AddEnumAltLower(ButtonTypesN, kit.NotBitFlag, gist.StylePropProps, "Button")

// AddNewButton adds a new button to given parent node, with given name.
func AddNewButton(parent ki.Ki, name string) *Button {
	return parent.AddNewChild(TypeButton, name).(*Button)
}

func (bt *Button) OnInit() {
	bt.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Cursor = cursor.HandPointing
		s.Border.Radius = gist.BorderRadiusFull
		s.Margin = gist.BoxShadowMargin(BoxShadow1).ToValues()
		s.Padding.Set(units.Em(0.625*Prefs.DensityMul()), units.Em(1.5*Prefs.DensityMul()))
		if !bt.Icon.IsNil() {
			s.Padding.Left.SetEm(1 * Prefs.DensityMul())
		}
		if bt.Text == "" {
			s.Padding.Right.SetEm(1 * Prefs.DensityMul())
		}
		s.Text.Align = gist.AlignCenter
		switch bt.Type {
		case ButtonFilled:
			s.BackgroundColor.SetSolid(ColorScheme.Primary)
			s.Color = ColorScheme.OnPrimary
		case ButtonTonal:
			s.BackgroundColor.SetSolid(ColorScheme.SecondaryContainer)
			s.Color = ColorScheme.OnSecondaryContainer
		case ButtonElevated:
			s.Margin = gist.BoxShadowMargin(BoxShadow2).ToValues()
			s.BackgroundColor.SetSolid(ColorScheme.SurfaceContainerLow)
			s.Color = ColorScheme.Primary
			s.BoxShadow = BoxShadow1
		case ButtonOutlined:
			s.BackgroundColor.SetSolid(ColorScheme.Surface)
			s.Color = ColorScheme.Primary
			s.Border.Style.Set(gist.BorderSolid)
			s.Border.Color.Set(ColorScheme.Outline)
			s.Border.Width.Set(units.Px(1))
		case ButtonText:
			s.BackgroundColor = bt.ParentBackgroundColor()
			s.Color = ColorScheme.Primary
		}
		if bt.IsHovered() {
			if bt.Type == ButtonElevated {
				s.BoxShadow = BoxShadow2
			} else {
				s.BoxShadow = BoxShadow1
			}
		}
		// STYTODO: add state styles for buttons
	})
}

func (bt *Button) OnChildAdded(child ki.Ki) {
	if w := KiAsWidget(child); w != nil {
		switch w.Name() {
		case "icon":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetEm(1.125)
				s.Height.SetEm(1.125)
				s.Margin.Set()
				s.Padding.Set()
			})
		case "space":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetEm(0.5)
				s.MinWidth.SetEm(0.5)
			})
		case "label":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				// need to override so label's default color
				// doesn't take control on state changes
				s.Color = bt.Style.Color
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = gist.AlignMiddle
			})
		case "ind-stretch":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetEm(0.5)
			})
		case "indicator":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetEm(1.125)
				s.Height.SetEm(1.125)
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = gist.AlignBottom
			})
		}
	}
}

func (bt *Button) CopyFieldsFrom(frm any) {
	fr := frm.(*Button)
	bt.ButtonBase.CopyFieldsFrom(&fr.ButtonBase)
}

///////////////////////////////////////////////////////////
// CheckBox

// CheckBox toggles between a checked and unchecked state
type CheckBox struct {
	ButtonBase

	// [view: show-name] icon to use for the off, unchecked state of the icon -- plain Icon holds the On state -- can be set with icon-off property
	IconOff icons.Icon `xml:"icon-off" view:"show-name" desc:"icon to use for the off, unchecked state of the icon -- plain Icon holds the On state -- can be set with icon-off property"`
}

var TypeCheckBox = kit.Types.AddType(&CheckBox{}, CheckBoxProps)

// AddNewCheckBox adds a new button to given parent node, with given name.
func AddNewCheckBox(parent ki.Ki, name string) *CheckBox {
	return parent.AddNewChild(TypeCheckBox, name).(*CheckBox)
}

func (cb *CheckBox) OnInit() {
	cb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Cursor = cursor.HandPointing
		s.Text.Align = gist.AlignLeft
		s.Color = ColorScheme.OnBackground
		s.BackgroundColor = cb.ParentBackgroundColor()
		s.Margin.Set(units.Px(1 * Prefs.DensityMul()))
		s.Padding.Set(units.Px(1 * Prefs.DensityMul()))
		s.Border.Style.Set(gist.BorderNone)
		// switch cb.State {
		// case ButtonActive:
		// 	s.BackgroundColor.SetSolid(ColorScheme.Background)
		// case ButtonInactive:
		// 	s.BackgroundColor.SetSolid(ColorScheme.Background)
		// 	s.Color.SetColor(ColorScheme.OnBackground.Highlight(30))
		// case ButtonFocus, ButtonSelected:
		// 	s.BackgroundColor.SetSolid(ColorScheme.Background.Highlight(10))
		// case ButtonHover:
		// 	s.BackgroundColor.SetSolid(ColorScheme.Background.Highlight(15))
		// case ButtonDown:
		// 	s.BackgroundColor.SetSolid(ColorScheme.Background.Highlight(20))
		// }
	})
}

func (cb *CheckBox) OnChildAdded(child ki.Ki) {
	if w := KiAsWidget(child); w != nil {
		switch w.Name() {
		case "icon0", "icon1":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetEm(1.5)
				s.Height.SetEm(1.5)
				s.Margin.Set()
				s.Padding.Set()
				s.BackgroundColor.SetSolid(colors.Transparent)
			})
		case "space":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetCh(0.1)
			})
		case "label":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = gist.AlignMiddle
			})
		}
	}
}

func (cb *CheckBox) CopyFieldsFrom(frm any) {
	fr := frm.(*CheckBox)
	cb.ButtonBase.CopyFieldsFrom(&fr.ButtonBase)
	cb.IconOff = fr.IconOff
}

var CheckBoxProps = ki.Props{
	ki.EnumTypeFlag: TypeButtonFlags,
}

// CheckBoxWidget interface

func (cb *CheckBox) AsButtonBase() *ButtonBase {
	return &(cb.ButtonBase)
}

// OnClicked calls the given function when the button is clicked,
// which is the default / standard way of activating the button
func (cb *CheckBox) OnClicked(fun func()) {
	cb.ButtonSig.Connect(cb.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(ButtonToggled) {
			fun()
		}
	})
}

func (cb *CheckBox) ButtonRelease() {
	cb.BaseButtonRelease()
}

// SetIcons sets the Icons (by name) for the On (checked) and Off (unchecked)
// states, and updates button
func (cb *CheckBox) SetIcons(icOn, icOff icons.Icon) {
	updt := cb.UpdateStart()
	cb.Icon = icOn
	cb.IconOff = icOff
	cb.This().(ButtonWidget).ConfigParts()
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
	cb.This().(ButtonWidget).ConfigParts()
}

func (cb *CheckBox) StyleParts() {
	cb.ButtonBase.StyleParts()
	if pv, ok := cb.PropInherit("icon-off", false, true); ok { // no inh, yes type
		pvs := kit.ToString(pv)
		cb.IconOff = icons.Icon(pvs)
	}
}

func (cb *CheckBox) ConfigParts() {
	cb.SetCheckable(true)
	if !TheIconMgr.IsValid(cb.Icon) {
		cb.Icon = icons.CheckBox // fallback
	}
	if !TheIconMgr.IsValid(cb.IconOff) {
		cb.IconOff = icons.CheckBoxOutlineBlank
	}
	config := kit.TypeAndNameList{}
	icIdx := 0 // always there
	lbIdx := -1
	config.Add(TypeLayout, "stack")
	if cb.Text != "" {
		config.Add(TypeSpace, "space")
		lbIdx = len(config)
		config.Add(TypeLabel, "label")
	}
	mods, updt := cb.Parts.ConfigChildren(config)
	ist := cb.Parts.Child(icIdx).(*Layout)
	if mods || gist.RebuildDefaultStyles {
		ist.Lay = LayoutStacked
		ist.SetNChildren(2, TypeIcon, "icon") // covered by above config update
		icon := ist.Child(0).(*Icon)
		if set, _ := icon.SetIcon(cb.Icon); set {
			cb.StylePart(Node2D(icon))
		}
		icoff := ist.Child(1).(*Icon)
		if set, _ := icoff.SetIcon(cb.IconOff); set {
			cb.StylePart(Node2D(icoff))
		}
	}
	if cb.IsChecked() {
		ist.StackTop = 0
	} else {
		ist.StackTop = 1
	}
	if lbIdx >= 0 {
		lbl := cb.Parts.Child(lbIdx).(*Label)
		if lbl.Text != cb.Text {
			cb.StylePart(cb.Parts.Child(lbIdx - 1).(Node2D)) // also get the space
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
		cb.This().(ButtonWidget).ConfigParts()
	}
	icIdx := 0 // always there
	ist := cb.Parts.Child(icIdx).(*Layout)
	if TheIconMgr.IsValid(cb.Icon) {
		icon := ist.Child(0).(*Icon)
		if !icon.HasChildren() || icon.Nm != string(cb.Icon) || cb.NeedsFullReRender() {
			if set, _ := icon.SetIcon(cb.Icon); set {
				cb.StylePart(Node2D(icon))
			}
		}
	}
	if TheIconMgr.IsValid(cb.IconOff) {
		icoff := ist.Child(1).(*Icon)
		if !icoff.HasChildren() || icoff.Nm != string(cb.IconOff) || cb.NeedsFullReRender() {
			if set, _ := icoff.SetIcon(cb.IconOff); set {
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
