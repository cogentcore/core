// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Vec3i is a 3D vector/point with X, Y and Z int32 components.
type Vec3i struct {
	X int32
	Y int32
	Z int32
}

// NewVec3i returns a new Vec3i with the specified x, y and y components.
func NewVec3i(x, y, z int32) Vec3i {
	return Vec3i{X: x, Y: y, Z: z}
}

// NewVec3iScalar returns a new Vec3 with all components set to scalar.
func NewVec3iScalar(s int32) Vec3i {
	return Vec3i{X: s, Y: s, Z: s}
}

// IsNil returns true if all values are 0 (uninitialized).
func (v Vec3i) IsNil() bool {
	if v.X == 0 && v.Y == 0 && v.Z == 0 {
		return true
	}
	return false
}

// Set sets this vector X, Y and Z components.
func (v *Vec3i) Set(x, y, z int32) {
	v.X = x
	v.Y = y
	v.Z = z
}

// SetScalar sets all vector X, Y and Z components to same scalar value.
func (v *Vec3i) SetScalar(s int32) {
	v.X = s
	v.Y = s
	v.Z = s
}

// SetFromVec3 sets from a Vec3 (float32) vector.
func (v *Vec3i) SetFromVec3(vf Vec3) {
	v.X = int32(vf.X)
	v.Y = int32(vf.Y)
	v.Z = int32(vf.Z)
}

// SetComponent sets this vector component value by component index.
// Returns the pointer to this updated vector
func (v *Vec3i) SetComponent(comp Components, value int32) {
	switch comp {
	case X:
		v.X = value
	case Y:
		v.Y = value
	case Z:
		v.Z = value
	default:
		panic("component is out of range: ")
	}
}

// Component returns this vector component
func (v Vec3i) Component(comp Components) int32 {
	switch comp {
	case X:
		return v.X
	case Y:
		return v.Y
	case Z:
		return v.Z
	default:
		panic("component is out of range")
	}
}

// SetByName sets this vector component value by its case insensitive name: "x", "y", or "z".
func (v *Vec3i) SetByName(name string, value int32) {
	switch name {
	case "x", "X":
		v.X = value
	case "y", "Y":
		v.Y = value
	case "z", "Z":
		v.Z = value
	default:
		panic("Invalid Vec3i component name: " + name)
	}
}

// SetZero sets this vector X, Y and Z components to be zero.
func (v *Vec3i) SetZero() {
	v.SetScalar(0)
}

// FromArray sets this vector's components from the specified array and offset
func (v *Vec3i) FromArray(array []int32, offset int) {
	v.X = array[offset]
	v.Y = array[offset+1]
	v.Z = array[offset+2]
}

// ToArray copies this vector's components to array starting at offset.
func (v Vec3i) ToArray(array []int32, offset int) {
	array[offset] = v.X
	array[offset+1] = v.Y
	array[offset+2] = v.Z
}

///////////////////////////////////////////////////////////////////////
//  Basic math operations

