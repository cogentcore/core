// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"sync"

	"cogentcore.org/core/mat32"
	"cogentcore.org/core/vgpu/vshape"
)

// MeshName is a mesh name -- provides an automatic gui chooser for meshes.
// Used on Solid to link to meshes by name.
type MeshName string

// Mesh parameterizes the mesh-based shape used for rendering a Solid.
// Only indexed triangle meshes are supported.
// All Mesh's must know in advance the number of vertex and index points
// they require, and the SetVerticies method operates on data from the
// vgpu staging buffer to set the relevant data post-allocation.
// The vgpu vshape library is used for all basic shapes, and it follows
// this same logic.
// Per-vertex Color is optional, as is the ability to update the data
// after initial SetVerticies call (default is to do nothing).
type Mesh interface {
	// Name returns name of the mesh
	Name() string

	// SetName sets the name of the mesh
	SetName(nm string)

	// AsMeshBase returns the MeshBase for this Mesh
	AsMeshBase() *MeshBase

	// Sizes returns the number of vertex and index elements required for this mesh
	// including a bool representing whether it has per-vertex color.
	Sizes() (nVtx, nIdx int, hasColor bool)

	// Set sets the mesh points into given arrays, which have been allocated
	// according to the Sizes() returned by this Mesh.
	// The mesh is automatically marked with SetMod so that does not need to be done here.
	Set(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32)

	// Update updates the mesh points into given arrays, which have previously
	// been set with SetVerticies -- this can optimize by only updating whatever might
	// need to be updated for dynamically changing meshes.
	// You must call SetMod if the mesh was actually updated at this point.
	Update(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32)

	// SetMod flags that the mesh data has been modified and will be sync'd
	// at next sync of the Scene.Phong render system.
	SetMod(sc *Scene)

	// ComputeNorms automatically computes the normals from existing vertex data
	ComputeNorms(pos, norm mat32.ArrayF32)

	// HasColor returns true if this mesh has vertex-specific colors available
	HasColor() bool

	// IsTransparent returns true if this mesh has vertex-specific colors available
	// and at least some are transparent.
	IsTransparent() bool
}

// MeshBase provides the core implementation of Mesh interface
type MeshBase struct { //gti:add -setters

	// name of mesh -- meshes are linked to Solids by name so this matters
	Nm string `set:"-"`

	// number of vertex points, as mat32.Vec3 -- always includes mat32.Vec3 normals and mat32.Vec2 texture coordinates -- only valid after Sizes() has been called
	NVtx int `set:"-"`

	// number of indexes, as mat32.ArrayU32 -- only valid after Sizes() has been called
	NIdx int `set:"-"`

	// has per-vertex colors, as mat32.Vec4 per vertex
	Color bool

	// if true, this mesh changes frequently -- otherwise considered to be static
	Dynamic bool

	// set to true if color has transparency -- not worth checking manually
	Trans bool

	// computed bounding-box and other gross solid properties
	BBox BBox `set:"-"`

	// mutex on bbox access
	BBoxMu sync.RWMutex `view:"-" copier:"-" json:"-" xml:"-" set:"-"`
}

func (ms *MeshBase) Name() string                           { return ms.Nm }
func (ms *MeshBase) SetName(nm string)                      { ms.Nm = nm }
func (ms *MeshBase) AsMeshBase() *MeshBase                  { return ms }
func (ms *MeshBase) HasColor() bool                         { return ms.Color }
func (ms *MeshBase) Sizes() (nVtx, nIdx int, hasColor bool) { return ms.NVtx, ms.NIdx, ms.Color }

func (ms *MeshBase) IsTransparent() bool {
	if !ms.HasColor() {
		return false
	}
	return ms.Trans
}

func (ms *MeshBase) Update(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	// nop: default mesh is static, not dynamic
}

func (ms *MeshBase) SetMod(sc *Scene) {
	sc.Phong.ModMeshByName(ms.Nm)
}

// todo!!
func (ms *MeshBase) ComputeNorms(pos, norm mat32.ArrayF32) {
	// norm := mat32.Normal(vtxs[0], vtxs[1], vtxs[2])
}

////////////////////////////////////////////////////////////////////////
// Scene management

// AddMesh adds given mesh to mesh collection.  Any existing mesh of the
// same name is deleted.
// see NewX for convenience methods to add specific shapes
func (sc *Scene) AddMesh(ms Mesh) {
	sc.Meshes.Add(ms.Name(), ms)
	sc.SetFlag(true, ScNeedsConfig)
}

// AddMeshUnique adds given mesh to mesh collection, ensuring that it has
// a unique name if one already exists.
func (sc *Scene) AddMeshUnique(ms Mesh) {
	nm := ms.Name()
	_, err := sc.MeshByNameTry(nm)
	if err == nil {
		nm += fmt.Sprintf("_%d", sc.Meshes.Len())
		ms.SetName(nm)
	}
	sc.Meshes.Add(ms.Name(), ms)
	sc.SetFlag(true, ScNeedsConfig)
}

