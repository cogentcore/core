// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"
	"image/draw"

	"cogentcore.org/core/mat32"
	"github.com/anthonynsimon/bild/transform"
)

// ObjectFits are the different ways in which a replaced element
// (image, video, etc) can be fit into its containing box.
type ObjectFits int32 //enums:enum -trim-prefix Fit -transform kebab

const (
	// FitFill indicates that the replaced object will fill
	// the element's entire content box, stretching if necessary.
	FitFill ObjectFits = iota

	// FitContain indicates that the replaced object will resize
	// as large as possible while fully fitting within the element's
	// content box and maintaining its aspect ratio. Therefore,
	// it may not fill the entire element.
	FitContain

	// FitCover indicates that the replaced object will fill
	// the element's entire content box, clipping if necessary.
	FitCover

	// FitNone indicates that the replaced object will not resize.
	FitNone

	// FitScaleDown indicates that the replaced object will size
	// as if [FitNone] or [FitContain] was specified, using
	// whichever will result in a smaller final size.
	FitScaleDown
)

// ResizeImage resizes the given image according to [Style.ObjectFit]
// in an object of the given size.
func (s *Style) ResizeImage(img image.Image, size mat32.Vec2) image.Image {
	sz := img.Bounds().Size()
	szx, szy := float32(sz.X), float32(sz.Y)
	// image and box aspect ratio
	iar := szx / szy
	bar := size.X / size.Y
	switch s.ObjectFit {
	case FitFill:
		return transform.Resize(img, int(size.X), int(size.Y), transform.Linear)
	case FitContain, FitScaleDown:
		var x, y float32
		if iar >= bar {
			// if we have a higher x:y than them, x is our limiting size
			x = size.X
			// and we make our y in proportion to that
			y = szy * (size.X / szx)
		} else {
			// if we have a lower x:y than them, y is our limiting size
			y = size.Y
			// and we make our x in proportion to that
			x = szx * (size.Y / szy)
		}
		// in FitScaleDown, if containing results in a larger image, we use
		// the original image instead
		if s.ObjectFit == FitScaleDown && x >= szx {
			return img
		}
		return transform.Resize(img, int(x), int(y), transform.Linear)
	case FitCover:
		var x, y float32
		if iar < bar {
			// if we have a lower x:y than them, x is our limiting size
			x = size.X
			// and we make our y in proportion to that
			y = szy * (size.X / szx)
		} else {
			// if we have a lower x:y than them, y is our limiting size
			y = size.Y
			// and we make our x in proportion to that
			x = szx * (size.Y / szy)
		}
		// our source image is the computed size
		rimg := transform.Resize(img, int(x), int(y), transform.Linear)
		// but we cap the destination size to the size of the containg object
		drect := image.Rect(0, 0, int(min(x, size.X)), int(min(y, size.Y)))
		dst := image.NewRGBA(drect)
		draw.Draw(dst, drect, rimg, image.Point{}, draw.Src)
		return dst
	}
	return img
}
