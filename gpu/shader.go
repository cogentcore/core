// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"log"
	"os"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// todo: should be able to handle multiple types per shader

// Shader manages a single Shader program,
// which can have multiple entry points.
type Shader struct {
	Name   string
	Type   ShaderTypes
	Module *wgpu.ShaderModule
}

// Init initializes the shader
func (sh *Shader) Init(name string, typ ShaderTypes) {
	sh.Name = name
	sh.Type = typ
}

// OpenFile loads given SPIR-V ".spv" code from file for the Shader.
func (sh *Shader) OpenFile(dev *Device, fname string) error {
	if sh.Name == "" {
		sh.Name = fname
	}
	b, err := os.ReadFile(fname)
	if err != nil {
		log.Printf("gpu.Shader OpenFile: %s\n", err)
		return err
	}
	return sh.OpenCode(dev, string(b))
}

// OpenCode loads given WGSL ".wgl" code for the Shader.
func (sh *Shader) OpenCode(dev *Device, code string) error {
	module, err := dev.Device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          sh.Name,
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: code},
	})
	sh.Module = module
	return nil
}

// Destroy destroys the shader
func (sh *Shader) Destroy() {
	if sh.Module == nil {
		return
	}
	sh.Module.Release()
	sh.Module = nil
}

// ShaderTypes is a list of GPU shader types
type ShaderTypes int32

const (
	UnknownShader ShaderTypes = iota
	VertexShader
	FragmentShader
	ComputeShader
)

var ShaderStageFlags = map[ShaderTypes]wgpu.ShaderStage{
	UnknownShader:  wgpu.ShaderStage_None,
	VertexShader:   wgpu.ShaderStage_Vertex,
	FragmentShader: wgpu.ShaderStage_Fragment,
	ComputeShader:  wgpu.ShaderStage_Compute,
}

/*
var ShaderPipelineFlags = map[ShaderTypes]vk.PipelineStageFlagBits{
	VertexShader:   vk.PipelineStageVertexShaderBit,
	TessCtrlShader: vk.PipelineStageTessellationControlShaderBit,
	TessEvalShader: vk.PipelineStageTessellationEvaluationShaderBit,
	GeometryShader: vk.PipelineStageGeometryShaderBit,
	FragmentShader: vk.PipelineStageFragmentShaderBit,
	ComputeShader:  vk.PipelineStageComputeShaderBit,
}
*/
