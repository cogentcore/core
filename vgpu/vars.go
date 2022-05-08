// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"
	"strings"

	"github.com/goki/ki/indent"
	"github.com/goki/ki/kit"

	vk "github.com/vulkan-go/vulkan"
)

// Var specifies a variable used in a pipeline, accessed in shader programs.
// A Var represents a type of input or output into the GPU program,
// including things like Vertex arrays, transformation matricies (Uniforms),
// Images (Textures), and arbitrary Structs for Compute shaders.
// Each Var belongs to a Set, and its binding location is allocated within that.
// Each set is updated at the same time scale, and all vars in the set have the same
// number of allocated Val instances representing a specific value of the variable.
// There must be a unique Val instance for each value of the variable used in
// a single render -- a previously-used Val's contents cannot be updated within
// the render pass, but new information can be written to an as-yet unused Val
// prior to using in a render (although this comes at a performance cost).
type Var struct {
	Name        string                 `desc:"variable name"`
	Type        Types                  `desc:"type of data in variable.  Note that there are strict contraints on the alignment of fields within structs -- if you can keep all fields at 4 byte increments, that works, but otherwise larger fields trigger a 16 byte alignment constraint.  Images are allocated by default in , "`
	ArrayN      int                    `desc:"number of elements if this is a fixed array -- use 1 if singular element, and 0 if a variable-sized array, where each Val can have its own specific size."`
	Role        VarRoles               `desc:"role of variable: Vertex is configured in the pipeline VkConfig structure, and everything else is configured in a DescriptorSet, etc. "`
	Shaders     vk.ShaderStageFlagBits `desc:"bit flags for set of shaders that this variable is used in"`
	Set         int                    `desc:"DescriptorSet associated with the timing of binding for this variable -- all vars updated at the same time should be in the same set"`
	BindLoc     int                    `desc:"binding or location number for variable -- Vertexs are assigned as one group sequentially in order listed in Vars, and rest are assigned uniform binding numbers via descriptor pools"`
	SizeOf      int                    `desc:"size in bytes of one element (not array size).  Note that arrays require 16 byte alignment for each element, so if using arrays, it is best to work within that constraint."`
	TextureOwns bool                   `desc:"texture manages its own memory allocation -- set this for texture objects that change size dynamically -- otherwise image host staging memory is allocated in a common buffer"`
	DynOffIdx   int                    `desc:"index into the dynamic offset list, where dynamic offsets of vals need to be set -- for Uniform and Storage roles -- set during Set:DescLayout"`
	Vals        ValList                `desc:"the array of values allocated for this variable.  The size of this array is determined by the Set membership of this Var, and the current index is updated at the set level.  For Texture Roles, there is a separate descriptor for each value (image) -- otherwise dynamic offset binding is used."`
	CurValIdx   int                    `desc:"current value index, selected prior to updating Set during render (e.g., using Set CurValIdx or selected by other logic)"`
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
// VarList

// VarList is a list of variables
type VarList struct {
	Vars   []*Var          `desc:"variables in order"`
	VarMap map[string]*Var `desc:"map of vars by name -- names must be unique"`
}

// AddVar adds given variable
func (vs *VarList) AddVar(vr *Var) {
	if vs.VarMap == nil {
		vs.VarMap = make(map[string]*Var)
	}
	vs.Vars = append(vs.Vars, vr)
	vs.VarMap[vr.Name] = vr
}

// Add adds a new variable of given type, role, set, and shaders where used
func (vs *VarList) Add(name string, typ Types, role VarRoles, set int, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.Init(name, typ, role, set, shaders...)
	vs.AddVar(vr)
	return vr
}

// AddStruct adds a new struct variable of given total number of bytes in size,
// type, role, set, and shaders where used
func (vs *VarList) AddStruct(name string, size int, role VarRoles, set int, shaders ...ShaderTypes) *Var {
	vr := &Var{}
	vr.Init(name, Struct, role, set, shaders...)
	vr.SizeOf = size
	vs.AddVar(vr)
	return vr
}

// VarByNameTry returns Var by name, returning error if not found
func (vs *VarList) VarByNameTry(name string) (*Var, error) {
	vr, ok := vs.VarMap[name]
	if !ok {
		err := fmt.Errorf("Variable named %s not found", name)
		if TheGPU.Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vr, nil
}

// ValByNameTry returns value by first looking up variable name, then value name,
// returning error if not found
func (vs *VarList) ValByNameTry(varName, valName string) (*Var, *Val, error) {
	vr, err := vs.VarByNameTry(varName)
	if err != nil {
		return nil, nil, err
	}
	vl, err := vr.Vals.ValByNameTry(valName)
	return vr, vl, err
}

// ValByIdxTry returns value by first looking up variable name, then value index,
// returning error if not found
func (vs *VarList) ValByIdxTry(varName string, valIdx int) (*Var, *Val, error) {
	vr, err := vs.VarByNameTry(varName)
	if err != nil {
		return nil, nil, err
	}
	vl, err := vr.Vals.ValByIdxTry(valIdx)
	return vr, vl, err
}

//////////////////////////////////////////////////////////////////
// Set

const VertexSet = -1

// Set contains a set of Var variables that are all updated at the same time
// and have the same number of distinct Vals values per Var per render pass.
// The first set at index -1 contains Vertex and Index data, handed separately.
type Set struct {
	VarList
	Set      int                 `desc:"set number"`
	NValsPer int                 `desc:"number of value instances to allocate per variable in this set: each value must be allocated in advance for each unique instance of a variable required across a complete scene rendering -- e.g., if this is an object position matrix, then one per object is required.  If a dynamic number are required, allocate the max possible"`
	RoleMap  map[VarRoles][]*Var `desc:"map of vars by different roles, within this set -- updated in Config(), after all vars added"`
	DynOffs  []uint32            `desc:"dynamic offsets for Uniform and Storage variables, indexed as DynOffIdx on Var -- offsets are set when Val is bound"`

	VkLayout   vk.DescriptorSetLayout `desc:"set layout info -- static description of each var type, role, binding, stages"`
	VkDescSets []vk.DescriptorSet     `desc:"allocated descriptor set -- one of these per Vars.NDescs -- can have multiple sets that can be independently updated, e.g., for parallel rendering passes.  If only rendering one at a time, only need one."`
}

// Config must be called after all variables have been added.
// configures binding / location for all vars based on sequential order.
// also does validation and returns error message.
func (st *Set) Config() error {
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
func (st *Set) ConfigVals(nvals int) {
	st.NValsPer = nvals
	for _, vr := range st.Vars {
		vr.Vals.ConfigVals(vr, nvals)
	}
}

// Destroy destroys infrastructure for Set, Vars and Vals -- assumes Free has
// already been called to free host and device memory.
func (st *Set) Destroy(dev vk.Device) {
	st.DestroyLayout(dev)
}

// DestroyLayout destroys layout
func (st *Set) DestroyLayout(dev vk.Device) {
	vk.DestroyDescriptorSetLayout(dev, sd.VkLayout, nil)
	st.VkLayout = nil
}

// DescLayout creates the DescriptorSetLayout in DescLayout for given set.
// Only for non-VertexSet sets.
// Must have set NValsPer for any TextureRole vars, which require separate descriptors per.
func (st *Set) DescLayout(dev vk.Device, descPool vk.DescriptorPool, ndescs int) {
	if st.Set == VertexSet {
		return
	}
	st.DestroyLayout(dev)
	var descLayout vk.DescriptorSetLayout
	var binds []vk.DescriptorSetLayoutBinding
	dyno := 0
	for _, vr := range st.Vars {
		bd := vk.DescriptorSetLayoutBinding{
			Binding:         uint32(vr.BindLoc),
			DescriptorType:  RoleDescriptors[vr.Role],
			DescriptorCount: 1, // needed for Texture!
			StageFlags:      vk.ShaderStageFlags(vr.Shaders),
		}
		if v.Role == TextureRole {
			bd.DescriptorCount = uint32(st.NValsPer)
		}
		binds = append(binds, bd)
		if vr.Role == Uniform || vr.Role == Storage {
			vr.DynOffIdx = dyno
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
	if dyno > 0 {
		st.DynOffs = make([]uint32, dyno)
	}

	st.VkDescSets = make([]vk.DescriptorSet, ndescs)
	for i := 0; i < ndescs; i++ {
		var dset vk.DescriptorSet
		ret := vk.AllocateDescriptorSets(dev, &vk.DescriptorSetAllocateInfo{
			SType:              vk.StructureTypeDescriptorSetAllocateInfo,
			DescriptorPool:     descPool,
			DescriptorSetCount: 1,
			PSetLayouts:        []vk.DescriptorSetLayout{st.VkLayout},
		}, &set)
		IfPanic(NewError(ret))
		sd.VkDescSets[i] = set
	}
}

// BindValName dynamically binds given uniform or storage value
// by name for given variable name, to use for subsequent pipeline
// command step, using given description set index.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (st *Set) BindValName(vs *Vars, descIdx int, varNm, valNm string) error {
	vr, vl, err := st.ValByNameTry(varNm, valNm)
	if err != nil {
		return err
	}
	st.BindVal(vs, descIdx, vl)
	return nil
}

// BindValIndex dynamically binds given uniform or storage value
// by index for given variable name, to use for subsequent pipeline
// command step, using given description set index.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (st *Set) BindValIndex(vs *Vars, descIdx int, varNm string, valIdx int) error {
	vr, vl, err := st.ValByIndexTry(varNm, valIdx)
	if err != nil {
		return err
	}
	return st.BindVal(vs, descIdx, vl)
}

// BindVal binds given value to use for subsequent pipeline command step.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (st *Set) BindVal(vs *Vars, descIdx int, vr *Var, vl *Val) error {
	if vl.Var.Role < Uniform || vl.Var.Role > Storage {
		err := fmt.Errorf("vgpu.Set:BindVal only valid for Uniform or Storage Vars, not: %s", vl.Var.Role.String())
		if TheGPU.Debug {
			log.Println(err)
		}
		return err
	}
	wd := vk.WriteDescriptorSet{
		SType:           vk.StructureTypeWriteDescriptorSet,
		DstSet:          st.VkDescSets[descIdx],
		DstBinding:      uint32(vr.BindLoc),
		DescriptorCount: 1,
		DescriptorType:  vr.Role.VkDescriptor(),
	}
	buff := vl.Buff // sy.Mem.Buffs[vl.BuffType()]
	wd.PBufferInfo = []vk.DescriptorBufferInfo{{
		Offset: 0, // dynamic
		Range:  vk.DeviceSize(vl.MemSize),
		Buffer: buff.Dev,
	}}
	st.DynOffs[vr.DynOffIdx] = uint32(vl.Offset)
	vs.VkWriteVals = append(vs.VkWriteVals, wd)
}

// todo: other static cases need same approach as images!
// also, need an option to allow a single val to be used in a static way, selecting from among multiple,
// instead of always assuming an array used.

// BindStaticVars binds all static vars to their current values
func (st *Set) BindStaticVars(vs *Vars, descIdx int) error {
	for _, vr := range st.Vars {
		if vl.Var.Role < Storage {
			continue
		}
		wd := vk.WriteDescriptorSet{
			SType:           vk.StructureTypeWriteDescriptorSet,
			DstSet:          st.VkDescSets[descIdx],
			DstBinding:      uint32(vr.BindLoc),
			DescriptorCount: 1,
			DescriptorType:  vr.Role.VkDescriptor(),
		}
		if vr.Role < TextureRole {
			off := vk.DeviceSize(vl.Offset)
			if vr.Role.IsDynamic() {
				off = 0 // off must be 0 for dynamic
			}
			buff := vl.Buff // sy.Mem.Buffs[vl.BuffType()]
			wd.PBufferInfo = []vk.DescriptorBufferInfo{{
				Offset: off,
				Range:  vk.DeviceSize(vl.MemSize),
				Buffer: buff.Dev,
			}}
			if vl.Var.Role.IsDynamic() {
				st.DynOffs[vr.DynOffIdx] = uint32(vl.Offset)
			}
		} else {
			wd.PImageInfo = []vk.DescriptorImageInfo{{
				ImageLayout: vk.ImageLayoutShaderReadOnlyOptimal,
				ImageView:   vl.Texture.View,
				Sampler:     vl.Texture.VkSampler,
			}}
		}
		vs.VkWriteVals = append(vs.VkWriteVals, wd)
	}
}

// VertexConfig fills in the relevant info into given vulkan config struct.
// for VertexSet only!
// Note: there is no support for interleaved arrays so each binding and location
// is assigned the same sequential number, recorded in var BindLoc
func (st *Set) VertexConfig() *vk.PipelineVertexInputStateCreateInfo {
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
	cfg.VertexBindingDescriptionCount = uint32(len(vtx))
	cfg.PVertexBindingDescriptions = bind
	cfg.VertexAttributeDescriptionCount = uint32(len(vtx))
	cfg.PVertexAttributeDescriptions = attr
	return cfg
}

//////////////////////////////////////////////////////////////////
// Vars

// Vars are all the variables that are used by a pipeline,
// organized into Sets (optionally including the special VertexSet).
// Vars are allocated to bindings / locations sequentially in the
// order added!
type Vars struct {
	SetMap    map[int]*Set        `desc:"map of sets, by set number -- VertexSet is -1, rest are added incrementally"`
	RoleMap   map[VarRoles][]*Var `desc:"map of vars by different roles across all sets -- updated in Config(), after all vars added.  This is needed for VkDescPool allocation."`
	HasVertex bool                `desc:"set to true if a VertexSet has been added"`
	NDescs    int                 `desc:"number of complete descriptor sets to construct -- each descriptor set can be bound to a specific pipeline at the start of rendering, and updated with specific Val instances to provide values for each Var used during rendering.  If multiple rendering passes are performed in parallel, then each requires a separate descriptor set (e.g., typically associated with a different Frame in the swapchain), so this number should be increased."`

	VkDescLayout vk.PipelineLayout       `view:"-" desc:"vulkan descriptor layout based on vars"`
	VkDescPool   vk.DescriptorPool       `view:"-" desc:"vulkan descriptor pool, allocated for NDescs and the different descriptor pools"`
	VkWriteVals  []vk.WriteDescriptorSet `view:"-" desc:"currently accumulating set of vals to write to update bindings -- initiated by BindValsStart, executed by BindValsEnd"`
}

func (vs *Vars) Destroy(dev vk.Device) {
	vk.DestroyPipelineLayout(dev, vs.VkDescLayout, nil)
	vk.DestroyDescriptorPool(dev, vs.VkDescPool, nil)
	for _, st := range vs.SetMap {
		st.Destroy(dev)
	}
}

// AddVertexSet adds a new Vertex Set -- this is a special Set holding Vertex, Index vars
func (vs *Vars) AddVertexSet() *Set {
	if vs.SetMap == nil {
		vs.SetMap = make(map[int]*Set)
	}
	st := &Set{Set: VertexSet}
	vs.SetMap[VertexSet] = st
	vs.HasVertex = true
	return st
}

// AddSet adds a new non-Vertex Set for holding Uniforms, Storage, etc
// Sets are automatically numbered sequentially
func (vs *Vars) AddSet() *Set {
	if vs.SetMap == nil {
		vs.SetMap = make(map[int]*Set)
	}
	idx := vs.NSets()
	st := &Set{Set: idx}
	vs.SetMap[idx] = st
	return st
}

// Config must be called after all variables have been added.
// Configures all Sets and also does validation, returning error
func (vs *Vars) Config() error {
	ns := vs.NSets()
	var cerr error
	vs.RoleMap = make(map[VarRoles][]*Var)
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.Sets[si]
		err := st.Config()
		if err != nil {
			cerr = err
		}
		for ri, rl := range st.RoleMap {
			vs.RoleMap[ri] = append(vs.RoleMap[ri], rl...)
		}
	}
	return cerr
}

// StringDoc returns info on variables
func (vs *Vars) StringDoc() string {
	ispc := 4
	var sb strings.Builder
	ns := vs.NSets()
	var cerr error
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.Sets[si]
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
		vl := vs.RoleMap[rl]
		if len(vl) == 0 {
			continue
		}
		pl := vk.DescriptorPoolSize{
			DescriptorCount: uint32(vs.NDescs * len(vl)),
			Type:            RoleDescriptors[rl],
		}
		pools = append(pools, pl)
	}

	var descPool vk.DescriptorPool
	ret = vk.CreateDescriptorPool(dev, &vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       uint32(vs.NDescs * nset),
		PoolSizeCount: uint32(len(pools)),
		PPoolSizes:    pools,
	}, nil, &descPool)
	IfPanic(NewError(ret))
	vs.VkDescPool = descPool

	dsets := make([]vk.DescriptorSetLayout, nset)
	for si := 0; si < nset; si++ {
		st := vs.SetMap[si]
		st.DescLayout(dev, vs.VkDescPool, vs.NDescs)
		dsets[si] = sd.VkLayout
	}
	var pipelineLayout vk.PipelineLayout
	ret := vk.CreatePipelineLayout(dev, &vk.PipelineLayoutCreateInfo{
		SType:          vk.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount: uint32(len(dsets)),
		PSetLayouts:    dsets,
	}, nil, &pipelineLayout)
	IfPanic(NewError(ret))
	vs.VkDescLayout = pipelineLayout
}

