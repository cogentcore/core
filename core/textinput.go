// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/system"
)

// SetTextInput sets the state of text input in the window of the widget
// for the input method (IME) of the operating system, with the rectangle
// of the text cursor, or the cursor of the input method within the text
// being composed, in window coordinates (see [system.Window.SetTextInput]).
// Focused text widgets call it when their cursor moves, and it does
// nothing if the state has not changed. The input method then sends the
// text being composed as [events.Composition] events.
func (wb *WidgetBase) SetTextInput(caret image.Rectangle) {
	rw := wb.textInputWindow()
	if rw == nil {
		return
	}
	if rw.textInput != nil && rw.textInput.Caret == caret && rw.textInputWidget == wb.This {
		return
	}
	rw.textInput = &system.TextInput{Caret: caret}
	rw.textInputWidget = wb.This.(Widget)
	rw.SystemWindow.SetTextInput(rw.textInput)
}

// ClearTextInput clears the state of text input in the window of the
// widget if the widget set it with [WidgetBase.SetTextInput], which
// ends any text being composed. Text widgets call it when they lose
// the focus.
func (wb *WidgetBase) ClearTextInput() {
	rw := wb.textInputWindow()
	if rw == nil || rw.textInputWidget == nil || rw.textInputWidget.AsWidget() != wb {
		return
	}
	rw.textInput = nil
	rw.textInputWidget = nil
	rw.SystemWindow.SetTextInput(nil)
}

// textInputWindow returns the render window of the widget,
// or nil if it does not have a system window.
func (wb *WidgetBase) textInputWindow() *renderWindow {
	if wb.Scene == nil {
		return nil
	}
	rw := wb.Scene.RenderWindow()
	if rw == nil || rw.SystemWindow == nil {
		return nil
	}
	return rw
}
