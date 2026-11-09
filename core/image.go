// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image/color"
	stddraw "image/draw"

	"cogentcore.org/core/paint"
	"image"
	"io/fs"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
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

	// NaturalDp is whether the natural size of the image (when it has no
	// Min size) is its size in pixels in dp, as for the CSS px of images
	// in web pages, rather than in dots: on a screen with two dots per dp,
	// it is twice as large, as in browsers (as the photos of the postings
	// of craigslist.org, which were half their size on retina screens).
	NaturalDp bool `set:"-" display:"-"`

	// Density is the number of pixels of the image per dp of its natural
	// size with NaturalDp (as of a candidate of the srcset of an img in a
	// web page, as an image of 2x is shown at half its size in pixels);
	// 0 is 1.
	Density float32 `set:"-" display:"-"`

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
		if dp := im.Styles.UnitContext.Dots(units.UnitDp); im.NaturalDp && dp > 0 {
			obj = obj.MulScalar(dp)
			if im.Density > 0 {
				obj = obj.DivScalar(im.Density)
			}
		}
		csz := sz.Actual.Content
		// if only one min size is specified, then allow the other to expand
		if im.Styles.Min.Y.Value != 0 && im.Styles.Min.X.Value == 0 {
			csz.X = math32.Floor((im.Styles.Min.Y.Dots / obj.Y) * obj.X)
		} else if im.Styles.Min.X.Value != 0 && im.Styles.Min.Y.Value == 0 {
			csz.Y = math32.Floor((im.Styles.Min.X.Dots / obj.X) * obj.Y)
		} else if im.Styles.Min.X.Value == 0 && im.Styles.Min.Y.Value == 0 {
			pwd := float32(0)
			pwb := im.parentWidget()
			if pwb != nil {
				psz := pwb.Geom.Size.Alloc.Content.Sub(pwb.Geom.Size.InnerSpace)
				pwd = 0.95 * psz.X
			}
			if pwd > 0 {
				csz.X = min(obj.X, pwd)
				csz.Y = math32.Floor((csz.X / obj.X) * obj.Y)
			} else {
				csz = obj
			}
		}
		osz := styles.ObjectSizeFromFit(im.Styles.ObjectFit, obj, csz)
		sz.Actual.Content = osz
		sz.SetTotalFromContent(&sz.Actual)
	}
}

// SizeDown sets the height of an image whose width is relative
// to its parent (and whose height is not set) from that width,
// which is only known now, keeping the aspect ratio of the image.
func (im *Image) SizeDown(iter int) bool {
	redo := im.WidgetBase.SizeDown(iter)
	minX, minY := im.Styles.Min.X, im.Styles.Min.Y
	if im.Image == nil || minY.Value != 0 || (minX.Unit != units.UnitPw && minX.Unit != units.UnitPh) {
		return redo
	}
	sz := &im.Geom.Size
	obj := math32.FromPoint(im.Image.Bounds().Size())
	w := sz.Actual.Content.X
	h := math32.Floor((w / obj.X) * obj.Y)
	if w <= 0 || sz.Actual.Content.Y == h {
		return redo
	}
	sz.Actual.Content.Y = h
	sz.SetTotalFromContent(&sz.Actual)
	return true
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

	useLocalResize := false
	if TheApp.Platform() == system.Web && im.Scene != nil {
		// we are using an image renderer, e.g. PDF, offline rendering.
		// note that there is no way to get transformed image back from javascript :(
		// which would be the cleaner solution here.
		if _, ok := im.Scene.Renderer.(*rasterx.Renderer); ok {
			if _, ok := im.prevRenderImage.(*imagex.JSRGBA); ok {
				im.prevImage = nil // clear
			}
			useLocalResize = true
		}
	}

	var rimg image.Image
	if im.prevImage == im.Image && im.Styles.ObjectFit == im.prevObjectFit && im.Geom.Size.Actual.Content == im.prevSize {
		rimg = im.prevRenderImage
	} else {
		im.prevImage = im.Image
		im.prevObjectFit = im.Styles.ObjectFit
		im.prevSize = im.Geom.Size.Actual.Content
		if useLocalResize {
			rimg = im.Styles.ResizeImageLocal(im.Image, im.Geom.Size.Actual.Content)
		} else {
			rimg = im.Styles.ResizeImage(im.Image, im.Geom.Size.Actual.Content)
		}
		// the image is in its rounded corners, as in CSS (as the round
		// avatars of lobste.rs)
		if radius := im.Styles.Border.Radius.Dots(); !sides.AreZero(radius.Sides) {
			rimg = roundCorners(rimg, radius)
		}
		im.prevRenderImage = rimg
	}
	pim := im.Scene.Painter.DrawImage(rimg, im.Image, r, sp, draw.Over)
	if id, ok := im.Properties["id"]; ok {
		pim.Anchor = id.(string)
	}
}

func (im *Image) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(im.Open).SetIcon(icons.Open)
	})
}

// roundCorners returns the given image with its corners rounded by the
// given radii (top left, top right, bottom right, bottom left), outside of
// which it is transparent, with antialiased edges.
func roundCorners(img image.Image, radius sides.Floats) image.Image {
	b := img.Bounds()
	w, h := float32(b.Dx()), float32(b.Dy())
	radius = paint.ClampBorderRadius(radius, w, h)
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	mask := image.NewAlpha(out.Bounds())
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			px, py := float32(x)+0.5, float32(y)+0.5
			// the center of the corner that the pixel is in, if any
			var r, cx, cy float32
			switch {
			case px < radius.Top && py < radius.Top:
				r, cx, cy = radius.Top, radius.Top, radius.Top
			case px > w-radius.Right && py < radius.Right:
				r, cx, cy = radius.Right, w-radius.Right, radius.Right
			case px > w-radius.Bottom && py > h-radius.Bottom:
				r, cx, cy = radius.Bottom, w-radius.Bottom, h-radius.Bottom
			case px < radius.Left && py > h-radius.Left:
				r, cx, cy = radius.Left, radius.Left, h-radius.Left
			}
			a := float32(1)
			if r > 0 {
				d := math32.Hypot(px-cx, py-cy)
				a = math32.Clamp(r-d+0.5, 0, 1)
			}
			mask.SetAlpha(x, y, color.Alpha{uint8(a*255 + 0.5)})
		}
	}
	stddraw.DrawMask(out, out.Bounds(), img, b.Min, mask, image.Point{}, stddraw.Over)
	return out
}
