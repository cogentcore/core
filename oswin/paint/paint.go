// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/mobile/event:
//
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package paint defines an event for the app being ready to paint.
package paint

import (
	"image"

	"github.com/rcoreilly/goki/gi/oswin"
)

// Event indicates that the app is ready to paint the next frame of the GUI.
//
//A frame is completed by calling the App's Publish method.
type Event struct {
	oswin.EventBase

	// External is true for paint events sent by the screen driver.
	//
	// An external event may be sent at any time in response to an
	// operating system event, for example the window opened, was
	// resized, or the screen memory was lost.
	//
	// Programs actively drawing to the screen as fast as vsync allows
	// should ignore external paint events to avoid a backlog of paint
	// events building up.
	External bool
}

/////////////////////////////
// oswin.Event interface

func (ev Event) Type() oswin.EventType {
	return oswin.PaintEvent
}

func (ev Event) HasPos() bool {
	return false
}

func (ev Event) Pos() image.Point {
	return image.ZP
}

func (ev Event) OnFocus() bool {
	return false
}

// check for interface implementation
var _ oswin.Event = &Event{}
