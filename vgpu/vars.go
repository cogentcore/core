// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"strings"

	"github.com/goki/ki/indent"

	vk "github.com/vulkan-go/vulkan"
)

// Vars are all the variables that are used by a pipeline,
// organized into Sets (optionally including the special VertexSet).
// Vars are allocated to bindings / locations sequentially in the
// order added!
type Vars struct {
	SetMap    map[int]*VarSet     `desc:"map of sets, by set number -- VertexSet is -1, rest are added incrementally"`
	RoleMap   map[VarRoles][]*Var `desc:"map of vars by different roles across all sets -- updated in Config(), after all vars added.  This is needed for VkDescPool allocation."`
	HasVertex bool                `desc:"set to true if a VertexSet has been added"`
	NDescs    int                 `desc:"number of complete descriptor sets to construct -- each descriptor set can be bound to a specific pipeline at the start of rendering, and updated with specific Val instances to provide values for each Var used during rendering.  If multiple rendering passes are performed in parallel, then each requires a separate descriptor set (e.g., typically associated with a different Frame in the swapchain), so this number should be increased."`
	Mem       *Memory             `view:"-" desc:"our parent memory manager"`

	VkDescLayout vk.PipelineLayout       `view:"-" desc:"vulkan descriptor layout based on vars"`
	VkDescPool   vk.DescriptorPool       `view:"-" desc:"vulkan descriptor pool, allocated for NDescs and the different descriptor pools"`
	VkDescSets   [][]vk.DescriptorSet    `desc:"allocated descriptor sets -- outer index is Vars.NDescs for different groups of descriptor sets, one of which can be bound to a pipeline at any given time.  The inner dimension is per VarSet to cover the different sets of variable updated at different times or with different numbers of items.  This variable is used for whole-pipline binding at start of rendering."`
	VkWriteVals  []vk.WriteDescriptorSet `view:"-" desc:"currently accumulating set of vals to write to update bindings -- initiated by BindValsStart, executed by BindValsEnd"`
	BindDescIdx  int                     `inactive:"-" desc:"current descriptor collection index, set in BindValsStart"`
	DynOffs      [][]uint32              `desc:"dynamic offsets for Uniform and Storage variables, -- outer index is Vars.NDescs for different groups of descriptor sets, one of which can be bound to a pipeline at any given time, inner index is DynOffIdx on Var -- offsets are set when Val is bound via BindDynVal*."`
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
	st := &VarSet{Set: VertexSet}
	vs.SetMap[VertexSet] = st
	vs.HasVertex = true
	return st
}

// AddSet adds a new non-Vertex Set for holding Uniforms, Storage, etc
// Sets are automatically numbered sequentially
func (vs *Vars) AddSet() *VarSet {
	if vs.SetMap == nil {
		vs.SetMap = make(map[int]*VarSet)
	}
	idx := vs.NSets()
	st := &VarSet{Set: idx}
	vs.SetMap[idx] = st
	return st
}

// VarByNameTry returns Var by name in given set number,
// returning error if not found
func (vs *Vars) VarByNameTry(set int, name string) (*Var, error) {
	st := vs.SetMap[set]
	return st.VarByNameTry(name)
}

// ValByNameTry returns value by first looking up variable name, then value name,
// within given set number, returning error if not found
func (vs *Vars) ValByNameTry(set int, varName, valName string) (*Var, *Val, error) {
	st := vs.SetMap[set]
	return st.ValByNameTry(varName, valName)
}

// ValByIdxTry returns value by first looking up variable name, then value index,
// returning error if not found
func (vs *Vars) ValByIdxTry(set int, varName string, valIdx int) (*Var, *Val, error) {
	st := vs.SetMap[set]
	return st.ValByIdxTry(varName, valIdx)
}

// Config must be called after all variables have been added.
// Configures all Sets and also does validation, returning error
// does DescLayout too, so all ready for Pipeline config.
func (vs *Vars) Config(dev vk.Device) error {
	ns := vs.NSets()
	var cerr error
	vs.RoleMap = make(map[VarRoles][]*Var)
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		err := st.Config()
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
	if vs.HasVertex {
		return len(vs.SetMap) - 1
	}
	return len(vs.SetMap)
}

// StartSet returns the starting set to use for iterating sets
func (vs *Vars) StartSet() int {
	if vs.HasVertex {
		return VertexSet
	}
	return 0
}

// VkVertexConfig fills in the relevant info into given vulkan config struct.
// for VertexSet only!
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

///////////////////////////////////////////////////////////////////
// Descriptors for Uniforms etc

// key info on descriptorCount -- very confusing.
// https://stackoverflow.com/questions/51715944/descriptor-set-count-ambiguity-in-vulkan

