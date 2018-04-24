// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"math"
	"strconv"

	"github.com/chewxy/math32"
	"github.com/rcoreilly/goki/ki/kit"
	"golang.org/x/image/math/fixed"
)

// SVG default coordinates are such that 0,0 is upper-left!

/*
This is heavily modified from: https://github.com/fogleman/gg

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

// could break this out as separate package, but no advantage in package-based
// naming

////////////////////////////////////////////////////////////////////////////////////////
//  Min / Max for other types..

// math provides Max/Min for 64bit -- these are for specific subtypes

func Max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func Min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// minimum excluding 0
func MinPos(a, b float64) float64 {
	if a > 0.0 && b > 0.0 {
		return math.Min(a, b)
	} else if a > 0.0 {
		return a
	} else if b > 0.0 {
		return b
	}
	return a
}

// minimum excluding 0
func MinPos32(a, b float32) float32 {
	if a > 0.0 && b > 0.0 {
		return Min32(a, b)
	} else if a > 0.0 {
		return a
	} else if b > 0.0 {
		return b
	}
	return a
}

// Truncate a floating point number to given level of precision -- slow.. uses string conversion
func Truncate(val float64, prec int) float64 {
	frep := strconv.FormatFloat(val, 'g', prec, 64)
	val, _ = strconv.ParseFloat(frep, 64)
	return val
}

// Truncate a floating point number to given level of precision -- slow.. uses string conversion
func Truncate32(val float32, prec int) float32 {
	frep := strconv.FormatFloat(float64(val), 'g', prec, 32)
	tval, _ := strconv.ParseFloat(frep, 32)
	return float32(tval)
}

// FloatMod ensures that a floating point value is an even multiple of a given value
func FloatMod(val, mod float64) float64 {
	return float64(int(math.Round(val/mod))) * mod
}

// FloatMod ensures that a floating point value is an even multiple of a given value
func FloatMod32(val, mod float32) float32 {
	return float32(int(math.Round(float64(val/mod)))) * mod
}

// dimensions
type Dims2D int32

const (
	X Dims2D = iota
	Y
	Dims2DN
)

var KiT_Dims2D = kit.Enums.AddEnumAltLower(Dims2DN, false, nil, "")

//go:generate stringer -type=Dims2D

// 2D vector -- a point or size in 2D
type Vec2D struct {
	X, Y float32
}

var Vec2DZero = Vec2D{0, 0}

func NewVec2D(x, y float32) Vec2D {
	return Vec2D{x, y}
}

func NewVec2DFmPoint(pt image.Point) Vec2D {
	v := Vec2D{}
	v.SetPoint(pt)
	return v
}

// return value along given dimension
func (a Vec2D) Dim(d Dims2D) float32 {
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
func (a *Vec2D) SetDim(d Dims2D, val float32) {
	switch d {
	case X:
		a.X = val
	case Y:
		a.Y = val
	}
}

// set values
func (a *Vec2D) Set(x, y float32) {
	a.X = x
	a.Y = y
}

// set both dims to same value
func (a *Vec2D) SetVal(val float32) {
	a.X = val
	a.Y = val
}

func (a Vec2D) IsZero() bool {
	return a.X == 0.0 && a.Y == 0.0
}

func (a Vec2D) Fixed() fixed.Point26_6 {
	return Float32ToFixedPoint(a.X, a.Y)
}

func (a Vec2D) Add(b Vec2D) Vec2D {
	return Vec2D{a.X + b.X, a.Y + b.Y}
}

func (a Vec2D) AddVal(val float32) Vec2D {
	return Vec2D{a.X + val, a.Y + val}
}

func (a *Vec2D) SetAdd(b Vec2D) {
	a.X += b.X
	a.Y += b.Y
}

func (a *Vec2D) SetAddVal(val float32) {
	a.X += val
	a.Y += val
}

func (a *Vec2D) SetAddDim(d Dims2D, val float32) {
	switch d {
	case X:
		a.X += val
	case Y:
		a.Y += val
	}
}

func (a Vec2D) Sub(b Vec2D) Vec2D {
	return Vec2D{a.X - b.X, a.Y - b.Y}
}

func (a *Vec2D) SetSub(b Vec2D) {
	a.X -= b.X
	a.Y -= b.Y
}

func (a *Vec2D) SetSubVal(val float32) {
	a.X -= val
	a.Y -= val
}

func (a *Vec2D) SetSubDim(d Dims2D, val float32) {
	switch d {
	case X:
		a.X -= val
	case Y:
		a.Y -= val
	}
}

func (a Vec2D) SubVal(val float32) Vec2D {
	return Vec2D{a.X - val, a.Y - val}
}

func (a Vec2D) Mul(b Vec2D) Vec2D {
	return Vec2D{a.X * b.X, a.Y * b.Y}
}

func (a *Vec2D) SetMul(b Vec2D) {
	a.X *= b.X
	a.Y *= b.Y
}

func (a Vec2D) MulVal(val float32) Vec2D {
	return Vec2D{a.X * val, a.Y * val}
}

func (a *Vec2D) SetMulVal(val float32) {
	a.X *= val
	a.Y *= val
}

func (a *Vec2D) SetMulDim(d Dims2D, val float32) {
	switch d {
	case X:
		a.X *= val
	case Y:
		a.Y *= val
	}
}

func (a Vec2D) Div(b Vec2D) Vec2D {
	return Vec2D{a.X / b.X, a.Y / b.Y}
}

func (a *Vec2D) SetDiv(b Vec2D) {
	a.X /= b.X
	a.Y /= b.Y
}

func (a *Vec2D) SetDivlVal(val float32) {
	a.X /= val
	a.Y /= val
}

func (a *Vec2D) SetDivDim(d Dims2D, val float32) {
	switch d {
	case X:
		a.X /= val
	case Y:
		a.Y /= val
	}
}

func (a Vec2D) DivVal(val float32) Vec2D {
	return Vec2D{a.X / val, a.Y / val}
}

func (a Vec2D) Max(b Vec2D) Vec2D {
	return Vec2D{Max32(a.X, b.X), Max32(a.Y, b.Y)}
}

func (a Vec2D) Min(b Vec2D) Vec2D {
	return Vec2D{Min32(a.X, b.X), Min32(a.Y, b.Y)}
}

// minimum of all positive (> 0) numbers
func (a Vec2D) MinPos(b Vec2D) Vec2D {
	return Vec2D{MinPos32(a.X, b.X), MinPos32(a.Y, b.Y)}
}

// set to max of current vs. b
func (a *Vec2D) SetMax(b Vec2D) {
	a.X = Max32(a.X, b.X)
	a.Y = Max32(a.Y, b.Y)
}

// set to min of current vs. b
func (a *Vec2D) SetMin(b Vec2D) {
	a.X = Min32(a.X, b.X)
	a.Y = Min32(a.Y, b.Y)
}

// set to minpos of current vs. b
func (a *Vec2D) SetMinPos(b Vec2D) {
	a.X = MinPos32(a.X, b.X)
	a.Y = MinPos32(a.Y, b.Y)
}

// set to max of current value and val
func (a *Vec2D) SetMaxVal(val float32) {
	a.X = Max32(a.X, val)
	a.Y = Max32(a.Y, val)
}

// set to min of current value and val
func (a *Vec2D) SetMinVal(val float32) {
	a.X = Min32(a.X, val)
	a.Y = Min32(a.Y, val)
}

// set to minpos of current value and val
func (a *Vec2D) SetMinPosVal(val float32) {
	a.X = MinPos32(a.X, val)
	a.Y = MinPos32(a.Y, val)
}

// set the value along a given dimension to max of current val and new val
func (a *Vec2D) SetMaxDim(d Dims2D, val float32) {
	switch d {
	case X:
		a.X = Max32(a.X, val)
	case Y:
		a.Y = Max32(a.Y, val)
	}
}

// set the value along a given dimension to min of current val and new val
func (a *Vec2D) SetMinDim(d Dims2D, val float32) {
	switch d {
	case X:
		a.X = Min32(a.X, val)
	case Y:
		a.Y = Min32(a.Y, val)
	}
}

// set the value along a given dimension to min of current val and new val
func (a *Vec2D) SetMinPosDim(d Dims2D, val float32) {
	switch d {
	case X:
		a.X = MinPos32(val, a.X)
	case Y:
		a.Y = MinPos32(val, a.Y)
	}
}

func (a *Vec2D) SetPoint(pt image.Point) {
	a.X = float32(pt.X)
	a.Y = float32(pt.Y)
}

func (a Vec2D) ToPoint() image.Point {
	return image.Point{int(a.X), int(a.Y)}
}

func (a Vec2D) ToPointCeil() image.Point {
	return image.Point{int(math32.Ceil(a.X)), int(math32.Ceil(a.Y))}
}

func (a Vec2D) ToPointFloor() image.Point {
	return image.Point{int(math32.Floor(a.X)), int(math32.Floor(a.Y))}
}

func (a Vec2D) ToPointRound() image.Point {
	return image.Point{int(math.Round(float64(a.X))), int(math.Round(float64(a.Y)))}
}

// RectFromPosSize returns an image.Rectangle from max dims of pos, size
// (floor on pos, ceil on size)
func RectFromPosSize(pos, sz Vec2D) image.Rectangle {
	tp := pos.ToPointFloor()
	ts := sz.ToPointCeil()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

func (a Vec2D) Distance(b Vec2D) float32 {
	return math32.Hypot(a.X-b.X, a.Y-b.Y)
}

func (a Vec2D) Interpolate(b Vec2D, t float32) Vec2D {
	x := a.X + (b.X-a.X)*t
	y := a.Y + (b.Y-a.Y)*t
	return Vec2D{x, y}
}

func (a Vec2D) String() string {
	return fmt.Sprintf("(%v, %v)", a.X, a.Y)
}

////////////////////////////////////////////////////////////////////////////////////////
// XFormMatrix2D

// todo: in theory a high-quality SVG implementation should use a 64bit xform
// matrix, but that is rather inconvenient and unlikely to be relevant here..
// revisit later

type XFormMatrix2D struct {
	XX, YX, XY, YY, X0, Y0 float32
}

func Identity2D() XFormMatrix2D {
	return XFormMatrix2D{
		1, 0,
		0, 1,
		0, 0,
	}
}

func Translate2D(x, y float32) XFormMatrix2D {
	return XFormMatrix2D{
		1, 0,
		0, 1,
		x, y,
	}
}

func Scale2D(x, y float32) XFormMatrix2D {
	return XFormMatrix2D{
		x, 0,
		0, y,
		0, 0,
	}
}

func Rotate2D(angle float32) XFormMatrix2D {
	c := float32(math32.Cos(angle))
	s := float32(math32.Sin(angle))
	return XFormMatrix2D{
		c, s,
		-s, c,
		0, 0,
	}
}

func Shear2D(x, y float32) XFormMatrix2D {
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

func (a XFormMatrix2D) TransformVector(x, y float32) (tx, ty float32) {
	tx = a.XX*x + a.XY*y
	ty = a.YX*x + a.YY*y
	return
}

func (a XFormMatrix2D) TransformPoint(x, y float32) (tx, ty float32) {
	tx = a.XX*x + a.XY*y + a.X0
	ty = a.YX*x + a.YY*y + a.Y0
	return
}

func (a XFormMatrix2D) TransformPointToInt(x, y float32) (tx, ty int) {
	tx = int(a.XX*x + a.XY*y + a.X0)
	ty = int(a.YX*x + a.YY*y + a.Y0)
	return
}

func (a XFormMatrix2D) Translate(x, y float32) XFormMatrix2D {
	return Translate2D(x, y).Multiply(a)
}

func (a XFormMatrix2D) Scale(x, y float32) XFormMatrix2D {
	return Scale2D(x, y).Multiply(a)
}

func (a XFormMatrix2D) Rotate(angle float32) XFormMatrix2D {
	return Rotate2D(angle).Multiply(a)
}

func (a XFormMatrix2D) Shear(x, y float32) XFormMatrix2D {
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

func Radians(degrees float32) float32 {
	return degrees * math32.Pi / 180
}

func Degrees(radians float32) float32 {
	return radians * 180 / math32.Pi
}

func Float32ToFixedPoint(x, y float32) fixed.Point26_6 {
	return fixed.Point26_6{Float32ToFixed(x), Float32ToFixed(y)}
}

func Float32ToFixed(x float32) fixed.Int26_6 {
	return fixed.Int26_6(x * 64)
}

func FixedToFloat32(x fixed.Int26_6) float32 {
	const shift, mask = 6, 1<<6 - 1
	if x >= 0 {
		return float32(x>>shift) + float32(x&mask)/64
	}
	x = -x
	if x >= 0 {
		return -(float32(x>>shift) + float32(x&mask)/64)
	}
	return 0
}
