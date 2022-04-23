// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import "github.com/goki/ki/kit"

// SetLoc is the descriptor set and location indexes for a variable
type SetLoc struct {
	Set int `desc:"descriptor set"`
	Loc int `desc:"location within set"`
}

// Var specifies a variable used in a pipeline, but does not manage
// actual values / storage -- see Val for that.
// A Var represents a type of input or output into the GPU program,
// including things like Vertex arrays, transformation matricies (Uniforms),
// Images (Textures), and arbitrary Structs for Compute shaders.
type Var struct {
	Name   string   `desc:"variable name"`
	Type   Types    `desc:"type"`
	Role   VarRoles `desc:"role of variable"`
	Loc    SetLoc   `desc:"descriptor set location for variable"`
	SizeOf int      `desc:"size in bytes of one element (not array size)"`
}

// Set sets the main values
func (vr *Var) Set(name string, typ Types, role VarRoles, set int) {
	vr.Name = name
	vr.Type = typ
	vr.Role = role
	vr.Loc.Set = set
	vr.SizeOf = TypeSizes[typ]
}

//////////////////////////////////////////////////////////////////

// Vars are all the variables that are used by a pipeline.
// Vars are allocated to locations within Sets in order added.
type Vars struct {
	Vars   []*Var          `desc:"all variables"`
	VarMap map[string]*Var `desc:"map of all vars -- names must be unique"`
	Sets   [][]*Var        `desc:"allocation of variables to sets, and locations within sets"`
}

// AddVar adds a new variable
func (vs *Vars) AddVar(vr *Var) {
	if vs.VarMap == nil {
		vs.VarMap = make(map[string]*Var)
	}
	vs.Vars = append(vs.Vars, vr)
	vs.VarMap[vr.Name] = vr
}

// Add adds a new variable
func (vs *Vars) Add(name string, typ Types, role VarRoles, set int) {
	vr := &Var{}
	vr.Set(name, typ, role, set)
	vs.AddVar(vr)
}

// AddStruct adds a new struct variable
func (vs *Vars) AddStruct(name string, size int, role VarRoles, set int) {
	vr := &Var{}
	vr.Set(name, Struct, role, set)
	vr.SizeOf = size
	vs.AddVar(vr)
}

// AllocSets allocates variables to sets
func (vs *Vars) AllocSets() {
}

// Sets the descriptor layout info for all the variables
func (vs *Vars) DescriptoLayout() {
	vs.AllocSets()
}

//////////////////////////////////////////////////////////////////

// VarRoles are the functional roles of variables.
type VarRoles int32

const (
	UndefVarRole VarRoles = iota
	VertexInput
	VertexOutput // is this needed?
	Indexes
	UniformVar
	StorageVar
	ImageVar
	VarRolesN
)

//go:generate stringer -type=VarRoles

var KiT_VarRoles = kit.Enums.AddEnum(VarRolesN, kit.NotBitFlag, nil)
