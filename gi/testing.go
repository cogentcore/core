// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"goki.dev/colors"
	"goki.dev/goosi"
	"goki.dev/goosi/driver"
	"goki.dev/goosi/events"
	"goki.dev/grows/images"
)

// RunTest is a simple helper function that runs the given
// function after calling [driver.Main] and [Init]. It should
// only be used in tests. For example:
//
//	func TestSomething(t *testing.T) {
//		gi.RunTest(func() {
//			sc := gi.NewScene()
//			gi.NewLabel(sc).SetText("Something")
//			sc.AssertPixelsOnShow(t, "something")
//		})
//	}
func RunTest(test func()) {
	driver.Main(func(a goosi.App) {
		Init()
		test()
	})
}

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...any)
}

// UpdateTestImages indicates whether to update currently saved test
// images in [AssertCaptureIs] instead of comparing against them.
// It is automatically set if the env variable "UPDATE_TEST_IMAGES" is "true",
// and it should typically only be set through that. It should only be
// set when behavior has been updated that causes test images to change,
// and it should only be set once and then turned back off.
var UpdateTestImages = os.Getenv("UPDATE_TEST_IMAGES") == "true"

// AssertPixelsOnShow is a helper function that makes a new window from
// the scene, waits until it is shown, calls [Scene.AssertPixels]
// with the given values, and then closes the window.
// It does not return until all of those steps are completed.
func (sc *Scene) AssertPixelsOnShow(t TestingT, filename string) {
	showed := make(chan struct{})
	sc.OnShow(func(e events.Event) {
		sc.AssertPixels(t, filename)
		showed <- struct{}{}
	})
	sc.NewWindow().Run()
	<-showed
	sc.Close()
}

// AssertPixels asserts that [Scene.Pixels] is equivalent
// to the image stored at the given filename in the testdata directory,
// with ".png" added to the filename if there is no extension
// (eg: "button" becomes "testdata/button.png").
// If it is not, it fails the test with an error, but continues its
// execution. If there is no image at the given filename in the testdata
// directory, it creates the image
func (sc *Scene) AssertPixels(t TestingT, filename string) {
	capture := sc.Pixels

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
		err := images.Save(capture, filename)
		if err != nil {
			t.Errorf("Scene.AssertPixels: error saving updated image: %v", err)
		}
		err = os.RemoveAll(failFilename)
		if err != nil {
			t.Errorf("Scene.AssertPixels: error removing old fail image: %v", err)
		}
		return
	}

	img, _, err := images.Open(filename)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Scene.AssertPixels: error opening saved image: %v", err)
			return
		}
		// we don't have the file yet, so we make it
		err := images.Save(capture, filename)
		if err != nil {
			t.Errorf("Scene.AssertPixels: error saving new image: %v", err)
		}
		return
	}

	failed := false

	cbounds := capture.Bounds()
	ibounds := img.Bounds()
	if cbounds != ibounds {
		t.Errorf("Scene.AssertPixels: expected bounds %v for image for %s, but got bounds %v; see %s", ibounds, filename, cbounds, failFilename)
		failed = true
	} else {
		for y := cbounds.Min.Y; y < cbounds.Max.Y; y++ {
			for x := cbounds.Min.X; x < cbounds.Max.X; x++ {
				cc := colors.AsRGBA(capture.At(x, y))
				ic := colors.AsRGBA(img.At(x, y))
				if cc != ic {
					t.Errorf("Scene.AssertPixels: image for %s is not the same as expected; see %s; expected color %v at (%d, %d), but got %v", filename, failFilename, ic, x, y, cc)
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
			t.Errorf("Scene.AssertPixels: error saving fail image: %v", err)
		}
	} else {
		err := os.RemoveAll(failFilename)
		if err != nil {
			t.Errorf("Scene.AssertPixels: error removing old fail image: %v", err)
		}
	}
}
