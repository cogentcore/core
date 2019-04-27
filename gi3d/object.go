// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

// https://www.khronos.org/opengl/wiki/Vertex_Specification_Best_Practices

// Object represents an individual 3D object or object element.
// It has its own unique transforms, and a material and mesh structure.
// The material is fully responsible for rendering the object.
type Object struct {
	Node3DBase
	Mat      MatName   `desc:"name of the material used for rendering this object -- all materials are collected on the Scene"`
	Mesh     MeshName  `desc:"name of the mesh shape information used for rendering this object -- all meshes are collected on the Scene"`
	Surface  Surface   `desc:"properties of the surface (color, shininess, etc)"`
	Material *Material `view:"-" desc:"pointer to material"`
	Mesh     *Mesh     `view:"-" desc:"pointer to mesh"`
}

// update triggers recompute of Vtx, Norm from mesh, and then re-render of scene @ parent
// update needs to update all children too -- i.e., this level of scoping of updating is
// relevant, but gl-level render always needs to be complete re-render of entire scene..

// todo: compute vtx, bbox, etc

func (ob *Object) Render() {
	//
}
