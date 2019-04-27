// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi"
	"github.com/goki/gi/mat32"
)

// shapes define different standard mesh shapes

// Box is a rectangular-shaped solid (cuboid)
type Box struct {
	MeshBase
	Size mat32.Vec3  `desc:"size along each dimension"`
	Segs mat32.Vec3i `desc:"number of segments to divide each plane into (enforced to be at least 1)"`
}

func (bx *Box) Make() {
	bx.Reset()

	halfSz := bx.Size
	halfSz.DivideScalar(2)

	clr := gi.Color{}

	// start with neg z as typically back
	bx.AddPlane(mat32.X, mat32.Y, -1, -1, bx.Size.X, bx.Size.Y, -halfSz.Z, bx.Segs.X, bx.Segs.Y, clr) // nz
	bx.AddPlane(mat32.Z, mat32.Y, -1, -1, bx.Size.Z, bx.Size.Y, halfSz.X, bx.Segs.Z, bx.Segs.Y, clr)  // px
	bx.AddPlane(mat32.Z, mat32.Y, 1, -1, bx.Size.Z, bx.Size.Y, -halfSz.X, bx.Segs.Z, bx.Segs.Y, clr)  // nx
	bx.AddPlane(mat32.X, mat32.Z, 1, 1, bx.Size.X, bx.Size.Z, halfSz.Y, bx.Segs.X, bx.Segs.Z, clr)    // py
	bx.AddPlane(mat32.X, mat32.Z, 1, -1, bx.Size.X, bx.Size.Z, -halfSz.Y, bx.Segs.X, bx.Segs.Z, clr)  // ny
	bx.AddPlane(mat32.X, mat32.Y, 1, -1, bx.Size.X, bx.Size.Y, halfSz.Z, bx.Segs.X, bx.Segs.Y, clr)   // pz
}
