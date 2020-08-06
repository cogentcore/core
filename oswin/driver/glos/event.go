// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"image"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
)

var lastMouseClickTime time.Time
var lastMousePos image.Point
var lastMouseButton mouse.Buttons
var lastMouseAction mouse.Actions
var lastMods int32
var lastKey key.Codes

func glfwMods(mod glfw.ModifierKey) int32 {
	m := int32(0)
	if mod&glfw.ModShift != 0 {
		m |= 1 << uint32(key.Shift)
	}
	if mod&glfw.ModControl != 0 {
		m |= 1 << uint32(key.Control)
	}
	if mod&glfw.ModAlt != 0 {
		m |= 1 << uint32(key.Alt)
	}
	if mod&glfw.ModSuper != 0 {
		m |= 1 << uint32(key.Meta)
	}
	return m
}

// physical key
func (w *windowImpl) keyEvent(gw *glfw.Window, ky glfw.Key, scancode int, action glfw.Action, mod glfw.ModifierKey) {
	em := glfwMods(mod)
	lastMods = em
	ec := glfwKeyCode(ky)
	lastKey = ec
	rn, mapped := key.CodeRuneMap[ec]
	act := key.Press
	if action == glfw.Release {
		act = key.Release
	} else if action == glfw.Repeat {
		act = key.Press
	}

	fw := theApp.WindowInFocus()
	if w != fw {
		if fw == nil {
			// fmt.Printf("glos key event focus window is nil!  window: %v\n", w.Nm)
			fw = w
			// } else {
			// fmt.Printf("glos key event window: %v != focus window: %v\n", w.Nm, fw.Name())
		}
	}

	event := &key.Event{
		Code:      ec,
		Rune:      rn,
		Modifiers: em,
		Action:    act,
	}
	event.Init()
	fw.Send(event)
	glfw.PostEmptyEvent()

	if act == key.Press && ec < key.CodeLeftControl &&
		(key.HasAnyModifierBits(em, key.Control, key.Meta) || // don't include alt here
			!mapped || ec == key.CodeTab) {
		// if key.HasAllModifierBits(em, key.Control) && ec == key.CodeY {
		// 	fmt.Printf("Ctrl-Y win: %v\n", w.Nm)
		// }
		// fmt.Printf("chord ky	: %v ec	: %v   mapped: %v\n", ky, ec, mapped)
		che := &key.ChordEvent{
			Event: key.Event{
				Code:      ec,
				Rune:      rn,
				Modifiers: em,
				Action:    act,
			},
		}
		fw.Send(che)
		glfw.PostEmptyEvent()
	}
}

// char input
func (w *windowImpl) charEvent(gw *glfw.Window, char rune, mods glfw.ModifierKey) {
	em := glfwMods(mods)
	act := key.Press
	che := &key.ChordEvent{
		Event: key.Event{
			Rune:      char,
			Modifiers: em,
			Action:    act,
		},
	}
	// fmt.Printf("che: %v\n", che)
	fw := theApp.WindowInFocus()
	if w != fw {
		if fw == nil {
			// fmt.Printf("glos char event focus window is nil!  window: %v\n", w.Nm)
			fw = w
		} else {
			// fmt.Printf("glos char event window: %v != focus window: %v\n", w.Nm, fw.Name())
			w = fw.(*windowImpl)
		}
	}
	fw.Send(che)
	glfw.PostEmptyEvent()
}

func (w *windowImpl) mouseButtonEvent(gw *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
	mods := glfwMods(mod)
	lastMods = mods
	but := mouse.Left
	switch button {
	case glfw.MouseButtonMiddle:
		but = mouse.Middle
	case glfw.MouseButtonRight:
		but = mouse.Right

	}
	act := mouse.Press
	if action == glfw.Release {
		act = mouse.Release
	}
	if action == glfw.Press {
		interval := time.Now().Sub(lastMouseClickTime)
		// fmt.Printf("interval: %v\n", interval)
		if (interval / time.Millisecond) < time.Duration(mouse.DoubleClickMSec) {
			act = mouse.DoubleClick
		}
	}
	if mod&glfw.ModControl != 0 {
		but = mouse.Right
	}
	lastMouseButton = but
	lastMouseAction = act
	where := w.curMousePosPoint(gw)
	event := &mouse.Event{
		Where:     where,
		Button:    but,
		Action:    act,
		Modifiers: mods,
	}
	event.Init()
	if act == mouse.Press {
		lastMouseClickTime = event.Time()
	}
	w.Send(event)
	glfw.PostEmptyEvent()
}

