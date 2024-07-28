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
// at a specific @group (from VarGroup owner) and @binding location.
// There must be a different Var for each type of input or output into
// the GPU program, including things like Vertex arrays, transformation
// matricies (Uniforms), Textures, and arbitrary Storage data (Structs)
// for Compute shaders.
// There are one or more corresponding Value items for each Var, which represent
// the actual value of the variable: Var only represents the type-level info.
// Each Var belongs to a VarGroup, and its Binding location is sequential within that.
// The entire Group is updated at the same time from the hardware perspective,
// and less-frequently-updated items should be in the lower-numbered groups.
// The Role
type Var struct {
	// variable name
	Name string

	// type of data in variable.  Note that there are strict contraints
	// on the alignment of fields within structs.  If you can keep all fields
	// at 4 byte increments, that works, but otherwise larger fields trigger
	// a 16 byte alignment constraint.  Texture Textures do not have such alignment
	// constraints, and are stored separately or in arrays organized by size.
	// Use Float32Matrix4 for model matricies in Vertex role, which will
	// automatically be sent as 4 interleaved Float32Vector4 chuncks.
	Type Types

	// number of elements if this is a fixed array. Use 1 if singular element,
	// and 0 if a variable-sized array, where each Value can have its own
	// specific size. This also works for arrays of Textures, up to 128 max.
	ArrayN int

	// Role of variable: Vertex is configured separately, and everything else
	// is configured in a BindGroup.  This is inherited from the VarGroup
	// and all Vars in a Group must have the same Role, except Index is also
	// included in the VertexGroup (-2).
	// Note: Push is not yet supported.
	Role VarRoles

	// VertexInstance is whether this Vertex role variable is specified
	// per instance (true) or per vertex (false, default).
	// Instance variables can be used for sending per-object data like
	// the model matrix (as Float32Matrix4 which is serialized as 4
	// Float32Vector4 values).  Can also send texture indexes,
	// per object color, etc.
	VertexInstance bool

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

	// Values is the the array of Values allocated for this variable.
	// Each value has its own corresponding Buffer or Texture.
	// The currently-active Value is specified by the Current index,
	// and this is what will be used for Binding.
	Values Values

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

// MemSize returns the memory allocation size for this value, in bytes
func (vr *Var) MemSize() int {
	n := vr.ArrayN
	if n == 0 {
		n = 1
	}
	switch {
	case vr.Role >= SampledTexture:
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

// Free resets the MemPtr for values, resets any self-owned resources (Textures)
func (vr *Var) Free() {
	vr.Values.Free()
	// todo: free anything in var
}

// SetNValues sets specified number of Values for this var.
// returns true if changed.
func (vr *Var) SetNValues(dev *Device, nvals int) bool {
	return vr.Values.SetN(vr, dev, nvals)
}

// SetCurrentValue sets the Current Value index, which is
// the Value that will be used in rendering, via BindGroup
func (vr *Var) SetCurrentValue(i int) {
	vr.Values.Current = i
}

// BindGroupEntry returns the BindGroupEntry for Current
// value for this variable.
func (vr *Var) BindGroupEntry() wgpu.BindGroupEntry {
	return vr.Values.BindGroupEntry(vr)
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
