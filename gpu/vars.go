// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"log"
	"strings"

	"cogentcore.org/core/base/indent"

	"github.com/rajveermalviya/go-webgpu/wgpu"
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
	vg := &VarGroup{Group: VertexGroup}
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
	vg := &VarGroup{Group: PushGroup}
	vs.Groups[PushGroup] = vg
	vs.hasPush = true
	return vg
}

// PushGroup returns the Push Group -- a special Group holding push constants
func (vs *Vars) PushGroup() *VarGroup {
	return vs.Groups[PushGroup]
}

// AddGroup adds a new non-Vertex Group for holding Uniforms, Storage, etc
// Groups are automatically numbered sequentially
func (vs *Vars) AddGroup() *VarGroup {
	if vs.Groups == nil {
		vs.Groups = make(map[int]*VarGroup)
	}
	idx := vs.NGroups()
	vg := &VarGroup{Group: idx}
	vs.Groups[idx] = vg
	return vg
}

// VarByNameTry returns Var by name in given set number,
// returning error if not found
func (vs *Vars) VarByNameTry(set int, name string) (*Var, error) {
	vg, err := vs.GroupTry(set)
	if err != nil {
		return nil, err
	}
	return vg.VarByNameTry(name)
}

// ValueByNameTry returns value by first looking up variable name, then value name,
// within given set number, returning error if not found
func (vs *Vars) ValueByNameTry(set int, varName, valName string) (*Var, *Value, error) {
	vg, err := vs.GroupTry(set)
	if err != nil {
		return nil, nil, err
	}
	return vg.ValueByNameTry(varName, valName)
}

// ValueByIndexTry returns value by first looking up variable name, then value index,
// returning error if not found
func (vs *Vars) ValueByIndexTry(set int, varName string, valIndex int) (*Var, *Value, error) {
	vg, err := vs.GroupTry(set)
	if err != nil {
		return nil, nil, err
	}
	return vg.ValueByIndexTry(varName, valIndex)
}

// Config must be called after all variables have been added.
// Configures all Groups and also does validation, returning error
// does DescLayout too, so all ready for Pipeline config.
func (vs *Vars) Config(dev *Device) error {
	ns := vs.NGroups()
	var cerr error
	vs.RoleMap = make(map[VarRoles][]*Var)
	for si := vs.StartGroup(); si < ns; si++ {
		vg := vs.Groups[si]
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
	for si := vs.StartGroup(); si < ns; si++ {
		vg := vs.Groups[si]
		if vg == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("Group: %d\n", vg.Group))

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

// NGroups returns the number of regular non-VertexGroup sets
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

// StartGroup returns the starting set to use for iterating sets
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

// GroupTry returns set by index, returning nil and error if not found
func (vs *Vars) GroupTry(set int) (*VarGroup, error) {
	vg, has := vs.Groups[set]
	if !has {
		err := fmt.Errorf("gpu.Vars:GroupTry set number %d not found", set)
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

///////////////////////////////////////////////////////////////////
// Binding, Layouts

// bindLayout configures the Layouts slice of BindGroupLayouts
// for all of the non-Vertex vars
func (vs *Vars) bindLayout(dev *Device) []*wgpu.BindGroupLayout {
	nset := vs.NGroups()
	if nset == 0 {
		vs.layouts = nil
		return nil
	}

	var lays []*wgpu.BindGroupLayout
	for si := 0; si < nset; si++ { // auto-skips vertex, push
		vg := vs.Groups[si]
		if vg == nil {
			continue
		}
		vg.bindLayout(vs)
		lays = append(lays, vg.layout)
	}
	vs.layouts = lays
	return lays
}

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
	// vg := vs.Groups[VertexGroup]
	// vr, vl, err := vg.ValueByNameTry(varNm, valNm)
	// if err != nil {
	// 	return err
	// }
	// vr.BindValueIndex[vs.BindDescIndex] = vl.Index // this is then consumed by draw command
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
	// vg := vs.Groups[VertexGroup]
	// vr, vl, err := vg.ValueByIndexTry(varNm, valIndex)
	// if err != nil {
	// 	return err
	// }
	// vr.BindValueIndex[vs.BindDescIndex] = vl.Index // this is then consumed by draw command
	return nil
}
