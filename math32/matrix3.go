// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit Cogent Core functionality.

package math32

import "errors"

// Matrix3 is 3x3 matrix organized internally as column matrix,
// with columns as the inner dimension: element (row i, column j) is at
// index j*3+i.
//
// All of the Mul methods multiply the vector or point on the right
// (m * v), as in [Matrix2], [Matrix4] and the WGSL mat3x3f type, so a
// chain of [Matrix3.Mul] calls applies its transforms right to left.
type Matrix3 [9]float32

// note: matrix indexes and dimensions are row,column.

// Mat3 constructs a new Matrix3 from given values, in column-wise order.
func Mat3(n11, n21, n31, n12, n22, n32, n13, n23, n33 float32) Matrix3 {
	m := Matrix3{}
	m.Set(n11, n21, n31, n12, n22, n32, n13, n23, n33)
	return m
}

// Identity3 returns a new identity [Matrix3] matrix.
func Identity3() Matrix3 {
	m := Matrix3{}
	m.SetIdentity()
	return m
}

func Matrix3FromMatrix2(m Matrix2) Matrix3 {
	nm := Matrix3{}
	nm.SetFromMatrix2(m)
	return nm
}

func Matrix3FromMatrix4(m *Matrix4) Matrix3 {
	nm := Matrix3{}
	nm.SetFromMatrix4(m)
	return nm
}

// Matrix3Translate2D returns a Matrix3 2D matrix with given translations
func Matrix3Translate2D(x, y float32) Matrix3 {
	return Matrix3FromMatrix2(Translate2D(x, y))
}

// Matrix3Scale2D returns a Matrix3 2D matrix with given scaling factors
func Matrix3Scale2D(x, y float32) Matrix3 {
	return Matrix3FromMatrix2(Scale2D(x, y))
}

// Rotate2D returns a Matrix2 2D matrix with given rotation, specified in radians
func Matrix3Rotate2D(angle float32) Matrix3 {
	return Matrix3FromMatrix2(Rotate2D(angle))
}

// Set sets all the elements of the matrix col by col starting at row1, column1,
// row2, column1, row3, column1 and so forth.
func (m *Matrix3) Set(n11, n21, n31, n12, n22, n32, n13, n23, n33 float32) {
	m[0] = n11
	m[1] = n21
	m[2] = n31
	m[3] = n12
	m[4] = n22
	m[5] = n32
	m[6] = n13
	m[7] = n23
	m[8] = n33
}

// SetFromMatrix4 sets the matrix elements based on a Matrix4.
func (m *Matrix3) SetFromMatrix4(src *Matrix4) {
	m.Set(
		src[0], src[1], src[2],
		src[4], src[5], src[6],
		src[8], src[9], src[10],
	)
}

// SetFromMatrix2 sets the matrix elements based on a Matrix2,
// as a standard homogeneous 2D affine matrix with the translation
// in the third column, so that [Matrix3.MulPoint] on the result
// matches [Matrix2.MulPoint] on the source.
func (m *Matrix3) SetFromMatrix2(src Matrix2) {
	m.Set(
		src.XX, src.YX, 0,
		src.XY, src.YY, 0,
		src.X0, src.Y0, 1,
	)
}

// FromArray sets this matrix array starting at offset.
func (m *Matrix3) FromArray(array []float32, offset int) {
	copy(m[:], array[offset:])
}

// ToArray copies this matrix to array starting at offset.
func (m Matrix3) ToArray(array []float32, offset int) {
	copy(array[offset:], m[:])
}

// SetIdentity sets this matrix as the identity matrix.
func (m *Matrix3) SetIdentity() {
	m.Set(
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	)
}

// SetZero sets this matrix as the zero matrix.
func (m *Matrix3) SetZero() {
	m.Set(
		0, 0, 0,
		0, 0, 0,
		0, 0, 0,
	)
}

// CopyFrom copies from source matrix into this matrix
// (a regular = assign does not copy data, just the pointer!)
func (m *Matrix3) CopyFrom(src Matrix3) {
	copy(m[:], src[:])
}

