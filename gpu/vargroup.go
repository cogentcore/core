// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"log"
	"log/slog"
	"strconv"

	"cogentcore.org/core/vgpu/szalloc"
	vk "github.com/goki/WebGPU"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// maxPerStageDescriptorSamplers is only 16 on mac -- this is the relevant limit on textures!
// also maxPerStageDescriptorSampledImages is basically the same:
// https://WebGPU.gpuinfo.org/displaydevicelimit.php?name=maxPerStageDescriptorSamplers&platform=all
// https://WebGPU.gpuinfo.org/displaydevicelimit.php?name=maxPerStageDescriptorSampledImages&platform=all

const (
	// MaxTexturesPerGroup is the maximum number of image variables that can be used
	// in one descriptor set.  This value is a lowest common denominator across
	// platforms.  To overcome this limitation, when more Texture vals are allocated,
	// multiple NDescs are used, setting the and switch
	// across those -- each such Desc set can hold this many textures.
	// NValuesPer on a Texture var can be set higher and only this many will be
	// allocated in the descriptor set, with bindings of values wrapping
	// around across as many such sets as are vals, with a warning if insufficient
	// numbers are present.
	MaxTexturesPerGroup = 16

	// MaxImageLayers is the maximum number of layers per image
	MaxImageLayers = 128
)

// NDescForTextures returns number of descriptors (NDesc) required for
// given number of texture values.
func NDescForTextures(nvals int) int {
	nDescGroupsReq := nvals / MaxTexturesPerGroup
	if nvals%MaxTexturesPerGroup > 0 {
		nDescGroupsReq++
	}
	return nDescGroupsReq
}

const (
	VertexGroup = -2
	PushGroup   = -1
)

// VarGroup contains a set of Var variables that are all updated at the same time
// and have the same number of distinct Values values per Var per render pass.
// The first set at index -1 contains Vertex and Index data, handed separately.
type VarGroup struct {
	VarList

	// set number
	Group int

	// number of value instances to allocate per variable in this Group.
	// Each value must be allocated in advance for each unique instance
	// of a variable required across a complete scene rendering.
	// e.g., if this is an object position matrix, then one per object is required.
	// If a dynamic number are required, allocate the max possible.
	// For Texture vars, each of the NDesc sets can have a maximum of
	// MaxTexturesPerGroup (16), so if NValuesPer > MaxTexturesPerGroup,
	// then vals are wrapped across sets, and accessing them requires using the
	// appropriate DescIndex, as in System.CmdBindTextureVarIndex.
	NValuesPer int

	// number of textures, at point of creating the DescLayout
	NTextures int

	// map of vars by different roles, within this set.
	// Updated in Config(), after all vars added
	RoleMap map[VarRoles][]*Var

	// the parent vars we belong to
	ParentVars *Vars

	// set layout info: description of each var type, role, binding, stages
	Layout *wgpu.BindGroupLayout
}

// AddVar adds given variable
func (vg *VarGroup) AddVar(vr *Var) {
	if vg.VarMap == nil {
		vg.VarMap = make(map[string]*Var)
	}
	vg.Vars = append(vg.Vars, vr)
	vg.VarMap[vr.Name] = vr
}

// Add adds a new variable of given type, role, arrayN, and shaders where used
func (vg *VarGroup) Add(name string, typ Types, arrayN int, role VarRoles, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.Init(name, typ, arrayN, role, vg.Group, shaders...)
	vg.AddVar(vr)
	return vr
}

// AddStruct adds a new struct variable of given total number of bytes in size,
// type, role, set, and shaders where used
func (vg *VarGroup) AddStruct(name string, size int, arrayN int, role VarRoles, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.Init(name, Struct, arrayN, role, vg.Group, shaders...)
	vr.SizeOf = size
	vg.AddVar(vr)
	return vr
}

