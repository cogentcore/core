// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"

	vk "github.com/vulkan-go/vulkan"
)

const VertexSet = -1

// VarSet contains a set of Var variables that are all updated at the same time
// and have the same number of distinct Vals values per Var per render pass.
// The first set at index -1 contains Vertex and Index data, handed separately.
type VarSet struct {
	VarList
	Set      int                 `desc:"set number"`
	NValsPer int                 `desc:"number of value instances to allocate per variable in this set: each value must be allocated in advance for each unique instance of a variable required across a complete scene rendering -- e.g., if this is an object position matrix, then one per object is required.  If a dynamic number are required, allocate the max possible"`
	RoleMap  map[VarRoles][]*Var `desc:"map of vars by different roles, within this set -- updated in Config(), after all vars added"`

	VkLayout   vk.DescriptorSetLayout `desc:"set layout info -- static description of each var type, role, binding, stages"`
	VkDescSets []vk.DescriptorSet     `desc:"allocated descriptor set -- one of these per Vars.NDescs -- can have multiple sets that can be independently updated, e.g., for parallel rendering passes.  If only rendering one at a time, only need one."`
}

// AddVar adds given variable
func (st *VarSet) AddVar(vr *Var) {
	if st.VarMap == nil {
		st.VarMap = make(map[string]*Var)
	}
	st.Vars = append(st.Vars, vr)
	st.VarMap[vr.Name] = vr
}

// Add adds a new variable of given type, role, arrayN, and shaders where used
func (st *VarSet) Add(name string, typ Types, arrayN int, role VarRoles, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.Init(name, typ, arrayN, role, st.Set, shaders...)
	st.AddVar(vr)
	return vr
}

// AddStruct adds a new struct variable of given total number of bytes in size,
// type, role, set, and shaders where used
func (st *VarSet) AddStruct(name string, size int, arrayN int, role VarRoles, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.Init(name, Struct, arrayN, role, st.Set, shaders...)
	vr.SizeOf = size
	st.AddVar(vr)
	return vr
}

// Config must be called after all variables have been added.
// configures binding / location for all vars based on sequential order.
// also does validation and returns error message.
func (st *VarSet) Config() error {
	st.RoleMap = make(map[VarRoles][]*Var)
	var cerr error
	bloc := 0
	for _, vr := range st.Vars {
		if st.Set == VertexSet && vr.Role > Index {
			err := fmt.Errorf("vgpu.Set:Config VertexSet cannot contain variables of role: %s  var: %s", vr.Role.String(), vr.Name)
			cerr = err
			if TheGPU.Debug {
				log.Println(err)
			}
			continue
		}
		rl := st.RoleMap[vr.Role]
		rl = append(rl, vr)
		st.RoleMap[vr.Role] = rl
		vr.BindLoc = bloc
		bloc++
	}
	return cerr
}

// ConfigVals configures the Vals for the vars in this set, allocating
// nvals per variable.  There must be a unique value available for each
// distinct value to be rendered within a single pass.  All Vars in the
// same set have the same number of vals.
// Any existing vals will be deleted -- must free all associated memory prior!
func (st *VarSet) ConfigVals(nvals int) {
	st.NValsPer = nvals
	for _, vr := range st.Vars {
		vr.Vals.ConfigVals(vr, nvals)
	}
}

// Destroy destroys infrastructure for Set, Vars and Vals -- assumes Free has
// already been called to free host and device memory.
func (st *VarSet) Destroy(dev vk.Device) {
	st.DestroyLayout(dev)
}

// DestroyLayout destroys layout
func (st *VarSet) DestroyLayout(dev vk.Device) {
	vk.DestroyDescriptorSetLayout(dev, st.VkLayout, nil)
	st.VkLayout = nil
}

