// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"io/ioutil"
	"log"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"
)

// Shader manages a single Shader program
type Shader struct {
	Name     string
	Type     ShaderTypes
	VkModule vk.ShaderModule
}

// Init initializes the shader
func (sh *Shader) Init(name string, typ ShaderTypes) {
	sh.Name = name
	sh.Type = typ
}

// OpenFile loads given SPIR-V ".spv" code from file for the Shader.
func (sh *Shader) OpenFile(fname string) error {
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Printf("vgpu.Shader OpenFile: %s\n", err)
		return err
	}
	return sh.OpenCode(b)
}

// OpenCode loads given SPIR-V ".spv" code for the Shader.
func (sh *Shader) OpenCode(code []byte) error {
	var module vk.ShaderModule
	ret := vk.CreateShaderModule(TheGPU.Device.Device, &vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint(len(code)),
		PCode:    SliceUint32(code),
	}, nil, &module)
	if IsError(ret) {
		return NewError(ret)
	}
	sh.VkModule = module
	return nil
}

// Free deletes the shader module, which can be done after the pipeline
// is created.
func (sh *Shader) Free() {
	if sh.VkModule == nil {
		return
	}
	vk.DestroyShaderModule(TheGPU.Device.Device, sh.VkModule, nil)
	sh.VkModule = nil
}

// Delete deletes the Shader
func (sh *Shader) Delete() {
	sh.Free()
}

// todo: use 1.17 unsafe.Slice function
// https://stackoverflow.com/questions/11924196/convert-between-slices-of-different-types

func SliceUint32(data []byte) []uint32 {
	const m = 0x7fffffff
	return (*[m / 4]uint32)(unsafe.Pointer(unsafe.Pointer(&(data[0]))))[:len(data)/4]
}

// ShaderTypes is a list of GPU shader types
type ShaderTypes int32

const (
	VertexShader   = ShaderTypes(vk.PipelineStageVertexShaderBit)
	TessCtrlShader = ShaderTypes(vk.PipelineStageTessellationControlShaderBit)
	TessEvalShader = ShaderTypes(vk.PipelineStageTessellationEvaluationShaderBit)
	GeometryShader = ShaderTypes(vk.PipelineStageGeometryShaderBit)
	FragmentShader = ShaderTypes(vk.PipelineStageFragmentShaderBit)
	ComputeShader  = ShaderTypes(vk.PipelineStageComputeShaderBit)
)
