// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

import (
	"fmt"
	"image"

	"github.com/chewxy/math32"
	"golang.org/x/image/math/fixed"
)

// Vec2 is a 2D vector/point with X and Y components.
type Vec2 struct {
	X float32
	Y float32
}

var (
	Vec2Zero = Vec2{0, 0}
	Vec2X    = Vec2{1, 0}
	Vec2Y    = Vec2{0, 1}
)

// NewVec2 returns new Vec2 with the specified x and y components.
func NewVec2(x, y float32) Vec2 {
	return Vec2{X: x, Y: y}
}

// NewVec2Scalar returns a new Vec2 with all components set to scalar.
func NewVec2Scalar(s float32) Vec2 {
	return Vec2{X: s, Y: s}
}

func NewVec2FmPoint(pt image.Point) Vec2 {
	v := Vec2{}
	v.SetPoint(pt)
	return v
}

func NewVec2FmFixed(pt fixed.Point26_6) Vec2 {
	v := Vec2{}
	v.SetFixed(pt)
	return v
}

// IsNil returns true if all values are 0 (uninitialized).
func (v Vec2) IsNil() bool {
	if v.X == 0 && v.Y == 0 {
		return true
	}
	return false
}

// Set sets this vector X and Y components.
func (v *Vec2) Set(x, y float32) {
	v.X = x
	v.Y = y
}

// SetScalar sets all vector components to same scalar value.
func (v *Vec2) SetScalar(s float32) {
	v.X = s
	v.Y = s
}

// SetFromVec2i sets from a Vec2i (int32) vector.
func (v *Vec2) SetFromVec2i(vi Vec2i) {
	v.X = float32(vi.X)
	v.Y = float32(vi.Y)
}

// SetDim sets this vector component value by its dimension index.
func (v *Vec2) SetDim(dim Dims, value float32) {
	switch dim {
	case X:
		v.X = value
	case Y:
		v.Y = value
	default:
		panic("dim is out of range")
	}
}

// Dim returns this vector component.
func (v Vec2) Dim(dim Dims) float32 {
	switch dim {
	case X:
		return v.X
	case Y:
		return v.Y
	default:
		panic("dim is out of range")
	}
}

// SetByName sets this vector component value by its case insensitive name: "x" or "y".
func (v *Vec2) SetByName(name string, value float32) {
	switch name {
	case "x", "X":
		v.X = value
	case "y", "Y":
		v.Y = value
	default:
		panic("Invalid Vec2 component name: " + name)
	}
}

func (a Vec2) String() string {
	return fmt.Sprintf("(%v, %v)", a.X, a.Y)
}

func (a Vec2) Fixed() fixed.Point26_6 {
	return ToFixedPoint(a.X, a.Y)
}

func (a *Vec2) SetAddDim(d Dims, val float32) {
	switch d {
	case X:
		a.X += val
	case Y:
		a.Y += val
	}
}

func (a *Vec2) SetSubDim(d Dims, val float32) {
	switch d {
	case X:
		a.X -= val
	case Y:
		a.Y -= val
	}
}

func (a *Vec2) SetMulDim(d Dims, val float32) {
	switch d {
	case X:
		a.X *= val
	case Y:
		a.Y *= val
	}
}

func (a *Vec2) SetDivDim(d Dims, val float32) {
	switch d {
	case X:
		a.X /= val
	case Y:
		a.Y /= val
	}
}

// set the value along a given dimension to max of current val and new val
func (a *Vec2) SetMaxDim(d Dims, val float32) {
	switch d {
	case X:
		a.X = Max(a.X, val)
	case Y:
		a.Y = Max(a.Y, val)
	}
}

// set the value along a given dimension to min of current val and new val
func (a *Vec2) SetMinDim(d Dims, val float32) {
	switch d {
	case X:
		a.X = Min(a.X, val)
	case Y:
		a.Y = Min(a.Y, val)
	}
}

// set the value along a given dimension to min of current val and new val
func (a *Vec2) SetMinPosDim(d Dims, val float32) {
	switch d {
	case X:
		a.X = MinPos(val, a.X)
	case Y:
		a.Y = MinPos(val, a.Y)
	}
}

