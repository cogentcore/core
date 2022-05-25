// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"sync"

	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
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
	Set(vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32)

	// Update updates the mesh points into given arrays, which have previously
	// been set with SetVerticies -- this can optimize by only updating whatever might
	// need to be updated for dynamically changing meshes.
	Update(vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32)

	// ComputeNorms automatically computes the normals from existing vertex data
	ComputeNorms(pos, norm mat32.ArrayF32)

	// HasColor returns true if this mesh has vertex-specific colors available
	HasColor() bool

	// IsTransparent returns true if this mesh has vertex-specific colors available
	// and at least some are transparent.
	IsTransparent() bool
}

// MeshBase provides the core implementation of Mesh interface
type MeshBase struct {
	Nm      string       `desc:"name of mesh -- meshes are linked to Solids by name so this matters"`
	NVtx    int          `desc:"number of vertex points, as mat32.Vec3 -- always includes mat32.Vec3 normals and mat32.Vec2 texture coordinates -- only valid after Sizes() has been called"`
	NIdx    int          `desc:"number of indexes, as mat32.ArrayU32 -- only valid after Sizes() has been called"`
	Color   bool         `desc:"has per-vertex colors, as mat32.Vec4 per vertex"`
	Dynamic bool         `desc:"if true, this mesh changes frequently -- otherwise considered to be static"`
	Trans   bool         `desc:"set to true if color has transparency -- not worth checking manually"`
	BBox    BBox         `desc:"computed bounding-box and other gross solid properties"`
	BBoxMu  sync.RWMutex `view:"-" copy:"-" json:"-" xml:"-" desc:"mutex on bbox access"`
}

var KiT_MeshBase = kit.Types.AddType(&MeshBase{}, nil)

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

func (ms *MeshBase) Update(vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	// nop: default mesh is static, not dynamic
}

// todo!!
func (ms *MeshBase) ComputeNorms(pos, norm mat32.ArrayF32) {
	// norm := mat32.Normal(vtxs[0], vtxs[1], vtxs[2])
}

////////////////////////////////////////////////////////////////////////
// Scene management

// AddMesh adds given mesh to mesh collection.  Any existing mesh of the
// same name is deleted.
// see AddNewX for convenience methods to add specific shapes
func (sc *Scene) AddMesh(ms Mesh) {
	sc.Meshes.Add(ms.Name(), ms)
}

/*
// AddMeshUniqe adds given mesh to mesh collection, ensuring that it has
// a unique name if one already exists.
func (sc *Scene) AddMeshUnique(ms Mesh) {
	nm := ms.Name()
	sc.MeshesInit()
	_, has := sc.Meshes[nm]
	if has {
		nm += fmt.Sprintf("_%d", len(sc.Meshes))
		ms.SetName(nm)
	}
	sc.Meshes[nm] = ms
}
*/

// MeshByName looks for mesh by name -- returns nil if not found
func (sc *Scene) MeshByName(nm string) Mesh {
	ms, ok := sc.Meshes.ValByKey(nm)
	if ok {
		return ms
	}
	return nil
}

// MeshByNameTry looks for mesh by name -- returns error if not found
func (sc *Scene) MeshByNameTry(nm string) (Mesh, error) {
	ms, ok := sc.Meshes.ValByKey(nm)
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
	tmp := AddNewPlane(sc, nm, 1, 1)
	tmp.NormAxis = mat32.Z
	tmp.NormNeg = false
	return tmp
}

// ConfigMeshes configures meshes for rendering
// must be called after adding or deleting any meshes or altering
// the number of verticies.
func (sc *Scene) ConfigMeshes() {
	ph := &sc.Phong
	ph.ResetMeshes()
	for _, kv := range sc.Meshes.Order {
		ms := kv.Val
		nVtx, nIdx, hasColor := ms.Sizes()
		ph.AddMesh(kv.Key, nVtx, nIdx, hasColor)
	}
	ph.AllocMeshes()
}

// SetMeshes sets the meshes after config
func (sc *Scene) SetMeshes() {
	ph := &sc.Phong
	for _, kv := range sc.Meshes.Order {
		ms := kv.Val
		vtxAry, normAry, texAry, clrAry, idxAry := ph.MeshFloatsByName(kv.Key)
		ms.Set(vtxAry, normAry, texAry, clrAry, idxAry)
		ph.ModMeshByName(kv.Key)
	}
	ph.Sync()
}

// ReconfigMeshes reconfigures meshes on the Phong renderer
// if there has been any change to the mesh structure.
// Init3D does a full configure of everything -- this is optimized
// just for mesh changes.
func (sc *Scene) ReconfigMeshes() {
	sc.ConfigMeshes()
	sc.Phong.Config()
	sc.SetMeshes()
}
