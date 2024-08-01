// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"fmt"
	"log"

	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
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

	//	buffers for mesh data, for shape.Set() method
	vertexArray, normArray, textureArray math32.ArrayF32
	indexArray                           math32.ArrayU32
}

// ResetMeshes resets the meshes for reconfiguring
func (ph *Phong) ResetMeshes() {
	ph.Lock()
	defer ph.Unlock()

	ph.meshes.Reset()
	vgp := ph.Sys.Vars.VertexGroup()
	vgp.SetNValues(1)
}

// AddMesh adds a Mesh with name and given number of vertices, indexes,
// and optional per-vertex color
func (ph *Phong) AddMesh(name string, nVertex, nIndex int, hasColor bool) *Mesh {
	ph.Lock()
	defer ph.Unlock()

	ph.meshes.Add(name, &Mesh{NVertex: nVertex, NIndex: nIndex, HasColor: hasColor})
	return ph.meshes.Order[ph.meshes.Len()-1].Value
}

// AddMeshFromShape adds a Mesh using the shape.Shape interface for the source
// of the mesh data, and sets the values directly.  Nothing further needs to
// be done to configure this mesh after calling this.
func (ph *Phong) AddMeshFromShape(name string, sh shape.Shape) {
	ph.Lock()
	defer ph.Unlock()

	nVertex, nIndex := sh.N()
	ph.meshes.Add(name, &Mesh{NVertex: nVertex, NIndex: nIndex})
	mv := ph.meshes.Order[ph.meshes.Len()-1].Value

	mv.vertexArray = slicesx.SetLength(mv.vertexArray, nVertex*3)
	mv.normArray = slicesx.SetLength(mv.normArray, nVertex*3)
	mv.textureArray = slicesx.SetLength(mv.textureArray, nVertex*2)
	mv.indexArray = slicesx.SetLength(mv.indexArray, nIndex)
	sh.Set(mv.vertexArray, mv.normArray, mv.textureArray, mv.indexArray)

	nm := ph.meshes.Len()
	vgp := ph.Sys.Vars.VertexGroup()
	vgp.SetNValues(nm) // add to all vars
	idx := nm - 1
	ph.configMesh(mv, idx)

	gpu.SetValueFrom(vgp.ValueByIndex("Pos", idx), mv.vertexArray)
	gpu.SetValueFrom(vgp.ValueByIndex("Norm", idx), mv.normArray)
	gpu.SetValueFrom(vgp.ValueByIndex("TexCoord", idx), mv.textureArray)
	gpu.SetValueFrom(vgp.ValueByIndex("Index", idx), mv.indexArray)
}

// DeleteMesh deletes Mesh with name
func (ph *Phong) DeleteMesh(name string) {
	ph.Lock()
	defer ph.Unlock()

	ph.meshes.DeleteKey(name)
}

// UseMeshName selects mesh by name for current render step
// If mesh has per-vertex colors, these are selected for rendering,
// and texture is turned off.  UseTexture* after this to override.
func (ph *Phong) UseMeshName(name string) error {
	idx, ok := ph.meshes.IndexByKeyTry(name)
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
	ph.Lock()
	defer ph.Unlock()

	sy := ph.Sys
	mesh := ph.meshes.ValueByIndex(idx)
	sy.Vars.SetCurrentValue(gpu.VertexGroup, "Pos", idx)
	sy.Vars.SetCurrentValue(gpu.VertexGroup, "Norm", idx)
	sy.Vars.SetCurrentValue(gpu.VertexGroup, "TexCoord", idx)
	sy.Vars.SetCurrentValue(gpu.VertexGroup, "Index", idx)
	if mesh.HasColor {
		sy.Vars.SetCurrentValue(gpu.VertexGroup, "Color", idx)
		ph.UseVertexColor = true
		ph.UseTexture = false
	} else {
		ph.UseVertexColor = false
	}
	return nil
}

// SetMeshName sets mesh vertex values, by mesh name.
func (ph *Phong) SetMeshName(name string) error {
	ph.Lock()
	defer ph.Unlock()

	idx, ok := ph.meshes.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("phong:UseMeshName -- name not found: %s", name)
		if gpu.Debug {
			log.Println(err)
		}
	}
	return ph.UseMeshIndex(idx)
}

// configMeshes configures values for all current meshes.
// If not using AddMeshFromShape then values need to be set
// manually after this.  Otherwise, they should be good to go.
func (ph *Phong) configMeshes() {
	nm := ph.meshes.Len()
	vgp := ph.Sys.Vars.VertexGroup()
	vgp.SetNValues(nm)
	for i, kv := range ph.meshes.Order {
		mv := kv.Value
		ph.configMesh(mv, i)
	}
}

func (ph *Phong) configMesh(mv *Mesh, idx int) {
	vgp := ph.Sys.Vars.VertexGroup()
	vgp.ValueByIndex("Pos", idx).DynamicN = mv.NVertex
	vgp.ValueByIndex("Norm", idx).DynamicN = mv.NVertex
	vgp.ValueByIndex("TexCoord", idx).DynamicN = mv.NVertex
	vgp.ValueByIndex("Index", idx).DynamicN = mv.NIndex
	// vc := vgp.ValueByIndex("VertexColor", idx)
	// if mv.HasColor {
	// 	vc.DynamicN = mv.NVertex
	// } else {
	// 	vc.DynamicN = 1
	// }
}