// Config must be called after all variables have been added.
// configures binding / location for all vars based on sequential order.
// also does validation and returns error message.
func (vg *VarGroup) Config(dev *Device) error {
	vg.RoleMap = make(map[VarRoles][]*Var)
	var cerr error
	bloc := 0
	for _, vr := range vg.Vars {
		if vg.Group == VertexGroup && vr.Role > Index {
			err := fmt.Errorf("gpu.VarGroup:Config VertexGroup cannot contain variables of role: %s  var: %s", vr.Role.String(), vr.Name)
			cerr = err
			if Debug {
				log.Println(err)
			}
			continue
		}
		if vg.Group >= 0 && vr.Role <= Index {
			err := fmt.Errorf("gpu.VarGroup:Config Vertex or Index Vars must be located in a VertexGroup!  Use AddVertexGroup() method instead of AddGroup()")
			cerr = err
			if Debug {
				log.Println(err)
			}
		}
		rl := vg.RoleMap[vr.Role]
		rl = append(rl, vr)
		vg.RoleMap[vr.Role] = rl
		if vr.Role == Index && len(rl) > 1 {
			err := fmt.Errorf("gpu.VarGroup:Config VertexGroup should not contain multiple Index variables: %v", rl)
			cerr = err
			if Debug {
				log.Println(err)
			}
		}
		if vr.Role > Storage && (len(vg.RoleMap[Uniform]) > 0 || len(vg.RoleMap[Storage]) > 0) {
			err := fmt.Errorf("gpu.VarGroup:Config Group with dynamic Uniform or Storage variables should not contain static variables (e.g., textures): %s", vr.Role.String())
			cerr = err
			if Debug {
				log.Println(err)
			}
		}
		vr.Binding = bloc
		if vr.Role == TextureRole {
			vr.SetTextureDev(dev)
		}
		bloc++
	}
	return cerr
}

// ConfigValues configures the Values for the vars in this set, allocating
// nvals per variable.  There must be a unique value available for each
// distinct value to be rendered within a single pass.  All Vars in the
// same set have the same number of vals.
// Any existing vals will be deleted -- must free all associated memory prior!
func (vg *VarGroup) ConfigValues(nvals int) {
	dev := vg.ParentVars.Mem.Device.Device
	gp := vg.ParentVars.Mem.GPU
	vg.NValuesPer = nvals
	for _, vr := range vg.Vars {
		vr.Values.ConfigValues(gp, dev, vr, nvals)
	}
}

// Destroy destroys infrastructure for Group, Vars and Values -- assumes Free has
// already been called to free host and device memory.
func (vg *VarGroup) Destroy(dev *Device) {
	vg.DestroyLayout()
}

// DestroyLayout destroys layout
func (vg *VarGroup) DestroyLayout() {
	if vg.Layout != nil {
		vg.Layout.Release()
		vg.Layout = nil
	}
}

// BindLayout creates the BindGroupLayout for given set.
// Only for non-VertexGroup sets.
// Must have set NValuesPer for any TextureRole vars,
// which require separate descriptors per.
func (vg *VarGroup) BindLayout(dev *Device, vs *Vars) error {
	vg.DestroyLayout(dev)
	vg.NTextures = 0
	var binds []wgpu.BindGroupLayoutEntry
	dyno := len(vs.DynOffs[0])
	nvar := len(vg.Vars)
	nVarDesc := 0
	vg.NTextureDescs = 1

	// https://toji.dev/webgpu-best-practices/bind-groups.html
	for vi, vr := range vg.Vars {
		bd := wgpu.BindGroupLayoutEntry{
			Binding:    uint32(vr.Binding),
			Visibility: fr.Shaders,
		}
		if vr.Role <= StorageImage {
			bd.Buffer = wgpu.BufferBindingLayout{
				Type:             vr.Role.BindingType(),
				HasDynamicOffset: vr.Role.IsDynamic(),
				MinBindingSize:   0, // 0 is fine
			}
		} else if vr.Role == TextureRole {
			bd.Sampler = wgpu.SamplerBindingLayout{
				Type: wgpu.SamplerBindingType_Filtering,
			}
			vals := vr.Values.ActiveValues()
			nvals := len(vals)
			vg.NTextures += nvals
			nVarDesc = min(nvals, MaxTexturesPerGroup) // per desc

			if nvals > MaxTexturesPerGroup {
				vg.NTextureDescs = NDescForTextures(nvals)
				if vg.NTextureDescs > vs.NDescs {
					fmt.Printf("gpu.VarGroup: Texture %s NValues: %d requires NDescs = %d, but it is only: %d -- this probably won't end well, but can't be fixed here\n", vr.Name, nvals, vg.NTextureDescs, vs.NDescs)
				}
			}
			bd.DescriptorCount = uint32(nVarDesc)
		}
		binds = append(binds, bd)
		if !vs.StaticVars {
			if vr.Role == Uniform || vr.Role == Storage {
				vr.BindValueIndex = make([]int, vs.NDescs)
				vr.DynOffIndex = dyno
				vs.AddDynOff()
				dyno++
			}
		}
	}

	bgld := BindGroupLayoutDescriptor{
		Label:   strconv.Itoa(vs.Group),
		Entries: binds,
	}

	bgl, err := dev.Device.CreateBindGroupLayout(&bgld)
	if err != nil {
		slog.Error(err)
		return err
	}
	vg.Layout = bgl
}

// todo: other static cases need same approach as images!
// also, need an option to allow a single val to be used in a static way, selecting from among multiple,
// instead of always assuming an array used.

