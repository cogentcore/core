// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build !3d

package macdriver

import "github.com/goki/gi/oswin"

type mainMenuImpl struct {
	win      *windowImpl
	callback func(win oswin.Window, title string, tag int)
}

func (mm *mainMenuImpl) Window() oswin.Window {
	return mm.win
}

func (mm *mainMenuImpl) SetWindow(win oswin.Window) {
	mm.win = win.(*windowImpl)
}

func (mm *mainMenuImpl) SetFunc(fun func(win oswin.Window, title string, tag int)) {
	mm.callback = fun
}

func (mm *mainMenuImpl) Triggered(win oswin.Window, title string, tag int) {
	if mm.callback == nil {
		return
	}
	mm.callback(win, title, tag)
}
