// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/goki/gi/oswin/gpu"
)

// Program manages a set of shaders and associated variables and uniforms.
// Multiple programs can be assembled into a Pipeline, which can create
// new Programs.  GPU.NewProgram() can also create standalone Programs.
// All uniforms must be added before compiling program.
type Program struct {
	init        bool
	handle      uint32
	name        string
	shaders     map[gpu.ShaderTypes]*Shader
	unis        map[string]*Uniform
	ubos        map[string]*Uniforms
	ins         map[string]*Vectors
	outs        map[string]*Vectors
	fragDataVar string
}

// Name returns name of Program
func (pr *Program) Name() string {
	return pr.name
}

// SetName sets name of program
func (pr *Program) SetName(name string) {
	pr.name = name
}

// AddShader adds shader of given type, unique name and source code.
// Any array Uniform's will add their #define NAME_LEN's to the top
// of the source code automatically, so the source can assume those exist
// when compiled.
func (pr *Program) AddShader(typ gpu.ShaderTypes, name string, src string) (gpu.Shader, error) {
	if pr.shaders == nil {
		pr.shaders = make(map[gpu.ShaderTypes]*Shader)
	}
	if _, has := pr.shaders[typ]; has {
		err := fmt.Errorf("glgpu gpu.AddShader: shader of that type: %s already added!", typ)
		log.Println(err)
		return nil, err
	}
	sh := &Shader{name: name, typ: typ, orgSrc: src}
	pr.shaders[typ] = sh
	return sh, nil
}

// ShaderByName returns shader by its unique name
func (pr *Program) ShaderByName(name string) gpu.Shader {
	for _, sh := range pr.shaders {
		if sh.name == name {
			return sh
		}
	}
	log.Printf("glos gpu.AddShader: shader of name: %s not found!\n", name)
	return nil
}

// ShaderByType returns shader by its type
func (pr *Program) ShaderByType(typ gpu.ShaderTypes) gpu.Shader {
	sh, ok := pr.shaders[typ]
	if !ok {
		log.Printf("glos gpu.AddShader: shader of that type: %s not found!\n", typ)
		return nil
	}
	return sh
}

// SetFragDataVar sets the variable name to use for the fragment shader's output
func (pr *Program) SetFragDataVar(name string) {
	pr.fragDataVar = name
}

// AddUniform adds an individual standalone Uniform variable to the Program of given type.
// Must add all Uniform variables before compiling, as they add to source.
func (pr *Program) AddUniform(name string, typ gpu.UniType, ary bool, ln int) gpu.Uniform {
	if pr.unis == nil {
		pr.unis = make(map[string]*Uniform)
	}
	u := &Uniform{name: name, typ: typ, array: ary, ln: ln}
	pr.unis[name] = u
	return u
}

// NewUniforms makes a new named set of Uniforms (i.e,. a Uniform Buffer Object)
// These Uniforms can be bound to Programs -- first add all the Uniform variables
// and then AddUniforms to each Program that uses it (already added to this one).
// Uniforms will be bound etc when the Program is compiled.
func (pr *Program) NewUniforms(name string) gpu.Uniforms {
	if pr.ubos == nil {
		pr.ubos = make(map[string]*Uniforms)
	}
	us := &Uniforms{name: name}
	pr.ubos[name] = us
	return us
}

// AddUniforms adds an existing Uniforms collection of Uniform variables to this
// Program.
// Uniforms will be bound etc when the Program is compiled.
func (pr *Program) AddUniforms(unis gpu.Uniforms) {
	if pr.ubos == nil {
		pr.ubos = make(map[string]*Uniforms)
	}
	us := unis.(*Uniforms)
	pr.ubos[us.name] = us
}

// UniformByName returns a Uniform based on unique name -- this could be in a
// collection of Uniforms (i.e., a Uniform Buffer Object in GL) or standalone
// Returns nil if not found (error auto logged)
func (pr *Program) UniformByName(name string) gpu.Uniform {
	u, ok := pr.unis[name]
	if !ok {
		log.Printf("glgpu Program: %v UniformByName: name %s not found\n", pr.name, name)
		return nil
	}
	return u
}

// UniformsByName returns Uniforms collection of given name
// Returns nil if not found (error auto logged)
func (pr *Program) UniformsByName(name string) gpu.Uniforms {
	us, ok := pr.ubos[name]
	if !ok {
		log.Printf("glgpu Program: %v UniformsByName: name %s not found\n", pr.name, name)
		return nil
	}
	return us
}

// AddInput adds a Vectors input variable to the Program -- name must = 'in' var name.
// This input will get bound to variable and handle updated when Program is compiled.
func (pr *Program) AddInput(name string, typ gpu.VectorType, role gpu.VectorRoles) gpu.Vectors {
	if pr.ins == nil {
		pr.ins = make(map[string]*Vectors)
	}
	v := &Vectors{name: name, typ: typ, role: role}
	pr.ins[name] = v
	return v
}

// AddOutput adds a Vectors output variable to the Program -- name must = 'out' var name.
// This output will get bound to variable and handle updated when Program is compiled.
func (pr *Program) AddOutput(name string, typ gpu.VectorType, role gpu.VectorRoles) gpu.Vectors {
	if pr.outs == nil {
		pr.outs = make(map[string]*Vectors)
	}
	v := &Vectors{name: name, typ: typ, role: role}
	pr.outs[name] = v
	return v
}

// Inputs returns a list (slice) of all the input ('in') Vectors defined for this Program.
func (pr *Program) Inputs() []gpu.Vectors {
	sz := len(pr.ins)
	if sz == 0 {
		return nil
	}
	vs := make([]gpu.Vectors, sz)
	ctr := 0
	for _, v := range pr.ins {
		vs[ctr] = v
		ctr++
	}
	return vs
}

