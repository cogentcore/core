// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"errors"
	"image"
	"log"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/math32"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

// Image is an SVG image (bitmap)
type Image struct {
	NodeBase

	// position of the top-left of the image
	Pos math32.Vector2 `xml:"{x,y}"`

	// rendered size of the image (imposes a scaling on image when it is rendered)
	Size math32.Vector2 `xml:"{width,height}"`

	// file name of image loaded -- set by OpenImage
	Filename string

	// how to scale and align the image
	ViewBox ViewBox `xml:"viewbox"`

	// Pixels are the image pixels, which has imagex.WrapJS already applied.
	Pixels image.Image `xml:"-" json:"-" display:"-"`
}

func (g *Image) SVGName() string { return "image" }

func (g *Image) SetNodePos(pos math32.Vector2) {
	g.Pos = pos
}

func (g *Image) SetNodeSize(sz math32.Vector2) {
	g.Size = sz
}

// pixelsOfSize returns the Pixels as an imagex.Image of given size.
// makes a new one if not already the correct size.
func (g *Image) pixelsOfSize(nwsz image.Point) image.Image {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return nil
	}
	if g.Pixels != nil && g.Pixels.Bounds().Size() == nwsz {
		return g.Pixels
	}
	g.Pixels = imagex.WrapJS(image.NewRGBA(image.Rectangle{Max: nwsz}))
	return g.Pixels
}

// SetImage sets an image for the bitmap, and resizes to the size of the image
// or the specified size. Pass 0 for width and/or height to use the actual image size
// for that dimension. Copies from given image into internal image for this bitmap.
func (g *Image) SetImage(img image.Image, width, height float32) {
	if img == nil {
		return
	}
	img = imagex.Unwrap(img)
	sz := img.Bounds().Size()
	if width <= 0 && height <= 0 {
		cp := imagex.CloneAsRGBA(img)
		g.Pixels = imagex.WrapJS(cp)
		if g.Size.X == 0 && g.Size.Y == 0 {
			g.Size = math32.FromPoint(sz)
		}
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
		pxi := g.pixelsOfSize(tsz)
		px := imagex.Unwrap(pxi).(*image.RGBA)
		m := math32.Scale2D(scx, scy)
		s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
		transformer.Transform(px, s2d, img, img.Bounds(), draw.Over, nil)
		if g.Size.X == 0 && g.Size.Y == 0 {
			g.Size = math32.FromPoint(tsz)
		}
	}
}

func (g *Image) DrawImage(sv *SVG) {
	if g.Pixels == nil {
		return
	}
	pc := g.Painter(sv)
	pc.DrawImageScaled(g.Pixels, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
}

func (g *Image) LocalBBox(sv *SVG) math32.Box2 {
	bb := math32.Box2{}
	bb.Min = g.Pos
	bb.Max = g.Pos.Add(g.Size)
	return bb.Canon()
}

func (g *Image) Render(sv *SVG) {
	vis := g.IsVisible(sv)
	if !vis {
		return
	}
	g.DrawImage(sv)
	g.RenderChildren(sv)
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Image) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	rot := xf.ExtractRot()
	if rot != 0 {
		g.Paint.Transform.SetMul(xf)
		g.SetTransformProperty()
	} else {
		g.Pos = xf.MulVector2AsPoint(g.Pos)
		g.Size = xf.MulVector2AsVector(g.Size)
		g.GradientApplyTransform(sv, xf)
	}
}

// OpenImage opens an image for the bitmap, and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension
func (g *Image) OpenImage(filename string, width, height float32) error {
	img, _, err := imagex.Open(filename)
	if err != nil {
		log.Printf("svg.OpenImage -- could not open file: %v, err: %v\n", filename, err)
		return err
	}
	g.Filename = filename
	g.SetImage(img, width, height)
	return nil
}

// SaveImage saves current image to a file
func (g *Image) SaveImage(filename string) error {
	if g.Pixels == nil {
		return errors.New("svg.SaveImage Pixels is nil")
	}
	return imagex.Save(g.Pixels, filename)
}
