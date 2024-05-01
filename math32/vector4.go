// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit Cogent Core functionality.

package math32

import "fmt"

// Vector4 is a vector/point in homogeneous coordinates with X, Y, Z and W components.
type Vector4 struct {
	X float32
	Y float32
	Z float32
	W float32
}

// Vec4 returns a new [Vector4] with the given x, y, z, and w components.
func Vec4(x, y, z, w float32) Vector4 {
	return Vector4{X: x, Y: y, Z: z, W: w}
}

// Vector4Scalar returns a new [Vector4] with all components set to the given scalar value.
func Vector4Scalar(s float32) Vector4 {
	return Vector4{X: s, Y: s, Z: s, W: s}
}

// Vector4FromVector3 returns a new [Vector4] from the given [Vector3] and w component.
func Vector4FromVector3(v Vector3, w float32) Vector4 {
	nv := Vector4{}
	nv.SetFromVector3(v, w)
	return nv
}

// Set sets this vector X, Y, Z and W components.
func (v *Vector4) Set(x, y, z, w float32) {
	v.X = x
	v.Y = y
	v.Z = z
	v.W = w
}

// SetFromVector3 sets this vector from a Vector3 and W
func (v *Vector4) SetFromVector3(other Vector3, w float32) {
	v.X = other.X
	v.Y = other.Y
	v.Z = other.Z
	v.W = w
}

// SetFromVector2 sets this vector from a Vector2 with 0,1 for Z,W
func (v *Vector4) SetFromVector2(other Vector2) {
	v.X = other.X
	v.Y = other.Y
	v.Z = 0
	v.W = 1
}

// SetDim sets this vector component value by dimension index.
func (v *Vector4) SetDim(dim Dims, value float32) {
	switch dim {
	case X:
		v.X = value
	case Y:
		v.Y = value
	case Z:
		v.Z = value
	case W:
		v.W = value
	default:
		panic("dim is out of range")
	}
}

// Dim returns this vector component.
func (v Vector4) Dim(dim Dims) float32 {
	switch dim {
	case X:
		return v.X
	case Y:
		return v.Y
	case Z:
		return v.Z
	case W:
		return v.W
	default:
		panic("dim is out of range")
	}
}

func (v Vector4) String() string {
	return fmt.Sprintf("(%v, %v, %v, %v)", v.X, v.Y, v.Z, v.W)
}

// SetZero sets all of the vector's components to zero.
func (v *Vector4) SetZero() {
	v.X = 0
	v.Y = 0
	v.Z = 0
	v.W = 1
}

// FromSlice sets this vector's components from the given slice, starting at offset.
func (v *Vector4) FromSlice(array []float32, offset int) {
	v.X = array[offset]
	v.Y = array[offset+1]
	v.Z = array[offset+2]
	v.W = array[offset+3]
}

// ToSlice copies this vector's components to the given slice, starting at offset.
func (v Vector4) ToSlice(array []float32, offset int) {
	array[offset] = v.X
	array[offset+1] = v.Y
	array[offset+2] = v.Z
	array[offset+3] = v.W
}

// Basic math operations:

// Add adds the other given vector to this one and returns the result as a new vector.
func (v Vector4) Add(other Vector4) Vector4 {
	return Vector4{v.X + other.X, v.Y + other.Y, v.Z + other.Z, v.W + other.W}
}

// AddScalar adds scalar s to each component of this vector and returns new vector.
func (v Vector4) AddScalar(s float32) Vector4 {
	return Vector4{v.X + s, v.Y + s, v.Z + s, v.W + s}
}

// SetAdd sets this to addition with other vector (i.e., += or plus-equals).
func (v *Vector4) SetAdd(other Vector4) {
	v.X += other.X
	v.Y += other.Y
	v.Z += other.Z
	v.W += other.W
}

// SetAddScalar sets this to addition with scalar.
func (v *Vector4) SetAddScalar(s float32) {
	v.X += s
	v.Y += s
	v.Z += s
	v.W += s
}

// Sub subtracts other vector from this one and returns result in new vector.
func (v Vector4) Sub(other Vector4) Vector4 {
	return Vector4{v.X - other.X, v.Y - other.Y, v.Z - other.Z, v.W - other.W}
}

// SubScalar subtracts scalar s from each component of this vector and returns new vector.
func (v Vector4) SubScalar(s float32) Vector4 {
	return Vector4{v.X - s, v.Y - s, v.Z - s, v.W - s}
}

// SetSub sets this to subtraction with other vector (i.e., -= or minus-equals).
func (v *Vector4) SetSub(other Vector4) {
	v.X -= other.X
	v.Y -= other.Y
	v.Z -= other.Z
	v.W -= other.W
}

// SetSubScalar sets this to subtraction of scalar.
func (v *Vector4) SetSubScalar(s float32) {
	v.X -= s
	v.Y -= s
	v.Z -= s
	v.W -= s
}

