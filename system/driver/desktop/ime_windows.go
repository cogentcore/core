// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package desktop

/*
#cgo LDFLAGS: -limm32 -lcomctl32
#include <windows.h>
void initIME(HWND hwnd);
void setTextInput(HWND hwnd, int active, int x, int y, int w, int h);
*/
import "C"

import (
	"unsafe"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/system"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// initIME sets up input methods (IME) for the window.
// It must be called on the main thread.
func (w *Window) initIME() {
	C.initIME(w.hwnd())
}

func (w *Window) SetTextInput(ti *system.TextInput) {
	if w.Glw == nil {
		return
	}
	if ti == nil {
		C.setTextInput(w.hwnd(), 0, 0, 0, 0, 0)
		return
	}
	// window coordinates are client pixels
	r := ti.Caret
	C.setTextInput(w.hwnd(), 1, C.int(r.Min.X), C.int(r.Min.Y), C.int(r.Dx()), C.int(r.Dy()))
}

// winIMEComposition is called by the input method of the given HWND
// with the text being composed, and its selected range in runes.
//
//export winIMEComposition
func winIMEComposition(hwnd C.HWND, text *C.char, start, end C.int) {
	w := dragWindow(uintptr(unsafe.Pointer(hwnd)))
	if w == nil {
		return
	}
	w.Event.Composition(C.GoString(text), int(start), int(end))
	glfw.PostEmptyEvent()
}

// winIMECommit is called by the input method of the given HWND with
// committed text, which is sent as KeyChord events, as typed text.
//
//export winIMECommit
func winIMECommit(hwnd C.HWND, text *C.char) {
	w := dragWindow(uintptr(unsafe.Pointer(hwnd)))
	if w == nil {
		return
	}
	for _, r := range C.GoString(text) {
		w.Event.KeyChord(r, key.CodeUnknown, 0)
	}
	glfw.PostEmptyEvent()
}
