// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
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
	Filename FileName    `desc:"file name of image loaded -- set by OpenImage"`
	Size     image.Point `desc:"size of the image"`
	Pixels   *image.RGBA `copy:"-", view:"-", xml:"-" json:"-" desc:"the bitmap image"`
}

var KiT_Bitmap = kit.Types.AddType(&Bitmap{}, BitmapProps)

// AddNewBitmap adds a new bitmap to given parent node, with given name.
func AddNewBitmap(parent ki.Ki, name string) *Bitmap {
	return parent.AddNewChild(KiT_Bitmap, name).(*Bitmap)
}

func (nb *Bitmap) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Bitmap)
	nb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	nb.Size = fr.Size
	nb.Filename = fr.Filename
}

// Resize resizes bitmap to given size
func (bm *Bitmap) Resize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return
	}
	bm.Size = nwsz // always make sure
	if bm.Pixels.Bounds().Size() == nwsz {
		return
	}
	bm.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
}

// OpenImage opens an image for the bitmap, and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension
func (bm *Bitmap) OpenImage(filename FileName, width, height float32) error {
	img, err := OpenImage(string(filename))
	if err != nil {
		log.Printf("gi.Bitmap.OpenImage -- could not open file: %v, err: %v\n", filename, err)
		return err
	}
	bm.Filename = filename
	bm.SetImage(img, width, height)
	return nil
}

// SetImage sets an image for the bitmap, and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension
func (bm *Bitmap) SetImage(img image.Image, width, height float32) {
	updt := bm.UpdateStart()
	defer bm.UpdateEnd(updt)

	sz := img.Bounds().Size()
	if width <= 0 && height <= 0 {
		bm.Resize(sz)
		draw.Draw(bm.Pixels, bm.Pixels.Bounds(), img, image.ZP, draw.Src)
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
		bm.Resize(tsz)
		m := Scale2D(scx, scy)
		s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
		transformer.Transform(bm.Pixels, s2d, img, img.Bounds(), draw.Over, nil)
	}
}

// GrabRenderFrom grabs the rendered image from given node
func (bm *Bitmap) GrabRenderFrom(nii Node2D) {
	img := GrabRenderFrom(nii)
	if img != nil {
		bm.Pixels = img
		bm.Size = bm.Pixels.Bounds().Size()
	}
}

func (bm *Bitmap) DrawIntoViewport(parVp *Viewport2D) {
	r := image.Rectangle{Max: bm.Size}
	sp := image.ZP
	if bm.Par != nil { // use parents children bbox to determine where we can draw
		pni, _ := KiToNode2D(bm.Par)
		nr := r.Intersect(pni.ChildrenBBox2D())
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			fmt.Printf("aberrant sp: %v\n", sp)
			return
		}
		r = nr
	}
	draw.Draw(parVp.Pixels, r, bm.Pixels, sp, draw.Over)
}

func (bm *Bitmap) Render2D() {
	if bm.PushBounds() {
		bm.DrawIntoViewport(bm.Viewport)
		bm.PopBounds()
	}
}

//////////////////////////////////////////////////////////////////////////////////
//  Image IO

// GrabRenderFrom grabs the rendered image from given node
// if nil, then image could not be grabbed
func GrabRenderFrom(nii Node2D) *image.RGBA {
	ni := nii.AsNode2D()
	nivp := nii.AsViewport2D()
	if nivp != nil && nivp.Pixels != nil {
		sz := nivp.Pixels.Bounds().Size()
		img := image.NewRGBA(image.Rectangle{Max: sz})
		draw.Draw(img, img.Bounds(), nivp.Pixels, image.ZP, draw.Src)
		return img
	}
	nivp = ni.Viewport
	if nivp == nil || nivp.Pixels == nil {
		log.Printf("gi.GrabRenderFrom could not grab from node, viewport or pixels nil: %v\n", ni.PathUnique())
		return nil
	}
	if ni.VpBBox.Empty() {
		return nil // offscreen -- can happen -- no warning -- just check rval
	}
	sz := ni.VpBBox.Size()
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), nivp.Pixels, ni.VpBBox.Min, draw.Src)
	return img
}

// OpenImage opens an image from given path filename -- format is inferred automatically.
func OpenImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	return im, err
}

// SaveImage saves image to file, with format inferred from filename -- JPEG and PNG
// supported by default.
func SaveImage(path string, im image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".png" {
		return png.Encode(file, im)
	} else if ext == ".jpg" || ext == ".jpeg" {
		return jpeg.Encode(file, im, &jpeg.Options{Quality: 90})
	} else {
		return fmt.Errorf("gi.SaveImage: extention: %s not recognized -- only .png and .jpg / jpeg supported", ext)
	}
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

//////////////////////////////////////////////////////////////////////////////////
//  Props

var BitmapProps = ki.Props{
	"background-color": &Prefs.Colors.Background,
	"ToolBar": ki.PropSlice{
		{"OpenImage", ki.Props{
			"desc": "Open an image for this bitmap.  if width and/or height is > 0, then image is rescaled to that dimension, preserving aspect ratio if other one is not set",
			"icon": "file-open",
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
