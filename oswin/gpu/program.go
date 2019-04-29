// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

// Program manages a set of shaders and associated variables and uniforms.
// Multiple programs can be assembled into a Pipeline, which can create
// new Programs.  GPU.NewProgram() can also create standalone Programs.
// All uniforms must be added before compiling program.
type Program interface {
	// Name returns name of program
	Name() string

	// SetName sets name of program
	SetName(name string)

	// AddShader adds shader of given type, unique name and source code.
	// Any array uniform's will add their #define NAME_LEN's to the top
	// of the source code automatically, so the source can assume those exist
	// when compiled (NAME is uppper-cased version of variable name).
	// Also the appropriate #version is added automatically.
	AddShader(typ ShaderTypes, name string, src string) (Shader, error)

	// ShaderByName returns shader by its unique name
	ShaderByName(name string) Shader

	// ShaderByType returns shader by its type
	ShaderByType(typ ShaderTypes) Shader

	// SetFragDataVar sets the variable name to use for the fragment shader's output
	SetFragDataVar(name string)

	// AddUniform adds an individual standalone uniform variable to the program of given type.
	// Must add all uniform variables before compiling, as they add to source.
	AddUniform(name string, typ UniType, ary bool, ln int) Uniform

	// NewUniforms makes a new named set of uniforms (i.e,. a Uniform Buffer Object)
	// These uniforms can be bound to programs -- first add all the uniform variables
	// and then AddUniforms to each program that uses it (already added to this one).
	// Uniforms will be bound etc when the program is compiled.
	NewUniforms(name string) Uniforms

	// AddUniforms adds an existing Uniforms collection of uniform variables to this
	// program.
	// Uniforms will be bound etc when the program is compiled.
	AddUniforms(unis Uniforms)

	// UniformByName returns a Uniform based on unique name -- this could be in a
	// collection of Uniforms (i.e., a Uniform Buffer Object in GL) or standalone
	// Returns nil if not found (error auto logged)
	UniformByName(name string) Uniform

	// UniformsByName returns Uniforms collection of given name
	// Returns nil if not found (error auto logged)
	UniformsByName(name string) Uniforms

	// AddInput adds a Vectors input variable to the program -- name must = 'in' var name.
	// This input will get bound to variable and handle updated when program is compiled.
	AddInput(name string, typ VectorType, role VectorRoles) Vectors

	// AddOutput adds a Vectors output variable to the program -- name must = 'out' var name.
	// This output will get bound to variable and handle updated when program is compiled.
	AddOutput(name string, typ VectorType, role VectorRoles) Vectors

	// Inputs returns a list (slice) of all the input ('in') vectors defined for this program.
	Inputs() []Vectors

	// Outputs returns a list (slice) of all the output ('out') vectors defined for this program.
	Outputs() []Vectors

	// InputByName returns given input vectors by name.
	// Returns nil if not found (error auto logged)
	InputByName(name string) Vectors

	// OutputByName returns given output vectors by name.
	// Returns nil if not found (error auto logged)
	OutputByName(name string) Vectors

	// InputByRole returns given input vectors by role.
	// Returns nil if not found (error auto logged)
	InputByRole(role VectorRoles) Vectors

	// OutputByRole returns given input vectors by role.
	// Returns nil if not found (error auto logged)
	OutputByRole(role VectorRoles) Vectors

	// Compile compiles all the shaders and links the program, binds the uniforms
	// and input / output vector variables, etc.
	// This must be called after setting the lengths of any array uniforms (e.g.,
	// the number of lights)
	// showSrc arg prints out the final compiled source, including automatic
	// defines etc at the top, even if there are no errors, which can be useful for debugging.
	Compile(showSrc bool) error

	// Handle returns the handle for the program -- only valid after a Compile call
	Handle() uint32

	// Activate activates this as the active program -- must have been Compiled first.
	Activate()

	// Delete deletes the GPU resources associated with this program
	// (requires Compile and Activate to re-establish a new one).
	// Should be called prior to Go object being deleted
	// (ref counting can be done externally).
	Delete()
}
