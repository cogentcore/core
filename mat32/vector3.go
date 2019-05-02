// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Vec3 is a 3D vector/point with X, Y and Z components.
type Vec3 struct {
	X float32
	Y float32
	Z float32
}

// NewVec3 returns a new Vec3 with the specified x, y and y components.
func NewVec3(x, y, z float32) Vec3 {
	return Vec3{X: x, Y: y, Z: z}
}

// NewVec3Scalar returns a new Vec3 with all components set to scalar.
func NewVec3Scalar(s float32) Vec3 {
	return Vec3{X: s, Y: s, Z: s}
}

// NewVec3FromVec4 returns a new Vec3 from a Vec4
func NewVec3FromVec4(v Vec4) Vec3 {
	nv := Vec3{}
	nv.SetFromVec4(v)
	return nv
}

// IsNil returns true if all values are 0 (uninitialized).
func (v Vec3) IsNil() bool {
	if v.X == 0 && v.Y == 0 && v.Z == 0 {
		return true
	}
	return false
}

// Set sets this vector X, Y and Z components.
func (v *Vec3) Set(x, y, z float32) {
	v.X = x
	v.Y = y
	v.Z = z
}

// SetScalar sets all vector X, Y and Z components to same scalar value.
func (v *Vec3) SetScalar(s float32) {
	v.X = s
	v.Y = s
	v.Z = s
}

// SetFromVec4 sets this vector from a Vec4
func (v *Vec3) SetFromVec4(other Vec4) {
	v.X = other.X
	v.Y = other.Y
	v.Z = other.Z
}

// SetFromVec3i sets from a Vec3i (int32) vector.
func (v *Vec3) SetFromVec3i(vi Vec3i) {
	v.X = float32(vi.X)
	v.Y = float32(vi.Y)
	v.Z = float32(vi.Z)
}

