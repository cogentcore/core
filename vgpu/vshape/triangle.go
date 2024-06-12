// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import (
	"cogentcore.org/core/math32"
)

////////////////////////////////////////////////////////////////
//   Triangle

// TriangleN returns 3, 3
func TriangleN() (numVertex, nIndex int) {
	return 3, 3
}

// SetTriangle sets one triangle of vertex data indexes, and optionally
// texUV coords, at given starting *vertex* index (i.e., multiply this *3
// to get actual float offset in Vtx array), and starting Index index.
// Norm is auto-computed, and bounds expanded.
// pos is a 3D position offset. returns 3D size of plane.
// returns bounding box.
func SetTriangle(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32, vtxOff, idxOff int, a, b, c math32.Vector3, texs []math32.Vector2, pos math32.Vector3) math32.Box3 {
	hasTex := texs != nil
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	norm := math32.Normal(a, b, c)

	a.Add(pos).ToSlice(vertexArray, vidx)
	norm.ToSlice(normArray, vidx)
	b.Add(pos).ToSlice(vertexArray, vidx+3)
	norm.ToSlice(normArray, vidx+3)
	c.Add(pos).ToSlice(vertexArray, vidx+6)
	norm.ToSlice(normArray, vidx+6)
	if hasTex {
		texs[0].ToSlice(textureArray, tidx)
		texs[1].ToSlice(textureArray, tidx+2)
		texs[2].ToSlice(textureArray, tidx+4)
	}

	indexArray.Set(idxOff, uint32(vtxOff), uint32(vtxOff+1), uint32(vtxOff+2))

	bb := math32.B3Empty()
	bb.ExpandByPoints([]math32.Vector3{a, b, c})
	return bb
}

////////////////////////////////////////////////////////////////
//   Quad

// QuadN returns 4, 6
func QuadN() (numVertex, nIndex int) {
	return 4, 6
}

// SetQuad sets quad vertex data (optionally texUV, color, and indexes)
// at given starting *vertex* index (i.e., multiply this *3 to get actual float
// offset in Vtx array), and starting Index index.
// Norm is auto-computed, and bbox expanded by points.
// pos is a 3D position offset. returns 3D size of plane.
// returns bounding box.
func SetQuad(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32, vtxOff, idxOff int, vtxs []math32.Vector3, texs []math32.Vector2, pos math32.Vector3) math32.Box3 {
	hasTex := texs != nil
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	norm := math32.Normal(vtxs[0], vtxs[1], vtxs[2])

	for vi := range vtxs {
		vtxs[vi].Add(pos).ToSlice(vertexArray, vidx)
		norm.ToSlice(normArray, vidx)
		vidx += 3
		if hasTex {
			texs[vi].ToSlice(textureArray, tidx)
			tidx += 2
		}
	}

	indexArray.Set(idxOff, uint32(vtxOff), uint32(vtxOff+1), uint32(vtxOff+2),
		uint32(vtxOff), uint32(vtxOff+2), uint32(vtxOff+3))

	bb := math32.B3Empty()
	bb.ExpandByPoints(vtxs)
	return bb
}
