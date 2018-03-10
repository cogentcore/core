// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/skelterjohn/go.wde"
	// "fmt"
)

// EventType determines which type of event is being sent
type EventType int64

const (
	MouseMovedEvent EventType = iota
	MouseDownEvent
	MouseUpEvent
	MouseDraggedEvent
	MagnifyEvent
	RotateEvent
	ScrollEvent
	KeyDownEvent
	KeyUpEvent
	KeyTypedEvent
	MouseEnteredEvent // entered window
	MouseExitedEvent  // exited window
	ResizeEvent
	CloseEvent
	EventTypeN
)

//go:generate stringer -type=EventType

func EventTypeFromEvent(ei interface{}) EventType {
	switch ei.(type) {
	case wde.MouseMovedEvent:
		return MouseMovedEvent
	case wde.MouseDownEvent:
		return MouseDownEvent
	case wde.MouseUpEvent:
		return MouseUpEvent
	case wde.MouseDraggedEvent:
		return MouseDraggedEvent
	case wde.MouseEnteredEvent:
		return MouseEnteredEvent
	case wde.MouseExitedEvent:
		return MouseExitedEvent
	case wde.MagnifyEvent:
		return MagnifyEvent
	case wde.RotateEvent:
		return RotateEvent
	case wde.ScrollEvent:
		return ScrollEvent
	case wde.KeyDownEvent:
		return KeyDownEvent
	case wde.KeyUpEvent:
		return KeyUpEvent
	case wde.KeyTypedEvent:
		return KeyTypedEvent
	case wde.ResizeEvent:
		return ResizeEvent
	case wde.CloseEvent:
		return CloseEvent
	default:
		return EventTypeN // not recognized..
	}
}
