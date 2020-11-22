// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gist"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// shapes define different standard mesh shapes

///////////////////////////////////////////////////////////////////////////
//   Plane

// Plane is a flat 2D plane, which can be oriented along any
// axis facing either positive or negative
type Plane struct {
	MeshBase
	NormAxis mat32.Dims  `desc:"axis along which the normal perpendicular to the plane points.  E.g., if the Y axis is specified, then it is a standard X-Z ground plane -- see also NormNeg for whether it is facing in the positive or negative of the given axis."`
	NormNeg  bool        `desc:"if false, the plane normal facing in the positive direction along specified NormAxis, otherwise it faces in the negative if true"`
	Size     mat32.Vec2  `desc:"2D size of plane"`
	Segs     mat32.Vec2i `desc:"number of segments to divide plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1"`
	Offset   float32     `desc:"offset from origin along direction of normal to the plane"`
}

var KiT_Plane = kit.Types.AddType(&Plane{}, nil)

// AddNewPlane adds Plane mesh to given scene,
// with given name and size, with its normal pointing
// by default in the positive Y axis (i.e., a "ground" plane).
// Offset is 0.
func AddNewPlane(sc *Scene, name string, width, height float32) *Plane {
	pl := &Plane{}
	pl.Nm = name
	pl.NormAxis = mat32.Y
	pl.NormNeg = false
	pl.Size.Set(width, height)
	pl.Segs.Set(1, 1)
	pl.Offset = 0
	sc.AddMesh(pl)
	return pl
}

func (pl *Plane) Make(sc *Scene) {
	pl.Reset()

	hSz := pl.Size.DivScalar(2)

	clr := gist.Color{}

	thin := float32(.0000001)
	sz := mat32.Vec3{}

	switch pl.NormAxis {
	case mat32.X:
		sz.Set(thin, hSz.Y, hSz.X)
		if pl.NormNeg {
			pl.AddPlane(mat32.Z, mat32.Y, 1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, -pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // nx
		} else {
			pl.AddPlane(mat32.Z, mat32.Y, -1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // px
		}
	case mat32.Y:
		sz.Set(hSz.X, thin, hSz.Y)
		if pl.NormNeg {
			pl.AddPlane(mat32.X, mat32.Z, 1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, -pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // ny
		} else {
			pl.AddPlane(mat32.X, mat32.Z, 1, 1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // py
		}
	case mat32.Z:
		sz.Set(hSz.X, hSz.Y, thin)
		if pl.NormNeg {
			pl.AddPlane(mat32.X, mat32.Y, -1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, -pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // nz
		} else {
			pl.AddPlane(mat32.X, mat32.Y, 1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // pz
		}
	}

	pl.BBox.SetBounds(sz.Negate(), sz)
}

///////////////////////////////////////////////////////////////////////////
//   Box

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

func (bx *Box) Make(sc *Scene) {
	bx.Reset()

	hSz := bx.Size.DivScalar(2)

	clr := gist.Color{}

	// start with neg z as typically back
	bx.AddPlane(mat32.X, mat32.Y, -1, -1, bx.Size.X, bx.Size.Y, -hSz.X, -hSz.Y, -hSz.Z, int(bx.Segs.X), int(bx.Segs.Y), clr) // nz
	bx.AddPlane(mat32.X, mat32.Z, 1, -1, bx.Size.X, bx.Size.Z, -hSz.X, -hSz.Z, -hSz.Y, int(bx.Segs.X), int(bx.Segs.Z), clr)  // ny
	bx.AddPlane(mat32.Z, mat32.Y, -1, -1, bx.Size.Z, bx.Size.Y, -hSz.Z, -hSz.Y, hSz.X, int(bx.Segs.Z), int(bx.Segs.Y), clr)  // px
	bx.AddPlane(mat32.Z, mat32.Y, 1, -1, bx.Size.Z, bx.Size.Y, -hSz.Z, -hSz.Y, -hSz.X, int(bx.Segs.Z), int(bx.Segs.Y), clr)  // nx
	bx.AddPlane(mat32.X, mat32.Z, 1, 1, bx.Size.X, bx.Size.Z, -hSz.X, -hSz.Z, hSz.Y, int(bx.Segs.X), int(bx.Segs.Z), clr)    // py
	bx.AddPlane(mat32.X, mat32.Y, 1, -1, bx.Size.X, bx.Size.Y, -hSz.X, -hSz.Y, hSz.Z, int(bx.Segs.X), int(bx.Segs.Y), clr)   // pz

	bx.BBox.SetBounds(hSz.Negate(), hSz)
}
