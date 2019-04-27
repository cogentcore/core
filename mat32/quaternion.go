// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Quat is quaternion with X,Y,Z and W components.
type Quat struct {
	X float32
	Y float32
	Z float32
	W float32
}

// NewQuat creates and returns a pointer to a new quaternion
// from the specified components.
func NewQuat(x, y, z, w float32) *Quat {
	return &Quat{
		X: x, Y: y, Z: z, W: w,
	}
}

// SetX sets this quaternion's X component.
// Returns pointer to this updated quaternion.
func (q *Quat) SetX(val float32) *Quat {
	q.X = val
	return q
}

// SetY sets this quaternion's Y component.
// Returns pointer to this updated quaternion.
func (q *Quat) SetY(val float32) *Quat {
	q.Y = val
	return q
}

// SetZ sets this quaternion's Z component.
// Returns pointer to this updated quaternion.
func (q *Quat) SetZ(val float32) *Quat {
	q.Z = val
	return q
}

// SetW sets this quaternion's W component.
// Returns pointer to this updated quaternion.
func (q *Quat) SetW(val float32) *Quat {
	q.W = val
	return q
}

// Set sets this quaternion's components.
// Returns pointer to this updated quaternion.
func (q *Quat) Set(x, y, z, w float32) *Quat {
	q.X = x
	q.Y = y
	q.Z = z
	q.W = w
	return q
}

// SetIdentity sets this quanternion to the identity quaternion.
// Returns pointer to this updated quaternion.
func (q *Quat) SetIdentity() *Quat {
	q.X = 0
	q.Y = 0
	q.Z = 0
	q.W = 1
	return q
}

// IsIdentity returns it this is an identity quaternion.
func (q *Quat) IsIdentity() bool {
	if q.X == 0 && q.Y == 0 && q.Z == 0 && q.W == 1 {
		return true
	}
	return false
}

// Copy copies the other quaternion into this one.
// Returns pointer to this updated quaternion.
func (q *Quat) Copy(other *Quat) *Quat {
	*q = *other
	return q
}

// SetFromEuler sets this quaternion from the specified vector with
// euler angles for each axis. It is assumed that the Euler angles
// are in XYZ order.
// Returns pointer to this updated quaternion.
func (q *Quat) SetFromEuler(euler *Vec3) *Quat {
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

	return q
}

// SetFromAxisAngle sets this quaternion with the rotation
// specified by the given axis and angle.
// Returns pointer to this updated quaternion.
func (q *Quat) SetFromAxisAngle(axis *Vec3, angle float32) *Quat {
	halfAngle := angle / 2
	s := Sin(halfAngle)
	q.X = axis.X * s
	q.Y = axis.Y * s
	q.Z = axis.Z * s
	q.W = Cos(halfAngle)
	return q
}

// SetFromRotationMatrix sets this quaternion from the specified rotation matrix.
// Returns pointer to this updated quaternion.
func (q *Quat) SetFromRotationMatrix(m *Mat4) *Quat {
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
	return q
}

// SetFromUnitVectors sets this quaternion to the rotation from vector vFrom to vTo.
// The vectors must be normalized.
// Returns pointer to this updated quaternion.
func (q *Quat) SetFromUnitVectors(vFrom, vTo *Vec3) *Quat {
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

		v1.CrossVectors(vFrom, vTo)

	}
	q.X = v1.X
	q.Y = v1.Y
	q.Z = v1.Z
	q.W = r

	q.Normalize()

	return q
}

// Inverse sets this quaternion to its inverse.
// Returns pointer to this updated quaternion.
func (q *Quat) Inverse() *Quat {
	q.Conjugate().Normalize()
	return q
}

// Conjugate sets this quaternion to its conjugate.
// Returns pointer to this updated quaternion.
func (q *Quat) Conjugate() *Quat {
	q.X *= -1
	q.Y *= -1
	q.Z *= -1
	return q
}

// Dot returns the dot products of this quaternion with other.
func (q *Quat) Dot(other *Quat) float32 {
	return q.X*other.X + q.Y*other.Y + q.Z*other.Z + q.W*other.W
}

// LengthSq returns this quanternion's length squared
func (q *Quat) lengthSq() float32 {
	return q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W
}

// Length returns the length of this quaternion
func (q *Quat) Length() float32 {
	return Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)
}

// Normalize normalizes this quaternion.
// Returns pointer to this updated quaternion.
func (q *Quat) Normalize() *Quat {
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
	return q
}

// NormalizeFast approximates normalizing this quaternion.
// Works best when the quaternion is already almost-normalized.
// Returns pointer to this updated quaternion.
func (q *Quat) NormalizeFast() *Quat {
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
	return q
}

// Multiply sets this quaternion to the multiplication of itself by other.
// Returns pointer to this updated quaternion.
func (q *Quat) Multiply(other *Quat) *Quat {
	return q.MultiplyQuats(q, other)
}

// MultiplyQuats set this quaternion to the multiplication of a by b.
// Returns pointer to this updated quaternion.
func (q *Quat) MultiplyQuats(a, b *Quat) *Quat {
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
	return q
}

// Slerp sets this quaternion to another quaternion which is the spherically linear interpolation
// from this quaternion to other using t.
// Returns pointer to this updated quaternion.
func (q *Quat) Slerp(other *Quat, t float32) *Quat {
	if t == 0 {
		return q
	}
	if t == 1 {
		return q.Copy(other)
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
		q.Copy(other)
	}

	if cosHalfTheta >= 1.0 {
		q.W = w
		q.X = x
		q.Y = y
		q.Z = z
		return q
	}

	sqrSinHalfTheta := 1.0 - cosHalfTheta*cosHalfTheta
	if sqrSinHalfTheta < 0.001 {
		s := 1 - t
		q.W = s*w + t*q.W
		q.X = s*x + t*q.X
		q.Y = s*y + t*q.Y
		q.Z = s*z + t*q.Z
		return q.Normalize()
	}

	sinHalfTheta := Sqrt(sqrSinHalfTheta)
	halfTheta := Atan2(sinHalfTheta, cosHalfTheta)
	ratioA := Sin((1-t)*halfTheta) / sinHalfTheta
	ratioB := Sin(t*halfTheta) / sinHalfTheta

	q.W = w*ratioA + q.W*ratioB
	q.X = x*ratioA + q.X*ratioB
	q.Y = y*ratioA + q.Y*ratioB
	q.Z = z*ratioA + q.Z*ratioB

	return q
}

// Equals returns if this quaternion is equal to other.
func (q *Quat) Equals(other *Quat) bool {
	return (other.X == q.X) && (other.Y == q.Y) && (other.Z == q.Z) && (other.W == q.W)
}

// FromArray sets this quaternion's components from array starting at offset.
// Returns pointer to this updated quaternion.
func (q *Quat) FromArray(array []float32, offset int) *Quat {
	q.X = array[offset]
	q.Y = array[offset+1]
	q.Z = array[offset+2]
	q.W = array[offset+3]
	return q
}

// ToArray copies this quaternions's components to array starting at offset.
// Returns pointer to this updated array.
func (q *Quat) ToArray(array []float32, offset int) []float32 {
	array[offset] = q.X
	array[offset+1] = q.Y
	array[offset+2] = q.Z
	array[offset+3] = q.W

	return array
}

// Clone returns a copy of this quaternion
func (q *Quat) Clone() *Quat {
	return NewQuat(q.X, q.Y, q.Z, q.W)
}
