// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"

	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
)

// MeshName is a [Mesh] name. This type provides an automatic GUI chooser for meshes.
// It is used on [Solid] to link to meshes by name.
type MeshName string

// Mesh parametrizes the mesh-based shape used for rendering a [Solid],
// using the [shape.Mesh] interface for basic shape data.
// Only indexed triangle meshes are supported.
// All Meshes must know in advance the number of vertex and index points
// they require, and the Set method writes the mesh data to arrays of
// appropriate vector data.
// Per-vertex Color is optional.
type Mesh interface {
	shape.Mesh

	// AsMeshBase returns the [MeshBase] for this Mesh,
	// which provides the core functionality of a mesh.
	AsMeshBase() *MeshBase
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

func (ms *MeshBase) MeshSize() (numVertex, numIndex int, hasColor bool) {
	return ms.NumVertex, ms.NumIndex, ms.HasColor
}

func (ms *MeshBase) MeshBBox() math32.Box3 {
	return ms.BBox.BBox
}

func (ms *MeshBase) Offsets() (vtxOffset, idxOffset int) {
	return 0, 0
}

func (ms *MeshBase) SetOffsets(vtxOffset, idxOffset int) {
	// nop
}

// todo!!
func (ms *MeshBase) ComputeNorms(pos, norm math32.ArrayF32) {
	// norm := math32.Normal(vtxs[0], vtxs[1], vtxs[2])
}

////////////////////////////////////////////////////////////////////////
// Scene management

// SetMesh sets / updates the given mesh, updating any existing
// mesh of the same name.
// See NewX for convenience methods to add specific shapes.
func (sc *Scene) SetMesh(ms Mesh) {
	name := ms.AsMeshBase().Name
	sc.Meshes.Add(name, ms) // does replace
	if sc.IsLive() {
		sc.Phong.SetMesh(name, ms)
	}
}

// setAllMeshes is called when the Phong system first is activated.
func (sc *Scene) setAllMeshes() {
	ph := sc.Phong
	for _, kv := range sc.Meshes.Order {
		ms := kv.Value
		ph.SetMesh(kv.Key, ms)
	}
}

// AddMeshUniqe adds given mesh, ensuring that it has
// a unique name if one already exists.
// This is used e.g., in loading external files which may not
// obey this constraint.
func (sc *Scene) AddMeshUnique(ms Mesh) {
	nm := ms.AsMeshBase().Name
	_, err := sc.MeshByNameTry(nm)
	if err == nil {
		nm += fmt.Sprintf("_%d", sc.Meshes.Len())
		ms.AsMeshBase().SetName(nm)
	}
	sc.SetMesh(ms)
}

// MeshByName looks for mesh by name, returning nil if not found.
func (sc *Scene) MeshByName(nm string) Mesh {
	ms, ok := sc.Meshes.ValueByKeyTry(nm)
	if ok {
		return ms
	}
	return nil
}

// MeshByNameTry looks for mesh by name, returning error if not found.
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

// ResetMeshes removes all meshes.
// Use this to remove unused meshes after significant update.
// Note that there is no Delete mechanism because the GPU Phong
// system does not support it for efficiency reasons.
func (sc *Scene) ResetMeshes() {
	if sc.IsLive() {
		sc.Phong.ResetMeshes()
	}
	sc.Meshes.Reset()
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
	tmp.NormNeg = true
	return tmp
}

///////////////////////////////////////////////////////////////
// GenMesh

// GenMesh is a generic, arbitrary Mesh, storing its values
type GenMesh struct {
	MeshBase
	Vertex   math32.ArrayF32
	Normal   math32.ArrayF32
	TexCoord math32.ArrayF32
	Color    math32.ArrayF32
	Index    math32.ArrayU32
}

func (ms *GenMesh) MeshSize() (numVertex, nIndex int, hasColor bool) {
	ms.NumVertex = len(ms.Vertex) / 3
	ms.NumIndex = len(ms.Index)
	ms.HasColor = len(ms.Color) > 0
	return ms.NumVertex, ms.NumIndex, ms.HasColor
}

func (ms *GenMesh) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	copy(vertex, ms.Vertex)
	copy(normal, ms.Normal)
	copy(texcoord, ms.TexCoord)
	if ms.HasColor {
		copy(clrs, ms.Color)
	}
	copy(index, ms.Index)
	bb := shape.BBoxFromVtxs(ms.Vertex, 0, ms.NumVertex)
	ms.BBox.SetBounds(bb.Min, bb.Max)
}
