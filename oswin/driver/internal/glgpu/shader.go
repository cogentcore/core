// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/goki/gi/oswin/gpu"
)

// Shader manages a single shader program
type shader struct {
	init   bool
	handle uint32
	name   string
	typ    gpu.ShaderTypes
	src    string
	orgSrc string // original source as provided by user -- program adds extra source..
}

// Name returns the unique name of this shader
func (sh *shader) Name() string {
	return sh.name
}

// Type returns the type of the shader
func (sh *shader) Type() gpu.ShaderTypes {
	return sh.typ
}

// Compile compiles given source code for the shader, of given type and unique name.
// Currently, source must be GLSL version 410, which is the supported version of OpenGL.
// The source does not need to be null terminated (with \x00 code) but that will be more
// efficient, skipping the extra step of adding the null terminator.
// Context must be set.
func (sh *shader) Compile(src string) error {
	handle := gl.CreateShader(sh.GPUType(sh.typ))

	sh.src = src
	csrc := gpu.CString(src)

	csources, free := gl.Strs(csrc)
	gl.ShaderSource(handle, 1, csources, nil)
	free()
	gl.CompileShader(handle)

	var status int32
	gl.GetShaderiv(handle, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(handle, gl.INFO_LOG_LENGTH, &logLength)

		msg := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(handle, logLength, nil, gl.Str(msg))

		err := fmt.Errorf("failed to compile:\n%v\nerror: %v", src, msg)
		log.Printf("glos GPU CompileShader %v\n", err)
		return err
	}

	sh.handle = handle
	sh.init = true
	return nil
}

// Handle returns the GPU handle for this shader
func (sh *shader) Handle() uint32 {
	return sh.handle
}

// Source returns the actual final source code for the shader
// excluding the null terminator (for display purposes).
// This includes extra auto-generated code from the Program.
func (sh *shader) Source() string {
	return gpu.GoString(sh.src)
}

// OrigSource returns the original user-supplied source code
// excluding the null terminator (for display purposes)
func (sh *shader) OrigSource() string {
	return gpu.GoString(sh.orgSrc)
}

// Delete deletes the shader
func (sh *shader) Delete() {
	if !sh.init {
		return
	}
	gl.DeleteShader(sh.handle)
	sh.handle = 0
	sh.init = false
}

// GPUType returns the GPU type id of the given shader type
func (sh *shader) GPUType(typ gpu.ShaderTypes) uint32 {
	return glShaders[typ]
}

var glShaders = map[gpu.ShaderTypes]uint32{
	gpu.VertexShader:   gl.VERTEX_SHADER,
	gpu.FragmentShader: gl.FRAGMENT_SHADER,
	gpu.ComputeShader:  gl.COMPUTE_SHADER,
	gpu.GeometryShader: gl.GEOMETRY_SHADER,
	gpu.TessCtrlShader: gl.TESS_CONTROL_SHADER,
	gpu.TessEvalShader: gl.TESS_EVALUATION_SHADER,
}
