// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"slices"
	"strconv"
	"sync"

	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
)

const (
	// MaxTextureLayers is the maximum number of layers per image
	MaxTextureLayers = 128

	// VertexGroup is the group number for Vertex and Index variables,
	// which have special treatment.
	VertexGroup = -2

	// PushGroup is the group number for Push Constants, which
	// do not appear in the BindGroupLayout and are managed separately.
	PushGroup = -1
)

// VarGroup contains a group of Var variables, accessed via @group number
// in shader code, with @binding allocated sequentially within group
// (or @location in the case of VertexGroup).
type VarGroup struct {
	// name is optional, for user reference, documentation
	Name string

	// variables in order
	Vars []*Var

	// map of vars by name; names must be unique
	VarMap map[string]*Var

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

	// number of variables with DynamicOffset set
	nDynamicOffsets int

	device Device

	// the alignment requirement in bytes for DynamicOffset variables.
	// This is 1 for Vertex buffer variables.
	alignBytes int

	// this counter updates when when a Values.current has been changed.
	// pipelines record the count that they last used to determine if time
	// to change. Use BindGroupUpdateCount() to access under lock.
	bindGroupUpdateCount int

	// mutex used for bindGroupsUpdateCount
	sync.Mutex
}

// addVar adds given variable
func (vg *VarGroup) addVar(vr *Var) {
	if vg.VarMap == nil {
		vg.VarMap = make(map[string]*Var)
	}
	vg.Vars = append(vg.Vars, vr)
	vg.VarMap[vr.Name] = vr
}

// Add adds a new variable of given type, role, arrayN, and shaders where used
func (vg *VarGroup) Add(name string, typ Types, arrayN int, shaders ...ShaderTypes) *Var {
	vr := NewVar(vg, name, typ, arrayN, shaders...)
	vg.addVar(vr)
	return vr
}

// AddStruct adds a new struct variable of given total number of bytes in size,
// type, role, set, and shaders where used
func (vg *VarGroup) AddStruct(name string, size int, arrayN int, shaders ...ShaderTypes) *Var {
	vr := NewVar(vg, name, Struct, arrayN, shaders...)
	vr.SizeOf = size
	vg.addVar(vr)
	return vr
}

// VarByName returns Var by name, returning error if not found
func (vg *VarGroup) VarByName(name string) (*Var, error) {
	vr, ok := vg.VarMap[name]
	if !ok {
		err := fmt.Errorf("Variable named %s not found", name)
		if Debug {
			errors.Log(err)
		}
		return nil, err
	}
	return vr, nil
}

// ValueByName returns value by first looking up variable name, then value name,
// returning error if not found.
func (vg *VarGroup) ValueByName(varName, valName string) (*Value, error) {
	vr, err := vg.VarByName(varName)
	if err != nil {
		return nil, err
	}
	vl, err := vr.Values.ValueByName(valName)
	return vl, err
}

// ValueByIndex returns value by first looking up variable name, then value index,
// returning error if not found.
func (vg *VarGroup) ValueByIndex(varName string, valIndex int) (*Value, error) {
	vr, err := vg.VarByName(varName)
	if err != nil {
		return nil, err
	}
	return vr.Values.ValueByIndex(valIndex)
}

// SetCurrentValue sets the index of the Current Value to use
// for given variable name.
func (vg *VarGroup) SetCurrentValue(name string, valueIndex int) (*Var, error) {
	vr, err := vg.VarByName(name)
	if err != nil {
		return nil, err
	}
	vr.Values.SetCurrentValue(valueIndex)
	return vr, nil
}

// SetDynamicIndex sets the dynamic offset index for Value to use
// for given variable name.
func (vg *VarGroup) SetDynamicIndex(name string, dynamicIndex int) (*Var, error) {
	vr, err := vg.VarByName(name)
	if err != nil {
		return nil, err
	}
	vr.Values.SetDynamicIndex(dynamicIndex)
	return vr, nil
}

// SetNValues sets all vars in this group to have specified
// number of Values.
func (vg *VarGroup) SetNValues(nvals int) {
	for _, vr := range vg.Vars {
		vr.SetNValues(&vg.device, nvals)
	}
}

// SetAllCurrentValue sets the Current Value index, which is
// the Value that will be used in rendering, via BindGroup,
// for all vars in group.
func (vg *VarGroup) SetAllCurrentValue(i int) {
	for _, vr := range vg.Vars {
		vr.Values.SetCurrentValue(i)
	}
}

