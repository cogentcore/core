// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shape

//go:generate core generate

import "cogentcore.org/core/math32"

// ShapeBase is the base shape element
type ShapeBase struct { //types:add -setters
	// vertex offset, in points
	VertexOffset int

	// index offset, in points
	IndexOffset int

	// cubic bounding box in local coords
	CBBox math32.Box3

	// all shapes take a 3D position offset to enable composition
	Pos math32.Vector3
}

// Offs returns starting offset for vertices, indexes in full shape array,
// in terms of points, not floats
func (sb *ShapeBase) Offsets() (vtxOffset, idxOffset int) {
	vtxOffset, idxOffset = sb.VertexOffset, sb.IndexOffset
	return
}

// SetOffs sets starting offsets for vertices, indexes in full shape array
func (sb *ShapeBase) SetOffsets(vtxOffset, idxOffset int) {
	sb.VertexOffset, sb.IndexOffset = vtxOffset, idxOffset
}

// BBox returns the bounding box for the shape, typically centered around 0
// This is only valid after Set has been called.
func (sb *ShapeBase) BBox() math32.Box3 {
	return sb.CBBox
}

// SetColor sets color for given range of vertex indexes
func SetColor(clrs math32.ArrayF32, vtxOff int, numVertex int, clr math32.Vector4) {
	cidx := vtxOff * 4
	for vi := 0; vi < numVertex; vi++ {
		clr.ToSlice(clrs, cidx+vi*4)
	}
}

// BBoxFromVtxs returns the bounding box updated from the range of vertex points
func BBoxFromVtxs(vertex math32.ArrayF32, vtxOff int, numVertex int) math32.Box3 {
	bb := math32.B3Empty()
	vidx := vtxOff * 3
	var vtx math32.Vector3
	for vi := 0; vi < numVertex; vi++ {
		vtx.FromSlice(vertex, vidx+vi*3)
		bb.ExpandByPoint(vtx)
	}
	return bb
}
