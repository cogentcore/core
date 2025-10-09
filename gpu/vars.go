// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"log"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/indent"

	"github.com/cogentcore/webgpu/wgpu"
)

// Vars are all the variables that are used by a pipeline,
// organized into Groups (optionally including the special VertexGroup
// or PushGroup).
// Vars are allocated to bindings sequentially in the order added.
type Vars struct {
	// map of Groups, by group number: VertexGroup is -2, PushGroup is -1,
	// rest are added incrementally.
	Groups map[int]*VarGroup

	// map of vars by different roles across all Groups, updated in Config(),
	// after all vars added.
	RoleMap map[VarRoles][]*Var

	// full set of BindGroupLayouts, one for each VarGroup >= 0
	layouts []*wgpu.BindGroupLayout `display:"-"`

	// true if a VertexGroup has been added
	hasVertex bool `edit:"-"`

	// true if PushGroup has been added.  Note: not yet supported in WebGPU.
	hasPush bool `edit:"-"`

	sys System

	device Device
}

func (vs *Vars) Release() {
	for _, vg := range vs.Groups {
		vg.Release()
	}
}

// AddVertexGroup adds a new Vertex Group.
// This is a special Group holding Vertex, Index vars
func (vs *Vars) AddVertexGroup() *VarGroup {
	if vs.Groups == nil {
		vs.Groups = make(map[int]*VarGroup)
	}
	vg := &VarGroup{Name: "Vertex", Group: VertexGroup, Role: Vertex, alignBytes: 1, device: vs.device}
	vs.Groups[VertexGroup] = vg
	vs.hasVertex = true
	return vg
}

// VertexGroup returns the Vertex Group -- a special Group holding Vertex, Index vars
func (vs *Vars) VertexGroup() *VarGroup {
	return vs.Groups[VertexGroup]
}

// AddPushGroup adds a new push constant Group -- this is a special Group holding
// values sent directly in the command buffer.
func (vs *Vars) AddPushGroup() *VarGroup {
	if vs.Groups == nil {
		vs.Groups = make(map[int]*VarGroup)
	}
	vg := &VarGroup{Name: "Push", Group: PushGroup, alignBytes: 1, device: vs.device}
	vs.Groups[PushGroup] = vg
	vs.hasPush = true
	return vg
}

// PushGroup returns the Push Group -- a special Group holding push constants
func (vs *Vars) PushGroup() *VarGroup {
	return vs.Groups[PushGroup]
}

// AddGroup adds a new non-Vertex Group for holding data for given Role
// (Uniform, Storage, etc).
// Groups are automatically numbered sequentially in order added.
// Name is optional and just provides documentation.
// Important limit: there can only be a maximum of 4 Groups!
func (vs *Vars) AddGroup(role VarRoles, name ...string) *VarGroup {
	if vs.Groups == nil {
		vs.Groups = make(map[int]*VarGroup)
	}
	idx := vs.NGroups()
	if idx >= 4 {
		panic("gpu.AddGroup: there is a hard limit of 4 on the number of VarGroups imposed by the WebGPU system, on Web platforms!")
	}
	vg := &VarGroup{Group: idx, Role: role, device: vs.device}
	if len(name) == 1 {
		vg.Name = name[0]
	}
	vg.alignBytes = 1
	if role == Uniform {
		vg.alignBytes = int(vs.sys.GPU().Limits.Limits.MinUniformBufferOffsetAlignment)
		// note: wgpu-native reports alignment sizes of 64
		// but then barfs when that is used.  256 seems to keep it happy
		vg.alignBytes = max(vg.alignBytes, 256)
	} else if role == Storage {
		vg.alignBytes = int(vs.sys.GPU().Limits.Limits.MinStorageBufferOffsetAlignment)
		vg.alignBytes = max(vg.alignBytes, 256)
	}
	vs.Groups[idx] = vg
	return vg
}

// VarByName returns Var by name in given group number,
// returning error if not found
func (vs *Vars) VarByName(group int, name string) (*Var, error) {
	vg, err := vs.Group(group)
	if err != nil {
		return nil, err
	}
	return vg.VarByName(name)
}

// ValueByName returns value by first looking up variable name, then value name,
// within given group number, returning error if not found
func (vs *Vars) ValueByName(group int, varName, valName string) (*Value, error) {
	vg, err := vs.Group(group)
	if err != nil {
		return nil, err
	}
	return vg.ValueByName(varName, valName)
}

