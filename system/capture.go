// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"image"

	"cogentcore.org/core/grr"
	"cogentcore.org/core/xio/images"
)

// Capture tells the app drawer to capture its next frame as an image.
// Once it gets that image, it returns it. It is currently only supported
// on platform [Offscreen].
func Capture() *image.RGBA {
	NeedsCapture = true
	TheApp.Window(0).Drawer().EndDraw() // triggers capture
	return CaptureImage
}

// CaptureAs is a helper function that saves the result of [Capture] to the given filename.
// It automatically logs any error in addition to returning it.
func CaptureAs(filename string) error {
	return grr.Log(images.Save(Capture(), filename))
}

var (
	// NeedsCapture is whether the app drawer needs to capture its next
	// frame. End-user code should just use [Capture].
	NeedsCapture bool
	// CaptureImage is the variable that stores the image captured in
	// [Capture]. End-user code should just use [Capture].
	CaptureImage *image.RGBA
)
