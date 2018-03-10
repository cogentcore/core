// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/rcoreilly/goki/ki"
	"image"
	"reflect"
)

// Widget base type -- a widget handles event management and layout, and is a viewport so that it can be separately re-rendered relative to the rest of the scene -- the position and size are in the ViewBox
type Widget struct {
	Viewport2D
	Sizing SizePrefs2D `desc:"prefered sizing of this widget -- used by layouts to allocate size to widgets"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtWidget = ki.KiTypes.AddType(&Widget{})

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

//go:generate stringer -type=EventType

// AbstractButton common button functionality -- properties: checkable, checked, autoRepeat, autoRepeatInterval, autoRepeatDelay
type AbstractButton struct {
	Widget
	Text     string
	Shortcut string
	// todo: icon -- should be an svg
	ButtonSig ki.Signal `json:"-",desc:"signal for button -- see ButtonSignalType for the types"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtAbstractButton = ki.KiTypes.AddType(&AbstractButton{})

// PushButton is a standard command button
type PushButton struct {
	AbstractButton
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtPushButton = ki.KiTypes.AddType(&PushButton{})

func (g *PushButton) InitNode2D(vp *Viewport2D) bool {
	g.ViewBox.Size = image.Point{20, 10}
	g.ReceiveEventType(MouseUpEvent, func(recv, send ki.Ki, sig ki.SignalType, d interface{}) {
		fmt.Printf("button %v pressed!\n", recv.PathUnique())
		ab, ok := recv.(*AbstractButton)
		if !ok {
			return
		}
		g.UpdateStart()
		ab.ButtonSig.Emit(recv.ThisKi(), ki.SendCustomSignal(int64(ButtonPressed)), d)
		g.UpdateEnd(false)
	})
	if len(g.Children) == 0 {
		rect1 := g.AddNewChildNamed(reflect.TypeOf(GiRect{}), "rect1").(*GiRect)
		rect1.SetProp("fill", "#008800")
		rect1.SetProp("stroke", "#0000FF")
		rect1.SetProp("stroke-width", 5.0)
		rect1.Size = Size2D{20, 10}
		// important: don't add until AFTER adding sub-node
	}
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *PushButton) Node2D() *GiNode2D {
	return &g.GiNode2D
}

func (g *PushButton) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return g.Viewport2D.Node2DBBox(vp)
}

func (g *PushButton) Render2D(vp *Viewport2D) bool {
	return g.Viewport2D.Render2D(vp)
}
