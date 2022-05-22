// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"

	"github.com/goki/ki/ints"
	vk "github.com/goki/vulkan"
)

// Var specifies a variable used in a pipeline, accessed in shader programs.
// A Var represents a type of input or output into the GPU program,
// including things like Vertex arrays, transformation matricies (Uniforms),
// Images (Textures), and arbitrary Structs for Compute shaders.
// Each Var belongs to a Set, and its binding location is allocated within that.
// Each set is updated at the same time scale, and all vars in the set have the same
// number of allocated Val instances representing a specific value of the variable.
// There must be a unique Val instance for each value of the variable used in
// a single render -- a previously-used Val's contents cannot be updated within
// the render pass, but new information can be written to an as-yet unused Val
// prior to using in a render (although this comes at a performance cost).
type Var struct {
	Name        string                 `desc:"variable name"`
	Type        Types                  `desc:"type of data in variable.  Note that there are strict contraints on the alignment of fields within structs -- if you can keep all fields at 4 byte increments, that works, but otherwise larger fields trigger a 16 byte alignment constraint.  Texture Images do not have such alignment constraints, and can be allocated in a big host buffer or in separate buffers depending on how frequently they are updated with different sizes."`
	ArrayN      int                    `desc:"number of elements if this is a fixed array -- use 1 if singular element, and 0 if a variable-sized array, where each Val can have its own specific size."`
	Role        VarRoles               `desc:"role of variable: Vertex is configured in the pipeline VkConfig structure, and everything else is configured in a DescriptorSet.  For TextureRole items, the last such Var in a set will automatically be flagged as variable sized, so the shader can specify: #extension GL_EXT_nonuniform_qualifier : require and the list of textures can be specified as a [] array."`
	Shaders     vk.ShaderStageFlagBits `desc:"bit flags for set of shaders that this variable is used in"`
	Set         int                    `desc:"DescriptorSet associated with the timing of binding for this variable -- all vars updated at the same time should be in the same set"`
	BindLoc     int                    `desc:"binding or location number for variable -- Vertexs are assigned as one group sequentially in order listed in Vars, and rest are assigned uniform binding numbers via descriptor pools"`
	SizeOf      int                    `desc:"size in bytes of one element (not array size).  Note that arrays require 16 byte alignment for each element, so if using arrays, it is best to work within that constraint.  For Push role, SizeOf must be set exactly -- no vals are created."`
	TextureOwns bool                   `desc:"texture manages its own memory allocation -- set this for texture objects that change size dynamically -- otherwise image host staging memory is allocated in a common buffer"`
	DynOffIdx   int                    `desc:"index into the dynamic offset list, where dynamic offsets of vals need to be set -- for Uniform and Storage roles -- set during Set:DescLayout"`
	Vals        Vals                   `desc:"the array of values allocated for this variable.  The size of this array is determined by the Set membership of this Var, and the current index is updated at the set level.  For Texture Roles, there is a separate descriptor for each value (image) -- otherwise dynamic offset binding is used."`
	BindValIdx  []int                  `desc:"for dynamically bound vars (Vertex, Uniform, Storage), this is the index of the currently bound value in Vals list -- index in this array is the descIdx out of Vars NDescs (see for docs) to allow for parallel update pathways -- only valid until set again -- only actually used for Vertex binding, as unforms etc have the WriteDescriptor mechanism."`
	Offset      int                    `desc:"offset -- only for push constants"`
}

// Init initializes the main values
func (vr *Var) Init(name string, typ Types, arrayN int, role VarRoles, set int, shaders ...ShaderTypes) {
	vr.Name = name
	vr.Type = typ
	vr.ArrayN = arrayN
	vr.Role = role
	vr.SizeOf = typ.Bytes()
	vr.Set = set
	vr.Shaders = 0
	for _, sh := range shaders {
		vr.Shaders |= ShaderStageFlags[sh]
	}
}

func (vr *Var) String() string {
	typ := vr.Type.String()
	if vr.ArrayN > 1 {
		typ = fmt.Sprintf("%s[%d]", typ, vr.ArrayN)
	}
	s := fmt.Sprintf("%d:\t%s\t%s\t(size: %d)\tVals: %d", vr.BindLoc, vr.Name, typ, vr.SizeOf, len(vr.Vals.Vals))
	return s
}

