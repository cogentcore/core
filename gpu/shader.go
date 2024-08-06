// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"io/fs"
	"os"
	"path/filepath"

	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
)

// Shader manages a single Shader program, which can have multiple
// entry points. See ShaderEntry for entry points into Shaders.
type Shader struct {
	Name string

	module *wgpu.ShaderModule

	device Device
}

func NewShader(name string, dev *Device) *Shader {
	sh := &Shader{Name: name}
	sh.device = *dev
	return sh
}

// OpenFile loads given WGSL ".wgl" code from file for the Shader.
func (sh *Shader) OpenFile(fname string) error {
	if sh.Name == "" {
		sh.Name = fname
	}
	b, err := os.ReadFile(fname)
	if errors.Log(err) != nil {
		return err
	}
	cp := errors.Log1(os.Getwd())
	return sh.OpenCode(IncludeFS(os.DirFS(cp), "", string(b))) // todo: maybe not ideal
}

// OpenFileFS loads given WGSL ".wgl" code from file for the Shader.
func (sh *Shader) OpenFileFS(fsys fs.FS, fname string) error {
	if sh.Name == "" {
		sh.Name = fname
	}
	b, err := fs.ReadFile(fsys, fname)
	if errors.Log(err) != nil {
		return err
	}
	return sh.OpenCode(IncludeFS(fsys, filepath.Dir(fname), string(b)))
}

// OpenCode loads given WGSL ".wgl" code for the Shader.
func (sh *Shader) OpenCode(code string) error {
	module, err := sh.device.Device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          sh.Name,
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: code},
	})
	if errors.Log(err) != nil {
		return err
	}
	sh.module = module
	return nil
}

// Release destroys the shader
func (sh *Shader) Release() {
	if sh.module == nil {
		return
	}
	sh.module.Release()
	sh.module = nil
}

// ShaderEntry is an entry point into a Shader.  There can be multiple
// entry points per shader.
type ShaderEntry struct {
	// Shader has the code
	Shader *Shader

	// Type of shader entry point.
	Type ShaderTypes

	// Entry is the name of the function to call for this Entry.
	// Conventionally, it is some variant on "main"
	Entry string
}

// NewShaderEntry returns a new ShaderEntry with given settings
func NewShaderEntry(sh *Shader, typ ShaderTypes, entry string) *ShaderEntry {
	se := &ShaderEntry{Shader: sh, Type: typ, Entry: entry}
	return se
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
	UnknownShader:  wgpu.ShaderStageNone,
	VertexShader:   wgpu.ShaderStageVertex,
	FragmentShader: wgpu.ShaderStageFragment,
	ComputeShader:  wgpu.ShaderStageCompute,
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
