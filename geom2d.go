// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/rcoreilly/goki/ki/kit"
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

// note: golang.org/x/image/math/f64 defines Vec2 as [2]float64
// elabored then by https://godoc.org/github.com/go-gl/mathgl/mgl64
// it is instead very convenient and clear to use .X .Y fields for 2D math
// original gg package used Point2D but Vec2D is more general, e.g., for sizes etc
// in go much better to use fewer types so only using Vec2D

// dimensions
type Dims2D int32

const (
	X Dims2D = iota
	Y
)

var KiT_Dims2D = kit.Enums.AddEnumAltLower(X, false, nil, "", int64(Y)+1)

//go:generate stringer -type=Dims2D

// 2D vector -- a point or size in 2D
type Vec2D struct {
	X, Y float64
}

var Vec2DZero = Vec2D{0, 0}

// return value along given dimension
func (a Vec2D) Dim(d Dims2D) float64 {
	switch d {
	case X:
		return a.X
	default:
		return a.Y
	}
}

// get the other dimension
func OtherDim(d Dims2D) Dims2D {
	switch d {
	case X:
		return Y
	default:
		return X
	}
}

// set the value along a given dimension
func (a *Vec2D) SetDim(d Dims2D, val float64) {
	switch d {
	case X:
		a.X = val
	case Y:
		a.Y = val
	}
}

// set values
func (a *Vec2D) Set(x, y float64) {
	a.X = x
	a.Y = y
}

// set both dims to same value
func (a *Vec2D) SetVal(val float64) {
	a.X = val
	a.Y = val
}

func (a Vec2D) IsZero() bool {
	return a.X == 0.0 && a.Y == 0.0
}

func (a Vec2D) Fixed() fixed.Point26_6 {
	return fixp(a.X, a.Y)
}

func (a Vec2D) Add(b Vec2D) Vec2D {
	return Vec2D{a.X + b.X, a.Y + b.Y}
}

func (a Vec2D) AddVal(val float64) Vec2D {
	return Vec2D{a.X + val, a.Y + val}
}

func (a *Vec2D) SetAdd(b Vec2D) {
	a.X += b.X
	a.Y += b.Y
}

func (a *Vec2D) SetAddVal(val float64) {
	a.X += val
	a.Y += val
}

func (a Vec2D) Sub(b Vec2D) Vec2D {
	return Vec2D{a.X - b.X, a.Y - b.Y}
}

func (a *Vec2D) SetSub(b Vec2D) {
	a.X -= b.X
	a.Y -= b.Y
}

func (a *Vec2D) SetSubVal(val float64) {
	a.X -= val
	a.Y -= val
}

func (a Vec2D) SubVal(val float64) Vec2D {
	return Vec2D{a.X - val, a.Y - val}
}

func (a Vec2D) Mul(b Vec2D) Vec2D {
	return Vec2D{a.X * b.X, a.Y * b.Y}
}

func (a *Vec2D) SetMul(b Vec2D) {
	a.X *= b.X
	a.Y *= b.Y
}

func (a Vec2D) MulVal(val float64) Vec2D {
	return Vec2D{a.X * val, a.Y * val}
}

func (a *Vec2D) SetMulVal(val float64) {
	a.X *= val
	a.Y *= val
}

func (a Vec2D) Div(b Vec2D) Vec2D {
	return Vec2D{a.X / b.X, a.Y / b.Y}
}

func (a *Vec2D) SetDiv(b Vec2D) {
	a.X /= b.X
	a.Y /= b.Y
}

func (a *Vec2D) SetDivlVal(val float64) {
	a.X /= val
	a.Y /= val
}

func (a Vec2D) DivVal(val float64) Vec2D {
	return Vec2D{a.X / val, a.Y / val}
}

func (a Vec2D) Max(b Vec2D) Vec2D {
	return Vec2D{math.Max(a.X, b.X), math.Max(a.Y, b.Y)}
}

func (a Vec2D) Min(b Vec2D) Vec2D {
	return Vec2D{math.Min(a.X, b.X), math.Min(a.Y, b.Y)}
}

// minimum of all positive (> 0) numbers
func (a Vec2D) MinPos(b Vec2D) Vec2D {
	return Vec2D{kit.MinPos(a.X, b.X), kit.MinPos(a.Y, b.Y)}
}

