// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows
// +build !3d

package windriver

import (
	"image"
	"syscall"
	"time"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
)

var lastMouseClickTime time.Time
var lastMousePos image.Point

func sendMouseEvent(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	where := image.Point{int(_GET_X_LPARAM(lParam)), int(_GET_Y_LPARAM(lParam))}
	from := lastMousePos
	mods := keyModifiers()

	button := mouse.NoButton
	switch uMsg {
	case _WM_MOUSEMOVE:
		if wParam&_MK_LBUTTON != 0 {
			button = mouse.Left
		} else if wParam&_MK_MBUTTON != 0 {
			button = mouse.Middle
		} else if wParam&_MK_RBUTTON != 0 {
			button = mouse.Right
		}
	case _WM_LBUTTONDOWN, _WM_LBUTTONUP:
		button = mouse.Left
	case _WM_MBUTTONDOWN, _WM_MBUTTONUP:
		button = mouse.Middle
	case _WM_RBUTTONDOWN, _WM_RBUTTONUP:
		button = mouse.Right
	}

	var event oswin.Event
	switch uMsg {
	case _WM_MOUSEMOVE:
		if button == mouse.NoButton {
			event = &mouse.MoveEvent{
				Event: mouse.Event{
					Where:     where,
					Button:    button,
					Action:    mouse.Move,
					Modifiers: mods,
				},
				From: from,
			}
		} else {
			event = &mouse.DragEvent{
				MoveEvent: mouse.MoveEvent{
					Event: mouse.Event{
						Where:     where,
						Button:    button,
						Action:    mouse.Drag,
						Modifiers: mods,
					},
					From: from,
				},
			}
		}
	case _WM_LBUTTONDOWN, _WM_MBUTTONDOWN, _WM_RBUTTONDOWN:
		act := mouse.Press
		interval := time.Now().Sub(lastMouseClickTime)
		// fmt.Printf("interval: %v\n", interval)
		if (interval / time.Millisecond) < time.Duration(mouse.DoubleClickMSec) {
			act = mouse.DoubleClick
		}
		event = &mouse.Event{
			Where:     where,
			Button:    button,
			Action:    act,
			Modifiers: mods,
		}
		event.SetTime()
		lastMouseClickTime = event.Time()
	case _WM_LBUTTONUP, _WM_MBUTTONUP, _WM_RBUTTONUP:
		event = &mouse.Event{
			Where:     where,
			Button:    button,
			Action:    mouse.Release,
			Modifiers: mods,
		}
	case _WM_MOUSEWHEEL:
		// TODO: handle horizontal scrolling
		delta := _GET_WHEEL_DELTA_WPARAM(wParam)
		// fmt.Printf("delta %v\n", delta)
		// Convert from screen to window coordinates.
		p := _POINT{
			int32(where.X),
			int32(where.Y),
		}
		_ScreenToClient(hwnd, &p)
		where.X = int(p.X)
		where.Y = int(p.Y)

		event = &mouse.ScrollEvent{
			Event: mouse.Event{
				Where:     where,
				Button:    button,
				Action:    mouse.Scroll,
				Modifiers: mods,
			},
			Delta: image.Point{0, int(-delta)}, // only vert
		}
	default:
		panic("sendMouseEvent() called on non-mouse message")
	}

	event.Init()
	lastMousePos = event.Pos()
	sendEvent(hwnd, event)

	return 0
}
