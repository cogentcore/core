// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"image"

	"goki.dev/grows/images"
	"goki.dev/grr"
)

// Capture tells the app drawer to capture its next frame as an image.
// Once it gets that image, it returns it. It is currently only supported
// with the offscreen build tag.
func Capture() *image.RGBA {
	NeedsCapture = true
	return <-CaptureImage
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
	// CaptureImage is a channel that sends the image captured after
	// setting [NeedsCapture] to true. End-user code should just use
	// [Capture].
	CaptureImage = make(chan *image.RGBA)
)