// Outputs returns a list (slice) of all the output ('out') Vectors defined for this Program.
func (pr *Program) Outputs() []gpu.Vectors {
	sz := len(pr.outs)
	if sz == 0 {
		return nil
	}
	vs := make([]gpu.Vectors, sz)
	ctr := 0
	for _, v := range pr.outs {
		vs[ctr] = v
		ctr++
	}
	return vs
}

// InputByName returns given input Vectors by name.
// Returns nil if not found (error auto logged)
func (pr *Program) InputByName(name string) gpu.Vectors {
	v, ok := pr.ins[name]
	if !ok {
		log.Printf("glgpu Program: %v InputByName: name %s not found\n", pr.name, name)
		return nil
	}
	return v
}

// OutputByName returns given output Vectors by name.
// Returns nil if not found (error auto logged)
func (pr *Program) OutputByName(name string) gpu.Vectors {
	v, ok := pr.outs[name]
	if !ok {
		log.Printf("glgpu Program: %v OutputByName: name %s not found\n", pr.name, name)
		return nil
	}
	return v
}

// InputByRole returns given input Vectors by role.
// Returns nil if not found (error auto logged)
func (pr *Program) InputByRole(role gpu.VectorRoles) gpu.Vectors {
	for _, v := range pr.ins {
		if v.role == role {
			return v
		}
	}
	log.Printf("glgpu Program: %v InputByRole: role %s not found\n", pr.name, role)
	return nil
}

// OutputByRole returns given input Vectors by role.
// Returns nil if not found (error auto logged)
func (pr *Program) OutputByRole(role gpu.VectorRoles) gpu.Vectors {
	for _, v := range pr.outs {
		if v.role == role {
			return v
		}
	}
	log.Printf("glgpu Program: %v OutputByRole: role %s not found\n", pr.name, role)
	return nil
}

// Compile compiles all the shaders and links the Program, binds the Uniforms
// and input / output vector variables, etc.
// This must be called after setting the lengths of any array Uniforms (e.g.,
// the number of lights).
// showSrc arg prints out the final compiled source, including automatic
// defines etc at the top, even if there are no errors, which can be useful for debugging.
func (pr *Program) Compile(showSrc bool) error {
	defs := "#version 330\n" // we have to append this as it must appear at the top of the program
	for _, u := range pr.unis {
		defs += u.LenDefine()
	}
	for _, u := range pr.ubos {
		defs += u.LenDefines()
	}

	handle := gl.CreateProgram()
	for _, sh := range pr.shaders {
		src := defs + sh.orgSrc
		err := sh.Compile(src)
		if err != nil {
			return err
		}
		gl.AttachShader(handle, sh.handle)
	}
	gl.LinkProgram(handle)
	gl.ValidateProgram(handle)

	// handle is needed for other steps, set now!
	pr.handle = handle
	pr.init = true

	for _, sh := range pr.shaders {
		if showSrc {
			fmt.Printf("\n#################################\nglgpu Shader: %v Source:\n%s\n", sh.name, sh.Source())
		}
		gl.DetachShader(handle, sh.handle)
		sh.Delete()
	}

	var status int32
	gl.GetProgramiv(handle, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var lgLength int32
		gl.GetProgramiv(handle, gl.INFO_LOG_LENGTH, &lgLength)

		lg := strings.Repeat("\x00", int(lgLength+1))
		gl.GetProgramInfoLog(handle, lgLength, nil, gl.Str(lg))

		err := fmt.Errorf("glgpu Program: %s Compile: failed to link Program: %v", pr.name, lg)
		log.Println(err)
		return err
	}

	// bind Uniforms
	for _, u := range pr.unis {
		u.handle = gl.GetUniformLocation(handle, gl.Str(gpu.CString(u.name)))
		if u.handle < 0 {
			err := fmt.Errorf("glgpu Program: %s Compile: Uniform named: %s not found", pr.name, u.name)
			log.Println(err)
			return err
		}
		u.init = true
	}
	// bind ubos
	for _, u := range pr.ubos {
		err := u.Activate()
		if err != nil {
			log.Println(err)
			return err
		}
		err = u.Bind(pr)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	// bind inputs
	for _, v := range pr.ins {
		v.handle = uint32(gl.GetAttribLocation(handle, gl.Str(gpu.CString(v.name))))
		if v.handle < 0 {
			err := fmt.Errorf("glgpu Program: %s Compile: input Vectors named: %s not found", pr.name, v.name)
			log.Println(err)
			return err
		}
		v.init = true
	}
	// bind outputs
	for _, v := range pr.outs {
		v.handle = uint32(gl.GetAttribLocation(handle, gl.Str(gpu.CString(v.name))))
		if v.handle < 0 {
			err := fmt.Errorf("glgpu Program: %s Compile: output Vectors named: %s not found", pr.name, v.name)
			log.Println(err)
			return err
		}
		v.init = true
	}
	if pr.fragDataVar != "" {
		gl.BindFragDataLocation(handle, 0, gl.Str(gpu.CString(pr.fragDataVar)))
	}

	return nil
}

// Handle returns the handle for the Program -- only valid after a Compile call
func (pr *Program) Handle() uint32 {
	return pr.handle
}

// Activate activates this as the active Program -- must have been Compiled first.
func (pr *Program) Activate() {
	if !pr.init {
		return
	}
	gl.UseProgram(pr.handle)
}

// Delete deletes the GPU resources associated with this Program
// (requires Compile and Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
func (pr *Program) Delete() {
	if !pr.init {
		return
	}
	gl.DeleteProgram(pr.handle)
	pr.handle = 0
	pr.init = false
}
