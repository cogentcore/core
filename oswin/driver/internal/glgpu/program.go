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

// Program manages a set of shaders and associated variables and uniforms.
// Multiple programs can be assembled into a Pipeline, which can create
// new Programs.  GPU.NewProgram() can also create standalone Programs.
// All uniforms must be added before compiling program.
type program struct {
	init        bool
	handle      uint32
	name        string
	shaders     map[gpu.ShaderTypes]*shader
	unis        map[string]*uniform
	ubos        map[string]*uniforms
	ins         map[string]*vectors
	outs        map[string]*vectors
	fragDataVar string
}

// Name returns name of program
func (pr *program) Name() string {
	return pr.name
}

// AddShader adds shader of given type, unique name and source code.
// Any array uniform's will add their #define NAME_LEN's to the top
// of the source code automatically, so the source can assume those exist
// when compiled.
func (pr *program) AddShader(typ gpu.ShaderTypes, name string, src string) (gpu.Shader, error) {
	if pr.shaders == nil {
		pr.shaders = make(map[gpu.ShaderTypes]*shader)
	}
	if _, has := pr.shaders[typ]; has {
		err := fmt.Errorf("glos gpu.AddShader: shader of that type: %s already added!", typ)
		log.Println(err)
		return nil, err
	}
	sh := &shader{name: name, typ: typ, orgSrc: src}
	pr.shaders[typ] = sh
	return sh, nil
}

// ShaderByName returns shader by its unique name
func (pr *program) ShaderByName(name string) gpu.Shader {
	for _, sh := range pr.shaders {
		if sh.name == name {
			return sh
		}
	}
	log.Println("glos gpu.AddShader: shader of name: %s not added\n", name)
	return nil
}

// ShaderByType returns shader by its type
func (pr *program) ShaderByType(typ gpu.ShaderTypes) gpu.Shader {
	sh, ok := pr.shaders[typ]
	if !ok {
		log.Println("glos gpu.AddShader: shader of that type: %s not added\n", typ)
		return nil
	}
	return sh
}

// SetFragDataVar sets the variable name to use for the fragment shader's output
func (pr *program) SetFragDataVar(name string) {
	pr.fragDataVar = name
}

// AddUniform adds an individual standalone uniform variable to the program of given type.
// Must add all uniform variables before compiling, as they add to source.
func (pr *program) AddUniform(name string, typ gpu.UniType, ary bool, ln int) gpu.Uniform {
	if pr.unis == nil {
		pr.unis = make(map[string]*uniform)
	}
	u := &uniform{name: name, typ: typ, array: ary, ln: ln}
	pr.unis[name] = u
	return u
}

// NewUniforms makes a new named set of uniforms (i.e,. a Uniform Buffer Object)
// These uniforms can be bound to programs -- first add all the uniform variables
// and then AddUniforms to each program that uses it (already added to this one).
// Uniforms will be bound etc when the program is compiled.
func (pr *program) NewUniforms(name string) gpu.Uniforms {
	if pr.ubos == nil {
		pr.ubos = make(map[string]*uniforms)
	}
	us := &uniforms{name: name}
	pr.ubos[name] = us
	return us
}

// AddUniforms adds an existing Uniforms collection of uniform variables to this
// program.
// Uniforms will be bound etc when the program is compiled.
func (pr *program) AddUniforms(unis gpu.Uniforms) {
	if pr.ubos == nil {
		pr.ubos = make(map[string]*uniforms)
	}
	us := unis.(*uniforms)
	pr.ubos[us.name] = us
}

// UniformByName returns a Uniform based on unique name -- this could be in a
// collection of Uniforms (i.e., a Uniform Buffer Object in GL) or standalone
// Returns nil if not found (error auto logged)
func (pr *program) UniformByName(name string) gpu.Uniform {
	u, ok := pr.unis[name]
	if !ok {
		log.Printf("glgpu Program UniformByName: name %s not found in %s\n", name, pr.name)
		return nil
	}
	return u
}

// UniformsByName returns Uniforms collection of given name
// Returns nil if not found (error auto logged)
func (pr *program) UniformsByName(name string) gpu.Uniforms {
	us, ok := pr.ubos[name]
	if !ok {
		log.Printf("glgpu Program UniformsByName: name %s not found in %s\n", name, pr.name)
		return nil
	}
	return us
}

// AddInput adds a Vectors input variable to the program -- name must = 'in' var name.
// This input will get bound to variable and handle updated when program is compiled.
func (pr *program) AddInput(name string, typ gpu.VectorType, role gpu.VectorRoles) gpu.Vectors {
	if pr.ins == nil {
		pr.ins = make(map[string]*vectors)
	}
	v := &vectors{name: name, typ: typ, role: role}
	pr.ins[name] = v
	return v
}