// BindValsStart starts a new step of binding specific vals for vars
// Subsequent calls of BindVal* methods will add to a list, which
// will be executed when BindValsEnd is called.
// This creates a set of entries in a list of WriteDescriptorSet's
func (vs *Vars) BindValsStart() {
	vs.VkWriteVals = []vk.WriteDescriptorSet{}
}

// BindValsEnd finishes a new step of binding specific vals for vars
// and actually does the binding updates, based on prior BindVal calls.
func (vs *Vars) BindValsEnd(dev vk.Device) {
	if len(vs.VkWriteVals) > 0 {
		vk.UpdateDescriptorSets(dev, uint32(len(ws)), ws, 0, nil)
	}
	vs.VkWriteVals = nil
}

// BindValName dynamically binds given uniform or storage value
// by name for given variable name, to use for subsequent pipeline
// command step, using given description set index.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (vs *Vars) BindValName(set int, descIdx int, varNm, valNm string) error {
	st := vs.SetMap[set]
	return st.BindValName(vs, descIdx, varNm, valIdx)
}

// BindValIndex dynamically binds given uniform or storage value
// by index for given variable name, to use for subsequent pipeline
// command step, using given description set index.
// Value must have already been updated into device memory prior to this,
// ideally through a batch update prior to starting rendering, so that
// all the values are ready to be used during the render pass.
// This only dynamically updates the offset to point to the specified val.
// Must have called BindValsStart prior to this.
// returns error if not found.
func (vs *Vars) BindValIndex(set int, descIdx int, varNm string, valIdx int) error {
	st := vs.SetMap[set]
	return st.BindValIndex(vs, descIdx, varNm, valIdx)
}

