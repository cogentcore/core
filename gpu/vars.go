// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"log"
	"strings"

	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/vgpu/szalloc"

	vk "github.com/goki/WebGPU"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Vars are all the variables that are used by a pipeline,
// organized into Groups (optionally including the special VertexGroup
// or PushGroup).
// Vars are allocated to bindings sequentially in the order added.
type Vars struct {

	// map of Groups, by group number: VertexGroup is -2, PushGroup is -1,
	// rest are added incrementally.
	GroupMap map[int]*VarGroup

	// map of vars by different roles across all Groups, updated in Config(),
	// after all vars added.
	RoleMap map[VarRoles][]*Var

	// true if a VertexGroup has been added
	HasVertex bool `edit:"-"`

	// true if PushGroup has been added.  Note: not yet supported in WebGPU.
	HasPush bool `edit:"-"`

	// our parent memory manager
	Mem *Memory `display:"-"`

	// if true, variables are statically bound to specific offsets in memory buffers, vs. dynamically bound offsets.  Typically a compute shader operating on fixed data variables can use static binding, while graphics (e.g., vphong) requires dynamic binding to efficiently use the same shader code for multiple different values of the same variable type
	StaticVars bool `edit:"-"`

	// full set of BindGroupLayouts, one for each VarGroup
	Layouts []*wgpu.BindGroupLayout `display:"-"`

	// currently accumulating set of vals to write to update bindings -- initiated by BindValuesStart, executed by BindValuesEnd
	VkWriteValues []vk.WriteDescriptorSet `display:"-"`

	// current descriptor collection index, set in BindValuesStart
	BindDescIndex int `edit:"-"`

	// dynamic offsets for Uniform and Storage variables, -- outer index is Vars.NDescs for different groups of descriptor sets, one of which can be bound to a pipeline at any given time, inner index is DynOffIndex on Var -- offsets are set when Value is bound via BindDynValue*.
	DynOffs [][]uint32

	// number of textures, at point of creating the DescLayout
	NTextures int
}

func (vs *Vars) Destroy(dev *Device) {
	vk.DestroyPipelineLayout(dev, vs.VkDescLayout, nil)
	vk.DestroyDescriptorPool(dev, vs.VkDescPool, nil)
	for _, st := range vs.GroupMap {
		st.Destroy(dev)
	}
}

// AddVertexGroup adds a new Vertex Group -- this is a special Group holding Vertex, Index vars
func (vs *Vars) AddVertexGroup() *VarGroup {
	if vs.GroupMap == nil {
		vs.GroupMap = make(map[int]*VarGroup)
	}
	st := &VarGroup{Group: VertexGroup, ParentVars: vs}
	vs.GroupMap[VertexGroup] = st
	vs.HasVertex = true
	return st
}

// VertexGroup returns the Vertex Group -- a special Group holding Vertex, Index vars
func (vs *Vars) VertexGroup() *VarGroup {
	return vs.GroupMap[VertexGroup]
}

// AddPushGroup adds a new push constant Group -- this is a special Group holding
// values sent directly in the command buffer.
func (vs *Vars) AddPushGroup() *VarGroup {
	if vs.GroupMap == nil {
		vs.GroupMap = make(map[int]*VarGroup)
	}
	st := &VarGroup{Group: PushGroup, ParentVars: vs}
	vs.GroupMap[PushGroup] = st
	vs.HasPush = true
	return st
}

// PushGroup returns the Push Group -- a special Group holding push constants
func (vs *Vars) PushGroup() *VarGroup {
	return vs.GroupMap[PushGroup]
}

// AddGroup adds a new non-Vertex Group for holding Uniforms, Storage, etc
// Groups are automatically numbered sequentially
func (vs *Vars) AddGroup() *VarGroup {
	if vs.GroupMap == nil {
		vs.GroupMap = make(map[int]*VarGroup)
	}
	idx := vs.NGroups()
	st := &VarGroup{Group: idx, ParentVars: vs}
	vs.GroupMap[idx] = st
	return st
}

// VarByNameTry returns Var by name in given set number,
// returning error if not found
func (vs *Vars) VarByNameTry(set int, name string) (*Var, error) {
	st, err := vs.GroupTry(set)
	if err != nil {
		return nil, err
	}
	return st.VarByNameTry(name)
}

// ValueByNameTry returns value by first looking up variable name, then value name,
// within given set number, returning error if not found
func (vs *Vars) ValueByNameTry(set int, varName, valName string) (*Var, *Value, error) {
	st, err := vs.GroupTry(set)
	if err != nil {
		return nil, nil, err
	}
	return st.ValueByNameTry(varName, valName)
}

