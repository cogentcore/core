// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/go-gl/mathgl/mgl32"
)

// defines a region in 2D space
type Region2D struct {
	Offset mgl32.Vec2
	Size   mgl32.Vec2
}

// defines how a region in 2D space is mapped
type RegionMap2D struct {
	Target  Region2D   `desc:"target region to render into (e.g., in RenderPlane)"`
	Rescale mgl32.Vec2 `desc:"rescaling that we provide for children nodes into the Target Size region -- we present region that is at Offset lower-left corner, of size Size * Rescale size"`
	Size    mgl32.Vec2 `desc:"Our overall size: Target.Size * Rescale"`
}

// 3D transform
type Transform3D struct {
	Transform   mgl32.Mat4 // overall compiled transform
	Scale       mgl32.Vec3
	Translation mgl32.Vec3
	Orientation mgl32.Quat
}

// todo common ops on 3D