// Add adds other vector to this one and returns result in a new vector.
func (v Vec3i) Add(other Vec3i) Vec3i {
	return Vec3i{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// AddScalar adds scalar s to each component of this vector and returns new vector.
func (v Vec3i) AddScalar(s int32) Vec3i {
	return Vec3i{v.X + s, v.Y + s, v.Z + s}
}

// SetAdd sets this to addition with other vector (i.e., += or plus-equals).
func (v *Vec3i) SetAdd(other Vec3i) {
	v.X += other.X
	v.Y += other.Y
	v.Z += other.Z
}

// SetAddScalar sets this to addition with scalar.
func (v *Vec3i) SetAddScalar(s int32) {
	v.X += s
	v.Y += s
	v.Z += s
}

// Sub subtracts other vector from this one and returns result in new vector.
func (v Vec3i) Sub(other Vec3i) Vec3i {
	return Vec3i{v.X - other.X, v.Y - other.Y, v.Z - other.Z}
}

// SubScalar subtracts scalar s from each component of this vector and returns new vector.
func (v Vec3i) SubScalar(s int32) Vec3i {
	return Vec3i{v.X - s, v.Y - s, v.Z - s}
}

// SetSub sets this to subtraction with other vector (i.e., -= or minus-equals).
func (v *Vec3i) SetSub(other Vec3i) {
	v.X -= other.X
	v.Y -= other.Y
	v.Z -= other.Z
}

// SetSubScalar sets this to subtraction of scalar.
func (v *Vec3i) SetSubScalar(s int32) {
	v.X -= s
	v.Y -= s
	v.Z -= s
}

// Mul multiplies each component of this vector by the corresponding one from other
// and returns resulting vector.
func (v Vec3i) Mul(other Vec3i) Vec3i {
	return Vec3i{v.X * other.X, v.Y * other.Y, v.Z * other.Z}
}

// MulScalar multiplies each component of this vector by the scalar s and returns resulting vector.
func (v Vec3i) MulScalar(s int32) Vec3i {
	return Vec3i{v.X * s, v.Y * s, v.Z * s}
}

// SetMul sets this to multiplication with other vector (i.e., *= or times-equals).
func (v *Vec3i) SetMul(other Vec3i) {
	v.X *= other.X
	v.Y *= other.Y
	v.Z *= other.Z
}

// SetMulScalar sets this to multiplication by scalar.
func (v *Vec3i) SetMulScalar(s int32) {
	v.X *= s
	v.Y *= s
	v.Z *= s
}

// Div divides each component of this vector by the corresponding one from other vector
// and returns resulting vector.
func (v Vec3i) Div(other Vec3i) Vec3i {
	return Vec3i{v.X / other.X, v.Y / other.Y, v.Z / other.Z}
}

// DivScalar divides each component of this vector by the scalar s and returns resulting vector.
// If scalar is zero, returns zero.
func (v Vec3i) DivScalar(scalar int32) Vec3i {
	if scalar != 0 {
		return v.MulScalar(1 / scalar)
	} else {
		return Vec3i{}
	}
}

// SetDiv sets this to division by other vector (i.e., /= or divide-equals).
func (v *Vec3i) SetDiv(other Vec3i) {
	v.X /= other.X
	v.Y /= other.Y
	v.Z /= other.Z
}

// SetDivScalar sets this to division by scalar.
func (v *Vec3i) SetDivScalar(s int32) {
	if s != 0 {
		v.SetMulScalar(1 / s)
	} else {
		v.SetZero()
	}
}

// Min returns min of this vector components vs. other vector.
func (v Vec3i) Min(other Vec3i) Vec3i {
	return Vec3i{Min32i(v.X, other.X), Min32i(v.Y, other.Y), Min32i(v.Z, other.Z)}
}

// SetMin sets this vector components to the minimum values of itself and other vector.
func (v *Vec3i) SetMin(other Vec3i) {
	v.X = Min32i(v.X, other.X)
	v.Y = Min32i(v.Y, other.Y)
	v.Z = Min32i(v.Z, other.Z)
}

// Max returns max of this vector components vs. other vector.
func (v Vec3i) Max(other Vec3i) Vec3i {
	return Vec3i{Max32i(v.X, other.X), Max32i(v.Y, other.Y), Max32i(v.Z, other.Z)}
}

// SetMax sets this vector components to the maximum value of itself and other vector.
func (v *Vec3i) SetMax(other Vec3i) {
	v.X = Max32i(v.X, other.X)
	v.Y = Max32i(v.Y, other.Y)
	v.Z = Max32i(v.Z, other.Z)
}

// Clamp sets this vector components to be no less than the corresponding components of min
// and not greater than the corresponding component of max.
// Assumes min < max, if this assumption isn't true it will not operate correctly.
func (v *Vec3i) Clamp(min, max Vec3i) {
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
	if v.Z < min.Z {
		v.Z = min.Z
	} else if v.Z > max.Z {
		v.Z = max.Z
	}
}

// ClampScalar sets this vector components to be no less than minVal and not greater than maxVal.
func (v *Vec3i) ClampScalar(minVal, maxVal int32) {
	v.Clamp(NewVec3iScalar(minVal), NewVec3iScalar(maxVal))
}

// Negate returns vector with each component negated.
func (v Vec3i) Negate() Vec3i {
	return Vec3i{-v.X, -v.Y, -v.Z}
}

// SetNegate negates each of this vector's components.
func (v *Vec3i) SetNegate() {
	v.X = -v.X
	v.Y = -v.Y
	v.Z = -v.Z
}

// IsEqual returns if this vector is equal to other.
func (v Vec3i) IsEqual(other Vec3i) bool {
	return (other.X == v.X) && (other.Y == v.Y) && (other.Z == v.Z)
}
