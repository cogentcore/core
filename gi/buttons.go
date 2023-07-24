// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"
	"sync"

	"github.com/goki/gi/gist"
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
	Text         string                    `xml:"text" desc:"label for the button -- if blank then no label is presented"`
	Icon         icons.Icon                `xml:"icon" view:"show-name" desc:"optional icon for the button -- different buttons can configure this in different ways relative to the text if both are present"`
	Indicator    icons.Icon                `xml:"indicator" view:"show-name" desc:"name of the menu indicator icon to present, or blank or 'nil' or 'none' -- shown automatically when there are Menu elements present unless 'none' is set"`
	Shortcut     key.Chord                 `xml:"shortcut" desc:"optional shortcut keyboard chord to trigger this action -- always window-wide in scope, and should generally not conflict other shortcuts (a log message will be emitted if so).  Shortcuts are processed after all other processing of keyboard input.  Use Command for Control / Meta (Mac Command key) per platform.  These are only set automatically for Menu items, NOT for items in ToolBar or buttons somewhere, but the tooltip for buttons will show the shortcut if set."`
	StateStyles  [ButtonStatesN]gist.Style `copy:"-" json:"-" xml:"-" desc:"styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State        ButtonStates              `copy:"-" json:"-" xml:"-" desc:"current state of the button based on gui interaction"`
	ButtonSig    ki.Signal                 `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for button -- see ButtonSignals for the types"`
	Menu         Menu                      `desc:"the menu items for this menu -- typically add Action elements for menus, along with separators"`
	MakeMenuFunc MakeMenuFunc              `copy:"-" json:"-" xml:"-" view:"-" desc:"set this to make a menu on demand -- if set then this button acts like a menu button"`
	ButStateMu   sync.Mutex                `copy:"-" json:"-" xml:"-" view:"-" desc:"button state mutex"`
}

var KiT_ButtonBase = kit.Types.AddType(&ButtonBase{}, ButtonBaseProps)

var ButtonBaseProps = ki.Props{
	"base-type":     true, // excludes type from user selections
	"EnumType:Flag": KiT_ButtonFlags,
}

// ButtonTypes is an enum containing the different
// possible types of buttons
type ButtonTypes int

const (
	// ButtonDefault is a default gray button
	// typically used in menus and checkboxes
	ButtonDefault ButtonTypes = iota
	// ButtonPrimary is a primary button colored
	// as the primary color; it is typically used for
	// primary actions like save, send, and new
	ButtonPrimary
	// ButtonSecondary is a secondary button colored as
	// the inverse of a primary button; it is typically used
	// for secondary actions like cancel, back, and options
	ButtonSecondary

	ButtonTypesN
)

var KiT_ButtonTypes = kit.Enums.AddEnumAltLower(ButtonTypesN, kit.NotBitFlag, gist.StylePropProps, "Button")

//go:generate stringer -type=ButtonTypes

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

//go:generate stringer -type=ButtonFlags

var KiT_ButtonFlags = kit.Enums.AddEnumExt(KiT_NodeFlags, ButtonFlagsN, kit.BitFlag, nil)

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

//go:generate stringer -type=ButtonSignals

// https://ux.stackexchange.com/questions/84872/what-is-the-buttons-unpressed-and-unhovered-state-called

// mutually-exclusive button states -- determines appearance
type ButtonStates int32

//go:generate stringer -type=ButtonStates

var KiT_ButtonStates = kit.Enums.AddEnumAltLower(ButtonStatesN, kit.NotBitFlag, gist.StylePropProps, "Button")

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
	if bb.IsInactive() {
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
	if bb.IsInactive() {
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
		oswin.TheApp.Cursor(bb.ParentWindow().OSWin).Push(cursor.HandPointing)
		bb.UpdateEnd(updt)
	}
}

// ButtonExitHover called when button exiting hover
func (bb *ButtonBase) ButtonExitHover() {
	if bb.State == ButtonHover {
		updt := bb.UpdateStart()
		bb.SetButtonState(ButtonActive)
		oswin.TheApp.Cursor(bb.ParentWindow().OSWin).Pop()
		bb.UpdateEnd(updt)
	}
}