func (w *windowImpl) scrollEvent(gw *glfw.Window, xoff, yoff float64) {
	mods := lastMods
	if theApp.Platform() == oswin.MacOS {
		xoff *= float64(mouse.ScrollWheelSpeed)
		yoff *= float64(mouse.ScrollWheelSpeed)
	} else { // others have lower multipliers in general
		xoff *= 4 * float64(mouse.ScrollWheelSpeed)
		yoff *= 4 * float64(mouse.ScrollWheelSpeed)
	}
	where := w.curMousePosPoint(gw)
	event := &mouse.ScrollEvent{
		Event: mouse.Event{
			Where:     where,
			Action:    mouse.Scroll,
			Modifiers: mods,
		},
		Delta: image.Point{int(-xoff), int(-yoff)},
	}
	event.Init()
	w.Send(event)
	glfw.PostEmptyEvent()
}

func (w *windowImpl) curMousePosPoint(gw *glfw.Window) image.Point {
	xp, yp := gw.GetCursorPos()
	return w.mousePosToPoint(xp, yp)
}

func (w *windowImpl) mousePosToPoint(x, y float64) image.Point {
	var where image.Point
	if theApp.Platform() == oswin.MacOS {
		where = image.Point{int(w.DevPixRatio * float32(x)), int(w.DevPixRatio * float32(y))}
	} else {
		where = image.Point{int(x), int(y)}
	}
	return where
}

func (w *windowImpl) cursorPosEvent(gw *glfw.Window, x, y float64) {
	if w.resettingPos {
		return
	}
	from := lastMousePos
	where := w.mousePosToPoint(x, y)
	if w.mouseDisabled {
		w.resettingPos = true
		if theApp.Platform() == oswin.MacOS {
			w.glw.SetCursorPos(float64(lastMousePos.X)/float64(w.DevPixRatio), float64(lastMousePos.Y)/float64(w.DevPixRatio))
		} else {
			w.glw.SetCursorPos(float64(lastMousePos.X), float64(lastMousePos.Y))
		}
		w.resettingPos = false
	} else {
		lastMousePos = where
	}
	if lastMouseAction == mouse.Press {
		event := &mouse.DragEvent{
			MoveEvent: mouse.MoveEvent{
				Event: mouse.Event{
					Where:     where,
					Button:    lastMouseButton,
					Action:    mouse.Drag,
					Modifiers: lastMods,
				},
				From: from,
			},
		}
		event.Init()
		w.Send(event)
		glfw.PostEmptyEvent()
	} else {
		event := &mouse.MoveEvent{
			Event: mouse.Event{
				Where:     where,
				Button:    mouse.NoButton,
				Action:    mouse.Move,
				Modifiers: lastMods,
			},
			From: from,
		}
		event.Init()
		w.Send(event)
		glfw.PostEmptyEvent()
	}
}

func (w *windowImpl) cursorEnterEvent(gw *glfw.Window, entered bool) {
}

func (w *windowImpl) dropEvent(gw *glfw.Window, names []string) {
	ln := len(names)
	if ln == 0 {
		return
	}
	md := mimedata.NewMimes(ln, ln)
	for i, s := range names {
		md[i] = mimedata.NewTextData(s)
	}
	where := w.curMousePosPoint(gw)
	event := &dnd.Event{
		Action:    dnd.External,
		Where:     where,
		Modifiers: lastMods,
		Data:      md,
	}
	event.DefaultMod()
	event.Init()
	w.Send(event)
}