// MulMatrices sets this matrix as the matrix multiplication a * b,
// in the standard sense: element (row i, column j) of the result is
// row i of a dotted with column j of b. This matches
// [Matrix4.MulMatrices] and the WGSL mat3x3f * operator.
func (m *Matrix3) MulMatrices(a, b Matrix3) {
	a11 := a[0]
	a21 := a[1]
	a31 := a[2]
	a12 := a[3]
	a22 := a[4]
	a32 := a[5]
	a13 := a[6]
	a23 := a[7]
	a33 := a[8]

	b11 := b[0]
	b21 := b[1]
	b31 := b[2]
	b12 := b[3]
	b22 := b[4]
	b32 := b[5]
	b13 := b[6]
	b23 := b[7]
	b33 := b[8]

	m[0] = a11*b11 + a12*b21 + a13*b31
	m[1] = a21*b11 + a22*b21 + a23*b31
	m[2] = a31*b11 + a32*b21 + a33*b31

	m[3] = a11*b12 + a12*b22 + a13*b32
	m[4] = a21*b12 + a22*b22 + a23*b32
	m[5] = a31*b12 + a32*b22 + a33*b32

	m[6] = a11*b13 + a12*b23 + a13*b33
	m[7] = a21*b13 + a22*b23 + a23*b33
	m[8] = a31*b13 + a32*b23 + a33*b33
}

// Mul returns this matrix times other matrix, as the standard product
// m * other (this matrix is unchanged). Because the Mul methods multiply
// the vector or point on the right, the transform of other is applied
// before the transform of this matrix.
func (m Matrix3) Mul(other Matrix3) Matrix3 {
	nm := Matrix3{}
	nm.MulMatrices(m, other)
	return nm
}

// SetMul sets this matrix to the standard product of this matrix * other.
func (m *Matrix3) SetMul(other Matrix3) {
	m.MulMatrices(*m, other)
}

// MulScalar returns each of this matrix's components multiplied by the specified
// scalar, leaving the original matrix unchanged.
func (m Matrix3) MulScalar(s float32) Matrix3 {
	m.SetMulScalar(s)
	return m
}

// SetMulScalar multiplies each of this matrix's components by the specified scalar.
func (m *Matrix3) SetMulScalar(s float32) {
	m[0] *= s
	m[1] *= s
	m[2] *= s
	m[3] *= s
	m[4] *= s
	m[5] *= s
	m[6] *= s
	m[7] *= s
	m[8] *= s
}

// MulVector multiplies the Vector2 as a vector without adding translations.
// This is for directional vectors and not points.
func (a Matrix3) MulVector(v Vector2) Vector2 {
	tx := a[0]*v.X + a[3]*v.Y
	ty := a[1]*v.X + a[4]*v.Y
	return Vec2(tx, ty)
}

// MulPoint multiplies the Vector2 as a point, including adding translations.
func (a Matrix3) MulPoint(v Vector2) Vector2 {
	tx := a[0]*v.X + a[3]*v.Y + a[6]
	ty := a[1]*v.X + a[4]*v.Y + a[7]
	return Vec2(tx, ty)
}

// MulVector3 multiplies the Vector3 on the right (a * v),
// as a standard matrix multiply.
func (a Matrix3) MulVector3(v Vector3) Vector3 {
	return Vec3(a[0]*v.X+a[3]*v.Y+a[6]*v.Z,
		a[1]*v.X+a[4]*v.Y+a[7]*v.Z,
		a[2]*v.X+a[5]*v.Y+a[8]*v.Z)
}

// MulVector3Array multiplies count vectors (i.e., 3 sequential array values per each increment in count)
// in the array starting at start index by this matrix.
func (m *Matrix3) MulVector3Array(array []float32, start, count int) {
	var v1 Vector3
	j := start
	for i := 0; i < count; i++ {
		v1.FromSlice(array, j)
		mv := v1.MulMatrix3(m)
		mv.ToSlice(array, j)
		j += 3
	}
}

// Determinant calculates and returns the determinant of this matrix.
func (m *Matrix3) Determinant() float32 {
	return m[0]*m[4]*m[8] -
		m[0]*m[5]*m[7] -
		m[1]*m[3]*m[8] +
		m[1]*m[5]*m[6] +
		m[2]*m[3]*m[7] -
		m[2]*m[4]*m[6]
}

