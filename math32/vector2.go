// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit Cogent Core functionality.

package math32

import (
	"fmt"
	"image"

	"golang.org/x/image/math/fixed"
)

// Vector2 is a 2D vector/point with X and Y components.
type Vector2 struct {
	X float32
	Y float32
}

// V2 returns a new [Vector2] with the given x and y components.
func V2(x, y float32) Vector2 {
	return Vector2{x, y}
}

// V2Scalar returns a new [Vector2] with all components set to the given scalar value.
func V2Scalar(s float32) Vector2 {
	return Vector2{s, s}
}

// V2FromPoint returns a new [Vector2] from the given [image.Point].
func V2FromPoint(pt image.Point) Vector2 {
	v := Vector2{}
	v.SetPoint(pt)
	return v
}

// V2FromFixed returns a new [Vector2] from the given [fixed.Point26_6].
func V2FromFixed(pt fixed.Point26_6) Vector2 {
	v := Vector2{}
	v.SetFixed(pt)
	return v
}

// Set sets this vector's X and Y components.
func (v *Vector2) Set(x, y float32) {
	v.X = x
	v.Y = y
}

// SetScalar sets all vector components to same scalar value.
func (v *Vector2) SetScalar(s float32) {
	v.X = s
	v.Y = s
}

// SetFromVector2i sets from a Vector2i (int32) vector.
func (v *Vector2) SetFromVector2i(vi Vector2i) {
	v.X = float32(vi.X)
	v.Y = float32(vi.Y)
}

