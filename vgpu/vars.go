// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"
	"strings"

	"github.com/goki/ki/indent"
	"goki.dev/vgpu/szalloc"

	vk "github.com/goki/vulkan"
)

// Vars are all the variables that are used by a pipeline,
// organized into Sets (optionally including the special VertexSet
// or PushSet).
// Vars are allocated to bindings / locations sequentially in the
// order added!
type Vars struct {

	// map of sets, by set number -- VertexSet is -2, PushSet is -1, rest are added incrementally
	SetMap map[int]*VarSet `desc:"map of sets, by set number -- VertexSet is -2, PushSet is -1, rest are added incrementally"`

	// map of vars by different roles across all sets -- updated in Config(), after all vars added.  This is needed for VkDescPool allocation.
	RoleMap map[VarRoles][]*Var `desc:"map of vars by different roles across all sets -- updated in Config(), after all vars added.  This is needed for VkDescPool allocation."`

	// true if a VertexSet has been added
	HasVertex bool `inactive:"+" desc:"true if a VertexSet has been added"`

	// true if PushSet has been added
	HasPush bool `inactive:"+" desc:"true if PushSet has been added"`

	// number of complete descriptor sets to construct -- each descriptor set can be bound to a specific pipeline at the start of rendering, and updated with specific Val instances to provide values for each Var used during rendering.  If multiple rendering passes are performed in parallel, then each requires a separate descriptor set (e.g., typically associated with a different Frame in the swapchain), so this number should be increased.
	NDescs int `desc:"number of complete descriptor sets to construct -- each descriptor set can be bound to a specific pipeline at the start of rendering, and updated with specific Val instances to provide values for each Var used during rendering.  If multiple rendering passes are performed in parallel, then each requires a separate descriptor set (e.g., typically associated with a different Frame in the swapchain), so this number should be increased."`

	// [view: -] our parent memory manager
	Mem *Memory `view:"-" desc:"our parent memory manager"`

	// if true, variables are statically bound to specific offsets in memory buffers, vs. dynamically bound offsets.  Typically a compute shader operating on fixed data variables can use static binding, while graphics (e.g., vphong) requires dynamic binding to efficiently use the same shader code for multiple different values of the same variable type
	StaticVars bool `inactive:"+" desc:"if true, variables are statically bound to specific offsets in memory buffers, vs. dynamically bound offsets.  Typically a compute shader operating on fixed data variables can use static binding, while graphics (e.g., vphong) requires dynamic binding to efficiently use the same shader code for multiple different values of the same variable type"`

	// [view: -] vulkan descriptor layout based on vars
	VkDescLayout vk.PipelineLayout `view:"-" desc:"vulkan descriptor layout based on vars"`

	// [view: -] vulkan descriptor pool, allocated for NDescs and the different descriptor pools
	VkDescPool vk.DescriptorPool `view:"-" desc:"vulkan descriptor pool, allocated for NDescs and the different descriptor pools"`

	// allocated descriptor sets -- outer index is Vars.NDescs for different groups of descriptor sets, one of which can be bound to a pipeline at any given time.  The inner dimension is per VarSet to cover the different sets of variable updated at different times or with different numbers of items.  This variable is used for whole-pipline binding at start of rendering.
	VkDescSets [][]vk.DescriptorSet `desc:"allocated descriptor sets -- outer index is Vars.NDescs for different groups of descriptor sets, one of which can be bound to a pipeline at any given time.  The inner dimension is per VarSet to cover the different sets of variable updated at different times or with different numbers of items.  This variable is used for whole-pipline binding at start of rendering."`

	// [view: -] currently accumulating set of vals to write to update bindings -- initiated by BindValsStart, executed by BindValsEnd
	VkWriteVals []vk.WriteDescriptorSet `view:"-" desc:"currently accumulating set of vals to write to update bindings -- initiated by BindValsStart, executed by BindValsEnd"`

	// current descriptor collection index, set in BindValsStart
	BindDescIdx int `inactive:"-" desc:"current descriptor collection index, set in BindValsStart"`

	// dynamic offsets for Uniform and Storage variables, -- outer index is Vars.NDescs for different groups of descriptor sets, one of which can be bound to a pipeline at any given time, inner index is DynOffIdx on Var -- offsets are set when Val is bound via BindDynVal*.
	DynOffs [][]uint32 `desc:"dynamic offsets for Uniform and Storage variables, -- outer index is Vars.NDescs for different groups of descriptor sets, one of which can be bound to a pipeline at any given time, inner index is DynOffIdx on Var -- offsets are set when Val is bound via BindDynVal*."`
}

