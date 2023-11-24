// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"goki.dev/girl/paint"
)

// AssertCaptureIs asserts that the result of [Capture] is equivalent
// to the image stored at the given filename in the testdata directory,
// with ".png" added to the filename if there is no extension
// (eg: "button" becomes "testdata/button.png").
// If it is not, it fails the test with an error, but continues its
// execution. If there is no image at the given filename in the testdata
// directory, it creates the image.
func AssertCaptureIs(t paint.TestingT, filename string) {
	paint.AssertImage(t, Capture(), filename)
}
