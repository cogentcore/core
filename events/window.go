// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
)

// WindowEvent reports on actions taken on a window.
// The system.Window Flags and other state information
// will always be updated prior to this event being sent,
// so those should be consulted directly for the new current state.
type WindowEvent struct {
	Base

	// Action taken on the window -- what has changed.
	// Window state fields have current values.
	Action WinActions
}

func NewWindow(act WinActions) *WindowEvent {
	ev := &WindowEvent{}
	ev.Action = act
	ev.Typ = Window
	ev.SetUnique()
	return ev
}

func NewWindowResize() *WindowEvent {
	ev := &WindowEvent{}
	ev.Typ = WindowResize
	// not unique
	return ev
}

func NewWindowPaint() *WindowEvent {
	ev := &WindowEvent{}
	ev.Typ = WindowPaint
	// not unique
	return ev
}

func (ev *WindowEvent) HasPos() bool {
	return false
}

func (ev *WindowEvent) String() string {
	return fmt.Sprintf("%v{Action: %v, Time: %v}", ev.Type(), ev.Action, ev.Time().Format("04:05"))
}

// WinActions is the action taken on the window by the user.
type WinActions int32 //enums:enum -trim-prefix Win

const (
	// NoWinAction is the zero value for special types (Resize, Paint)
	NoWinAction WinActions = iota

	// WinClose means that the window is about to close, but has not yet closed.
	WinClose

	// WinMinimize means that the window has been iconified / miniaturized / is no
	// longer visible.
	WinMinimize

	// WinMove means that the window was moved but NOT resized or changed in any
	// other way -- does not require a redraw, but anything tracking positions
	// will want to update.
	WinMove

	// WinFocus indicates that the window has been activated for receiving user
	// input.
	WinFocus

	// WinFocusLost indicates that the window is no longer activated for
	// receiving input.
	WinFocusLost

	// WinShow is for the WindowShow event -- sent by the system shortly
	// after the window has opened, to ensure that full rendering
	// is completed with the proper size, and to trigger one-time actions such as
	// configuring the main menu after the window has opened.
	WinShow

	// ScreenUpdate occurs when any of the screen information is updated
	// This event is sent to the first window on the list of active windows
	// and it should then perform any necessary updating
	ScreenUpdate
)
