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

	"github.com/anthonynsimon/bild/transform"
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/grows/images"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
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

	// size of the image
	Size image.Point `set:"-"`

	// the bitmap image
	Pixels *image.RGBA `copy:"-" view:"-" xml:"-" json:"-" set:"-"`
}

func (im *Image) CopyFieldsFrom(frm any) {
	fr := frm.(*Image)
	im.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	im.Size = fr.Size
	im.Filename = fr.Filename
}

func (im *Image) OnInit() {
	im.HandleWidgetEvents()
	im.ImageStyles()
}

func (im *Image) ImageStyles() {
	im.Style(func(s *styles.Style) {
		s.Min.X.Dp(float32(im.Size.X))
		s.Min.Y.Dp(float32(im.Size.Y))
	})
}

// SetSize sets size of the bitmap image.
// This does not resize any existing image, just makes a new image
// if the size is different
func (im *Image) SetSize(nwsz image.Point) *Image {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return im
	}
	im.Size = nwsz // always make sure
	if im.Pixels != nil && im.Pixels.Bounds().Size() == nwsz {
		return im
	}
	im.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
	return im
}

// OpenImage opens an image for the bitmap, and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension
func (im *Image) OpenImage(filename FileName, width, height float32) error {
	img, _, err := images.Open(string(filename))
	if err != nil {
		slog.Error("gi.Image.OpenImage: could not open", "file", filename, "err", err)
		return err
	}
	im.Filename = filename
	im.SetImage(img, width, height)
	return nil
}

// OpenImageFS opens an image for the bitmap, and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension
func (im *Image) OpenImageFS(fsys fs.FS, filename FileName, width, height float32) error {
	img, _, err := images.OpenFS(fsys, string(filename))
	if err != nil {
		slog.Error("gi.Image.OpenImage: could not open", "file", filename, "err", err)
		return err
	}
	im.Filename = filename
	im.SetImage(img, width, height)
	return nil
}

// SetImage sets an image for the bitmap , and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension.  Copies from given image into internal image for this bitmap.
func (im *Image) SetImage(img image.Image, width, height float32) {
	updt := im.UpdateStart()
	defer im.UpdateEnd(updt)

	sz := img.Bounds().Size()
	if width <= 0 && height <= 0 {
		im.SetSize(sz)
		draw.Draw(im.Pixels, im.Pixels.Bounds(), img, image.Point{}, draw.Src)
	} else {
		tsz := sz
		transformer := draw.BiLinear
		scx := float32(1)
		scy := float32(1)
		if width > 0 {
			scx = width / float32(sz.X)
			tsz.X = int(width)
		}
		if height > 0 {
			scy = height / float32(sz.Y)
			tsz.Y = int(height)
		}
		im.SetSize(tsz)
		m := mat32.Scale2D(scx, scy)
		s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
		transformer.Transform(im.Pixels, s2d, img, img.Bounds(), draw.Src, nil)
	}
}

// GrabRenderFrom grabs the rendered image from given node
func (im *Image) GrabRenderFrom(wi Widget) {
	img := GrabRenderFrom(wi)
	if img != nil {
		im.Pixels = img
		im.Size = im.Pixels.Bounds().Size()
	}
}

func (im *Image) DrawIntoScene(sc *Scene) {
	if im.Pixels == nil {
		return
	}
	r := im.Alloc.BBox
	sp := image.Point{}
	if im.Par != nil { // use parents children bbox to determine where we can draw
		_, pwb := AsWidget(im.Par)
		pbb := pwb.Alloc.ContentBBox
		nr := r.Intersect(pbb)
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			slog.Error("gi.Image bad bounding box", "path", im, "startPos", sp, "bbox", r, "parBBox", pbb)
			return
		}
		r = nr
	}
	draw.Draw(sc.Pixels, r, im.Pixels, sp, draw.Over)
}

func (im *Image) Render(sc *Scene) {
	if im.PushBounds(sc) {
		im.RenderChildren(sc)
		im.DrawIntoScene(im.Sc)
		im.PopBounds(sc)
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
	if wb.Alloc.BBox.Empty() {
		return nil // offscreen -- can happen -- no warning -- just check rval
	}
	sz := wb.Alloc.BBox.Size()
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), sc.Pixels, wb.Alloc.BBox.Min, draw.Src)
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

// ImageSizeMax computes the size of image where the largest size (X or Y) is set to maxSz
func ImageSizeMax(sz image.Point, maxSz int) image.Point {
	tsz := sz
	if sz.X > sz.Y {
		tsz.X = maxSz
		tsz.Y = int(float32(sz.Y) * (float32(tsz.X) / float32(sz.X)))
	} else {
		tsz.Y = maxSz
		tsz.X = int(float32(sz.X) * (float32(tsz.Y) / float32(sz.Y)))
	}
	return tsz
}

// ImageResize returns new image that has been resized to given size
// uses sensible default level of smoothing (Linear interpolation)
func ImageResize(img image.Image, szX, szY int) image.Image {
	return transform.Resize(img, szX, szY, transform.Linear)
}

// ImageResizeMax resizes image so that the largest size (X or Y) is set to maxSz
func ImageResizeMax(img image.Image, maxSz int) image.Image {
	sz := img.Bounds().Size()
	tsz := ImageSizeMax(sz, maxSz)
	if tsz != sz {
		img = transform.Resize(img, tsz.X, tsz.Y, transform.Linear)
	}
	return img
}

//////////////////////////////////////////////////////////////////////////////////
//  Props

// TODO: move this to comment directives

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
