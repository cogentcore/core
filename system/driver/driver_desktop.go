// Copyright 2018 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !(android || ios || js || offscreen)

package driver

import (
	"os"
	"slices"
	"testing"

	"cogentcore.org/core/system/driver/desktop"
	"cogentcore.org/core/system/driver/offscreen"
)

func init() {
	// TODO(kai/binsize): consider figuring out how to do this without
	// increasing binary sizes; also supporting running tests on mobile and web
	if testing.Testing() || slices.Contains(os.Args, "-nogui") {
		offscreen.Init()
		return
	}
	desktop.Init()
}
