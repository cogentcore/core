// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"errors"
	"io/fs"
	"path/filepath"

	"goki.dev/grows/images"
)

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
