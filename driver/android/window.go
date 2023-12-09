// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package android

import (
	"goki.dev/goosi/driver/base"
)

// Window is the implementation of [goosi.Window] for the Android platform.
type Window struct { //gti:add
	base.WindowSingle[*App]
}

func (w *Window) Handle() any {
	return w.App.Winptr
}