// BindStatVarsAll statically binds all uniform, storage values
// in given set, for all variables, for all values.
//
// Must call BindVarStart / End around this.
func (vg *VarGroup) BindStatVarsAll(vs *Vars) {
	for _, vr := range vg.Vars {
		if vr.Role < Uniform || vr.Role > Storage {
			continue
		}
		vg.BindStatVar(vs, vr)
	}
}

// BindStatVars binds all static vars to their current values,
// for non-Uniform, Storage, variables (e.g., Textures).
// Each Value for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
func (vg *VarGroup) BindStatVars(vs *Vars) {
	for _, vr := range vg.Vars {
		if vr.Role <= Storage {
			continue
		}
		vg.BindStatVar(vs, vr)
	}
}

// BindStatVarName does static variable binding for given var
// looked up by name, For non-Uniform, Storage, variables (e.g., Textures).
// Each Value for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
func (vg *VarGroup) BindStatVarName(vs *Vars, varNm string) error {
	vr, err := vg.VarByNameTry(varNm)
	if err != nil {
		return err
	}
	vg.BindStatVar(vs, vr)
	return nil
}

// BindStatVar does static variable binding for given var,
// Each Value for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
func (vg *VarGroup) BindStatVar(vs *Vars, vr *Var) {

	vals := vr.Values.ActiveValues()
	nvals := len(vals)
	wd := vk.WriteDescriptorSet{
		SType:          vk.StructureTypeWriteDescriptorSet,
		DstSet:         vg.VkDescSets[vs.BindDescIndex],
		DstBinding:     uint32(vr.Binding),
		DescriptorType: vr.Role.VkDescriptorStatic(),
	}
	bt := vr.BuffType()
	buff := vs.Mem.Buffs[bt]
	if bt == StorageBuffer {
		buff = vs.Mem.StorageBuffers[vr.StorageBuffer]
	}
	if vr.Role < TextureRole {
		bis := make([]vk.DescriptorBufferInfo, nvals)
		for i, vl := range vals {
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
		nvals := len(vals)
		if nvals > MaxTexturesPerGroup {
			sti := vs.BindDescIndex * MaxTexturesPerGroup
			if sti > nvals-MaxTexturesPerGroup {
				sti = nvals - MaxTexturesPerGroup
			}
			mx := sti + MaxTexturesPerGroup
			for vi := sti; vi < mx; vi++ {
				vl := vals[vi]
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
			for _, vl := range vals {
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
		if len(imgs) == 0 {
			return // don't add
		}
		wd.PImageInfo = imgs
		wd.DescriptorCount = uint32(len(imgs))
	}
	vs.VkWriteValues = append(vs.VkWriteValues, wd)
}

// VkVertexConfig fills in the relevant info into given WebGPU config struct.
// for VertexSet only!
// Note: there is no support for interleaved arrays so each binding and location
// is assigned the same sequential number, recorded in var Binding
func (vg *VarGroup) VkVertexConfig() *vk.PipelineVertexInputStateCreateInfo {
	cfg := &vk.PipelineVertexInputStateCreateInfo{}
	cfg.SType = vk.StructureTypePipelineVertexInputStateCreateInfo
	var bind []vk.VertexInputBindingDescription
	var attr []vk.VertexInputAttributeDescription
	for _, vr := range vg.Vars {
		if vr.Role != Vertex { // not Index
			continue
		}
		bind = append(bind, vk.VertexInputBindingDescription{
			Binding:   uint32(vr.Binding),
			Stride:    uint32(vr.SizeOf),
			InputRate: vk.VertexInputRateVertex,
		})
		attr = append(attr, vk.VertexInputAttributeDescription{
			Location: uint32(vr.Binding),
			Binding:  uint32(vr.Binding),
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

/*
// VkPushConfig returns WebGPU push constant ranges
func (vs *VarGroup) VkPushConfig() []vk.PushConstantRange {
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
		if Debug {
			fmt.Printf("gpu.VarGroup:VkPushConfig total push constant memory exceeds nominal minimum size of 128 bytes: %d\n", tsz)
		}
	}
	return ranges
}
*/

// TextureGroupSizeIndexes for texture at given index, allocated in groups by size
// using Values.AllocTexBySize, returns the indexes for the texture
// and layer to actually select the texture in the shader, and proportion
// of the Gp allocated texture size occupied by the texture.
func (vg *VarGroup) TextureGroupSizeIndexes(vs *Vars, varNm string, valIndex int) *szalloc.Indexes {
	vr, err := vg.VarByNameTry(varNm)
	if err != nil {
		log.Println(err)
		return nil
	}
	idxs := vr.Values.TexSzAlloc.ItemIndexes[valIndex]
	return idxs
}
