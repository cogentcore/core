// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/g3n/engine/math32"
	"github.com/goki/gi/oswin/gpu"
)

// todo: need to be able to trigger update based just on updating colors (e.g. netview render)

// MeshName is a mesh name -- provides an automatic gui chooser for meshes
type MeshName string

// Mesh holds the mesh-based shape used for rendering an object.
// The object always refers back to this structure for the indexes, colors, and UV elements
// All vertex data is stored in Vectors structure, computed from mesh with WorldXform
// applied, with core vertex, normal interleaved: [XYZ,NNN].. then texture coords
// pasted next so they can be efficiently copied directly from the mesh [UV]..
// and finally optional per-vertex color data appended at the end.
// There is only one type of Mesh: indexed triangle.
type Mesh struct {
	Name    string
	Vtx     math32.ArrayF32 `desc:"verticies for triangle shapes that make up the mesh -- all mesh structures must use indexed triangle meshes"`
	Norm    math32.ArrayF32 `desc:"computed normals for each vertex"`
	Idx     math32.ArrayU32 `desc:"indexes that sequentially in groups of 3 define the actual triangle faces"`
	Color   math32.ArrayF32 `desc:"if per-vertex color material type is used for this mesh, then these are the per-vertex colors -- may not be defined in which case per-vertex materials are not possible for such meshes"`
	TexUV   math32.ArrayF32 `desc:"texture U,V coordinates for mapping textures onto vertexes"`
	Vectors gpu.BufferMgr   `desc:"computed verticies, normals, indices, etc for rendering"`
}

// todo: lots of methods for rendering different standard geometries into meshes
// no need to make those different actual Mesh objects -- can just be methods and they
// can add to mesh data -- include reset method to start fresh -- so it becomes
// a bit of a flexible rendering library
