// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package desktop

/*
#import <Cocoa/Cocoa.h>
void setAppIconPNG(const void* data, int len);
*/
import "C"

import (
	"bytes"
	"image"
	"image/png"
	"unsafe"

	"cogentcore.org/core/base/errors"
)

// appIcon is the image last set as the app icon by [setAppIcon].
var appIcon image.Image

// minAppIcon is the minimum size of the image of the app icon, as in the
// Dock: the smaller window icon images do not replace the icon of the app.
const minAppIcon = 256

// setAppIcon sets the icon of the app, shown in the Dock while it runs,
// to the largest of the given window icon images if it is at least
// [minAppIcon], as windows have no icons on macOS (glfw does not set
// them). This works without an app bundle. It must be called on the
// main thread.
func setAppIcon(images []image.Image) {
	var img image.Image
	for _, im := range images {
		if img == nil || im.Bounds().Dx() > img.Bounds().Dx() {
			img = im
		}
	}
	if img == nil || img == appIcon || img.Bounds().Dx() < minAppIcon {
		return
	}
	var b bytes.Buffer
	if errors.Log(png.Encode(&b, img)) != nil {
		return
	}
	appIcon = img
	C.setAppIconPNG(unsafe.Pointer(&b.Bytes()[0]), C.int(b.Len()))
}
