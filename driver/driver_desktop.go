// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !(android || ios || js || offscreen)

package driver

import (
	"testing"

	"goki.dev/goosi"
	"goki.dev/goosi/driver/desktop"
	"goki.dev/goosi/driver/offscreen"
)

func driverMain(f func(goosi.App)) {
	// TODO(kai): consider supporting running tests on mobile and web
	if testing.Testing() {
		offscreen.Main(f)
		return
	}
	desktop.Main(f)
}
