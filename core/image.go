// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"io/fs"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"golang.org/x/image/draw"
)

// Image is a widget that renders an [image.Image].
// See [styles.Style.ObjectFit] to control the image rendering within
// the allocated size. The default minimum requested size is the pixel
// size in [units.Dp] units (1/160th of an inch).
type Image struct {
	WidgetBase

	// Image is the [image.Image].
	Image image.Image `xml:"-" json:"-"`

	// prevImage is the cached last [Image.Image].
	prevImage image.Image

	// prevRenderImage is the cached last rendered image with any transformations applied.
	prevRenderImage image.Image

	// prevObjectFit is the cached [styles.Style.ObjectFit] of the last rendered image.
	prevObjectFit styles.ObjectFits

	// prevSize is the cached allocated size for the last rendered image.
	prevSize math32.Vector2
}

func (im *Image) WidgetValue() any { return &im.Image }

func (im *Image) Init() {
	im.WidgetBase.Init()
	im.Styler(func(s *styles.Style) {
		s.ObjectFit = styles.FitContain
		if im.Image != nil {
			sz := im.Image.Bounds().Size()
			s.Min.X.SetCustom(func(uc *units.Context) float32 {
				return min(uc.Dp(float32(sz.X)), uc.Pw(95))
			})
			s.Min.Y.Dp(float32(sz.Y))
		}
	})
}

// Open sets the image to the image located at the given filename.
func (im *Image) Open(filename Filename) error { //types:add
	img, _, err := imagex.Open(string(filename))
	if err != nil {
		return err
	}
	im.SetImage(img)
	return nil
}

// OpenFS sets the image to the image located at the given filename in the given fs.
func (im *Image) OpenFS(fsys fs.FS, filename string) error {
	img, _, err := imagex.OpenFS(fsys, filename)
	if err != nil {
		return err
	}
	im.SetImage(img)
	return nil
}

func (im *Image) SizeUp() {
	im.WidgetBase.SizeUp()
	if im.Image != nil {
		sz := &im.Geom.Size
		obj := math32.FromPoint(im.Image.Bounds().Size())
		osz := styles.ObjectSizeFromFit(im.Styles.ObjectFit, obj, sz.Actual.Content)
		sz.Actual.Content = osz
		sz.setTotalFromContent(&sz.Actual)
	}
}

func (im *Image) Render() {
	im.WidgetBase.Render()

	if im.Image == nil {
		return
	}
	r := im.Geom.ContentBBox
	if r == (image.Rectangle{}) || im.Image.Bounds().Size() == (image.Point{}) {
		return
	}
	sp := im.Geom.ScrollOffset()

	var rimg image.Image
	if im.prevImage == im.Image && im.Styles.ObjectFit == im.prevObjectFit && im.Geom.Size.Actual.Content == im.prevSize {
		rimg = im.prevRenderImage
	} else {
		im.prevImage = im.Image
		im.prevObjectFit = im.Styles.ObjectFit
		im.prevSize = im.Geom.Size.Actual.Content
		rimg = im.Styles.ResizeImage(im.Image, im.Geom.Size.Actual.Content)
		im.prevRenderImage = rimg
	}
	im.Scene.Painter.DrawImage(rimg, r, sp, draw.Over)
}

func (im *Image) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(im.Open).SetIcon(icons.Open)
	})
}