func glfwKeyCode(kcode glfw.Key) key.Codes {
	switch kcode {
	case glfw.KeyA:
		return key.CodeA
	case glfw.KeyB:
		return key.CodeB
	case glfw.KeyC:
		return key.CodeC
	case glfw.KeyD:
		return key.CodeD
	case glfw.KeyE:
		return key.CodeE
	case glfw.KeyF:
		return key.CodeF
	case glfw.KeyG:
		return key.CodeG
	case glfw.KeyH:
		return key.CodeH
	case glfw.KeyI:
		return key.CodeI
	case glfw.KeyJ:
		return key.CodeJ
	case glfw.KeyK:
		return key.CodeK
	case glfw.KeyL:
		return key.CodeL
	case glfw.KeyM:
		return key.CodeM
	case glfw.KeyN:
		return key.CodeN
	case glfw.KeyO:
		return key.CodeO
	case glfw.KeyP:
		return key.CodeP
	case glfw.KeyQ:
		return key.CodeQ
	case glfw.KeyR:
		return key.CodeR
	case glfw.KeyS:
		return key.CodeS
	case glfw.KeyT:
		return key.CodeT
	case glfw.KeyU:
		return key.CodeU
	case glfw.KeyV:
		return key.CodeV
	case glfw.KeyW:
		return key.CodeW
	case glfw.KeyX:
		return key.CodeX
	case glfw.KeyY:
		return key.CodeY
	case glfw.KeyZ:
		return key.CodeZ
	case glfw.Key1:
		return key.Code1
	case glfw.Key2:
		return key.Code2
	case glfw.Key3:
		return key.Code3
	case glfw.Key4:
		return key.Code4
	case glfw.Key5:
		return key.Code5
	case glfw.Key6:
		return key.Code6
	case glfw.Key7:
		return key.Code7
	case glfw.Key8:
		return key.Code8
	case glfw.Key9:
		return key.Code9
	case glfw.Key0:
		return key.Code0
	case glfw.KeyEnter:
		return key.CodeReturnEnter
	case glfw.KeyEscape:
		return key.CodeEscape
	case glfw.KeyBackspace:
		return key.CodeDeleteBackspace
	case glfw.KeyTab:
		return key.CodeTab
	case glfw.KeySpace:
		return key.CodeSpacebar
	case glfw.KeyMinus:
		return key.CodeHyphenMinus
	case glfw.KeyEqual:
		return key.CodeEqualSign
	case glfw.KeyLeftBracket:
		return key.CodeLeftSquareBracket
	case glfw.KeyRightBracket:
		return key.CodeRightSquareBracket
	case glfw.KeyBackslash:
		return key.CodeBackslash
	case glfw.KeySemicolon:
		return key.CodeSemicolon
	case glfw.KeyApostrophe:
		return key.CodeApostrophe
	case glfw.KeyGraveAccent:
		return key.CodeGraveAccent
	case glfw.KeyComma:
		return key.CodeComma
	case glfw.KeyPeriod:
		return key.CodeFullStop
	case glfw.KeySlash:
		return key.CodeSlash
	case glfw.KeyCapsLock:
		return key.CodeCapsLock
	case glfw.KeyF1:
		return key.CodeF1
	case glfw.KeyF2:
		return key.CodeF2
	case glfw.KeyF3:
		return key.CodeF3
	case glfw.KeyF4:
		return key.CodeF4
	case glfw.KeyF5:
		return key.CodeF5
	case glfw.KeyF6:
		return key.CodeF6
	case glfw.KeyF7:
		return key.CodeF7
	case glfw.KeyF8:
		return key.CodeF8
	case glfw.KeyF9:
		return key.CodeF9
	case glfw.KeyF10:
		return key.CodeF10
	case glfw.KeyF11:
		return key.CodeF11
	case glfw.KeyF12:
		return key.CodeF12
	// 70: PrintScreen
	// 71: Scroll Lock
	// 72: Pause
	// 73: Insert
	case glfw.KeyHome:
		return key.CodeHome
	case glfw.KeyPageUp:
		return key.CodePageUp
	case glfw.KeyDelete:
		return key.CodeDeleteForward
	case glfw.KeyEnd:
		return key.CodeEnd
	case glfw.KeyPageDown:
		return key.CodePageDown
	case glfw.KeyRight:
		return key.CodeRightArrow
	case glfw.KeyLeft:
		return key.CodeLeftArrow
	case glfw.KeyDown:
		return key.CodeDownArrow
	case glfw.KeyUp:
		return key.CodeUpArrow
	case glfw.KeyNumLock:
		return key.CodeKeypadNumLock
	case glfw.KeyKPDivide:
		return key.CodeKeypadSlash
	case glfw.KeyKPMultiply:
		return key.CodeKeypadAsterisk
	case glfw.KeyKPSubtract:
		return key.CodeKeypadHyphenMinus
	case glfw.KeyKPAdd:
		return key.CodeKeypadPlusSign
	case glfw.KeyKPEnter:
		return key.CodeKeypadEnter
	case glfw.KeyKP1:
		return key.CodeKeypad1
	case glfw.KeyKP2:
		return key.CodeKeypad2
	case glfw.KeyKP3:
		return key.CodeKeypad3
	case glfw.KeyKP4:
		return key.CodeKeypad4
	case glfw.KeyKP5:
		return key.CodeKeypad5
	case glfw.KeyKP6:
		return key.CodeKeypad6
	case glfw.KeyKP7:
		return key.CodeKeypad7
	case glfw.KeyKP8:
		return key.CodeKeypad8
	case glfw.KeyKP9:
		return key.CodeKeypad9
	case glfw.KeyKP0:
		return key.CodeKeypad0
	case glfw.KeyKPDecimal:
		return key.CodeKeypadFullStop
	case glfw.KeyKPEqual:
		return key.CodeKeypadEqualSign
	case glfw.KeyF13:
		return key.CodeF13
	case glfw.KeyF14:
		return key.CodeF14
	case glfw.KeyF15:
		return key.CodeF15
	case glfw.KeyF16:
		return key.CodeF16
	case glfw.KeyF17:
		return key.CodeF17
	case glfw.KeyF18:
		return key.CodeF18
	case glfw.KeyF19:
		return key.CodeF19
	case glfw.KeyF20:
		return key.CodeF20
	// 116: Keyboard Execute
	// case glfw.KeyHelp:
	// 	return key.CodeHelp
	// 118: Keyboard Menu
	// 119: Keyboard Select
	// 120: Keyboard Stop
	// 121: Keyboard Again
	// 122: Keyboard Undo
	// 123: Keyboard Cut
	// 124: Keyboard Copy
	// 125: Keyboard Paste
	// 126: Keyboard Find
	// case glfw.KeyMute:
	// 	return key.CodeMute
	// case glfw.KeyVolumeUp:
	// 	return key.CodeVolumeUp
	// case glfw.KeyVolumeDown:
	// 	return key.CodeVolumeDown
	// 130: Keyboard Locking Caps Lock
	// 131: Keyboard Locking Num Lock
	// 132: Keyboard Locking Scroll Lock
	// 133: Keyboard Comma
	// 134: Keyboard Equal Sign
	case glfw.KeyLeftControl:
		return key.CodeLeftControl
	case glfw.KeyLeftShift:
		return key.CodeLeftShift
	case glfw.KeyLeftAlt:
		return key.CodeLeftAlt
	case glfw.KeyLeftSuper:
		return key.CodeLeftMeta
	case glfw.KeyRightControl:
		return key.CodeRightControl
	case glfw.KeyRightShift:
		return key.CodeRightShift
	case glfw.KeyRightAlt:
		return key.CodeRightAlt
	case glfw.KeyRightSuper:
		return key.CodeRightMeta
	case glfw.KeyLast:
		return lastKey
	default:
		return key.CodeUnknown
	}
}
