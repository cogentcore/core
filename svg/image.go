// Copyright (c) 2021, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

// Image is an SVG image (bitmap)
type Image struct {
	NodeBase
	Pos                 mat32.Vec2  `xml:"{x,y}" desc:"position of the top-left of the image"`
	Size                mat32.Vec2  `xml:"{width,height}" desc:"rendered size of the image (imposes a scaling on image when it is rendered)"`
	PreserveAspectRatio bool        `xml:"preserveAspectRatio" desc:"directs resize operations to preserve aspect ratio"`
	Filename            gi.FileName `desc:"file name of image loaded -- set by OpenImage"`
	Pixels              *image.RGBA `copy:"-" xml:"-" json:"-" view:"-" desc:"the image pixels"`
}

var KiT_Image = kit.Types.AddType(&Image{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewImage adds a new image to given parent node, with given name and pos
func AddNewImage(parent ki.Ki, name string, x, y float32) *Image {
	g := parent.AddNewChild(KiT_Image, name).(*Image)
	g.Pos.Set(x, y)
	return g
}

func (g *Image) SVGName() string { return "image" }

func (g *Image) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Image)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Size = fr.Size
	g.PreserveAspectRatio = fr.PreserveAspectRatio
	g.Filename = fr.Filename
	g.Pixels = fr.Pixels
}

// SetSize sets size of the bitmap image.
// This does not resize any existing image, just makes a new image
// if the size is different
func (g *Image) SetSize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return
	}
	if g.Pixels != nil && g.Pixels.Bounds().Size() == nwsz {
		return
	}
	g.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
}

// OpenImage opens an image for the bitmap, and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension
func (g *Image) OpenImage(filename gi.FileName, width, height float32) error {
	img, err := gi.OpenImage(string(filename))
	if err != nil {
		log.Printf("gi.Bitmap.OpenImage -- could not open file: %v, err: %v\n", filename, err)
		return err
	}
	g.Filename = filename
	g.SetImage(img, width, height)
	return nil
}

// SetImage sets an image for the bitmap , and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension.  Copies from given image into internal image for this bitmap.
func (g *Image) SetImage(img image.Image, width, height float32) {
	sz := img.Bounds().Size()
	if width <= 0 && height <= 0 {
		g.SetSize(sz)
		draw.Draw(g.Pixels, g.Pixels.Bounds(), img, image.ZP, draw.Src)
		if g.Size.X == 0 && g.Size.Y == 0 {
			g.Size = mat32.NewVec2FmPoint(sz)
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
		g.SetSize(tsz)
		m := mat32.Scale2D(scx, scy)
		s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
		transformer.Transform(g.Pixels, s2d, img, img.Bounds(), draw.Over, nil)
		if g.Size.X == 0 && g.Size.Y == 0 {
			g.Size = mat32.NewVec2FmPoint(tsz)
		}
	}
}

func (g *Image) DrawImage() {
	if g.Pixels == nil {
		return
	}

	rs := g.Render()
	pc := &g.Pnt
	pc.DrawImageScaled(rs, g.Pixels, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
}

func (g *Image) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	pos := rs.XForm.MulVec2AsPt(g.Pos)
	max := rs.XForm.MulVec2AsPt(g.Pos.Add(g.Size))
	posi := pos.ToPointCeil()
	maxi := max.ToPointCeil()
	return image.Rectangle{posi, maxi}
}

func (g *Image) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := g.Render()
	rs.PushXForm(pc.XForm)
	g.DrawImage()
	g.ComputeBBoxSVG()
	g.Render2DChildren()
	rs.PopXForm()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Image) ApplyXForm(xf mat32.Mat2) {
	g.Pos = xf.MulVec2AsPt(g.Pos)
	g.Size = xf.MulVec2AsVec(g.Size)
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Image) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	xf, lpt := g.DeltaXForm(trans, scale, rot, pt)
	g.Pos = xf.MulVec2AsPtCtr(g.Pos, lpt)
	g.Size = xf.MulVec2AsVec(g.Size)
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Image) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 4)
	(*dat)[0] = g.Pos.X
	(*dat)[1] = g.Pos.Y
	(*dat)[2] = g.Size.X
	(*dat)[3] = g.Size.Y
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Image) ReadGeom(dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Size.X = dat[2]
	g.Size.Y = dat[3]
}
