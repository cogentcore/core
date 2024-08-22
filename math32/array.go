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
	"unsafe"

	"cogentcore.org/core/base/slicesx"
)

// ArrayF32 is a slice of float32 with additional convenience methods
// for other math32 data types.  Use slicesx.SetLength to set length
// efficiently.
type ArrayF32 []float32

// NewArrayF32 creates a returns a new array of floats
// with the specified initial size and capacity
func NewArrayF32(size, capacity int) ArrayF32 {
	return make([]float32, size, capacity)
}

// NumBytes returns the size of the array in bytes
func (a *ArrayF32) NumBytes() int {
	return len(*a) * int(unsafe.Sizeof(float32(0)))
}

// Append appends any number of values to the array
func (a *ArrayF32) Append(v ...float32) {
	*a = append(*a, v...)
}

// AppendVector2 appends any number of Vector2 to the array
func (a *ArrayF32) AppendVector2(v ...Vector2) {
	for i := 0; i < len(v); i++ {
		*a = append(*a, v[i].X, v[i].Y)
	}
}

// AppendVector3 appends any number of Vector3 to the array
func (a *ArrayF32) AppendVector3(v ...Vector3) {
	for i := 0; i < len(v); i++ {
		*a = append(*a, v[i].X, v[i].Y, v[i].Z)
	}
}

// AppendVector4 appends any number of Vector4 to the array
func (a *ArrayF32) AppendVector4(v ...Vector4) {
	for i := 0; i < len(v); i++ {
		*a = append(*a, v[i].X, v[i].Y, v[i].Z, v[i].W)
	}
}

// CopyFloat32s copies a []float32 slice from src into target,
// ensuring that the target is the correct size.
func CopyFloat32s(trg *[]float32, src []float32) {
	*trg = slicesx.SetLength(*trg, len(src))
	copy(*trg, src)
}

// CopyFloat64s copies a []float64 slice from src into target,
// ensuring that the target is the correct size.
func CopyFloat64s(trg *[]float64, src []float64) {
	*trg = slicesx.SetLength(*trg, len(src))
	copy(*trg, src)
}

func (a *ArrayF32) CopyFrom(src ArrayF32) {
	CopyFloat32s((*[]float32)(a), src)
}

// GetVector2 stores in the specified Vector2 the
// values from the array starting at the specified pos.
func (a ArrayF32) GetVector2(pos int, v *Vector2) {
	v.X = a[pos]
	v.Y = a[pos+1]
}

// GetVector3 stores in the specified Vector3 the
// values from the array starting at the specified pos.
func (a ArrayF32) GetVector3(pos int, v *Vector3) {
	v.X = a[pos]
	v.Y = a[pos+1]
	v.Z = a[pos+2]
}

// GetVector4 stores in the specified Vector4 the
// values from the array starting at the specified pos.
func (a ArrayF32) GetVector4(pos int, v *Vector4) {
	v.X = a[pos]
	v.Y = a[pos+1]
	v.Z = a[pos+2]
	v.W = a[pos+3]
}

// GetMatrix4 stores in the specified Matrix4 the
// values from the array starting at the specified pos.
func (a ArrayF32) GetMatrix4(pos int, m *Matrix4) {
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

// SetVector2 sets the values of the array at the specified pos
// from the XY values of the specified Vector2
func (a ArrayF32) SetVector2(pos int, v Vector2) {
	a[pos] = v.X
	a[pos+1] = v.Y
}

// SetVector3 sets the values of the array at the specified pos
// from the XYZ values of the specified Vector3
func (a ArrayF32) SetVector3(pos int, v Vector3) {
	a[pos] = v.X
	a[pos+1] = v.Y
	a[pos+2] = v.Z
}

// SetVector4 sets the values of the array at the specified pos
// from the XYZ values of the specified Vector4
func (a ArrayF32) SetVector4(pos int, v Vector4) {
	a[pos] = v.X
	a[pos+1] = v.Y
	a[pos+2] = v.Z
	a[pos+3] = v.W
}

/////////////////////////////////////////////////////////////////////////////////////
//   ArrayU32

// ArrayU32 is a slice of uint32 with additional convenience methods.
// Use slicesx.SetLength to set length efficiently.
type ArrayU32 []uint32

// NewArrayU32 creates a returns a new array of uint32
// with the specified initial size and capacity
func NewArrayU32(size, capacity int) ArrayU32 {
	return make([]uint32, size, capacity)
}

// NumBytes returns the size of the array in bytes
func (a *ArrayU32) NumBytes() int {
	return len(*a) * int(unsafe.Sizeof(uint32(0)))
}

// Append appends n elements to the array updating the slice if necessary
func (a *ArrayU32) Append(v ...uint32) {
	*a = append(*a, v...)
}

// Set sets the values of the array starting at the specified pos
// from the specified values
func (a ArrayU32) Set(pos int, v ...uint32) {
	for i, vv := range v {
		a[pos+i] = vv
	}
}
