// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/goki/ki/ints"
	vk "github.com/goki/vulkan"
)

// maxPerStageDescriptorSamplers is only 16 on mac -- this is the relevant limit on textures!
// also maxPerStageDescriptorSampledImages is basically the same:
// https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxPerStageDescriptorSamplers&platform=all
// https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxPerStageDescriptorSampledImages&platform=all

const (
	// MaxTexturesPerSet is the maximum number of image variables that can be used
	// in one descriptor set.  This value is a lowest common denominator across
	// platforms.  To overcome this limitation, when more Texture vals are allocated,
	// multiple NDescs are used, setting the and switch
	// across those -- each such Desc set can hold this many textures.
	// NValsPer on a Texture var can be set higher and only this many will be
	// allocated in the descriptor set, with bindings of values wrapping
	// around across as many such sets as are vals, with a warning if insufficient
	// numbers are present.
	MaxTexturesPerSet = 16
)

// NDescForTextures returns number of descriptors (NDesc) required for
// given number of texture values.
func NDescForTextures(nvals int) int {
	nDescSetsReq := nvals / MaxTexturesPerSet
	if nvals%MaxTexturesPerSet > 0 {
		nDescSetsReq++
	}
	return nDescSetsReq
}

const (
	VertexSet = -2
	PushSet   = -1
)

