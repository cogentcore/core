// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/rcoreilly/goki/ki"
	"image"
	// "reflect"
)

// Widget base type
type WidgetBase struct {
	Node2DBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_WidgetBase = ki.KiTypes.AddType(&WidgetBase{})

// Styling notes:
// simple elemental widgets (buttons etc) have a DefaultRender method that renders based on
// Style, with full css styling support -- code has built-in initial defaults for a default
// style based on fusion style parameters on QML Qt Quick Controls

// Alternatively they support custom svg code for rendering each state as appropriate in a Stack
// more complex widgets such as a TreeView automatically render and don't support custom svg

// WidgetBase supports full Box rendering model, so Button just calls these methods to render
// -- base function needs to take a Style arg.

////////////////////////////////////////////////////////////////////////////////////////
// Buttons

// signals that buttons can send -- offset by SignalTypeBaseN when sent
type ButtonSignalType int64

const (
	// main signal -- button pressed down and up
	ButtonClicked ButtonSignalType = iota
	// button pushed down but not yet up
	ButtonPressed
	ButtonReleased
	ButtonToggled
	ButtonSignalTypeN
)

//go:generate stringer -type=ButtonSignalType

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
	// button is currently being pressed
	ButtonPress
	// total number of button states
	ButtonStatesN
)

//go:generate stringer -type=ButtonStates

// ButtonBase has common button functionality -- properties: checkable, checked, autoRepeat, autoRepeatInterval, autoRepeatDelay
type ButtonBase struct {
	WidgetBase
	Radius   float64              `svg:"border-radius",desc:"radius for rounded buttons"`
	Text     string               `svg:"text",desc:"label for the button"`
	Shortcut string               `svg:"shortcut",desc:"keyboard shortcut -- todo: need to figure out ctrl, alt etc"`
	Styles   [ButtonStatesN]Style `desc:"styles for the button, one for each state -- everything inherits from the first one which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State    ButtonStates
	// todo: icon -- should be an svg
	ButtonSig ki.Signal `json:"-",desc:"signal for button -- see ButtonSignalType for the types"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_ButtonBase = ki.KiTypes.AddType(&ButtonBase{})

func (g *ButtonBase) PaintProps2DBase() {
	g.Radius = g.PropNumberDefault("border-radius", 4.0)
}

///////////////////////////////////////////////////////////

// Button is a standard command button -- PushButton in Qt Widgets, and Button in Qt Quick
type Button struct {
	ButtonBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Button = ki.KiTypes.AddType(&Button{})

func (g *Button) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Button) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Button) InitNode2D() {
	g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		fmt.Printf("button %v pressed!\n", recv.PathUnique())
		ab, ok := recv.(*ButtonBase)
		if !ok {
			return
		}
		g.UpdateStart()
		ab.ButtonSig.Emit(recv.ThisKi(), ki.SendCustomSignal(int64(ButtonPressed)), d)
		g.UpdateEnd()
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		// todo: convert d to event, get key, check for shortcut, etc
		fmt.Printf("key pressed on %v!\n", recv.PathUnique())
		ab, ok := recv.(*ButtonBase)
		if !ok {
			return
		}
		g.UpdateStart()
		ab.ButtonSig.Emit(recv.ThisKi(), ki.SendCustomSignal(int64(ButtonPressed)), d)
		g.UpdateEnd()
	})
}

func (g *Button) DefaultStyle() {
	// set all our default style info, before parsing user-set ones
}

func (g *Button) PaintProps2D() {
	// todo: get all styling info -- due to diff between widgets and SVG, we need to call this explicitly here and cannot rely on base-case
}

func (g *Button) Layout2D(iter int) {
	if iter == 0 {
		pc := &g.MyPaint
		var w, h float64
		w, h = pc.MeasureString(g.Text)
		if g.Size.X > 0 {
			w = ki.Max64(g.Size.X, w)
		}
		if g.Size.Y > 0 {
			h = ki.Max64(g.Size.Y, h)
		}
		g.Layout.AllocSize = Size2D{w, h}
		g.SetWinBBox(g.Node2DBBox())
	}
}

func (g *Button) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

// todo: need color brigher / darker functions

func (g *Button) Render2D() {
	g.DefaultGeom()
	if g.IsLeaf() {
		g.Render2DDefaultStyle()
	} else {
		// todo: manage stacked layout to select appropriate image based on state
		return
	}
}

// render using a default style if not otherwise styled
func (g *Button) Render2DDefaultStyle() {
	pc := &g.MyPaint
	rs := &g.Viewport.Render
	if g.Radius == 0.0 {
		pc.DrawRectangle(rs, g.Layout.AllocPos.X, g.Layout.AllocPos.Y, g.Layout.AllocSize.X, g.Layout.AllocSize.Y)
	} else {
		pc.DrawRoundedRectangle(rs, g.Layout.AllocPos.X, g.Layout.AllocPos.Y, g.Layout.AllocSize.X, g.Layout.AllocSize.Y, g.Radius)
	}
	pc.FillStrokeClear(rs)
	pc.DrawStringAnchored(rs, g.Text, g.Layout.AllocPos.X, g.Layout.AllocPos.Y, 0.0, 0.9)
}

func (g *Button) CanReRender2D() bool {
	return true
}

// check for interface implementation
var _ Node2D = &Button{}
