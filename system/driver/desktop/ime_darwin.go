// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package desktop

/*
#include <stdlib.h>
void initIME(void* nswin);
void setTextInput(void* nswin, int active, double x, double y, double w, double h);
*/
import "C"

import (
	"unsafe"

	"cogentcore.org/core/system"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// initIME sets up input methods (IME) for the window.
// It must be called on the main thread.
func (w *Window) initIME() {
	C.initIME(w.Glw.GetCocoaWindow())
}

func (w *Window) SetTextInput(ti *system.TextInput) {
	if w.Glw == nil {
		return
	}
	if ti == nil {
		C.setTextInput(w.Glw.GetCocoaWindow(), 0, 0, 0, 0, 0)
		return
	}
	// window coordinates are pixels, and the view is in points
	dpr := float64(w.DevicePixelRatio)
	r := ti.Caret
	C.setTextInput(w.Glw.GetCocoaWindow(), 1, C.double(float64(r.Min.X)/dpr), C.double(float64(r.Min.Y)/dpr),
		C.double(float64(r.Dx())/dpr), C.double(float64(r.Dy())/dpr))
}

// macIMEComposition is called by the input method of the given NSWindow
// with the text being composed, and its selected range in runes.
//
//export macIMEComposition
func macIMEComposition(nswin unsafe.Pointer, text *C.char, start, end C.int) {
	w := dragWindow(uintptr(nswin))
	if w == nil {
		return
	}
	w.Event.Composition(C.GoString(text), int(start), int(end))
	glfw.PostEmptyEvent()
}