func (vs *Vars) Destroy(dev vk.Device) {
	vk.DestroyPipelineLayout(dev, vs.VkDescLayout, nil)
	vk.DestroyDescriptorPool(dev, vs.VkDescPool, nil)
	for _, st := range vs.SetMap {
		st.Destroy(dev)
	}
}

// AddVertexSet adds a new Vertex Set -- this is a special Set holding Vertex, Index vars
func (vs *Vars) AddVertexSet() *VarSet {
	if vs.SetMap == nil {
		vs.SetMap = make(map[int]*VarSet)
	}
	st := &VarSet{Set: VertexSet, ParentVars: vs}
	vs.SetMap[VertexSet] = st
	vs.HasVertex = true
	return st
}

// VertexSet returns the Vertex Set -- a special Set holding Vertex, Index vars
func (vs *Vars) VertexSet() *VarSet {
	return vs.SetMap[VertexSet]
}

// AddPushSet adds a new push constant Set -- this is a special Set holding
// values sent directly in the command buffer.
func (vs *Vars) AddPushSet() *VarSet {
	if vs.SetMap == nil {
		vs.SetMap = make(map[int]*VarSet)
	}
	st := &VarSet{Set: PushSet, ParentVars: vs}
	vs.SetMap[PushSet] = st
	vs.HasPush = true
	return st
}

// PushSet returns the Push Set -- a special Set holding push constants
func (vs *Vars) PushSet() *VarSet {
	return vs.SetMap[PushSet]
}

// AddSet adds a new non-Vertex Set for holding Uniforms, Storage, etc
// Sets are automatically numbered sequentially
func (vs *Vars) AddSet() *VarSet {
	if vs.SetMap == nil {
		vs.SetMap = make(map[int]*VarSet)
	}
	idx := vs.NSets()
	st := &VarSet{Set: idx, ParentVars: vs}
	vs.SetMap[idx] = st
	return st
}

// VarByNameTry returns Var by name in given set number,
// returning error if not found
func (vs *Vars) VarByNameTry(set int, name string) (*Var, error) {
	st, err := vs.SetTry(set)
	if err != nil {
		return nil, err
	}
	return st.VarByNameTry(name)
}

// ValByNameTry returns value by first looking up variable name, then value name,
// within given set number, returning error if not found
func (vs *Vars) ValByNameTry(set int, varName, valName string) (*Var, *Val, error) {
	st, err := vs.SetTry(set)
	if err != nil {
		return nil, nil, err
	}
	return st.ValByNameTry(varName, valName)
}

// ValByIdxTry returns value by first looking up variable name, then value index,
// returning error if not found
func (vs *Vars) ValByIdxTry(set int, varName string, valIdx int) (*Var, *Val, error) {
	st, err := vs.SetTry(set)
	if err != nil {
		return nil, nil, err
	}
	return st.ValByIdxTry(varName, valIdx)
}

// Config must be called after all variables have been added.
// Configures all Sets and also does validation, returning error
// does DescLayout too, so all ready for Pipeline config.
func (vs *Vars) Config() error {
	dev := vs.Mem.Device.Device
	ns := vs.NSets()
	var cerr error
	vs.RoleMap = make(map[VarRoles][]*Var)
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
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
	vs.DescLayout(dev)
	return cerr
}

// StringDoc returns info on variables
func (vs *Vars) StringDoc() string {
	ispc := 4
	var sb strings.Builder
	ns := vs.NSets()
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("Set: %d\n", st.Set))

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

// NSets returns the number of regular non-VertexSet sets
func (vs *Vars) NSets() int {
	ex := 0
	if vs.HasVertex {
		ex++
	}
	if vs.HasPush {
		ex++
	}
	return len(vs.SetMap) - ex
}

