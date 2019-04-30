// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

// Uniform represents a single uniform variable, which can be contained within a
// Uniform Buffer Object or used as a separate independent uniform.
// This can be an array of values as well, in which case a NAME_LEN macro is
// always defined to reflect the length of the array.
// These uniforms are used directly to generate the shader code.
// See Program.AddUniform to create a new standalone one, and
// Program.NewUniforms to create a new set of them (i.e., Uniform Buffer Object)
type Uniform interface {
	// Name returns name of the uniform
	Name() string

	// Type returns type of the uniform
	Type() UniType

	// Array returns true if this is an array uniform.
	// If so, then it automatically generates a #define NAME_LEN <Len> definition prior
	// to the uniform definition, and if Len == 0 then it is *not* defined at all.
	// All code referencing this uniform should use #if NAME_LEN>0 wrapper.
	Array() bool

	// Len returns number of array elements, if an Array (can be 0)
	Len() int

	// SetLen sets the number of array elements -- if this is changed, then the associated
	// Shader program needs to be re-generated and recompiled.
	SetLen(ln int)

	// Offset returns byte-wise offset into the UBO where this uniform starts (only for UBO's)
	Offset() int

	// Size() returns actual byte-wise size of this uniform raw data (c.f., StdSize)
	Size() int

	// StdSize() returns byte-wise size of this uniform, *including padding* for representation
	// on the GPU -- e.g., as determined by the std140 standard opengl layout
	StdSize() int

	// Handle() returns the unique id for this uniform.
	// if in a UBO, then this is the index of the item within the list of UBO's
	Handle() int32

	// SetValue sets the value of the uniform to given value, which must be of the corresponding
	// elemental or mat32.Vector or mat32.Matrix type.  Proper context must be bound, etc.
	SetValue(val interface{}) error

	// LenDefine returns the #define NAME_LEN source code for this uniform, empty if not an array
	LenDefine() string
}

// Uniforms is a set of Uniform objects that are all organized together
// (i.e., a UniformBufferObject in OpenGL)
type Uniforms interface {
	// Name returns the name of this set of uniforms
	Name() string

	// SetName sets the name of this set of uniforms
	SetName(name string)

	// AddUniform adds a uniform variable to this collection of uniforms of given type
	AddUniform(name string, typ UniType, ary bool, ln int) Uniform

	// UniformByName returns a Uniform based on unique name.
	// returns nil and logs an error if not found
	UniformByName(name string) Uniform

	// LenDefines returns the #define NAME_LEN source code for all uniforms, empty if no arrays
	LenDefines() string

	// Activate generates the Uniform Buffer Object structure and reserves the binding point
	Activate() error

	// Resize resizes the buffer if needed -- call if any of the member uniforms
	// might have been resized.  Calls Activate if not already activated.
	Resize() error

	// Bind binds the Uniform Buffer Object structure to given program
	Bind(prog Program) error

	// Handle returns the handle for the program -- only valid after a Compile call
	Handle() uint32

	// BindingPoint returns the unique binding point for this set of Uniforms --
	// needed for connecting to programs
	BindingPoint() uint32
}
