// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"github.com/cogentcore/webgpu/wgpu"
)

// See: https://www.khronos.org/opengl/wiki/Data_Type_(GLSL)

// Types is a list of supported GPU data types, which can be stored
// properly aligned in device memory, and used by the shader code.
// Note that a Vector3 or arrays of single scalar values such as Float32
// are not well supported outside of Vertex due to the std410 convention:
// http://www.opengl.org/registry/doc/glspec45.core.pdf#page=159
// The Struct type is particularly challenging as each member
// must be aligned in general on a 16 byte boundary (i.e., vector4)
// (unless all elements are exactly 4 bytes, which might work?).
// Go automatically aligns members to 8 bytes on 64 bit machines,
// but that doesn't quite cut it.
type Types int32 //enums:enum

const (
	UndefinedType Types = iota
	Bool32

	Int16
	Uint16

	Int32
	Int32Vector2
	Int32Vector4

	Uint32
	Uint32Vector2
	Uint32Vector4

	Float32
	Float32Vector2
	Float32Vector3 // note: only use for vertex data -- not properly aligned for uniforms
	Float32Vector4

	Float32Matrix4 // std transform matrix: math32.Matrix4 works directly
	Float32Matrix3 // std transform matrix: math32.Matrix3 works directly

	TextureRGBA32 // 32 bits with 8 bits per component of R,G,B,A -- std image format
	TextureBGRA32

	Depth32         // standard float32 depth buffer
	Depth24Stencil8 // standard 24 bit float with 8 bit stencil

	Struct
)

// VertexFormat returns the WebGPU VertexFormat for given type.
func (tp Types) VertexFormat() wgpu.VertexFormat {
	return TypeToVertexFormat[tp]
}

// TextureFormat returns the WebGPU TextureFormat for given type.
func (tp Types) TextureFormat() wgpu.TextureFormat {
	return TypeToTextureFormat[tp]
}

// IndexType returns the WebGPU VertexFormat for Index var.
// must be either Uint16 or Uint32.
func (tp Types) IndexType() wgpu.IndexFormat {
	if tp == Uint16 {
		return wgpu.IndexFormatUint16
	}
	return wgpu.IndexFormatUint32
}

// Bytes returns number of bytes for this type
func (tp Types) Bytes() int {
	return TypeSizes[tp]
}

var TypeToTextureFormat = map[Types]wgpu.TextureFormat{
	TextureRGBA32:   wgpu.TextureFormatRGBA8UnormSrgb,
	TextureBGRA32:   wgpu.TextureFormatBGRA8UnormSrgb,
	Depth32:         wgpu.TextureFormatDepth32Float,
	Depth24Stencil8: wgpu.TextureFormatDepth24PlusStencil8,
}

// TextureFormatSizes gives size of known WebGPU
// TextureFormats in bytes
var TextureFormatSizes = map[wgpu.TextureFormat]int{
	wgpu.TextureFormatUndefined:           0,
	wgpu.TextureFormatR16Sint:             2,
	wgpu.TextureFormatR16Uint:             2,
	wgpu.TextureFormatR32Sint:             4,
	wgpu.TextureFormatRG32Sint:            8,
	wgpu.TextureFormatR32Uint:             4,
	wgpu.TextureFormatRG32Uint:            8,
	wgpu.TextureFormatRGBA32Uint:          16,
	wgpu.TextureFormatR32Float:            4,
	wgpu.TextureFormatRG32Float:           8,
	wgpu.TextureFormatRGBA32Float:         16,
	wgpu.TextureFormatRGBA8Sint:           4,
	wgpu.TextureFormatRGBA8Unorm:          4,
	wgpu.TextureFormatRGBA8UnormSrgb:      4,
	wgpu.TextureFormatBGRA8Unorm:          4,
	wgpu.TextureFormatBGRA8UnormSrgb:      4,
	wgpu.TextureFormatDepth32Float:        4,
	wgpu.TextureFormatDepth24PlusStencil8: 4,
}

// TypeSizes gives our data type sizes in bytes
var TypeSizes = map[Types]int{
	Bool32: 4,

	Int16:  2,
	Uint16: 2,

	Int32:        4,
	Int32Vector2: 8,
	Int32Vector4: 16,

	Uint32:        4,
	Uint32Vector2: 8,
	Uint32Vector4: 16,

	Float32:        4,
	Float32Vector2: 8,
	Float32Vector3: 12,
	Float32Vector4: 16,

	Float32Matrix4: 64,
	Float32Matrix3: 36,

	TextureRGBA32: 4,

	Depth32:         4,
	Depth24Stencil8: 4,
}

// TypeToVertexFormat maps gpu.Types to WebGPU VertexFormat
var TypeToVertexFormat = map[Types]wgpu.VertexFormat{
	UndefinedType: wgpu.VertexFormatUndefined,
	// Bool32:         wgpu.VertexFormatUint32,
	// Int16:          wgpu.VertexFormatR16Sint,
	// Uint16:         wgpu.VertexFormatR16Uint,
	Int32:          wgpu.VertexFormatSint32,
	Int32Vector2:   wgpu.VertexFormatSint32x2,
	Int32Vector4:   wgpu.VertexFormatSint32x4,
	Uint32:         wgpu.VertexFormatUint32,
	Uint32Vector2:  wgpu.VertexFormatUint32x2,
	Uint32Vector4:  wgpu.VertexFormatUint32x4,
	Float32:        wgpu.VertexFormatFloat32,
	Float32Vector2: wgpu.VertexFormatFloat32x2,
	Float32Vector3: wgpu.VertexFormatFloat32x3,
	Float32Vector4: wgpu.VertexFormatFloat32x4,
	// TextureRGBA32:    wgpu.TextureFormatR8g8b8a8Srgb,
	// Depth32:        wgpu.VertexFormatD32Sfloat,
	// Depth24Stencil8:   wgpu.VertexFormatD24UnormS8Uint,
}

// most commonly available formats: https://WebGPU.gpuinfo.org/listsurfaceformats.php

// TextureFormatNames translates image format into human-readable string
// for most commonly available formats
var TextureFormatNames = map[wgpu.TextureFormat]string{
	wgpu.TextureFormatRGBA8UnormSrgb: "RGBA 8bit sRGB colorspace",
	wgpu.TextureFormatRGBA8Unorm:     "RGBA 8bit unsigned linear colorspace",
	// wgpu.TextureFormatR5g6b5UnormPack16:      "RGB 5bit (pack 16bit total) unsigned linear colorspace",
	// wgpu.TextureFormatA2b10g10r10UnormPack32: "ABGR 10bit, 2bit alpha (pack 32bit total), unsigned linear colorspace",
	// wgpu.TextureFormatB8g8r8a8Srgb:           "BGRA 8bit sRGB colorspace",
	// wgpu.TextureFormatB8g8r8a8Unorm:          "BGRA 8bit unsigned linear colorspace",
	// wgpu.TextureFormatR16g16b16a16Sfloat:     "RGBA 16bit signed floating point linear colorspace",
	// wgpu.TextureFormatA2r10g10b10UnormPack32: "ARGB 10bit, 2bit alpha (pack 32bit total), unsigned linear colorspace",
}
