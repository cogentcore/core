// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/g3n/engine/math32"
	"github.com/goki/ki/ki"
)

// todo: rename gi.Viewport2D -> gi.Viewport

// Node3DBase is the basic 3D scenegraph node, which has the full transform information
// relative to parent, and computed bounding boxes, etc
type Node3DBase struct {
	NodeBase
	Pos         math32.Vector3
	Scale       math32.Vector3    // Node scale (relative to parent)
	Dir         math32.Vector3    // Initial direction (relative to parent)
	Rot         math32.Vector3    // Node rotation specified in Euler angles (relative to parent)
	Quat        math32.Quaternion // Node rotation specified as a Quaternion (relative to parent)
	Xform       math32.Matrix4    // Local transform matrix. Contains all position/rotation/scale information (relative to parent)
	WorldXform  math32.Matrix4    // World transform matrix. Contains all absolute position/rotation/scale information (i.e. relative to very top parent, generally the scene)	Pos   math32.Vector3
	BoundBox    math32.Box3       // Last calculated bounding box
	BoundSphere math32.Sphere     // Last calculated bounding sphere
	Area        float32           // Last calculated area
	Volume      float32           // Last calculated volume
	RotInertia  math32.Matrix3    // Last calculated rotational inertia matrix

	Scene *Scene `json:"-" xml:"-" view:"-" desc:"our sceneviewport -- set in Init2D (Base typically) and used thereafter"`
}

type Node3D interface {
	// nodes are Ki elements -- this comes for free by embedding ki.Node in
	// all Node3D elements.
	ki.Ki
}

// Group collects individual elements in a scene but does not have geometry of
// its own.  It does have a transform that applies to all nodes under it.
type Group struct {
	Node3DBase
}

// https://www.khronos.org/opengl/wiki/Vertex_Specification_Best_Practices

// Object represents an individual 3D object or object element.
// It has its own unique transforms, and a material and mesh structure.
// The material is fully responsible for rendering the object.
type Object struct {
	Node3DBase
	Mat   MatName         `desc:"name of the material used for rendering this object -- all materials are collected on the Scene"`
	Mesh  MeshName        `desc:"name of the mesh shape information used for rendering this object -- all meshes are collected on the Scene"`
	Vtx   math32.ArrayF32 `desc:"computed verticies from Mesh that have WorldXform transform applied so they are ready for global rendering"`
	Norm  math32.ArrayF32 `desc:"computed normals from Mesh that have WorldXform transform applied so they are ready for global rendering"`
	Color math32.ArrayF32 `desc:"optional per-vertex colors for this object -- can otherwise use Mesh's shared Colors or a Uniform color"`
}

// update triggers recompute of Vtx, Norm from mesh, and then re-render of scene @ parent
// update needs to update all children too -- i.e., this level of scoping of updating is
// relevant, but gl-level render always needs to be complete re-render of entire scene..

// todo: compute vtx, bbox, etc
