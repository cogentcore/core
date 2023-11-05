// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"goki.dev/goosi"
)

func (app *appImpl) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
	// TODO(kai)
}

func (app *appImpl) HideVirtualKeyboard() {
	// TODO(kai)
}

func (app *appImpl) mainLoop() {
	for {
		select {
		case <-app.mainDone:
			return
		case f := <-app.mainQueue:
			f.f()
			if f.done != nil {
				f.done <- true
			}
		}
	}
}