// MouseEvents handles button MouseEvent
func (bb *ButtonBase) MouseEvent() {
	bb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		bw := recv.(ButtonWidget)
		bbb := bw.AsButtonBase()
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
		if bbb.IsInactive() {
			return
		}
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
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
		if bbb.IsInactive() {
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
		wbb := recv.Embed(KiT_ButtonBase).(*ButtonBase)
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
	bb.State = ButtonActive
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
		bb.SetFlagState(!bb.IsInactive() && !nf.(bool), int(CanFocus))
	} else {
		bb.SetFlagState(!bb.IsInactive(), int(CanFocus))
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
	bb.LayState.SetFromStyle(&bb.Style.Layout) // also does reset
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
	Type ButtonTypes `desc:"the type of button (default, primary, secondary, etc)"`
}

var KiT_Button = kit.Types.AddType(&Button{}, nil)

// AddNewButton adds a new button to given parent node, with given name.
func AddNewButton(parent ki.Ki, name string) *Button {
	return parent.AddNewChild(KiT_Button, name).(*Button)
}

func (bt *Button) CopyFieldsFrom(frm any) {
	fr := frm.(*Button)
	bt.ButtonBase.CopyFieldsFrom(&fr.ButtonBase)
}

// // DefaultStyle implements the [DefaultStyler] interface
// func (bt *Button) DefaultStyle() {
// 	s := &bt.Style

// 	s.Border.Style.Set(gist.BorderNone)
// 	s.Border.Width.Set()
// 	s.Border.Color.Set()
// 	s.Border.Radius.Set(units.Px(4))
// 	s.Layout.Padding.Set(units.Px(4))
// 	s.Layout.Margin.Set(units.Px(2))
// 	s.Layout.MinWidth.SetEm(1)
// 	s.Layout.MinHeight.SetEm(1)
// 	s.Text.Align = gist.AlignCenter

// 	switch bt.Type {
// 	case ButtonDefault:
// 		c := TheColorScheme.Secondary
// 		switch bt.State {
// 		case ButtonHover:
// 			c = c.Highlight(10)
// 		case ButtonDown:
// 			c = c.Highlight(20)
// 		}
// 		s.Font.BgColor.SetColor(c)
// 		s.Font.Color.SetColor(c.ContrastColor())
// 	case ButtonPrimary:
// 		c := TheColorScheme.Primary
// 		switch bt.State {
// 		case ButtonHover:
// 			c = c.Highlight(20)
// 		case ButtonDown:
// 			c = c.Highlight(30)
// 		}
// 		s.Font.BgColor.SetColor(c)
// 		s.Font.Color.SetColor(c.ContrastColor())
// 	case ButtonSecondary:
// 		c := TheColorScheme.Background
// 		switch bt.State {
// 		case ButtonHover:
// 			c = c.Highlight(20)
// 		case ButtonDown:
// 			c = c.Highlight(30)
// 		}
// 		s.Font.BgColor.SetColor(c)
// 		s.Font.Color.SetColor(TheColorScheme.Primary)
// 		s.Border.Style.Set(gist.BorderSolid)
// 		s.Border.Width.Set(units.Px(1))
// 		s.Border.Color.Set(TheColorScheme.Primary)
// 	}
// }

var ButtonProps = ki.Props{
	"EnumType:Flag":    KiT_ButtonFlags,
	"border-width":     units.Px(1),
	"border-radius":    units.Px(4),
	"border-color":     &Prefs.Colors.Border,
	"padding":          units.Px(4),
	"margin":           units.Px(2),
	"min-width":        units.Em(1),
	"min-height":       units.Em(1),
	"text-align":       gist.AlignCenter,
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	"#space": ki.Props{
		"width":     units.Ch(.5),
		"min-width": units.Ch(.5),
	},
	"#icon": ki.Props{
		"width":   units.Em(1),
		"height":  units.Em(1),
		"margin":  units.Px(0),
		"padding": units.Px(0),
		"fill":    &Prefs.Colors.Icon,
		"stroke":  &Prefs.Colors.Font,
	},
	"#label": ki.Props{
		"margin":  units.Px(0),
		"padding": units.Px(0),
		// "font-size": units.NewPt(24),
	},
	"#indicator": ki.Props{
		"width":          units.Ex(1.5),
		"height":         units.Ex(1.5),
		"margin":         units.Px(0),
		"padding":        units.Px(0),
		"vertical-align": gist.AlignBottom,
		"fill":           &Prefs.Colors.Icon,
		"stroke":         &Prefs.Colors.Font,
	},
	"#ind-stretch": ki.Props{
		"width": units.Em(1),
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
		"border-width":     units.Px(2),
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

///////////////////////////////////////////////////////////
// CheckBox

// CheckBox toggles between a checked and unchecked state
type CheckBox struct {
	ButtonBase
	IconOff icons.Icon `xml:"icon-off" view:"show-name" desc:"icon to use for the off, unchecked state of the icon -- plain Icon holds the On state -- can be set with icon-off property"`
}

var KiT_CheckBox = kit.Types.AddType(&CheckBox{}, nil)

// AddNewCheckBox adds a new button to given parent node, with given name.
func AddNewCheckBox(parent ki.Ki, name string) *CheckBox {
	return parent.AddNewChild(KiT_CheckBox, name).(*CheckBox)
}

func (cb *CheckBox) CopyFieldsFrom(frm any) {
	fr := frm.(*CheckBox)
	cb.ButtonBase.CopyFieldsFrom(&fr.ButtonBase)
	cb.IconOff = fr.IconOff
}

var CheckBoxProps = ki.Props{
	"EnumType:Flag":    KiT_ButtonFlags,
	"icon":             icons.CheckBox,
	"icon-off":         icons.CheckBoxOutlineBlank,
	"text-align":       gist.AlignLeft,
	"color":            &Prefs.Colors.Font,
	"background-color": &Prefs.Colors.Control,
	"margin":           units.Px(1),
	"padding":          units.Px(1),
	"border-width":     units.Px(0),
	"#icon0": ki.Props{
		"width":            units.Em(1),
		"height":           units.Em(1),
		"margin":           units.Px(0),
		"padding":          units.Px(0),
		"background-color": color.Transparent,
		"fill":             &Prefs.Colors.Control,
		"stroke":           &Prefs.Colors.Font,
	},
	"#icon1": ki.Props{
		"width":            units.Em(1),
		"height":           units.Em(1),
		"margin":           units.Px(0),
		"padding":          units.Px(0),
		"background-color": color.Transparent,
		"fill":             &Prefs.Colors.Control,
		"stroke":           &Prefs.Colors.Font,
	},
	"#space": ki.Props{
		"width": units.Ch(0.1),
	},
	"#label": ki.Props{
		"margin":  units.Px(0),
		"padding": units.Px(0),
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
		"border-width":     units.Px(2),
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
func (cb *CheckBox) SetIcons(icOn, icOff string) {
	updt := cb.UpdateStart()
	cb.Icon = icons.Icon(icOn)
	cb.IconOff = icons.Icon(icOff)
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
	cb.ConfigStyles()
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
	config.Add(KiT_Layout, "stack")
	if cb.Text != "" {
		config.Add(KiT_Space, "space")
		lbIdx = len(config)
		config.Add(KiT_Label, "label")
	}
	mods, updt := cb.Parts.ConfigChildren(config)
	ist := cb.Parts.Child(icIdx).(*Layout)
	if mods || gist.RebuildDefaultStyles {
		ist.Lay = LayoutStacked
		ist.SetNChildren(2, KiT_Icon, "icon") // covered by above config update
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

func (cb *CheckBox) ConfigStyles() {
	cb.AddStyleFunc(StyleFuncDefault, func() {
		cb.Style.Font.Color.SetColor(Colors.Font)
		switch cb.State {
		default:
			cb.Style.Font.BgColor.SetColor(Colors.Background)
		case ButtonHover:
			cb.Style.Font.BgColor.SetColor(Colors.Background.Highlight(10))
		case ButtonDown:
			cb.Style.Font.BgColor.SetColor(Colors.Background.Highlight(20))
		}
	})
}
