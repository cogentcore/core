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

	vk "github.com/goki/vulkan"
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
func (sh *Shader) OpenFile(dev vk.Device, fname string) error {
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Printf("vgpu.Shader OpenFile: %s\n", err)
		return err
	}
	return sh.OpenCode(dev, b)
}

// OpenCode loads given SPIR-V ".spv" code for the Shader.
func (sh *Shader) OpenCode(dev vk.Device, code []byte) error {
	uicode := SliceUint32(code)
	var module vk.ShaderModule
	ret := vk.CreateShaderModule(dev, &vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint64(len(code)),
		PCode:    uicode,
	}, nil, &module)
	if IsError(ret) {
		return NewError(ret)
	}
	sh.VkModule = module
	return nil
}

// Free deletes the shader module, which can be done after the pipeline
// is created.
func (sh *Shader) Free(dev vk.Device) {
	if sh.VkModule == nil {
		return
	}
	vk.DestroyShaderModule(dev, sh.VkModule, nil)
	sh.VkModule = nil
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
	VertexShader ShaderTypes = iota
	TessCtrlShader
	TessEvalShader
	GeometryShader
	FragmentShader
	ComputeShader
	AllShaders
)

var ShaderStageFlags = map[ShaderTypes]vk.ShaderStageFlagBits{
	VertexShader:   vk.ShaderStageVertexBit,
	TessCtrlShader: vk.ShaderStageTessellationControlBit,
	TessEvalShader: vk.ShaderStageTessellationEvaluationBit,
	GeometryShader: vk.ShaderStageGeometryBit,
	FragmentShader: vk.ShaderStageFragmentBit,
	ComputeShader:  vk.ShaderStageComputeBit,
	AllShaders:     vk.ShaderStageAll,
}

var ShaderPipelineFlags = map[ShaderTypes]vk.PipelineStageFlagBits{
	VertexShader:   vk.PipelineStageVertexShaderBit,
	TessCtrlShader: vk.PipelineStageTessellationControlShaderBit,
	TessEvalShader: vk.PipelineStageTessellationEvaluationShaderBit,
	GeometryShader: vk.PipelineStageGeometryShaderBit,
	FragmentShader: vk.PipelineStageFragmentShaderBit,
	ComputeShader:  vk.PipelineStageComputeShaderBit,
}