// BuffType returns the memory buffer type for this variable, based on Var.Role
func (vr *Var) BuffType() BuffTypes {
	return vr.Role.BuffType()
}

// BindVal returns the currently bound value at given descriptor collection index
// as set by BindDyn* methods.  Returns nil, error if not valid.
func (vr *Var) BindVal(descIdx int) (*Val, error) {
	idx := vr.BindValIdx[descIdx]
	return vr.Vals.ValByIdxTry(idx)
}

// ValsMemSize returns the memory allocation size
// for all values for this Var, in bytes
func (vr *Var) ValsMemSize(alignBytes int) int {
	return vr.Vals.MemSize(vr, alignBytes)
}

// MemSize returns the memory allocation size for this value, in bytes
func (vr *Var) MemSize() int {
	n := vr.ArrayN
	if n == 0 {
		n = 1
	}
	switch {
	case vr.Role >= TextureRole:
		return 0
	case n == 1 || vr.Role < Uniform:
		return vr.SizeOf * n
	default:
		sz := MemSizeAlign(vr.SizeOf, 16) // todo: test this!
		return sz * n
	}
}

// AllocHost allocates values at given offset in given Memory buffer.
// Computes the MemPtr for each item, and returns TotSize
// across all vals.  The effective offset increment (based on size) is
// aligned at the given align byte level, which should be
// MinUniformBufferOffsetAlignment from gpu.
func (vr *Var) AllocHost(buff *MemBuff, offset int) int {
	return vr.Vals.AllocHost(vr, buff, offset)
}

// Free resets the MemPtr for values, resets any self-owned resources (Textures)
func (vr *Var) Free() {
	vr.Vals.Free()
	// todo: free anything in var
}

// ModRegs returns the regions of Vals that have been modified
func (vr *Var) ModRegs() []MemReg {
	return vr.Vals.ModRegs()
}

// SetTextureDev sets Device for textures
// only called on Role = TextureRole
func (vr *Var) SetTextureDev(dev vk.Device) {
	for _, vl := range vr.Vals.Vals {
		if vl.Texture == nil {
			continue
		}
		vl.Texture.Dev = dev
	}
}

// AllocTextures allocates images on device memory
// only called on Role = TextureRole
func (vr *Var) AllocTextures(mm *Memory) {
	vr.Vals.AllocTextures(mm)
}

// TextureValidIdx returns the index of the given texture value at our
// index in list of vals, starting at given index, skipping over any
// inactive textures which do not show up when accessed in the shader.
// You must use this value when passing a texture index to the shader!
// returns -1 if idx is not valid
func (vr *Var) TextureValidIdx(stIdx, idx int) int {
	vidx := 0
	mx := ints.MinInt(stIdx+MaxTexturesPerSet, len(vr.Vals.Vals))
	for i := stIdx; i < mx; i++ {
		vl := vr.Vals.Vals[i]
		if i == idx {
			return vidx
		}
		if vl.Texture == nil || !vl.Texture.IsActive() {
			if i == idx {
				return -1
			}
			continue
		}
		vidx++
	}
	return -1
}

//////////////////////////////////////////////////////////////////
// VarList

// VarList is a list of variables
type VarList struct {
	Vars   []*Var          `desc:"variables in order"`
	VarMap map[string]*Var `desc:"map of vars by name -- names must be unique"`
}

// VarByNameTry returns Var by name, returning error if not found
func (vs *VarList) VarByNameTry(name string) (*Var, error) {
	vr, ok := vs.VarMap[name]
	if !ok {
		err := fmt.Errorf("Variable named %s not found", name)
		if TheGPU.Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vr, nil
}

// ValByNameTry returns value by first looking up variable name, then value name,
// returning error if not found
func (vs *VarList) ValByNameTry(varName, valName string) (*Var, *Val, error) {
	vr, err := vs.VarByNameTry(varName)
	if err != nil {
		return nil, nil, err
	}
	vl, err := vr.Vals.ValByNameTry(valName)
	return vr, vl, err
}

// ValByIdxTry returns value by first looking up variable name, then value index,
// returning error if not found
func (vs *VarList) ValByIdxTry(varName string, valIdx int) (*Var, *Val, error) {
	vr, err := vs.VarByNameTry(varName)
	if err != nil {
		return nil, nil, err
	}
	vl, err := vr.Vals.ValByIdxTry(valIdx)
	return vr, vl, err
}