// DescLayout creates the DescriptorSetLayout in DescLayout for given set.
// Only for non-VertexSet sets.
// Must have set NValsPer for any TextureRole vars, which require separate descriptors per.
func (st *VarSet) DescLayout(dev vk.Device, vs *Vars) {
	if st.Set == VertexSet {
		for _, vr := range st.Vars {
			vr.BindValIdx = make([]int, vs.NDescs)
		}
		return
	}
	st.DestroyLayout(dev)
	var descLayout vk.DescriptorSetLayout
	var binds []vk.DescriptorSetLayoutBinding
	dyno := len(vs.DynOffs[0])
	for _, vr := range st.Vars {
		bd := vk.DescriptorSetLayoutBinding{
			Binding:         uint32(vr.BindLoc),
			DescriptorType:  RoleDescriptors[vr.Role],
			DescriptorCount: 1,
			StageFlags:      vk.ShaderStageFlags(vr.Shaders),
		}
		if vr.Role > Storage {
			bd.DescriptorCount = uint32(st.NValsPer)
		}
		binds = append(binds, bd)
		if vr.Role == Uniform || vr.Role == Storage {
			vr.BindValIdx = make([]int, vs.NDescs)
			vr.DynOffIdx = dyno
			vs.AddDynOff()
			dyno++
		}
	}
	ret := vk.CreateDescriptorSetLayout(dev, &vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: uint32(len(binds)),
		PBindings:    binds,
	}, nil, &descLayout)
	IfPanic(NewError(ret))
	st.VkLayout = descLayout

	st.VkDescSets = make([]vk.DescriptorSet, vs.NDescs)
	for i := 0; i < vs.NDescs; i++ {
		var dset vk.DescriptorSet
		ret := vk.AllocateDescriptorSets(dev, &vk.DescriptorSetAllocateInfo{
			SType:              vk.StructureTypeDescriptorSetAllocateInfo,
			DescriptorPool:     vs.VkDescPool,
			DescriptorSetCount: 1,
			PSetLayouts:        []vk.DescriptorSetLayout{st.VkLayout},
		}, &dset)
		IfPanic(NewError(ret))
		st.VkDescSets[i] = dset
	}
}

// BindDynValName dynamically binds given uniform or storage value
// by name for given variable name.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (st *VarSet) BindDynValName(vs *Vars, varNm, valNm string) error {
	vr, vl, err := st.ValByNameTry(varNm, valNm)
	if err != nil {
		return err
	}
	st.BindDynVal(vs, vr, vl)
	return nil
}

// BindDynValIdx dynamically binds given uniform or storage value
// by index for given variable name.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (st *VarSet) BindDynValIdx(vs *Vars, varNm string, valIdx int) error {
	vr, vl, err := st.ValByIdxTry(varNm, valIdx)
	if err != nil {
		return err
	}
	return st.BindDynVal(vs, vr, vl)
}

// BindDynVal dynamically binds given uniform or storage value
// for given variable name.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (st *VarSet) BindDynVal(vs *Vars, vr *Var, vl *Val) error {
	if vr.Role < Uniform || vr.Role > Storage {
		err := fmt.Errorf("vgpu.Set:BindDynVal dynamic binding only valid for Uniform or Storage Vars, not: %s", vr.Role.String())
		if TheGPU.Debug {
			log.Println(err)
		}
		return err
	}
	vr.BindValIdx[vs.BindDescIdx] = vl.Idx // note: not used but potentially informative
	// todo: we probably do not need to create this for dyn binding, but we DO need to do
	// it at least at the start -- should create a separate pathway for that.
	wd := vk.WriteDescriptorSet{
		SType:           vk.StructureTypeWriteDescriptorSet,
		DstSet:          st.VkDescSets[vs.BindDescIdx],
		DstBinding:      uint32(vr.BindLoc),
		DescriptorCount: 1,
		DescriptorType:  vr.Role.VkDescriptor(),
	}
	buff := vs.Mem.Buffs[vr.BuffType()]
	wd.PBufferInfo = []vk.DescriptorBufferInfo{{
		Offset: 0, // dynamic
		Range:  vk.DeviceSize(vl.AllocSize),
		Buffer: buff.Dev,
	}}
	vs.DynOffs[vs.BindDescIdx][vr.DynOffIdx] = uint32(vl.Offset)
	vs.VkWriteVals = append(vs.VkWriteVals, wd)
	return nil
}