// SetInverse sets this matrix to the inverse of the src matrix.
// If the src matrix cannot be inverted returns error and
// sets this matrix to the identity matrix.
func (m *Matrix3) SetInverse(src Matrix3) error {
	n11 := src[0]
	n21 := src[1]
	n31 := src[2]
	n12 := src[3]
	n22 := src[4]
	n32 := src[5]
	n13 := src[6]
	n23 := src[7]
	n33 := src[8]

	t11 := n33*n22 - n32*n23
	t12 := n32*n13 - n33*n12
	t13 := n23*n12 - n22*n13

	det := n11*t11 + n21*t12 + n31*t13

	// no inverse
	if det == 0 {
		m.SetIdentity()
		return errors.New("cannot invert matrix, determinant is 0")
	}

	detInv := 1 / det

	m[0] = t11 * detInv
	m[1] = (n31*n23 - n33*n21) * detInv
	m[2] = (n32*n21 - n31*n22) * detInv
	m[3] = t12 * detInv
	m[4] = (n33*n11 - n31*n13) * detInv
	m[5] = (n31*n12 - n32*n11) * detInv
	m[6] = t13 * detInv
	m[7] = (n21*n13 - n23*n11) * detInv
	m[8] = (n22*n11 - n21*n12) * detInv

	return nil
}

// Inverse returns the inverse of this matrix.
// If the matrix cannot be inverted it silently
// sets this matrix to the identity matrix.
// See Try version for error.
func (m Matrix3) Inverse() Matrix3 {
	nm := Matrix3{}
	nm.SetInverse(m)
	return nm
}

// InverseTry returns the inverse of this matrix.
// If the matrix cannot be inverted returns error and
// sets this matrix to the identity matrix.
func (m Matrix3) InverseTry() (Matrix3, error) {
	nm := Matrix3{}
	err := nm.SetInverse(m)
	return nm, err
}

// SetTranspose transposes this matrix.
func (m *Matrix3) SetTranspose() {
	m[1], m[3] = m[3], m[1]
	m[2], m[6] = m[6], m[2]
	m[5], m[7] = m[7], m[5]
}

// Transpose returns the transpose of this matrix.
func (m Matrix3) Transpose() Matrix3 {
	nm := m
	nm.SetTranspose()
	return nm
}

// ScaleCols returns matrix with columns multiplied by the vector components.
// This can be used when multiplying this matrix by a diagonal matrix if we store
// the diagonal components as a vector.
func (m *Matrix3) ScaleCols(v Vector3) *Matrix3 {
	nm := &Matrix3{}
	*nm = *m
	nm.SetScaleCols(v)
	return nm
}

// SetScaleCols multiplies the matrix columns by the vector components.
// This can be used when multiplying this matrix by a diagonal matrix if we store
// the diagonal components as a vector.
func (m *Matrix3) SetScaleCols(v Vector3) {
	m[0] *= v.X
	m[1] *= v.X
	m[2] *= v.X
	m[3] *= v.Y
	m[4] *= v.Y
	m[5] *= v.Y
	m[6] *= v.Z
	m[7] *= v.Z
	m[8] *= v.Z
}

//////// Special functions

// SetNormalMatrix set this matrix to the matrix that can transform the normal vectors
// from the src matrix which is used transform the vertices (e.g., a ModelView matrix).
// If the src matrix cannot be inverted returns error.
func (m *Matrix3) SetNormalMatrix(src *Matrix4) error {
	var err error
	*m, err = Matrix3FromMatrix4(src).InverseTry()
	m.SetTranspose()
	return err
}

// SetRotationFromQuat sets this matrix as a rotation matrix from the specified [Quat].
func (m *Matrix3) SetRotationFromQuat(q Quat) {
	x := q.X
	y := q.Y
	z := q.Z
	w := q.W
	x2 := x + x
	y2 := y + y
	z2 := z + z
	xx := x * x2
	xy := x * y2
	xz := x * z2
	yy := y * y2
	yz := y * z2
	zz := z * z2
	wx := w * x2
	wy := w * y2
	wz := w * z2

	m[0] = 1 - (yy + zz)
	m[3] = xy - wz
	m[6] = xz + wy

	m[1] = xy + wz
	m[4] = 1 - (xx + zz)
	m[7] = yz - wx

	m[2] = xz - wy
	m[5] = yz + wx
	m[8] = 1 - (xx + yy)
}
