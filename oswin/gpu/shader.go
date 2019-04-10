// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "github.com/goki/ki/kit"

// ShaderTypes is a list of GPU shader types
type ShaderTypes int32

const (
	UndefShader ShaderTypes = iota
	VertexShader
	FragmentShader
	ComputeShader
	// GeometryShader
	// TessCtrlShader
	// TessEvalShader
	ShaderTypesN
)

//go:generate stringer -type=ShaderTypes

var KiT_ShaderTypes = kit.Enums.AddEnum(ShaderTypesN, false, nil)

// Shader manages a single shader program
type Shader interface {
	// Compile compiles given source code for the shader, of given type and unique name.
	// Currently, source must be GLSL version 410, which is the supported version of OpenGL.
	// The source does not need to be null terminated (with \x00 code) but that will be more
	// efficient, skipping the extra step of adding the null terminator.
	// Context must be set.
	Compile(typ ShaderTypes, name string, src string) error

	// Name returns the unique name of this shader
	Name() string

	// Type returns the type of the shader
	Type() ShaderTypes

	// Source returns the source code for the shader, excluding the null terminator
	// (for display purposes)
	Source() string
}
