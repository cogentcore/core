// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android || ios

package driver

import "C"

import (
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/mobile"
)

func driverMain(f func(oswin.App)) {
	mobile.Main(f)
}