// todo: other static cases need same approach as images!
// also, need an option to allow a single val to be used in a static way, selecting from among multiple,
// instead of always assuming an array used.

// BindStatVars binds all static vars to their current values,
// for non-Uniform, Storage, variables (e.g., Textures).
// Each Val for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
func (st *VarSet) BindStatVars(vs *Vars) {
	for _, vr := range st.Vars {
		if vr.Role < Storage {
			continue
		}
		st.BindStatVar(vs, vr)
	}
}

// BindStatVarName does static variable binding for given var
// looked up by name, For non-Uniform, Storage, variables (e.g., Textures).
// Each Val for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
func (st *VarSet) BindStatVarName(vs *Vars, varNm string) error {
	vr, err := st.VarByNameTry(varNm)
	if err != nil {
		return err
	}
	st.BindStatVar(vs, vr)
	return nil
}

// BindStatVar does static variable binding for given var,
// For non-Uniform, Storage, variables (e.g., Textures).
// Each Val for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
func (st *VarSet) BindStatVar(vs *Vars, vr *Var) {
	nvals := len(vr.Vals.Vals)
	wd := vk.WriteDescriptorSet{
		SType:           vk.StructureTypeWriteDescriptorSet,
		DstSet:          st.VkDescSets[vs.BindDescIdx],
		DstBinding:      uint32(vr.BindLoc),
		DescriptorCount: uint32(nvals),
		DescriptorType:  vr.Role.VkDescriptor(),
	}
	buff := vs.Mem.Buffs[vr.BuffType()]
	if vr.Role < TextureRole {
		bis := make([]vk.DescriptorBufferInfo, nvals)
		for i, vl := range vr.Vals.Vals {
			bis[i] = vk.DescriptorBufferInfo{
				Offset: vk.DeviceSize(vl.Offset),
				Range:  vk.DeviceSize(vl.AllocSize),
				Buffer: buff.Dev,
			}
		}
		wd.PBufferInfo = bis
	} else {
		imgs := make([]vk.DescriptorImageInfo, nvals)
		for i, vl := range vr.Vals.Vals {
			imgs[i] = vk.DescriptorImageInfo{
				ImageLayout: vk.ImageLayoutShaderReadOnlyOptimal,
				ImageView:   vl.Texture.View,
				Sampler:     vl.Texture.VkSampler,
			}
		}
		wd.PImageInfo = imgs
	}
	vs.VkWriteVals = append(vs.VkWriteVals, wd)
}

// VkVertexConfig fills in the relevant info into given vulkan config struct.
// for VertexSet only!
// Note: there is no support for interleaved arrays so each binding and location
// is assigned the same sequential number, recorded in var BindLoc
func (st *VarSet) VkVertexConfig() *vk.PipelineVertexInputStateCreateInfo {
	cfg := &vk.PipelineVertexInputStateCreateInfo{}
	cfg.SType = vk.StructureTypePipelineVertexInputStateCreateInfo
	var bind []vk.VertexInputBindingDescription
	var attr []vk.VertexInputAttributeDescription
	for _, vr := range st.Vars {
		if vr.Role != Vertex { // not Index
			continue
		}
		bind = append(bind, vk.VertexInputBindingDescription{
			Binding:   uint32(vr.BindLoc),
			Stride:    uint32(vr.SizeOf),
			InputRate: vk.VertexInputRateVertex,
		})
		attr = append(attr, vk.VertexInputAttributeDescription{
			Location: uint32(vr.BindLoc),
			Binding:  uint32(vr.BindLoc),
			Format:   vr.Type.VkFormat(),
			Offset:   0,
		})
	}
	cfg.VertexBindingDescriptionCount = uint32(len(bind))
	cfg.PVertexBindingDescriptions = bind
	cfg.VertexAttributeDescriptionCount = uint32(len(attr))
	cfg.PVertexAttributeDescriptions = attr
	return cfg
}