// VarSet contains a set of Var variables that are all updated at the same time
// and have the same number of distinct Vals values per Var per render pass.
// The first set at index -1 contains Vertex and Index data, handed separately.
type VarSet struct {
	VarList
	Set           int                 `desc:"set number"`
	NValsPer      int                 `desc:"number of value instances to allocate per variable in this set: each value must be allocated in advance for each unique instance of a variable required across a complete scene rendering -- e.g., if this is an object position matrix, then one per object is required.  If a dynamic number are required, allocate the max possible.  For Texture vars, each of the NDesc sets can have a maximum of MaxTexturesPerSet (16) -- if NValsPer > MaxTexturesPerSet, then vals are wrapped across sets, and accessing them requires using the appropriate DescIdx, as in System.CmdBindTextureVarIdx."`
	NTextureDescs int                 `desc:"for texture vars, this is the number of descriptor sets required to represent all of the different Texture image Vals that have been allocated.  Use Vars.BindAllTextureVals to bind all such vals, and System.CmdBindTextureVarIdx to automatically bind the correct set."`
	RoleMap       map[VarRoles][]*Var `desc:"map of vars by different roles, within this set -- updated in Config(), after all vars added"`

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
func (st *VarSet) Config(dev vk.Device) error {
	st.RoleMap = make(map[VarRoles][]*Var)
	var cerr error
	bloc := 0
	for _, vr := range st.Vars {
		if st.Set == VertexSet && vr.Role > Index {
			err := fmt.Errorf("vgpu.VarSet:Config VertexSet cannot contain variables of role: %s  var: %s", vr.Role.String(), vr.Name)
			cerr = err
			if TheGPU.Debug {
				log.Println(err)
			}
			continue
		}
		if st.Set >= 0 && vr.Role <= Index {
			err := fmt.Errorf("vgpu.VarSet:Config Vertex or Index Vars must be located in a VertexSet!  Use AddVertexSet() method instead of AddSet()")
			cerr = err
			if TheGPU.Debug {
				log.Println(err)
			}
		}
		rl := st.RoleMap[vr.Role]
		rl = append(rl, vr)
		st.RoleMap[vr.Role] = rl
		if vr.Role == Index && len(rl) > 1 {
			err := fmt.Errorf("vgpu.VarSet:Config VertexSet should not contain multiple Index variables: %v", rl)
			cerr = err
			if TheGPU.Debug {
				log.Println(err)
			}
		}
		if vr.Role > Storage && (len(st.RoleMap[Uniform]) > 0 || len(st.RoleMap[Storage]) > 0) {
			err := fmt.Errorf("vgpu.VarSet:Config Set with dynamic Uniform or Storage variables should not contain static variables (e.g., textures): %s", vr.Role.String())
			cerr = err
			if TheGPU.Debug {
				log.Println(err)
			}
		}
		vr.BindLoc = bloc
		if vr.Role == TextureRole {
			vr.SetTextureDev(dev)
		}
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
	st.DestroyLayout(dev)
	var descLayout vk.DescriptorSetLayout
	var binds []vk.DescriptorSetLayoutBinding
	dyno := len(vs.DynOffs[0])
	var flags vk.DescriptorSetLayoutCreateFlags
	var dbf []vk.DescriptorBindingFlags
	nvar := len(st.Vars)
	nVarDesc := 0
	st.NTextureDescs = 1
	for vi, vr := range st.Vars {
		bd := vk.DescriptorSetLayoutBinding{
			Binding:         uint32(vr.BindLoc),
			DescriptorType:  RoleDescriptors[vr.Role],
			DescriptorCount: 1,
			StageFlags:      vk.ShaderStageFlags(vr.Shaders),
		}
		if vr.Role > Storage {
			nVarDesc = ints.MinInt(st.NValsPer, MaxTexturesPerSet) // per desc

			if st.NValsPer > MaxTexturesPerSet {
				st.NTextureDescs = NDescForTextures(st.NValsPer)
				if st.NTextureDescs > vs.NDescs {
					fmt.Printf("vgpu.VarSet: Texture %s NValsPer: %d requires NDescs = %d, but it is only: %d -- this probably won't end well, but can't be fixed here\n", vr.Name, st.NValsPer, st.NTextureDescs, vs.NDescs)
				}
			}
			bd.DescriptorCount = uint32(nVarDesc)
			dbfFlags := vk.DescriptorBindingPartiallyBoundBit
			//  | vk.DescriptorBindingUpdateAfterBindBit | vk.DescriptorBindingUpdateUnusedWhilePendingBit
			if vi == nvar-1 {
				dbfFlags |= vk.DescriptorBindingVariableDescriptorCountBit // can only be for last one
			}
			dbf = append(dbf, vk.DescriptorBindingFlags(dbfFlags))
			// flags = vk.DescriptorSetLayoutCreateFlags(vk.DescriptorSetLayoutCreateUpdateAfterBindPoolBit)
		}
		binds = append(binds, bd)
		if vr.Role == Uniform || vr.Role == Storage {
			vr.BindValIdx = make([]int, vs.NDescs)
			vr.DynOffIdx = dyno
			vs.AddDynOff()
			dyno++
		}
	}

	// https://www.khronos.org/registry/vulkan/specs/1.3-extensions/man/html/VkDescriptorSetLayoutBindingFlagsCreateInfo.html

	// PNext of following contains
	dslci := &vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: uint32(len(binds)),
		PBindings:    binds,
		Flags:        flags,
	}
	if len(dbf) > 0 {
		dslci.PNext = unsafe.Pointer(&vk.DescriptorSetLayoutBindingFlagsCreateInfo{
			SType:         vk.StructureTypeDescriptorSetLayoutBindingFlagsCreateInfo,
			PBindingFlags: dbf,
			BindingCount:  uint32(len(dbf)),
		})
	}
	ret := vk.CreateDescriptorSetLayout(dev, dslci, nil, &descLayout)
	IfPanic(NewError(ret))
	st.VkLayout = descLayout

	st.VkDescSets = make([]vk.DescriptorSet, vs.NDescs)
	for i := 0; i < vs.NDescs; i++ {
		dalloc := &vk.DescriptorSetAllocateInfo{
			SType:              vk.StructureTypeDescriptorSetAllocateInfo,
			DescriptorPool:     vs.VkDescPool,
			DescriptorSetCount: 1,
			PSetLayouts:        []vk.DescriptorSetLayout{st.VkLayout},
		}
		if nVarDesc > 0 {
			dalloc.PNext = unsafe.Pointer(&vk.DescriptorSetVariableDescriptorCountAllocateInfo{
				SType:              vk.StructureTypeDescriptorSetVariableDescriptorCountAllocateInfo,
				DescriptorSetCount: 1,
				PDescriptorCounts:  []uint32{uint32(nVarDesc)},
			})
		}
		var dset vk.DescriptorSet
		ret := vk.AllocateDescriptorSets(dev, dalloc, &dset)
		IfPanic(NewError(ret))
		st.VkDescSets[i] = dset
	}
}

// BindDynVars binds all dynamic vars in set, to be able to
// use dynamic vars, in subsequent BindDynVal* calls during the
// render pass, which update the offsets.
// For Uniform & Storage variables, which use dynamic binding.
//
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update actual values during a render pass.
// The memory buffer is essentially what is bound here.
//
// Must have called BindVarsStart prior to this.
func (st *VarSet) BindDynVars(vs *Vars) {
	for _, vr := range st.Vars {
		if vr.Role < Uniform || vr.Role > Storage {
			continue
		}
		st.BindDynVar(vs, vr)
	}
}

// BindDynVarName binds dynamic variable for given var
// looked up by name, for Uniform, Storage variables.
//
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update actual values during a render pass.
// The memory buffer is essentially what is bound here.
//
// Must have called BindVarsStart prior to this.
func (st *VarSet) BindDynVarName(vs *Vars, varNm string) error {
	vr, err := st.VarByNameTry(varNm)
	if err != nil {
		return err
	}
	st.BindDynVar(vs, vr)
	return nil
}

// BindDynVar binds dynamic variable for given var
// for Uniform, Storage variables.
//
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update actual values during a render pass.
// The memory buffer is essentially what is bound here.
//
// Must have called BindVarsStart prior to this.
func (st *VarSet) BindDynVar(vs *Vars, vr *Var) error {
	if vr.Role < Uniform || vr.Role > Storage {
		err := fmt.Errorf("vgpu.Set:BindDynVar dynamic binding only valid for Uniform or Storage Vars, not: %s", vr.Role.String())
		if TheGPU.Debug {
			log.Println(err)
		}
		return err
	}
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
		Range:  vk.DeviceSize(vr.MemSize()),
		Buffer: buff.Dev,
	}}
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
		if vr.Role <= Storage {
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
		SType:          vk.StructureTypeWriteDescriptorSet,
		DstSet:         st.VkDescSets[vs.BindDescIdx],
		DstBinding:     uint32(vr.BindLoc),
		DescriptorType: vr.Role.VkDescriptor(),
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
		wd.DescriptorCount = uint32(nvals)
	} else {
		imgs := []vk.DescriptorImageInfo{}
		nvals := len(vr.Vals.Vals)
		if nvals > MaxTexturesPerSet {
			sti := vs.BindDescIdx * MaxTexturesPerSet
			if sti > nvals-MaxTexturesPerSet {
				sti = nvals - MaxTexturesPerSet
			}
			mx := sti + MaxTexturesPerSet
			for vi := sti; vi < mx; vi++ {
				vl := vr.Vals.Vals[vi]
				if vl.Texture != nil && vl.Texture.IsActive() {
					di := vk.DescriptorImageInfo{
						ImageLayout: vk.ImageLayoutShaderReadOnlyOptimal,
						ImageView:   vl.Texture.View,
						Sampler:     vl.Texture.VkSampler,
					}
					imgs = append(imgs, di)
				}
			}

		} else {
			for _, vl := range vr.Vals.Vals {
				if vl.Texture != nil && vl.Texture.IsActive() {
					di := vk.DescriptorImageInfo{
						ImageLayout: vk.ImageLayoutShaderReadOnlyOptimal,
						ImageView:   vl.Texture.View,
						Sampler:     vl.Texture.VkSampler,
					}
					imgs = append(imgs, di)
				}
			}
		}
		wd.PImageInfo = imgs
		wd.DescriptorCount = uint32(len(imgs))
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

// VkPushConfig returns vulkan push constant ranges
func (vs *VarSet) VkPushConfig() []vk.PushConstantRange {
	alignBytes := 8 // unclear what alignment is
	var ranges []vk.PushConstantRange
	offset := 0
	tsz := 0
	for _, vr := range vs.Vars {
		vr.Offset = offset
		sz := vr.SizeOf
		rg := vk.PushConstantRange{
			Offset:     uint32(offset),
			Size:       uint32(sz),
			StageFlags: vk.ShaderStageFlags(vr.Shaders),
		}
		esz := MemSizeAlign(sz, alignBytes)
		offset += esz
		tsz += esz
		ranges = append(ranges, rg)
	}
	if tsz > 128 {
		if TheGPU.Debug {
			fmt.Printf("vgpu.VarSet:VkPushConfig total push constant memory exceeds nominal minimum size of 128 bytes: %d\n", tsz)
		}
	}
	return ranges
}

/////////////////////////////////////////////////////////////////////////
// Dynamic Binding

// BindDynValName dynamically binds given uniform or storage value
// by name for given variable name, in given set.
//
// This only dynamically updates the offset to point to the specified val.
// MUST call System.BindVars prior to any subsequent draw calls for this
// new offset to be bound at the proper point in the command buffer prior
// (call after all such dynamic bindings are updated.)
//
// Do NOT call BindValsStart / End around this.
//
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
// by index for given variable name, in given set.
//
// This only dynamically updates the offset to point to the specified val.
// MUST call System.BindVars prior to any subsequent draw calls for this
// new offset to be bound at the proper point in the command buffer prior
// (call after all such dynamic bindings are updated.)
//
// Do NOT call BindValsStart / End around this.
//
// returns error if not found.
func (st *VarSet) BindDynValIdx(vs *Vars, varNm string, valIdx int) error {
	vr, vl, err := st.ValByIdxTry(varNm, valIdx)
	if err != nil {
		return err
	}
	return st.BindDynVal(vs, vr, vl)
}

// BindDynVal dynamically binds given uniform or storage value
// for given variable in given set.
//
// This only dynamically updates the offset to point to the specified val.
// MUST call System.BindVars prior to any subsequent draw calls for this
// new offset to be bound at the proper point in the command buffer prior
// (call after all such dynamic bindings are updated.)
//
// Do NOT call BindValsStart / End around this.
//
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
	vs.DynOffs[vs.BindDescIdx][vr.DynOffIdx] = uint32(vl.Offset)
	return nil
}
