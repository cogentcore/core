// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"errors"
	"image"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/num"
)

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...any)
}

// UpdateTestImages indicates whether to update currently saved test
// images in [AssertImage] instead of comparing against them.
// It is automatically set if the build tag "update" is specified,
// or if the environment variable "CORE_UPDATE_TESTDATA" is set to "true".
// It should typically only be set through those methods. It should only be
// set when behavior has been updated that causes test images to change,
// and it should only be set once and then turned back off.
var UpdateTestImages = updateTestImages

// CompareUint8 returns true if two numbers are more different than tol
func CompareUint8(cc, ic uint8, tol int) bool {
	d := int(cc) - int(ic)
	if d < -tol {
		return false
	}
	if d > tol {
		return false
	}
	return true
}

// CompareColors returns true if two colors are more different than tol
func CompareColors(cc, ic color.RGBA, tol int) bool {
	if !CompareUint8(cc.R, ic.R, tol) {
		return false
	}
	if !CompareUint8(cc.G, ic.G, tol) {
		return false
	}
	if !CompareUint8(cc.B, ic.B, tol) {
		return false
	}
	if !CompareUint8(cc.A, ic.A, tol) {
		return false
	}
	return true
}

// DiffImage returns the difference between two images,
// with pixels having the abs of the difference between pixels.
func DiffImage(a, b image.Image) image.Image {
	ab := a.Bounds()
	di := image.NewRGBA(ab)
	for y := ab.Min.Y; y < ab.Max.Y; y++ {
		for x := ab.Min.X; x < ab.Max.X; x++ {
			cc := color.RGBAModel.Convert(a.At(x, y)).(color.RGBA)
			ic := color.RGBAModel.Convert(b.At(x, y)).(color.RGBA)
			r := uint8(num.Abs(int(cc.R) - int(ic.R)))
			g := uint8(num.Abs(int(cc.G) - int(ic.G)))
			b := uint8(num.Abs(int(cc.B) - int(ic.B)))
			c := color.RGBA{r, g, b, 255}
			di.Set(x, y, c)
		}
	}
	return di
}

// Assert asserts that the given image is equivalent
// to the image stored at the given filename in the testdata directory,
// with ".png" added to the filename if there is no extension
// (eg: "button" becomes "testdata/button.png"). Forward slashes are
// automatically replaced with backslashes on Windows.
// If it is not, it fails the test with an error, but continues its
// execution. If there is no image at the given filename in the testdata
// directory, it creates the image.
func Assert(t TestingT, img image.Image, filename string) {
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
	diffFilename := strings.TrimSuffix(filename, ext) + ".diff" + ext

	if UpdateTestImages {
		err := Save(img, filename)
		if err != nil {
			t.Errorf("AssertImage: error saving updated image: %v", err)
		}
		err = os.RemoveAll(failFilename)
		if err != nil {
			t.Errorf("AssertImage: error removing old fail image: %v", err)
		}
		os.RemoveAll(diffFilename)
		return
	}

	fimg, _, err := Open(filename)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("AssertImage: error opening saved image: %v", err)
			return
		}
		// we don't have the file yet, so we make it
		err := Save(img, filename)
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
				cc := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
				ic := color.RGBAModel.Convert(fimg.At(x, y)).(color.RGBA)
				// TODO(#1456): reduce tolerance to 1 after we fix rendering inconsistencies
				if !CompareColors(cc, ic, 10) {
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
		err := Save(img, failFilename)
		if err != nil {
			t.Errorf("AssertImage: error saving fail image: %v", err)
		}
		err = Save(DiffImage(img, fimg), diffFilename)
		if err != nil {
			t.Errorf("AssertImage: error saving diff image: %v", err)
		}
	} else {
		err := os.RemoveAll(failFilename)
		if err != nil {
			t.Errorf("AssertImage: error removing old fail image: %v", err)
		}
		os.RemoveAll(diffFilename)
	}
}
