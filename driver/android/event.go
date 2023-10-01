// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package android

import "C"
import (
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
)

//export keyboardTyped
func keyboardTyped(str *C.char) {
	for _, r := range C.GoString(str) {
		code := convAndroidKeyCode(r)
		theApp.window.EvMgr.Key(events.KeyDown, r, code, 0) // TODO: modifiers
		theApp.window.EvMgr.Key(events.KeyUp, r, code, 0)   // TODO: modifiers
	}
}

//export keyboardDelete
func keyboardDelete() {
	theApp.window.EvMgr.Key(events.KeyDown, 0, key.CodeDeleteBackspace, 0) // TODO: modifiers
	theApp.window.EvMgr.Key(events.KeyUp, 0, key.CodeDeleteBackspace, 0)   // TODO: modifiers
}
