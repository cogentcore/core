// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package core

import "syscall/js"

// init makes the window of the tests on the web (see goscript_test.go) the
// size of a desktop window, as the tests expect, rather than that of a
// phone, which is the default size of the frame of the tests in the browser,
// with the spacing of the offscreen platform of the other tests (see
// [DefaultSpacing]), rather than that of the desktop platform of the browser.
func init() {
	LoadAllSettings() // (first, as in NewBody, which then does not load them again)
	AppearanceSettings.Spacing = 100
	if fe := js.Global().Get("frameElement"); fe.Truthy() {
		style := fe.Get("style")
		style.Set("width", "1280px")
		style.Set("height", "1024px")
	}
}
