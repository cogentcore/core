// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"

	"github.com/golang/freetype/raster"
)

/*
This is modified and extended from: https://github.com/fogleman/gg

Copyright (C) 2016 Michael Fogleman

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// PaintServers provide patterned colors for stroke and fill operations
type PaintServerType int

const (
	PaintSolidcolor PaintServerType = iota
	PaintLinearGradient
	PaintRadialGradient
	PaintMeshGradient
	PaintPattern
	PaintHatch
	PaintHatchpath
	PaintImage // apparently not SVG-standard but we have it.
)

// contrary to some docs, apparently need to run go generate manually
//go:generate stringer -type=PaintServerType

// todo: implement all the other types of paint servers

// todo: this is not SVG standard and needs to be updated
// for pattern paint server, the way in which the pattern repeats
type RepeatOp int

const (
	RepeatBoth RepeatOp = iota
	RepeatX
	RepeatY
	RepeatNone
)

type PaintServer interface {
	ColorAt(x, y int) color.Color
	ServerType() PaintServerType
}

// Solid PaintServer
type SolidcolorPaintServer struct {
	Color color.Color
}

func (p *SolidcolorPaintServer) ColorAt(x, y int) color.Color {
	return p.Color
}

func (p *SolidcolorPaintServer) ServerType() PaintServerType {
	return PaintSolidcolor
}

func NewSolidcolorPaintServer(color color.Color) PaintServer {
	return &SolidcolorPaintServer{Color: color}
}

// Image PaintServer
type ImagePaintServer struct {
	Image image.Image
	Op    RepeatOp
}

func (p *ImagePaintServer) ColorAt(x, y int) color.Color {
	b := p.Image.Bounds()
	switch p.Op {
	case RepeatX:
		if y >= b.Dy() {
			return color.Transparent
		}
	case RepeatY:
		if x >= b.Dx() {
			return color.Transparent
		}
	case RepeatNone:
		if x >= b.Dx() || y >= b.Dy() {
			return color.Transparent
		}
	}
	x = x%b.Dx() + b.Min.X
	y = y%b.Dy() + b.Min.Y
	return p.Image.At(x, y)
}

func (p *ImagePaintServer) ServerType() PaintServerType {
	return PaintImage
}

func NewImagePaintServer(im image.Image, op RepeatOp) PaintServer {
	return &ImagePaintServer{Image: im, Op: op}
}

type serverPainter struct {
	im   *image.RGBA
	mask *image.Alpha
	p    PaintServer
}

// Paint satisfies the Painter interface.
func (r *serverPainter) Paint(ss []raster.Span, done bool) {
	b := r.im.Bounds()
	for _, s := range ss {
		if s.Y < b.Min.Y {
			continue
		}
		if s.Y >= b.Max.Y {
			return
		}
		if s.X0 < b.Min.X {
			s.X0 = b.Min.X
		}
		if s.X1 > b.Max.X {
			s.X1 = b.Max.X
		}
		if s.X0 >= s.X1 {
			continue
		}
		const m = 1<<16 - 1
		y := s.Y - r.im.Rect.Min.Y
		x0 := s.X0 - r.im.Rect.Min.X
		// RGBAPainter.Paint() in $GOPATH/src/github.com/golang/freetype/raster/paint.go
		i0 := (s.Y-r.im.Rect.Min.Y)*r.im.Stride + (s.X0-r.im.Rect.Min.X)*4
		i1 := i0 + (s.X1-s.X0)*4
		for i, x := i0, x0; i < i1; i, x = i+4, x+1 {
			ma := s.Alpha
			if r.mask != nil {
				ma = ma * uint32(r.mask.AlphaAt(x, y).A) / 255
				if ma == 0 {
					continue
				}
			}
			c := r.p.ColorAt(x, y)
			cr, cg, cb, ca := c.RGBA()
			dr := uint32(r.im.Pix[i+0])
			dg := uint32(r.im.Pix[i+1])
			db := uint32(r.im.Pix[i+2])
			da := uint32(r.im.Pix[i+3])
			a := (m - (ca * ma / m)) * 0x101
			r.im.Pix[i+0] = uint8((dr*a + cr*ma) / m >> 8)
			r.im.Pix[i+1] = uint8((dg*a + cg*ma) / m >> 8)
			r.im.Pix[i+2] = uint8((db*a + cb*ma) / m >> 8)
			r.im.Pix[i+3] = uint8((da*a + ca*ma) / m >> 8)
		}
	}
}

func newPaintServerPainter(im *image.RGBA, mask *image.Alpha, p PaintServer) *serverPainter {
	return &serverPainter{im, mask, p}
}
