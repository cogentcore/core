// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"

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

	// variable name
	Name string

	// type of data in variable.  Note that there are strict contraints on the alignment of fields within structs -- if you can keep all fields at 4 byte increments, that works, but otherwise larger fields trigger a 16 byte alignment constraint.  Texture Images do not have such alignment constraints, and can be allocated in a big host buffer or in separate buffers depending on how frequently they are updated with different sizes.
	Type Types

	// number of elements if this is a fixed array -- use 1 if singular element, and 0 if a variable-sized array, where each Val can have its own specific size. This also works for arrays of Textures -- up to 128 max.
	ArrayN int

	// role of variable: Vertex is configured in the pipeline VkConfig structure, and everything else is configured in a DescriptorSet.  For TextureRole items, the last such Var in a set will automatically be flagged as variable sized, so the shader can specify: #extension GL_EXT_nonuniform_qualifier : require and the list of textures can be specified as a array.
	Role VarRoles

	// bit flags for set of shaders that this variable is used in
	Shaders vk.ShaderStageFlagBits

	// DescriptorSet associated with the timing of binding for this variable -- all vars updated at the same time should be in the same set
	Set int

	// binding or location number for variable -- Vertexs are assigned as one group sequentially in order listed in Vars, and rest are assigned uniform binding numbers via descriptor pools
	BindLoc int

	// size in bytes of one element (not array size).  Note that arrays in Uniform require 16 byte alignment for each element, so if using arrays, it is best to work within that constraint.  In Storage, with HLSL compute shaders, 4 byte (e.g., float32 or int32) works fine as an array type.  For Push role, SizeOf must be set exactly -- no vals are created.
	SizeOf int

	// texture manages its own memory allocation -- set this for texture objects that change size dynamically -- otherwise image host staging memory is allocated in a common buffer
	TextureOwns bool `edit:"-"`

	// index into the dynamic offset list, where dynamic offsets of vals need to be set -- for Uniform and Storage roles -- set during Set:DescLayout
	DynOffIdx int `edit:"-"`

	// the array of values allocated for this variable.  The size of this array is determined by the Set membership of this Var, and the current index is updated at the set level.  For Texture Roles, there is a separate descriptor for each value (image) -- otherwise dynamic offset binding is used.
	Vals Vals

	// for dynamically bound vars (Vertex, Uniform, Storage), this is the index of the currently bound value in Vals list -- index in this array is the descIdx out of Vars NDescs (see for docs) to allow for parallel update pathways -- only valid until set again -- only actually used for Vertex binding, as unforms etc have the WriteDescriptor mechanism.
	BindValIdx []int `edit:"-"`

	// index of the storage buffer in Memory that holds this Var -- for Storage buffer types.  Due to support for dynamic binding, all Vals of a given Var must be stored in the same buffer, and the allocation mechanism ensures this.  This constrains large vars approaching the MaxStorageBufferRange capacity to only have 1 val, which is typically reasonable given that compute shaders use large data and tend to use static binding anyway, and graphics uses tend to be smaller.
	StorageBuff int `edit:"-"`

	// offset -- only for push constants
	Offset int `edit:"-"`
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
		if vr.ArrayN > 10000 {
			typ = fmt.Sprintf("%s[0x%X]", typ, vr.ArrayN)
		} else {
			typ = fmt.Sprintf("%s[%d]", typ, vr.ArrayN)
		}
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

// MemSizeStorage adds a Storage memory allocation record to Memory
// for all values for this Var
func (vr *Var) MemSizeStorage(mm *Memory, alignBytes int) {
	tsz := vr.Vals.MemSize(vr, alignBytes)
	mm.StorageMems = append(mm.StorageMems, &VarMem{Var: vr, Size: tsz})
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
	case vr.Role == Uniform:
		sz := MemSizeAlign(vr.SizeOf, 16) // todo: test this!
		return sz * n
	default:
		return vr.SizeOf * n
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
	vr.StorageBuff = -1
	// todo: free anything in var
}

// ModRegs returns the regions of Vals that have been modified
func (vr *Var) ModRegs() []MemReg {
	return vr.Vals.ModRegs(vr)
}

// SetTextureDev sets Device for textures
// only called on Role = TextureRole
func (vr *Var) SetTextureDev(dev vk.Device) {
	vals := vr.Vals.ActiveVals()
	for _, vl := range vals {
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
	vals := vr.Vals.ActiveVals()
	vidx := 0
	mx := min(stIdx+MaxTexturesPerSet, len(vals))
	for i := stIdx; i < mx; i++ {
		vl := vals[i]
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

	// variables in order
	Vars []*Var

	// map of vars by name -- names must be unique
	VarMap map[string]*Var
}

// VarByNameTry returns Var by name, returning error if not found
func (vs *VarList) VarByNameTry(name string) (*Var, error) {
	vr, ok := vs.VarMap[name]
	if !ok {
		err := fmt.Errorf("variable named %s not found", name)
		if Debug {
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
