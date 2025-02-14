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

	"github.com/chewxy/math32"
	"golang.org/x/image/math/fixed"
)

// Vector2 is a 2D vector/point with X and Y components.
type Vector2 struct {
	X float32
	Y float32
}

// Vec2 returns a new [Vector2] with the given x and y components.
func Vec2(x, y float32) Vector2 {
	return Vector2{x, y}
}

// Vector2Scalar returns a new [Vector2] with all components set to the given scalar value.
func Vector2Scalar(scalar float32) Vector2 {
	return Vector2{scalar, scalar}
}

// Vector2Polar returns a new [Vector2] from polar coordinates,
// with angle in radians CCW and radius the distance from (0,0).
func Vector2Polar(angle, radius float32) Vector2 {
	return Vector2{radius * math32.Cos(angle), radius * math32.Sin(angle)}
}

// FromPoint returns a new [Vector2] from the given [image.Point].
func FromPoint(pt image.Point) Vector2 {
	v := Vector2{}
	v.SetPoint(pt)
	return v
}

// Vector2FromFixed returns a new [Vector2] from the given [fixed.Point26_6].
func Vector2FromFixed(pt fixed.Point26_6) Vector2 {
	v := Vector2{}
	v.SetFixed(pt)
	return v
}

// Set sets this vector's X and Y components.
func (v *Vector2) Set(x, y float32) {
	v.X = x
	v.Y = y
}

// SetScalar sets all vector components to the same scalar value.
func (v *Vector2) SetScalar(scalar float32) {
	v.X = scalar
	v.Y = scalar
}

// SetFromVector2i sets from a [Vector2i] (int32) vector.
func (v *Vector2) SetFromVector2i(vi Vector2i) {
	v.X = float32(vi.X)
	v.Y = float32(vi.Y)
}

// SetDim sets the given vector component value by its dimension index.
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

// Dim returns the given vector component.
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

// SetPointDim sets the given dimension of the given [image.Point] to the given value.
func SetPointDim(pt *image.Point, dim Dims, value int) {
	switch dim {
	case X:
		pt.X = value
	case Y:
		pt.Y = value
	default:
		panic("dim is out of range")
	}
}

// PointDim returns the given dimension of the given [image.Point].
func PointDim(pt image.Point, dim Dims) int {
	switch dim {
	case X:
		return pt.X
	case Y:
		return pt.Y
	default:
		panic("dim is out of range")
	}
}

func (a Vector2) String() string {
	return fmt.Sprintf("(%v, %v)", a.X, a.Y)
}

// SetPoint sets the vector from the given [image.Point].
func (a *Vector2) SetPoint(pt image.Point) {
	a.X = float32(pt.X)
	a.Y = float32(pt.Y)
}

// SetFixed sets the vector from the given [fixed.Point26_6].
func (a *Vector2) SetFixed(pt fixed.Point26_6) {
	a.X = FromFixed(pt.X)
	a.Y = FromFixed(pt.Y)
}

// ToPoint returns the vector as an [image.Point].
func (a Vector2) ToPoint() image.Point {
	return image.Point{int(a.X), int(a.Y)}
}

// ToPointFloor returns the vector as an [image.Point] with all values [Floor]ed.
func (a Vector2) ToPointFloor() image.Point {
	return image.Point{int(Floor(a.X)), int(Floor(a.Y))}
}

// ToPointCeil returns the vector as an [image.Point] with all values [Ceil]ed.
func (a Vector2) ToPointCeil() image.Point {
	return image.Point{int(Ceil(a.X)), int(Ceil(a.Y))}
}

// ToPointRound returns the vector as an [image.Point] with all values [Round]ed.
func (a Vector2) ToPointRound() image.Point {
	return image.Point{int(Round(a.X)), int(Round(a.Y))}
}

// ToFixed returns the vector as a [fixed.Point26_6].
func (a Vector2) ToFixed() fixed.Point26_6 {
	return ToFixedPoint(a.X, a.Y)
}

