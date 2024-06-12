// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import "cogentcore.org/core/math32"

// ShapeGroup is a group of shapes -- returns summary data for shape elements
type ShapeGroup struct {
	ShapeBase

	// list of shapes in group
	Shapes []Shape
}

// N returns number of vertex, index points in this shape element.
func (sb *ShapeGroup) N() (numVertex, nIndex int) {
	numVertex = 0
	nIndex = 0
	for _, sh := range sb.Shapes {
		nv, ni := sh.N()
		numVertex += nv
		nIndex += ni
	}
	return
}

// Set sets points in given allocated arrays, also updates offsets
func (sb *ShapeGroup) Set(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32) {
	vo := sb.VtxOff
	io := sb.IndexOff
	sb.CBBox.SetEmpty()
	for _, sh := range sb.Shapes {
		sh.SetOffs(vo, io)
		sh.Set(vertexArray, normArray, textureArray, indexArray)
		sb.CBBox.ExpandByBox(sh.BBox())
		nv, ni := sh.N()
		vo += nv
		io += ni
	}
}
