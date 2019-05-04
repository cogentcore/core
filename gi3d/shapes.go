// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/ki/kit"
)

// shapes define different standard mesh shapes

// Box is a rectangular-shaped solid (cuboid)
type Box struct {
	MeshBase
	Size mat32.Vec3  `desc:"size along each dimension"`
	Segs mat32.Vec3i `desc:"number of segments to divide each plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1"`
}

var KiT_Box = kit.Types.AddType(&Box{}, nil)

// AddNewBox adds Box mesh to given scene, with given name and size
func AddNewBox(sc *Scene, name string, width, height, depth float32) *Box {
	bx := &Box{}
	bx.Nm = name
	bx.Size.Set(width, height, depth)
	bx.Segs.Set(1, 1, 1)
	sc.AddMesh(bx)
	return bx
}

func (bx *Box) Make() {
	bx.Reset()

	hSz := bx.Size.DivScalar(2)

	clr := gi.Color{}

	// start with neg z as typically back
	bx.AddPlane(mat32.X, mat32.Y, -1, -1, bx.Size.X, bx.Size.Y, -hSz.X, -hSz.Y, -hSz.Z, int(bx.Segs.X), int(bx.Segs.Y), clr) // nz
	bx.AddPlane(mat32.X, mat32.Z, 1, -1, bx.Size.X, bx.Size.Z, -hSz.X, -hSz.Z, -hSz.Y, int(bx.Segs.X), int(bx.Segs.Z), clr)  // ny
	bx.AddPlane(mat32.Z, mat32.Y, -1, -1, bx.Size.Z, bx.Size.Y, -hSz.Z, -hSz.Y, hSz.X, int(bx.Segs.Z), int(bx.Segs.Y), clr)  // px
	bx.AddPlane(mat32.Z, mat32.Y, 1, -1, bx.Size.Z, bx.Size.Y, -hSz.Z, -hSz.Y, -hSz.X, int(bx.Segs.Z), int(bx.Segs.Y), clr)  // nx
	bx.AddPlane(mat32.X, mat32.Z, 1, 1, bx.Size.X, bx.Size.Z, -hSz.X, -hSz.Z, hSz.Y, int(bx.Segs.X), int(bx.Segs.Z), clr)    // py
	bx.AddPlane(mat32.X, mat32.Y, 1, -1, bx.Size.X, bx.Size.Y, -hSz.X, -hSz.Y, hSz.Z, int(bx.Segs.X), int(bx.Segs.Y), clr)   // pz

	bx.BBox.BBox.Min = hSz.Negate()
	bx.BBox.BBox.Max = hSz

	bx.BBox.BSphere.Radius = hSz.Length()
	bx.BBox.Area = 2*bx.Size.X + 2*bx.Size.Y + 2*bx.Size.Z
	bx.BBox.Volume = bx.Size.X * bx.Size.Y * bx.Size.Z
}