// SetComponent sets this vector component value by component index.
func (v *Vec3) SetComponent(comp Components, value float32) {
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
func (v Vec3) Component(comp Components) float32 {
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
func (v *Vec3) SetByName(name string, value float32) {
	switch name {
	case "x", "X":
		v.X = value
	case "y", "Y":
		v.Y = value
	case "z", "Z":
		v.Z = value
	default:
		panic("Invalid Vec3 component name: " + name)
	}
}

// SetZero sets this vector X, Y and Z components to be zero.
func (v *Vec3) SetZero() {
	v.SetScalar(0)
}

// FromArray sets this vector's components from the specified array and offset.
func (v *Vec3) FromArray(array []float32, offset int) {
	v.X = array[offset]
	v.Y = array[offset+1]
	v.Z = array[offset+2]
}

// ToArray copies this vector's components to array starting at offset.
func (v Vec3) ToArray(array []float32, offset int) {
	array[offset] = v.X
	array[offset+1] = v.Y
	array[offset+2] = v.Z
}

///////////////////////////////////////////////////////////////////////
//  Basic math operations

// Add adds other vector to this one and returns result in a new vector.
func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// AddScalar adds scalar s to each component of this vector and returns new vector.
func (v Vec3) AddScalar(s float32) Vec3 {
	return Vec3{v.X + s, v.Y + s, v.Z + s}
}

// SetAdd sets this to addition with other vector (i.e., += or plus-equals).
func (v *Vec3) SetAdd(other Vec3) {
	v.X += other.X
	v.Y += other.Y
	v.Z += other.Z
}

// SetAddScalar sets this to addition with scalar.
func (v *Vec3) SetAddScalar(s float32) {
	v.X += s
	v.Y += s
	v.Z += s
}

// Sub subtracts other vector from this one and returns result in new vector.
func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{v.X - other.X, v.Y - other.Y, v.Z - other.Z}
}

// SubScalar subtracts scalar s from each component of this vector and returns new vector.
func (v Vec3) SubScalar(s float32) Vec3 {
	return Vec3{v.X - s, v.Y - s, v.Z - s}
}

// SetSub sets this to subtraction with other vector (i.e., -= or minus-equals).
func (v *Vec3) SetSub(other Vec3) {
	v.X -= other.X
	v.Y -= other.Y
	v.Z -= other.Z
}

// SetSubScalar sets this to subtraction of scalar.
func (v *Vec3) SetSubScalar(s float32) {
	v.X -= s
	v.Y -= s
	v.Z -= s
}

// Mul multiplies each component of this vector by the corresponding one from other
// and returns resulting vector.
func (v Vec3) Mul(other Vec3) Vec3 {
	return Vec3{v.X * other.X, v.Y * other.Y, v.Z * other.Z}
}

// MulScalar multiplies each component of this vector by the scalar s and returns resulting vector.
func (v Vec3) MulScalar(s float32) Vec3 {
	return Vec3{v.X * s, v.Y * s, v.Z * s}
}

// SetMul sets this to multiplication with other vector (i.e., *= or times-equals).
func (v *Vec3) SetMul(other Vec3) {
	v.X *= other.X
	v.Y *= other.Y
	v.Z *= other.Z
}

// SetMulScalar sets this to multiplication by scalar.
func (v *Vec3) SetMulScalar(s float32) {
	v.X *= s
	v.Y *= s
	v.Z *= s
}

// Div divides each component of this vector by the corresponding one from other vector
// and returns resulting vector.
func (v Vec3) Div(other Vec3) Vec3 {
	return Vec3{v.X / other.X, v.Y / other.Y, v.Z / other.Z}
}

// DivScalar divides each component of this vector by the scalar s and returns resulting vector.
// If scalar is zero, returns zero.
func (v Vec3) DivScalar(scalar float32) Vec3 {
	if scalar != 0 {
		return v.MulScalar(1 / scalar)
	} else {
		return Vec3{}
	}
}

// SetDiv sets this to division by other vector (i.e., /= or divide-equals).
func (v *Vec3) SetDiv(other Vec3) {
	v.X /= other.X
	v.Y /= other.Y
	v.Z /= other.Z
}

// SetDivScalar sets this to division by scalar.
func (v *Vec3) SetDivScalar(s float32) {
	if s != 0 {
		v.SetMulScalar(1 / s)
	} else {
		v.SetZero()
	}
}

// Min returns min of this vector components vs. other vector.
func (v Vec3) Min(other Vec3) Vec3 {
	return Vec3{Min(v.X, other.X), Min(v.Y, other.Y), Min(v.Z, other.Z)}
}

// SetMin sets this vector components to the minimum values of itself and other vector.
func (v *Vec3) SetMin(other Vec3) {
	v.X = Min(v.X, other.X)
	v.Y = Min(v.Y, other.Y)
	v.Z = Min(v.Z, other.Z)
}

// Max returns max of this vector components vs. other vector.
func (v Vec3) Max(other Vec3) Vec3 {
	return Vec3{Max(v.X, other.X), Max(v.Y, other.Y), Max(v.Z, other.Z)}
}

// SetMax sets this vector components to the maximum value of itself and other vector.
func (v *Vec3) SetMax(other Vec3) {
	v.X = Max(v.X, other.X)
	v.Y = Max(v.Y, other.Y)
	v.Z = Max(v.Z, other.Z)
}

// Clamp sets this vector components to be no less than the corresponding components of min
// and not greater than the corresponding component of max.
// Assumes min < max, if this assumption isn't true it will not operate correctly.
func (v *Vec3) Clamp(min, max Vec3) {
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
func (v *Vec3) ClampScalar(minVal, maxVal float32) {
	v.Clamp(NewVec3Scalar(minVal), NewVec3Scalar(maxVal))
}

// Floor returns vector with mat32.Floor() applied to each of this vector's components.
func (v Vec3) Floor() Vec3 {
	return Vec3{Floor(v.X), Floor(v.Y), Floor(v.Z)}
}

// SetFloor applies mat32.Floor() to each of this vector's components.
func (v *Vec3) SetFloor() {
	v.X = Floor(v.X)
	v.Y = Floor(v.Y)
	v.Z = Floor(v.Z)
}

// Ceil returns vector with mat32.Ceil() applied to each of this vector's components.
func (v Vec3) Ceil() Vec3 {
	return Vec3{Ceil(v.X), Ceil(v.Y), Ceil(v.Z)}
}

// SetCeil applies mat32.Ceil() to each of this vector's components.
func (v *Vec3) SetCeil() {
	v.X = Ceil(v.X)
	v.Y = Ceil(v.Y)
	v.Z = Ceil(v.Z)
}

// Round returns vector with mat32.Round() applied to each of this vector's components.
func (v Vec3) Round() Vec3 {
	return Vec3{Round(v.X), Round(v.Y), Round(v.Z)}
}

// SetRound rounds each of this vector's components.
func (v *Vec3) SetRound() {
	v.X = Round(v.X)
	v.Y = Round(v.Y)
	v.Z = Round(v.Z)
}

// Negate returns vector with each component negated.
func (v Vec3) Negate() Vec3 {
	return Vec3{-v.X, -v.Y, -v.Z}
}

// SetNegate negates each of this vector's components.
func (v *Vec3) SetNegate() {
	v.X = -v.X
	v.Y = -v.Y
	v.Z = -v.Z
}

//////////////////////////////////////////////////////////////////////////////////
//  Distance, Norm

// IsEqual returns if this vector is equal to other.
func (v Vec3) IsEqual(other Vec3) bool {
	return (other.X == v.X) && (other.Y == v.Y) && (other.Z == v.Z)
}

// AlmostEqual returns whether the vector is almost equal to another vector within the specified tolerance.
func (v *Vec3) AlmostEqual(other Vec3, tol float32) bool {
	if (Abs(v.X-other.X) < tol) &&
		(Abs(v.Y-other.Y) < tol) &&
		(Abs(v.Z-other.Z) < tol) {
		return true
	}
	return false
}

// Dot returns the dot product of this vector with other.
func (v Vec3) Dot(other Vec3) float32 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// LengthSq returns the length squared of this vector.
// LengthSq can be used to compare vectors' lengths without the need to perform a square root.
func (v Vec3) LengthSq() float32 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

// Length returns the length of this vector.
func (v Vec3) Length() float32 {
	return Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// Normal returns this vector divided by its length
func (v Vec3) Normal() Vec3 {
	return v.DivScalar(v.Length())
}

// SetNormal normalizes this vector so its length will be 1.
func (v *Vec3) SetNormal() {
	v.SetDivScalar(v.Length())
}

// Normalize normalizes this vector so its length will be 1.
func (v *Vec3) Normalize() {
	v.SetDivScalar(v.Length())
}

// DistTo returns the distance of this point to other.
func (v Vec3) DistTo(other Vec3) float32 {
	return Sqrt(v.DistToSquared(other))
}

// DistToSquared returns the distance squared of this point to other.
func (v Vec3) DistToSquared(other Vec3) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	dz := v.Z - other.Z
	return dx*dx + dy*dy + dz*dz
}

// SetLength sets this vector to have the specified length.
// If the current length is zero, does nothing.
func (v *Vec3) SetLength(l float32) {
	oldLength := v.Length()
	if oldLength != 0 && l != oldLength {
		v.SetMulScalar(l / oldLength)
	}
}

// Lerp returns vector with each components as the linear interpolated value of
// alpha between itself and the corresponding other component.
func (v Vec3) Lerp(other Vec3, alpha float32) Vec3 {
	return Vec3{v.X + (other.X-v.X)*alpha, v.Y + (other.Y-v.Y)*alpha, v.Z + (other.Z-v.Z)*alpha}
}

// SetLerp sets each of this vector's components to the linear interpolated value of
// alpha between itself and the corresponding other component.
func (v *Vec3) SetLerp(other Vec3, alpha float32) {
	v.X += (other.X - v.X) * alpha
	v.Y += (other.Y - v.Y) * alpha
	v.Z += (other.Z - v.Z) * alpha
}

/////////////////////////////////////////////////////////////////////////////
//  Matrix operations

// RotateAxisAngle returns vector rotated around axis by angle.
func (v Vec3) RotateAxisAngle(axis Vec3, angle float32) Vec3 {
	return v.MulQuat(NewQuatAxisAngle(axis, angle))
}

// SetRotateAxisAngle sets vector rotated around axis by angle.
func (v *Vec3) SetRotateAxisAngle(axis Vec3, angle float32) {
	*v = v.RotateAxisAngle(axis, angle)
}

// MulMat3 returns vector multiplied by specified 3x3 matrix.
func (v Vec3) MulMat3(m *Mat3) Vec3 {
	return Vec3{m[0]*v.X + m[3]*v.Y + m[6]*v.Z,
		m[1]*v.X + m[4]*v.Y + m[7]*v.Z,
		m[2]*v.X + m[5]*v.Y + m[8]*v.Z}
}

// SetMulMat3 sets vector multiplied by specified 3x3 matrix.
func (v *Vec3) SetMulMat3(m *Mat3) {
	*v = v.MulMat3(m)
}

// MulMat4 returns vector multiplied by specified 4x4 matrix.
func (v Vec3) MulMat4(m *Mat4) Vec3 {
	return Vec3{m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12],
		m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13],
		m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]}
}

