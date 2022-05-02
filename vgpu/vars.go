// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"strings"

	"github.com/goki/ki/indent"
	"github.com/goki/ki/kit"

	vk "github.com/vulkan-go/vulkan"
)

// Var specifies a variable used in a pipeline, but does not manage
// actual values / storage -- see Val for that.
// A Var represents a type of input or output into the GPU program,
// including things like Vertex arrays, transformation matricies (Uniforms),
// Images (Textures), and arbitrary Structs for Compute shaders.
type Var struct {
	Name      string                 `desc:"variable name"`
	Type      Types                  `desc:"type of data in variable.  Note that there are strict contraints on the alignment of fields within structs -- if you can keep all fields at 4 byte increments, that works, but otherwise larger fields trigger a 16 byte alignment constraint.  For images, "`
	Role      VarRoles               `desc:"role of variable: Vertex is configured in the pipeline VkConfig structure, and everything else is configured in a DescriptorSet, etc. "`
	Shaders   vk.ShaderStageFlagBits `desc:"bit flags for set of shaders that this variable is used in"`
	Set       int                    `desc:"DescriptorSet associated with the timing of binding for this variable -- all vars updated at the same time should be in the same set"`
	BindLoc   int                    `desc:"binding or location number for variable -- Vertexs are assigned as one group sequentially in order listed in Vars, and rest are assigned uniform binding numbers via descriptor pools"`
	SizeOf    int                    `desc:"size in bytes of one element (not array size).  Note that arrays require 16 byte alignment for each element, so if using arrays, it is best to work within that constraint."`
	DynOffIdx int                    `desc:"index into the dynamic offset list, where dynamic offsets of vals need to be set"`
	CurVal    *Val                   `desc:"last (current) value set for this variable -- set by System SetVals"`
}

// Init initializes the main values
func (vr *Var) Init(name string, typ Types, role VarRoles, set int, shaders ...ShaderTypes) {
	vr.Name = name
	vr.Type = typ
	vr.Role = role
	vr.SizeOf = typ.Bytes()
	vr.Set = set
	vr.Shaders = 0
	for _, sh := range shaders {
		vr.Shaders |= ShaderStageFlags[sh]
	}
}

func (vr *Var) String() string {
	s := fmt.Sprintf("%d:\t%s\t%s\t(size: %d)", vr.BindLoc, vr.Name, vr.Type.String(), vr.SizeOf)
	return s
}

// BuffType returns the memory buffer type for this variable, based on Var.Role
func (vr *Var) BuffType() BuffTypes {
	return vr.Role.BuffType()
}

//////////////////////////////////////////////////////////////////

// SetDesc contains descriptor information for each set
type SetDesc struct {
	Set     int                    `desc:"set number"`
	Vars    []*Var                 `desc:"variables in order by role, descriptor"`
	Layout  vk.DescriptorSetLayout `desc:"layout info"`
	DescSet vk.DescriptorSet       `desc:"set allocation info"`
}

// Vars are all the variables that are used by a pipeline.
// Vars are allocated to bindings / locations sequentially in the
// order added!
type Vars struct {
	Vars         []*Var                      `desc:"all variables"`
	VarMap       map[string]*Var             `desc:"map of all vars -- names must be unique"`
	RoleMap      map[VarRoles][]*Var         `desc:"map of vars by different roles -- updated in Config(), after all vars added"`
	SetMap       map[int]map[VarRoles][]*Var `desc:"map of vars by set by different roles -- updated in Config(), after all vars added"`
	SetDesc      []*SetDesc                  `desc:"descriptor information for each set"`
	VkDescLayout vk.PipelineLayout           `desc:"vulkan descriptor layout based on vars"`
	VkDescPool   vk.DescriptorPool           `desc:"vulkan descriptor pool"`
	VkDescSets   []vk.DescriptorSet          `desc:"vulkan descriptor sets"`
	DynOffs      []uint32                    `desc:"dynamic offsets for Uniform and Storage variables, indexed as DynOffIdx on Var -- offsets are set when Val is bound"`
}

func (vs *Vars) Destroy(dev vk.Device) {
	vk.DestroyPipelineLayout(dev, vs.VkDescLayout, nil)
	vk.DestroyDescriptorPool(dev, vs.VkDescPool, nil)
	for _, sd := range vs.SetDesc {
		vk.DestroyDescriptorSetLayout(dev, sd.Layout, nil)
	}
}

// AddVar adds given variable
func (vs *Vars) AddVar(vr *Var) {
	if vs.VarMap == nil {
		vs.VarMap = make(map[string]*Var)
	}
	vs.Vars = append(vs.Vars, vr)
	vs.VarMap[vr.Name] = vr
}

// Add adds a new variable of given type, role, set, and shaders where used
func (vs *Vars) Add(name string, typ Types, role VarRoles, set int, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.Init(name, typ, role, set, shaders...)
	vs.AddVar(vr)
	return vr
}

