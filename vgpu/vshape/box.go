// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import "cogentcore.org/core/math32"

// Box is a rectangular-shaped solid (cuboid)
type Box struct {
	ShapeBase

	// size along each dimension
	Size math32.Vector3

	// number of segments to divide each plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1
	Segs math32.Vector3i
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

func (bx *Box) N() (numVertex, nIndex int) {
	numVertex, nIndex = BoxN(bx.Segs)
	return
}

// SetBox sets points in given allocated arrays
func (bx *Box) Set(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32) {
	hSz := SetBox(vertexArray, normArray, textureArray, indexArray, bx.VtxOff, bx.IndexOff, bx.Size, bx.Segs, bx.Pos)

	mn := bx.Pos.Sub(hSz)
	mx := bx.Pos.Add(hSz)
	bx.CBBox.Set(&mn, &mx)
}

// PlaneN returns the N's for a single plane's worth of
// vertex and index data with given number of segments.
// Note: In *vertex* units, not float units (i.e., x3 to get
// actual float offset in Vtx array).
func BoxN(segs math32.Vector3i) (numVertex, nIndex int) {
	nv, ni := PlaneN(int(segs.X), int(segs.Y))
	numVertex += 2 * nv
	nIndex += 2 * ni
	nv, ni = PlaneN(int(segs.X), int(segs.Z))
	numVertex += 2 * nv
	nIndex += 2 * ni
	nv, ni = PlaneN(int(segs.Z), int(segs.Y))
	numVertex += 2 * nv
	nIndex += 2 * ni
	return
}

// SetBox sets box vertex, norm, tex, index data at
// given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Index index.
// for given 3D size, and given number of segments per side.
// finely subdividing a plane allows for higher-quality lighting
// and texture rendering (minimum of 1 will be enforced).
// pos is a 3D position offset. returns 3D size of plane.
func SetBox(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32, vtxOff, idxOff int, size math32.Vector3, segs math32.Vector3i, pos math32.Vector3) math32.Vector3 {
	hSz := size.DivScalar(2)

	numVertex, nIndex := PlaneN(int(segs.X), int(segs.Y))

	voff := vtxOff
	ioff := idxOff

	// start with neg z as typically back
	SetPlane(vertexArray, normArray, textureArray, indexArray, voff, ioff, math32.X, math32.Y, -1, -1, size.X, size.Y, -hSz.X, -hSz.Y, -hSz.Z, int(segs.X), int(segs.Y), pos) // nz
	voff += numVertex
	ioff += nIndex
	SetPlane(vertexArray, normArray, textureArray, indexArray, voff, ioff, math32.X, math32.Z, 1, -1, size.X, size.Z, -hSz.X, -hSz.Z, -hSz.Y, int(segs.X), int(segs.Z), pos) // ny
	voff += numVertex
	ioff += nIndex
	SetPlane(vertexArray, normArray, textureArray, indexArray, voff, ioff, math32.Z, math32.Y, -1, -1, size.Z, size.Y, -hSz.Z, -hSz.Y, hSz.X, int(segs.Z), int(segs.Y), pos) // px
	voff += numVertex
	ioff += nIndex
	SetPlane(vertexArray, normArray, textureArray, indexArray, voff, ioff, math32.Z, math32.Y, 1, -1, size.Z, size.Y, -hSz.Z, -hSz.Y, -hSz.X, int(segs.Z), int(segs.Y), pos) // nx
	voff += numVertex
	ioff += nIndex
	SetPlane(vertexArray, normArray, textureArray, indexArray, voff, ioff, math32.X, math32.Z, 1, 1, size.X, size.Z, -hSz.X, -hSz.Z, hSz.Y, int(segs.X), int(segs.Z), pos) // py
	voff += numVertex
	ioff += nIndex
	SetPlane(vertexArray, normArray, textureArray, indexArray, voff, ioff, math32.X, math32.Y, 1, -1, size.X, size.Y, -hSz.X, -hSz.Y, hSz.Z, int(segs.X), int(segs.Y), pos) // pz
	return hSz
}
