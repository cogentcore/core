// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import "goki.dev/mat32/v2"

// Shape is an interface for all shape-constructing elements
type Shape interface {
	// N returns number of vertex, index points in this shape element
	N() (nVtx, nIdx int)

	// Offs returns starting offset for verticies, indexes in full shape array,
	// in terms of points, not floats
	Offs() (vtxOff, idxOff int)

	// SetOffs sets starting offset for verticies, indexes in full shape array,
	// in terms of points, not floats
	SetOffs(vtxOff, idxOff int)

	// Set sets points in given allocated arrays
	Set(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32)

	// BBox returns the bounding box for the shape, typically centered around 0
	// This is only valid after Set has been called.
	BBox() mat32.Box3
}

// ShapeBase is the base shape element
type ShapeBase struct {

	// vertex offset, in points
	VtxOff int `desc:"vertex offset, in points"`

	// index offset, in points
	IdxOff int `desc:"index offset, in points"`

	// cubic bounding box in local coords
	CBBox mat32.Box3 `desc:"cubic bounding box in local coords"`

	// all shapes take a 3D position offset to enable composition
	Pos mat32.Vec3 `desc:"all shapes take a 3D position offset to enable composition"`
}

// Offs returns starting offset for verticies, indexes in full shape array,
// in terms of points, not floats
func (sb *ShapeBase) Offs() (vtxOff, idxOff int) {
	vtxOff, idxOff = sb.VtxOff, sb.IdxOff
	return
}

// SetOffs sets starting offsets for verticies, indexes in full shape array
func (sb *ShapeBase) SetOffs(vtxOff, idxOff int) {
	sb.VtxOff, sb.IdxOff = vtxOff, idxOff
}

// BBox returns the bounding box for the shape, typically centered around 0
// This is only valid after Set has been called.
func (sb *ShapeBase) BBox() mat32.Box3 {
	return sb.CBBox
}

// SetColor sets color for given range of vertex indexes
func SetColor(clrAry mat32.ArrayF32, vtxOff int, nvtxs int, clr mat32.Vec4) {
	cidx := vtxOff * 4
	for vi := 0; vi < nvtxs; vi++ {
		clr.ToArray(clrAry, cidx+vi*4)
	}
}

// BBoxFromVtxs returns the bounding box updated from the range of vertex points
func BBoxFromVtxs(vtxAry mat32.ArrayF32, vtxOff int, nvtxs int) mat32.Box3 {
	bb := mat32.NewEmptyBox3()
	vidx := vtxOff * 3
	var vtx mat32.Vec3
	for vi := 0; vi < nvtxs; vi++ {
		vtx.FromArray(vtxAry, vidx+vi*3)
		bb.ExpandByPoint(vtx)
	}
	return bb
}
