// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "github.com/goki/ki/kit"

// Shader manages a single shader program.
// call Program.AddShader to add a new Shader.
type Shader interface {
	// Name returns the unique name of this shader
	Name() string

	// Type returns the type of the shader
	Type() ShaderTypes

	// Compile compiles given source code for the shader, of given type and unique name.
	// Currently, source must be GLSL version 410, which is the supported version of OpenGL.
	// The source does not need to be null terminated (with \x00 code) but that will be more
	// efficient, skipping the extra step of adding the null terminator.
	// Context must be set.
	Compile(src string) error

	// Handle returns the GPU handle for this shader
	Handle() uint32

	// Source returns the actual final source code for the shader
	// excluding the null terminator (for display purposes).
	// This includes extra auto-generated code from the Program.
	Source() string

	// OrigSource returns the original user-supplied source code
	// excluding the null terminator (for display purposes)
	OrigSource() string

	// Delete deletes the GPU resources for shader -- should be deleted after linked into a program.
	Delete()

	// GPUType returns the GPU type id for given shader type
	GPUType(typ ShaderTypes) uint32
}

// ShaderTypes is a list of GPU shader types
type ShaderTypes int32

const (
	VertexShader ShaderTypes = iota
	FragmentShader
	ComputeShader
	GeometryShader
	TessCtrlShader
	TessEvalShader
	ShaderTypesN
)

//go:generate stringer -type=ShaderTypes

var KiT_ShaderTypes = kit.Enums.AddEnum(ShaderTypesN, kit.NotBitFlag, nil)
