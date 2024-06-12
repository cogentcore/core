// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/vgpu/vshape"
)

// MeshName is a [Mesh] name. This type provides an automatic GUI chooser for meshes.
// It is used on [Solid] to link to meshes by name.
type MeshName string

// Mesh parametrizes the mesh-based shape used for rendering a [Solid].
// Only indexed triangle meshes are supported.
// All Meshes must know in advance the number of vertex and index points
// they require, and the SetVertices method operates on data from the
// vgpu staging buffer to set the relevant data post-allocation.
// The vgpu vshape library is used for all basic shapes, and it follows
// this same logic.
// Per-vertex Color is optional, as is the ability to update the data
// after initial SetVertices call (default is to do nothing).
type Mesh interface {

	// AsMeshBase returns the [MeshBase] for this Mesh,
	// which provides the core functionality of a mesh.
	AsMeshBase() *MeshBase

	// Sizes returns the number of vertex and index elements required for this mesh
	// including a bool representing whether it has per-vertex color.
	Sizes() (numVertex, numIndex int, hasColor bool)

	// Set sets the mesh points into given arrays, which have been allocated
	// according to the Sizes() returned by this Mesh.
	// The mesh is automatically marked with SetMod so that does not need to be done here.
	Set(sc *Scene, vertexArray, normArray, textureArray, colorArray math32.ArrayF32, indexArray math32.ArrayU32)

	// Update updates the mesh points into given arrays, which have previously
	// been set with SetVertices; this can optimize by only updating whatever might
	// need to be updated for dynamically changing meshes.
	// You must call SetMod if the mesh was actually updated at this point.
	Update(sc *Scene, vertexArray, normArray, textureArray, colorArray math32.ArrayF32, indexArray math32.ArrayU32)
}

// MeshBase provides the core implementation of the [Mesh] interface.
type MeshBase struct { //types:add -setters

	// Name is the name of the mesh. [Mesh]es are linked to [Solid]s
	// by name so this matters.
	Name string

	// NumVertex is the number of [math32.Vector3] vertex points. This always
	// includes [math32.Vector3] normals and [math32.Vector2] texture coordinates.
	// This is only valid after [Mesh.Sizes] has been called.
	NumVertex int `set:"-"`

	// NumIndex is the number of [math32.ArrayU32] indexes.
	// This is only valid after [Mesh.Sizes] has been called.
	NumIndex int `set:"-"`

	// HasColor is whether the mesh has per-vertex colors
	// as [math32.Vector4] per vertex.
	HasColor bool

	// Dynamic is whether this mesh changes frequently;
	// otherwise considered to be static.
	Dynamic bool

	// Transparent is whether the color has transparency;
	// not worth checking manually. This is only valid if
	// [MeshBase.HasColor] is true.
	Transparent bool

	// BBox has the computed bounding-box and other gross solid properties.
	BBox BBox `set:"-"`
}

func (ms *MeshBase) AsMeshBase() *MeshBase {
	return ms
}

func (ms *MeshBase) Sizes() (numVertex, numIndex int, hasColor bool) {
	return ms.NumVertex, ms.NumIndex, ms.HasColor
}

func (ms *MeshBase) Update(sc *Scene, vertextureArray, normArray, texArray, colorArray math32.ArrayF32, indexArray math32.ArrayU32) {
	// nop: default mesh is static, not dynamic
}

func (ms *MeshBase) SetMod(sc *Scene) {
	sc.Phong.ModMeshByName(ms.Name)
}

// todo!!
func (ms *MeshBase) ComputeNorms(pos, norm math32.ArrayF32) {
	// norm := math32.Normal(vtxs[0], vtxs[1], vtxs[2])
}

////////////////////////////////////////////////////////////////////////
// Scene management

// AddMesh adds given mesh to mesh collection.  Any existing mesh of the
// same name is deleted.
// see NewX for convenience methods to add specific shapes
func (sc *Scene) AddMesh(ms Mesh) {
	sc.Meshes.Add(ms.AsMeshBase().Name, ms)
	sc.SetFlag(true, ScNeedsConfig)
}

// AddMeshUniqe adds given mesh to mesh collection, ensuring that it has
// a unique name if one already exists.
func (sc *Scene) AddMeshUnique(ms Mesh) {
	nm := ms.AsMeshBase().Name
	_, err := sc.MeshByNameTry(nm)
	if err == nil {
		nm += fmt.Sprintf("_%d", sc.Meshes.Len())
		ms.AsMeshBase().SetName(nm)
	}
	sc.Meshes.Add(ms.AsMeshBase().Name, ms)
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
	return nil, fmt.Errorf("Mesh named: %v not found in Scene: %v", nm, sc.Name)
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
	tmp.NormAxis = math32.Z
	tmp.NormNeg = false
	return tmp
}

// ConfigMeshes configures meshes for rendering
// must be called after adding or deleting any meshes or altering
// the number of vertices.
func (sc *Scene) ConfigMeshes() {
	ph := &sc.Phong
	ph.UpdateMu.Lock()
	ph.ResetMeshes()
	for _, kv := range sc.Meshes.Order {
		ms := kv.Value
		numVertex, nIndex, hasColor := ms.Sizes()
		ph.AddMesh(kv.Key, numVertex, nIndex, hasColor)
	}
	ph.ConfigMeshes()
	ph.UpdateMu.Unlock()
}

// SetMeshes sets the meshes after config
func (sc *Scene) SetMeshes() {
	ph := &sc.Phong
	ph.UpdateMu.Lock()
	for _, kv := range sc.Meshes.Order {
		ms := kv.Value
		vertexArray, normArray, textureArray, colorArray, indexArray := ph.MeshFloatsByName(kv.Key)
		ms.Set(sc, vertexArray, normArray, textureArray, colorArray, indexArray)
		ph.ModMeshByName(kv.Key)
	}
	ph.UpdateMu.Unlock()
	ph.Sync()
}

// UpdateMeshes iterates over meshes and calls their Update method
// each mesh Update method must call SetMod to trigger the update
func (sc *Scene) UpdateMeshes() {
	ph := &sc.Phong
	ph.UpdateMu.Lock()
	for _, kv := range sc.Meshes.Order {
		ms := kv.Value
		vertexArray, normArray, textureArray, colorArray, indexArray := ph.MeshFloatsByName(kv.Key)
		ms.Update(sc, vertexArray, normArray, textureArray, colorArray, indexArray)
	}
	ph.UpdateMu.Unlock()
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
	Vertex  math32.ArrayF32
	Norm    math32.ArrayF32
	Texture math32.ArrayF32
	Color   math32.ArrayF32
	Index   math32.ArrayU32
}

func (ms *GenMesh) Sizes() (numVertex, nIndex int, hasColor bool) {
	ms.NumVertex = len(ms.Vertex) / 3
	ms.NumIndex = len(ms.Index)
	ms.HasColor = len(ms.Color) > 0
	return ms.NumVertex, ms.NumIndex, ms.HasColor
}

func (ms *GenMesh) Set(sc *Scene, vertexArray, normArray, textureArray, colorArray math32.ArrayF32, indexArray math32.ArrayU32) {
	copy(vertexArray, ms.Vertex)
	copy(normArray, ms.Norm)
	copy(textureArray, ms.Texture)
	if ms.HasColor {
		copy(colorArray, ms.Color)
	}
	copy(indexArray, ms.Index)
	bb := vshape.BBoxFromVtxs(ms.Vertex, 0, ms.NumVertex)
	ms.BBox.SetBounds(bb.Min, bb.Max)
}
