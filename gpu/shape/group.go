// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shape

import "cogentcore.org/core/math32"

// ShapeGroup is a group of shapes.
// Returns summary data for shape elements.
type ShapeGroup struct { //types:add -setters
	ShapeBase

	// list of shapes in group
	Shapes []Mesh
}

// Size returns number of vertex, index points in this shape element.
func (sb *ShapeGroup) Size() (numVertex, numIndex int, hasColor bool) {
	numVertex = 0
	numIndex = 0
	hasColor = false
	for _, sh := range sb.Shapes {
		nv, ni, hc := sh.Size()
		numVertex += nv
		numIndex += ni
		hasColor = hasColor || hc // todo: not good if inconsistent..
	}
	return
}

// Set sets points in given allocated arrays, also updates offsets
func (sb *ShapeGroup) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	vo := sb.VertexOffset
	io := sb.IndexOffset
	sb.CBBox.SetEmpty()
	for _, sh := range sb.Shapes {
		sh.SetOffsets(vo, io)
		sh.Set(vertex, normal, texcoord, clrs, index)
		sb.CBBox.ExpandByBox(sh.BBox())
		nv, ni, _ := sh.Size()
		vo += nv
		io += ni
	}
}