// DescLayout configures the PipelineLayout of DescriptorSetLayout
// info for all of the non-Vertex vars
func (vs *Vars) DescLayout(dev vk.Device) {
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
		vs.VkDescPool = nil
		vs.VkDescSets = nil
		return
	}

	var pools []vk.DescriptorPoolSize
	for ri := Uniform; ri < VarRolesN; ri++ {
		vl := vs.RoleMap[ri]
		if len(vl) == 0 {
			continue
		}
		pl := vk.DescriptorPoolSize{
			DescriptorCount: uint32(vs.NDescs * len(vl)),
			Type:            RoleDescriptors[ri],
		}
		pools = append(pools, pl)
	}

	var descPool vk.DescriptorPool
	ret := vk.CreateDescriptorPool(dev, &vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       uint32(vs.NDescs * nset),
		PoolSizeCount: uint32(len(pools)),
		PPoolSizes:    pools,
	}, nil, &descPool)
	IfPanic(NewError(ret))
	vs.VkDescPool = descPool

	vs.DynOffs = make([][]uint32, vs.NDescs)
	dlays := make([]vk.DescriptorSetLayout, nset)
	for si := 0; si < nset; si++ {
		st := vs.SetMap[si]
		st.DescLayout(dev, vs)
		dlays[si] = st.VkLayout
	}

	dsets := make([][]vk.DescriptorSet, vs.NDescs)
	for i := 0; i < vs.NDescs; i++ {
		dsets[i] = make([]vk.DescriptorSet, nset)
		for si := 0; si < nset; si++ {
			st := vs.SetMap[si]
			dsets[i][si] = st.VkDescSets[i]
		}
	}
	vs.VkDescSets = dsets // for pipeline binding

	var pipelineLayout vk.PipelineLayout
	ret = vk.CreatePipelineLayout(dev, &vk.PipelineLayoutCreateInfo{
		SType:          vk.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount: uint32(len(dlays)),
		PSetLayouts:    dlays,
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

// BindVertexValName dynamically binds given VertexSet value
// by name for given variable name.
// using given descIdx description set index (among the NDescs allocated).
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (vs *Vars) BindVertexValName(descIdx int, varNm, valNm string) error {
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
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (vs *Vars) BindVertexValIdx(descIdx int, varNm string, valIdx int) error {
	st := vs.SetMap[VertexSet]
	vr, vl, err := st.ValByIdxTry(varNm, valIdx)
	if err != nil {
		return err
	}
	vr.BindValIdx[vs.BindDescIdx] = vl.Idx // this is then consumed by draw command
	return nil
}

// BindValsStart starts a new step of binding specific vals for vars,
// using given descIdx description set index (among the NDescs allocated).
// BoundVals determine what value the shader programs see,
// in subsequent calls to Pipeline commands.
// Subsequent calls of BindVal* methods will add to a list, which
// will be executed when BindValsEnd is called.
// This creates a set of entries in a list of WriteDescriptorSet's
func (vs *Vars) BindValsStart(descIdx int) {
	vs.VkWriteVals = []vk.WriteDescriptorSet{}
	vs.BindDescIdx = descIdx
}

// BindValsEnd finishes a new step of binding started by BindValsStart.
// Actually executes the binding updates, based on prior BindVal calls.
func (vs *Vars) BindValsEnd(dev vk.Device) {
	if len(vs.VkWriteVals) > 0 {
		vk.UpdateDescriptorSets(dev, uint32(len(vs.VkWriteVals)), vs.VkWriteVals, 0, nil)
	}
	vs.VkWriteVals = nil
}

// BindDynValName dynamically binds given uniform or storage value
// by name for given variable name, in given set.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (vs *Vars) BindDynValName(set int, varNm, valNm string) error {
	st := vs.SetMap[set]
	return st.BindDynValName(vs, varNm, valNm)
}

// BindDynValIdx dynamically binds given uniform or storage value
// by index for given variable name, in given set.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (vs *Vars) BindDynValIdx(set int, varNm string, valIdx int) error {
	st := vs.SetMap[set]
	return st.BindDynValIdx(vs, varNm, valIdx)
}

// todo: need an option to allow a single val to be used in a static way, selecting from among multiple,
// instead of always assuming an array used.

// BindStatVars binds all static vars to their current values,
// for given set, for non-Uniform, Storage, variables (e.g., Textures).
// Each Val for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
// Must have called BindValsStart prior to this.
func (vs *Vars) BindStatVars(set int) {
	st := vs.SetMap[set]
	st.BindStatVars(vs)
}

// BindStatVarName does static variable binding for given var
// looked up by name, for non-Uniform, Storage, variables (e.g., Textures).
// Each Val for a given Var is given a descriptor binding
// and the shader sees an array of values of corresponding length.
// All vals must be uploaded to Device memory prior to this,
// and it is not possible to update anything during a render pass.
// Must have called BindValsStart prior to this.
func (vs *Vars) BindStatVarName(set int, varNm string) error {
	st := vs.SetMap[set]
	return st.BindStatVarName(vs, varNm)
}

///////////////////////////////////////////////////////////
// Memory allocation

func (vs *Vars) MemSize(buff *MemBuff) int {
	tsz := 0
	ns := vs.NSets()
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != buff.Type {
				continue
			}
			tsz += vr.MemSize(buff.AlignBytes)
		}
	}
	return tsz
}

func (vs *Vars) AllocHost(buff *MemBuff, offset int) int {
	ns := vs.NSets()
	tsz := 0
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
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
		for _, vr := range st.Vars {
			if vr.Role.BuffType() != bt {
				continue
			}
			md := vr.Vals.ModRegs()
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
		for _, vr := range st.Vars {
			if vr.Role != TextureRole {
				continue
			}
			vr.Vals.AllocTextures(mm)
		}
	}
}