// ValuesUpdated is called whenever values have been updated in a way
// that requires new a new bind group to be computed.
func (vg *VarGroup) ValuesUpdated() {
	vg.Lock()
	vg.bindGroupUpdateCount++
	vg.Unlock()
}

// ValuesUpdated is called whenever values have been updated in a way
// that requires new a new bind group to be computed.
func (vg *VarGroup) BindGroupUpdateCount() int {
	vg.Lock()
	defer vg.Unlock()
	return vg.bindGroupUpdateCount
}

// Config must be called after all variables have been added.
// Configures binding / location for all vars based on sequential order.
// also does validation and returns error message.
func (vg *VarGroup) Config(dev *Device) error {
	if vg.Name == "" {
		switch vg.Group {
		case VertexGroup:
			vg.Name = "VertexGroup"
		case PushGroup:
			vg.Name = "PushGroup"
		default:
			vg.Name = fmt.Sprintf("Group%d", vg.Group)
		}
	}
	vg.device = *dev
	vg.RoleMap = make(map[VarRoles][]*Var)
	var errs []error
	bnum := 0
	for _, vr := range vg.Vars {
		if vg.Group == VertexGroup && vr.Role > Index {
			err := fmt.Errorf("gpu.VarGroup:Config VertexGroup cannot contain variables of role: %s  var: %s", vr.Role.String(), vr.Name)
			errs = append(errs, err)
			errors.Log(err)
			continue
		}
		if vg.Group >= 0 && vr.Role <= Index {
			err := errors.New("gpu.VarGroup:Config Vertex or Index Vars must be located in a VertexGroup!  Use AddVertexGroup() method instead of AddGroup()")
			errs = append(errs, err)
			errors.Log(err)
		}
		rl := vg.RoleMap[vr.Role]
		rl = append(rl, vr)
		vg.RoleMap[vr.Role] = rl
		if vr.Role == Index && len(rl) > 1 {
			err := fmt.Errorf("gpu.VarGroup:Config VertexGroup should not contain multiple Index variables: %v", rl)
			errs = append(errs, err)
			errors.Log(err)
		}
		if vr.Role > Storage && (len(vg.RoleMap[Uniform]) > 0 || len(vg.RoleMap[Storage]) > 0) {
			err := fmt.Errorf("gpu.VarGroup:Config Group with dynamic Uniform or Storage variables should not contain static variables (e.g., textures): %s", vr.Role.String())
			errs = append(errs, err)
			errors.Log(err)
		}
		if vr.Role != Index { // index doesn't count
			vr.Binding = bnum
			bnum++
		}
		if vr.Role == Vertex && vr.Type == Float32Matrix4 { // special case
			bnum += 3
		}
		if vr.Role == SampledTexture { // sampler too
			bnum++
		}
	}
	return errors.Join(errs...)
}

// Release destroys infrastructure for Group, Vars and Values.
func (vg *VarGroup) Release() {
	for _, vr := range vg.Vars {
		vr.Release()
	}
}

// bindLayout creates the BindGroupLayout for given group.
// Only for non-VertexGroup sets.
// Must have set NValuesPer for any SampledTexture vars,
// which require separate descriptors per.
func (vg *VarGroup) bindLayout(vs *Vars, used ...*Var) (*wgpu.BindGroupLayout, error) {
	var binds []wgpu.BindGroupLayoutEntry

	vg.nDynamicOffsets = 0
	// https://eliemichel.github.io/LearnWebGPU/basic-3d-rendering/shader-uniforms/dynamic-uniforms.html
	// https://toji.dev/webgpu-best-practices/bind-groups.html
	for _, vr := range vg.Vars {
		if len(used) > 0 && !slices.Contains(used, vr) {
			continue
		}
		if vr.Role == Vertex || vr.Role == Index { // shouldn't happen
			continue
		}
		bd := wgpu.BindGroupLayoutEntry{
			Binding:    uint32(vr.Binding),
			Visibility: vr.shaders,
		}
		switch {
		case vr.Role == SampledTexture:
			binds = append(binds, wgpu.BindGroupLayoutEntry{
				Binding:    uint32(vr.Binding),
				Visibility: vr.shaders,
				Texture: wgpu.TextureBindingLayout{
					Multisampled:  false,
					ViewDimension: wgpu.TextureViewDimension2D, // todo:
					SampleType:    wgpu.TextureSampleTypeFloat,
				},
			})
			bd.Binding = uint32(vr.Binding + 1)
			bd.Sampler = wgpu.SamplerBindingLayout{
				Type: wgpu.SamplerBindingTypeFiltering,
			}
		default:
			bd.Buffer = wgpu.BufferBindingLayout{
				Type:             vr.bindingType(),
				HasDynamicOffset: false,
				MinBindingSize:   0, // 0 is fine
			}
			if vr.DynamicOffset {
				bd.Buffer.HasDynamicOffset = true
				vg.nDynamicOffsets++
			}
		}
		binds = append(binds, bd)
	}

	// fmt.Println(reflectx.StringJSON(binds))

	bgld := wgpu.BindGroupLayoutDescriptor{
		Label:   strconv.Itoa(vg.Group),
		Entries: binds,
	}

	bgl, err := vg.device.Device.CreateBindGroupLayout(&bgld)
	if errors.Log(err) != nil {
		return nil, err
	}
	return bgl, nil
}

