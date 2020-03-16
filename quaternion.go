// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

import "fmt"

// Quat is quaternion with X,Y,Z and W components.
type Quat struct {
	X float32
	Y float32
	Z float32
	W float32
}

// NewQuat returns a new quaternion from the specified components.
func NewQuat(x, y, z, w float32) Quat {
	return Quat{X: x, Y: y, Z: z, W: w}
}

// NewQuatAxisAngle returns a new quaternion from given axis and angle rotation (radians).
func NewQuatAxisAngle(axis Vec3, angle float32) Quat {
	nq := Quat{}
	nq.SetFromAxisAngle(axis, angle)
	return nq
}

// NewQuatEuler returns a new quaternion from given Euler angles.
func NewQuatEuler(euler Vec3) Quat {
	nq := Quat{}
	nq.SetFromEuler(euler)
	return nq
}

// Set sets this quaternion's components.
func (q *Quat) Set(x, y, z, w float32) {
	q.X = x
	q.Y = y
	q.Z = z
	q.W = w
}

// FromArray sets this quaternion's components from array starting at offset.
func (q *Quat) FromArray(array []float32, offset int) {
	q.X = array[offset]
	q.Y = array[offset+1]
	q.Z = array[offset+2]
	q.W = array[offset+3]
}

// ToArray copies this quaternions's components to array starting at offset.
func (q *Quat) ToArray(array []float32, offset int) {
	array[offset] = q.X
	array[offset+1] = q.Y
	array[offset+2] = q.Z
	array[offset+3] = q.W
}

// SetIdentity sets this quanternion to the identity quaternion.
func (q *Quat) SetIdentity() {
	q.X = 0
	q.Y = 0
	q.Z = 0
	q.W = 1
}

// IsIdentity returns if this is an identity quaternion.
func (q *Quat) IsIdentity() bool {
	if q.X == 0 && q.Y == 0 && q.Z == 0 && q.W == 1 {
		return true
	}
	return false
}

// IsNil returns true if all values are 0 (uninitialized).
func (q *Quat) IsNil() bool {
	if q.X == 0 && q.Y == 0 && q.Z == 0 && q.W == 0 {
		return true
	}
	return false
}

// SetFromEuler sets this quaternion from the specified vector with
// Euler angles for each axis. It is assumed that the Euler angles
// are in XYZ order.
func (q *Quat) SetFromEuler(euler Vec3) {
	c1 := Cos(euler.X / 2)
	c2 := Cos(euler.Y / 2)
	c3 := Cos(euler.Z / 2)
	s1 := Sin(euler.X / 2)
	s2 := Sin(euler.Y / 2)
	s3 := Sin(euler.Z / 2)

	q.X = s1*c2*c3 - c1*s2*s3
	q.Y = c1*s2*c3 + s1*c2*s3
	q.Z = c1*c2*s3 - s1*s2*c3
	q.W = c1*c2*c3 + s1*s2*s3
}

// ToEuler returns a Vec3 with components as the Euler angles
// from the given quaternion.
func (q *Quat) ToEuler() Vec3 {
	rot := Vec3{}
	rot.SetEulerAnglesFromQuat(*q)
	return rot
}

// SetFromAxisAngle sets this quaternion with the rotation
// specified by the given axis and angle.
func (q *Quat) SetFromAxisAngle(axis Vec3, angle float32) {
	halfAngle := angle / 2
	s := Sin(halfAngle)
	q.X = axis.X * s
	q.Y = axis.Y * s
	q.Z = axis.Z * s
	q.W = Cos(halfAngle)
}

// ToAxisAngle returns the Vec4 holding axis and angle of this Quaternion
func (q *Quat) ToAxisAngle() Vec4 {
	aa := Vec4{}
	aa.SetAxisAngleFromQuat(*q)
	return aa
}

// GenGoSet returns code to set values in object at given path (var.member etc)
func (q *Quat) GenGoSet(path string) string {
	aa := q.ToAxisAngle()
	return fmt.Sprintf("%s.SetFromAxisAngle(mat32.Vec3{%g, %g, %g}, %g)", path, aa.X, aa.Y, aa.Z, aa.W)
}

// GenGoNew returns code to create new
func (q *Quat) GenGoNew() string {
	return fmt.Sprintf("mat32.Quat{%g, %g, %g, %g}", q.X, q.Y, q.Z, q.W)
}

// SetFromRotationMatrix sets this quaternion from the specified rotation matrix.
func (q *Quat) SetFromRotationMatrix(m *Mat4) {
	m11 := m[0]
	m12 := m[4]
	m13 := m[8]
	m21 := m[1]
	m22 := m[5]
	m23 := m[9]
	m31 := m[2]
	m32 := m[6]
	m33 := m[10]
	trace := m11 + m22 + m33

	var s float32
	if trace > 0 {
		s = 0.5 / Sqrt(trace+1.0)
		q.W = 0.25 / s
		q.X = (m32 - m23) * s
		q.Y = (m13 - m31) * s
		q.Z = (m21 - m12) * s
	} else if m11 > m22 && m11 > m33 {
		s = 2.0 * Sqrt(1.0+m11-m22-m33)
		q.W = (m32 - m23) / s
		q.X = 0.25 * s
		q.Y = (m12 + m21) / s
		q.Z = (m13 + m31) / s
	} else if m22 > m33 {
		s = 2.0 * Sqrt(1.0+m22-m11-m33)
		q.W = (m13 - m31) / s
		q.X = (m12 + m21) / s
		q.Y = 0.25 * s
		q.Z = (m23 + m32) / s
	} else {
		s = 2.0 * Sqrt(1.0+m33-m11-m22)
		q.W = (m21 - m12) / s
		q.X = (m13 + m31) / s
		q.Y = (m23 + m32) / s
		q.Z = 0.25 * s
	}
}