// MulMat4AsVec4 returns 3-dim vector multiplied by specified 4x4 matrix
// using a 4-dim vector with given 4th dimensional value, then reduced back to
// a 3-dimensional vector.  This is somehow different from just straight
// MulMat4 on the 3-dim vector.  Use 0 for normals and 1 for positions
// as the 4th dim to set.
func (v Vec3) MulMat4AsVec4(m *Mat4, w float32) Vec3 {
	return NewVec3FromVec4(NewVec4FromVec3(v, w).MulMat4(m))
}

// SetMulMat4 sets vector multiplied by specified 4x4 matrix.
func (v *Vec3) SetMulMat4(m *Mat4) {
	*v = v.MulMat4(m)
}

// MulProjection returns vector multiplied by the projection matrix m
func (v Vec3) MulProjection(m *Mat4) Vec3 {
	d := 1 / (m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]) // perspective divide
	return Vec3{(m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]) * d,
		(m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]) * d,
		(m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]) * d}
}

// MulQuat returns vector multiplied by specified quaternion and
// then by the quaternion inverse.
// It basically applies the rotation encoded in the quaternion to this vector.
func (v Vec3) MulQuat(q Quat) Vec3 {
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
	return Vec3{ix*qw + iw*-qx + iy*-qz - iz*-qy,
		iy*qw + iw*-qy + iz*-qx - ix*-qz,
		iz*qw + iw*-qz + ix*-qy - iy*-qx}
}

