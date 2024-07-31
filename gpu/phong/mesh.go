// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"fmt"
	"log"

	"cogentcore.org/core/gpu"
)

// Mesh records the number of elements in an indexed triangle mesh,
// which always includes normals and texture coordinates, and
// optionally per-vertex colors.
type Mesh struct {
	// number of vertex points, as math32.Vector3.
	// Always includes math32.Vector3 normals and math32.Vector2 texture coordinates
	NVertex int

	// number of indexes, as math32.ArrayU32
	NIndex int

	// has per-vertex colors, as math32.Vector4 per vertex
	HasColor bool
}

// ConfigMeshes configures vals for meshes -- this is the first of
// two passes in configuring and setting mesh values -- Phong.Sys.Config is
// called after this method (and everything else is configured).
func (ph *Phong) ConfigMeshes() {
	sy := ph.Sys
	nm := ph.Meshes.Len()
	vgp := sy.Vars.VertexGroup()
	vgp.SetNValues(nm)
	for i, kv := range ph.Meshes.Order {
		mv := kv.Value
		vgp.ValueByIndex("Pos", i).DynamicN = mv.NVertex
		vgp.ValueByIndex("Norm", i).DynamicN = mv.NVertex
		vgp.ValueByIndex("TexCoord", i).DynamicN = mv.NVertex
		vgp.ValueByIndex("Index", i).DynamicN = mv.NIndex
		vc := vgp.ValueByIndex("VertexColor", i)
		if mv.HasColor {
			vc.DynamicN = mv.NVertex
		} else {
			vc.DynamicN = 1
		}
	}
}

// ResetMeshes resets the meshes for reconfiguring
func (ph *Phong) ResetMeshes() {
	ph.Meshes.Reset()
}

// AddMesh adds a Mesh with name and given number of vertices, indexes,
// and optional per-vertex color
func (ph *Phong) AddMesh(name string, numVertex, nIndex int, hasColor bool) {
	ph.Meshes.Add(name, &Mesh{NVertex: numVertex, NIndex: nIndex, HasColor: hasColor})
}

// DeleteMesh deletes Mesh with name
func (ph *Phong) DeleteMesh(name string) {
	ph.Meshes.DeleteKey(name)
}

// UseMeshName selects mesh by name for current render step
// If mesh has per-vertex colors, these are selected for rendering,
// and texture is turned off.  UseTexture* after this to override.
func (ph *Phong) UseMeshName(name string) error {
	idx, ok := ph.Meshes.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("phong:UseMeshName -- name not found: %s", name)
		if gpu.Debug {
			log.Println(err)
		}
	}
	return ph.UseMeshIndex(idx)
}

// UseMeshIndex selects mesh by index for current render step.
// If mesh has per-vertex colors, these are selected for rendering,
// and texture is turned off.  UseTexture* after this to override.
func (ph *Phong) UseMeshIndex(idx int) error {
	mesh := ph.Meshes.ValueByIndex(idx)
	sy := ph.Sys
	sy.Vars.SetCurrentValue(gpu.VertexGroup, "Pos", idx)
	sy.Vars.SetCurrentValue(gpu.VertexGroup, "Norm", idx)
	sy.Vars.SetCurrentValue(gpu.VertexGroup, "TexCoord", idx)
	sy.Vars.SetCurrentValue(gpu.VertexGroup, "Index", idx)
	if mesh.HasColor {
		sy.Vars.SetCurrentValue(gpu.VertexGroup, "Color", idx)
		ph.Cur.UseVertexColor = true
		ph.Cur.UseTexture = false
	}
	return nil
}

// todo: SetMeshIndex
