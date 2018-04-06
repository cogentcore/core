// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"

	"github.com/rcoreilly/goki/gi/oswin"
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
	// main signal -- button pressed down and up
	ButtonClicked ButtonSignals = iota
	// button pushed down but not yet up
	ButtonPressed
	ButtonReleased
	// toggled is for checked / unchecked state
	ButtonToggled
	ButtonSignalsN
)

//go:generate stringer -type=ButtonSignals

// https://ux.stackexchange.com/questions/84872/what-is-the-buttons-unpressed-and-unhovered-state-called

// mutually-exclusive button states -- determines appearance
type ButtonStates int32

const (
	// normal state -- there but not being interacted with
	ButtonNormal ButtonStates = iota
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

// ButtonBase has common button functionality -- properties: checkable, checked, autoRepeat, autoRepeatInterval, autoRepeatDelay
type ButtonBase struct {
	WidgetBase
	Text        string               `xml:"text" desc:"label for the button -- if blank then no label is presented"`
	Icon        *Icon                `desc:"optional icon for the button -- different button can configure this in different ways relative to the text if both are present"`
	Shortcut    string               `xml:"shortcut" desc:"keyboard shortcut -- todo: need to figure out ctrl, alt etc"`
	StateStyles [ButtonStatesN]Style `desc:"styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State       ButtonStates         `json:"-" xml:"-" desc:"current state of the button based on gui interaction"`
	ButtonSig   ki.Signal            `json:"-" xml:"-" desc:"signal for button -- see ButtonSignals for the types"`
	// todo: icon -- should be an xml
}

var KiT_ButtonBase = kit.Types.AddType(&ButtonBase{}, nil)

// is this button selected?
func (g *ButtonBase) IsSelected() bool {
	return bitflag.Has(g.NodeFlags, int(ButtonFlagSelected))
}

// is this button checkable
func (g *ButtonBase) IsCheckable() bool {
	return bitflag.Has(g.NodeFlags, int(ButtonFlagCheckable))
}

// is this button checked
func (g *ButtonBase) IsChecked() bool {
	return bitflag.Has(g.NodeFlags, int(ButtonFlagChecked))
}

// set the selected state of this button
func (g *ButtonBase) SetSelected(sel bool) {
	bitflag.SetState(&g.NodeFlags, sel, int(ButtonFlagSelected))
	g.SetButtonState(ButtonNormal) // update state
}

// set the checked state of this button
func (g *ButtonBase) SetChecked(chk bool) {
	bitflag.SetState(&g.NodeFlags, chk, int(ButtonFlagChecked))
}

// set the button state to target
func (g *ButtonBase) SetButtonState(state ButtonStates) {
	// todo: process disabled state -- probably just deal with the property directly?
	// it overrides any choice here and just sets state to disabled..
	if state == ButtonNormal && g.IsSelected() {
		state = ButtonSelected
	} else if state == ButtonNormal && g.HasFocus() {
		state = ButtonFocus
	}
	g.State = state
	g.Style = g.StateStyles[state] // get relevant styles
}

// set the button in the down state -- mouse clicked down but not yet up --
// emits ButtonPressed signal -- ButtonClicked is down and up
func (g *ButtonBase) ButtonPressed() {
	g.UpdateStart()
	g.SetButtonState(ButtonDown)
	g.ButtonSig.Emit(g.This, int64(ButtonPressed), nil)
	g.UpdateEnd()
}

// the button has just been released -- sends a released signal and returns
// state to normal, and emits clicked signal if if it was previously in pressed state
func (g *ButtonBase) ButtonReleased() {
	wasPressed := (g.State == ButtonDown)
	g.UpdateStart()
	g.SetButtonState(ButtonNormal)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
	}
	g.UpdateEnd()
}

// button starting hover-- todo: keep track of time and popup a tooltip -- signal?
func (g *ButtonBase) ButtonEnterHover() {
	if g.State != ButtonHover {
		g.UpdateStart()
		g.SetButtonState(ButtonHover)
		g.UpdateEnd()
	}
}

// button exiting hover
func (g *ButtonBase) ButtonExitHover() {
	if g.State == ButtonHover {
		g.UpdateStart()
		g.SetButtonState(ButtonNormal)
		g.UpdateEnd()
	}
}

// interface for button widgets -- can extend as needed
type ButtonWidget interface {
	// get the button base for most basic functions -- reduces interface size
	ButtonAsBase() *ButtonBase
	// called for release of button -- this is where buttons actually differ in functionality
	ButtonRelease()
	// configure the parts of the button
	ConfigParts()
}

// set the text and update button
func SetButtonText(bw ButtonWidget, txt string) {
	g := bw.ButtonAsBase()
	g.UpdateStart()
	g.Text = txt
	bw.ConfigParts()
	g.UpdateEnd()
}

// set the Icon (could be nil) and update button
func SetButtonIcon(bw ButtonWidget, ic *Icon) {
	g := bw.ButtonAsBase()
	g.UpdateStart()
	g.Icon = ic // this is jut the pointer
	bw.ConfigParts()
	g.UpdateEnd()
}

// handles all the basic button events
func Init2DButtonEvents(bw ButtonWidget) {
	g := bw.ButtonAsBase()
	g.ReceiveEventType(oswin.MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := (interface{})(recv).(ButtonWidget)
		ab.ButtonAsBase().ButtonPressed()
	})
	g.ReceiveEventType(oswin.MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := (interface{})(recv).(ButtonWidget)
		ab.ButtonRelease() // special one
	})
	g.ReceiveEventType(oswin.MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := (interface{})(recv).(ButtonWidget)
		ab.ButtonAsBase().ButtonEnterHover()
	})
	g.ReceiveEventType(oswin.MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := (interface{})(recv).(ButtonWidget)
		ab.ButtonAsBase().ButtonExitHover()
	})
	g.ReceiveEventType(oswin.KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := (interface{})(recv).(ButtonWidget)
		bb := ab.ButtonAsBase()
		kt := d.(*oswin.KeyTypedEvent)
		// todo: register shortcuts with window, and generalize these keybindings
		kf := KeyFun(kt.Key, kt.Chord)
		if kf == KeyFunSelectItem || kt.Key == "space" {
			bb.ButtonPressed()
			// todo: brief delay??
			ab.ButtonRelease() // special one
			kt.SetProcessed()
		}
	})
}

///////////////////////////////////////////////////////////

// Button is a standard command button -- PushButton in Qt Widgets, and Button
// in Qt Quick -- by default it puts the icon to the left and the text to the
// right
type Button struct {
	ButtonBase
}

var KiT_Button = kit.Types.AddType(&Button{}, nil)

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

func (g *Button) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Button) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Button) AsLayout2D() *Layout {
	return nil
}

func (g *Button) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
	Init2DButtonEvents(g)
}

var ButtonProps = []map[string]interface{}{
	{
		"border-width":        units.NewValue(1, units.Px),
		"border-radius":       units.NewValue(4, units.Px),
		"border-color":        color.Black,
		"border-style":        BorderSolid,
		"padding":             units.NewValue(4, units.Px),
		"margin":              units.NewValue(4, units.Px),
		"box-shadow.h-offset": units.NewValue(4, units.Px),
		"box-shadow.v-offset": units.NewValue(4, units.Px),
		"box-shadow.blur":     units.NewValue(4, units.Px),
		"box-shadow.color":    "#CCC",
		"text-align":          AlignCenter,
		"vertical-align":      AlignTop,
		"color":               color.Black,
		"background-color":    "#EEF",
		"#icon": map[string]interface{}{
			"width":   units.NewValue(1, units.Em),
			"height":  units.NewValue(1, units.Em),
			"margin":  units.NewValue(0, units.Px),
			"padding": units.NewValue(0, units.Px),
		},
		"#label": map[string]interface{}{
			"margin":           units.NewValue(0, units.Px),
			"padding":          units.NewValue(0, units.Px),
			"background-color": "none",
		},
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

func (g *Button) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, g.Text)
	g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(g.Icon, g.Text, icIdx, lbIdx, ButtonProps[ButtonNormal])
}

// todo: add PartsNeedUpdate to check if text, icon are diff, and call update in render.
//

func (g *Button) Style2D() {
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.Style2DWidget(ButtonProps[ButtonNormal])
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, ButtonProps[i])
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
	g.ConfigParts()
}

func (g *Button) Size2D() {
	g.Size2DWidget()
}

func (g *Button) Layout2D(parBBox image.Rectangle) {
	g.ConfigParts()
	g.Layout2DWidget(parBBox) // lays out parts
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *Button) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *Button) ComputeBBox2D(parBBox image.Rectangle) {
	g.ComputeBBox2DWidget(parBBox)
}

func (g *Button) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

func (g *Button) Move2D(delta Vec2D, parBBox image.Rectangle) {
	g.Move2DWidget(delta, parBBox) // moves parts
	g.Move2DChildren(delta)
}

// todo: need color brigher / darker functions

func (g *Button) Render2D() {
	if g.PushBounds() {
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

func (g *Button) ReRender2D() (node Node2D, layout bool) {
	node = g.This.(Node2D)
	layout = false
	return
}

func (g *Button) FocusChanged2D(gotFocus bool) {
	g.UpdateStart()
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonNormal) // lose any hover state but whatever..
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &Button{}
