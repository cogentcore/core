// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"
	"image/draw"

	"cogentcore.org/core/math32"
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

// ObjectSizeFromFit returns the target object size based on the given
// ObjectFits setting, original object size, and target box size
// for the object to fit into.
func ObjectSizeFromFit(fit ObjectFits, obj, box math32.Vector2) math32.Vector2 {
	oar := obj.X / obj.Y
	bar := box.X / box.Y
	var sz math32.Vector2
	switch fit {
	case FitFill:
		return box
	case FitContain, FitScaleDown:
		if oar >= bar {
			// if we have a higher x:y than them, x is our limiting size
			sz.X = box.X
			// and we make our y in proportion to that
			sz.Y = obj.Y * (box.X / obj.X)
		} else {
			// if we have a lower x:y than them, y is our limiting size
			sz.Y = box.Y
			// and we make our x in proportion to that
			sz.X = obj.X * (box.Y / obj.Y)
		}
	case FitCover:
		if oar < bar {
			// if we have a lower x:y than them, x is our limiting size
			sz.X = box.X
			// and we make our y in proportion to that
			sz.Y = obj.Y * (box.X / obj.X)
		} else {
			// if we have a lower x:y than them, y is our limiting size
			sz.Y = box.Y
			// and we make our x in proportion to that
			sz.X = obj.X * (box.Y / obj.Y)
		}
	}
	return sz
}

// ResizeImage resizes the given image according to [Style.ObjectFit]
// in an object of the given box size.
func (s *Style) ResizeImage(img image.Image, box math32.Vector2) image.Image {
	obj := math32.Vector2FromPoint(img.Bounds().Size())
	sz := ObjectSizeFromFit(s.ObjectFit, obj, box)

	if s.ObjectFit == FitScaleDown && sz.X >= obj.X {
		return img
	}
	rimg := transform.Resize(img, int(sz.X), int(sz.Y), transform.NearestNeighbor)
	if s.ObjectFit != FitCover {
		return rimg
	}
	// but we cap the destination size to the size of the containing object
	drect := image.Rect(0, 0, int(min(sz.X, box.X)), int(min(sz.Y, box.Y)))
	dst := image.NewRGBA(drect)
	draw.Draw(dst, drect, rimg, image.Point{}, draw.Src)
	return dst
}
