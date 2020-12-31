// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/mat32"
)

// Uniform represents a single uniform variable, which can be contained within a
// Uniform Buffer Object or used as a separate independent uniform.
// This can be an array of values as well, in which case a NAME_LEN macro is
// always defined to reflect the length of the array.
// These uniforms are used directly to generate the shader code.
// See Program.AddUniform to create a new standalone one, and
// Program.NewUniforms to create a new set of them (i.e., Uniform Buffer Object)
type Uniform struct {
	init    bool
	name    string
	handle  int32
	typ     gpu.UniType
	array   bool
	ln      int
	offset  int
	size    int       // only set if part of a Uniforms UBO
	stdSize int       // ditto
	ubo     *Uniforms // set if a member of a ubo
}

// Name returns name of the Uniform
func (un *Uniform) Name() string {
	return un.name
}

// Type returns type of the Uniform
func (un *Uniform) Type() gpu.UniType {
	return un.typ
}

// Array returns true if this is an array Uniform.
// If so, then it automatically generates a #define NAME_LEN <Len> definition prior
// to the Uniform definition, and if Len == 0 then it is *not* defined at all.
// All code referencing this Uniform should use #if NAME_LEN>0 wrapper.
func (un *Uniform) Array() bool {
	return un.array
}

// Len returns number of array elements, if an Array (can be 0)
func (un *Uniform) Len() int {
	return un.ln
}

// SetLen sets the number of array elements -- if this is changed, then the associated
// Shader program needs to be re-generated and recompiled.
// Unless this is in a Uniforms, must be recompiled before calling SetValue
func (un *Uniform) SetLen(ln int) {
	un.ln = ln
}

// Offset returns byte-wise offset into the UBO where this Uniform starts (only for UBO's)
func (un *Uniform) Offset() int {
	return un.offset
}

// Size() returns actual byte-wise size of this uniform raw data (c.f., StdSize)
func (un *Uniform) Size() int {
	if un.array {
		un.size = un.ln * un.typ.Bytes()
	} else {
		un.size = un.typ.Bytes()
	}
	return un.size
}

// StdSize() returns byte-wise size of this uniform, *including padding* for representation
// on the GPU -- e.g., as determined by the std140 standard opengl layout
func (un *Uniform) StdSize() int {
	if un.array {
		un.stdSize = un.ln * un.typ.StdBytes()
	} else {
		un.stdSize = un.typ.StdBytes()
	}
	return un.stdSize
}

// Handle() returns the unique id for this Uniform.
// if in a UBO, then this is the index of the item within the list of UBO's
func (un *Uniform) Handle() int32 {
	return un.handle
}

// SetValue sets the value of the Uniform to given value, which must be of the corresponding
// elemental or mat32.Vector or mat32.Matrix type.  Proper context must be bound, etc.
func (un *Uniform) SetValue(val interface{}) error {
	err := un.SetValueImpl(val)
	// if err != nil {
	// 	log.Println(err)
	// }
	return err
}