// StartSet returns the starting set to use for iterating sets
func (vs *Vars) StartSet() int {
	switch {
	case vs.HasVertex:
		return VertexSet
	case vs.HasPush:
		return PushSet
	default:
		return 0
	}
}

// SetTry returns set by index, returning nil and error if not found
func (vs *Vars) SetTry(set int) (*VarSet, error) {
	st, has := vs.SetMap[set]
	if !has {
		err := fmt.Errorf("vgpu.Vars:SetTry set number %d not found", set)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return st, nil
}

// VkVertexConfig returns vulkan vertex config struct, for VertexSet only!
// Note: there is no support for interleaved arrays so each binding and location
// is assigned the same sequential number, recorded in var BindLoc
func (vs *Vars) VkVertexConfig() *vk.PipelineVertexInputStateCreateInfo {
	if vs.HasVertex {
		return vs.SetMap[VertexSet].VkVertexConfig()
	}
	cfg := &vk.PipelineVertexInputStateCreateInfo{}
	cfg.SType = vk.StructureTypePipelineVertexInputStateCreateInfo
	return cfg
}

// VkPushConfig returns vulkan push constant ranges, only if PushSet used.
func (vs *Vars) VkPushConfig() []vk.PushConstantRange {
	if vs.HasPush {
		return vs.SetMap[PushSet].VkPushConfig()
	}
	return nil
}

///////////////////////////////////////////////////////////////////
// Descriptors for Uniforms etc

// key info on descriptorCount -- very confusing.
// https://stackoverflow.com/questions/51715944/descriptor-set-count-ambiguity-in-vulkan

// DescLayout configures the PipelineLayout of DescriptorSetLayout
// info for all of the non-Vertex vars
func (vs *Vars) DescLayout(dev vk.Device) {
	if vs.VkDescLayout != vk.NullPipelineLayout {
		vk.DestroyPipelineLayout(dev, vs.VkDescLayout, nil)
	}
	if vs.VkDescPool != vk.NullDescriptorPool {
		vk.DestroyDescriptorPool(dev, vs.VkDescPool, nil)
	}
	if vs.NDescs < 1 {
		vs.NDescs = 1
	}
	nset := vs.NSets()
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
				vals := vr.Vals.ActiveVals()
				dcount += vs.NDescs * len(vals)
			}
		}
		if dcount == 0 {
			continue
		}
		pl := vk.DescriptorPoolSize{
			DescriptorCount: uint32(dcount),
			Type:            RoleDescriptors[ri],
		}
		if vs.StaticVars {
			pl.Type = StaticRoleDescriptors[ri]
		}
		pools = append(pools, pl)
	}

	var flags vk.DescriptorPoolCreateFlagBits
	flags |= vk.DescriptorPoolCreateUpdateAfterBindBit

	var descPool vk.DescriptorPool
	ret := vk.CreateDescriptorPool(dev, &vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       uint32(vs.NDescs * nset),
		PoolSizeCount: uint32(len(pools)),
		PPoolSizes:    pools,
		Flags:         vk.DescriptorPoolCreateFlags(flags),
	}, nil, &descPool)
	IfPanic(NewError(ret))
	vs.VkDescPool = descPool

	vs.DynOffs = make([][]uint32, vs.NDescs)
	dlays := make([]vk.DescriptorSetLayout, nset)
	for si := 0; si < nset; si++ {
		st := vs.SetMap[si]
		if st == nil {
			continue
		}
		st.DescLayout(dev, vs)
		dlays[si] = st.VkLayout
	}

	if vs.HasVertex {
		vset := vs.SetMap[VertexSet]
		for _, vr := range vset.Vars {
			vr.BindValIdx = make([]int, vs.NDescs)
		}
	}

	dsets := make([][]vk.DescriptorSet, vs.NDescs)
	for i := 0; i < vs.NDescs; i++ {
		dsets[i] = make([]vk.DescriptorSet, nset)
		for si := 0; si < nset; si++ {
			st := vs.SetMap[si]
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
// using given descIdx description set index (among the NDescs allocated).
// Bound vars determine what the shader programs see,
// in subsequent calls to Pipeline commands.
//
// This must be called *prior* to a render pass, never within it.
// Only BindDyn* and BindVertex* calls can be called within render.
//
// Do NOT use this around BindDynVal or BindVertexVal calls
// only for BindVar* methods.
//
// Subsequent calls of BindVar* methods will add to a list, which
// will be executed when BindValsEnd is called.
//
// This creates a set of entries in a list of WriteDescriptorSet's
func (vs *Vars) BindVarsStart(descIdx int) {
	vs.VkWriteVals = []vk.WriteDescriptorSet{}
	vs.BindDescIdx = descIdx
}

// BindVarsEnd finishes a new step of binding started by BindVarsStart.
// Actually executes the binding updates, based on prior BindVar* calls.
func (vs *Vars) BindVarsEnd() {
	dev := vs.Mem.Device.Device
	if len(vs.VkWriteVals) > 0 {
		vk.UpdateDescriptorSets(dev, uint32(len(vs.VkWriteVals)), vs.VkWriteVals, 0, nil)
	}
	vs.VkWriteVals = nil
}

// BindDynVars binds all dynamic vars in given set, to be able to
// use dynamic vars, in subsequent BindDynVal* calls during the
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
	st, err := vs.SetTry(set)
	if err != nil {
		return err
	}
	st.BindDynVars(vs)
	return nil
}

// BindDynVarsAll binds all dynamic vars across all sets.
// Called during system config.
func (vs *Vars) BindDynVarsAll() {
	nset := vs.NSets()
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
		for si, st := range vs.SetMap {
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
	st, err := vs.SetTry(set)
	if err != nil {
		return err
	}
	return st.BindDynVarName(vs, varNm)
}

// BindStatVars binds all static vars to their current values,
// for given set, for non-Uniform, Storage, variables (e.g., Textures).
// Each Val for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
//
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
//
// Must have called BindVarsStart prior to this.
func (vs *Vars) BindStatVars(set int) error {
	st, err := vs.SetTry(set)
	if err != nil {
		return err
	}
	st.BindStatVars(vs)
	return nil
}

// BindStatVarName does static variable binding for given var
// looked up by name, for non-Uniform, Storage, variables (e.g., Textures).
// Each Val for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
//
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
//
// Must have called BindVarsStart prior to this.
func (vs *Vars) BindStatVarName(set int, varNm string) error {
	st, err := vs.SetTry(set)
	if err != nil {
		return err
	}
	return st.BindStatVarName(vs, varNm)
}

// BindAllTextureVars binds all Texture vars in given set to their current values,
// iterating over NTextureDescs in case there are multiple Desc sets
// required to represent more than MaxTexturesPerSet.
// Each Val for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
// This calls BindStart / Bind
func (vs *Vars) BindAllTextureVars(set int) error {
	st, err := vs.SetTry(set)
	if err != nil {
		return err
	}
	cbi := vs.BindDescIdx
	for i := 0; i < st.NTextureDescs; i++ {
		vs.BindVarsStart(i)
		st.BindStatVars(vs)
		vs.BindVarsEnd()
	}
	vs.BindDescIdx = cbi
	return nil
}

/////////////////////////////////////////////////////////////////////////
// Dynamic Binding

// BindVertexValName dynamically binds given VertexSet value
// by name for given variable name.
// using given descIdx description set index (among the NDescs allocated).
//
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This dynamically updates the offset to point to the specified val.
//
// Do NOT call BindValsStart / End around this.
//
// returns error if not found.
func (vs *Vars) BindVertexValName(varNm, valNm string) error {
	st := vs.SetMap[VertexSet]
	vr, vl, err := st.ValByNameTry(varNm, valNm)
	if err != nil {
		return err
	}
	vr.BindValIdx[vs.BindDescIdx] = vl.Idx // this is then consumed by draw command
	return nil
}

// BindVertexValIdx dynamically binds given VertexSet value
// by index for given variable name.
// using given descIdx description set index (among the NDescs allocated).
//
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
//
// Do NOT call BindValsStart / End around this.
//
// returns error if not found.
func (vs *Vars) BindVertexValIdx(varNm string, valIdx int) error {
	st := vs.SetMap[VertexSet]
	vr, vl, err := st.ValByIdxTry(varNm, valIdx)
	if err != nil {
		return err
	}
	vr.BindValIdx[vs.BindDescIdx] = vl.Idx // this is then consumed by draw command
	return nil
}

// BindDynValsAllIdx dynamically binds all uniform, storage values
// by index for all variables in all sets.
//
// This only dynamically updates the offset to point to the specified val.
// MUST call System.BindVars prior to any subsequent draw calls for this
// new offset to be bound at the proper point in the command buffer prior
// (call after all such dynamic bindings are updated.)
//
// Do NOT call BindValsStart / End around this.
func (vs *Vars) BindDynValsAllIdx(idx int) {
	for si, st := range vs.SetMap {
		if si < 0 {
			continue
		}
		st.BindDynValsAllIdx(vs, idx)
	}
}

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
func (vs *Vars) BindDynValName(set int, varNm, valNm string) error {
	st, err := vs.SetTry(set)
	if err != nil {
		return err
	}
	return st.BindDynValName(vs, varNm, valNm)
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
func (vs *Vars) BindDynValIdx(set int, varNm string, valIdx int) error {
	st, err := vs.SetTry(set)
	if err != nil {
		return err
	}
	return st.BindDynValIdx(vs, varNm, valIdx)
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
func (vs *Vars) BindDynVal(set int, vr *Var, vl *Val) error {
	st, err := vs.SetTry(set)
	if err != nil {
		return err
	}
	return st.BindDynVal(vs, vr, vl)
}

// TexGpSzIdxs for texture at given index, allocated in groups by size
// using Vals.AllocTexBySize, returns the indexes for the texture
// and layer to actually select the texture in the shader, and proportion
// of the Gp allocated texture size occupied by the texture.
func (vs *Vars) TexGpSzIdxs(set int, varNm string, valIdx int) *szalloc.Idxs {
	st, err := vs.SetTry(set)
	if err != nil {
		return nil
	}
	return st.TexGpSzIdxs(vs, varNm, valIdx)
}

///////////////////////////////////////////////////////////
// Memory allocation

func (vs *Vars) MemSize(buff *MemBuff) int {
	tsz := 0
	ns := vs.NSets()
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != buff.Type {
				continue
			}
			tsz += vr.ValsMemSize(buff.AlignBytes)
		}
	}
	return tsz
}

func (vs *Vars) MemSizeStorage(mm *Memory, alignBytes int) {
	ns := vs.NSets()
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != StorageBuff {
				continue
			}
			vr.MemSizeStorage(mm, alignBytes)
		}
	}
}

