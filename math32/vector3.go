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

// Vector3 is a 3D vector/point with X, Y and Z components.
type Vector3 struct {
	X float32
	Y float32
	Z float32
}

// Vec3 returns a new [Vector3] with the given x, y and z components.
func Vec3(x, y, z float32) Vector3 {
	return Vector3{x, y, z}
}

// Vector3Scalar returns a new [Vector3] with all components set to the given scalar value.
func Vector3Scalar(scalar float32) Vector3 {
	return Vector3{scalar, scalar, scalar}
}

// Vector3FromVector4 returns a new [Vector3] from the given [Vector4].
func Vector3FromVector4(v Vector4) Vector3 {
	nv := Vector3{}
	nv.SetFromVector4(v)
	return nv
}

// Set sets this vector X, Y and Z components.
func (v *Vector3) Set(x, y, z float32) {
	v.X = x
	v.Y = y
	v.Z = z
}

// SetScalar sets all vector components to the same scalar value.
func (v *Vector3) SetScalar(scalar float32) {
	v.X = scalar
	v.Y = scalar
	v.Z = scalar
}

// SetFromVector4 sets this vector from a Vector4
func (v *Vector3) SetFromVector4(other Vector4) {
	v.X = other.X
	v.Y = other.Y
	v.Z = other.Z
}

// SetFromVector3i sets from a Vector3i (int32) vector.
func (v *Vector3) SetFromVector3i(vi Vector3i) {
	v.X = float32(vi.X)
	v.Y = float32(vi.Y)
	v.Z = float32(vi.Z)
}

// SetDim sets this vector component value by dimension index.
func (v *Vector3) SetDim(dim Dims, value float32) {
	switch dim {
	case X:
		v.X = value
	case Y:
		v.Y = value
	case Z:
		v.Z = value
	default:
		panic("dim is out of range: ")
	}
}

// Dim returns this vector component
func (v Vector3) Dim(dim Dims) float32 {
	switch dim {
	case X:
		return v.X
	case Y:
		return v.Y
	case Z:
		return v.Z
	default:
		panic("dim is out of range")
	}
}

func (a Vector3) String() string {
	return fmt.Sprintf("(%v, %v, %v)", a.X, a.Y, a.Z)
}

// GenGoSet returns code to set values in object at given path (var.member etc).
func (v *Vector3) GenGoSet(path string) string {
	return fmt.Sprintf("%s.Set(%g, %g, %g)", path, v.X, v.Y, v.Z)
}

// SetZero sets all of the vector's components to zero.
func (v *Vector3) SetZero() {
	v.SetScalar(0)
}

// FromSlice sets this vector's components from the given slice, starting at offset.
func (v *Vector3) FromSlice(array []float32, offset int) {
	v.X = array[offset]
	v.Y = array[offset+1]
	v.Z = array[offset+2]
}

// ToSlice copies this vector's components to the given slice, starting at offset.
func (v Vector3) ToSlice(array []float32, offset int) {
	array[offset] = v.X
	array[offset+1] = v.Y
	array[offset+2] = v.Z
}

// Basic math operations:

// Add adds the other given vector to this one and returns the result as a new vector.
func (v Vector3) Add(other Vector3) Vector3 {
	return Vec3(v.X+other.X, v.Y+other.Y, v.Z+other.Z)
}

// AddScalar adds scalar s to each component of this vector and returns new vector.
func (v Vector3) AddScalar(s float32) Vector3 {
	return Vec3(v.X+s, v.Y+s, v.Z+s)
}

// SetAdd sets this to addition with other vector (i.e., += or plus-equals).
func (v *Vector3) SetAdd(other Vector3) {
	v.X += other.X
	v.Y += other.Y
	v.Z += other.Z
}

// SetAddScalar sets this to addition with scalar.
func (v *Vector3) SetAddScalar(s float32) {
	v.X += s
	v.Y += s
	v.Z += s
}

// Sub subtracts other vector from this one and returns result in new vector.
func (v Vector3) Sub(other Vector3) Vector3 {
	return Vec3(v.X-other.X, v.Y-other.Y, v.Z-other.Z)
}

// SubScalar subtracts scalar s from each component of this vector and returns new vector.
func (v Vector3) SubScalar(s float32) Vector3 {
	return Vec3(v.X-s, v.Y-s, v.Z-s)
}

// SetSub sets this to subtraction with other vector (i.e., -= or minus-equals).
func (v *Vector3) SetSub(other Vector3) {
	v.X -= other.X
	v.Y -= other.Y
	v.Z -= other.Z
}

