// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"github.com/goki/ki/kit"

	vk "github.com/vulkan-go/vulkan"
)

// See: https://www.khronos.org/opengl/wiki/Data_Type_(GLSL)

// Types is a list of supported GPU data types, which can be stored
// properly aligned in device memory, and used by the shader code.
// Note that a Vec3 or arrays of single scalar values such as Float32
// are not well supported outside of Vertex due to the std410 convention:
// http://www.opengl.org/registry/doc/glspec45.core.pdf#page=159
// The Struct type is particularly challenging as each member
// must be aligned in general on a 16 byte boundary (i.e., vec4)
// (unless all elements are exactly 4 bytes, which might work?).
// Go automatically aligns members to 8 bytes on 64 bit machines,
// but that doesn't quite cut it.
type Types int32

const (
	UndefType Types = iota
	Bool32

	Int32
	Int32Vec2
	Int32Vec4

	Uint32
	Uint32Vec2
	Uint32Vec4

	Float32
	Float32Vec2
	Float32Vec3 // note: only use for vertex data -- not properly aligned for uniforms
	Float32Vec4

	Float64
	Float64Vec2
	Float64Vec3
	Float64Vec4

	Float32Mat4 // std xform matrix: mat32.Mat4 works directly

	ImageRGBA32 // 32 bits with 8 bits per component of R,G,B,A -- std image format

	Struct
	TypesN
)

//go:generate stringer -type=Types

var KiT_Types = kit.Enums.AddEnum(TypesN, kit.NotBitFlag, nil)

// TypeSizes gives size of each type in bytes
var TypeSizes = map[Types]int{
	UndefType:   0,
	Int32:       4,
	Int32Vec2:   8,
	Int32Vec4:   16,
	Uint32:      4,
	Uint32Vec2:  8,
	Uint32Vec4:  16,
	Float32:     4,
	Float32Vec2: 8,
	Float32Vec3: 12,
	Float32Vec4: 16,
	Float64:     8,
	Float64Vec2: 16,
	Float64Vec3: 24,
	Float64Vec4: 32,
	Float32Mat4: 64,
	ImageRGBA32: 32,
	Struct:      0,
}

var VulkanTypes = map[Types]vk.Format{
	Bool32:      vk.FormatR32Uint,
	Int32:       vk.FormatR32Sint,
	Int32Vec2:   vk.FormatR32g32Sint,
	Int32Vec4:   vk.FormatR32g32b32a32Sint,
	Uint32:      vk.FormatR32Uint,
	Uint32Vec2:  vk.FormatR32g32Uint,
	Uint32Vec4:  vk.FormatR32g32b32a32Uint,
	Float32:     vk.FormatR32Sfloat,
	Float32Vec2: vk.FormatR32g32Sfloat,
	Float32Vec3: vk.FormatR32g32b32Sfloat,
	Float32Vec4: vk.FormatR32g32b32a32Sfloat,
	Float64:     vk.FormatR64Sfloat,
	Float64Vec2: vk.FormatR64g64Sfloat,
	Float64Vec3: vk.FormatR64g64b64Sfloat,
	Float64Vec4: vk.FormatR64g64b64a64Sfloat,
	ImageRGBA32: vk.FormatR8g8b8a8Srgb,
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
