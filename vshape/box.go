// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import "github.com/goki/mat32"

// Box is a rectangular-shaped solid (cuboid)
type Box struct {
	ShapeBase
	Size mat32.Vec3  `desc:"size along each dimension"`
	Segs mat32.Vec3i `desc:"number of segments to divide each plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1"`
}

// NewBox returns a Box shape with given size
func NewBox(width, height, depth float32) *Box {
	bx := &Box{}
	bx.Defaults()
	bx.Size.Set(width, height, depth)
	return bx
}

func (bx *Box) Defaults() {
	bx.Size.Set(1, 1, 1)
	bx.Segs.Set(1, 1, 1)
}

func (bx *Box) N() (nVtx, nIdx int) {
	nVtx, nIdx = PlaneSize(int(bx.Segs.X), int(bx.Segs.Y))
	nVtx *= 6
	nIdx *= 6
	return
}

// Set sets points in given allocated arrays
func (bx *Box) Set(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	hSz := bx.Size.DivScalar(2)

	nVtx, nIdx := PlaneSize(int(bx.Segs.X), int(bx.Segs.Y))

	voff := bx.VtxOff
	ioff := bx.IdxOff

	// start with neg z as typically back
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.X, mat32.Y, -1, -1, bx.Size.X, bx.Size.Y, -hSz.X, -hSz.Y, -hSz.Z, int(bx.Segs.X), int(bx.Segs.Y)) // nz
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.X, mat32.Z, 1, -1, bx.Size.X, bx.Size.Z, -hSz.X, -hSz.Z, -hSz.Y, int(bx.Segs.X), int(bx.Segs.Z)) // ny
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.Z, mat32.Y, -1, -1, bx.Size.Z, bx.Size.Y, -hSz.Z, -hSz.Y, hSz.X, int(bx.Segs.Z), int(bx.Segs.Y)) // px
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.Z, mat32.Y, 1, -1, bx.Size.Z, bx.Size.Y, -hSz.Z, -hSz.Y, -hSz.X, int(bx.Segs.Z), int(bx.Segs.Y)) // nx
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.X, mat32.Z, 1, 1, bx.Size.X, bx.Size.Z, -hSz.X, -hSz.Z, hSz.Y, int(bx.Segs.X), int(bx.Segs.Z)) // py
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.X, mat32.Y, 1, -1, bx.Size.X, bx.Size.Y, -hSz.X, -hSz.Y, hSz.Z, int(bx.Segs.X), int(bx.Segs.Y)) // pz

	mn := hSz.Negate()
	bx.CBBox.Set(&mn, &hSz)
}