// SetSubScalar sets this to subtraction of scalar.
func (v *Vector3) SetSubScalar(s float32) {
	v.X -= s
	v.Y -= s
	v.Z -= s
}

// Mul multiplies each component of this vector by the corresponding one from other
// and returns resulting vector.
func (v Vector3) Mul(other Vector3) Vector3 {
	return Vec3(v.X*other.X, v.Y*other.Y, v.Z*other.Z)
}

// MulScalar multiplies each component of this vector by the scalar s and returns resulting vector.
func (v Vector3) MulScalar(s float32) Vector3 {
	return Vec3(v.X*s, v.Y*s, v.Z*s)
}

// SetMul sets this to multiplication with other vector (i.e., *= or times-equals).
func (v *Vector3) SetMul(other Vector3) {
	v.X *= other.X
	v.Y *= other.Y
	v.Z *= other.Z
}

// SetMulScalar sets this to multiplication by scalar.
func (v *Vector3) SetMulScalar(s float32) {
	v.X *= s
	v.Y *= s
	v.Z *= s
}

// Div divides each component of this vector by the corresponding one from other vector
// and returns resulting vector.
func (v Vector3) Div(other Vector3) Vector3 {
	return Vec3(v.X/other.X, v.Y/other.Y, v.Z/other.Z)
}

// DivScalar divides each component of this vector by the scalar s and returns resulting vector.
// If scalar is zero, returns zero.
func (v Vector3) DivScalar(scalar float32) Vector3 {
	if scalar != 0 {
		return v.MulScalar(1 / scalar)
	}
	return Vector3{}
}

// SetDiv sets this to division by other vector (i.e., /= or divide-equals).
func (v *Vector3) SetDiv(other Vector3) {
	v.X /= other.X
	v.Y /= other.Y
	v.Z /= other.Z
}

// SetDivScalar sets this to division by scalar.
func (v *Vector3) SetDivScalar(scalar float32) {
	if scalar != 0 {
		v.SetMulScalar(1 / scalar)
	} else {
		v.SetZero()
	}
}

// Min returns min of this vector components vs. other vector.
func (v Vector3) Min(other Vector3) Vector3 {
	return Vec3(Min(v.X, other.X), Min(v.Y, other.Y), Min(v.Z, other.Z))
}

// SetMin sets this vector components to the minimum values of itself and other vector.
func (v *Vector3) SetMin(other Vector3) {
	v.X = Min(v.X, other.X)
	v.Y = Min(v.Y, other.Y)
	v.Z = Min(v.Z, other.Z)
}

// Max returns max of this vector components vs. other vector.
func (v Vector3) Max(other Vector3) Vector3 {
	return Vec3(Max(v.X, other.X), Max(v.Y, other.Y), Max(v.Z, other.Z))
}

// SetMax sets this vector components to the maximum value of itself and other vector.
func (v *Vector3) SetMax(other Vector3) {
	v.X = Max(v.X, other.X)
	v.Y = Max(v.Y, other.Y)
	v.Z = Max(v.Z, other.Z)
}

