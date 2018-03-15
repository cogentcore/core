// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/rcoreilly/goki/ki"
	"golang.org/x/image/math/fixed"
	"image"
	"math"
)

// SVG default coordinates are such that 0,0 is upper-left!

/*
This is essentially verbatim from: https://github.com/fogleman/gg

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

type Point2D struct {
	X, Y float64
}

var Point2DZero = Point2D{0, 0}

type Size2D Point2D

var Size2DZero = Size2D{0, 0}

func (a Point2D) IsZero() bool {
	return a.X == 0.0 && a.Y == 0.0
}

func (a Size2D) IsZero() bool {
	return a.X == 0.0 && a.Y == 0.0
}

func (a Point2D) Fixed() fixed.Point26_6 {
	return fixp(a.X, a.Y)
}

func (a Point2D) Distance(b Point2D) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

func (a Point2D) Interpolate(b Point2D, t float64) Point2D {
	x := a.X + (b.X-a.X)*t
	y := a.Y + (b.Y-a.Y)*t
	return Point2D{x, y}
}

func (a Point2D) Add(b Point2D) Point2D {
	return Point2D{a.X + b.X, a.Y + b.Y}
}

func (a Point2D) AddVal(val float64) Point2D {
	return Point2D{a.X + val, a.Y + val}
}

func (a Point2D) Max(b Point2D) Point2D {
	return Point2D{ki.Max64(a.X, b.X), ki.Max64(a.Y, b.Y)}
}

func (a Point2D) Min(b Point2D) Point2D {
	return Point2D{ki.Min64(a.X, b.X), ki.Min64(a.Y, b.Y)}
}

func (a Size2D) Add(b Size2D) Size2D {
	return Size2D{a.X + b.X, a.Y + b.Y}
}

func (a Size2D) AddVal(val float64) Size2D {
	return Size2D{a.X + val, a.Y + val}
}

func (a Size2D) Max(b Size2D) Size2D {
	return Size2D{ki.Max64(a.X, b.X), ki.Max64(a.Y, b.Y)}
}

func (a Size2D) Min(b Size2D) Size2D {
	return Size2D{ki.Min64(a.X, b.X), ki.Min64(a.Y, b.Y)}
}

func (a *Size2D) SetFromPoint(pt image.Point) {
	a.X = float64(pt.X)
	a.Y = float64(pt.Y)
}

type XFormMatrix2D struct {
	XX, YX, XY, YY, X0, Y0 float64
}

func Identity2D() XFormMatrix2D {
	return XFormMatrix2D{
		1, 0,
		0, 1,
		0, 0,
	}
}

func Translate2D(x, y float64) XFormMatrix2D {
	return XFormMatrix2D{
		1, 0,
		0, 1,
		x, y,
	}
}

func Scale2D(x, y float64) XFormMatrix2D {
	return XFormMatrix2D{
		x, 0,
		0, y,
		0, 0,
	}
}

func Rotate2D(angle float64) XFormMatrix2D {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return XFormMatrix2D{
		c, s,
		-s, c,
		0, 0,
	}
}

func Shear2D(x, y float64) XFormMatrix2D {
	return XFormMatrix2D{
		1, y,
		x, 1,
		0, 0,
	}
}

func (a XFormMatrix2D) Multiply(b XFormMatrix2D) XFormMatrix2D {
	return XFormMatrix2D{
		a.XX*b.XX + a.YX*b.XY,
		a.XX*b.YX + a.YX*b.YY,
		a.XY*b.XX + a.YY*b.XY,
		a.XY*b.YX + a.YY*b.YY,
		a.X0*b.XX + a.Y0*b.XY + b.X0,
		a.X0*b.YX + a.Y0*b.YY + b.Y0,
	}
}

func (a XFormMatrix2D) TransformVector(x, y float64) (tx, ty float64) {
	tx = a.XX*x + a.XY*y
	ty = a.YX*x + a.YY*y
	return
}

func (a XFormMatrix2D) TransformPoint(x, y float64) (tx, ty float64) {
	tx = a.XX*x + a.XY*y + a.X0
	ty = a.YX*x + a.YY*y + a.Y0
	return
}

func (a XFormMatrix2D) TransformPointToInt(x, y float64) (tx, ty int) {
	tx = int(a.XX*x + a.XY*y + a.X0)
	ty = int(a.YX*x + a.YY*y + a.Y0)
	return
}

func (a XFormMatrix2D) Translate(x, y float64) XFormMatrix2D {
	return Translate2D(x, y).Multiply(a)
}

func (a XFormMatrix2D) Scale(x, y float64) XFormMatrix2D {
	return Scale2D(x, y).Multiply(a)
}

func (a XFormMatrix2D) Rotate(angle float64) XFormMatrix2D {
	return Rotate2D(angle).Multiply(a)
}

func (a XFormMatrix2D) Shear(x, y float64) XFormMatrix2D {
	return Shear2D(x, y).Multiply(a)
}

// ViewBoxAlign defines values for the PreserveAspectRatio alignment factor
type ViewBoxAlign int32

const (
	None  ViewBoxAlign = 0                  // do not preserve uniform scaling
	XMin  ViewBoxAlign = 1 << iota          // align ViewBox.Min with smallest values of Viewport
	XMid  ViewBoxAlign = 1 << iota          // align ViewBox.Min with midpoint values of Viewport
	XMax  ViewBoxAlign = 1 << iota          // align ViewBox.Min+Size with maximum values of Viewport
	XMask ViewBoxAlign = XMin + XMid + XMax // mask for X values -- clear all X before setting new one
	YMin  ViewBoxAlign = 1 << iota          // align ViewBox.Min with smallest values of Viewport
	YMid  ViewBoxAlign = 1 << iota          // align ViewBox.Min with midpoint values of Viewport
	YMax  ViewBoxAlign = 1 << iota          // align ViewBox.Min+Size with maximum values of Viewport
	YMask ViewBoxAlign = YMin + YMid + YMax // mask for Y values -- clear all Y before setting new one
)

// ViewBoxMeetOrSlice defines values for the PreserveAspectRatio meet or slice factor
type ViewBoxMeetOrSlice int32

const (
	Meet  ViewBoxMeetOrSlice = iota // the entire ViewBox is visible within Viewport, and it is scaled up as much as possible to meet the align constraints
	Slice ViewBoxMeetOrSlice = iota // the entire ViewBox is covered by the ViewBox, and the ViewBox is scaled down as much as possible, while still meeting the align constraints
)

//go:generate stringer -type=ViewBoxMeetOrSlice

// ViewBoxPreserveAspectRatio determines how to scale the view box within parent Viewport2D
type ViewBoxPreserveAspectRatio struct {
	Align       ViewBoxAlign       `svg:"align",desc:"how to align x,y coordinates within viewbox"`
	MeetOrSlice ViewBoxMeetOrSlice `svg:"meetOrSlice",desc:"how to scale the view box relative to the viewport"`
}

// ViewBox defines a region in 2D bitmap image space -- it must ALWAYS be in terms of underlying pixels
type ViewBox2D struct {
	Min                 image.Point                `svg:"{min-x,min-y}",desc:"offset or starting point in parent Viewport2D"`
	Size                image.Point                `svg:"{width,height}",desc:"size of viewbox within parent Viewport2D"`
	PreserveAspectRatio ViewBoxPreserveAspectRatio `svg:"preserveAspectRatio",desc:"how to scale the view box within parent Viewport2D"`
}

// todo: need to implement the viewbox preserve aspect ratio logic!

// convert viewbox to bounds
func (vb *ViewBox2D) Bounds() image.Rectangle {
	return image.Rect(vb.Min.X, vb.Min.Y, vb.Min.X+vb.Size.X, vb.Min.Y+vb.Size.Y)
}

// convert viewbox to rect version of size
func (vb *ViewBox2D) SizeRect() image.Rectangle {
	return image.Rect(0, 0, vb.Size.X, vb.Size.Y)
}

///////////////////////////////////////////////////////////
// utlities

func Radians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func Degrees(radians float64) float64 {
	return radians * 180 / math.Pi
}

func fixp(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{fix(x), fix(y)}
}

func fix(x float64) fixed.Int26_6 {
	return fixed.Int26_6(x * 64)
}

func unfix(x fixed.Int26_6) float64 {
	const shift, mask = 6, 1<<6 - 1
	if x >= 0 {
		return float64(x>>shift) + float64(x&mask)/64
	}
	x = -x
	if x >= 0 {
		return -(float64(x>>shift) + float64(x&mask)/64)
	}
	return 0
}
