// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Vec2i is a 2D vector/point with X and Y integer32 components.
type Vec2i struct {
	X int32
	Y int32
}

// NewVec2i returns a new Vec2i with the specified x and y components.
func NewVec2i(x, y int32) Vec2i {
	return Vec2i{X: x, Y: y}
}

// NewVec2iScalar returns a new Vec2i with all components set to scalar.
func NewVec2iScalar(s int32) Vec2i {
	return Vec2i{X: s, Y: s}
}

// IsNil returns true if all values are 0 (uninitialized).
func (v Vec2i) IsNil() bool {
	if v.X == 0 && v.Y == 0 {
		return true
	}
	return false
}

// Set sets this vector X and Y components.
func (v *Vec2i) Set(x, y int32) {
	v.X = x
	v.Y = y
}

// SetScalar sets all vector components to same scalar value.
func (v *Vec2i) SetScalar(s int32) {
	v.X = s
	v.Y = s
}

// SetFromVec2 sets from a Vec2 (float32) vector.
func (v *Vec2i) SetFromVec3(vf Vec2) {
	v.X = int32(vf.X)
	v.Y = int32(vf.Y)
}

// SetDim sets this vector component value by its dimension index
func (v *Vec2i) SetDim(dim Dims, value int32) {
	switch dim {
	case X:
		v.X = value
	case Y:
		v.Y = value
	default:
		panic("dim is out of range")
	}
}

// Dim returns this vector component
func (v Vec2i) Dim(dim Dims) int32 {
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
func (v *Vec2i) SetByName(name string, value int32) {
	switch name {
	case "x", "X":
		v.X = value
	case "y", "Y":
		v.Y = value
	default:
		panic("Invalid Vec2i component name: " + name)
	}
}

// SetZero sets this vector X and Y components to be zero.
func (v *Vec2i) SetZero() {
	v.SetScalar(0)
}

// FromArray sets this vector's components from the specified array and offset.
func (v *Vec2i) FromArray(array []int32, offset int) {
	v.X = array[offset]
	v.Y = array[offset+1]
}

// ToArray copies this vector's components to array starting at offset.
func (v Vec2i) ToArray(array []int32, offset int) {
	array[offset] = v.X
	array[offset+1] = v.Y
}

///////////////////////////////////////////////////////////////////////
//  Basic math operations

// Add adds other vector to this one and returns result in a new vector.
func (v Vec2i) Add(other Vec2i) Vec2i {
	return Vec2i{v.X + other.X, v.Y + other.Y}
}

// AddScalar adds scalar s to each component of this vector and returns new vector.
func (v Vec2i) AddScalar(s int32) Vec2i {
	return Vec2i{v.X + s, v.Y + s}
}

// SetAdd sets this to addition with other vector (i.e., += or plus-equals).
func (v *Vec2i) SetAdd(other Vec2i) {
	v.X += other.X
	v.Y += other.Y
}

// SetAddScalar sets this to addition with scalar.
func (v *Vec2i) SetAddScalar(s int32) {
	v.X += s
	v.Y += s
}

// Sub subtracts other vector from this one and returns result in new vector.
func (v Vec2i) Sub(other Vec2i) Vec2i {
	return Vec2i{v.X - other.X, v.Y - other.Y}
}

// SubScalar subtracts scalar s from each component of this vector and returns new vector.
func (v Vec2i) SubScalar(s int32) Vec2i {
	return Vec2i{v.X - s, v.Y - s}
}

// SetSub sets this to subtraction with other vector (i.e., -= or minus-equals).
func (v *Vec2i) SetSub(other Vec2i) {
	v.X -= other.X
	v.Y -= other.Y
}

// SetSubScalar sets this to subtraction of scalar.
func (v *Vec2i) SetSubScalar(s int32) {
	v.X -= s
	v.Y -= s
}

// Mul multiplies each component of this vector by the corresponding one from other
// and returns resulting vector.
func (v Vec2i) Mul(other Vec2i) Vec2i {
	return Vec2i{v.X * other.X, v.Y * other.Y}
}

// MulScalar multiplies each component of this vector by the scalar s and returns resulting vector.
func (v Vec2i) MulScalar(s int32) Vec2i {
	return Vec2i{v.X * s, v.Y * s}
}

// SetMul sets this to multiplication with other vector (i.e., *= or times-equals).
func (v *Vec2i) SetMul(other Vec2i) {
	v.X *= other.X
	v.Y *= other.Y
}

// SetMulScalar sets this to multiplication by scalar.
func (v *Vec2i) SetMulScalar(s int32) {
	v.X *= s
	v.Y *= s
}

// Div divides each component of this vector by the corresponding one from other vector
// and returns resulting vector.
func (v Vec2i) Div(other Vec2i) Vec2i {
	return Vec2i{v.X / other.X, v.Y / other.Y}
}

// DivScalar divides each component of this vector by the scalar s and returns resulting vector.
// If scalar is zero, returns zero.
func (v Vec2i) DivScalar(scalar int32) Vec2i {
	if scalar != 0 {
		return v.MulScalar(1 / scalar)
	} else {
		return Vec2i{}
	}
}

// SetDiv sets this to division by other vector (i.e., /= or divide-equals).
func (v *Vec2i) SetDiv(other Vec2i) {
	v.X /= other.X
	v.Y /= other.Y
}

// SetDivScalar sets this to division by scalar.
func (v *Vec2i) SetDivScalar(s int32) {
	if s != 0 {
		v.SetMulScalar(1 / s)
	} else {
		v.SetZero()
	}
}

// Min32i returns the min of the two int32 numbers.
func Min32i(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

// Max32i returns the max of the two int32 numbers.
func Max32i(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// Min returns min of this vector components vs. other vector.
func (v Vec2i) Min(other Vec2i) Vec2i {
	return Vec2i{Min32i(v.X, other.X), Min32i(v.Y, other.Y)}
}

// SetMin sets this vector components to the minimum values of itself and other vector.
func (v *Vec2i) SetMin(other Vec2i) {
	v.X = Min32i(v.X, other.X)
	v.Y = Min32i(v.Y, other.Y)
}

// Max returns max of this vector components vs. other vector.
func (v Vec2i) Max(other Vec2i) Vec2i {
	return Vec2i{Max32i(v.X, other.X), Max32i(v.Y, other.Y)}
}

// SetMax sets this vector components to the maximum value of itself and other vector.
func (v *Vec2i) SetMax(other Vec2i) {
	v.X = Max32i(v.X, other.X)
	v.Y = Max32i(v.Y, other.Y)
}

// Clamp sets this vector components to be no less than the corresponding components of min
// and not greater than the corresponding component of max.
// Assumes min < max, if this assumption isn't true it will not operate correctly.
func (v *Vec2i) Clamp(min, max Vec2i) {
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
func (v *Vec2i) ClampScalar(minVal, maxVal int32) {
	v.Clamp(NewVec2iScalar(minVal), NewVec2iScalar(maxVal))
}

// Negate returns vector with each component negated.
func (v Vec2i) Negate() Vec2i {
	return Vec2i{-v.X, -v.Y}
}

// SetNegate negates each of this vector's components.
func (v *Vec2i) SetNegate() {
	v.X = -v.X
	v.Y = -v.Y
}

// IsEqual returns if this vector is equal to other.
func (v Vec2i) IsEqual(other Vec2i) bool {
	return (other.X == v.X) && (other.Y == v.Y)
}