// Clamp sets this vector's components to be no less than the corresponding
// components of min and not greater than the corresponding component of max.
// Assumes min < max; if this assumption isn't true, it will not operate correctly.
func (v *Vector3) Clamp(min, max Vector3) {
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

// Floor returns this vector with [Floor] applied to each of its components.
func (v Vector3) Floor() Vector3 {
	return Vec3(Floor(v.X), Floor(v.Y), Floor(v.Z))
}

// Ceil returns this vector with [Ceil] applied to each of its components.
func (v Vector3) Ceil() Vector3 {
	return Vec3(Ceil(v.X), Ceil(v.Y), Ceil(v.Z))
}

// Round returns this vector with [Round] applied to each of its components.
func (v Vector3) Round() Vector3 {
	return Vec3(Round(v.X), Round(v.Y), Round(v.Z))
}

// Negate returns the vector with each component negated.
func (v Vector3) Negate() Vector3 {
	return Vec3(-v.X, -v.Y, -v.Z)
}

// Abs returns the vector with [Abs] applied to each component.
func (v Vector3) Abs() Vector3 {
	return Vec3(Abs(v.X), Abs(v.Y), Abs(v.Z))
}

// Distance, Normal:

// Dot returns the dot product of this vector with the given other vector.
func (v Vector3) Dot(other Vector3) float32 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Length returns the length (magnitude) of this vector.
func (v Vector3) Length() float32 {
	return Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// LengthSquared returns the length squared of this vector.
// LengthSquared can be used to compare the lengths of vectors
// without the need to perform a square root.
func (v Vector3) LengthSquared() float32 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

// Normal returns this vector divided by its length (its unit vector).
func (v Vector3) Normal() Vector3 {
	return v.DivScalar(v.Length())
}

// SetNormal normalizes this vector so its length will be 1.
func (v *Vector3) SetNormal() {
	v.SetDivScalar(v.Length())
}

// DistanceTo returns the distance between these two vectors as points.
func (v Vector3) DistanceTo(other Vector3) float32 {
	return Sqrt(v.DistanceToSquared(other))
}

// DistanceToSquared returns the squared distance between these two vectors as points.
func (v Vector3) DistanceToSquared(other Vector3) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	dz := v.Z - other.Z
	return dx*dx + dy*dy + dz*dz
}

// Lerp returns vector with each components as the linear interpolated value of
// alpha between itself and the corresponding other component.
func (v Vector3) Lerp(other Vector3, alpha float32) Vector3 {
	return Vec3(v.X+(other.X-v.X)*alpha, v.Y+(other.Y-v.Y)*alpha, v.Z+(other.Z-v.Z)*alpha)
}

// Matrix operations:

// MulMatrix3 returns the vector multiplied by the given 3x3 matrix.
func (v Vector3) MulMatrix3(m *Matrix3) Vector3 {
	return Vector3{m[0]*v.X + m[3]*v.Y + m[6]*v.Z,
		m[1]*v.X + m[4]*v.Y + m[7]*v.Z,
		m[2]*v.X + m[5]*v.Y + m[8]*v.Z}
}

// MulMatrix4 returns the vector multiplied by the given 4x4 matrix.
func (v Vector3) MulMatrix4(m *Matrix4) Vector3 {
	return Vector3{m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12],
		m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13],
		m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]}
}

// MulMatrix4AsVector4 returns 3-dim vector multiplied by specified 4x4 matrix
// using a 4-dim vector with given 4th dimensional value, then reduced back to
// a 3-dimensional vector.  This is somehow different from just straight
// MulMatrix4 on the 3-dim vector.  Use 0 for normals and 1 for positions
// as the 4th dim to set.
func (v Vector3) MulMatrix4AsVector4(m *Matrix4, w float32) Vector3 {
	return Vector3FromVector4(Vector4FromVector3(v, w).MulMatrix4(m))
}

// NDCToWindow converts normalized display coordinates (NDC) to window
// (pixel) coordinates, using given window size parameters.
// near, far are 0, 1 by default (glDepthRange defaults).
// flipY if true means flip the Y axis (top = 0 for windows vs. bottom = 0 for 3D coords)
func (v Vector3) NDCToWindow(size, off Vector2, near, far float32, flipY bool) Vector3 {
	w := Vector3{}
	half := size.MulScalar(0.5)
	w.X = half.X*v.X + half.X
	w.Y = half.Y*v.Y + half.Y
	w.Z = 0.5*(far-near)*v.Z + 0.5*(far+near)
	if flipY {
		w.Y = size.Y - w.Y
	}
	w.X += off.X
	w.Y += off.Y
	return w
}

// WindowToNDC converts window (pixel) coordinates to
// normalized display coordinates (NDC), using given window size parameters.
// The Z depth coordinate (0-1) must be set manually or by reading from framebuffer
// flipY if true means flip the Y axis (top = 0 for windows vs. bottom = 0 for 3D coords)
func (v Vector2) WindowToNDC(size, off Vector2, flipY bool) Vector3 {
	n := Vector3{}
	half := size.MulScalar(0.5)
	n.X = v.X - off.X
	n.Y = v.Y - off.Y
	if flipY {
		n.Y = size.Y - n.Y
	}
	n.X = n.X/half.X - 1
	n.Y = n.Y/half.Y - 1
	return n
}

// MulProjection returns vector multiplied by the projection matrix m.
func (v Vector3) MulProjection(m *Matrix4) Vector3 {
	d := 1 / (m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]) // perspective divide
	return Vector3{(m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]) * d,
		(m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]) * d,
		(m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]) * d}
}

// MulQuat returns vector multiplied by specified quaternion and
// then by the quaternion inverse.
// It basically applies the rotation encoded in the quaternion to this vector.
func (v Vector3) MulQuat(q Quat) Vector3 {
	qx := q.X
	qy := q.Y
	qz := q.Z
	qw := q.W
	// calculate quat * vector
	ix := qw*v.X + qy*v.Z - qz*v.Y
	iy := qw*v.Y + qz*v.X - qx*v.Z
	iz := qw*v.Z + qx*v.Y - qy*v.X
	iw := -qx*v.X - qy*v.Y - qz*v.Z
	// calculate result * inverse quat
	return Vector3{ix*qw + iw*-qx + iy*-qz - iz*-qy,
		iy*qw + iw*-qy + iz*-qx - ix*-qz,
		iz*qw + iw*-qz + ix*-qy - iy*-qx}
}

