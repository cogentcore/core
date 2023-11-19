// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"errors"
	"image"
	"io/fs"
	"path/filepath"

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
	return grr.Log0(images.Save(Capture(), filename))
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

////// Testing

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...any)
}

// AssertCaptureIs asserts that the result of [Capture] is equivalent
// to the image stored at the given filename in the testdata directory,
// with ".png" added to the filename if there is no extension
// (eg: "button" becomes "testdata/button.png").
// If it is not, it fails the test with an error, but continues its
// execution. If there is no image at the given filename in the testdata
// directory, it creates the image
func AssertCaptureIs(t TestingT, filename string) {
	capture := Capture()

	filename = filepath.Join("testdata", filename)
	if filepath.Ext(filename) == "" {
		filename += ".png"
	}
	img, _, err := images.Open(filename)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("goosi.AssertCaptureIs: error opening saved image: %v", err)
			return
		}
		// we don't have the file yet, so we make it
		err := images.Save(capture, filename)
		if err != nil {
			t.Errorf("goosi.AssertCaptureIs: error saving image: %v", err)
		}
		return
	}

	if capture != img {
		badFilename := filename + ".bad"
		t.Errorf("goosi.AssertCaptureIs: image %q is not the same as expected; see %q", filename, badFilename)
		err := images.Save(capture, badFilename)
		if err != nil {
			t.Errorf("goosi.AssertCaptureIs: error saving bad image: %v", err)
		}
	}
}
