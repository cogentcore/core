// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import (
	"goki.dev/mat32/v2"
)

////////////////////////////////////////////////////////////////
//   Triangle

// TriangleN returns 3, 3
func TriangleN() (nVtx, nIdx int) {
	return 3, 3
}

// SetTriangle sets one triangle of vertex data indexes, and optionally
// texUV coords, at given starting *vertex* index (i.e., multiply this *3
// to get actual float offset in Vtx array), and starting Idx index.
// Norm is auto-computed, and bounds expanded.
// pos is a 3D position offset. returns 3D size of plane.
// returns bounding box.
func SetTriangle(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32, vtxOff, idxOff int, a, b, c mat32.Vec3, texs []mat32.Vec2, pos mat32.Vec3) mat32.Box3 {
	hasTex := texs != nil
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	norm := mat32.Normal(a, b, c)

	a.Add(pos).ToArray(vtxAry, vidx)
	norm.ToArray(normAry, vidx)
	b.Add(pos).ToArray(vtxAry, vidx+3)
	norm.ToArray(normAry, vidx+3)
	c.Add(pos).ToArray(vtxAry, vidx+6)
	norm.ToArray(normAry, vidx+6)
	if hasTex {
		texs[0].ToArray(texAry, tidx)
		texs[1].ToArray(texAry, tidx+2)
		texs[2].ToArray(texAry, tidx+4)
	}

	idxAry.Set(idxOff, uint32(vtxOff), uint32(vtxOff+1), uint32(vtxOff+2))

	bb := mat32.NewEmptyBox3()
	bb.ExpandByPoints([]mat32.Vec3{a, b, c})
	return bb
}

////////////////////////////////////////////////////////////////
//   Quad

// QuadN returns 4, 6
func QuadN() (nVtx, nIdx int) {
	return 4, 6
}

// SetQuad sets quad vertex data (optionally texUV, color, and indexes)
// at given starting *vertex* index (i.e., multiply this *3 to get actual float
// offset in Vtx array), and starting Idx index.
// Norm is auto-computed, and bbox expanded by points.
// pos is a 3D position offset. returns 3D size of plane.
// returns bounding box.
func SetQuad(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32, vtxOff, idxOff int, vtxs []mat32.Vec3, texs []mat32.Vec2, pos mat32.Vec3) mat32.Box3 {
	hasTex := texs != nil
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	norm := mat32.Normal(vtxs[0], vtxs[1], vtxs[2])

	for vi := range vtxs {
		vtxs[vi].Add(pos).ToArray(vtxAry, vidx)
		norm.ToArray(normAry, vidx)
		vidx += 3
		if hasTex {
			texs[vi].ToArray(texAry, tidx)
			tidx += 2
		}
	}

	idxAry.Set(idxOff, uint32(vtxOff), uint32(vtxOff+1), uint32(vtxOff+2),
		uint32(vtxOff), uint32(vtxOff+2), uint32(vtxOff+3))

	bb := mat32.NewEmptyBox3()
	bb.ExpandByPoints(vtxs)
	return bb
}
