// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shape

import (
	"cogentcore.org/core/math32"
)

// Plane is a flat 2D plane, which can be oriented along any
// axis facing either positive or negative
type Plane struct { //types:add -setters
	ShapeBase

	// axis along which the normal perpendicular to the plane points.
	// E.g., if the Y axis is specified, then it is a standard X-Z ground plane.
	// See also NormalNeg for whether it is facing in the positive or
	// negative of the given axis.
	NormalAxis math32.Dims

	// if false, the plane normal is facing in the positive direction
	// along specified NormalAxis, otherwise it faces in the negative.
	NormalNeg bool

	// 2D size of plane
	Size math32.Vector2

	// number of segments to divide plane into (enforced to be at least 1).
	// Can generally increase rendering quality to have > 1.
	Segs math32.Vector2i

	// offset from origin along direction of normal to the plane.
	Offset float32
}

// NewPlane returns a Plane shape with given size.
func NewPlane(axis math32.Dims, width, height float32) *Plane {
	pl := &Plane{}
	pl.Defaults()
	pl.NormalAxis = axis
	pl.Size.Set(width, height)
	return pl
}

func (pl *Plane) Defaults() {
	pl.NormalAxis = math32.Y
	pl.NormalNeg = false
	pl.Size.Set(1, 1)
	pl.Segs.Set(1, 1)
	pl.Offset = 0
}

// MeshSize returns number of vertex, index points in this shape element
func (pl *Plane) MeshSize() (numVertex, nIndex int, hasColor bool) {
	numVertex, nIndex = PlaneN(int(pl.Segs.X), int(pl.Segs.Y))
	return
}

// Set sets points in given allocated arrays.
func (pl *Plane) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	sz := SetPlaneAxisSize(vertex, normal, texcoord, index, pl.VertexOffset, pl.IndexOffset, pl.NormalAxis, pl.NormalNeg, pl.Size, pl.Segs, pl.Offset, pl.Pos)
	mn := pl.Pos.Sub(sz)
	mx := pl.Pos.Add(sz)
	pl.CBBox.Set(&mn, &mx)
}

// PlaneN returns the N's for a single plane's worth of
// vertex and index data with given number of segments.
// Note: In *vertex* units, not float units (i.e., x3 to get
// actual float offset in Vtx array).
// numVertex = (wsegs + 1) * (hsegs + 1)
// nIndex = wsegs * hsegs * 6.
func PlaneN(wsegs, hsegs int) (numVertex, nIndex int) {
	wsegs = max(wsegs, 1)
	hsegs = max(hsegs, 1)
	numVertex = (wsegs + 1) * (hsegs + 1)
	nIndex = wsegs * hsegs * 6
	return
}

// SetPlaneAxisSize sets plane vertex, normal, tex, index data at
// given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Index index.
// using Normal Axis, offset, and size params.
// wsegs, hsegs = number of segments to create in each dimension --
// more finely subdividing a plane allows for higher-quality lighting
// and texture rendering (minimum of 1 will be enforced).
// offset is the distance to place the plane along the orthogonal axis.
// pos is a 3D position offset. returns 3D size of plane.
// returns bounding box.
func SetPlaneAxisSize(vertex, normal, texcoord math32.ArrayF32, index math32.ArrayU32, vtxOff, idxOff int, normalAxis math32.Dims, normalNeg bool, size math32.Vector2, segs math32.Vector2i, offset float32, pos math32.Vector3) math32.Vector3 {
	hSz := size.DivScalar(2)
	thin := float32(.0000001)
	sz := math32.Vector3{}
	switch normalAxis {
	case math32.X:
		sz.Set(thin, hSz.Y, hSz.X)
		if normalNeg {
			SetPlane(vertex, normal, texcoord, index, vtxOff, idxOff, math32.Z, math32.Y, 1, -1, size.X, size.Y, -hSz.X, -hSz.Y, -offset, int(segs.X), int(segs.Y), pos) // nx
			sz.X += -offset
		} else {
			SetPlane(vertex, normal, texcoord, index, vtxOff, idxOff, math32.Z, math32.Y, -1, -1, size.X, size.Y, -hSz.X, -hSz.Y, offset, int(segs.X), int(segs.Y), pos) // px
			sz.X += offset
		}
	case math32.Y:
		sz.Set(hSz.X, thin, hSz.Y)
		if normalNeg {
			SetPlane(vertex, normal, texcoord, index, vtxOff, idxOff, math32.X, math32.Z, 1, -1, size.X, size.Y, -hSz.X, -hSz.Y, -offset, int(segs.X), int(segs.Y), pos) // ny
			sz.Y += -offset
		} else {
			SetPlane(vertex, normal, texcoord, index, vtxOff, idxOff, math32.X, math32.Z, 1, 1, size.X, size.Y, -hSz.X, -hSz.Y, offset, int(segs.X), int(segs.Y), pos) // py
			sz.Y += offset
		}
	case math32.Z:
		sz.Set(hSz.X, hSz.Y, thin)
		if normalNeg {
			SetPlane(vertex, normal, texcoord, index, vtxOff, idxOff, math32.X, math32.Y, -1, -1, size.X, size.Y, -hSz.X, -hSz.Y, -offset, int(segs.X), int(segs.Y), pos) // nz
			sz.Z += -offset
		} else {
			SetPlane(vertex, normal, texcoord, index, vtxOff, idxOff, math32.X, math32.Y, 1, -1, size.X, size.Y, -hSz.X, -hSz.Y, offset, int(segs.X), int(segs.Y), pos) // pz
			sz.Z += offset
		}
	}
	return sz
}

