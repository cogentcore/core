// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

import (
	"unsafe"
)

// ArrayF32 is a slice of float32 with additional convenience methods
type ArrayF32 []float32

// NewArrayF32 creates a returns a new array of floats
// with the specified initial size and capacity
func NewArrayF32(size, capacity int) ArrayF32 {
	return make([]float32, size, capacity)
}

// Bytes returns the size of the array in bytes
func (a *ArrayF32) Bytes() int {
	return len(*a) * int(unsafe.Sizeof(float32(0)))
}

// Size returns the number of float32 elements in the array
func (a *ArrayF32) Size() int {
	return len(*a)
}

// Len returns the number of float32 elements in the array
// It is equivalent to Size()
func (a *ArrayF32) Len() int {
	return len(*a)
}

// Extend appends given number of new float elements to end of existing array
func (a *ArrayF32) Extend(addLen int) {
	*a = append(*a, make([]float32, addLen)...)
}

// Append appends any number of values to the array
func (a *ArrayF32) Append(v ...float32) {
	*a = append(*a, v...)
}

// AppendVec2 appends any number of Vec2 to the array
func (a *ArrayF32) AppendVec2(v ...Vec2) {
	for i := 0; i < len(v); i++ {
		*a = append(*a, v[i].X, v[i].Y)
	}
}

// AppendVec3 appends any number of Vec3 to the array
func (a *ArrayF32) AppendVec3(v ...Vec3) {
	for i := 0; i < len(v); i++ {
		*a = append(*a, v[i].X, v[i].Y, v[i].Z)
	}
}

// AppendVec4 appends any number of Vec4 to the array
func (a *ArrayF32) AppendVec4(v ...Vec4) {
	for i := 0; i < len(v); i++ {
		*a = append(*a, v[i].X, v[i].Y, v[i].Z, v[i].W)
	}
}

// CopyFloat32s copies a []float32 slice from src into target
func CopyFloat32s(trg *[]float32, src []float32) {
	*trg = make([]float32, len(src))
	copy(*trg, src)
}

func (a *ArrayF32) CopyFrom(src ArrayF32) {
	CopyFloat32s((*[]float32)(a), src)
}

// GetVec2 stores in the specified Vec2 the
// values from the array starting at the specified pos.
func (a ArrayF32) GetVec2(pos int, v *Vec2) {
	v.X = a[pos]
	v.Y = a[pos+1]
}

// GetVec3 stores in the specified Vec3 the
// values from the array starting at the specified pos.
func (a ArrayF32) GetVec3(pos int, v *Vec3) {
	v.X = a[pos]
	v.Y = a[pos+1]
	v.Z = a[pos+2]
}

// GetVec4 stores in the specified Vec4 the
// values from the array starting at the specified pos.
func (a ArrayF32) GetVec4(pos int, v *Vec4) {
	v.X = a[pos]
	v.Y = a[pos+1]
	v.Z = a[pos+2]
	v.W = a[pos+3]
}

// GetMat4 stores in the specified Mat4 the
// values from the array starting at the specified pos.
func (a ArrayF32) GetMat4(pos int, m *Mat4) {
	m[0] = a[pos]
	m[1] = a[pos+1]
	m[2] = a[pos+2]
	m[3] = a[pos+3]
	m[4] = a[pos+4]
	m[5] = a[pos+5]
	m[6] = a[pos+6]
	m[7] = a[pos+7]
	m[8] = a[pos+8]
	m[9] = a[pos+9]
	m[10] = a[pos+10]
	m[11] = a[pos+11]
	m[12] = a[pos+12]
	m[13] = a[pos+13]
	m[14] = a[pos+14]
	m[15] = a[pos+15]
}

// Set sets the values of the array starting at the specified pos
// from the specified values
func (a ArrayF32) Set(pos int, v ...float32) {
	for i, vv := range v {
		a[pos+i] = vv
	}
}

// SetVec2 sets the values of the array at the specified pos
// from the XY values of the specified Vec2
func (a ArrayF32) SetVec2(pos int, v Vec2) {
	a[pos] = v.X
	a[pos+1] = v.Y
}

// SetVec3 sets the values of the array at the specified pos
// from the XYZ values of the specified Vec3
func (a ArrayF32) SetVec3(pos int, v Vec3) {
	a[pos] = v.X
	a[pos+1] = v.Y
	a[pos+2] = v.Z
}

// SetVec4 sets the values of the array at the specified pos
// from the XYZ values of the specified Vec4
func (a ArrayF32) SetVec4(pos int, v Vec4) {
	a[pos] = v.X
	a[pos+1] = v.Y
	a[pos+2] = v.Z
	a[pos+3] = v.W
}

/////////////////////////////////////////////////////////////////////////////////////
//   ArrayU32

// ArrayU32 is a slice of uint32 with additional convenience methods
type ArrayU32 []uint32

// NewArrayU32 creates a returns a new array of uint32
// with the specified initial size and capacity
func NewArrayU32(size, capacity int) ArrayU32 {
	return make([]uint32, size, capacity)
}

// Bytes returns the size of the array in bytes
func (a *ArrayU32) Bytes() int {
	return len(*a) * int(unsafe.Sizeof(uint32(0)))
}

// Size returns the number of float32 elements in the array
func (a *ArrayU32) Size() int {
	return len(*a)
}

// Len returns the number of float32 elements in the array
func (a *ArrayU32) Len() int {
	return len(*a)
}

// Append appends n elements to the array updating the slice if necessary
func (a *ArrayU32) Append(v ...uint32) {
	*a = append(*a, v...)
}

// Extend appends given number of new elements to end of existing array
func (a *ArrayU32) Extend(addLen int) {
	*a = append(*a, make([]uint32, addLen)...)
}

// Set sets the values of the array starting at the specified pos
// from the specified values
func (a ArrayU32) Set(pos int, v ...uint32) {
	for i, vv := range v {
		a[pos+i] = vv
	}
}
