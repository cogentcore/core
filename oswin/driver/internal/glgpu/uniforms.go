// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"fmt"
	"log"
	"strings"

	"github.com/goki/gi/oswin/gpu"
)

// Uniform represents a single uniform variable, which can be contained within a
// Uniform Buffer Object or used as a separate independent uniform.
// This can be an array of values as well, in which case a NAME_LEN macro is
// always defined to reflect the length of the array.
// These uniforms are used directly to generate the shader code.
// See Program.AddUniform to create a new standalone one, and
// Program.NewUniforms to create a new set of them (i.e., Uniform Buffer Object)
type uniform struct {
	init   bool
	name   string
	handle int32
	typ    gpu.UniType
	array  bool
	ln     int
	offset int
	size   int
}

// Name returns name of the uniform
func (un *uniform) Name() string {
	return un.name
}

// Type returns type of the uniform
func (un *uniform) Type() gpu.UniType {
	return un.typ
}

// Array returns true if this is an array uniform.
// If so, then it automatically generates a #define NAME_LEN <Len> definition prior
// to the uniform definition, and if Len == 0 then it is *not* defined at all.
// All code referencing this uniform should use #if NAME_LEN>0 wrapper.
func (un *uniform) Array() bool {
	return un.array
}

// Len returns number of array elements, if an Array (can be 0)
func (un *uniform) Len() int {
	return un.ln
}

// SetLen sets the number of array elements -- if this is changed, then the associated
// Shader program needs to be re-generated and recompiled.
func (un *uniform) SetLen(ln int) {
	un.ln = ln
}

// Offset returns byte-wise offset into the UBO where this uniform starts (only for UBO's)
func (un *uniform) Offset() int {
	return un.offset
}

// Size() returns byte-wise size of this uniform, *including padding*,
// as determined by the std140 standard opengl layout
func (un *uniform) Size() int {
	return un.size
}

// Handle() returns the unique id for this uniform.
// if in a UBO, then this is the index of the item within the list of UBO's
func (un *uniform) Handle() int32 {
	return un.handle
}

// SetValue sets the value of the uniform to given value, which must be of the corresponding
// elemental or mat32.Vector or mat32.Matrix type.  Proper context must be bound, etc.
func (un *uniform) SetValue(val interface{}) error {
	return nil
}

// LenDefine returns the #define NAME_LEN source code for this uniform, empty if not an array
func (un *uniform) LenDefine() string {
	if !un.array {
		return ""
	}
	unm := strings.ToUpper(un.name)
	return fmt.Sprintf("#define %s_LEN %d\n", unm, un.ln)
}

//////////////////////////////////////////////
// Uniforms

// Uniforms is a set of Uniform objects that are all organized together
// (i.e., a UniformBufferObject in OpenGL)
type uniforms struct {
	init   bool
	name   string
	handle int32
	bindPt int32
	unis   map[string]*uniform
}

// Name returns the name of this set of uniforms
func (un *uniforms) Name() string {
	return un.name
}

// AddUniform adds a uniform variable to this collection of uniforms of given type
func (un *uniforms) AddUniform(name string, typ gpu.UniType, ary bool, ln int) gpu.Uniform {
	if un.unis == nil {
		un.unis = make(map[string]*uniform)
	}
	u := &uniform{name: name, typ: typ, array: ary, ln: ln}
	un.unis[name] = u
	return u
}

// UniformByName returns a Uniform based on unique name -- this could be in a
// collection of Uniforms (i.e., a Uniform Buffer Object in GL) or standalone
func (un *uniforms) UniformByName(name string) gpu.Uniform {
	for _, u := range un.unis {
		if u.name == name {
			return u
		}
	}
	log.Printf("glgpu Uniforms: name %s not found in Uniforms: %s\n", name, un.name)
	return nil
}

// LenDefines returns the #define NAME_LEN source code for all uniforms, empty if no arrays
func (un *uniforms) LenDefines() string {
	defs := ""
	for _, u := range un.unis {
		defs += u.LenDefine()
	}
	return defs
}

// Activate generates the Uniform Buffer Object structure and reserves the binding point
func (un *uniforms) Activate() error {
	un.bindPt = gpu.TheGPU.NextUniformBindingPoint()
	return nil
}

// Bind binds the Uniform Buffer Object structure to given program
func (un *uniforms) Bind(prog gpu.Program) error {
	return nil
}

// Handle returns the handle for the program -- only valid after a Compile call
func (un *uniforms) Handle() int32 {
	return un.handle
}

// BindingPoint returns the unique binding point for this set of Uniforms --
// needed for connecting to programs
func (un *uniforms) BindingPoint() int32 {
	return un.bindPt
}
