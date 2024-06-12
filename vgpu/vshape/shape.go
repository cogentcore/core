// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import "cogentcore.org/core/math32"

// Shape is an interface for all shape-constructing elements
type Shape interface {
	// N returns number of vertex, index points in this shape element
	N() (numVertex, nIndex int)

	// Offs returns starting offset for vertices, indexes in full shape array,
	// in terms of points, not floats
	Offs() (vtxOff, idxOff int)

	// SetOffs sets starting offset for vertices, indexes in full shape array,
	// in terms of points, not floats
	SetOffs(vtxOff, idxOff int)

	// Set sets points in given allocated arrays
	Set(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32)

	// BBox returns the bounding box for the shape, typically centered around 0
	// This is only valid after Set has been called.
	BBox() math32.Box3
}

// ShapeBase is the base shape element
type ShapeBase struct {

	// vertex offset, in points
	VtxOff int

	// index offset, in points
	IndexOff int

	// cubic bounding box in local coords
	CBBox math32.Box3

	// all shapes take a 3D position offset to enable composition
	Pos math32.Vector3
}

// Offs returns starting offset for vertices, indexes in full shape array,
// in terms of points, not floats
func (sb *ShapeBase) Offs() (vtxOff, idxOff int) {
	vtxOff, idxOff = sb.VtxOff, sb.IndexOff
	return
}

// SetOffs sets starting offsets for vertices, indexes in full shape array
func (sb *ShapeBase) SetOffs(vtxOff, idxOff int) {
	sb.VtxOff, sb.IndexOff = vtxOff, idxOff
}

// BBox returns the bounding box for the shape, typically centered around 0
// This is only valid after Set has been called.
func (sb *ShapeBase) BBox() math32.Box3 {
	return sb.CBBox
}

// SetColor sets color for given range of vertex indexes
func SetColor(colorArray math32.ArrayF32, vtxOff int, numVertex int, clr math32.Vector4) {
	cidx := vtxOff * 4
	for vi := 0; vi < numVertex; vi++ {
		clr.ToSlice(colorArray, cidx+vi*4)
	}
}

// BBoxFromVtxs returns the bounding box updated from the range of vertex points
func BBoxFromVtxs(vertexArray math32.ArrayF32, vtxOff int, numVertex int) math32.Box3 {
	bb := math32.B3Empty()
	vidx := vtxOff * 3
	var vtx math32.Vector3
	for vi := 0; vi < numVertex; vi++ {
		vtx.FromSlice(vertexArray, vidx+vi*3)
		bb.ExpandByPoint(vtx)
	}
	return bb
}
