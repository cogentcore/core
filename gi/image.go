// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/png"
	"io/fs"
	"log"
	"log/slog"
	"os"

	"github.com/anthonynsimon/bild/clone"
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/grows/images"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"golang.org/x/image/draw"
)

// image contains various image-related elements, including the Image node
// for showing images, and image processing utilities

// Image is a Widget that is optimized to render a static bitmap image --
// it expects to be a terminal node and does NOT call rendering etc on its
// children.  It is particularly useful for overlays in drag-n-drop uses --
// can grab the image of another vp and show that
type Image struct {
	WidgetBase

	// file name of image loaded -- set by OpenImage
	Filename FileName `set:"-"`

	// the bitmap image
	Pixels *image.RGBA `copy:"-" view:"-" xml:"-" json:"-" set:"-"`

	// cached size of the last rendered image
	PrevSize image.Point `copy:"-" xml:"-" json:"-" set:"-"`
}

func (im *Image) CopyFieldsFrom(frm any) {
	fr := frm.(*Image)
	im.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	im.Filename = fr.Filename
}

func (im *Image) OnInit() {
	im.HandleWidgetEvents()
	im.ImageStyles()
}

func (im *Image) ImageStyles() {
	im.Style(func(s *styles.Style) {
		if im.Pixels != nil {
			sz := im.Pixels.Bounds().Size()
			s.Min.X.Dp(float32(sz.X))
			s.Min.Y.Dp(float32(sz.Y))
		}
	})
}

// OpenImage sets the image to the image located at the given filename.
func (im *Image) OpenImage(filename FileName) error {
	img, _, err := images.Open(string(filename))
	if err != nil {
		slog.Error("gi.Image.OpenImage: could not open", "file", filename, "err", err)
		return err
	}
	im.Filename = filename
	im.SetImage(img)
	return nil
}

// OpenImageFS sets the image to the image located at the given filename in the given fs.
func (im *Image) OpenImageFS(fsys fs.FS, filename FileName) error {
	img, _, err := images.OpenFS(fsys, string(filename))
	if err != nil {
		slog.Error("gi.Image.OpenImage: could not open", "file", filename, "err", err)
		return err
	}
	im.Filename = filename
	im.SetImage(img)
	return nil
}

// SetImage sets the image to the given image.
// It copies from the given image into an internal image.
func (im *Image) SetImage(img image.Image) {
	updt := im.UpdateStart()
	defer im.UpdateEnd(updt)

	im.Pixels = clone.AsRGBA(img)
}

// GrabRenderFrom grabs the rendered image from given node
func (im *Image) GrabRenderFrom(wi Widget) {
	img := GrabRenderFrom(wi)
	if img != nil {
		im.Pixels = img
	}
}

func (im *Image) DrawIntoScene() {
	if im.Pixels == nil {
		return
	}
	r := im.Geom.TotalBBox
	sp := image.Point{}
	if im.Par != nil { // use parents children bbox to determine where we can draw
		_, pwb := AsWidget(im.Par)
		pbb := pwb.Geom.ContentBBox
		nr := r.Intersect(pbb)
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			slog.Error("gi.Image bad bounding box", "path", im, "startPos", sp, "bbox", r, "parBBox", pbb)
			return
		}
		r = nr
	}
	rimg := im.Styles.ResizeImage(im.Pixels, im.Geom.Size.Actual.Content)
	draw.Draw(im.Sc.Pixels, r, rimg, sp, draw.Over)
}

func (im *Image) Render() {
	if im.PushBounds() {
		im.RenderChildren()
		im.DrawIntoScene()
		im.PopBounds()
	}
}

//////////////////////////////////////////////////////////////////////////////////
//  Image IO

// GrabRenderFrom grabs the rendered image from given node
// if nil, then image could not be grabbed
func GrabRenderFrom(wi Widget) *image.RGBA {
	wb := wi.AsWidget()
	sc := wb.Sc
	if sc == nil || sc.Pixels == nil {
		log.Printf("gi.GrabRenderFrom could not grab from node, scene or pixels nil: %v\n", wb.Path())
		return nil
	}
	if wb.Geom.TotalBBox.Empty() {
		return nil // offscreen -- can happen -- no warning -- just check rval
	}
	sz := wb.Geom.TotalBBox.Size()
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), sc.Pixels, wb.Geom.TotalBBox.Min, draw.Src)
	return img
}

// ImageToRGBA returns given image as an image.RGBA (no conversion if it is already)
func ImageToRGBA(img image.Image) *image.RGBA {
	if rg, ok := img.(*image.RGBA); ok {
		return rg
	}
	return images.CloneAsRGBA(img)
}

// OpenPNG opens an image encoded in the PNG format
func OpenPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

// SavePNG saves an image encoded in the PNG format
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
// affects the image itself -- make a copy if you want to keep the original
// or see bild/blend/multiply -- this is specifically used for gi DND etc
func ImageClearer(im *image.RGBA, pct float32) {
	pct = mat32.Clamp(pct, 0, 100.0)
	fact := pct / 100.0
	sz := im.Bounds().Size()
	for y := 0; y < sz.Y; y++ {
		for x := 0; x < sz.X; x++ {
			f32 := colors.NRGBAF32Model.Convert(im.At(x, y)).(colors.NRGBAF32)
			f32.A -= f32.A * fact
			im.Set(x, y, f32)
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////
//  Props

// TODO(kai): move this to new system

var ImageProps = ki.Props{
	"Toolbar": ki.PropSlice{
		{"OpenImage", ki.Props{
			"desc": "Open an image for this bitmap.  if width and/or height is > 0, then image is rescaled to that dimension, preserving aspect ratio if other one is not set",
			"icon": icons.Open,
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".png,.jpg",
				}},
				{"Width", ki.Props{
					"desc": "width in raw display dots -- use image size if 0",
				}},
				{"Height", ki.Props{
					"desc": "height in raw display dots -- use image size if 0",
				}},
			},
		}},
	},
}
