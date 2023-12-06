// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"image"

	"goki.dev/goosi"
)

// Window contains the data and logic common to all implementations of [goosi.Window]
// on multi-window platforms (desktop), as opposed to single-window
// platforms (mobile, web, and offscreen), for which you should use [WindowSingle].
// A WindowMulti is associated with a corresponding [goosi.App] type.
// The [goosi.App] type should embed [AppMulti].
type WindowMulti[A goosi.App] struct {
	Window[A]

	// Pos is the position of the window
	Pos image.Point

	// WnSize is the size of the window in window-manager coords
	WnSize image.Point

	// PixSize is the pixel size of the window in raw display dots
	PixSize image.Point

	// DevicePixelRatio is a factor that scales the screen's
	// "natural" pixel coordinates into actual device pixels.
	// On OS-X, it is backingScaleFactor = 2.0 on "retina"
	DevicePixelRatio float32

	// PhysicalDPI is the physical dots per inch of the screen,
	// for generating true-to-physical-size output.
	// It is computed as 25.4 * (PixSize.X / PhysicalSize.X)
	// where 25.4 is the number of mm per inch.
	PhysDPI float32 `label:"Physical DPI"`

	// LogicalDPI is the logical dots per inch of the screen,
	// which is used for all rendering.
	// It is: transient zoom factor * screen-specific multiplier * PhysicalDPI
	LogDPI float32 `label:"Logical DPI"`
}