// Mul multiplies each component of this vector by the corresponding one from other
// and returns resulting vector.
func (v Vector4) Mul(other Vector4) Vector4 {
	return Vector4{v.X * other.X, v.Y * other.Y, v.Z * other.Z, v.W * other.W}
}

// MulScalar multiplies each component of this vector by the scalar s and returns resulting vector.
func (v Vector4) MulScalar(s float32) Vector4 {
	return Vector4{v.X * s, v.Y * s, v.Z * s, v.W * s}
}

// SetMul sets this to multiplication with other vector (i.e., *= or times-equals).
func (v *Vector4) SetMul(other Vector4) {
	v.X *= other.X
	v.Y *= other.Y
	v.Z *= other.Z
	v.W *= other.W
}

// SetMulScalar sets this to multiplication by scalar.
func (v *Vector4) SetMulScalar(s float32) {
	v.X *= s
	v.Y *= s
	v.Z *= s
	v.W *= s
}

// Div divides each component of this vector by the corresponding one from other vector
// and returns resulting vector.
func (v Vector4) Div(other Vector4) Vector4 {
	return Vector4{v.X / other.X, v.Y / other.Y, v.Z / other.Z, v.W / other.W}
}

// DivScalar divides each component of this vector by the scalar s and returns resulting vector.
// If scalar is zero, returns zero.
func (v Vector4) DivScalar(scalar float32) Vector4 {
	if scalar != 0 {
		return v.MulScalar(1 / scalar)
	} else {
		return Vector4{}
	}
}

// SetDiv sets this to division by other vector (i.e., /= or divide-equals).
func (v *Vector4) SetDiv(other Vector4) {
	v.X /= other.X
	v.Y /= other.Y
	v.Z /= other.Z
	v.W /= other.W
}

// SetDivScalar sets this to division by scalar.
func (v *Vector4) SetDivScalar(s float32) {
	if s != 0 {
		v.SetMulScalar(1 / s)
	} else {
		v.SetZero()
	}
}

// Min returns min of this vector components vs. other vector.
func (v Vector4) Min(other Vector4) Vector4 {
	return Vector4{Min(v.X, other.X), Min(v.Y, other.Y), Min(v.Z, other.Z), Min(v.W, other.W)}
}

// SetMin sets this vector components to the minimum values of itself and other vector.
func (v *Vector4) SetMin(other Vector4) {
	v.X = Min(v.X, other.X)
	v.Y = Min(v.Y, other.Y)
	v.Z = Min(v.Z, other.Z)
	v.W = Min(v.W, other.W)
}

// Max returns max of this vector components vs. other vector.
func (v Vector4) Max(other Vector4) Vector4 {
	return Vector4{Max(v.X, other.X), Max(v.Y, other.Y), Max(v.Z, other.Z), Max(v.W, other.W)}
}

// SetMax sets this vector components to the maximum value of itself and other vector.
func (v *Vector4) SetMax(other Vector4) {
	v.X = Max(v.X, other.X)
	v.Y = Max(v.Y, other.Y)
	v.Z = Max(v.Z, other.Z)
	v.W = Max(v.W, other.W)
}

// Clamp sets this vector's components to be no less than the corresponding
// components of min and not greater than the corresponding component of max.
// Assumes min < max; if this assumption isn't true, it will not operate correctly.
func (v *Vector4) Clamp(min, max Vector4) {
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
	if v.W < min.W {
		v.W = min.W
	} else if v.W > max.W {
		v.W = max.W
	}
}

// Floor returns this vector with [Floor] applied to each of its components.
func (v Vector4) Floor() Vector4 {
	return Vector4{Floor(v.X), Floor(v.Y), Floor(v.Z), Floor(v.W)}
}

// Ceil returns this vector with [Ceil] applied to each of its components.
func (v Vector4) Ceil() Vector4 {
	return Vector4{Ceil(v.X), Ceil(v.Y), Ceil(v.Z), Ceil(v.W)}
}

// Round returns this vector with [Round] applied to each of its components.
func (v Vector4) Round() Vector4 {
	return Vector4{Round(v.X), Round(v.Y), Round(v.Z), Round(v.W)}
}

// Negate returns the vector with each component negated.
func (v Vector4) Negate() Vector4 {
	return Vector4{-v.X, -v.Y, -v.Z, -v.W}
}

// Distance, Normal:

// Dot returns the dot product of this vector with the given other vector.
func (v Vector4) Dot(other Vector4) float32 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z + v.W*other.W
}

// Length returns the length (magnitude) of this vector.
func (v Vector4) Length() float32 {
	return Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z + v.W*v.W)
}

// LengthSquared returns the length squared of this vector.
// LengthSquared can be used to compare the lengths of vectors
// without the need to perform a square root.
func (v Vector4) LengthSquared() float32 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z + v.W*v.W
}

// Normal returns this vector divided by its length (its unit vector).
func (v Vector4) Normal() Vector4 {
	return v.DivScalar(v.Length())
}