// set to max of current vs. b
func (a *Vec2D) SetMax(b Vec2D) {
	a.X = math.Max(a.X, b.X)
	a.Y = math.Max(a.Y, b.Y)
}

// set to min of current vs. b
func (a *Vec2D) SetMin(b Vec2D) {
	a.X = math.Min(a.X, b.X)
	a.Y = math.Min(a.Y, b.Y)
}

// set to minpos of current vs. b
func (a *Vec2D) SetMinPos(b Vec2D) {
	a.X = kit.MinPos(a.X, b.X)
	a.Y = kit.MinPos(a.Y, b.Y)
}

// set to max of current value and val
func (a *Vec2D) SetMaxVal(val float64) {
	a.X = math.Max(a.X, val)
	a.Y = math.Max(a.Y, val)
}

// set to min of current value and val
func (a *Vec2D) SetMinVal(val float64) {
	a.X = math.Min(a.X, val)
	a.Y = math.Min(a.Y, val)
}

// set to minpos of current value and val
func (a *Vec2D) SetMinPosVal(val float64) {
	a.X = kit.MinPos(a.X, val)
	a.Y = kit.MinPos(a.Y, val)
}

// set the value along a given dimension to max of current val and new val
func (a *Vec2D) SetMaxDim(d Dims2D, val float64) {
	switch d {
	case X:
		a.X = math.Max(a.X, val)
	case Y:
		a.Y = math.Max(a.Y, val)
	}
}

// set the value along a given dimension to min of current val and new val
func (a *Vec2D) SetMinDim(d Dims2D, val float64) {
	switch d {
	case X:
		a.X = math.Min(a.X, val)
	case Y:
		a.Y = math.Min(a.Y, val)
	}
}

// set the value along a given dimension to min of current val and new val
func (a *Vec2D) SetMinPosDim(d Dims2D, val float64) {
	switch d {
	case X:
		a.X = kit.MinPos(val, a.X)
	case Y:
		a.Y = kit.MinPos(val, a.Y)
	}
}

func (a *Vec2D) SetFromPoint(pt image.Point) {
	a.X = float64(pt.X)
	a.Y = float64(pt.Y)
}

func (a Vec2D) Distance(b Vec2D) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

func (a Vec2D) Interpolate(b Vec2D, t float64) Vec2D {
	x := a.X + (b.X-a.X)*t
	y := a.Y + (b.Y-a.Y)*t
	return Vec2D{x, y}
}

func (a Vec2D) String() string {
	return fmt.Sprintf("%v, %v", a.X, a.Y)
}

////////////////////////////////////////////////////////////////////////////////////////
// XFormMatrix2D

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
	None  ViewBoxAlign = 1 << iota          // do not preserve uniform scaling
	XMin                                    // align ViewBox.Min with smallest values of Viewport
	XMid                                    // align ViewBox.Min with midpoint values of Viewport
	XMax                                    // align ViewBox.Min+Size with maximum values of Viewport
	XMask ViewBoxAlign = XMin + XMid + XMax // mask for X values -- clear all X before setting new one
	YMin  ViewBoxAlign = 1 << iota          // align ViewBox.Min with smallest values of Viewport
	YMid                                    // align ViewBox.Min with midpoint values of Viewport
	YMax                                    // align ViewBox.Min+Size with maximum values of Viewport
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
	Align       ViewBoxAlign       `svg:"align" desc:"how to align x,y coordinates within viewbox"`
	MeetOrSlice ViewBoxMeetOrSlice `svg:"meetOrSlice" desc:"how to scale the view box relative to the viewport"`
}

// ViewBox defines a region in 2D bitmap image space -- it must ALWAYS be in terms of underlying pixels
type ViewBox2D struct {
	Min                 image.Point                `svg:"{min-x,min-y}" desc:"offset or starting point in parent Viewport2D"`
	Size                image.Point                `svg:"{width,height}" desc:"size of viewbox within parent Viewport2D"`
	PreserveAspectRatio ViewBoxPreserveAspectRatio `svg:"preserveAspectRatio" desc:"how to scale the view box within parent Viewport2D"`
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