func (un *Uniform) SetValueImpl(val interface{}) error {
	if un.ubo != nil {
		un.ubo.Activate()
	}
	switch un.typ.Type {
	case gpu.Float32:
		if un.array {
			switch {
			case un.typ.Mat == 3:
				fv, ok := val.([]mat32.Mat3)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Mat3", un.name)
				}
				if un.ubo != nil {
					// todo: this is incorrect!  mat3 needs to be converted to mat4
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(fv))
				} else {
					gl.UniformMatrix3fv(un.handle, int32(un.ln), false, &fv[0][0])
				}
			case un.typ.Mat == 4:
				fv, ok := val.([]mat32.Mat4)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Mat4", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(fv))
				} else {
					gl.UniformMatrix4fv(un.handle, int32(un.ln), false, &fv[0][0])
				}
			case un.typ.Vec == 2:
				fv, ok := val.([]mat32.Vec2)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Vec2", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(fv))
				} else {
					gl.Uniform2fv(un.handle, int32(un.ln), &fv[0].X)
				}
			case un.typ.Vec == 3:
				fv, ok := val.([]mat32.Vec3)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Vec3", un.name)
				}
				if un.ubo != nil {
					// need separate writes b/c alignment is vec4
					for i := 0; i < un.ln; i++ {
						gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset+i*4*4, 4*3, gl.Ptr(&fv[i].X))
					}
				} else {
					gl.Uniform3fv(un.handle, int32(un.ln), &fv[0].X)
				}
			case un.typ.Vec == 4:
				fv, ok := val.([]mat32.Vec4)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Vec4", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv[0].X))
				} else {
					gl.Uniform4fv(un.handle, int32(un.ln), &fv[0].X)
				}
			case un.typ.Vec == 0:
				fv, ok := val.([]float32)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []float32", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(fv))
				} else {
					gl.Uniform1fv(un.handle, int32(un.ln), &fv[0])
				}
			}
		} else {
			switch {
			case un.typ.Mat == 3:
				fv, ok := val.(mat32.Mat3)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Mat3", un.name)
				}
				if un.ubo != nil {
					m4 := mat32.Mat4{} // stored internally as effectively a mat4 without the last column
					m4.SetFromMat3(&fv)
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.stdSize, gl.Ptr(&m4[0]))
				} else {
					gl.UniformMatrix3fv(un.handle, 1, false, &fv[0])
				}
			case un.typ.Mat == 4:
				fv, ok := val.(mat32.Mat4)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Mat4", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv[0]))
				} else {
					gl.UniformMatrix4fv(un.handle, 1, false, &fv[0])
				}
			case un.typ.Vec == 2:
				fv, ok := val.(mat32.Vec2)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Vec2", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform2f(un.handle, fv.X, fv.Y)
				}
			case un.typ.Vec == 3:
				fv, ok := val.(mat32.Vec3)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Vec3", un.name)
				}
				if un.ubo != nil { // note: stored as vec4 but only transfer 3
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform3f(un.handle, fv.X, fv.Y, fv.Z)
				}
			case un.typ.Vec == 4:
				fv, ok := val.(mat32.Vec4)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Vec4", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform4f(un.handle, fv.X, fv.Y, fv.Z, fv.W)
				}
			case un.typ.Vec == 0:
				fv, ok := val.(float32)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be float32", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform1f(un.handle, fv)
				}
			}
		}
	case gpu.Int:
		if un.array {
			switch {
			case un.typ.Vec == 2:
				fv, ok := val.([]mat32.Vec2i)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Vec2i", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(fv))
				} else {
					gl.Uniform2iv(un.handle, int32(un.ln), &fv[0].X)
				}
			case un.typ.Vec == 3:
				fv, ok := val.([]mat32.Vec3i)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Vec3i", un.name)
				}
				if un.ubo != nil {
					// need separate writes b/c alignment is vec4
					for i := 0; i < un.ln; i++ {
						gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset+i*4*4, 4*3, gl.Ptr(&fv[i].X))
					}
				} else {
					gl.Uniform3iv(un.handle, int32(un.ln), &fv[0].X)
				}
			// case un.typ.Vec == 4:
			// 	fv, ok := val.([]mat32.Vec4)
			// 	if !ok || len(fv) != un.ln {
			// 		return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Vec4", un.name)
			// 	}
			// 	if un.ubo != nil {
			// 		gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv[0].X))
			// 	} else {
			// 		gl.Uniform4fv(un.handle, int32(un.ln), &fv[0].X)
			// 	}
			case un.typ.Vec == 0:
				fv, ok := val.([]int32)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []int32", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(fv))
				} else {
					gl.Uniform1iv(un.handle, int32(un.ln), &fv[0])
				}
			}
		} else {
			switch {
			case un.typ.Vec == 2:
				fv, ok := val.(mat32.Vec2i)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Vec2i", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform2i(un.handle, fv.X, fv.Y)
				}
			case un.typ.Vec == 3:
				fv, ok := val.(mat32.Vec3i)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Vec3i", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform3i(un.handle, fv.X, fv.Y, fv.Z)
				}
			// case un.typ.Vec == 4:
			// 	fv, ok := val.(mat32.Vec4)
			// 	if !ok {
			// 		return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Vec4", un.name)
			// 	}
			// 	if un.ubo != nil {
			// 		gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
			// 	} else {
			// 		gl.Uniform4f(un.handle, fv.X, fv.Y, fv.Z, fv.W)
			// 	}
			case un.typ.Vec == 0:
				fv, ok := val.(int32)
				if !ok {
					fvi, ok := val.(int)
					if !ok {
						return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be int or int32", un.name)
					}
					fv = int32(fvi)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform1i(un.handle, fv)
				}
			}
		}
	case gpu.Bool:
		if un.array {
			switch {
			case un.typ.Vec == 2:
				fv, ok := val.([]mat32.Vec2i)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Vec2i", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(fv))
				} else {
					gl.Uniform2iv(un.handle, int32(un.ln), &fv[0].X)
				}
			case un.typ.Vec == 3:
				fv, ok := val.([]mat32.Vec3i)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Vec3i", un.name)
				}
				if un.ubo != nil {
					// need separate writes b/c alignment is vec4
					for i := 0; i < un.ln; i++ {
						gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset+i*4*4, 4*3, gl.Ptr(&fv[i].X))
					}
				} else {
					gl.Uniform3iv(un.handle, int32(un.ln), &fv[0].X)
				}
			// case un.typ.Vec == 4:
			// 	fv, ok := val.([]mat32.Vec4)
			// 	if !ok || len(fv) != un.ln {
			// 		return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []mat32.Vec4", un.name)
			// 	}
			// 	if un.ubo != nil {
			// 		gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv[0].X))
			// 	} else {
			// 		gl.Uniform4fv(un.handle, int32(un.ln), &fv[0].X)
			// 	}
			case un.typ.Vec == 0:
				fv, ok := val.([]int32)
				if !ok || len(fv) != un.ln {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be []int32", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(fv))
				} else {
					gl.Uniform1iv(un.handle, int32(un.ln), &fv[0])
				}
			}
		} else {
			switch {
			case un.typ.Vec == 2:
				fv, ok := val.(mat32.Vec2i)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Vec2i", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform2i(un.handle, fv.X, fv.Y)
				}
			case un.typ.Vec == 3:
				fv, ok := val.(mat32.Vec3i)
				if !ok {
					return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Vec3i", un.name)
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform3i(un.handle, fv.X, fv.Y, fv.Z)
				}
			// case un.typ.Vec == 4:
			// 	fv, ok := val.(mat32.Vec4)
			// 	if !ok {
			// 		return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be mat32.Vec4", un.name)
			// 	}
			// 	if un.ubo != nil {
			// 		gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
			// 	} else {
			// 		gl.Uniform4f(un.handle, fv.X, fv.Y, fv.Z, fv.W)
			// 	}
			case un.typ.Vec == 0:
				fv, ok := val.(int32)
				if !ok {
					fvi, ok := val.(bool)
					if !ok {
						return fmt.Errorf("glgpu Uniform SetValue: Uniform: %s val must be bool or int32", un.name)
					}
					if fvi {
						fv = 1
					} else {
						fv = 0
					}
				}
				if un.ubo != nil {
					gl.BufferSubData(gl.UNIFORM_BUFFER, un.offset, un.size, gl.Ptr(&fv))
				} else {
					gl.Uniform1i(un.handle, fv)
				}
			}
		}
	}
	return gpu.TheGPU.ErrCheck(fmt.Sprintf("Uniform SetValue: %v type: %v", un.name, un.typ))
}

