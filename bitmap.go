// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"

	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// bitmap contains various bitmap-related elements, including the Bitmap node
// for showing bitmaps, and image processing utilities

// Bitmap is a Viewport2D that is optimized to render a static bitmap image --
// it expects to be a terminal node and does NOT call rendering etc on its
// children.  It is particularly useful for overlays in drag-n-drop uses --
// can grab the image of another vp and show that
type Bitmap struct {
	Viewport2D
}

var KiT_Bitmap = kit.Types.AddType(&Bitmap{}, BitmapProps)

var BitmapProps = ki.Props{
	"background-color": &Prefs.Colors.Background,
}

// OpenImage opens an image for the bitmap, and resizes to that size
func (bm *Bitmap) OpenImage(filename string) error {
	img, err := OpenImage(filename)
	if err != nil {
		log.Printf("gi.Bitmap.OpenImage -- could not open file: %v, err: %v\n", filename, err)
		return err
	}
	sz := img.Bounds().Size()
	bm.Resize(sz)
	draw.Draw(bm.Pixels, bm.Pixels.Bounds(), img, image.ZP, draw.Src)
	return nil
}

// GrabRenderFrom grabs the rendered image from given node -- copies the
// vpBBox from parent viewport of node (or from viewport directly if node is a
// viewport) -- returns success
func (bm *Bitmap) GrabRenderFrom(nii Node2D) bool {
	ni := nii.AsNode2D()
	nivp := nii.AsViewport2D()
	if nivp != nil && nivp.Pixels != nil {
		sz := nivp.Pixels.Bounds().Size()
		bm.Resize(sz)
		draw.Draw(bm.Pixels, bm.Pixels.Bounds(), nivp.Pixels, image.ZP, draw.Src)
		return true
	}
	nivp = ni.Viewport
	if nivp == nil || nivp.Pixels == nil {
		log.Printf("Bitmap GrabRenderFrom could not grab from node, viewport or pixels nil: %v\n", ni.PathUnique())
		return false
	}
	if ni.VpBBox.Empty() {
		return false // offscreen -- can happen -- no warning -- just check rval
	}
	sz := ni.VpBBox.Size()
	bm.Resize(sz)
	draw.Draw(bm.Pixels, bm.Pixels.Bounds(), nivp.Pixels, ni.VpBBox.Min, draw.Src)
	return true
}

func (bm *Bitmap) Render2D() {
	if bm.PushBounds() {
		bm.DrawIntoParent(bm.Viewport)
		bm.PopBounds()
	}
}

//////////////////////////////////////////////////////////////////////////////////
//  Image IO

func OpenImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	return im, err
}

func OpenPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func SavePNG(path string, im image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, im)
}

//////////////////////////////////////////////////////////////////////////////////
//  Image Manipulations

// see https://github.com/anthonynsimon/bild for a great image manip library
// with parallel speedup

// only put gi-specific, specialized utilities here

// ImageClearer makes an image more transparent -- pct is amount to alter
// alpha transparency factor by -- 100 = fully transparent, 0 = no change --
// affects the image itself -- make a copy of you want to keept the original
// or see bild/blend/multiply -- this is specifically used for gi DND etc
func ImageClearer(im *image.RGBA, pct float32) {
	pct = InRange32(pct, 0, 100.0)
	fact := pct / 100.0
	sz := im.Bounds().Size()
	for y := 0; y < sz.Y; y++ {
		for x := 0; x < sz.X; x++ {
			f32 := NRGBAf32Model.Convert(im.At(x, y)).(NRGBAf32)
			f32.A -= f32.A * fact
			im.Set(x, y, f32)
		}
	}
}