// AddStruct adds a new struct variable of given total number of bytes in size,
// type, role, set, and shaders where used
func (vs *Vars) AddStruct(name string, size int, role VarRoles, set int, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.Init(name, Struct, role, set, shaders...)
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

		if vr.Role < Uniform {
			vr.BindLoc = len(rl) - 1
			continue
		}
		sm := vs.SetMap[vr.Set]
		if sm == nil {
			sm = make(map[VarRoles][]*Var)
		}
		sl := sm[vr.Role]
		vr.BindLoc = len(sl)
		sl = append(sl, vr)
		sm[vr.Role] = sl
		vs.SetMap[vr.Set] = sm
	}
}

func (vs *Vars) StringDoc() string {
	ispc := 4
	var sb strings.Builder
	ns := len(vs.SetMap)
	for si := 0; si < ns; si++ {
		sb.WriteString(fmt.Sprintf("Set: %d\n", si))
		set := vs.SetMap[si]
		for ri := Uniform; ri < VarRolesN; ri++ {
			rl, has := set[ri]
			if !has || len(rl) == 0 {
				continue
			}
			sb.WriteString(fmt.Sprintf("%sRole: %s\n", indent.Spaces(1, ispc), ri.String()))
			for _, vr := range rl {
				sb.WriteString(fmt.Sprintf("%sVar: %s\n", indent.Spaces(2, ispc), vr.String()))
			}
		}
	}
	return sb.String()
}

///////////////////////////////////////////////////////////////////
// Vertex Info