// Cross returns the cross product of this vector with other.
func (v Vector3) Cross(other Vector3) Vector3 {
	return Vec3(v.Y*other.Z-v.Z*other.Y, v.Z*other.X-v.X*other.Z, v.X*other.Y-v.Y*other.X)
}

// ProjectOnVector returns vector projected on other vector.
func (v *Vector3) ProjectOnVector(other Vector3) Vector3 {
	on := other.Normal()
	return on.MulScalar(v.Dot(on))
}

// ProjectOnPlane returns vector projected on the plane specified by normal vector.
func (v *Vector3) ProjectOnPlane(planeNormal Vector3) Vector3 {
	return v.Sub(v.ProjectOnVector(planeNormal))
}

// Reflect returns vector reflected relative to the normal vector (assumed to be
// already normalized).
func (v *Vector3) Reflect(normal Vector3) Vector3 {
	return v.Sub(normal.MulScalar(2 * v.Dot(normal)))
}

// CosTo returns the cosine (normalized dot product) between this vector and other.
func (v Vector3) CosTo(other Vector3) float32 {
	return v.Dot(other) / (v.Length() * other.Length())
}

// AngleTo returns the angle between this vector and other.
// Returns angles in range of -PI to PI (not 0 to 2 PI).
func (v Vector3) AngleTo(other Vector3) float32 {
	ang := Acos(Clamp(v.CosTo(other), -1, 1))
	cross := v.Cross(other)
	switch {
	case Abs(cross.Z) >= Abs(cross.Y) && Abs(cross.Z) >= Abs(cross.X):
		if cross.Z > 0 {
			ang = -ang
		}
	case Abs(cross.Y) >= Abs(cross.Z) && Abs(cross.Y) >= Abs(cross.X):
		if cross.Y > 0 {
			ang = -ang
		}
	case Abs(cross.X) >= Abs(cross.Z) && Abs(cross.X) >= Abs(cross.Y):
		if cross.X > 0 {
			ang = -ang
		}
	}
	return ang
}

// SetFromMatrixPos set this vector from the translation coordinates
// in the specified transformation matrix.
func (v *Vector3) SetFromMatrixPos(m *Matrix4) {
	v.X = m[12]
	v.Y = m[13]
	v.Z = m[14]
}

// SetEulerAnglesFromMatrix sets this vector components to the Euler angles
// from the specified pure rotation matrix.
func (v *Vector3) SetEulerAnglesFromMatrix(m *Matrix4) {
	m11 := m[0]
	m12 := m[4]
	m13 := m[8]
	m22 := m[5]
	m23 := m[9]
	m32 := m[6]
	m33 := m[10]

	v.Y = Asin(Clamp(m13, -1, 1))
	if Abs(m13) < 0.99999 {
		v.X = Atan2(-m23, m33)
		v.Z = Atan2(-m12, m11)
	} else {
		v.X = Atan2(m32, m22)
		v.Z = 0
	}
}

// NewEulerAnglesFromMatrix returns a Vector3 with components as the Euler angles
// from the specified pure rotation matrix.
func NewEulerAnglesFromMatrix(m *Matrix4) Vector3 {
	rot := Vector3{}
	rot.SetEulerAnglesFromMatrix(m)
	return rot
}

// SetEulerAnglesFromQuat sets this vector components to the Euler angles
// from the specified quaternion.
func (v *Vector3) SetEulerAnglesFromQuat(q Quat) {
	mat := Identity4()
	mat.SetRotationFromQuat(q)
	v.SetEulerAnglesFromMatrix(mat)
}

// RandomTangents computes and returns two arbitrary tangents to the vector.
func (v *Vector3) RandomTangents() (Vector3, Vector3) {
	t1 := Vector3{}
	t2 := Vector3{}
	length := v.Length()
	if length > 0 {
		n := v.Normal()
		randVec := Vector3{}
		if Abs(n.X) < 0.9 {
			randVec.X = 1
			t1 = n.Cross(randVec)
		} else if Abs(n.Y) < 0.9 {
			randVec.Y = 1
			t1 = n.Cross(randVec)
		} else {
			randVec.Z = 1
			t1 = n.Cross(randVec)
		}
		t2 = n.Cross(t1)
	} else {
		t1.X = 1
		t2.Y = 1
	}
	return t1, t2
}
