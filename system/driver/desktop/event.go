// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"image"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func GlfwMods(mod glfw.ModifierKey) key.Modifiers {
	var m key.Modifiers
	if mod&glfw.ModShift != 0 {
		m.SetFlag(true, key.Shift)
	}
	if mod&glfw.ModControl != 0 {
		m.SetFlag(true, key.Control)
	}
	if mod&glfw.ModAlt != 0 {
		m.SetFlag(true, key.Alt)
	}
	if mod&glfw.ModSuper != 0 {
		m.SetFlag(true, key.Meta)
	}
	return m
}

func (w *Window) FocusWindow() *Window {
	fw := TheApp.WindowInFocus()
	if w != fw {
		if fw == nil {
			fw = w
		}
	}
	return fw.(*Window)
}

// physical key
func (w *Window) KeyEvent(gw *glfw.Window, ky glfw.Key, scancode int, action glfw.Action, mod glfw.ModifierKey) {
	mods := GlfwMods(mod)
	ec := GlfwKeyCode(ky)
	rn := key.CodeRuneMap[ec]
	typ := events.KeyDown
	if action == glfw.Release {
		typ = events.KeyUp
	} else if action == glfw.Repeat {
		typ = events.KeyDown
	}
	fw := w.FocusWindow()
	fw.Event.Key(typ, rn, ec, mods)
	glfw.PostEmptyEvent() // todo: why??
}

// char input
func (w *Window) CharEvent(gw *glfw.Window, char rune, mod glfw.ModifierKey) {
	mods := GlfwMods(mod)
	fw := w.FocusWindow()
	fw.Event.KeyChord(char, key.CodeUnknown, mods)
	glfw.PostEmptyEvent() // todo: why?
}

func (w *Window) CurMousePosPoint(gw *glfw.Window) image.Point {
	xp, yp := gw.GetCursorPos()
	return w.MousePosToPoint(xp, yp)
}

func (w *Window) MousePosToPoint(x, y float64) image.Point {
	var where image.Point
	if TheApp.Platform() == system.MacOS {
		where = image.Pt(int(w.DevicePixelRatio*float32(x)), int(w.DevicePixelRatio*float32(y)))
	} else {
		where = image.Pt(int(x), int(y))
	}
	return where
}

func (w *Window) MouseButtonEvent(gw *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
	mods := GlfwMods(mod)
	but := events.Left
	switch button {
	case glfw.MouseButtonMiddle:
		but = events.Middle
	case glfw.MouseButtonRight:
		but = events.Right
	}
	typ := events.MouseDown
	if action == glfw.Release {
		typ = events.MouseUp
	}
	if mod&glfw.ModControl != 0 { // special case: control always = RMB  can undo this downstream..
		but = events.Right
	}
	where := w.CurMousePosPoint(gw)
	w.Event.MouseButton(typ, but, where, mods)
	glfw.PostEmptyEvent() // why?
}

func (w *Window) ScrollEvent(gw *glfw.Window, xoff, yoff float64) {
	delta := math32.Vec2(float32(xoff), float32(yoff)).MulScalar(-10)
	if TheApp.Platform() == system.MacOS {
		delta.SetMulScalar(w.DevicePixelRatio)
	} else {
		delta.SetMulScalar(4) // other platforms need a bigger multiplier in general
	}
	where := w.CurMousePosPoint(gw)
	w.Event.Scroll(where, delta)
	glfw.PostEmptyEvent()
}

func (w *Window) CursorPosEvent(gw *glfw.Window, x, y float64) {
	if w.Event.ResettingPos {
		return
	}
	where := w.MousePosToPoint(x, y)
	if !w.CursorEnabled {
		w.Event.ResettingPos = true
		if TheApp.Platform() == system.MacOS {
			w.Glw.SetCursorPos(float64(w.Event.Last.MousePos.X)/float64(w.DevicePixelRatio), float64(w.Event.Last.MousePos.Y)/float64(w.DevicePixelRatio))
		} else {
			w.Glw.SetCursorPos(float64(w.Event.Last.MousePos.X), float64(w.Event.Last.MousePos.Y))
		}
		w.Event.ResettingPos = false
	}
	w.Event.MouseMove(where)
	glfw.PostEmptyEvent()
}

func (w *Window) CursorEnterEvent(gw *glfw.Window, entered bool) {
}

func (w *Window) DropEvent(gw *glfw.Window, names []string) {
	ln := len(names)
	if ln == 0 {
		return
	}
	md := mimedata.NewMimes(ln, ln)
	for i, s := range names {
		md[i] = mimedata.NewTextData(s)
	}
	where := w.CurMousePosPoint(gw)
	w.Event.DropExternal(where, md)
}

// TODO: should this be a map?
func GlfwKeyCode(kcode glfw.Key) key.Codes {
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
		return key.CodeBackspace
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
		return key.CodeDelete
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
	default:
		return key.CodeUnknown
	}
}