// todo: other static cases need same approach as images!
// also, need an option to allow a single val to be used in a static way, selecting from among multiple,
// instead of always assuming an array used.

// BindStaticVars binds all static vars to their current values, for given set
func (vs *Vars) BindStaticVars(set int, descIdx int) error {
	st := vs.SetMap[set]
	st.BindStaticVars(vs, descIdx)
}

//////////////////////////////////////////////////////////////////

// VarRoles are the functional roles of variables, corresponding
// to Vertex input vectors and all the different "uniform" types
// as enumerated in vk.DescriptorType.  This does NOT map directly
// to DescriptorType because we combine vertex and uniform data
// and require a different ordering.
type VarRoles int32

const (
	UndefVarRole VarRoles = iota
	Vertex                // vertex shader input data: mesh geometry points, normals, etc.  These are automatically located in a separate Set, VertexSet (-1), and managed separately.
	Index                 // for indexed access to Vertex data, also located in VertexSet (-1)
	Uniform               // read-only general purpose data, uses UniformBufferDynamic with offset specified at binding time, not during initial configuration -- compared to Storage, Uniform items can be put in local cache for each shader and thus can be much faster to access -- use for a smaller number of parameters such as transformation matricies
	Storage               // read-write general purpose data, in StorageBufferDynamic (offset set at binding) -- this is a larger but slower pool of memory, with more flexible alignment constraints, used primarily for compute data
	UniformTexel          // read-only image-formatted data, which cannot be accessed via ImageView or Sampler -- only for rare cases where optimized image format (e.g., rgb values of specific bit count) is useful.  No Dynamic mode is available, so this can only be used for a fixed Val.
	StorageTexel          // read-write image-formatted data, which cannot be accessed via ImageView or Sampler -- only for rare cases where optimized image format (e.g., rgb values of specific bit count) is useful. No Dynamic mode is available, so this can only be used for a fixed Val.
	StorageImage          // read-write access through an ImageView (but not a Sampler) of an Image
	TextureRole           // a Texture is a CombinedImageSampler in Vulkan terminology -- a combination of a Sampler and a specific Image, which appears as a single entity in the shader.
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

// VkDescriptor returns the vk.DescriptorType
func (vr VarRoles) VkDescriptor() vk.DescriptorType {
	return RoleDescriptors[vr]
}

var RoleDescriptors = map[VarRoles]vk.DescriptorType{
	Uniform:      vk.DescriptorTypeUniformBufferDynamic,
	Storage:      vk.DescriptorTypeStorageBufferDynamic,
	UniformTexel: vk.DescriptorTypeUniformTexelBuffer,
	StorageTexel: vk.DescriptorTypeStorageTexelBuffer,
	StorageImage: vk.DescriptorTypeStorageImage,
	TextureRole:  vk.DescriptorTypeCombinedImageSampler,
}

// RoleBuffers maps VarRoles onto type of memory buffer
var RoleBuffers = map[VarRoles]BuffTypes{
	UndefVarRole: StorageBuff,
	Vertex:       VtxIdxBuff,
	Index:        VtxIdxBuff,
	Uniform:      UniformBuff,
	Storage:      StorageBuff,
	UniformTexel: UniformBuff,
	StorageTexel: StorageBuff,
	StorageImage: StorageBuff,
	TextureRole:  ImageBuff,
}
