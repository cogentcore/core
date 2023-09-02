// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import "goki.dev/mat32/v2"

// Box is a rectangular-shaped solid (cuboid)
type Box struct {
	ShapeBase

	// size along each dimension
	Size mat32.Vec3 `desc:"size along each dimension"`

	// number of segments to divide each plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1
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
	nVtx, nIdx = BoxN(bx.Segs)
	return
}

// SetBox sets points in given allocated arrays
func (bx *Box) Set(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	hSz := SetBox(vtxAry, normAry, texAry, idxAry, bx.VtxOff, bx.IdxOff, bx.Size, bx.Segs, bx.Pos)

	mn := bx.Pos.Sub(hSz)
	mx := bx.Pos.Add(hSz)
	bx.CBBox.Set(&mn, &mx)
}

// PlaneN returns the N's for a single plane's worth of
// vertex and index data with given number of segments.
// Note: In *vertex* units, not float units (i.e., x3 to get
// actual float offset in Vtx array).
func BoxN(segs mat32.Vec3i) (nVtx, nIdx int) {
	nv, ni := PlaneN(int(segs.X), int(segs.Y))
	nVtx += 2 * nv
	nIdx += 2 * ni
	nv, ni = PlaneN(int(segs.X), int(segs.Z))
	nVtx += 2 * nv
	nIdx += 2 * ni
	nv, ni = PlaneN(int(segs.Z), int(segs.Y))
	nVtx += 2 * nv
	nIdx += 2 * ni
	return
}

// SetBox sets box vertex, norm, tex, index data at
// given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Idx index.
// for given 3D size, and given number of segments per side.
// finely subdividing a plane allows for higher-quality lighting
// and texture rendering (minimum of 1 will be enforced).
// pos is a 3D position offset. returns 3D size of plane.
func SetBox(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32, vtxOff, idxOff int, size mat32.Vec3, segs mat32.Vec3i, pos mat32.Vec3) mat32.Vec3 {
	hSz := size.DivScalar(2)

	nVtx, nIdx := PlaneN(int(segs.X), int(segs.Y))

	voff := vtxOff
	ioff := idxOff

	// start with neg z as typically back
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.X, mat32.Y, -1, -1, size.X, size.Y, -hSz.X, -hSz.Y, -hSz.Z, int(segs.X), int(segs.Y), pos) // nz
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.X, mat32.Z, 1, -1, size.X, size.Z, -hSz.X, -hSz.Z, -hSz.Y, int(segs.X), int(segs.Z), pos) // ny
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.Z, mat32.Y, -1, -1, size.Z, size.Y, -hSz.Z, -hSz.Y, hSz.X, int(segs.Z), int(segs.Y), pos) // px
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.Z, mat32.Y, 1, -1, size.Z, size.Y, -hSz.Z, -hSz.Y, -hSz.X, int(segs.Z), int(segs.Y), pos) // nx
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.X, mat32.Z, 1, 1, size.X, size.Z, -hSz.X, -hSz.Z, hSz.Y, int(segs.X), int(segs.Z), pos) // py
	voff += nVtx
	ioff += nIdx
	SetPlane(vtxAry, normAry, texAry, idxAry, voff, ioff, mat32.X, mat32.Y, 1, -1, size.X, size.Y, -hSz.X, -hSz.Y, hSz.Z, int(segs.X), int(segs.Y), pos) // pz
	return hSz
}