// ValueByIndex returns value by first looking up variable name, then value index,
// returning error if not found
func (vs *Vars) ValueByIndex(group int, varName string, valIndex int) (*Value, error) {
	vg, err := vs.Group(group)
	if err != nil {
		return nil, err
	}
	return vg.ValueByIndex(varName, valIndex)
}

// SetCurrentValue sets the index of the current Value to use
// for given variable name, in given group number.
func (vs *Vars) SetCurrentValue(group int, name string, valueIndex int) (*Var, error) {
	vg, err := vs.Group(group)
	if err != nil {
		return nil, err
	}
	vr, err := vg.VarByName(name)
	if err != nil {
		return nil, err
	}
	vr.Values.SetCurrentValue(valueIndex)
	return vr, nil
}

// SetDynamicIndex sets the dynamic offset index for Value to use
// for given variable name, in given group number.
func (vs *Vars) SetDynamicIndex(group int, name string, dynamicIndex int) *Var {
	vr, err := vs.VarByName(group, name)
	if errors.Log(err) != nil {
		return nil
	}
	vr.Values.SetDynamicIndex(dynamicIndex)
	return vr
}

// Config must be called after all variables have been added.
// Configures all Groups and also does validation, returning error
// does DescLayout too, so all ready for Pipeline config.
func (vs *Vars) Config(dev *Device) error {
	ns := vs.NGroups()
	var cerr error
	vs.RoleMap = make(map[VarRoles][]*Var)
	for gi := vs.StartGroup(); gi < ns; gi++ {
		vg := vs.Groups[gi]
		if vg == nil {
			continue
		}
		err := vg.Config(dev)
		if err != nil {
			cerr = err
		}
		for ri, rl := range vg.RoleMap {
			vs.RoleMap[ri] = append(vs.RoleMap[ri], rl...)
		}
	}
	vs.bindLayout(dev)
	return cerr
}

// StringDoc returns info on variables
func (vs *Vars) StringDoc() string {
	ispc := 4
	var sb strings.Builder
	ns := vs.NGroups()
	for gi := vs.StartGroup(); gi < ns; gi++ {
		vg := vs.Groups[gi]
		if vg == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("Group: %d %s\n", vg.Group, vg.Name))

		for ri := Vertex; ri < VarRolesN; ri++ {
			rl, has := vg.RoleMap[ri]
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

// NGroups returns the number of regular non-VertexGroup groups
func (vs *Vars) NGroups() int {
	ex := 0
	if vs.hasVertex {
		ex++
	}
	if vs.hasPush {
		ex++
	}
	return len(vs.Groups) - ex
}

// StartGroup returns the starting group to use for iterating groups
func (vs *Vars) StartGroup() int {
	switch {
	case vs.hasVertex:
		return VertexGroup
	case vs.hasPush:
		return PushGroup
	default:
		return 0
	}
}

// Group returns group by index, returning nil and error if not found
func (vs *Vars) Group(group int) (*VarGroup, error) {
	vg, has := vs.Groups[group]
	if !has {
		err := fmt.Errorf("gpu.Vars:GroupTry gp number %d not found", group)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vg, nil
}

// VertexLayout returns WebGPU vertex layout, for VertexGroup only!
func (vs *Vars) VertexLayout() []wgpu.VertexBufferLayout {
	if vs.hasVertex {
		return vs.Groups[VertexGroup].vertexLayout()
	}
	return nil
}

/*
// VkPushConfig returns WebGPU push constant ranges, only if PushGroup used.
func (vs *Vars) VkPushConfig() []vk.PushConstantRange {
	if vs.hasPush {
		return vs.Groups[PushGroup].VkPushConfig()
	}
	return nil
}
*/

// bindLayout configures the Layouts slice of BindGroupLayouts
// for all of the non-Vertex vars
func (vs *Vars) bindLayout(dev *Device, used ...*Var) []*wgpu.BindGroupLayout {
	ngp := vs.NGroups()
	if ngp == 0 {
		vs.layouts = nil
		return nil
	}

	var lays []*wgpu.BindGroupLayout
	for gi := 0; gi < ngp; gi++ { // auto-skips vertex, push
		vg := vs.Groups[gi]
		if vg == nil {
			continue
		}
		vgl, err := vg.bindLayout(vs, used...)
		if err != nil {
			continue
		}
		lays = append(lays, vgl)
	}
	return lays
}
