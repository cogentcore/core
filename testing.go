// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"goki.dev/grows/images"
)

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...any)
}

// UpdateTestImages indicates whether to update currently saved test
// images in [AssertCaptureIs] instead of comparing against them.
// It is automatically set if the env variable "UPDATE_TEST_IMAGES" is "true",
// and it should typically only be set through that. It should only be
// set when behavior has been updated that causes test images to change.
var UpdateTestImages = os.Getenv("UPDATE_TEST_IMAGES") == "true"

// AssertCaptureIs asserts that the result of [Capture] is equivalent
// to the image stored at the given filename in the testdata directory,
// with ".png" added to the filename if there is no extension
// (eg: "button" becomes "testdata/button.png").
// If it is not, it fails the test with an error, but continues its
// execution. If there is no image at the given filename in the testdata
// directory, it creates the image
func AssertCaptureIs(t TestingT, filename string) {
	capture := Capture()

	err := os.MkdirAll("testdata", 0750)
	if err != nil {
		t.Errorf("error making testdata directory: %v", err)
	}

	filename = filepath.Join("testdata", filename)
	if filepath.Ext(filename) == "" {
		filename += ".png"
	}

	ext := filepath.Ext(filename)
	failFilename := strings.TrimSuffix(filename, ext) + ".fail" + ext

	if UpdateTestImages {
		err := images.Save(capture, filename)
		if err != nil {
			t.Errorf("goosi.AssertCaptureIs: error saving updated image: %v", err)
		}
		err = os.RemoveAll(failFilename)
		if err != nil {
			t.Errorf("goosi.AssertCaptureIs: error removing old fail image: %v", err)
		}
		return
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
			t.Errorf("goosi.AssertCaptureIs: error saving new image: %v", err)
		}
		return
	}

	failed := false

	cbounds := capture.Bounds()
	ibounds := img.Bounds()
	if cbounds != ibounds {
		failed = true
	} else {
		for y := cbounds.Min.Y; y < cbounds.Max.Y; y++ {
			for x := cbounds.Min.X; x < cbounds.Max.X; x++ {
				cc := capture.At(x, y)
				ic := img.At(x, y)
				if cc != ic {
					t.Errorf("goosi.AssertCaptureIs: image for %q is not the same as expected; see %q; expected color %v at (%d, %d), but got %v", filename, failFilename, ic, x, y, cc)
					failed = true
					break
				}
			}
			if failed {
				break
			}
		}
	}

	if failed {
		err := images.Save(capture, failFilename)
		if err != nil {
			t.Errorf("goosi.AssertCaptureIs: error saving fail image: %v", err)
		}
	} else {
		err := os.RemoveAll(failFilename)
		if err != nil {
			t.Errorf("goosi.AssertCaptureIs: error removing old fail image: %v", err)
		}
	}
}
