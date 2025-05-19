// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"

	"github.com/cogentcore/webgpu/wgpu"
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

	// Type of data in variable.  Note that there are strict contraints
	// on the alignment of fields within structs.  If you can keep all fields
	// at 4 byte increments, that works, but otherwise larger fields trigger
	// a 16 byte alignment constraint.  Textures do not have such alignment
	// constraints, and are stored separately or in arrays organized by size.
	// Use Float32Matrix4 for model matricies in Vertex role, which will
	// automatically be sent as 4 interleaved Float32Vector4 chuncks.
	Type Types

	// ArrayN is the number of elements in an array, only if there is a
	// fixed array size. Otherwise, for single elements or dynamic arrays
	// use a value of 1. There can be alignment issues with arrays
	// so make sure your elemental types are compatible.
	// Note that DynamicOffset variables can have Value buffers with multiple
	// instances of the variable (with proper alignment stride),
	// which goes on top of any array value for the variable itself.
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
	shaders wgpu.ShaderStage `edit:"-"`

	// Group binding for this variable, indicated by @group in WGSL shader.
	// In general, put data that is updated at the same point in time in the same
	// group, as everything within a group is updated together.
	Group int

	// binding number for this variable, indicated by @binding in WGSL shader.
	// These are automatically assigned sequentially within Group.
	Binding int `edit:"-"`

	// size in bytes of one element (exclusive of array size).
	// Note that arrays in Uniform require 16 byte alignment for each element,
	// so if using arrays, it is best to work within that constraint.
	// In Storage, 4 byte (e.g., float32 or int32) works fine as an array type.
	// For Push role, SizeOf must be set exactly, as no vals are created.
	SizeOf int

	// DynamicOffset indicates whether the specific Value to use
	// is specified using a dynamic offset specified in the Value
	// via DynamicIndex.  There are limits on the number of dynamic
	// variables within each group (as few as 4).
	// Only for Uniform and Storage variables.
	DynamicOffset bool

	// ReadOnly applies only to [Storage] variables, and indicates that
	// they are never read back from the GPU, so the additional staging
	// buffers needed to do so are not created for these variables.
	ReadOnly bool

	// Values is the the array of Values allocated for this variable.
	// Each value has its own corresponding Buffer or Texture.
	// The currently-active Value is specified by the Current index,
	// and this is what will be used for Binding.
	Values Values

	// offset: for push constants, not currently used.
	offset int `edit:"-"`

	// the alignment requirement in bytes for DynamicOffset variables.
	// This is 1 for Vertex buffer variables.
	alignBytes int

	// var group we are in
	VarGroup *VarGroup
}

// NewVar returns a new Var in given var group
func NewVar(vg *VarGroup, name string, typ Types, arrayN int, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.init(vg, name, typ, arrayN, shaders...)
	return vr
}

// init initializes the main values
func (vr *Var) init(vg *VarGroup, name string, typ Types, arrayN int, shaders ...ShaderTypes) {
	vr.VarGroup = vg
	vr.Name = name
	vr.Type = typ
	vr.ArrayN = max(arrayN, 1)
	vr.Role = vg.Role
	vr.SizeOf = typ.Bytes()
	vr.Group = vg.Group
	vr.alignBytes = vg.alignBytes
	vr.shaders = 0
	for _, sh := range shaders {
		vr.shaders |= ShaderStageFlags[sh]
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
	if len(vr.Values.Values) == 1 {
		s += "\t" + vr.Values.Values[0].String()
	}
	return s
}

// MemSize returns the memory allocation size for this value, in bytes
func (vr *Var) MemSize() int {
	if vr.ArrayN < 1 {
		vr.ArrayN = 1
	}
	switch {
	case vr.Role >= SampledTexture:
		return 0
	default:
		return vr.SizeOf * vr.ArrayN
	}
}

// Release resets the MemPtr for values, resets any self-owned resources (Textures)
func (vr *Var) Release() {
	vr.Values.Release()
}

// SetNValues sets specified number of Values for this var.
// returns true if changed.
func (vr *Var) SetNValues(dev *Device, nvals int) bool {
	return vr.Values.SetN(vr, dev, nvals)
}

// SetCurrentValue sets the Current Value index, which is
// the Value that will be used in rendering, via BindGroup
func (vr *Var) SetCurrentValue(i int) {
	vr.Values.SetCurrentValue(i)
}

// bindGroupEntry returns the BindGroupEntry for Current
// value for this variable.
func (vr *Var) bindGroupEntry() []wgpu.BindGroupEntry {
	return vr.Values.bindGroupEntry(vr)
}

func (vr *Var) bindingType() wgpu.BufferBindingType {
	if vr.Role == Storage && vr.ReadOnly {
		return wgpu.BufferBindingTypeReadOnlyStorage
	}
	return vr.Role.BindingType()
}

func (vr *Var) bufferUsages() wgpu.BufferUsage {
	if vr.Role == Storage && vr.ReadOnly {
		return wgpu.BufferUsageStorage | wgpu.BufferUsageCopyDst
	}
	return vr.Role.BufferUsages()
}
