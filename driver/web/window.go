// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"syscall/js"

	"goki.dev/goosi"
	"goki.dev/goosi/driver/base"
)

// Window is the implementation of [goosi.Window] for the web platform.
type Window struct {
	base.WindowSingle[*App]
}

var _ goosi.Window = &Window{}

func (w *Window) Handle() any {
	return js.Global()
}
