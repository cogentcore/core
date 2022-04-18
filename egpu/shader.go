// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import (
	"unsafe"

	vk "github.com/vulkan-go/vulkan"
)

// Shader manages a single Shader program
type Shader struct {
	init   bool
	shader vk.ShaderModule
	name   string
	// typ    gpu.ShaderTypes
	src    string
	orgSrc string // original source as provided by user -- program adds extra source..
}

// Name returns the unique name of this Shader
func (sh *Shader) Name() string {
	return sh.name
}

// // Type returns the type of the Shader
// func (sh *Shader) Type() gpu.ShaderTypes {
// 	return sh.typ
// }

// Compile compiles given source code for the Shader, of given type and unique name.
// Currently, source must be GLSL version 410, which is the supported version of OpenGL.
// The source does not need to be null terminated (with \x00 code) but that will be more
// efficient, skipping the extra step of adding the null terminator.
// Context must be set.
func (sh *Shader) Compile(src string) error {
	data := []byte{src}
	var module vk.ShaderModule
	ret := vk.CreateShaderModule(device, &vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint(len(data)),
		PCode:    SliceUint32(data),
	}, nil, &module)
	if IsError(ret) {
		return NewError(ret)
	}
	sh.shader = module
	sh.src = src
	sh.init = true
	return nil
}

// Handle returns the GPU handle for this Shader
func (sh *Shader) Handle() vk.ShaderModule {
	return sh.shader
}

// Source returns the actual final source code for the Shader
// excluding the null terminator (for display purposes).
// This includes extra auto-generated code from the Program.
func (sh *Shader) Source() string {
	return sh.src
}

// OrigSource returns the original user-supplied source code
// excluding the null terminator (for display purposes)
func (sh *Shader) OrigSource() string {
	return sh.orgSrc
}

// Delete deletes the Shader
func (sh *Shader) Delete() {
	if !sh.init {
		return
	}
	// gl.DeleteShader(sh.handle)
	sh.shader = nil
	sh.init = false
}

// GPUType returns the GPU type id of the given Shader type
// func (sh *Shader) GPUType(typ gpu.ShaderTypes) uint32 {
// 	return glShaders[typ]
// }

// var glShaders = map[gpu.ShaderTypes]uint32{
// 	gpu.VertexShader:   gl.VERTEX_SHADER,
// 	gpu.FragmentShader: gl.FRAGMENT_SHADER,
// 	gpu.ComputeShader:  gl.COMPUTE_SHADER,
// 	gpu.GeometryShader: gl.GEOMETRY_SHADER,
// 	gpu.TessCtrlShader: gl.TESS_CONTROL_SHADER,
// 	gpu.TessEvalShader: gl.TESS_EVALUATION_SHADER,
// }

// todo: use 1.17 unsafe.Slice function
// https://stackoverflow.com/questions/11924196/convert-between-slices-of-different-types

func SliceUint32(data []byte) []uint32 {
	const m = 0x7fffffff
	return (*[m / 4]uint32)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&data)).Data))[:len(data)/4]
}

type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
