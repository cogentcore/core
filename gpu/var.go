// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"log"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Var specifies a variable used in a pipeline, accessed in shader programs
// A Var represents a type of input or output into the GPU program,
// including things like Vertex arrays, transformation matricies (Uniforms),
// Images (Textures), and arbitrary Structs for Compute shaders.
// There are one or more corresponding Value items for each Var, which represent
// the actual value of the variable: Var only represents all the type-level info.
// Each Var belongs to a Group, and its Binding location is allocated within that,
// and these numbers are used in WGSL shader via @group and @binding to refer to
// the variables. The entire Group is updated at the same time scale,
// and all vars in the Group have the same number of allocated Value instances
// representing a specific value of the variable.
// There must be a unique Value instance for each value of the variable used in
// a single render: a previously used Value's contents cannot be updated within
// the render pass, but new information can be written to an as-yet unused Value
// prior to using in a render (although this comes at a performance cost).
type Var struct {

	// variable name
	Name string

	// type of data in variable.  Note that there are strict contraints
	// on the alignment of fields within structs.  If you can keep all fields
	// at 4 byte increments, that works, but otherwise larger fields trigger
	// a 16 byte alignment constraint.  Texture Images do not have such alignment
	// constraints, and can be allocated in a big host buffer or in separate
	// buffers depending on how frequently they are updated with different sizes.
	Type Types

	// number of elements if this is a fixed array. Use 1 if singular element,
	// and 0 if a variable-sized array, where each Value can have its own
	// specific size. This also works for arrays of Textures, up to 128 max.
	ArrayN int

	// role of variable: Vertex is configured separately, and everything else
	// is configured in a BindGroup.  Textures are accessed via arrays always.
	// Note: Push is not yet supported.
	Role VarRoles

	// bit flags for set of shaders that this variable is used in, determined
	// by the Role.
	Shaders wgpu.ShaderStage `edit:"-"`

	// Group binding for this variable, indicated by @group in WGSL shader.
	// In general, put data that is updated at the same point in time in the same
	// group, as everything within a group is updated together.
	Group int

	// binding number for this variable, indicated by @binding in WGSL shader.
	// These are automatically assigned sequentially within Group.
	Binding int `edit:"-"`

	// size in bytes of one element (not array size).
	// Note that arrays in Uniform require 16 byte alignment for each element,
	// so if using arrays, it is best to work within that constraint.
	// In Storage, 4 byte (e.g., float32 or int32) works fine as an array type.
	// For Push role, SizeOf must be set exactly, as no vals are created.
	SizeOf int

	// Texture manages its own memory allocation, and this indicates that
	// the texture object can change size dynamically.
	// Otherwise image host staging memory is allocated in a common buffer
	TextureOwns bool `edit:"-"`

	// todo: still needed?

	// Index into the dynamic offset list, where dynamic offsets of vals
	// need to be set. For Uniform and Storage Roles, set during BindGroupLayout.
	DynOffIndex int `edit:"-"`

	// Values is the the array of Values allocated for this variable.
	// The size of this array is determined by the Group membership of this Var,
	// and the current index is updated at the group level.
	// For Texture Roles, there is a separate descriptor for each value (image),
	// otherwise dynamic offset binding is used.
	Values Values

	// For dynamically bound vars (Vertex, Uniform, Storage), this is the
	// index of the currently bound value in the Values list.
	// The index in this array is the descIndex out of Vars NDescs
	// (see for docs) to allow for parallel update pathways.
	// Only valid until set again, and only actually used for Vertex binding,
	// as unforms etc have the WriteDescriptor mechanism.
	BindValueIndex []int `edit:"-"`

	// index of the storage buffer in Memory that holds this Var, for Storage
	// buffer types.  Due to support for dynamic binding, all Values of a
	// given Var must be stored in the same buffer, and the allocation
	// mechanism ensures this.  This constrains large vars approaching the
	// MaxStorageBuffererRange capacity to only have 1 val, which is typically
	// reasonable given that compute shaders use large data and tend to use static
	// binding anyway, and graphics uses tend to be smaller.
	StorageBuffer int `edit:"-"`

	// offset: only for push constants
	Offset int `edit:"-"`
}