// AddOutput adds a Vectors output variable to the program -- name must = 'out' var name.
// This output will get bound to variable and handle updated when program is compiled.
func (pr *program) AddOutput(name string, typ gpu.VectorType, role gpu.VectorRoles) gpu.Vectors {
	if pr.outs == nil {
		pr.outs = make(map[string]*vectors)
	}
	v := &vectors{name: name, typ: typ, role: role}
	pr.outs[name] = v
	return v
}

// Inputs returns a list (slice) of all the input ('in') vectors defined for this program.
func (pr *program) Inputs() []gpu.Vectors {
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

// Outputs returns a list (slice) of all the output ('out') vectors defined for this program.
func (pr *program) Outputs() []gpu.Vectors {
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

// InputByName returns given input vectors by name.
// Returns nil if not found (error auto logged)
func (pr *program) InputByName(name string) gpu.Vectors {
	v, ok := pr.ins[name]
	if !ok {
		log.Printf("glgpu Program InputByName: name %s not found in %s\n", name, pr.name)
		return nil
	}
	return v
}

// OutputByName returns given output vectors by name.
// Returns nil if not found (error auto logged)
func (pr *program) OutputByName(name string) gpu.Vectors {
	v, ok := pr.outs[name]
	if !ok {
		log.Printf("glgpu Program OutputByName: name %s not found in %s\n", name, pr.name)
		return nil
	}
	return v
}

// InputByRole returns given input vectors by role.
// Returns nil if not found (error auto logged)
func (pr *program) InputByRole(role gpu.VectorRoles) gpu.Vectors {
	for _, v := range pr.ins {
		if v.role == role {
			return v
		}
	}
	log.Printf("glgpu Program InputByRole: role %s not found in %s\n", role, pr.name)
	return nil
}

// OutputByRole returns given input vectors by role.
// Returns nil if not found (error auto logged)
func (pr *program) OutputByRole(role gpu.VectorRoles) gpu.Vectors {
	for _, v := range pr.outs {
		if v.role == role {
			return v
		}
	}
	log.Printf("glgpu Program OutputByRole: role %s not found in %s\n", role, pr.name)
	return nil
}

// Compile compiles all the shaders and links the program, binds the uniforms
// and input / output vector variables, etc.
// This must be called after setting the lengths of any array uniforms (e.g.,
// the number of lights)
func (pr *program) Compile() error {
	defs := ""
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
	// if glosDebug {
	gl.ValidateProgram(handle)
	// }

	for _, sh := range pr.shaders {
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

		err := fmt.Errorf("glgpu Program %s Compile: failed to link program: %v", lg)
		log.Println(err)
		return err
	}

	// bind uniforms
	for _, u := range pr.unis {
		u.handle = gl.GetUniformLocation(handle, gl.Str(gpu.CString(u.name)))
		if u.handle < 0 {
			err := fmt.Errorf("glgpu Program %s Compile: uniform named: %s not found", u.name)
			log.Println(err)
			return err
		}
	}
	// bind inputs
	for _, v := range pr.ins {
		v.handle = uint32(gl.GetAttribLocation(handle, gl.Str(gpu.CString(v.name))))
		if v.handle < 0 {
			err := fmt.Errorf("glgpu Program %s Compile: input Vectors named: %s not found", v.name)
			log.Println(err)
			return err
		}
	}
	// bind outputs
	for _, v := range pr.outs {
		v.handle = uint32(gl.GetAttribLocation(handle, gl.Str(gpu.CString(v.name))))
		if v.handle < 0 {
			err := fmt.Errorf("glgpu Program %s Compile: output Vectors named: %s not found", v.name)
			log.Println(err)
			return err
		}
	}
	if pr.fragDataVar != "" {
		gl.BindFragDataLocation(handle, 0, gl.Str(gpu.CString(pr.fragDataVar)))
	}

	pr.handle = handle
	pr.init = true
	return nil
}

// Handle returns the handle for the program -- only valid after a Compile call
func (pr *program) Handle() uint32 {
	return pr.handle
}

// Activate activates this as the active program -- must have been Compiled first.
func (pr *program) Activate() {
	if !pr.init {
		return
	}
	gl.UseProgram(pr.handle)
}

// Delete deletes the GPU resources associated with this program
// (requires Compile and Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
func (pr *program) Delete() {
	if !pr.init {
		return
	}
	gl.DeleteProgram(pr.handle)
	pr.handle = 0
	pr.init = false
}