func (a *Vec2) SetPoint(pt image.Point) {
	a.X = float32(pt.X)
	a.Y = float32(pt.Y)
}

func (a *Vec2) SetFixed(pt fixed.Point26_6) {
	a.X = FromFixed(pt.X)
	a.Y = FromFixed(pt.Y)
}

func (a Vec2) ToPoint() image.Point {
	return image.Point{int(a.X), int(a.Y)}
}

func (a Vec2) ToPointCeil() image.Point {
	return image.Point{int(math32.Ceil(a.X)), int(math32.Ceil(a.Y))}
}

func (a Vec2) ToPointFloor() image.Point {
	return image.Point{int(math32.Floor(a.X)), int(math32.Floor(a.Y))}
}

func (a Vec2) ToPointRound() image.Point {
	return image.Point{int(Round(a.X)), int(Round(a.Y))}
}

// RectFromPosSizeMax returns an image.Rectangle from max dims of pos, size
// (floor on pos, ceil on size)
func RectFromPosSizeMax(pos, sz Vec2) image.Rectangle {
	tp := pos.ToPointFloor()
	ts := sz.ToPointCeil()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

// RectFromPosSizeMin returns an image.Rectangle from min dims of pos, size
// (ceil on pos, floor on size)
func RectFromPosSizeMin(pos, sz Vec2) image.Rectangle {
	tp := pos.ToPointCeil()
	ts := sz.ToPointFloor()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

// SetZero sets this vector X and Y components to be zero.
func (v *Vec2) SetZero() {
	v.SetScalar(0)
}

// FromArray sets this vector's components from the specified array and offset.
func (v *Vec2) FromArray(array []float32, offset int) {
	v.X = array[offset]
	v.Y = array[offset+1]
}

// ToArray copies this vector's components to array starting at offset.
func (v Vec2) ToArray(array []float32, offset int) {
	array[offset] = v.X
	array[offset+1] = v.Y
}

///////////////////////////////////////////////////////////////////////
//  Basic math operations

// Add adds other vector to this one and returns result in a new vector.
func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{v.X + other.X, v.Y + other.Y}
}

// AddScalar adds scalar s to each component of this vector and returns new vector.
func (v Vec2) AddScalar(s float32) Vec2 {
	return Vec2{v.X + s, v.Y + s}
}

// SetAdd sets this to addition with other vector (i.e., += or plus-equals).
func (v *Vec2) SetAdd(other Vec2) {
	v.X += other.X
	v.Y += other.Y
}

// SetAddScalar sets this to addition with scalar.
func (v *Vec2) SetAddScalar(s float32) {
	v.X += s
	v.Y += s
}

// Sub subtracts other vector from this one and returns result in new vector.
func (v Vec2) Sub(other Vec2) Vec2 {
	return Vec2{v.X - other.X, v.Y - other.Y}
}

// SubScalar subtracts scalar s from each component of this vector and returns new vector.
func (v Vec2) SubScalar(s float32) Vec2 {
	return Vec2{v.X - s, v.Y - s}
}

// SetSub sets this to subtraction with other vector (i.e., -= or minus-equals).
func (v *Vec2) SetSub(other Vec2) {
	v.X -= other.X
	v.Y -= other.Y
}

// SetSubScalar sets this to subtraction of scalar.
func (v *Vec2) SetSubScalar(s float32) {
	v.X -= s
	v.Y -= s
}

// Mul multiplies each component of this vector by the corresponding one from other
// and returns resulting vector.
func (v Vec2) Mul(other Vec2) Vec2 {
	return Vec2{v.X * other.X, v.Y * other.Y}
}

// MulScalar multiplies each component of this vector by the scalar s and returns resulting vector.
func (v Vec2) MulScalar(s float32) Vec2 {
	return Vec2{v.X * s, v.Y * s}
}

// SetMul sets this to multiplication with other vector (i.e., *= or times-equals).
func (v *Vec2) SetMul(other Vec2) {
	v.X *= other.X
	v.Y *= other.Y
}

// SetMulScalar sets this to multiplication by scalar.
func (v *Vec2) SetMulScalar(s float32) {
	v.X *= s
	v.Y *= s
}

// Div divides each component of this vector by the corresponding one from other vector
// and returns resulting vector.
func (v Vec2) Div(other Vec2) Vec2 {
	return Vec2{v.X / other.X, v.Y / other.Y}
}

// DivScalar divides each component of this vector by the scalar s and returns resulting vector.
// If scalar is zero, returns zero.
func (v Vec2) DivScalar(scalar float32) Vec2 {
	if scalar != 0 {
		return v.MulScalar(1 / scalar)
	} else {
		return Vec2{}
	}
}

// SetDiv sets this to division by other vector (i.e., /= or divide-equals).
func (v *Vec2) SetDiv(other Vec2) {
	v.X /= other.X
	v.Y /= other.Y
}

// SetDivScalar sets this to division by scalar.
func (v *Vec2) SetDivScalar(s float32) {
	if s != 0 {
		v.SetMulScalar(1 / s)
	} else {
		v.SetZero()
	}
}

// Abs returns the absolute value for each dimension
func (v Vec2) Abs() Vec2 {
	return Vec2{Abs(v.X), Abs(v.Y)}
}

// Min returns min of this vector components vs. other vector.
func (v Vec2) Min(other Vec2) Vec2 {
	return Vec2{Min(v.X, other.X), Min(v.Y, other.Y)}
}

// SetMin sets this vector components to the minimum values of itself and other vector.
func (v *Vec2) SetMin(other Vec2) {
	v.X = Min(v.X, other.X)
	v.Y = Min(v.Y, other.Y)
}

// Max returns max of this vector components vs. other vector.
func (v Vec2) Max(other Vec2) Vec2 {
	return Vec2{Max(v.X, other.X), Max(v.Y, other.Y)}
}

// SetMax sets this vector components to the maximum value of itself and other vector.
func (v *Vec2) SetMax(other Vec2) {
	v.X = Max(v.X, other.X)
	v.Y = Max(v.Y, other.Y)
}

// MinPos returns minimum of all positive (> 0) numbers
func (a Vec2) MinPos(b Vec2) Vec2 {
	return Vec2{MinPos(a.X, b.X), MinPos(a.Y, b.Y)}
}

// SetMinPos set to minpos of current vs. other
func (v *Vec2) SetMinPos(b Vec2) {
	v.X = MinPos(v.X, b.X)
	v.Y = MinPos(v.Y, b.Y)
}

// SetMaxScalar sets to max of current value and scalar val
func (v *Vec2) SetMaxScalar(val float32) {
	v.X = Max(v.X, val)
	v.Y = Max(v.Y, val)
}

// SetMinScalar sets to min of current value and scalar val
func (v *Vec2) SetMinScalar(val float32) {
	v.X = Min(v.X, val)
	v.Y = Min(v.Y, val)
}

// SetMinPosScalar sets to minpos of current value and scalar val
func (v *Vec2) SetMinPosScalar(val float32) {
	v.X = MinPos(v.X, val)
	v.Y = MinPos(v.Y, val)
}

// Clamp sets this vector components to be no less than the corresponding components of min
// and not greater than the corresponding component of max.
// Assumes min < max, if this assumption isn't true it will not operate correctly.
func (v *Vec2) Clamp(min, max Vec2) {
	if v.X < min.X {
		v.X = min.X
	} else if v.X > max.X {
		v.X = max.X
	}
	if v.Y < min.Y {
		v.Y = min.Y
	} else if v.Y > max.Y {
		v.Y = max.Y
	}
}

// ClampScalar sets this vector components to be no less than minVal and not greater than maxVal.
func (v *Vec2) ClampScalar(minVal, maxVal float32) {
	v.Clamp(NewVec2Scalar(minVal), NewVec2Scalar(maxVal))
}

// Floor returns vector with mat32.Floor() applied to each of this vector's components.
func (v Vec2) Floor() Vec2 {
	return Vec2{Floor(v.X), Floor(v.Y)}
}

// SetFloor applies mat32.Floor() to each of this vector's components.
func (v *Vec2) SetFloor() {
	v.X = Floor(v.X)
	v.Y = Floor(v.Y)
}

// Ceil returns vector with mat32.Ceil() applied to each of this vector's components.
func (v Vec2) Ceil() Vec2 {
	return Vec2{Ceil(v.X), Ceil(v.Y)}
}

// SetCeil applies mat32.Ceil() to each of this vector's components.
func (v *Vec2) SetCeil() {
	v.X = Ceil(v.X)
	v.Y = Ceil(v.Y)
}

// Round returns vector with mat32.Round() applied to each of this vector's components.
func (v Vec2) Round() Vec2 {
	return Vec2{Round(v.X), Round(v.Y)}
}

// SetRound rounds each of this vector's components.
func (v *Vec2) SetRound() {
	v.X = Round(v.X)
	v.Y = Round(v.Y)
}

// Negate returns vector with each component negated.
func (v Vec2) Negate() Vec2 {
	return Vec2{-v.X, -v.Y}
}

// SetNegate negates each of this vector's components.
func (v *Vec2) SetNegate() {
	v.X = -v.X
	v.Y = -v.Y
}

//////////////////////////////////////////////////////////////////////////////////
//  Distance, Norm

// IsEqual returns if this vector is equal to other.
func (v Vec2) IsEqual(other Vec2) bool {
	return (other.X == v.X) && (other.Y == v.Y)
}

// AlmostEqual returns whether the vector is almost equal to another vector within the specified tolerance.
func (v Vec2) AlmostEqual(other Vec2, tol float32) bool {
	if (Abs(v.X-other.X) < tol) && (Abs(v.Y-other.Y) < tol) {
		return true
	}
	return false
}

// Dot returns the dot product of this vector with other.
func (v Vec2) Dot(other Vec2) float32 {
	return v.X*other.X + v.Y*other.Y
}

// LengthSq returns the length squared of this vector.
// LengthSq can be used to compare vectors' lengths without the need to perform a square root.
func (v Vec2) LengthSq() float32 {
	return v.X*v.X + v.Y*v.Y
}

// Length returns the length of this vector.
func (v Vec2) Length() float32 {
	return Sqrt(v.X*v.X + v.Y*v.Y)
}

// Normal returns this vector divided by its length
func (v Vec2) Normal() Vec2 {
	return v.DivScalar(v.Length())
}

// SetNormal normalizes this vector so its length will be 1.
func (v *Vec2) SetNormal() {
	v.SetDivScalar(v.Length())
}

// Normalize normalizes this vector so its length will be 1.
func (v *Vec2) Normalize() {
	v.SetDivScalar(v.Length())
}

// DistTo returns the distance of this point to other.
func (v Vec2) DistTo(other Vec2) float32 {
	return Sqrt(v.DistToSquared(other))
}

// DistToSquared returns the distance squared of this point to other.
func (v Vec2) DistToSquared(other Vec2) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	return dx*dx + dy*dy
}

