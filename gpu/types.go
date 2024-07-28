// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	vk "github.com/goki/WebGPU"
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
	UndefType Types = iota
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

	ImageRGBA32 // 32 bits with 8 bits per component of R,G,B,A -- std image format

	Depth32      // standard float32 depth buffer
	Depth24Sten8 // standard 24 bit float with 8 bit stencil

	Struct
)

// VertexFormat returns the WebGPU VertexFormat for given type
func (tp Types) VertexFormat() wgpu.VertexFormat {
	return TypeToVertexFormat[tp]
}

// VkIndexType returns the Vulkan vk.IndexType for var
// must be either Uint16 or Uint32
func (tp Types) VkIndexType() vk.IndexType {
	if tp == Uint16 {
		return vk.IndexTypeUint16
	}
	return vk.IndexTypeUint32
}

// Bytes returns number of bytes for this type
func (tp Types) Bytes() int {
	switch tp {
	case Float32Matrix4:
		return 64
	case Float32Matrix3:
		return 36
	}
	if vf, has := WebGPUTypes[tp]; has {
		return FormatSizes[vf]
	}
	return 0
}

// TextureFormatSizes gives size of known WebGPU
// TextureFormats in bytes
var TextureFormatSizes = map[wgpu.TextureFormat]int{
	wgpu.TextureFormat_Undefined:          0,
	wgpu.TextureFormat_R16Sint:            2,
	wgpu.TextureFormat_R16Uint:            2,
	wgpu.TextureFormat_R32Sint:            4,
	wgpu.TextureFormat_RG32Sint:           8,
	wgpu.TextureFormat_RGA32Sint:          16,
	wgpu.TextureFormat_R32Uint:            4,
	wgpu.TextureFormat_RG32Uint:           8,
	wgpu.TextureFormat_RGBA32Uint:         16,
	wgpu.TextureFormat_R32Float:           4,
	wgpu.TextureFormat_RG32Float:          8,
	wgpu.TextureFormat_RGB32Float:         12,
	wgpu.TextureFormat_RGBA32F:            16,
	wgpu.TextureFormat_RGBA8Sint:          4,
	wgpu.TextureFormat_RGBA8Unorm:         4,
	wgpu.TextureFormat_Depth32SFloat:      4,
	wgpu.TextureFormat_Depth24UnormS8Uint: 4,
}

// TypeToVertexFormats maps gpu.Types to WebGPU VertexFormat
var TypeToVertexFormats = map[Types]wgpu.VertexFormat{
	UndefType: wgpu.VertexFormat_Undefined,
	// Bool32:         wgpu.VertexFormat_Uint32,
	// Int16:          wgpu.VertexFormat_R16Sint,
	// Uint16:         wgpu.VertexFormat_R16Uint,
	Int32:          wgpu.VertexFormat_Sint32,
	Int32Vector2:   wgpu.VertexFormat_Sint32x2,
	Int32Vector4:   wgpu.VertexFormat_Sint32x4,
	Uint32:         wgpu.VertexFormat_Uint32,
	Uint32Vector2:  wgpu.VertexFormat_Uint32x2,
	Uint32Vector4:  wgpu.VertexFormat_Uint32x4,
	Float32:        wgpu.VertexFormat_Float32,
	Float32Vector2: wgpu.VertexFormat_Float32x2,
	Float32Vector3: wgpu.VertexFormat_Float32x3,
	Float32Vector4: wgpu.VertexFormat_Float32x4,
	// ImageRGBA32:    wgpu.TextureFormat_R8g8b8a8Srgb,
	// Depth32:        wgpu.VertexFormat_D32Sfloat,
	// Depth24Sten8:   wgpu.VertexFormat_D24UnormS8Uint,
}

// most commonly available formats: https://WebGPU.gpuinfo.org/listsurfaceformats.php

// ImageFormatNames translates image format into human-readable string
// for most commonly available formats
var ImageFormatNames = map[wgpu.TextureFormat]string{
	wgpu.TextureFormat_RGBA8UnormSrgb: "RGBA 8bit sRGB colorspace",
	wgpu.TextureFormat_RGBA8Unorm:     "RGBA 8bit unsigned linear colorspace",
	// wgpu.TextureFormatR5g6b5UnormPack16:      "RGB 5bit (pack 16bit total) unsigned linear colorspace",
	// wgpu.TextureFormatA2b10g10r10UnormPack32: "ABGR 10bit, 2bit alpha (pack 32bit total), unsigned linear colorspace",
	// wgpu.TextureFormatB8g8r8a8Srgb:           "BGRA 8bit sRGB colorspace",
	// wgpu.TextureFormatB8g8r8a8Unorm:          "BGRA 8bit unsigned linear colorspace",
	// wgpu.TextureFormatR16g16b16a16Sfloat:     "RGBA 16bit signed floating point linear colorspace",
	// wgpu.TextureFormatA2r10g10b10UnormPack32: "ARGB 10bit, 2bit alpha (pack 32bit total), unsigned linear colorspace",
}