// SetDim sets this vector component value by its dimension index.
func (v *Vector2) SetDim(dim Dims, value float32) {
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
func (v Vector2) Dim(dim Dims) float32 {
	switch dim {
	case X:
		return v.X
	case Y:
		return v.Y
	default:
		panic("dim is out of range")
	}
}

// SetPointDim is a helper function for image.Point for setting given dimension
func SetPointDim(v *image.Point, dim Dims, value int) {
	switch dim {
	case X:
		v.X = value
	case Y:
		v.Y = value
	default:
		panic("dim is out of range")
	}
}

// Dim returns this vector component from given image.Point
func PointDim(v image.Point, dim Dims) int {
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
func (v *Vector2) SetByName(name string, value float32) {
	switch name {
	case "x", "X":
		v.X = value
	case "y", "Y":
		v.Y = value
	default:
		panic("Invalid Vector2 component name: " + name)
	}
}

func (a Vector2) String() string {
	return fmt.Sprintf("(%v, %v)", a.X, a.Y)
}

func (a Vector2) Fixed() fixed.Point26_6 {
	return ToFixedPoint(a.X, a.Y)
}

func (a Vector2) AddDim(d Dims, val float32) Vector2 {
	switch d {
	case X:
		a.X += val
	case Y:
		a.Y += val
	}
	return a
}

func (a *Vector2) SetAddDim(d Dims, val float32) {
	switch d {
	case X:
		a.X += val
	case Y:
		a.Y += val
	}
}

func (a Vector2) SubDim(d Dims, val float32) Vector2 {
	switch d {
	case X:
		a.X -= val
	case Y:
		a.Y -= val
	}
	return a
}

func (a *Vector2) SetSubDim(d Dims, val float32) {
	switch d {
	case X:
		a.X -= val
	case Y:
		a.Y -= val
	}
}

func (a Vector2) MulDim(d Dims, val float32) Vector2 {
	switch d {
	case X:
		a.X *= val
	case Y:
		a.Y *= val
	}
	return a
}

func (a *Vector2) SetMulDim(d Dims, val float32) {
	switch d {
	case X:
		a.X *= val
	case Y:
		a.Y *= val
	}
}

func (a Vector2) DivDim(d Dims, val float32) Vector2 {
	switch d {
	case X:
		a.X /= val
	case Y:
		a.Y /= val
	}
	return a
}

func (a *Vector2) SetDivDim(d Dims, val float32) {
	switch d {
	case X:
		a.X /= val
	case Y:
		a.Y /= val
	}
}

// set the value along a given dimension to max of current val and new val
func (a *Vector2) SetMaxDim(d Dims, val float32) {
	switch d {
	case X:
		a.X = Max(a.X, val)
	case Y:
		a.Y = Max(a.Y, val)
	}
}

// set the value along a given dimension to min of current val and new val
func (a *Vector2) SetMinDim(d Dims, val float32) {
	switch d {
	case X:
		a.X = Min(a.X, val)
	case Y:
		a.Y = Min(a.Y, val)
	}
}

// set the value along a given dimension to min of current val and new val
func (a *Vector2) SetMinPosDim(d Dims, val float32) {
	switch d {
	case X:
		a.X = MinPos(val, a.X)
	case Y:
		a.Y = MinPos(val, a.Y)
	}
}

func (a *Vector2) SetPoint(pt image.Point) {
	a.X = float32(pt.X)
	a.Y = float32(pt.Y)
}

func (a *Vector2) SetFixed(pt fixed.Point26_6) {
	a.X = FromFixed(pt.X)
	a.Y = FromFixed(pt.Y)
}

func (a Vector2) ToCeil() Vector2 {
	return V2(Ceil(a.X), Ceil(a.Y))
}

func (a Vector2) ToFloor() Vector2 {
	return V2(Floor(a.X), Floor(a.Y))
}

func (a Vector2) ToRound() Vector2 {
	return V2(Round(a.X), Round(a.Y))
}

func (a Vector2) ToPoint() image.Point {
	return image.Point{int(a.X), int(a.Y)}
}

func (a Vector2) ToPointCeil() image.Point {
	return image.Point{int(Ceil(a.X)), int(Ceil(a.Y))}
}

func (a Vector2) ToPointFloor() image.Point {
	return image.Point{int(Floor(a.X)), int(Floor(a.Y))}
}

func (a Vector2) ToPointRound() image.Point {
	return image.Point{int(Round(a.X)), int(Round(a.Y))}
}

// RectFromPosSizeMax returns an image.Rectangle from max dims of pos, size
// (floor on pos, ceil on size)
func RectFromPosSizeMax(pos, sz Vector2) image.Rectangle {
	tp := pos.ToPointFloor()
	ts := sz.ToPointCeil()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

// RectFromPosSizeMin returns an image.Rectangle from min dims of pos, size
// (ceil on pos, floor on size)
func RectFromPosSizeMin(pos, sz Vector2) image.Rectangle {
	tp := pos.ToPointCeil()
	ts := sz.ToPointFloor()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

// SetZero sets this vector X and Y components to be zero.
func (v *Vector2) SetZero() {
	v.SetScalar(0)
}

// FromArray sets this vector's components from the specified array and offset.
func (v *Vector2) FromArray(array []float32, offset int) {
	v.X = array[offset]
	v.Y = array[offset+1]
}

// ToArray copies this vector's components to array starting at offset.
func (v Vector2) ToArray(array []float32, offset int) {
	array[offset] = v.X
	array[offset+1] = v.Y
}

///////////////////////////////////////////////////////////////////////
//  Basic math operations

// Add adds other vector to this one and returns result in a new vector.
func (v Vector2) Add(other Vector2) Vector2 {
	return V2(v.X+other.X, v.Y+other.Y)
}

// AddScalar adds scalar s to each component of this vector and returns new vector.
func (v Vector2) AddScalar(s float32) Vector2 {
	return V2(v.X+s, v.Y+s)
}

// SetAdd sets this to addition with other vector (i.e., += or plus-equals).
func (v *Vector2) SetAdd(other Vector2) {
	v.X += other.X
	v.Y += other.Y
}

// SetAddScalar sets this to addition with scalar.
func (v *Vector2) SetAddScalar(s float32) {
	v.X += s
	v.Y += s
}

// Sub subtracts other vector from this one and returns result in new vector.
func (v Vector2) Sub(other Vector2) Vector2 {
	return V2(v.X-other.X, v.Y-other.Y)
}

// SubScalar subtracts scalar s from each component of this vector and returns new vector.
func (v Vector2) SubScalar(s float32) Vector2 {
	return V2(v.X-s, v.Y-s)
}

// SetSub sets this to subtraction with other vector (i.e., -= or minus-equals).
func (v *Vector2) SetSub(other Vector2) {
	v.X -= other.X
	v.Y -= other.Y
}

// SetSubScalar sets this to subtraction of scalar.
func (v *Vector2) SetSubScalar(s float32) {
	v.X -= s
	v.Y -= s
}

// Mul multiplies each component of this vector by the corresponding one from other
// and returns resulting vector.
func (v Vector2) Mul(other Vector2) Vector2 {
	return V2(v.X*other.X, v.Y*other.Y)
}

// MulScalar multiplies each component of this vector by the scalar s and returns resulting vector.
func (v Vector2) MulScalar(s float32) Vector2 {
	return V2(v.X*s, v.Y*s)
}

// SetMul sets this to multiplication with other vector (i.e., *= or times-equals).
func (v *Vector2) SetMul(other Vector2) {
	v.X *= other.X
	v.Y *= other.Y
}

// SetMulScalar sets this to multiplication by scalar.
func (v *Vector2) SetMulScalar(s float32) {
	v.X *= s
	v.Y *= s
}

// Div divides each component of this vector by the corresponding one from other vector
// and returns resulting vector.
func (v Vector2) Div(other Vector2) Vector2 {
	return V2(v.X/other.X, v.Y/other.Y)
}

// DivScalar divides each component of this vector by the scalar s and returns resulting vector.
// If scalar is zero, returns zero.
func (v Vector2) DivScalar(scalar float32) Vector2 {
	if scalar != 0 {
		return v.MulScalar(1 / scalar)
	} else {
		return Vector2{}
	}
}

// SetDiv sets this to division by other vector (i.e., /= or divide-equals).
func (v *Vector2) SetDiv(other Vector2) {
	v.X /= other.X
	v.Y /= other.Y
}

// SetDivScalar sets this to division by scalar.
func (v *Vector2) SetDivScalar(s float32) {
	if s != 0 {
		v.SetMulScalar(1 / s)
	} else {
		v.SetZero()
	}
}

// Abs returns the absolute value for each dimension
func (v Vector2) Abs() Vector2 {
	return V2(Abs(v.X), Abs(v.Y))
}

// Min returns min of this vector components vs. other vector.
func (v Vector2) Min(other Vector2) Vector2 {
	return V2(Min(v.X, other.X), Min(v.Y, other.Y))
}

// SetMin sets this vector components to the minimum values of itself and other vector.
func (v *Vector2) SetMin(other Vector2) {
	v.X = Min(v.X, other.X)
	v.Y = Min(v.Y, other.Y)
}

// Max returns max of this vector components vs. other vector.
func (v Vector2) Max(other Vector2) Vector2 {
	return V2(Max(v.X, other.X), Max(v.Y, other.Y))
}

// SetMax sets this vector components to the maximum value of itself and other vector.
func (v *Vector2) SetMax(other Vector2) {
	v.X = Max(v.X, other.X)
	v.Y = Max(v.Y, other.Y)
}

// MinPos returns minimum of all positive (> 0) numbers
func (a Vector2) MinPos(b Vector2) Vector2 {
	return V2(MinPos(a.X, b.X), MinPos(a.Y, b.Y))
}

// SetMinPos set to minpos of current vs. other
func (v *Vector2) SetMinPos(b Vector2) {
	v.X = MinPos(v.X, b.X)
	v.Y = MinPos(v.Y, b.Y)
}

// set the value along a given dimension to min of current val and new val
func (a *Vector2) SetMaxPos(o Vector2) {
	a.X = MaxPos(o.X, a.X)
	a.Y = MaxPos(o.Y, a.Y)
}

// SetMaxScalar sets to max of current value and scalar val
func (v *Vector2) SetMaxScalar(val float32) {
	v.X = Max(v.X, val)
	v.Y = Max(v.Y, val)
}

// SetMinScalar sets to min of current value and scalar val
func (v *Vector2) SetMinScalar(val float32) {
	v.X = Min(v.X, val)
	v.Y = Min(v.Y, val)
}

// SetMinPosScalar sets to minpos of current value and scalar val
func (v *Vector2) SetMinPosScalar(val float32) {
	v.X = MinPos(v.X, val)
	v.Y = MinPos(v.Y, val)
}

// Clamp sets this vector components to be no less than the corresponding components of min
// and not greater than the corresponding component of max.
// Assumes min < max, if this assumption isn't true it will not operate correctly.
func (v *Vector2) Clamp(min, max Vector2) {
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
func (v *Vector2) ClampScalar(minVal, maxVal float32) {
	v.Clamp(V2Scalar(minVal), V2Scalar(maxVal))
}

// Floor returns vector with math32.Floor() applied to each of this vector's components.
func (v Vector2) Floor() Vector2 {
	return V2(Floor(v.X), Floor(v.Y))
}

// SetFloor applies math32.Floor() to each of this vector's components.
func (v *Vector2) SetFloor() {
	v.X = Floor(v.X)
	v.Y = Floor(v.Y)
}

// Ceil returns vector with math32.Ceil() applied to each of this vector's components.
func (v Vector2) Ceil() Vector2 {
	return V2(Ceil(v.X), Ceil(v.Y))
}

// SetCeil applies math32.Ceil() to each of this vector's components.
func (v *Vector2) SetCeil() {
	v.X = Ceil(v.X)
	v.Y = Ceil(v.Y)
}

// Round returns vector with math32.Round() applied to each of this vector's components.
func (v Vector2) Round() Vector2 {
	return V2(Round(v.X), Round(v.Y))
}

// SetRound rounds each of this vector's components.
func (v *Vector2) SetRound() {
	v.X = Round(v.X)
	v.Y = Round(v.Y)
}

// Negate returns vector with each component negated.
func (v Vector2) Negate() Vector2 {
	return V2(-v.X, -v.Y)
}

// SetNegate negates each of this vector's components.
func (v *Vector2) SetNegate() {
	v.X = -v.X
	v.Y = -v.Y
}

//////////////////////////////////////////////////////////////////////////////////
//  Distance, Norm

// IsEqual returns if this vector is equal to other.
func (v Vector2) IsEqual(other Vector2) bool {
	return (other.X == v.X) && (other.Y == v.Y)
}

// AlmostEqual returns whether the vector is almost equal to another vector within the specified tolerance.
func (v Vector2) AlmostEqual(other Vector2, tol float32) bool {
	return (Abs(v.X-other.X) < tol) && (Abs(v.Y-other.Y) < tol)
}

// Dot returns the dot product of this vector with other.
func (v Vector2) Dot(other Vector2) float32 {
	return v.X*other.X + v.Y*other.Y
}

// LengthSq returns the length squared of this vector.
// LengthSq can be used to compare vectors' lengths without the need to perform a square root.
func (v Vector2) LengthSq() float32 {
	return v.X*v.X + v.Y*v.Y
}

// Length returns the length of this vector.
func (v Vector2) Length() float32 {
	return Sqrt(v.X*v.X + v.Y*v.Y)
}

// Normal returns this vector divided by its length
func (v Vector2) Normal() Vector2 {
	return v.DivScalar(v.Length())
}

// SetNormal normalizes this vector so its length will be 1.
func (v *Vector2) SetNormal() {
	v.SetDivScalar(v.Length())
}

// Normalize normalizes this vector so its length will be 1.
func (v *Vector2) Normalize() {
	v.SetDivScalar(v.Length())
}

// DistTo returns the distance of this point to other.
func (v Vector2) DistTo(other Vector2) float32 {
	return Sqrt(v.DistToSquared(other))
}

// DistToSquared returns the distance squared of this point to other.
func (v Vector2) DistToSquared(other Vector2) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	return dx*dx + dy*dy
}

// SetLength sets this vector to have the specified length.
func (v *Vector2) SetLength(l float32) {
	oldLength := v.Length()
	if oldLength != 0 && l != oldLength {
		v.SetMulScalar(l / oldLength)
	}
}

// Cross returns the cross product of this vector with other
// which is a scalar, equivalent to the Z coord in 3D: X1 * Y2 - X2 Y1
func (v Vector2) Cross(other Vector2) float32 {
	return v.X*other.Y - v.Y*other.X
}

// CosTo returns the cosine (normalized dot product) between this vector and other.
func (v Vector2) CosTo(other Vector2) float32 {
	return v.Dot(other) / (v.Length() * other.Length())
}

// AngleTo returns the angle between this vector and other.
// Returns angles in range of -PI to PI (not 0 to 2 PI).
func (v Vector2) AngleTo(other Vector2) float32 {
	ang := Acos(Clamp(v.CosTo(other), -1, 1))
	cross := v.Cross(other)
	if cross > 0 {
		ang = -ang
	}
	return ang
}

// Lerp returns vector with each components as the linear interpolated value of
// alpha between itself and the corresponding other component.
func (v Vector2) Lerp(other Vector2, alpha float32) Vector2 {
	return V2(v.X+(other.X-v.X)*alpha, v.Y+(other.Y-v.Y)*alpha)
}

// Lerp sets each of this vector's components to the linear interpolated value of
// alpha between ifself and the corresponding other component.
func (v *Vector2) SetLerp(other Vector2, alpha float32) {
	v.X += (other.X - v.X) * alpha
	v.Y += (other.Y - v.Y) * alpha
}

// InTriangle returns whether the vector is inside the specified triangle.
func (v Vector2) InTriangle(p0, p1, p2 Vector2) bool {
	A := 0.5 * (-p1.Y*p2.X + p0.Y*(-p1.X+p2.X) + p0.X*(p1.Y-p2.Y) + p1.X*p2.Y)
	sign := float32(1)
	if A < 0 {
		sign = float32(-1)
	}
	s := (p0.Y*p2.X - p0.X*p2.Y + (p2.Y-p0.Y)*v.X + (p0.X-p2.X)*v.Y) * sign
	t := (p0.X*p1.Y - p0.Y*p1.X + (p0.Y-p1.Y)*v.X + (p1.X-p0.X)*v.Y) * sign

	return s >= 0 && t >= 0 && (s+t) < 2*A*sign
}
