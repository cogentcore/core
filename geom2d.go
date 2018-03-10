// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"golang.org/x/image/math/fixed"
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

type Size2D Point2D

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

type XFormMatrix2D struct {
	XX, YX, XY, YY, X0, Y0 float64
}

// todo: make these methods on the XForm to scope them

func Identity() XFormMatrix2D {
	return XFormMatrix2D{
		1, 0,
		0, 1,
		0, 0,
	}
}

func Translate(x, y float64) XFormMatrix2D {
	return XFormMatrix2D{
		1, 0,
		0, 1,
		x, y,
	}
}

func Scale(x, y float64) XFormMatrix2D {
	return XFormMatrix2D{
		x, 0,
		0, y,
		0, 0,
	}
}

func Rotate(angle float64) XFormMatrix2D {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return XFormMatrix2D{
		c, s,
		-s, c,
		0, 0,
	}
}

func Shear(x, y float64) XFormMatrix2D {
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

func (a XFormMatrix2D) Translate(x, y float64) XFormMatrix2D {
	return Translate(x, y).Multiply(a)
}

func (a XFormMatrix2D) Scale(x, y float64) XFormMatrix2D {
	return Scale(x, y).Multiply(a)
}

func (a XFormMatrix2D) Rotate(angle float64) XFormMatrix2D {
	return Rotate(angle).Multiply(a)
}

func (a XFormMatrix2D) Shear(x, y float64) XFormMatrix2D {
	return Shear(x, y).Multiply(a)
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

// contrary to some docs, apparently need to run go generate manually
//go:generate stringer -type=ViewBoxMeetOrSlice

// ViewBoxPreserveAspectRatio determines how to scale the view box within parent Viewport2D
type ViewBoxPreserveAspectRatio struct {
	Align       ViewBoxAlign       `svg:"align",desc:"how to align x,y coordinates within viewbox"`
	MeetOrSlice ViewBoxMeetOrSlice `svg:"meetOrSlice",desc:"how to scale the view box relative to the viewport"`
}

// ViewBox defines a region in 2D space
type ViewBox2D struct {
	Min                 Point2D                    `svg:"{min-x,min-y}",desc:"offset or starting point in parent Viewport2D"`
	Size                Size2D                     `svg:"{width,height}",desc:"size of viewbox within parent Viewport2D"`
	PreserveAspectRatio ViewBoxPreserveAspectRatio `svg:"preserveAspectRatio",desc:"how to scale the view box within parent Viewport2D"`
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
