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
	Pose        Pose          // complete specification of position and orientation
	BoundBox    mat32.Box3    // Last calculated bounding box
	BoundSphere mat32.Sphere  // Last calculated bounding sphere
	Area        float32       // Last calculated area
	Volume      float32       // Last calculated volume
	RotInertia  mat32.Matrix3 // Last calculated rotational inertia matrix

	Scene *Scene `json:"-" xml:"-" view:"-" desc:"our sceneviewport -- set in Init2D (Base typically) and used thereafter"`
}

type Node3D interface {
	// nodes are Ki elements -- this comes for free by embedding ki.Node in
	// all Node3D elements.
	ki.Ki
}
