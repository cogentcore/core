// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"syscall/js"

	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/base"
)

// Window is the implementation of [system.Window] for the web platform.
type Window struct {
	base.WindowSingle[*App]
}

var _ system.Window = &Window{}

func (w *Window) Handle() any {
	return js.Global()
}

func (w *Window) SetTitle(title string) {
	w.WindowSingle.SetTitle(title)
	js.Global().Get("document").Set("title", title)
}
