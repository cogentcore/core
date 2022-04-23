// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"github.com/goki/ki/kit"

	vk "github.com/vulkan-go/vulkan"
)

// See: https://www.khronos.org/opengl/wiki/Data_Type_(GLSL)

// Types is a list of GPU data types
type Types int32

const (
	UndefType = Types(vk.FormatUndefined)
	Bool32    = Types(vk.FormatR32Uint) // not officially supported -- just an "annotation"

	Int32     = Types(vk.FormatR32Sint)
	Int32Vec2 = Types(vk.FormatR32g32Sint)
	Int32Vec3 = Types(vk.FormatR32g32b32Sint)
	Int32Vec4 = Types(vk.FormatR32g32b32a32Sint)

	Uint32     = Types(vk.FormatR32Uint)
	Uint32Vec2 = Types(vk.FormatR32g32Uint)
	Uint32Vec3 = Types(vk.FormatR32g32b32Uint)
	Uint32Vec4 = Types(vk.FormatR32g32b32a32Uint)

	Float32     = Types(vk.FormatR32Sfloat)
	Float32Vec2 = Types(vk.FormatR32g32Sfloat)
	Float32Vec3 = Types(vk.FormatR32g32b32Sfloat)
	Float32Vec4 = Types(vk.FormatR32g32b32a32Sfloat)

	Float64     = Types(vk.FormatR64Sfloat)
	Float64Vec2 = Types(vk.FormatR64g64Sfloat)
	Float64Vec3 = Types(vk.FormatR64g64b64Sfloat)
	Float64Vec4 = Types(vk.FormatR64g64b64a64Sfloat)

	Struct = Types(vk.FormatEndRange + 1)
	TypesN
)

//go:generate stringer -type=Types

var KiT_Types = kit.Enums.AddEnum(TypesN, kit.NotBitFlag, nil)

// TypeSizes gives size of each type in bytes
var TypeSizes = map[Types]int{
	UndefType:   0,
	Int32:       4,
	Int32Vec2:   8,
	Int32Vec3:   12,
	Int32Vec4:   16,
	Uint32:      4,
	Uint32Vec2:  8,
	Uint32Vec3:  12,
	Uint32Vec4:  16,
	Float32:     4,
	Float32Vec2: 8,
	Float32Vec3: 12,
	Float32Vec4: 16,
	Float64:     8,
	Float64Vec2: 16,
	Float64Vec3: 24,
	Float64Vec4: 32,
	Struct:      0,
}

// // GLSL type names
// var TypeNames = map[Types]string{
// 	UndefType: "none",
// 	Bool:      "bool",
// 	Int:       "int",
// 	UInt:      "uint",
// 	Float32:   "float",
// 	Float64:   "double",
// }

// TypeBytes returns number of bytes for given type -- 4 except Float64 = 8
func TypeBytes(tp Types) int {
	if tp >= Float64 {
		return 8
	}
	return 4
}

// UniType represents a fully-specified GPU uniform type, including vectors and matricies
type UniType struct {
	Type Types `desc:"data type"`
	Vec  int   `desc:"if a vector, this is the length of the vector, 0 for scalar (valid values are 2,3,4)"`
	Mat  int   `desc:"square matrix dimensions, if a matrix (valid values are 3,4)"`
}

// Commonly-used types:

// FUniType is a single float32
var FUniType = UniType{Type: Float32}

// IUniType is a single int32
var IUniType = UniType{Type: Int32}

// BUniType is a single bool
var BUniType = UniType{Type: Bool32}

// Vec2fUniType is a 2-vector of float32
var Vec2fUniType = UniType{Type: Float32, Vec: 2}

// Vec3fUniType is a 3-vector of float32
var Vec3fUniType = UniType{Type: Float32, Vec: 3}

// Vec4fUniType is a 4-vector of float32
var Vec4fUniType = UniType{Type: Float32, Vec: 4}

// Mat3fUniType is a 3x3 matrix of float32
var Mat3fUniType = UniType{Type: Float32, Mat: 3}

// Mat4fUniType is a 4x4 matrix of float32
var Mat4fUniType = UniType{Type: Float32, Mat: 4}

// Name returns the full GLSL type name for the type
func (ty *UniType) Name() string {
	// if ty.Vec == 0 && ty.Mat == 0 {
	// 	return TypeNames[ty.Type]
	// }
	// pfx := TypeNames[ty.Type][0:1]
	// if ty.Type == Float32 {
	// 	pfx = ""
	// }
	// if ty.Vec > 0 {
	// 	return fmt.Sprintf("%svec%d", pfx, ty.Vec)
	// } else {
	// 	return fmt.Sprintf("%smat%d", pfx, ty.Mat)
	// }
	return ""
}

// Bytes returns actual size of this element in bytes
func (ty *UniType) Bytes() int {
	n := TypeBytes(ty.Type)
	if ty.Vec == 0 && ty.Mat == 0 {
		return n
	}
	if ty.Vec > 0 {
		return ty.Vec * n
	}
	return ty.Mat * ty.Mat * n
}

// StdBytes returns number of bytes taken up by this element, in std140 format (including padding)
// https://learnopengl.com/Advanced-OpenGL/Advanced-GLSL
func (ty *UniType) StdBytes() int {
	n := TypeBytes(ty.Type)
	if ty.Vec == 0 && ty.Mat == 0 {
		return n
	}
	if ty.Vec > 0 {
		if ty.Vec <= 2 {
			return 2 * n
		}
		return 4 * n
	}
	return ty.Mat * 4 * n
}

// VectorType represents a fully-specified GPU vector type, e.g., for inputs / outputs
// to shader programs
type VectorType struct {
	Type Types `desc:"data type"`
	Vec  int   `desc:"length of vector (valid values are 2,3,4)"`
}

// commonly-used vector types:

// Vec2fVecType is a 2-vector of float32
var Vec2fVecType = VectorType{Type: Float32, Vec: 2}

// Vec3fVecType is a 3-vector of float32
var Vec3fVecType = VectorType{Type: Float32, Vec: 3}

// Vec4fVecType is a 4-vector of float32
var Vec4fVecType = VectorType{Type: Float32, Vec: 4}

// Bytes returns number of bytes per Vector element (len * 4 basically)
func (ty *VectorType) Bytes() int {
	n := TypeBytes(ty.Type)
	return n * ty.Vec
}
