// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"fmt"
	"syscall/js"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/system"
)

// Input methods (IME) compose text in the hidden app-text-field input,
// which has the focus while a text widget is focused (see
// [App.ShowVirtualKeyboard]). Its composition events are sent as
// Composition events, and the committed text as KeyChord events, and it
// is moved to the caret of the text widget, so that the browser shows
// the candidate window of the input method there.

// textField returns the hidden app-text-field input.
func textField() js.Value {
	return js.Global().Get("document").Call("getElementById", "app-text-field")
}

// addCompositionListeners adds the listeners for input method compositions.
func (a *App) addCompositionListeners() {
	g := js.Global()
	AddEventListener(g, "compositionstart", appInput(a.onCompositionUpdate))
	AddEventListener(g, "compositionupdate", appInput(a.onCompositionUpdate))
	AddEventListener(g, "compositionend", appInput(a.onCompositionEnd))
}

func (a *App) onCompositionUpdate(this js.Value, args []js.Value) any {
	data := args[0].Get("data")
	txt := ""
	if data.Truthy() {
		txt = data.String()
	}
	n := len([]rune(txt))
	if txt == "" {
		a.composing = false
	} else {
		a.composing = true
	}
	a.Event.Composition(txt, n, n)
	return nil
}

func (a *App) onCompositionEnd(this js.Value, args []js.Value) any {
	if a.composing {
		a.composing = false
		a.Event.Composition("", 0, 0)
	}
	if data := args[0].Get("data"); data.Truthy() {
		for _, r := range data.String() {
			a.Event.KeyChord(r, key.CodeUnknown, 0)
		}
	}
	textField().Set("value", "")
	return nil
}

// isComposing returns whether the given keyboard or input
// event is part of an input method composition.
func isComposing(e js.Value) bool {
	if e.Get("isComposing").Truthy() {
		return true
	}
	if kc := e.Get("keyCode"); kc.Truthy() && kc.Int() == 229 { // Safari, older browsers
		return true
	}
	switch e.Get("inputType").String() {
	case "insertCompositionText", "insertFromComposition", "deleteCompositionText", "deleteByComposition":
		return true
	}
	return false
}

func (w *Window) SetTextInput(ti *system.TextInput) {
	tf := textField()
	st := tf.Get("style")
	if ti == nil {
		st.Set("left", "-10px")
		st.Set("top", "-10px")
		st.Set("height", "0")
		return
	}
	// window coordinates are canvas pixels
	dpr := w.App.Scrn.DevicePixelRatio
	r := ti.Caret
	st.Set("left", fmt.Sprintf("%gpx", float32(r.Min.X)/dpr))
	st.Set("top", fmt.Sprintf("%gpx", float32(r.Min.Y)/dpr))
	st.Set("width", "1px")
	st.Set("height", fmt.Sprintf("%gpx", float32(r.Dy())/dpr))
}
