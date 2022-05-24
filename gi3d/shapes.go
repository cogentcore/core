// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/vgpu/vshape"
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

func (pl *Plane) Sizes() (nVtx, nIdx int, hasColor bool) {
	pl.NVtx, pl.NIdx = vshape.PlaneN(int(pl.Segs.X), int(pl.Segs.Y))
	pl.Color = false
	return pl.NVtx, pl.NIdx, pl.Color
}

// Set sets points in given allocated arrays
func (pl *Plane) Set(vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	pos := mat32.Vec3{}
	sz := vshape.SetPlaneAxisSize(vtxAry, normAry, texAry, idxAry, 0, 0, pl.NormAxis, pl.NormNeg, pl.Size, pl.Segs, pl.Offset, pos)
	mn := pos.Sub(sz)
	mx := pos.Add(sz)
	pl.BBox.SetBounds(mn, mx)
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

func (bx *Box) Sizes() (nVtx, nIdx int, hasColor bool) {
	bx.NVtx, bx.NIdx = vshape.BoxN(bx.Segs)
	bx.Color = false
	return bx.NVtx, bx.NIdx, bx.Color
}

// SetBox sets points in given allocated arrays
func (bx *Box) Set(vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	pos := mat32.Vec3{}
	hSz := vshape.SetBox(vtxAry, normAry, texAry, idxAry, 0, 0, bx.Size, bx.Segs, pos)
	mn := pos.Sub(hSz)
	mx := pos.Add(hSz)
	bx.BBox.SetBounds(mn, mx)
}
