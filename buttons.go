// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
	"image"
	// "math"
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
	Text        string               `xml:"text" desc:"label for the button"`
	Shortcut    string               `xml:"shortcut" desc:"keyboard shortcut -- todo: need to figure out ctrl, alt etc"`
	StateStyles [ButtonStatesN]Style `desc:"styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State       ButtonStates         `json:"-" xml:"-" desc:"current state of the button based on gui interaction"`
	ButtonSig   ki.Signal            `json:"-" xml:"-" desc:"signal for button -- see ButtonSignals for the types"`
	// todo: icon -- should be an xml
}

// must register all new types so type names can be looked up by name -- e.g., for json
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

///////////////////////////////////////////////////////////

// Button is a standard command button -- PushButton in Qt Widgets, and Button in Qt Quick
type Button struct {
	ButtonBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Button = kit.Types.AddType(&Button{}, nil)

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
	g.Init2DBase()
	g.ReceiveEventType(MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button) // note: will fail for any derived classes..
		if ok {
			ab.ButtonPressed()
		}
	})
	g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if ok {
			ab.ButtonReleased()
		}
	})
	g.ReceiveEventType(MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if ok {
			ab.ButtonEnterHover()
		}
	})
	g.ReceiveEventType(MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if ok {
			ab.ButtonExitHover()
		}
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if ok {
			kt, ok := d.(KeyTypedEvent)
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

var ButtonProps = []map[string]interface{}{
	{
		"border-width":        "1px",
		"border-radius":       "4px",
		"border-color":        "black",
		"border-style":        "solid",
		"padding":             "8px",
		"margin":              "4px",
		"box-shadow.h-offset": "4px",
		"box-shadow.v-offset": "4px",
		"box-shadow.blur":     "4px",
		"box-shadow.color":    "#CCC",
		// "font-family":         "Arial", // this is crashing
		"font-size":        "24pt",
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

func (g *Button) Style2D() {
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.Style.SetStyle(nil, &StyleDefault, ButtonProps[ButtonNormal])
	g.Style2DWidget()
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, ButtonProps[i])
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *Button) Size2D() {
	g.InitLayout2D()
	g.Size2DFromText(g.Text)
}

func (g *Button) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *Button) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *Button) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

// todo: need color brigher / darker functions

func (g *Button) Render2D() {
	if g.PushBounds() {
		if g.IsLeaf() {
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
func (g *Button) Render2DDefaultStyle() {
	st := &g.Style
	g.RenderStdBox(st)
	g.Render2DText(g.Text)
	// fmt.Printf("button %v text-style: %v\n", g.Name, st.Text)
}

func (g *Button) CanReRender2D() bool {
	return true
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
