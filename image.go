// Copyright (c) 2021, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

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

	"goki.dev/ki/v2/ki"
	"goki.dev/mat32/v2"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

// Image is an SVG image (bitmap)
type Image struct {
	NodeBase

	// position of the top-left of the image
	Pos mat32.Vec2 `xml:"{x,y}" desc:"position of the top-left of the image"`

	// rendered size of the image (imposes a scaling on image when it is rendered)
	Size mat32.Vec2 `xml:"{width,height}" desc:"rendered size of the image (imposes a scaling on image when it is rendered)"`

	// directs resize operations to preserve aspect ratio
	PreserveAspectRatio bool `xml:"preserveAspectRatio" desc:"directs resize operations to preserve aspect ratio"`

	// file name of image loaded -- set by OpenImage
	Filename string `desc:"file name of image loaded -- set by OpenImage"`

	// [view: -] the image pixels
	Pixels *image.RGBA `copy:"-" xml:"-" json:"-" view:"-" desc:"the image pixels"`
}

// AddNewImage adds a new image to given parent node, with given name and pos
func AddNewImage(parent ki.Ki, name string, x, y float32) *Image {
	g := parent.AddNewChild(ImageType, name).(*Image)
	g.Pos.Set(x, y)
	return g
}

func (g *Image) SVGName() string { return "image" }

func (g *Image) CopyFieldsFrom(frm any) {
	fr := frm.(*Image)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Size = fr.Size
	g.PreserveAspectRatio = fr.PreserveAspectRatio
	g.Filename = fr.Filename
	g.Pixels = fr.Pixels
}

func (g *Image) SetPos(pos mat32.Vec2) {
	g.Pos = pos
}

func (g *Image) SetSize(sz mat32.Vec2) {
	g.Size = sz
}

// SetImageSize sets size of the bitmap image.
// This does not resize any existing image, just makes a new image
// if the size is different
func (g *Image) SetImageSize(nwsz image.Point) {
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
func (g *Image) OpenImage(filename string, width, height float32) error {
	img, err := OpenImage(filename)
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
	return SaveImage(filename, g.Pixels)
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
		return fmt.Errorf("svg.SaveImage: extension: %s not recognized -- only .png and .jpg / jpeg supported", ext)
	}
}

// SetImage sets an image for the bitmap , and resizes to the size of the image
// or the specified size -- pass 0 for width and/or height to use the actual image size
// for that dimension.  Copies from given image into internal image for this bitmap.
func (g *Image) SetImage(img image.Image, width, height float32) {
	sz := img.Bounds().Size()
	if width <= 0 && height <= 0 {
		g.SetImageSize(sz)
		draw.Draw(g.Pixels, g.Pixels.Bounds(), img, image.Point{}, draw.Src)
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
		g.SetImageSize(tsz)
		m := mat32.Scale2D(scx, scy)
		s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
		transformer.Transform(g.Pixels, s2d, img, img.Bounds(), draw.Over, nil)
		if g.Size.X == 0 && g.Size.Y == 0 {
			g.Size = mat32.NewVec2FmPoint(tsz)
		}
	}
}

func (g *Image) DrawImage(sv *SVG) {
	if g.Pixels == nil {
		return
	}

	rs := &sv.RenderState
	pc := &g.Pnt
	pc.DrawImageScaled(rs, g.Pixels, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
}

func (g *Image) NodeBBox(sv *SVG) image.Rectangle {
	rs := &sv.RenderState
	pos := rs.XForm.MulVec2AsPt(g.Pos)
	max := rs.XForm.MulVec2AsPt(g.Pos.Add(g.Size))
	posi := pos.ToPointCeil()
	maxi := max.ToPointCeil()
	return image.Rectangle{posi, maxi}.Canon()
}

func (g *Image) LocalBBox() mat32.Box2 {
	bb := mat32.Box2{}
	bb.Min = g.Pos
	bb.Max = g.Pos.Add(g.Size)
	return bb
}

func (g *Image) Render(sv *SVG) {
	vis, rs := g.PushXForm(sv)
	if !vis {
		return
	}
	rs.Lock()
	g.DrawImage(sv)
	rs.Unlock()
	g.BBoxes(sv)
	g.RenderChildren(sv)
	rs.PopXFormLock()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Image) ApplyXForm(xf mat32.Mat2) {
	rot := xf.ExtractRot()
	if rot != 0 || !g.Pnt.XForm.IsIdentity() {
		g.Pnt.XForm = g.Pnt.XForm.Mul(xf)
		g.SetProp("transform", g.Pnt.XForm.String())
	} else {
		g.Pos = xf.MulVec2AsPt(g.Pos)
		g.Size = xf.MulVec2AsVec(g.Size)
	}
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Image) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	crot := g.Pnt.XForm.ExtractRot()
	if rot != 0 || crot != 0 {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, false) // exclude self
		mat := g.Pnt.XForm.MulCtr(xf, lpt)
		g.Pnt.XForm = mat
		g.SetProp("transform", g.Pnt.XForm.String())
	} else {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, true) // include self
		g.Pos = xf.MulVec2AsPtCtr(g.Pos, lpt)
		g.Size = xf.MulVec2AsVec(g.Size)
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Image) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 4+6)
	(*dat)[0] = g.Pos.X
	(*dat)[1] = g.Pos.Y
	(*dat)[2] = g.Size.X
	(*dat)[3] = g.Size.Y
	g.WriteXForm(*dat, 4)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Image) ReadGeom(dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Size.X = dat[2]
	g.Size.Y = dat[3]
	g.ReadXForm(dat, 4)
}

/*
// ImageProps define the ToolBar for images
var ImageProps = ki.Props{
	"ToolBar": ki.PropSlice{
		{"OpenImage", ki.Props{
			"desc": "Open image file for this image node, rescaling to given size -- use 0, 0 to use native image size.",
			"icon": icons.FileOpen,
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".png,.jpg,.jpeg",
				}},
				{"Width", ki.Props{}},
				{"Height", ki.Props{}},
			},
		}},
		{"SaveImage", ki.Props{
			"desc": "Save image to a file.",
			"icon": icons.SaveAs,
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".png,.jpg,.jpeg",
				}},
			},
		}},
	},
}
*/

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
