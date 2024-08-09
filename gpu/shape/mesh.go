// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shape

import (
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/math32"
)

// Mesh is an interface for specifying the triangle Mesh data for 3D shapes.
// This is used for all mesh / shape data in the phong system.
type Mesh interface {
	// MeshSize returns number of vertex, index points in this shape element,
	// and whether it has per-vertex color values.
	MeshSize() (numVertex, nIndex int, hasColor bool)

	// Set sets points in given allocated arrays.
	Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32)

	// Offsets returns starting offset for vertices, indexes in full shape array,
	// in terms of points, not floats.
	Offsets() (vtxOffset, idxOffset int)

	// SetOffsets sets starting offset for vertices, indexes in full shape array,
	// in terms of points, not floats.
	SetOffsets(vtxOffset, idxOffset int)

	// MeshBBox returns the bounding box for the shape vertex points,
	// typically centered around 0.
	// This is only valid after Set has been called.
	MeshBBox() math32.Box3
}

// MeshData provides storage buffers for shapes specified using the [Mesh]
// interface.
type MeshData struct {
	// number of vertex points allocated for the Vertex, Normal, TexCoord,
	// and optional Colors data (if HasColor).
	NumVertex int

	// number of indexes.
	NumIndex int

	// whether this mech has per-vertex colors, as math32.Vector4 per vertex.
	HasColor bool

	// MeshBBox is the bounding box for the shape vertex points,
	// typically centered around 0.
	MeshBBox math32.Box3

	//	buffers that hold mesh data for the [Mesh.Set] method.
	Vertex, Normal, TexCoord, Colors math32.ArrayF32
	Index                            math32.ArrayU32
}

// NewMeshData returns a new MeshData and sets our buffer data
// from [Mesh] interface.
func NewMeshData(mesh Mesh) *MeshData {
	md := &MeshData{}
	return md.Set(mesh)
}

// Set sets mesh data into our buffers, from [shape.Mesh].
func (md *MeshData) Set(mesh Mesh) *MeshData {
	md.NumVertex, md.NumIndex, md.HasColor = mesh.MeshSize()

	md.Vertex = slicesx.SetLength(md.Vertex, md.NumVertex*3)
	md.Normal = slicesx.SetLength(md.Normal, md.NumVertex*3)
	md.TexCoord = slicesx.SetLength(md.TexCoord, md.NumVertex*2)
	md.Index = slicesx.SetLength(md.Index, md.NumIndex)
	if md.HasColor {
		md.Colors = slicesx.SetLength(md.Colors, md.NumVertex*4)
	}
	mesh.Set(md.Vertex, md.Normal, md.TexCoord, md.Colors, md.Index)
	md.MeshBBox = mesh.MeshBBox()
	return md
}
