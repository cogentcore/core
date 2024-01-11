// Copyright (c) 2018, The Goki Authors. All rights reserved.
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
	"goki.dev/goki/colors"
	"goki.dev/goki/events"
	"goki.dev/goki/grows/images"
	"goki.dev/goki/mat32"
	"goki.dev/goki/styles"
	"goki.dev/icons"
	"golang.org/x/image/draw"
)

// Image is a Widget that renders a static bitmap image.
// See [Styles.ObjectFits] for how to control the image rendering within
// the allocated size.  The minimum requested size is the pixel size in
// Dp units (1/160th of an inch).
type Image struct {
	WidgetBase

	// file name of image loaded -- set by OpenImage
	Filename FileName `set:"-"`

	// the bitmap image
	Pixels *image.RGBA `copy:"-" view:"-" xml:"-" json:"-" set:"-"`

	// cached last rendered image
	PrevPixels image.Image `copy:"-" xml:"-" json:"-" set:"-"`

	// cached [styles.Style.ObjectFit] of the last rendered image
	PrevObjectFit styles.ObjectFits `copy:"-" xml:"-" json:"-" set:"-"`

	// cached allocated size for the last rendered image
	PrevSize mat32.Vec2 `copy:"-" xml:"-" json:"-" set:"-"`
}

func (im *Image) CopyFieldsFrom(frm any) {
	fr := frm.(*Image)
	im.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	im.Filename = fr.Filename
}

func (im *Image) OnInit() {
	im.WidgetBase.OnInit()
	im.SetStyles()
}

func (im *Image) SetStyles() {
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
	defer im.UpdateEndRender(updt)

	im.Pixels = clone.AsRGBA(img)
	im.PrevPixels = nil
}

// SetSize is a convenience method to ensure that the image
// is given size.  A new image will be created of the given size
// if the current one is not of the specified size.
func (im *Image) SetSize(sz image.Point) {
	updt := im.UpdateStart()
	defer im.UpdateEndRender(updt)

	if im.Pixels != nil {
		csz := im.Pixels.Bounds().Size()
		if sz == csz {
			return
		}
	}
	im.Pixels = image.NewRGBA(image.Rectangle{Max: sz})
	im.PrevPixels = nil
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
	var rimg image.Image
	if im.PrevPixels != nil && im.Styles.ObjectFit == im.PrevObjectFit && im.Geom.Size.Actual.Content == im.PrevSize {
		rimg = im.PrevPixels
	} else {
		rimg = im.Styles.ResizeImage(im.Pixels, im.Geom.Size.Actual.Content)
		im.PrevPixels = rimg
		im.PrevObjectFit = im.Styles.ObjectFit
		im.PrevSize = im.Geom.Size.Actual.Content
	}
	draw.Draw(im.Sc.Pixels, r, rimg, sp, draw.Over)
}

func (im *Image) Render() {
	if im.PushBounds() {
		im.DrawIntoScene()
		im.RenderChildren()
		im.PopBounds()
	}
}

// ConfigToolbar can be used to configure a toolbar for image
func (im *Image) ConfigToolbar(tb *Toolbar) {
	NewButton(tb).SetText("OpenImage").SetIcon(icons.Open).
		OnClick(func(e events.Event) {
			CallFunc(im, im.OpenImage)
		})
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
