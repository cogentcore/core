// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !linux

package desktop

import "github.com/go-gl/glfw/v3.4/glfw"

// imeInitApp sets up input methods (IME) for the app before GLFW is
// initialized, which is only needed on Linux, returning a function
// to call after it is initialized.
func imeInitApp() (restore func()) { return func() {} }

// imeFocus is called when the window gets or loses the focus,
// which only the input methods on Linux need.
func (w *Window) imeFocus(focused bool) {}

// imeKey returns whether the input method handled the key, which only
// happens on Linux: elsewhere, it gets the keys through the window system.
func (w *Window) imeKey(ky glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) bool {
	return false
}

// imeChar returns whether the char callback is for a key that
// the input method handled, which only happens on Linux.
func (w *Window) imeChar() bool { return false }