// dynamicOffsets returns any current dynamic offsets for this group
func (vg *VarGroup) dynamicOffsets(used ...*Var) []uint32 {
	var do []uint32
	if vg.Role == Vertex || vg.Role == SampledTexture {
		return do
	}
	curDynIdx := 0
	for _, vr := range vg.Vars {
		if len(used) > 0 && !slices.Contains(used, vr) {
			continue
		}
		if vr.DynamicOffset {
			do = append(do, vr.Values.dynamicOffset())
			curDynIdx++
		}
	}
	return do
}

// bindGroup returns the Current Value bindings for all variables
// within this Group, subject to the used filter if present.
// This determines what Values of the Vars the current Render
// actions will use. Only for non-VertexGroup groups.
func (vg *VarGroup) bindGroup(vs *Vars, used ...*Var) (*wgpu.BindGroup, error) {
	bgl, err := vg.bindLayout(vs, used...)
	if err != nil {
		return nil, err
	}
	defer bgl.Release()

	var bgs []wgpu.BindGroupEntry
	for _, vr := range vg.Vars {
		if len(used) > 0 && !slices.Contains(used, vr) {
			continue
		}
		bgs = append(bgs, vr.bindGroupEntry()...)
	}
	bg, err := vg.device.Device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout:  bgl,
		Entries: bgs,
		Label:   vg.Name,
	})
	if errors.Log(err) != nil {
		return nil, err
	}
	return bg, nil
}

// IndexVar returns the Index variable within this VertexGroup.
// returns nil if not found.
func (vg *VarGroup) IndexVar() *Var {
	n := len(vg.Vars)
	for i := n - 1; i >= 0; i-- { // typically at end, go in reverse
		vr := vg.Vars[i]
		if vr.Role == Index {
			return vr
		}
	}
	return nil
}

// vertexLayout returns the VertexBufferLayout based on Vertex role
// variables within this VertexGroup.
// Note: there is no support for interleaved arrays
// so each location is sequential number, recorded in var Binding
func (vg *VarGroup) vertexLayout() []wgpu.VertexBufferLayout {
	var vbls []wgpu.VertexBufferLayout
	for _, vr := range vg.Vars {
		if vr.Role != Vertex { // not Index
			continue
		}
		stepMode := wgpu.VertexStepModeVertex
		if vr.VertexInstance {
			stepMode = wgpu.VertexStepModeInstance
		}
		if vr.Type == Float32Matrix4 {
			vbls = append(vbls, wgpu.VertexBufferLayout{
				ArrayStride: uint64(vr.SizeOf),
				StepMode:    stepMode,
				Attributes: []wgpu.VertexAttribute{
					{
						Offset:         0,
						ShaderLocation: uint32(vr.Binding),
						Format:         Float32Vector4.VertexFormat(),
					},
					{
						Offset:         4,
						ShaderLocation: uint32(vr.Binding + 1),
						Format:         Float32Vector4.VertexFormat(),
					},
					{
						Offset:         8,
						ShaderLocation: uint32(vr.Binding + 2),
						Format:         Float32Vector4.VertexFormat(),
					},
					{
						Offset:         12,
						ShaderLocation: uint32(vr.Binding + 3),
						Format:         Float32Vector4.VertexFormat(),
					},
				},
			})
		} else {
			vbls = append(vbls, wgpu.VertexBufferLayout{
				ArrayStride: uint64(vr.SizeOf),
				StepMode:    stepMode,
				Attributes: []wgpu.VertexAttribute{
					{
						Offset:         0,
						ShaderLocation: uint32(vr.Binding),
						Format:         vr.Type.VertexFormat(),
					},
				},
			})
		}
	}
	return vbls
}