// LenDefine returns the #define NAME_LEN source code for this Uniform, empty if not an array
func (un *Uniform) LenDefine() string {
	if !un.array {
		return ""
	}
	unm := strings.ToUpper(un.name)
	return fmt.Sprintf("#define %s_LEN %d\n", unm, un.ln)
}

//////////////////////////////////////////////
// Uniforms

// https://learnopengl.com/Advanced-OpenGL/Advanced-GLSL

// Uniforms is a set of Uniform objects that are all organized together
// (i.e., a UniformBufferObject in OpenGL)
type Uniforms struct {
	init   bool
	name   string
	handle uint32
	bindPt uint32
	unis   map[string]*Uniform
	uniOrd []*Uniform
	size   int // overall size of buffer, accumulating stdSize of elements
}

// Name returns the name of this set of Uniforms
func (un *Uniforms) Name() string {
	return un.name
}

// SetName sets the name of this set of uniforms
func (un *Uniforms) SetName(name string) {
	un.name = name
}

// AddUniform adds a Uniform variable to this collection of Uniforms of given type
func (un *Uniforms) AddUniform(name string, typ gpu.UniType, ary bool, ln int) gpu.Uniform {
	if un.unis == nil {
		un.unis = make(map[string]*Uniform)
	}
	u := &Uniform{name: name, typ: typ, array: ary, ln: ln}
	un.unis[name] = u
	un.uniOrd = append(un.uniOrd, u)
	return u
}