// SetFromUnitVectors sets this quaternion to the rotation from vector vFrom to vTo.
// The vectors must be normalized.
func (q *Quat) SetFromUnitVectors(vFrom, vTo Vec3) {
	var v1 Vec3
	var EPS float32 = 0.000001

	r := vFrom.Dot(vTo) + 1
	if r < EPS {
		r = 0
		if Abs(vFrom.X) > Abs(vFrom.Z) {
			v1.Set(-vFrom.Y, vFrom.X, 0)
		} else {
			v1.Set(0, -vFrom.Z, vFrom.Y)
		}

	} else {
		v1 = vFrom.Cross(vTo)
	}
	q.X = v1.X
	q.Y = v1.Y
	q.Z = v1.Z
	q.W = r

	q.Normalize()
}

// SetInverse sets this quaternion to its inverse.
func (q *Quat) SetInverse() {
	q.SetConjugate()
	q.Normalize()
}

// Inverse returns the inverse of this quaternion.
func (q *Quat) Inverse() Quat {
	nq := *q
	nq.SetInverse()
	return nq
}

// SetConjugate sets this quaternion to its conjugate.
func (q *Quat) SetConjugate() {
	q.X *= -1
	q.Y *= -1
	q.Z *= -1
}

// Conjugate returns the conjugate of this quaternion.
func (q *Quat) Conjugate() Quat {
	nq := *q
	nq.SetConjugate()
	return nq
}

// Dot returns the dot products of this quaternion with other.
func (q *Quat) Dot(other Quat) float32 {
	return q.X*other.X + q.Y*other.Y + q.Z*other.Z + q.W*other.W
}

// LengthSq returns this quanternion's length squared
func (q Quat) LengthSq() float32 {
	return q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W
}

// Length returns the length of this quaternion
func (q Quat) Length() float32 {
	return Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)
}

// Normalize normalizes this quaternion.
func (q *Quat) Normalize() {
	l := q.Length()
	if l == 0 {
		q.X = 0
		q.Y = 0
		q.Z = 0
		q.W = 1
	} else {
		l = 1 / l
		q.X *= l
		q.Y *= l
		q.Z *= l
		q.W *= l
	}
}

// NormalizeFast approximates normalizing this quaternion.
// Works best when the quaternion is already almost-normalized.
func (q *Quat) NormalizeFast() {
	f := (3.0 - (q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)) / 2.0
	if f == 0 {
		q.X = 0
		q.Y = 0
		q.Z = 0
		q.W = 1
	} else {
		q.X *= f
		q.Y *= f
		q.Z *= f
		q.W *= f
	}
}

// MulQuats set this quaternion to the multiplication of a by b.
func (q *Quat) MulQuats(a, b Quat) {
	// from http://www.euclideanspace.com/maths/algebra/realNormedAlgebra/quaternions/code/index.htm
	qax := a.X
	qay := a.Y
	qaz := a.Z
	qaw := a.W
	qbx := b.X
	qby := b.Y
	qbz := b.Z
	qbw := b.W

	q.X = qax*qbw + qaw*qbx + qay*qbz - qaz*qby
	q.Y = qay*qbw + qaw*qby + qaz*qbx - qax*qbz
	q.Z = qaz*qbw + qaw*qbz + qax*qby - qay*qbx
	q.W = qaw*qbw - qax*qbx - qay*qby - qaz*qbz
}

// SetMul sets this quaternion to the multiplication of itself by other.
func (q *Quat) SetMul(other Quat) {
	q.MulQuats(*q, other)
}

// Mul returns returns multiplication of this quaternion with other
func (q *Quat) Mul(other Quat) Quat {
	nq := *q
	nq.SetMul(other)
	return nq
}

// Slerp sets this quaternion to another quaternion which is the spherically linear interpolation
// from this quaternion to other using t.
func (q *Quat) Slerp(other Quat, t float32) {
	if t == 0 {
		return
	}
	if t == 1 {
		*q = other
		return
	}

	x := q.X
	y := q.Y
	z := q.Z
	w := q.W

	cosHalfTheta := w*other.W + x*other.X + y*other.Y + z*other.Z

	if cosHalfTheta < 0 {
		q.W = -other.W
		q.X = -other.X
		q.Y = -other.Y
		q.Z = -other.Z
		cosHalfTheta = -cosHalfTheta
	} else {
		*q = other
	}

	if cosHalfTheta >= 1.0 {
		q.W = w
		q.X = x
		q.Y = y
		q.Z = z
		return
	}

	sqrSinHalfTheta := 1.0 - cosHalfTheta*cosHalfTheta
	if sqrSinHalfTheta < 0.001 {
		s := 1 - t
		q.W = s*w + t*q.W
		q.X = s*x + t*q.X
		q.Y = s*y + t*q.Y
		q.Z = s*z + t*q.Z
		q.Normalize()
		return
	}

	sinHalfTheta := Sqrt(sqrSinHalfTheta)
	halfTheta := Atan2(sinHalfTheta, cosHalfTheta)
	ratioA := Sin((1-t)*halfTheta) / sinHalfTheta
	ratioB := Sin(t*halfTheta) / sinHalfTheta

	q.W = w*ratioA + q.W*ratioB
	q.X = x*ratioA + q.X*ratioB
	q.Y = y*ratioA + q.Y*ratioB
	q.Z = z*ratioA + q.Z*ratioB
}

// IsEqual returns if this quaternion is equal to other.
func (q *Quat) IsEqual(other Quat) bool {
	return (other.X == q.X) && (other.Y == q.Y) && (other.Z == q.Z) && (other.W == q.W)
}
