// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"syscall/js"

	"cogentcore.org/core/goosi"
	"cogentcore.org/core/goosi/driver/base"
)

// Window is the implementation of [goosi.Window] for the web platform.
type Window struct { //gti:add
	base.WindowSingle[*App]
}

var _ goosi.Window = &Window{}

func (w *Window) Handle() any {
	return js.Global()
}

func (w *Window) SetTitle(title string) {
	w.WindowSingle.SetTitle(title)
	js.Global().Get("document").Set("title", title)
}