// Init initializes the main values
func (vr *Var) Init(name string, typ Types, arrayN int, role VarRoles, group int, shaders ...ShaderTypes) {
	vr.Name = name
	vr.Type = typ
	vr.ArrayN = arrayN
	vr.Role = role
	vr.SizeOf = typ.Bytes()
	vr.Group = group
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
	s := fmt.Sprintf("%d:\t%s\t%s\t(size: %d)\tValues: %d", vr.Binding, vr.Name, typ, vr.SizeOf, len(vr.Values.Values))
	return s
}

// BuffType returns the memory buffer type for this variable, based on Var.Role
func (vr *Var) BuffType() BufferTypes {
	return vr.Role.BuffType()
}

// BindValue returns the currently bound value at given descriptor collection index
// as set by BindDyn* methods.  Returns nil, error if not valid.
func (vr *Var) BindValue(descIndex int) (*Value, error) {
	idx := vr.BindValueIndex[descIndex]
	return vr.Values.ValueByIndexTry(idx)
}

// ValuesMemSize returns the memory allocation size
// for all values for this Var, in bytes
func (vr *Var) ValuesMemSize(alignBytes int) int {
	return vr.Values.MemSize(vr, alignBytes)
}

// MemSizeStorage adds a Storage memory allocation record to Memory
// for all values for this Var
func (vr *Var) MemSizeStorage(mm *Memory, alignBytes int) {
	tsz := vr.Values.MemSize(vr, alignBytes)
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

// AllocMem allocates values at given offset in given Memory buffer.
// Computes the MemPtr for each item, and returns TotSize
// across all vals.  The effective offset increment (based on size) is
// aligned at the given align byte level, which should be
// MinUniformBuffererOffsetAlignment from gpu.
func (vr *Var) AllocMem(buff *Buffer, offset int) int {
	return vr.Values.AllocMem(vr, buff, offset)
}

// Free resets the MemPtr for values, resets any self-owned resources (Textures)
func (vr *Var) Free() {
	vr.Values.Free()
	vr.StorageBuffer = -1
	// todo: free anything in var
}

// ModRegs returns the regions of Values that have been modified
func (vr *Var) ModRegs() []MemReg {
	return vr.Values.ModRegs(vr)
}

// SetTextureDev sets Device for textures
// only called on Role = TextureRole
func (vr *Var) SetTextureDev(dev *Device) {
	vals := vr.Values.ActiveValues()
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
	vr.Values.AllocTextures(mm)
}

// TextureValidIndex returns the index of the given texture value at our
// index in list of vals, starting at given index, skipping over any
// inactive textures which do not show up when accessed in the shader.
// You must use this value when passing a texture index to the shader!
// returns -1 if idx is not valid
func (vr *Var) TextureValidIndex(stIndex, idx int) int {
	vals := vr.Values.ActiveValues()
	vidx := 0
	mx := min(stIndex+MaxTexturesPerGroup, len(vals))
	for i := stIndex; i < mx; i++ {
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
		err := fmt.Errorf("Variable named %s not found", name)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vr, nil
}

// ValueByNameTry returns value by first looking up variable name, then value name,
// returning error if not found
func (vs *VarList) ValueByNameTry(varName, valName string) (*Var, *Value, error) {
	vr, err := vs.VarByNameTry(varName)
	if err != nil {
		return nil, nil, err
	}
	vl, err := vr.Values.ValueByNameTry(valName)
	return vr, vl, err
}

// ValueByIndexTry returns value by first looking up variable name, then value index,
// returning error if not found
func (vs *VarList) ValueByIndexTry(varName string, valIndex int) (*Var, *Value, error) {
	vr, err := vs.VarByNameTry(varName)
	if err != nil {
		return nil, nil, err
	}
	vl, err := vr.Values.ValueByIndexTry(valIndex)
	return vr, vl, err
}
