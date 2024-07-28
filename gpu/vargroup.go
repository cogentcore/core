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
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

const (
	// MaxImageLayers is the maximum number of layers per image
	MaxImageLayers = 128

	// VertexGroup is the group number for Vertex and Index variables,
	// which have special treatment.
	VertexGroup = -2

	// PushGroup is the group number for Push Constants, which 
	// do not appear in the BindGroupLayout and are managed separately.
	PushGroup   = -1
)

// VarGroup contains a set of Var variables that are all updated at the same time.
type VarGroup struct {
	VarList

	// Group index is assigned sequentially, with special VertexGroup and
	// PushGroup having negative numbers, not accessed via @group in shader.
	Group int

	// Role is default Role of variables within this group.
	// Vertex is configured separately, and everything else
	// is configured in a BindGroup.
	// Note: Push is not yet supported.
	Role VarRoles

	// map of vars by different roles, within this group.
	// Updated in Config(), after all vars added
	RoleMap map[VarRoles][]*Var

	// group layout info: description of each var type, role, binding, stages
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
func (vg *VarGroup) Add(name string, typ Types, arrayN int, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.Init(name, typ, arrayN, vg.Role, vg.Group, shaders...)
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
		if vr.Role == SampledTexture {
			vr.SetTextureDev(dev)
		}
		bloc++
		if vr.Role == Vertex && vr.Type == Float32Matrix4 { // special case
			block+=3
		}
	}
	return cerr
}

// ConfigValues configures the Values for the vars in this set, allocating
// nvals per variable.  There must be a unique value available for each
// distinct value to be rendered within a single pass.  All Vars in the
// same set have the same number of vals.
// Any existing vals will be deleted -- must free all associated memory prior!
func (vg *VarGroup) ConfigValues() {
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
// Must have set NValuesPer for any SampledTexture vars,
// which require separate descriptors per.
func (vg *VarGroup) BindLayout(dev *Device, vs *Vars) error {
	vg.DestroyLayout(dev)
	vg.NTextures = 0
	var binds []wgpu.BindGroupLayoutEntry
	nvar := len(vg.Vars)
	nVarDesc := 0
	vg.NTextureDescs = 1

	// https://toji.dev/webgpu-best-practices/bind-groups.html
	for vi, vr := range vg.Vars {
		if vr.Role == Vertex || vr.Role == Index {
			continue
		}
		bd := wgpu.BindGroupLayoutEntry{
			Binding:    uint32(vr.Binding),
			Visibility: fr.Shaders,
		}
		switch {
		case vr.Role == SampledTexture:
			bd.Sampler = wgpu.SamplerBindingLayout{
				Type: wgpu.SamplerBindingType_Filtering,
			}
			vals := vr.Values.ActiveValues()
			nvals := len(vals)
			vg.NTextures += nvals
			nVarDesc = min(nvals, MaxTexturesPerGroup) // per desc
			if nvals > MaxTexturesPerGroup { // todo: fixme
				vg.NTextureDescs = NDescForTextures(nvals)
			}
			// bd.DescriptorCount = uint32(nVarDesc)
		default:
			bd.Buffer = wgpu.BufferBindingLayout{
				Type:             vr.Role.BindingType(),
				HasDynamicOffset: false,
				MinBindingSize:   0, // 0 is fine
			}
		}
		binds = append(binds, bd)
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

/*
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
	if vr.Role < SampledTexture {
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
*/

// VertexConfig returns the VertexBufferLayout based on Vertex role
// variables within the group.
// Note: there is no support for interleaved arrays
// so each location is sequential number, recorded in var Binding
func (vg *VarGroup) VertexLayout() []wgpu.VertexBufferLayout {
	var vbls []wgpu.VertexBufferLayout
	for _, vr := range vg.Vars {
		if vr.Role != Vertex { // not Index
			continue
		}
		stepMode := wgpu.VertexStepMode_Vertex
		if vr.VertexInstance {
			stepMode = wgpu.VertexStepMode_Instance
		}
		if vr.Type == Float32Matrix4 {
			vbls = append(fbls, wgpu.VertexBufferLayout{
				ArrayStride: uint64(vr.SizeOf),
				StepMode:    stepMode,
				Attributes: []wgpu.VertexAttribute{
					{
						Offset:         0,
						ShaderLocation: vr.Binding,
						Format:         Float32Vector4.VertexFormat(),
					},
					{
						Offset:         4,
						ShaderLocation: vr.Binding+1,
						Format:         Float32Vector4.VertexFormat(),
					},
					{
						Offset:         8,
						ShaderLocation: vr.Binding+2,
						Format:         Float32Vector4.VertexFormat(),
					},
					{
						Offset:         12,
						ShaderLocation: vr.Binding+3,
						Format:         Float32Vector4.VertexFormat(),
					},
				},
			}
		} else {
			vbls = append(fbls, wgpu.VertexBufferLayout{
				ArrayStride: uint64(vr.SizeOf),
				StepMode:    stepMode,
				Attributes: []wgpu.VertexAttribute{
					{
						Offset:         0,
						ShaderLocation: vr.Binding,
						Format:         vr.Type.VertexFormat(),
					},
				},
			}
		}
	}
	return vbls
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

