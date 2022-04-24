// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"github.com/goki/ki/kit"

	vk "github.com/vulkan-go/vulkan"
)

// Var specifies a variable used in a pipeline, but does not manage
// actual values / storage -- see Val for that.
// A Var represents a type of input or output into the GPU program,
// including things like Vertex arrays, transformation matricies (Uniforms),
// Images (Textures), and arbitrary Structs for Compute shaders.
type Var struct {
	Name    string   `desc:"variable name"`
	Type    Types    `desc:"type of data in variable"`
	Role    VarRoles `desc:"role of variable: VertexInput is configured in the pipeline VkConfig structure, and "`
	Set     int      `desc:"DescriptorSet associated with the timing of binding for this variable -- all vars updated at the same time should be in the same set"`
	BindLoc int      `desc:"binding or location number for variable -- VertexInputs are assigned as one group sequentially in order listed in Vars, and rest are assigned uniform binding numbers via descriptor pools"`
	SizeOf  int      `desc:"size in bytes of one element (not array size)"`
}

// Init initializes the main values
func (vr *Var) Init(name string, typ Types, role VarRoles, set int) {
	vr.Name = name
	vr.Type = typ
	vr.Role = role
	vr.SizeOf = TypeSizes[typ]
	vr.Set = set
}

//////////////////////////////////////////////////////////////////

// Vars are all the variables that are used by a pipeline.
// Vars are allocated to bindings / locations sequentially in the
// order added!
type Vars struct {
	Vars    []*Var                      `desc:"all variables"`
	VarMap  map[string]*Var             `desc:"map of all vars -- names must be unique"`
	RoleMap map[VarRoles][]*Var         `desc:"map of vars by different roles -- updated in Config(), after all vars added"`
	SetMap  map[int]map[VarRoles][]*Var `desc:"map of vars by set by different roles -- updated in Config(), after all vars added"`
}

// AddVar adds given variable
func (vs *Vars) AddVar(vr *Var) {
	if vs.VarMap == nil {
		vs.VarMap = make(map[string]*Var)
	}
	vs.Vars = append(vs.Vars, vr)
	vs.VarMap[vr.Name] = vr
}

// Add adds a new variable of given type, role, and set
func (vs *Vars) Add(name string, typ Types, role VarRoles, set int) *Var {
	vr := &Var{}
	vr.Init(name, typ, role, set)
	vs.AddVar(vr)
	return vr
}

// AddStruct adds a new struct variable of given total number of bytes in size,
// type, role, and set
func (vs *Vars) AddStruct(name string, size int, role VarRoles, set int) *Var {
	vr := &Var{}
	vr.Init(name, Struct, role, set)
	vr.SizeOf = size
	vs.AddVar(vr)
	return vr
}

// Config must be called after all variables have been added.
// configures additional maps by Set and Role to manage variables.
func (vs *Vars) Config() {
	vs.RoleMap = make(map[VarRoles][]*Var)
	vs.SetMap = make(map[int]map[VarRoles][]*Var)
	for _, vr := range vs.Vars {
		rl := vs.RoleMap[vr.Role]
		rl = append(rl, vr)
		vs.RoleMap[vr.Role] = rl

		if vr.Role < UniformVar {
			vr.BindLoc = len(rl) - 1
			continue
		}
		sm := vs.SetMap[vr.Set]
		if sm == nil {
			sm = make(map[VarRoles][]*Var)
		}
		sl := sm[vr.Role]
		vr.BindLoc = len(sl)
		rl = append(rl, vr)
		sm[vr.Role] = rl
		vs.SetMap[vr.Set] = sm
	}
}

///////////////////////////////////////////////////////////////////
// VertexInput Info

// VkVertexConfig fills in the relevant info into given vulkan config struct
// looking for all vars in order marked as VertexInput.
// Note: there is no support for interleaved arrays so each binding and location
// is assigned the same sequential number, recorded in var BindLoc
func (vs *Vars) VkVertexConfig() *vk.PipelineVertexInputStateCreateInfo {
	cfg := &vk.PipelineVertexInputStateCreateInfo{}
	cfg.SType = vk.StructureTypePipelineVertexInputStateCreateInfo
	var bind []vk.VertexInputBindingDescription
	var attr []vk.VertexInputAttributeDescription
	vtx := vs.RoleMap[VertexInput]
	for _, vr := range vtx {
		bind = append(bind, vk.VertexInputBindingDescription{
			Binding:   uint32(vr.BindLoc),
			Stride:    uint32(vr.SizeOf),
			InputRate: vk.VertexInputRateVertex,
		})
		attr = append(attr, vk.VertexInputAttributeDescription{
			Location: uint32(vr.BindLoc),
			Binding:  uint32(vr.BindLoc),
			Format:   vk.Format(vr.Type),
			Offset:   0,
		})
	}
	cfg.VertexBindingDescriptionCount = uint32(len(vtx))
	cfg.PVertexBindingDescriptions = bind
	cfg.VertexAttributeDescriptionCount = uint32(len(vtx))
	cfg.PVertexAttributeDescriptions = attr
	return cfg
}

///////////////////////////////////////////////////////////////////
// Descriptors for Uniforms etc

// DescSets returns the DescriptorSets for Uniform* vars
// func (vs *Vars) DescSets() []vk.DescriptorPoolSize {
// }

// DescPools returns the collection of each Role of variable
func (vs *Vars) DescPools() []vk.DescriptorPoolSize {
	var pools []vk.DescriptorPoolSize
	for rl := UniformVar; rl < VarRolesN; rl++ {
		vl := vs.RoleMap[rl]
		if len(vl) == 0 {
			continue
		}
		pl := vk.DescriptorPoolSize{DescriptorCount: uint32(len(vl))}
		switch rl {
		case UniformVar:
			pl.Type = vk.DescriptorTypeUniformBufferDynamic
		case StorageVar:
			pl.Type = vk.DescriptorTypeStorageBufferDynamic
			// todo: images!
		}
		pools = append(pools, pl)
	}
	return pools
}

//////////////////////////////////////////////////////////////////

// VarRoles are the functional roles of variables.
type VarRoles int32

const (
	UndefVarRole VarRoles = iota
	VertexInput
	VertexOutput // is this needed?
	Indexes
	UniformVar // in UniformBuffer Descriptor Pool
	StorageVar // in StorageBuffer Descriptor Pool
	ImageVar   // todo: need more options: SampledImage, StorageImage, etc
	VarRolesN
)

//go:generate stringer -type=VarRoles

var KiT_VarRoles = kit.Enums.AddEnum(VarRolesN, kit.NotBitFlag, nil)
