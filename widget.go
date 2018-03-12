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
// simple elemental widgets (buttons etc) have a DefaultStyle render method that incorporates
// property hints but is fairly generic, and alternatively support a (stack of) custom svg
// code for rendering each state as appropriate
// more complex widgets such as a TreeView automatically render and don't support custom svg

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
	// disabled -- not pressable
	ButtonDisabled ButtonStates = iota
	// normal state -- there but not being interacted with
	ButtonNormal
	// hover over state
	ButtonHover
	// button is the focus -- will respond to keyboard input
	ButtonFocus
	// button is currently being pressed
	ButtonStatesN
)

//go:generate stringer -type=ButtonStates

// ButtonBase has common button functionality -- properties: checkable, checked, autoRepeat, autoRepeatInterval, autoRepeatDelay
type ButtonBase struct {
	WidgetBase
	Radius   float64 `svg:"border-radius",desc:"radius for rounded buttons"`
	Text     string  `svg:"text",desc:"label for the button"`
	Shortcut string  `svg:"shortcut",desc:"keyboard shortcut -- todo: need to figure out ctrl, alt etc"`
	State    ButtonStates
	// todo: icon -- should be an svg
	ButtonSig ki.Signal `json:"-",desc:"signal for button -- see ButtonSignalType for the types"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_ButtonBase = ki.KiTypes.AddType(&ButtonBase{})

func (g *ButtonBase) PaintProps2DBase() {
	if val, got := g.PropNumber("border-radius"); got {
		g.Radius = val
	}
	// todo: default fill etc
}

// PushButton is a standard command button
type PushButton struct {
	ButtonBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_PushButton = ki.KiTypes.AddType(&PushButton{})

// todo: support direct re-rendering option, where a background rect is painted then everything goes on top
// only makes sense for opaque renders but much more efficient than managing images all over the place
// parent vp can check this and directly re-render or not -- test for equality of bbox and xform!?

// todo: not so clear about how geom / xform relates to sizing of image in a sub-viewport..

func (g *PushButton) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *PushButton) GiViewport2D() *Viewport2D {
	return nil
}

func (g *PushButton) InitNode2D() {
	g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig ki.SignalType, d interface{}) {
		fmt.Printf("button %v pressed!\n", recv.PathUnique())
		ab, ok := recv.(*ButtonBase)
		if !ok {
			return
		}
		g.UpdateStart()
		ab.ButtonSig.Emit(recv.ThisKi(), ki.SendCustomSignal(int64(ButtonPressed)), d)
		g.UpdateEnd()
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig ki.SignalType, d interface{}) {
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

func (g *PushButton) PaintProps2D() {

}

func (g *PushButton) Layout2D(iter int) {
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

func (g *PushButton) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

// todo: need color brigher / darker functions

func (g *PushButton) Render2D() {
	g.GeomFromLayout()
	if g.IsLeaf() {
		g.Render2DDefaultStyle()
	} else {
		// todo: manage stacked layout to select appropriate image based on state
		return
	}
}

// render using a default style if not otherwise styled
func (g *PushButton) Render2DDefaultStyle() {
	pc := &g.MyPaint
	rs := &g.Viewport.Render
	if g.Radius == 0.0 {
		pc.DrawRectangle(rs, g.Layout.AllocPos.X, g.Layout.AllocPos.Y, g.Layout.AllocSize.X, g.Layout.AllocSize.Y)
	} else {
		pc.DrawRoundedRectangle(rs, g.Layout.AllocPos.X, g.Layout.AllocPos.Y, g.Layout.AllocSize.X, g.Layout.AllocSize.Y, g.Radius)
	}
	pc.FillStrokeClear(rs)
	pc.DrawString(rs, g.Text, g.Layout.AllocPos.X, g.Layout.AllocPos.Y, g.Layout.AllocSize.X)
}

func (g *PushButton) CanReRender2D() bool {
	return true
}

// check for interface implementation
var _ Node2D = &PushButton{}
