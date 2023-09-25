// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"log"
	"os"

	"github.com/anthonynsimon/bild/clone"
	"github.com/anthonynsimon/bild/transform"
	"goki.dev/colors"
	"goki.dev/girl/gist"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/svg"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

// bitmap contains various bitmap-related elements, including the Bitmap node
// for showing bitmaps, and image processing utilities

// Bitmap is a Widget that is optimized to render a static bitmap image --
// it expects to be a terminal node and does NOT call rendering etc on its
// children.  It is particularly useful for overlays in drag-n-drop uses --
// can grab the image of another vp and show that
type Bitmap struct {
	WidgetBase

	// file name of image loaded -- set by OpenImage
	Filename FileName `desc:"file name of image loaded -- set by OpenImage"`

	// size of the image
	Size image.Point `desc:"size of the image"`

	// [view: -] the bitmap image
	Pixels *image.RGBA `copy:"-" view:"-" xml:"-" json:"-" desc:"the bitmap image"`
}

// event functions for this type
var BitmapEventFuncs WidgetEvents

func (bm *Bitmap) OnInit() {
	bm.AddEvents(&BitmapEventFuncs)
	bm.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.MinWidth.SetPx(float32(bm.Size.X))
		s.MinHeight.SetPx(float32(bm.Size.Y))
		s.BackgroundColor.SetSolid(ColorScheme.Background)
	})
}

func (bm *Bitmap) CopyFieldsFrom(frm any) {
	fr := frm.(*Bitmap)
	bm.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	bm.Size = fr.Size
	bm.Filename = fr.Filename
}

// SetSize sets size of the bitmap image.
// This does not resize any existing image, just makes a new image
// if the size is different
func (bm *Bitmap) SetSize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return
	}
	bm.Size = nwsz // always make sure
	if bm.Pixels != nil && bm.Pixels.Bounds().Size() == nwsz {
		return
	}
	bm.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
}

// OpenImage opens an image for the bitmap, and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension
func (bm *Bitmap) OpenImage(filename FileName, width, height float32) error {
	img, err := svg.OpenImage(string(filename))
	if err != nil {
		log.Printf("gi.Bitmap.OpenImage -- could not open file: %v, err: %v\n", filename, err)
		return err
	}
	bm.Filename = filename
	bm.SetImage(img, width, height)
	return nil
}

// OpenImageFS opens an image for the bitmap, and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension
func (bm *Bitmap) OpenImageFS(fsys fs.FS, filename FileName, width, height float32) error {
	img, err := OpenImageFS(fsys, string(filename))
	if err != nil {
		log.Printf("gi.Bitmap.OpenImage -- could not open file: %v, err: %v\n", filename, err)
		return err
	}
	bm.Filename = filename
	bm.SetImage(img, width, height)
	return nil
}

// SetImage sets an image for the bitmap , and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension.  Copies from given image into internal image for this bitmap.
func (bm *Bitmap) SetImage(img image.Image, width, height float32) {
	updt := bm.UpdateStart()
	defer bm.UpdateEnd(updt)

	sz := img.Bounds().Size()
	if width <= 0 && height <= 0 {
		bm.SetSize(sz)
		draw.Draw(bm.Pixels, bm.Pixels.Bounds(), img, image.Point{}, draw.Src)
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
		bm.SetSize(tsz)
		m := mat32.Scale2D(scx, scy)
		s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
		transformer.Transform(bm.Pixels, s2d, img, img.Bounds(), draw.Src, nil)
	}
}

// GrabRenderFrom grabs the rendered image from given node
func (bm *Bitmap) GrabRenderFrom(wi Widget) {
	img := GrabRenderFrom(wi)
	if img != nil {
		bm.Pixels = img
		bm.Size = bm.Pixels.Bounds().Size()
	}
}

func (bm *Bitmap) DrawIntoScene(sc *Scene) {
	if bm.Pixels == nil {
		return
	}
	pos := bm.LayState.Alloc.Pos.ToPointCeil()
	max := pos.Add(bm.Size)
	r := image.Rectangle{Min: pos, Max: max}
	sp := image.Point{}
	if bm.Par != nil { // use parents children bbox to determine where we can draw
		pni, _ := AsWidget(bm.Par)
		pbb := pni.ChildrenBBoxes(sc)
		nr := r.Intersect(pbb)
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			fmt.Printf("aberrant sp: %v\n", sp)
			return
		}
		r = nr
	}
	draw.Draw(sc.Pixels, r, bm.Pixels, sp, draw.Over)
}

func (bm *Bitmap) Render(sc *Scene) {
	wi := bm.This().(Widget)
	if bm.PushBounds(sc) {
		wi.FilterEvents()
		bm.RenderChildren(sc)
		bm.DrawIntoScene(bm.Sc)
		bm.PopBounds(sc)
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
	if wb.ScBBox.Empty() {
		return nil // offscreen -- can happen -- no warning -- just check rval
	}
	sz := wb.ScBBox.Size()
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), sc.Pixels, wb.ScBBox.Min, draw.Src)
	return img
}

// OpenImageFS opens an image from given path filename -- format is inferred automatically.
func OpenImageFS(fsys fs.FS, fname string) (image.Image, error) {
	file, err := fsys.Open(fname)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	return im, err
}

// ImageToRGBA returns given image as an image.RGBA (no conversion if it is already)
func ImageToRGBA(img image.Image) *image.RGBA {
	if rg, ok := img.(*image.RGBA); ok {
		return rg
	}
	return clone.AsRGBA(img)
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

var BitmapProps = ki.Props{
	"ToolBar": ki.PropSlice{
		{"OpenImage", ki.Props{
			"desc": "Open an image for this bitmap.  if width and/or height is > 0, then image is rescaled to that dimension, preserving aspect ratio if other one is not set",
			"icon": icons.FileOpen,
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