// SetMulQuat multiplies vector by specified quaternion and
// then by the quaternion inverse.
// It basically applies the rotation encoded in the quaternion to this vector.
func (v *Vec3) SetMulQuat(q Quat) {
	*v = v.MulQuat(q)
}

// Cross returns the cross product of this vector with other.
func (v Vec3) Cross(other Vec3) Vec3 {
	return Vec3{v.Y*other.Z - v.Z*other.Y, v.Z*other.X - v.X*other.Z, v.X*other.Y - v.Y*other.X}
}

// ProjectOnVector returns vector projected on other vector.
func (v *Vec3) ProjectOnVector(other Vec3) Vec3 {
	on := other.Normal()
	return on.MulScalar(v.Dot(on))
}

// ProjectOnPlane returns vector projected on the plane specified by normal vector.
func (v *Vec3) ProjectOnPlane(planeNormal Vec3) Vec3 {
	return v.Sub(v.ProjectOnVector(planeNormal))
}

// Reflect returns vector reflected relative to the normal vector (assumed to be
// already normalized).
func (v *Vec3) Reflect(normal Vec3) Vec3 {
	return v.Sub(normal.MulScalar(2 * v.Dot(normal)))
}

// AngleTo returns the angle between this vector and other.
func (v Vec3) AngleTo(other Vec3) float32 {
	theta := v.Dot(other) / (v.Length() * other.Length())
	// clip, to handle numerical problems
	return Acos(Clamp(theta, -1, 1))
}

// SetFromMatrixPos set this vector from the translation coordinates
// in the specified transformation matrix.
func (v *Vec3) SetFromMatrixPos(m *Mat4) {
	v.X = m[12]
	v.Y = m[13]
	v.Z = m[14]
}

// SetFromMatrixCol set this vector with the column at index of the m matrix.
func (v *Vec3) SetFromMatrixCol(index int, m *Mat4) {
	offset := index * 4
	v.X = m[offset]
	v.Y = m[offset+1]
	v.Z = m[offset+2]
}

// SetEulerAnglesFromMatrix sets this vector components to the Euler angles
// from the specified pure rotation matrix.
func (v *Vec3) SetEulerAnglesFromMatrix(m *Mat4) {
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

// NewEulerAnglesFromMatrix returns a Vec3 with components as the Euler angles
// from the specified pure rotation matrix.
func NewEulerAnglesFromMatrix(m *Mat4) Vec3 {
	rot := Vec3{}
	rot.SetEulerAnglesFromMatrix(m)
	return rot
}

// SetEulerAnglesFromQuat sets this vector components to the Euler angles
// from the specified quaternion.
func (v *Vec3) SetEulerAnglesFromQuat(q Quat) {
	mat := NewMat4()
	mat.SetRotationFromQuat(q)
	v.SetEulerAnglesFromMatrix(mat)
}

// NewEulerAnglesFromQuat returns a Vec3 with components as the Euler angles
// from the specified quaternion.
func NewEulerAnglesFromQuat(q Quat) Vec3 {
	rot := Vec3{}
	rot.SetEulerAnglesFromQuat(q)
	return rot
}

// RandomTangents computes and returns two arbitrary tangents to the vector.
func (v *Vec3) RandomTangents() (Vec3, Vec3) {
	t1 := Vec3{}
	t2 := Vec3{}
	length := v.Length()
	if length > 0 {
		n := v.Normal()
		randVec := Vec3{}
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