// SetPlane sets plane vertex, normal, tex, index data at given starting *vertex* index
// (i.e., multiply this *3 to get actual float offset in Vtx array), and starting Index index.
// waxis, haxis = width, height axis, wdir, hdir are the directions for width
// and height dimensions.
// wsegs, hsegs = number of segments to create in each dimension --
// more finely subdividing a plane allows for higher-quality lighting
// and texture rendering (minimum of 1 will be enforced).
// offset is the distance to place the plane along the orthogonal axis.
// pos is a 3D position offset.
func SetPlane(vertex, normal, texcoord math32.ArrayF32, index math32.ArrayU32, vtxOff, idxOff int, waxis, haxis math32.Dims, wdir, hdir int, width, height, woff, hoff, zoff float32, wsegs, hsegs int, pos math32.Vector3) {
	w := math32.Z
	if (waxis == math32.X && haxis == math32.Y) || (waxis == math32.Y && haxis == math32.X) {
		w = math32.Z
	} else if (waxis == math32.X && haxis == math32.Z) || (waxis == math32.Z && haxis == math32.X) {
		w = math32.Y
	} else if (waxis == math32.Z && haxis == math32.Y) || (waxis == math32.Y && haxis == math32.Z) {
		w = math32.X
	}
	wsegs = max(wsegs, 1)
	hsegs = max(hsegs, 1)

	norm := math32.Vector3{}
	if zoff > 0 {
		norm.SetDim(w, 1)
	} else {
		norm.SetDim(w, -1)
	}

	wsegs1 := wsegs + 1
	hsegs1 := hsegs + 1
	segWidth := width / float32(wsegs)
	segHeight := height / float32(hsegs)

	fwdir := float32(wdir)
	fhdir := float32(hdir)
	if wdir < 0 {
		woff = width + woff
	}
	if hdir < 0 {
		hoff = height + hoff
	}

	vtx := math32.Vector3{}
	tex := math32.Vector2{}
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	for iy := 0; iy < hsegs1; iy++ {
		for ix := 0; ix < wsegs1; ix++ {
			vtx.SetDim(waxis, (float32(ix)*segWidth)*fwdir+woff)
			vtx.SetDim(haxis, (float32(iy)*segHeight)*fhdir+hoff)
			vtx.SetDim(w, zoff)
			vtx.Add(pos)
			vtx.ToSlice(vertex, vidx)
			norm.ToSlice(normal, vidx)
			tex.Set(float32(ix)/float32(wsegs), float32(1)-(float32(iy)/float32(hsegs)))
			tex.ToSlice(texcoord, tidx)
			vidx += 3
			tidx += 2
		}
	}

	sidx := idxOff
	for iy := 0; iy < hsegs; iy++ {
		for ix := 0; ix < wsegs; ix++ {
			a := ix + wsegs1*iy
			b := ix + wsegs1*(iy+1)
			c := (ix + 1) + wsegs1*(iy+1)
			d := (ix + 1) + wsegs1*iy
			index.Set(sidx, uint32(a+vtxOff), uint32(b+vtxOff), uint32(d+vtxOff), uint32(b+vtxOff), uint32(c+vtxOff), uint32(d+vtxOff))
			sidx += 6
		}
	}
}