// UniformByName returns a Uniform based on unique name -- this could be in a
// collection of Uniforms (i.e., a Uniform Buffer Object in GL) or standalone
func (un *Uniforms) UniformByName(name string) gpu.Uniform {
	u, ok := un.unis[name]
	if !ok {
		log.Printf("glgpu Uniforms: name %s not found in Uniforms: %s\n", name, un.name)
		return nil
	}
	return u
}

// LenDefines returns the #define NAME_LEN source code for all Uniforms, empty if no arrays
func (un *Uniforms) LenDefines() string {
	defs := ""
	for _, u := range un.unis {
		defs += u.LenDefine()
	}
	return defs
}

// getSize computes the size based on all the elements
func (un *Uniforms) getSize() int {
	sz := 0
	for i, u := range un.uniOrd {
		u.ubo = un
		u.Size()           // compute actual size
		usz := u.StdSize() // use std size
		if usz == 0 {
			u.offset = 0
			u.handle = 0
			continue
		}
		u.offset = sz
		u.handle = int32(i)
		sz += usz
	}
	return sz
}

// Resize resizes the buffer if needed -- call if any of the member uniforms
// might have been resized.  Calls Activate if not already activated.
func (un *Uniforms) Resize() error {
	err := un.Activate()
	if err != nil {
		return err
	}
	nwsz := un.getSize()
	if nwsz == un.size {
		return nil
	}
	un.size = nwsz
	gl.BufferData(gl.UNIFORM_BUFFER, un.size, nil, gl.STATIC_DRAW)
	return nil
}

// Activate generates the Uniform Buffer Object structure and reserves the binding point
func (un *Uniforms) Activate() error {
	if !un.init {
		un.bindPt = uint32(gpu.TheGPU.NextUniformBindingPoint())
		un.size = un.getSize()
		gl.GenBuffers(1, &un.handle)
		gl.BindBuffer(gl.UNIFORM_BUFFER, un.handle)
		gl.BufferData(gl.UNIFORM_BUFFER, un.size, nil, gl.STATIC_DRAW)
		gl.BindBufferBase(gl.UNIFORM_BUFFER, un.bindPt, un.handle)
		un.init = true
	} else {
		gl.BindBuffer(gl.UNIFORM_BUFFER, un.handle)
	}
	return nil
}

// Bind binds the Uniform Buffer Object structure to given program
// Activate must be called first
func (un *Uniforms) Bind(prog gpu.Program) error {
	pr := prog.(*Program)
	ubidx := gl.GetUniformBlockIndex(pr.handle, gl.Str(gpu.CString(un.name)))
	if ubidx == gl.INVALID_INDEX {
		err := fmt.Errorf("glgpu Uniforms named: %s not found in Program: %v", un.name, pr.name)
		log.Println(err)
		return err
	}
	if !un.init {
		un.Activate()
	}
	pr.Activate()
	gl.UniformBlockBinding(pr.handle, ubidx, un.bindPt)
	gpu.TheGPU.ErrCheck("uniforms bind to program")
	return nil
}

// Handle returns the handle for the Program -- only valid after a Compile call
func (un *Uniforms) Handle() uint32 {
	return un.handle
}

// BindingPoint returns the unique binding point for this set of Uniforms --
// needed for connecting to Programs
func (un *Uniforms) BindingPoint() uint32 {
	return un.bindPt
}
