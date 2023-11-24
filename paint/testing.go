// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"errors"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"goki.dev/colors"
	"goki.dev/grows/images"
)

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...any)
}

// UpdateTestImages indicates whether to update currently saved test
// images in [AssertImage] instead of comparing against them.
// It is automatically set if the env variable "UPDATE_TEST_IMAGES" is "true",
// and it should typically only be set through that. It should only be
// set when behavior has been updated that causes test images to change,
// and it should only be set once and then turned back off.
var UpdateTestImages = os.Getenv("UPDATE_TEST_IMAGES") == "true"

// AssertImage asserts that the given image is equivalent
// to the image stored at the given filename in the testdata directory,
// with ".png" added to the filename if there is no extension
// (eg: "button" becomes "testdata/button.png").
// If it is not, it fails the test with an error, but continues its
// execution. If there is no image at the given filename in the testdata
// directory, it creates the image.
func AssertImage(t TestingT, img image.Image, filename string) {
	filename = filepath.Join("testdata", filename)
	if filepath.Ext(filename) == "" {
		filename += ".png"
	}

	err := os.MkdirAll(filepath.Dir(filename), 0750)
	if err != nil {
		t.Errorf("error making testdata directory: %v", err)
	}

	ext := filepath.Ext(filename)
	failFilename := strings.TrimSuffix(filename, ext) + ".fail" + ext

	if UpdateTestImages {
		err := images.Save(img, filename)
		if err != nil {
			t.Errorf("AssertImage: error saving updated image: %v", err)
		}
		err = os.RemoveAll(failFilename)
		if err != nil {
			t.Errorf("AssertImage: error removing old fail image: %v", err)
		}
		return
	}

	fimg, _, err := images.Open(filename)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("AssertImage: error opening saved image: %v", err)
			return
		}
		// we don't have the file yet, so we make it
		err := images.Save(img, filename)
		if err != nil {
			t.Errorf("AssertImage: error saving new image: %v", err)
		}
		return
	}

	failed := false

	ibounds := img.Bounds()
	fbounds := fimg.Bounds()
	if ibounds != fbounds {
		t.Errorf("AssertImage: expected bounds %v for image for %s, but got bounds %v; see %s", fbounds, filename, ibounds, failFilename)
		failed = true
	} else {
		for y := ibounds.Min.Y; y < ibounds.Max.Y; y++ {
			for x := ibounds.Min.X; x < ibounds.Max.X; x++ {
				cc := colors.AsRGBA(img.At(x, y))
				ic := colors.AsRGBA(fimg.At(x, y))
				if cc != ic {
					t.Errorf("AssertImage: image for %s is not the same as expected; see %s; expected color %v at (%d, %d), but got %v", filename, failFilename, ic, x, y, cc)
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
		err := images.Save(img, failFilename)
		if err != nil {
			t.Errorf("AssertImage: error saving fail image: %v", err)
		}
	} else {
		err := os.RemoveAll(failFilename)
		if err != nil {
			t.Errorf("AssertImage: error removing old fail image: %v", err)
		}
	}
}