// VkVertexConfig fills in the relevant info into given vulkan config struct
// looking for all vars in order marked as Vertex.
// Note: there is no support for interleaved arrays so each binding and location
// is assigned the same sequential number, recorded in var BindLoc
func (vs *Vars) VkVertexConfig() *vk.PipelineVertexInputStateCreateInfo {
	cfg := &vk.PipelineVertexInputStateCreateInfo{}
	cfg.SType = vk.StructureTypePipelineVertexInputStateCreateInfo
	var bind []vk.VertexInputBindingDescription
	var attr []vk.VertexInputAttributeDescription
	vtx, has := vs.RoleMap[Vertex]
	if !has {
		return cfg
	}
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

// ShaderSet returns the bit flags of all shaders used in variables in given list
func ShaderSet(vl []*Var) vk.ShaderStageFlagBits {
	var sh vk.ShaderStageFlagBits
	for _, vr := range vl {
		sh |= vr.Shaders
	}
	return sh
}

// key info on descriptorCount -- very confusing.
// https://stackoverflow.com/questions/51715944/descriptor-set-count-ambiguity-in-vulkan

// DescLayout returns the PipelineLayout of DescriptorSetLayout
// info for all of the non-Vertex vars
func (vs *Vars) DescLayout(dev vk.Device) {
	nset := len(vs.SetMap)
	vs.SetDesc = make([]*SetDesc, nset)
	dsets := make([]vk.DescriptorSetLayout, nset)
	dyno := 0
	for si := range vs.SetDesc {
		set := vs.SetMap[si]
		sd := &SetDesc{}
		vs.SetDesc[si] = sd
		var descLayout vk.DescriptorSetLayout
		var binds []vk.DescriptorSetLayoutBinding
		var vars []*Var
		bi := 0
		for ri := Uniform; ri < VarRolesN; ri++ {
			rl, has := set[ri]
			if !has || len(rl) == 0 {
				continue
			}
			for _, vr := range rl {
				bd := vk.DescriptorSetLayoutBinding{
					Binding:         uint32(bi),
					DescriptorType:  RoleDescriptors[ri],
					DescriptorCount: 1, // note: only if need an array of *descriptors* -- very rare?
					StageFlags:      vk.ShaderStageFlags(vr.Shaders),
				}
				binds = append(binds, bd)
				bi++
				vars = append(vars, vr)
				if vr.Role == Uniform || vr.Role == Storage {
					vr.DynOffIdx = dyno
					dyno++
				}
			}
		}
		ret := vk.CreateDescriptorSetLayout(dev, &vk.DescriptorSetLayoutCreateInfo{
			SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
			BindingCount: uint32(len(binds)),
			PBindings:    binds,
		}, nil, &descLayout)
		IfPanic(NewError(ret))
		dsets[si] = descLayout
		sd.Layout = descLayout
		sd.Set = si
		sd.Vars = vars
	}

	var pipelineLayout vk.PipelineLayout
	ret := vk.CreatePipelineLayout(dev, &vk.PipelineLayoutCreateInfo{
		SType:          vk.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount: uint32(len(dsets)),
		PSetLayouts:    dsets,
	}, nil, &pipelineLayout)
	IfPanic(NewError(ret))
	vs.VkDescLayout = pipelineLayout

	vs.DynOffs = make([]uint32, dyno)

	if nset == 0 {
		vs.VkDescPool = nil
		vs.VkDescSets = nil
	} else {
		var pools []vk.DescriptorPoolSize
		for rl := Uniform; rl < VarRolesN; rl++ {
			vl := vs.RoleMap[rl]
			if len(vl) == 0 {
				continue
			}
			pl := vk.DescriptorPoolSize{
				DescriptorCount: uint32(len(vl)),
				Type:            RoleDescriptors[rl],
			}
			pools = append(pools, pl)
		}
		var descPool vk.DescriptorPool
		ret = vk.CreateDescriptorPool(dev, &vk.DescriptorPoolCreateInfo{
			SType:         vk.StructureTypeDescriptorPoolCreateInfo,
			MaxSets:       uint32(nset),
			PoolSizeCount: uint32(len(pools)),
			PPoolSizes:    pools,
		}, nil, &descPool)
		IfPanic(NewError(ret))

		vs.VkDescPool = descPool

		vs.VkDescSets = make([]vk.DescriptorSet, len(vs.SetDesc))
		for i, sd := range vs.SetDesc {
			var set vk.DescriptorSet
			ret := vk.AllocateDescriptorSets(dev, &vk.DescriptorSetAllocateInfo{
				SType:              vk.StructureTypeDescriptorSetAllocateInfo,
				DescriptorPool:     vs.VkDescPool,
				DescriptorSetCount: 1,
				PSetLayouts:        []vk.DescriptorSetLayout{sd.Layout},
			}, &set)
			IfPanic(NewError(ret))
			sd.DescSet = set
			vs.VkDescSets[i] = set
		}
	}
}

//////////////////////////////////////////////////////////////////

// VarRoles are the functional roles of variables, corresponding
// to Vertex input vectors and all the different "uniform" types
// as enumerated in vk.DescriptorType.  This does NOT map directly
// to DescriptorType because we combine vertex and uniform data
// and require a different ordering.
type VarRoles int32

const (
	UndefVarRole  VarRoles = iota
	Vertex                 // vertex shader input data: mesh geometry points, normals, etc
	Index                  // for indexed access to Vertex data
	Uniform                // read-only general purpose data, uses UniformBufferDynamic with offset specified at binding time, not during initial configuration -- compared to Storage, Uniform items can be put in local cache for each shader and thus can be much faster to access -- use for a smaller number of parameters such as transformation matricies
	Storage                // read-write general purpose data, in StorageBufferDynamic (offset set at binding) -- this is a larger but slower pool of memory, with more flexible alignment constraints, used primarily for compute data
	UniformTexel           // read-only image-formatted data, which cannot be accessed via ImageView or Sampler -- only for rare cases where optimized image format (e.g., rgb values of specific bit count) is useful.  No Dynamic mode is available, so this can only be used for a fixed Val.
	StorageTexel           // read-write image-formatted data, which cannot be accessed via ImageView or Sampler -- only for rare cases where optimized image format (e.g., rgb values of specific bit count) is useful. No Dynamic mode is available, so this can only be used for a fixed Val.
	StorageImage           // read-write access through an ImageView (but not a Sampler) of an Image
	SamplerVar             // this does not have a corresponding Val data, but rather specifies the unique name of a Sampler on the System -- it is here as a variable so Vars can fully specify the descriptor layout.
	SampledImage           // a read-only Image Val that can be fed to the Sampler in a shader -- must be presented via an ImageView?
	CombinedImage          // a combination of a Sampler and a specific Image, which appears as a single entity in the shader.  The Var specifies the name of the Sampler, but the corresponding Val that points to this Var holds the image.  This is the simplest way to specify a texture for texture mapping.
	VarRolesN
)

//go:generate stringer -type=VarRoles

var KiT_VarRoles = kit.Enums.AddEnum(VarRolesN, kit.NotBitFlag, nil)

// IsDynamic returns true if role has dynamic offset binding
func (vr VarRoles) IsDynamic() bool {
	return vr == Uniform || vr == Storage
}

// BuffType returns type of memory buffer for this role
func (vr VarRoles) BuffType() BuffTypes {
	return RoleBuffers[vr]
}

var RoleDescriptors = map[VarRoles]vk.DescriptorType{
	Uniform:       vk.DescriptorTypeUniformBufferDynamic,
	Storage:       vk.DescriptorTypeStorageBufferDynamic,
	UniformTexel:  vk.DescriptorTypeUniformTexelBuffer,
	StorageTexel:  vk.DescriptorTypeStorageTexelBuffer,
	StorageImage:  vk.DescriptorTypeStorageImage,
	SamplerVar:    vk.DescriptorTypeSampler,
	SampledImage:  vk.DescriptorTypeSampledImage,
	CombinedImage: vk.DescriptorTypeCombinedImageSampler,
}

// RoleBuffers maps VarRoles onto type of memory buffer
var RoleBuffers = map[VarRoles]BuffTypes{
	UndefVarRole:  StorageBuff,
	Vertex:        VtxIdxBuff,
	Index:         VtxIdxBuff,
	Uniform:       UniformBuff,
	Storage:       StorageBuff,
	UniformTexel:  UniformBuff,
	StorageTexel:  StorageBuff,
	StorageImage:  StorageBuff,
	SamplerVar:    ImageBuff,
	SampledImage:  ImageBuff,
	CombinedImage: ImageBuff,
}
