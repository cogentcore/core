// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthonynsimon/bild/clone"
	"github.com/anthonynsimon/bild/transform"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
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
	Pixels   *image.RGBA `copy:"-" view:"-" xml:"-" json:"-" desc:"the bitmap image"`
}

var KiT_Bitmap = kit.Types.AddType(&Bitmap{}, BitmapProps)

// AddNewBitmap adds a new bitmap to given parent node, with given name.
func AddNewBitmap(parent ki.Ki, name string) *Bitmap {
	return parent.AddNewChild(KiT_Bitmap, name).(*Bitmap)
}

func (bm *Bitmap) CopyFieldsFrom(frm interface{}) {
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

// LayoutToImgSize sets the width, height properties to the current Size
// so it will request that size during layout
func (bm *Bitmap) LayoutToImgSize() {
	bm.SetProp("width", units.NewValue(float32(bm.Size.X), units.Dot))
	bm.SetProp("height", units.NewValue(float32(bm.Size.Y), units.Dot))
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

// SetImage sets an image for the bitmap , and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension.  Copies from given image into internal image for this bitmap.
func (bm *Bitmap) SetImage(img image.Image, width, height float32) {
	updt := bm.UpdateStart()
	defer bm.UpdateEnd(updt)

	sz := img.Bounds().Size()
	if width <= 0 && height <= 0 {
		bm.SetSize(sz)
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
		bm.SetSize(tsz)
		m := mat32.Scale2D(scx, scy)
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
	if bm.Pixels == nil {
		return
	}
	pos := bm.LayState.Alloc.Pos.ToPointCeil()
	max := pos.Add(bm.Size)
	r := image.Rectangle{Min: pos, Max: max}
	sp := image.ZP
	if bm.Par != nil { // use parents children bbox to determine where we can draw
		pni, _ := KiToNode2D(bm.Par)
		pbb := pni.ChildrenBBox2D()
		nr := r.Intersect(pbb)
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
	if bm.FullReRenderIfNeeded() {
		return
	}
	if bm.PushBounds() {
		bm.This().(Node2D).ConnectEvents2D()
		bm.DrawIntoViewport(bm.Viewport)
		bm.PopBounds()
	} else {
		bm.DisconnectAllEvents(AllPris)
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
		log.Printf("gi.GrabRenderFrom could not grab from node, viewport or pixels nil: %v\n", ni.Path())
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

// ImageToRGBA returns given image as an image.RGBA (no conversion if it is already)
func ImageToRGBA(img image.Image) *image.RGBA {
	if rg, ok := img.(*image.RGBA); ok {
		return rg
	}
	return clone.AsRGBA(img)
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
		return fmt.Errorf("gi.SaveImage: extension: %s not recognized -- only .png and .jpg / jpeg supported", ext)
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

// ImageToBase64PNG returns bytes of image encoded as a PNG in Base64 format
// with "image/png" mimetype returned
func ImageToBase64PNG(img image.Image) ([]byte, string) {
	ibuf := &bytes.Buffer{}
	png.Encode(ibuf, img)
	ib := ibuf.Bytes()
	eb := make([]byte, base64.StdEncoding.EncodedLen(len(ib)))
	base64.StdEncoding.Encode(eb, ib)
	return eb, "image/png"
}

// ImageToBase64JPG returns bytes image encoded as a JPG in Base64 format
// with "image/jpeg" mimetype returned
func ImageToBase64JPG(img image.Image) ([]byte, string) {
	ibuf := &bytes.Buffer{}
	jpeg.Encode(ibuf, img, &jpeg.Options{Quality: 90})
	ib := ibuf.Bytes()
	eb := make([]byte, base64.StdEncoding.EncodedLen(len(ib)))
	base64.StdEncoding.Encode(eb, ib)
	return eb, "image/jpeg"
}

// Base64SplitLines splits the encoded Base64 bytes into standard lines of 76
// chars each.  The last line also ends in a newline
func Base64SplitLines(b []byte) []byte {
	ll := 76
	sz := len(b)
	nl := (sz / ll)
	rb := make([]byte, sz+nl+1)
	for i := 0; i < nl; i++ {
		st := ll * i
		rst := ll*i + i
		copy(rb[rst:rst+ll], b[st:st+ll])
		rb[rst+ll] = '\n'
	}
	st := ll * nl
	rst := ll*nl + nl
	ln := sz - st
	copy(rb[rst:rst+ln], b[st:st+ln])
	rb[rst+ln] = '\n'
	return rb
}

// ImageFmBase64PNG returns image from Base64-encoded bytes in PNG format
func ImageFmBase64PNG(eb []byte) (image.Image, error) {
	if eb[76] == ' ' {
		eb = bytes.ReplaceAll(eb, []byte(" "), []byte("\n"))
	}
	db := make([]byte, base64.StdEncoding.DecodedLen(len(eb)))
	_, err := base64.StdEncoding.Decode(db, eb)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	rb := bytes.NewReader(db)
	return png.Decode(rb)
}

// ImageFmBase64PNG returns image from Base64-encoded bytes in PNG format
func ImageFmBase64JPG(eb []byte) (image.Image, error) {
	if eb[76] == ' ' {
		eb = bytes.ReplaceAll(eb, []byte(" "), []byte("\n"))
	}
	db := make([]byte, base64.StdEncoding.DecodedLen(len(eb)))
	_, err := base64.StdEncoding.Decode(db, eb)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	rb := bytes.NewReader(db)
	return jpeg.Decode(rb)
}

// ImageFmBase64 returns image from Base64-encoded bytes in either PNG or JPEG format
// based on fmt which must end in either png or jpeg
func ImageFmBase64(fmt string, eb []byte) (image.Image, error) {
	if strings.HasSuffix(fmt, "png") {
		return ImageFmBase64PNG(eb)
	}
	if strings.HasSuffix(fmt, "jpeg") {
		return ImageFmBase64JPG(eb)
	}
	return nil, errors.New("image format must be either png or jpeg")
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
			f32 := gist.NRGBAf32Model.Convert(im.At(x, y)).(gist.NRGBAf32)
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
	"EnumType:Flag":    KiT_NodeFlags,
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
