// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android && !offscreen

package driver

import "C"

import (
	"goki.dev/goosi"
	"goki.dev/goosi/driver/android"
)

func driverMain(f func(goosi.App)) {
	android.Main(f)
}