// SetNormal normalizes this vector so its length will be 1.
func (v *Vector4) SetNormal() {
	v.SetDivScalar(v.Length())
}

// Lerp returns vector with each components as the linear interpolated value of
// alpha between itself and the corresponding other component.
func (v Vector4) Lerp(other Vector4, alpha float32) Vector4 {
	return Vector4{v.X + (other.X-v.X)*alpha, v.Y + (other.Y-v.Y)*alpha, v.Z + (other.Z-v.Z)*alpha,
		v.W + (other.W-v.W)*alpha}
}

// Matrix operations:

// MulMatrix4 returns vector multiplied by specified 4x4 matrix.
func (v Vector4) MulMatrix4(m *Matrix4) Vector4 {
	return Vector4{m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]*v.W,
		m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]*v.W,
		m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]*v.W,
		m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]*v.W}
}

// SetAxisAngleFromQuat set this vector to be the axis (x, y, z) and angle (w)
// of a rotation specified the quaternion q.
// Assumes q is normalized.
func (v *Vector4) SetAxisAngleFromQuat(q Quat) {
	// http://www.euclideanspace.com/maths/geometry/rotations/conversions/quaternionToAngle/index.htm
	qw := Clamp(q.W, -1, 1)
	v.W = 2 * Acos(qw)
	s := Sqrt(1 - qw*qw)
	if s < 0.0001 {
		v.X = 1
		v.Y = 0
		v.Z = 0
	} else {
		v.X = q.X / s
		v.Y = q.Y / s
		v.Z = q.Z / s
	}
}

// SetAxisFromRotationMatrix sets this vector to be the axis (x, y, z) and angle (w)
// of a rotation specified the matrix m.
// Assumes the upper 3x3 of m is a pure rotation matrix (i.e, unscaled).
func (v *Vector4) SetAxisFromRotationMatrix(m *Matrix4) {
	// http://www.euclideanspace.com/maths/geometry/rotations/conversions/matrixToAngle/index.htm
	var angle, x, y, z float32 // variables for result
	var epsilon float32 = 0.01 // margin to allow for rounding errors
	var epsilon2 float32 = 0.1 // margin to distinguish between 0 and 180 degrees

	m11 := m[0]
	m12 := m[4]
	m13 := m[8]
	m21 := m[1]
	m22 := m[5]
	m23 := m[9]
	m31 := m[2]
	m32 := m[6]
	m33 := m[10]

	if (Abs(m12-m21) < epsilon) && (Abs(m13-m31) < epsilon) && (Abs(m23-m32) < epsilon) {
		// singularity found
		// first check for identity matrix which must have +1 for all terms
		// in leading diagonal and zero in other terms

		if (Abs(m12+m21) < epsilon2) && (Abs(m13+m31) < epsilon2) && (Abs(m23+m32) < epsilon2) && (Abs(m11+m22+m33-3) < epsilon2) {
			// v singularity is identity matrix so angle = 0
			v.Set(1, 0, 0, 0)
		}
		// otherwise this singularity is angle = 180
		angle = Pi

		var xx = (m11 + 1) / 2
		var yy = (m22 + 1) / 2
		var zz = (m33 + 1) / 2
		var xy = (m12 + m21) / 4
		var xz = (m13 + m31) / 4
		var yz = (m23 + m32) / 4

		if (xx > yy) && (xx > zz) { // m11 is the largest diagonal term
			if xx < epsilon {
				x = 0
				y = 0.707106781
				z = 0.707106781
			} else {
				x = Sqrt(xx)
				y = xy / x
				z = xz / x
			}
		} else if yy > zz { // m22 is the largest diagonal term
			if yy < epsilon {
				x = 0.707106781
				y = 0
				z = 0.707106781
			} else {
				y = Sqrt(yy)
				x = xy / y
				z = yz / y
			}
		} else { // m33 is the largest diagonal term so base result on this
			if zz < epsilon {
				x = 0.707106781
				y = 0.707106781
				z = 0
			} else {
				z = Sqrt(zz)
				x = xz / z
				y = yz / z
			}
		}

		v.Set(x, y, z, angle)
	}

	// as we have reached here there are no singularities so we can handle normally
	s := Sqrt((m32-m23)*(m32-m23) + (m13-m31)*(m13-m31) + (m21-m12)*(m21-m12)) // used to normalize

	if Abs(s) < 0.001 {
		s = 1
	}

	// prevent divide by zero, should not happen if matrix is orthogonal and should be
	// caught by singularity test above, but I've left it in just in case

	v.X = (m32 - m23) / s
	v.Y = (m13 - m31) / s
	v.Z = (m21 - m12) / s
	v.W = Acos((m11 + m22 + m33 - 1) / 2)
}

// PerspDiv returns the 3-vector of normalized display coordinates (NDC) from given 4-vector
// By dividing by the 4th W component
func (v Vector4) PerspDiv() Vector3 {
	return Vec3(v.X/v.W, v.Y/v.W, v.Z/v.W)
}
