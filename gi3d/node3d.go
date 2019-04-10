// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/mat32"
	"github.com/goki/ki/ki"
)

// todo: rename gi.Viewport2D -> gi.Viewport

// Node3DBase is the basic 3D scenegraph node, which has the full transform information
// relative to parent, and computed bounding boxes, etc
type Node3DBase struct {
	NodeBase
	Pos         mat32.Vector3
	Scale       mat32.Vector3    // Node scale (relative to parent)
	Dir         mat32.Vector3    // Initial direction (relative to parent)
	Rot         mat32.Vector3    // Node rotation specified in Euler angles (relative to parent)
	Quat        mat32.Quaternion // Node rotation specified as a Quaternion (relative to parent)
	Xform       mat32.Matrix4    // Local transform matrix. Contains all position/rotation/scale information (relative to parent)
	WorldXform  mat32.Matrix4    // World transform matrix. Contains all absolute position/rotation/scale information (i.e. relative to very top parent, generally the scene)	Pos   mat32.Vector3
	BoundBox    mat32.Box3       // Last calculated bounding box
	BoundSphere mat32.Sphere     // Last calculated bounding sphere
	Area        float32          // Last calculated area
	Volume      float32          // Last calculated volume
	RotInertia  mat32.Matrix3    // Last calculated rotational inertia matrix

	Scene *Scene `json:"-" xml:"-" view:"-" desc:"our sceneviewport -- set in Init2D (Base typically) and used thereafter"`
}

type Node3D interface {
	// nodes are Ki elements -- this comes for free by embedding ki.Node in
	// all Node3D elements.
	ki.Ki
}