// RectFromPosSizeMax returns an [image.Rectangle] from the floor of pos
// and ceil of size.
func RectFromPosSizeMax(pos, size Vector2) image.Rectangle {
	tp := pos.ToPointFloor()
	ts := size.ToPointCeil()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

// RectFromPosSizeMin returns an [image.Rectangle] from the ceil of pos
// and floor of size.
func RectFromPosSizeMin(pos, size Vector2) image.Rectangle {
	tp := pos.ToPointCeil()
	ts := size.ToPointFloor()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

// SetZero sets all of the vector's components to zero.
func (v *Vector2) SetZero() {
	v.SetScalar(0)
}

// FromSlice sets this vector's components from the given slice, starting at offset.
func (v *Vector2) FromSlice(slice []float32, offset int) {
	v.X = slice[offset]
	v.Y = slice[offset+1]
}

// ToSlice copies this vector's components to the given slice, starting at offset.
func (v Vector2) ToSlice(slice []float32, offset int) {
	slice[offset] = v.X
	slice[offset+1] = v.Y
}

// Basic math operations:

// Add adds the other given vector to this one and returns the result as a new vector.
func (v Vector2) Add(other Vector2) Vector2 {
	return Vec2(v.X+other.X, v.Y+other.Y)
}

// AddScalar adds scalar s to each component of this vector and returns new vector.
func (v Vector2) AddScalar(s float32) Vector2 {
	return Vec2(v.X+s, v.Y+s)
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
	return Vec2(v.X-other.X, v.Y-other.Y)
}

// SubScalar subtracts scalar s from each component of this vector and returns new vector.
func (v Vector2) SubScalar(s float32) Vector2 {
	return Vec2(v.X-s, v.Y-s)
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
	return Vec2(v.X*other.X, v.Y*other.Y)
}

// MulScalar multiplies each component of this vector by the scalar s and returns resulting vector.
func (v Vector2) MulScalar(s float32) Vector2 {
	return Vec2(v.X*s, v.Y*s)
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
	return Vec2(v.X/other.X, v.Y/other.Y)
}

// DivScalar divides each component of this vector by the scalar s and returns resulting vector.
// If scalar is zero, returns zero.
func (v Vector2) DivScalar(scalar float32) Vector2 {
	if scalar != 0 {
		return v.MulScalar(1 / scalar)
	}
	return Vector2{}
}

// SetDiv sets this to division by other vector (i.e., /= or divide-equals).
func (v *Vector2) SetDiv(other Vector2) {
	v.X /= other.X
	v.Y /= other.Y
}

// SetDivScalar sets this to division by scalar.
func (v *Vector2) SetDivScalar(scalar float32) {
	if scalar != 0 {
		v.SetMulScalar(1 / scalar)
	} else {
		v.SetZero()
	}
}

// Abs returns the vector with [Abs] applied to each component.
func (v Vector2) Abs() Vector2 {
	return Vec2(Abs(v.X), Abs(v.Y))
}

// Min returns min of this vector components vs. other vector.
func (v Vector2) Min(other Vector2) Vector2 {
	return Vec2(Min(v.X, other.X), Min(v.Y, other.Y))
}

// SetMin sets this vector components to the minimum values of itself and other vector.
func (v *Vector2) SetMin(other Vector2) {
	v.X = Min(v.X, other.X)
	v.Y = Min(v.Y, other.Y)
}

// Max returns max of this vector components vs. other vector.
func (v Vector2) Max(other Vector2) Vector2 {
	return Vec2(Max(v.X, other.X), Max(v.Y, other.Y))
}

// SetMax sets this vector components to the maximum value of itself and other vector.
func (v *Vector2) SetMax(other Vector2) {
	v.X = Max(v.X, other.X)
	v.Y = Max(v.Y, other.Y)
}

// Clamp sets this vector's components to be no less than the corresponding
// components of min and not greater than the corresponding component of max.
// Assumes min < max; if this assumption isn't true, it will not operate correctly.
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

// Floor returns this vector with [Floor] applied to each of its components.
func (v Vector2) Floor() Vector2 {
	return Vec2(Floor(v.X), Floor(v.Y))
}

// Ceil returns this vector with [Ceil] applied to each of its components.
func (v Vector2) Ceil() Vector2 {
	return Vec2(Ceil(v.X), Ceil(v.Y))
}

// Round returns this vector with [Round] applied to each of its components.
func (v Vector2) Round() Vector2 {
	return Vec2(Round(v.X), Round(v.Y))
}

// Negate returns the vector with each component negated.
func (v Vector2) Negate() Vector2 {
	return Vec2(-v.X, -v.Y)
}

// AddDim returns the vector with the given value added on the given dimension.
func (a Vector2) AddDim(d Dims, value float32) Vector2 {
	switch d {
	case X:
		a.X += value
	case Y:
		a.Y += value
	}
	return a
}

// SubDim returns the vector with the given value subtracted on the given dimension.
func (a Vector2) SubDim(d Dims, value float32) Vector2 {
	switch d {
	case X:
		a.X -= value
	case Y:
		a.Y -= value
	}
	return a
}

// MulDim returns the vector with the given value multiplied by on the given dimension.
func (a Vector2) MulDim(d Dims, value float32) Vector2 {
	switch d {
	case X:
		a.X *= value
	case Y:
		a.Y *= value
	}
	return a
}

// DivDim returns the vector with the given value divided by on the given dimension.
func (a Vector2) DivDim(d Dims, value float32) Vector2 {
	switch d {
	case X:
		a.X /= value
	case Y:
		a.Y /= value
	}
	return a
}

// Distance, Normal:

// Dot returns the dot product of this vector with the given other vector.
func (v Vector2) Dot(other Vector2) float32 {
	return v.X*other.X + v.Y*other.Y
}

// Length returns the length (magnitude) of this vector.
func (v Vector2) Length() float32 {
	return Sqrt(v.LengthSquared())
}

// LengthSquared returns the length squared of this vector.
// LengthSquared can be used to compare the lengths of vectors
// without the need to perform a square root.
func (v Vector2) LengthSquared() float32 {
	return v.X*v.X + v.Y*v.Y
}

// Normal returns this vector divided by its length (its unit vector).
func (v Vector2) Normal() Vector2 {
	l := v.Length()
	if l == 0 {
		return Vector2{}
	}
	return v.DivScalar(l)
}

// DistanceTo returns the distance between these two vectors as points.
func (v Vector2) DistanceTo(other Vector2) float32 {
	return Sqrt(v.DistanceToSquared(other))
}

// DistanceToSquared returns the squared distance between these two vectors as points.
func (v Vector2) DistanceToSquared(other Vector2) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	return dx*dx + dy*dy
}

// Cross returns the cross product of this vector with other.
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
	return Vec2(v.X+(other.X-v.X)*alpha, v.Y+(other.Y-v.Y)*alpha)
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

// Rot90CW rotates the line OP by 90 degrees CW.
func (v Vector2) Rot90CW() Vector2 {
	return Vector2{v.Y, -v.X}
}

// Rot90CCW rotates the line OP by 90 degrees CCW.
func (v Vector2) Rot90CCW() Vector2 {
	return Vector2{-v.Y, v.X}
}

// Rot rotates the line OP by phi radians CCW.
func (v Vector2) Rot(phi float32, p0 Vector2) Vector2 {
	sinphi, cosphi := math32.Sincos(phi)
	return Vector2{
		p0.X + cosphi*(v.X-p0.X) - sinphi*(v.Y-p0.Y),
		p0.Y + sinphi*(v.X-p0.X) + cosphi*(v.Y-p0.Y),
	}
}
