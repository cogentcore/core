// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"fmt"
	"log"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

// Mesh records the number of elements in an indexed triangle mesh,
// which always includes normals and texture coordinates, and
// optionally per-vertex colors.
type Mesh struct {
	NVtx     int  `desc:"number of vertex points, as mat32.Vec3 -- always includes mat32.Vec3 normals and mat32.Vec2 texture coordinates"`
	NIdx     int  `desc:"number of indexes, as mat32.ArrayU32"`
	HasColor bool `desc:"has per-vertex colors, as mat32.Vec4 per vertex"`
}

// AllocMeshes allocates vals for meshes
func (ph *Phong) AllocMeshes() {
	nm := ph.Meshes.Len()
	vars := ph.Sys.Vars()
	vset := vars.VertexSet()
	vset.ConfigVals(nm)
	for i, mesh := range ph.Meshes.Order {
		mv := mesh.Val
		_, vp, _ := vset.ValByIdxTry("Pos", i)
		vp.N = mv.NVtx
		_, vn, _ := vset.ValByIdxTry("Norm", i)
		vn.N = mv.NVtx
		_, vt, _ := vset.ValByIdxTry("Tex", i)
		vt.N = mv.NVtx
		_, vi, _ := vset.ValByIdxTry("Index", i)
		vi.N = mv.NIdx
		_, vc, _ := vset.ValByIdxTry("Color", i)
		if mv.HasColor {
			vc.N = mesh.Val.NVtx
		} else {
			vc.N = 1 // todo: should be 0
		}
	}
}

// ConfigMeshes configures the rendering for the meshes
func (ph *Phong) ConfigMeshes() {
	// vars := ph.Sys.Vars()
	// vset := vars.VertexSet()
}

// AddMesh adds a Mesh with name and given number of verticies, indexes,
// and optional per-vertex color
func (ph *Phong) AddMesh(name string, nVtx, nIdx int, hasColor bool) {
	ph.Meshes.Add(name, &Mesh{NVtx: nVtx, NIdx: nIdx, HasColor: hasColor})
}

// UseMeshName selects mesh by name for current render step
// If mesh has per-vertex colors, these are selected for rendering,
// and texture is turned off.  UseTexture* after this to override.
func (ph *Phong) UseMeshName(name string) error {
	idx, ok := ph.Meshes.IdxByKey(name)
	if !ok {
		err := fmt.Errorf("vphong:UseMeshName -- name not found: %s", name)
		if vgpu.TheGPU.Debug {
			log.Println(err)
		}
	}
	return ph.UseMeshIdx(idx)
}

// UseMeshIdx selects mesh by index for current render step.
// If mesh has per-vertex colors, these are selected for rendering,
// and texture is turned off.  UseTexture* after this to override.
func (ph *Phong) UseMeshIdx(idx int) error {
	mesh := ph.Meshes.ValByIdx(idx)
	vars := ph.Sys.Vars()
	vars.BindVertexValIdx("Pos", idx)
	vars.BindVertexValIdx("Norm", idx)
	vars.BindVertexValIdx("Tex", idx)
	vars.BindVertexValIdx("Index", idx)
	if mesh.HasColor {
		vars.BindVertexValIdx("Color", idx)
		ph.Cur.UseVtxColor = true
		ph.Cur.UseTexture = false
	}
	return nil
}

// MeshFloatsByName returns the mat32.ArrayF32's and mat32.ArrayU32 for given mesh
// for assigning values to the mesh.
// Must call ModMeshByName after setting these values to mark as modified.
func (ph *Phong) MeshFloatsByName(name string) (pos, norm, tex, clr mat32.ArrayF32, idx mat32.ArrayU32) {
	i, ok := ph.Meshes.IdxByKey(name)
	if !ok {
		err := fmt.Errorf("vphong:UseMeshName -- name not found: %s", name)
		if vgpu.TheGPU.Debug {
			log.Println(err)
		}
	}
	return ph.MeshFloatsByIdx(i)
}

// MeshFloatsByIdx returns the mat32.ArrayF32's and mat32.ArrayU32 for given mesh
// for assigning values to the mesh.
// Must call ModMeshByIdx after setting these values to mark as modified.
func (ph *Phong) MeshFloatsByIdx(i int) (pos, norm, tex, clr mat32.ArrayF32, idx mat32.ArrayU32) {
	vars := ph.Sys.Vars()
	vset := vars.VertexSet()
	_, vp, _ := vset.ValByIdxTry("Pos", i)
	_, vn, _ := vset.ValByIdxTry("Norm", i)
	_, vt, _ := vset.ValByIdxTry("Tex", i)
	_, vi, _ := vset.ValByIdxTry("Index", i)
	_, vc, _ := vset.ValByIdxTry("Color", i)
	return vp.Floats32(), vn.Floats32(), vt.Floats32(), vc.Floats32(), vi.UInts32()
}

// ModMeshByName marks given mesh by name as modified.
// Must call after modifying mesh values, to mark for syncing
func (ph *Phong) ModMeshByName(name string) {
	i, ok := ph.Meshes.IdxByKey(name)
	if !ok {
		err := fmt.Errorf("vphong:UseMeshName -- name not found: %s", name)
		if vgpu.TheGPU.Debug {
			log.Println(err)
		}
	}
	ph.ModMeshByIdx(i)
}

// ModMeshByIdx marks given mesh by index as modified.
// Must call after modifying mesh values, to mark for syncing
func (ph *Phong) ModMeshByIdx(i int) {
	vars := ph.Sys.Vars()
	vset := vars.VertexSet()
	nms := []string{"Pos", "Norm", "Tex", "Index", "Color"}
	for _, nm := range nms {
		_, vl, _ := vset.ValByIdxTry(nm, i)
		vl.SetMod()
	}
}
