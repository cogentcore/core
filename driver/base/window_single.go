// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"goki.dev/goosi"
)

// WindowSingle contains the data and logic common to all implementations of [goosi.Window]
// on single-window platforms (mobile, web, and offscreen), as opposed to multi-window
// platforms (desktop), for which you should use [WindowMulti].
// A WindowSingle is associated with a corresponding [goosi.App] type.
// The [goosi.App] type should embed [AppSingle].
type WindowSingle[A goosi.App] struct {
	Window[A]
}