// SetLength sets this vector to have the specified length.
func (v *Vec2) SetLength(l float32) {
	oldLength := v.Length()
	if oldLength != 0 && l != oldLength {
		v.SetMulScalar(l / oldLength)
	}
}

// Lerp returns vector with each components as the linear interpolated value of
// alpha between itself and the corresponding other component.
func (v Vec2) Lerp(other Vec2, alpha float32) Vec2 {
	return Vec2{v.X + (other.X-v.X)*alpha, v.Y + (other.Y-v.Y)*alpha}
}

// Lerp sets each of this vector's components to the linear interpolated value of
// alpha between ifself and the corresponding other component.
func (v *Vec2) SetLerp(other Vec2, alpha float32) {
	v.X += (other.X - v.X) * alpha
	v.Y += (other.Y - v.Y) * alpha
}

// InTriangle returns whether the vector is inside the specified triangle.
func (v Vec2) InTriangle(p0, p1, p2 Vec2) bool {
	A := 0.5 * (-p1.Y*p2.X + p0.Y*(-p1.X+p2.X) + p0.X*(p1.Y-p2.Y) + p1.X*p2.Y)
	sign := float32(1)
	if A < 0 {
		sign = float32(-1)
	}
	s := (p0.Y*p2.X - p0.X*p2.Y + (p2.Y-p0.Y)*v.X + (p0.X-p2.X)*v.Y) * sign
	t := (p0.X*p1.Y - p0.Y*p1.X + (p0.Y-p1.Y)*v.X + (p1.X-p0.X)*v.Y) * sign

	return s >= 0 && t >= 0 && (s+t) < 2*A*sign
}
