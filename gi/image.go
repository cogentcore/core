// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"io/fs"

	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"github.com/anthonynsimon/bild/clone"
	"golang.org/x/image/draw"
)

// Image is a widget that renders a static bitmap image.
// See [styles.ObjectFits] for how to control the image rendering within
// the allocated size. The default minimum requested size is the pixel
// size in [units.Dp] units (1/160th of an inch). See [giv.ConfigImageToolbar]
// for a toolbar with I/O buttons.
type Image struct {
	Box

	// Image is the bitmap image.
	Image *image.RGBA `xml:"-" json:"-" set:"-"`

	// prevPixels is the cached last rendered image.
	prevPixels image.Image `xml:"-" json:"-" set:"-"`

	// prevObjectFit is the cached [styles.Style.ObjectFit] of the last rendered image.
	prevObjectFit styles.ObjectFits `xml:"-" json:"-" set:"-"`

	// prevSize is the cached allocated size for the last rendered image.
	prevSize mat32.Vec2 `xml:"-" json:"-" set:"-"`
}

func (im *Image) OnInit() {
	im.WidgetBase.OnInit()
	im.Style(func(s *styles.Style) {
		if im.Image != nil {
			sz := im.Image.Bounds().Size()
			s.Min.X.Dp(float32(sz.X))
			s.Min.Y.Dp(float32(sz.Y))
		}
	})
}

// Open sets the image to the image located at the given filename.
func (im *Image) Open(filename Filename) error { //gti:add
	img, _, err := images.Open(string(filename))
	if grr.Log(err) != nil {
		return err
	}
	im.SetImage(img)
	return nil
}

// OpenFS sets the image to the image located at the given filename in the given fs.
func (im *Image) OpenFS(fsys fs.FS, filename string) error {
	img, _, err := images.OpenFS(fsys, filename)
	if grr.Log(err) != nil {
		return err
	}
	im.SetImage(img)
	return nil
}

// SetImage sets the image to the given image.
// It copies from the given image into an internal image.
func (im *Image) SetImage(img image.Image) *Image {
	im.Image = clone.AsRGBA(img)
	im.prevPixels = nil
	im.NeedsRender()
	return im
}

func (im *Image) Render() {
	im.Box.Render()

	if im.Image == nil {
		return
	}
	r := im.Geom.ContentBBox
	sp := im.Geom.ScrollOffset()

	var rimg image.Image
	if im.prevPixels != nil && im.Styles.ObjectFit == im.prevObjectFit && im.Geom.Size.Actual.Content == im.prevSize {
		rimg = im.prevPixels
	} else {
		rimg = im.Styles.ResizeImage(im.Image, im.Geom.Size.Actual.Content)
		im.prevPixels = rimg
		im.prevObjectFit = im.Styles.ObjectFit
		im.prevSize = im.Geom.Size.Actual.Content
	}
	draw.Draw(im.Scene.Pixels, r, rimg, sp, draw.Over)
}
