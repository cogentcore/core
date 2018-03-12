// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki"
	"image"
	// "reflect"
)

// Widget base type -- a widget handles event management and layout
type WidgetBase struct {
	Node2DBase
	Layout LayoutData `desc:"all the layout information for this item"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_WidgetBase = ki.KiTypes.AddType(&WidgetBase{})

func (w *WidgetBase) Defaults() {
	w.Layout.Defaults()
}

// Widget interface -- handles layout infrastructure etc
type Widget interface {
	// does the layout for widgets -- resulting position and size are in Layout.AllocPos, .AllocSize -- terminal non-layout widgets can compute preferred sizes based on content at this stage
}

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

// AbstractButton common button functionality -- properties: checkable, checked, autoRepeat, autoRepeatInterval, autoRepeatDelay
type AbstractButton struct {
	WidgetBase
	Text     string
	Shortcut string
	State    ButtonStates
	// todo: icon -- should be an svg
	ButtonSig ki.Signal `json:"-",desc:"signal for button -- see ButtonSignalType for the types"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_AbstractButton = ki.KiTypes.AddType(&AbstractButton{})

// PushButton is a standard command button
type PushButton struct {
	AbstractButton
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
	// width := 60
	// height := 50
	// g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig ki.SignalType, d interface{}) {
	// 	fmt.Printf("button %v pressed!\n", recv.PathUnique())
	// 	ab, ok := recv.(*AbstractButton)
	// 	if !ok {
	// 		return
	// 	}
	// 	g.UpdateStart()
	// 	ab.ButtonSig.Emit(recv.ThisKi(), ki.SendCustomSignal(int64(ButtonPressed)), d)
	// 	g.UpdateEnd()
	// })
}

func (g *PushButton) PaintProps2D() {
}

func (g *PushButton) Layout2D() {
	g.SetWinBBox(g.Node2DBBox())
}

func (g *PushButton) Node2DBBox() image.Rectangle {
	// todo:
	return image.Rectangle{}
}

// todo: need color brigher / darker functions

func (g *PushButton) Render2D() {
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
	// rad, got := g.PropNumber("border-radius")
	// if !got {
	// 	rad := 4.0
	// }
	// if rad == 0 {
	// 	vp.DrawRectangle(g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
	// } else {
	// 	vp.DrawRoundedRectangle(g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y, rad)
	// }
	pc.FillStrokeClear(rs)
}

func (g *PushButton) CanReRender2D() bool {
	return true
}

// check for interface implementation
var _ Node2D = &PushButton{}
