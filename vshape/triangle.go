// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build notyet

package vshape

import (
	"github.com/goki/gi/gist"
	"github.com/goki/mat32"
)

////////////////////////////////////////////////////////////////
//   Triangle

// AddTriangle adds one triangle of vertex data (optionally texUV, color) to mesh.
// norm is auto-computed, and bounds expanded.  Must have 3 texs if not nil.
func (ms *MeshBase) AddTriangle(a, b, c mat32.Vec3, texs []mat32.Vec2, clr gist.Color) {
	stVtxIdx := ms.Vtx.Len() / 3 // starting index based on what's there already
	stIdxIdx := ms.Idx.Len()     // starting index based on what's there already
	ms.SetTriangle(stVtxIdx, stIdxIdx, true, a, b, c, texs, clr)
}

// SetTriangle sets one triangle of vertex data (optionally texUV, color,
// and indexes) at given starting *vertex* index (i.e., multiply this *3
// to get actual float offset in Vtx array), and starting Idx index.
// Norm is auto-computed, and bounds expanded.
func (ms *MeshBase) SetTriangle(stVtxIdx, stIdxIdx int, setIdx bool, a, b, c mat32.Vec3, texs []mat32.Vec2, clr gist.Color) {
	hasTex := texs != nil
	hasColor := !clr.IsNil()
	sz := len(ms.Vtx) / 3
	vtxSz, idxSz := 3, 3
	if stVtxIdx+vtxSz > sz {
		dif := (stVtxIdx + vtxSz) - sz
		ms.Vtx.Extend(dif * 3)
		ms.Norm.Extend(dif * 3) // assuming same
		if hasTex {
			ms.Tex.Extend(dif * 2) // assuming same
		}
		if hasColor {
			ms.Color.Extend(dif * 4)
		}
	}

	norm := mat32.Normal(a, b, c)

	clrv := ColorToVec4f(clr)
	vidx := stVtxIdx * 3
	tidx := stVtxIdx * 2
	cidx := stVtxIdx * 4
	a.ToArray(ms.Vtx, vidx)
	norm.ToArray(ms.Norm, vidx)
	b.ToArray(ms.Vtx, vidx+3)
	norm.ToArray(ms.Norm, vidx+3)
	c.ToArray(ms.Vtx, vidx+6)
	norm.ToArray(ms.Norm, vidx+6)
	if hasTex {
		texs[0].ToArray(ms.Tex, tidx)
		texs[1].ToArray(ms.Tex, tidx+2)
		texs[2].ToArray(ms.Tex, tidx+4)
	}
	if hasColor {
		clrv.ToArray(ms.Color, cidx)
		clrv.ToArray(ms.Color, cidx+4)
		clrv.ToArray(ms.Color, cidx+8)
	}

	if setIdx {
		lidx := len(ms.Idx)
		if stIdxIdx+idxSz > lidx {
			ms.Idx.Extend((stIdxIdx + idxSz) - lidx)
		}
		sidx := stIdxIdx
		ms.Idx.Set(sidx, uint32(stVtxIdx), uint32(stVtxIdx+1), uint32(stVtxIdx+2))
	}

	ms.BBoxMu.Lock()
	ms.BBox.BBox.ExpandByPoints([]mat32.Vec3{a, b, c})
	ms.BBoxMu.Unlock()
}

////////////////////////////////////////////////////////////////
//   Quad

// AddQuad adds quad vertex data (optionally texUV, color) to mesh.
// Must have 4 vtxs, 4 texs if !nil.
// Norm is auto-computed, and bbox expanded by points.
func (ms *MeshBase) AddQuad(vtxs []mat32.Vec3, texs []mat32.Vec2, clr gist.Color) {
	stVtxIdx := ms.Vtx.Len() / 3 // starting index based on what's there already
	stIdxIdx := ms.Idx.Len()     // starting index based on what's there already
	ms.SetQuad(stVtxIdx, stIdxIdx, true, vtxs, texs, clr)
}

// SetQuad sets quad vertex data (optionally texUV, color, and indexes)
// at given starting *vertex* index (i.e., multiply this *3 to get actual float
// offset in Vtx array), and starting Idx index.
// Norm is auto-computed, and bbox expanded by points.
func (ms *MeshBase) SetQuad(stVtxIdx, stIdxIdx int, setIdx bool, vtxs []mat32.Vec3, texs []mat32.Vec2, clr gist.Color) {
	hasTex := texs != nil
	hasColor := !clr.IsNil()
	sz := len(ms.Vtx) / 3
	vtxSz, idxSz := 4, 6
	if stVtxIdx+vtxSz > sz {
		dif := (stVtxIdx + vtxSz) - sz
		ms.Vtx.Extend(dif * 3)
		ms.Norm.Extend(dif * 3) // assuming same
		if hasTex {
			ms.Tex.Extend(dif * 2) // assuming same
		}
		if hasColor {
			ms.Color.Extend(dif * 4)
		}
	}

	norm := mat32.Normal(vtxs[0], vtxs[1], vtxs[2])

	clrv := ColorToVec4f(clr)
	vidx := stVtxIdx * 3
	tidx := stVtxIdx * 2
	cidx := stVtxIdx * 4
	for vi := range vtxs {
		vtxs[vi].ToArray(ms.Vtx, vidx)
		norm.ToArray(ms.Norm, vidx)
		vidx += 3
		if hasTex {
			texs[vi].ToArray(ms.Tex, tidx)
			tidx += 2
		}
		if hasColor {
			clrv.ToArray(ms.Color, cidx)
			cidx += 4
		}
	}

	if setIdx {
		lidx := len(ms.Idx)
		if stIdxIdx+idxSz > lidx {
			ms.Idx.Extend((stIdxIdx + idxSz) - lidx)
		}
		sidx := stIdxIdx
		ms.Idx.Set(sidx, uint32(stVtxIdx), uint32(stVtxIdx+1), uint32(stVtxIdx+2),
			uint32(stVtxIdx), uint32(stVtxIdx+2), uint32(stVtxIdx+3))
	}
	ms.BBoxMu.Lock()
	ms.BBox.BBox.ExpandByPoints(vtxs)
	ms.BBoxMu.Unlock()
}
