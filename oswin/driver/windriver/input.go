// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

import (
	"fmt"
	"image"
	"syscall"
	"time"
	"unicode/utf16"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/winkey"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
)

var lastMouseClickEvent oswin.Event
var lastMouseEvent oswin.Event

func sendMouseEvent(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	where := image.Point{int(_GET_X_LPARAM(lParam)), int(_GET_Y_LPARAM(lParam))}
	from := image.ZP
	if lastMouseEvent != nil {
		from = lastMouseEvent.Pos()
	}
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
		if lastMouseClickEvent != nil {
			interval := time.Now().Sub(lastMouseClickEvent.Time())
			// fmt.Printf("interval: %v\n", interval)
			if (interval / time.Millisecond) < time.Duration(mouse.DoubleClickMSec) {
				act = mouse.DoubleClick
			}
		}
		event = &mouse.Event{
			Where:     where,
			Button:    button,
			Action:    act,
			Modifiers: mods,
		}
		event.SetTime()
		lastMouseClickEvent = event
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
			Delta: image.Point{0, int(delta)}, // only vert
		}
	default:
		panic("sendMouseEvent() called on non-mouse message")
	}

	event.Init()
	lastMouseEvent = event
	SendEvent(hwnd, event)

	return 0
}

// Precondition: this is called in immediate response to the message that triggered the event (so not after w.Send).
func keyModifiers() int32 {
	var m key.Modifiers
	down := func(x int32) bool {
		// GetKeyState gets the key state at the time of the message, so this is what we want.
		return _GetKeyState(x)&0x80 != 0
	}

	if down(_VK_CONTROL) {
		m |= 1 << uint32(key.Control)
	}
	if down(_VK_MENU) {
		m |= 1 << uint32(key.Alt)
	}
	if down(_VK_SHIFT) {
		m |= 1 << uint32(key.Shift)
	}
	if down(_VK_LWIN) || down(_VK_RWIN) {
		m |= 1 << uint32(key.Meta)
	}
	return int32(m)
}

func readRune(vKey uint32, scanCode uint8) rune {
	var (
		keystate [256]byte
		buf      [4]uint16
	)
	if err := _GetKeyboardState(&keystate[0]); err != nil {
		panic(fmt.Sprintf("win32: %v", err))
	}
	// TODO: cache GetKeyboardLayout result, update on WM_INPUTLANGCHANGE
	layout := _GetKeyboardLayout(0)
	ret := _ToUnicodeEx(vKey, uint32(scanCode), &keystate[0], &buf[0], int32(len(buf)), 0, layout)
	if ret < 1 {
		return -1
	}
	return utf16.Decode(buf[:ret])[0]
}

func sendKeyEvent(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	r = readRune(uint32(wParam), uint8(lParam>>16))
	c, mod := winkey.DecodeKeyEvent(r, hwnd, uMsg, wParam, lParam)
	var act key.Actions
	switch uMsg {
	case _WM_KEYDOWN:
		const prevMask = 1 << 30
		if repeat := lParam&prevMask == prevMask; repeat {
			act = key.None
		} else {
			act = key.Press
		}
	case _WM_KEYUP:
		act = key.Release
	default:
		panic(fmt.Sprintf("windriver: unexpected key message: %d", uMsg))
	}

	event := &key.Event{
		Rune:      r,
		Code:      c,
		Modifiers: mod,
		Action:    act,
	}

	SendEvent(hwnd, event)

	// do ChordEvent -- only for non-modifier key presses -- call
	// key.ChordString to convert the event into a parsable string for GUI
	// events
	if act == key.Press && !key.CodeIsModifier(c) {
		che := &key.ChordEvent{Event: *event}
		SendEvent(hwnd, che)
	}
	return 0
}
