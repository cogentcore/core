// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package world

import (
	"image"

	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/colors/colormap"
	"cogentcore.org/core/math32"
)

// DepthNorm renders a normalized linear depth map from GPU (0-1 normalized floats) to
// given float slice, which is resized if not already appropriate size.
// if flipY then Y axis is flipped -- input is bottom-Y = 0.
// Camera params determine whether log is used, and max cutoff distance for sensitive
// range of distances -- also has Near / Far required to transform numbers into
// linearized distance values.
func DepthNorm(nd *[]float32, depth []float32, cam *Camera, flipY bool) {
	sz := cam.Size
	totn := sz.X * sz.Y
	*nd = slicesx.SetLength(*nd, totn)
	fpn := cam.Far + cam.Near
	fmn := cam.Far - cam.Near
	var norm float32
	if cam.LogD {
		norm = 1 / math32.Log(1+cam.MaxD)
	} else {
		norm = 1 / cam.MaxD
	}

	twonf := (2.0 * cam.Near * cam.Far)
	for y := 0; y < sz.Y; y++ {
		for x := 0; x < sz.X; x++ {
			oi := y*sz.X + x
			ii := oi
			if flipY {
				ii = (sz.Y-y-1)*sz.X + x
			}
			d := depth[ii]
			z := d*2 - 1                      // convert from 0..1 to -1..1
			lind := twonf / (fpn - (z * fmn)) // untransform
			effd := float32(1)
			if lind < cam.MaxD {
				if cam.LogD {
					effd = norm * math32.Log(1+lind)
				} else {
					effd = norm * lind
				}
			}
			(*nd)[oi] = effd
		}
	}
}

// DepthImage renders an image of linear depth map from GPU (0-1 normalized floats) to
// given image, which must be of appropriate size for map, using given colormap name.
// Camera params determine whether log is used, and max cutoff distance for sensitive
// range of distances -- also has Near / Far required to transform numbers into
// linearized distance values.  Y axis is always flipped.
func DepthImage(img *image.RGBA, depth []float32, cmap *colormap.Map, cam *Camera) {
	if img == nil {
		return
	}
	sz := img.Bounds().Size()
	fpn := cam.Far + cam.Near
	fmn := cam.Far - cam.Near
	var norm float32
	if cam.LogD {
		norm = 1 / math32.Log(1+cam.MaxD)
	} else {
		norm = 1 / cam.MaxD
	}

	twonf := (2.0 * cam.Near * cam.Far)
	for y := 0; y < sz.Y; y++ {
		for x := 0; x < sz.X; x++ {
			ii := (sz.Y-y-1)*sz.X + x // always flip for images
			d := depth[ii]
			z := d*2 - 1                      // convert from 0..1 to -1..1
			lind := twonf / (fpn - (z * fmn)) // untransform
			effd := float32(1)
			if lind < cam.MaxD {
				if cam.LogD {
					effd = norm * math32.Log(1+lind)
				} else {
					effd = norm * lind
				}
			}
			clr := cmap.Map(effd)
			img.Set(x, y, clr)
		}
	}
}