// MeshByName looks for mesh by name -- returns nil if not found
func (sc *Scene) MeshByName(nm string) Mesh {
	ms, ok := sc.Meshes.ValueByKeyTry(nm)
	if ok {
		return ms
	}
	return nil
}

// MeshByNameTry looks for mesh by name -- returns error if not found
func (sc *Scene) MeshByNameTry(nm string) (Mesh, error) {
	ms, ok := sc.Meshes.ValueByKeyTry(nm)
	if ok {
		return ms, nil
	}
	return nil, fmt.Errorf("Mesh named: %v not found in Scene: %v", nm, sc.Nm)
}

// MeshList returns a list of available meshes (e.g., for chooser)
func (sc *Scene) MeshList() []string {
	return sc.Meshes.Keys()
}

// DeleteMesh removes given mesh -- returns error if mesh not found.
func (sc *Scene) DeleteMesh(nm string) {
	sc.Meshes.DeleteKey(nm)
}

// DeleteMeshes removes all meshes
func (sc *Scene) DeleteMeshes() {
	sc.Phong.Meshes.Reset()
}

// PlaneMesh2D returns the special Plane mesh used for Text2D and Embed2D
// (creating it if it does not yet exist).
// This is a 1x1 plane with a normal pointing in +Z direction.
func (sc *Scene) PlaneMesh2D() Mesh {
	nm := Plane2DMeshName
	tm, err := sc.MeshByNameTry(nm)
	if err == nil {
		return tm
	}
	tmp := NewPlane(sc, nm, 1, 1)
	tmp.NormAxis = mat32.Z
	tmp.NormNeg = false
	return tmp
}

// ConfigMeshes configures meshes for rendering
// must be called after adding or deleting any meshes or altering
// the number of verticies.
func (sc *Scene) ConfigMeshes() {
	ph := &sc.Phong
	ph.UpdtMu.Lock()
	ph.ResetMeshes()
	for _, kv := range sc.Meshes.Order {
		ms := kv.Value
		nVtx, nIdx, hasColor := ms.Sizes()
		ph.AddMesh(kv.Key, nVtx, nIdx, hasColor)
	}
	ph.ConfigMeshes()
	ph.UpdtMu.Unlock()
}

// SetMeshes sets the meshes after config
func (sc *Scene) SetMeshes() {
	ph := &sc.Phong
	ph.UpdtMu.Lock()
	for _, kv := range sc.Meshes.Order {
		ms := kv.Value
		vtxAry, normAry, texAry, clrAry, idxAry := ph.MeshFloatsByName(kv.Key)
		ms.Set(sc, vtxAry, normAry, texAry, clrAry, idxAry)
		ph.ModMeshByName(kv.Key)
	}
	ph.UpdtMu.Unlock()
	ph.Sync()
}

// UpdateMeshes iterates over meshes and calls their Update method
// each mesh Update method must call SetMod to trigger the update
func (sc *Scene) UpdateMeshes() {
	ph := &sc.Phong
	ph.UpdtMu.Lock()
	for _, kv := range sc.Meshes.Order {
		ms := kv.Value
		vtxAry, normAry, texAry, clrAry, idxAry := ph.MeshFloatsByName(kv.Key)
		ms.Update(sc, vtxAry, normAry, texAry, clrAry, idxAry)
	}
	ph.UpdtMu.Unlock()
	ph.Sync()
}

// ReconfigMeshes reconfigures meshes on the Phong renderer
// if there has been any change to the mesh structure.
// Config does a full configure of everything -- this is optimized
// just for mesh changes.
func (sc *Scene) ReconfigMeshes() {
	sc.ConfigMeshes()
	sc.Phong.Config()
	sc.SetMeshes()

}

///////////////////////////////////////////////////////////////
// GenMesh

// GenMesh is a generic, arbitrary Mesh, storing its values
type GenMesh struct {
	MeshBase
	Vtx  mat32.ArrayF32
	Norm mat32.ArrayF32
	Tex  mat32.ArrayF32
	Clr  mat32.ArrayF32
	Idx  mat32.ArrayU32
}

func (ms *GenMesh) Sizes() (nVtx, nIdx int, hasColor bool) {
	ms.NVtx = len(ms.Vtx) / 3
	ms.NIdx = len(ms.Idx)
	ms.Color = len(ms.Clr) > 0
	return ms.NVtx, ms.NIdx, ms.Color
}

func (ms *GenMesh) Set(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	copy(vtxAry, ms.Vtx)
	copy(normAry, ms.Norm)
	copy(texAry, ms.Tex)
	if ms.Color {
		copy(clrAry, ms.Clr)
	}
	copy(idxAry, ms.Idx)
	bb := vshape.BBoxFromVtxs(ms.Vtx, 0, ms.NVtx)
	ms.BBoxMu.Lock()
	ms.BBox.SetBounds(bb.Min, bb.Max)
	ms.BBoxMu.Unlock()
}