// ValueByIndexTry returns value by first looking up variable name, then value index,
// returning error if not found
func (vs *Vars) ValueByIndexTry(set int, varName string, valIndex int) (*Var, *Value, error) {
	st, err := vs.GroupTry(set)
	if err != nil {
		return nil, nil, err
	}
	return st.ValueByIndexTry(varName, valIndex)
}

// Config must be called after all variables have been added.
// Configures all Groups and also does validation, returning error
// does DescLayout too, so all ready for Pipeline config.
func (vs *Vars) Config() error {
	dev := vs.Mem.Device.Device
	ns := vs.NGroups()
	var cerr error
	vs.RoleMap = make(map[VarRoles][]*Var)
	for si := vs.StartGroup(); si < ns; si++ {
		st := vs.GroupMap[si]
		if st == nil {
			continue
		}
		err := st.Config(dev)
		if err != nil {
			cerr = err
		}
		for ri, rl := range st.RoleMap {
			vs.RoleMap[ri] = append(vs.RoleMap[ri], rl...)
		}
	}
	vs.BindLayout(dev)
	return cerr
}

// StringDoc returns info on variables
func (vs *Vars) StringDoc() string {
	ispc := 4
	var sb strings.Builder
	ns := vs.NGroups()
	for si := vs.StartGroup(); si < ns; si++ {
		st := vs.GroupMap[si]
		if st == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("Group: %d\n", st.Group))

		for ri := Vertex; ri < VarRolesN; ri++ {
			rl, has := st.RoleMap[ri]
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

// NGroups returns the number of regular non-VertexGroup sets
func (vs *Vars) NGroups() int {
	ex := 0
	if vs.HasVertex {
		ex++
	}
	if vs.HasPush {
		ex++
	}
	return len(vs.GroupMap) - ex
}

// StartGroup returns the starting set to use for iterating sets
func (vs *Vars) StartGroup() int {
	switch {
	case vs.HasVertex:
		return VertexGroup
	case vs.HasPush:
		return PushGroup
	default:
		return 0
	}
}

// GroupTry returns set by index, returning nil and error if not found
func (vs *Vars) GroupTry(set int) (*VarGroup, error) {
	st, has := vs.GroupMap[set]
	if !has {
		err := fmt.Errorf("gpu.Vars:GroupTry set number %d not found", set)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return st, nil
}

// VkVertexConfig returns WebGPU vertex config struct, for VertexGroup only!
// Note: there is no support for interleaved arrays so each binding and location
// is assigned the same sequential number, recorded in var Binding
func (vs *Vars) VkVertexConfig() *vk.PipelineVertexInputStateCreateInfo {
	if vs.HasVertex {
		return vs.GroupMap[VertexGroup].VkVertexConfig()
	}
	cfg := &vk.PipelineVertexInputStateCreateInfo{}
	cfg.SType = vk.StructureTypePipelineVertexInputStateCreateInfo
	return cfg
}

// VkPushConfig returns WebGPU push constant ranges, only if PushGroup used.
func (vs *Vars) VkPushConfig() []vk.PushConstantRange {
	if vs.HasPush {
		return vs.GroupMap[PushGroup].VkPushConfig()
	}
	return nil
}

///////////////////////////////////////////////////////////////////
// Descriptors for Uniforms etc

// DescLayout configures the PipelineLayout of DescriptorGroupLayout
// info for all of the non-Vertex vars
func (vs *Vars) BindLayout(dev *Device) {
	vs.NTextures = 0
	if vs.NDescs < 1 {
		vs.NDescs = 1
	}
	nset := vs.NGroups()
	if nset == 0 {
		var pipelineLayout vk.PipelineLayout
		ret := vk.CreatePipelineLayout(dev, &vk.PipelineLayoutCreateInfo{
			SType:          vk.StructureTypePipelineLayoutCreateInfo,
			SetLayoutCount: 0,
			PSetLayouts:    nil,
		}, nil, &pipelineLayout)
		IfPanic(NewError(ret))
		vs.VkDescLayout = pipelineLayout
		vs.VkDescPool = vk.NullDescriptorPool
		vs.VkDescSets = nil
		return
	}

	var pools []vk.DescriptorPoolSize
	for ri := Uniform; ri < VarRolesN; ri++ {
		vl := vs.RoleMap[ri]
		if len(vl) == 0 {
			continue
		}
		dcount := vs.NDescs * len(vl)
		if ri > Storage {
			dcount = 0
			for _, vr := range vl {
				vals := vr.Values.ActiveValues()
				dcount += vs.NDescs * len(vals)
			}
		}
		if dcount == 0 {
			continue
		}
	}

	vs.DynOffs = make([][]uint32, vs.NDescs)
	dlays := make([]vk.DescriptorSetLayout, nset)
	for si := 0; si < nset; si++ {
		st := vs.GroupMap[si]
		if st == nil {
			continue
		}
		st.BindLayout(dev, vs)
		vs.NTextures += st.NTextures
		dlays[si] = st.VkLayout
	}

	if vs.HasVertex {
		vset := vs.GroupMap[VertexGroup]
		for _, vr := range vset.Vars {
			vr.BindValueIndex = make([]int, vs.NDescs)
		}
	}

	dsets := make([][]vk.DescriptorSet, vs.NDescs)
	for i := 0; i < vs.NDescs; i++ {
		dsets[i] = make([]vk.DescriptorSet, nset)
		for si := 0; si < nset; si++ {
			st := vs.GroupMap[si]
			if st == nil {
				continue
			}
			dsets[i][si] = st.VkDescSets[i]
		}
	}
	vs.VkDescSets = dsets // for pipeline binding

	pushc := vs.VkPushConfig()

	var pipelineLayout vk.PipelineLayout
	ret = vk.CreatePipelineLayout(dev, &vk.PipelineLayoutCreateInfo{
		SType:                  vk.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount:         uint32(len(dlays)),
		PSetLayouts:            dlays,
		PPushConstantRanges:    pushc,
		PushConstantRangeCount: uint32(len(pushc)),
	}, nil, &pipelineLayout)
	IfPanic(NewError(ret))
	vs.VkDescLayout = pipelineLayout
}

// AddDynOff adds one more dynamic offset across all NDescs
func (vs *Vars) AddDynOff() {
	for i := 0; i < vs.NDescs; i++ {
		vs.DynOffs[i] = append(vs.DynOffs[i], 0)
	}
}

/////////////////////////////////////////////////////////////////////////
// Bind Vars -- call prior to render pass only!!

// BindVarsStart starts a new step of binding vars to descriptor sets,
// using given descIndex description set index (among the NDescs allocated).
// Bound vars determine what the shader programs see,
// in subsequent calls to Pipeline commands.
//
// This must be called *prior* to a render pass, never within it.
// Only BindDyn* and BindVertex* calls can be called within render.
//
// Do NOT use this around BindDynValue or BindVertexValue calls
// only for BindVar* methods.
//
// Subsequent calls of BindVar* methods will add to a list, which
// will be executed when BindValuesEnd is called.
//
// This creates a set of entries in a list of WriteDescriptorSet's
func (vs *Vars) BindVarsStart(descIndex int) {
	vs.VkWriteValues = []vk.WriteDescriptorSet{}
	vs.BindDescIndex = descIndex
}

// BindVarsEnd finishes a new step of binding started by BindVarsStart.
// Actually executes the binding updates, based on prior BindVar* calls.
func (vs *Vars) BindVarsEnd() {
	dev := vs.Mem.Device.Device
	if len(vs.VkWriteValues) > 0 {
		vk.UpdateDescriptorSets(dev, uint32(len(vs.VkWriteValues)), vs.VkWriteValues, 0, nil)
	}
	vs.VkWriteValues = nil
}

// BindDynVars binds all dynamic vars in given set, to be able to
// use dynamic vars, in subsequent BindDynValue* calls during the
// render pass, which update the offsets.
// For Uniform & Storage variables, which use dynamic binding.
//
// This is automatically called during Config on the System,
// and usually does not need to be called again.
//
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update actual values during a render pass.
// The memory buffer is essentially what is bound here.
//
// Must have called BindVarsStart prior to this.
func (vs *Vars) BindDynVars(set int) error {
	st, err := vs.GroupTry(set)
	if err != nil {
		return err
	}
	st.BindDynVars(vs)
	return nil
}

// BindDynVarsAll binds all dynamic vars across all sets.
// Called during system config.
func (vs *Vars) BindDynVarsAll() {
	nset := vs.NGroups()
	for i := 0; i < vs.NDescs; i++ {
		vs.BindVarsStart(i)
		for si := 0; si < nset; si++ {
			vs.BindDynVars(si)
		}
		vs.BindVarsEnd()
	}
}

// BindStatVarsAll binds all static vars to their current values,
// for all Uniform and Storage variables, when using Static value binding.
//
// All vals must be uploaded to Device memory prior to this.
func (vs *Vars) BindStatVarsAll() {
	for i := 0; i < vs.NDescs; i++ {
		vs.BindVarsStart(i)
		for si, st := range vs.GroupMap {
			if si < 0 {
				continue
			}
			st.BindStatVarsAll(vs)
		}
		vs.BindVarsEnd()
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
func (vs *Vars) BindDynVarName(set int, varNm string) error {
	st, err := vs.GroupTry(set)
	if err != nil {
		return err
	}
	return st.BindDynVarName(vs, varNm)
}

// BindStatVars binds all static vars to their current values,
// for given set, for non-Uniform, Storage, variables (e.g., Textures).
// Each Value for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
//
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
//
// Must have called BindVarsStart prior to this.
func (vs *Vars) BindStatVars(set int) error {
	st, err := vs.GroupTry(set)
	if err != nil {
		return err
	}
	st.BindStatVars(vs)
	return nil
}

// BindStatVarName does static variable binding for given var
// looked up by name, for non-Uniform, Storage, variables (e.g., Textures).
// Each Value for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
//
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
//
// Must have called BindVarsStart prior to this.
func (vs *Vars) BindStatVarName(set int, varNm string) error {
	st, err := vs.GroupTry(set)
	if err != nil {
		return err
	}
	return st.BindStatVarName(vs, varNm)
}

// BindAllTextureVars binds all Texture vars in given set to their current values,
// iterating over NTextureDescs in case there are multiple Desc sets
// required to represent more than MaxTexturesPerSet.
// Each Value for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
// This calls BindStart / Bind
func (vs *Vars) BindAllTextureVars(set int) error {
	st, err := vs.GroupTry(set)
	if err != nil {
		return err
	}
	cbi := vs.BindDescIndex
	for i := 0; i < st.NTextureDescs; i++ {
		vs.BindVarsStart(i)
		st.BindStatVars(vs)
		vs.BindVarsEnd()
	}
	vs.BindDescIndex = cbi
	return nil
}

/////////////////////////////////////////////////////////////////////////
// Dynamic Binding

// BindVertexValueName dynamically binds given VertexGroup value
// by name for given variable name.
// using given descIndex description set index (among the NDescs allocated).
//
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This dynamically updates the offset to point to the specified val.
//
// Do NOT call BindValuesStart / End around this.
//
// returns error if not found.
func (vs *Vars) BindVertexValueName(varNm, valNm string) error {
	st := vs.GroupMap[VertexGroup]
	vr, vl, err := st.ValueByNameTry(varNm, valNm)
	if err != nil {
		return err
	}
	vr.BindValueIndex[vs.BindDescIndex] = vl.Index // this is then consumed by draw command
	return nil
}

// BindVertexValueIndex dynamically binds given VertexGroup value
// by index for given variable name.
// using given descIndex description set index (among the NDescs allocated).
//
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
//
// Do NOT call BindValuesStart / End around this.
//
// returns error if not found.
func (vs *Vars) BindVertexValueIndex(varNm string, valIndex int) error {
	st := vs.GroupMap[VertexGroup]
	vr, vl, err := st.ValueByIndexTry(varNm, valIndex)
	if err != nil {
		return err
	}
	vr.BindValueIndex[vs.BindDescIndex] = vl.Index // this is then consumed by draw command
	return nil
}

// BindDynValuesAllIndex dynamically binds all uniform, storage values
// by index for all variables in all sets.
//
// This only dynamically updates the offset to point to the specified val.
// MUST call System.BindVars prior to any subsequent draw calls for this
// new offset to be bound at the proper point in the command buffer prior
// (call after all such dynamic bindings are updated.)
//
// Do NOT call BindValuesStart / End around this.
func (vs *Vars) BindDynValuesAllIndex(idx int) {
	for si, st := range vs.GroupMap {
		if si < 0 {
			continue
		}
		st.BindDynValuesAllIndex(vs, idx)
	}
}

// BindDynamicValueName dynamically binds given uniform or storage value
// by name for given variable name, in given set.
//
// This only dynamically updates the offset to point to the specified val.
// MUST call System.BindVars prior to any subsequent draw calls for this
// new offset to be bound at the proper point in the command buffer prior
// (call after all such dynamic bindings are updated.)
//
// Do NOT call BindValuesStart / End around this.
//
// returns error if not found.
func (vs *Vars) BindDynamicValueName(set int, varNm, valNm string) error {
	st, err := vs.GroupTry(set)
	if err != nil {
		return err
	}
	return st.BindDynValueName(vs, varNm, valNm)
}

// BindDynamicValueIndex dynamically binds given uniform or storage value
// by index for given variable name, in given set.
//
// This only dynamically updates the offset to point to the specified val.
// MUST call System.BindVars prior to any subsequent draw calls for this
// new offset to be bound at the proper point in the command buffer prior
// (call after all such dynamic bindings are updated.)
//
// Do NOT call BindValuesStart / End around this.
//
// returns error if not found.
func (vs *Vars) BindDynamicValueIndex(set int, varNm string, valIndex int) error {
	st, err := vs.GroupTry(set)
	if err != nil {
		return err
	}
	return st.BindDynValueIndex(vs, varNm, valIndex)
}

// BindDynamicValue dynamically binds given uniform or storage value
// for given variable in given set.
//
// This only dynamically updates the offset to point to the specified val.
// MUST call System.BindVars prior to any subsequent draw calls for this
// new offset to be bound at the proper point in the command buffer prior
// (call after all such dynamic bindings are updated.)
//
// Do NOT call BindValuesStart / End around this.
//
// returns error if not found.
func (vs *Vars) BindDynamicValue(set int, vr *Var, vl *Value) error {
	st, err := vs.GroupTry(set)
	if err != nil {
		return err
	}
	return st.BindDynValue(vs, vr, vl)
}

// TextureGroupSizeIndexes for texture at given index, allocated in groups by size
// using Values.AllocTexBySize, returns the indexes for the texture
// and layer to actually select the texture in the shader, and proportion
// of the Gp allocated texture size occupied by the texture.
func (vs *Vars) TextureGroupSizeIndexes(set int, varNm string, valIndex int) *szalloc.Indexes {
	st, err := vs.GroupTry(set)
	if err != nil {
		return nil
	}
	return st.TextureGroupSizeIndexes(vs, varNm, valIndex)
}

///////////////////////////////////////////////////////////
// Memory allocation

func (vs *Vars) MemSize(buff *Buffer) int {
	tsz := 0
	ns := vs.NGroups()
	for si := vs.StartGroup(); si < ns; si++ {
		st := vs.GroupMap[si]
		if st == nil {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != buff.Type {
				continue
			}
			tsz += vr.ValuesMemSize(buff.AlignBytes)
		}
	}
	return tsz
}

func (vs *Vars) MemSizeStorage(mm *Memory, alignBytes int) {
	ns := vs.NGroups()
	for si := vs.StartGroup(); si < ns; si++ {
		st := vs.GroupMap[si]
		if st == nil {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != StorageBuffer {
				continue
			}
			vr.MemSizeStorage(mm, alignBytes)
		}
	}
}

func (vs *Vars) AllocMem(buff *Buffer, offset int) int {
	ns := vs.NGroups()
	tsz := 0
	for si := vs.StartGroup(); si < ns; si++ {
		st := vs.GroupMap[si]
		if st == nil || st.Group == PushGroup {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != buff.Type {
				continue
			}
			sz := vr.Values.AllocMem(vr, buff, offset)
			offset += sz
			tsz += sz
		}
	}
	return tsz
}

// Free resets the MemPtr for values, resets any self-owned resources (Textures)
func (vs *Vars) Free(buff *Buffer) {
	ns := vs.NGroups()
	for si := vs.StartGroup(); si < ns; si++ {
		st := vs.GroupMap[si]
		if st == nil || st.Group == PushGroup {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != buff.Type {
				continue
			}
			vr.Values.Free()
		}
	}
}

// ModRegs returns the regions of Values that have been modified
func (vs *Vars) ModRegs(bt BufferTypes) []MemReg {
	ns := vs.NGroups()
	var mods []MemReg
	for si := vs.StartGroup(); si < ns; si++ {
		st := vs.GroupMap[si]
		if st == nil || st.Group == PushSet {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != bt {
				continue
			}
			md := vr.Values.ModRegs(vr)
			mods = append(mods, md...)
		}
	}
	return mods
}

// ModRegStorage returns the regions of Storage Values that have been modified
func (vs *Vars) ModRegsStorage(bufIndex int, buff *Buffer) []MemReg {
	ns := vs.NSets()
	var mods []MemReg
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil || st.Set == PushSet {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != StorageBuffer {
				continue
			}
			if vr.StorageBuffer != bufIndex {
				continue
			}
			md := vr.Values.ModRegs(vr)
			mods = append(mods, md...)
		}
	}
	return mods
}

// AllocTextures allocates images on device memory
func (vs *Vars) AllocTextures(mm *Memory) {
	ns := vs.NSets()
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil || st.Set == PushSet {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role != TextureRole {
				continue
			}
			vr.Values.AllocTextures(mm)
		}
	}
}
