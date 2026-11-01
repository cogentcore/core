// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
	"image"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
)

var (
	// ScrollWheelSpeed controls how fast the scroll wheel moves (typically
	// interpreted as pixels per wheel step).
	// This is also in core.DeviceSettings and updated from there
	ScrollWheelSpeed = float32(1)
)

// Buttons is a mouse button.
type Buttons int32 //enums:enum

// TODO: have a separate axis concept for wheel up/down? How does that relate
// to joystick events?

const (
	NoButton Buttons = iota
	Left
	Middle
	Right
)

// Mouse is a basic mouse event for all mouse events except Scroll
type Mouse struct {
	Base

	// TODO: have a field to hold what other buttons are down, for detecting
	// drags or button-chords.

	// TODO: add a Device ID, for multiple input devices?
}

func NewMouse(typ Types, but Buttons, where image.Point, mods key.Modifiers) *Mouse {
	ev := &Mouse{}
	ev.Typ = typ
	ev.SetUnique()
	ev.Button = but
	ev.Where = where
	ev.Mods = mods
	return ev
}

func (ev *Mouse) String() string {
	return fmt.Sprintf("%v{Button: %v, Pos: %v, Mods: %v, Time: %v}", ev.Type(), ev.Button, ev.Where, ev.Mods.ModifiersString(), ev.Time().Format("04:05"))
}

func (ev Mouse) HasPos() bool {
	return true
}

func NewMouseMove(but Buttons, where, prev image.Point, mods key.Modifiers) *Mouse {
	ev := &Mouse{}
	ev.Typ = MouseMove
	// not unique
	ev.Button = but
	ev.Where = where
	ev.Prev = prev
	ev.Mods = mods
	return ev
}

func NewMouseDrag(but Buttons, where, prev, start image.Point, mods key.Modifiers) *Mouse {
	ev := &Mouse{}
	ev.Typ = MouseDrag
	// not unique
	ev.Button = but
	ev.Where = where
	ev.Prev = prev
	ev.Start = start
	ev.Mods = mods
	return ev
}

// MouseScroll is for mouse scrolling, recording the delta of the scroll
type MouseScroll struct {
	Mouse

	// Delta is the amount of scrolling in each axis, which is always in pixel/dot
	// units (see [Scroll]).
	Delta math32.Vector2
}

func (ev *MouseScroll) String() string {
	return fmt.Sprintf("%v{Delta: %v, Pos: %v, Mods: %v, Time: %v}", ev.Type(), ev.Delta, ev.Where, ev.Mods.ModifiersString(), ev.Time().Format("04:05"))
}

func NewScroll(where image.Point, delta math32.Vector2, mods key.Modifiers) *MouseScroll {
	ev := &MouseScroll{}
	ev.Typ = Scroll
	// not unique, but delta integrated!
	ev.Where = where
	ev.Delta = delta
	ev.Mods = mods
	return ev
}