func (vs *Vars) AllocHost(buff *MemBuff, offset int) int {
	ns := vs.NSets()
	tsz := 0
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil || st.Set == PushSet {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != buff.Type {
				continue
			}
			sz := vr.Vals.AllocHost(vr, buff, offset)
			offset += sz
			tsz += sz
		}
	}
	return tsz
}

// Free resets the MemPtr for values, resets any self-owned resources (Textures)
func (vs *Vars) Free(buff *MemBuff) {
	ns := vs.NSets()
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil || st.Set == PushSet {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != buff.Type {
				continue
			}
			vr.Vals.Free()
		}
	}
}

// ModRegs returns the regions of Vals that have been modified
func (vs *Vars) ModRegs(bt BuffTypes) []MemReg {
	ns := vs.NSets()
	var mods []MemReg
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil || st.Set == PushSet {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != bt {
				continue
			}
			md := vr.Vals.ModRegs(vr)
			mods = append(mods, md...)
		}
	}
	return mods
}

// ModRegStorage returns the regions of Storage Vals that have been modified
func (vs *Vars) ModRegsStorage(bufIdx int, buff *MemBuff) []MemReg {
	ns := vs.NSets()
	var mods []MemReg
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil || st.Set == PushSet {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != StorageBuff {
				continue
			}
			if vr.StorageBuff != bufIdx {
				continue
			}
			md := vr.Vals.ModRegs(vr)
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
			vr.Vals.AllocTextures(mm)
		}
	}
}
