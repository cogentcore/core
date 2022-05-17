// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import (
	"github.com/goki/ki/ints"
	"github.com/goki/mat32"
)

// Plane is a flat 2D plane, which can be oriented along any
// axis facing either positive or negative
type Plane struct {
	ShapeBase
	NormAxis mat32.Dims  `desc:"axis along which the normal perpendicular to the plane points.  E.g., if the Y axis is specified, then it is a standard X-Z ground plane -- see also NormNeg for whether it is facing in the positive or negative of the given axis."`
	NormNeg  bool        `desc:"if false, the plane normal facing in the positive direction along specified NormAxis, otherwise it faces in the negative if true"`
	Size     mat32.Vec2  `desc:"2D size of plane"`
	Segs     mat32.Vec2i `desc:"number of segments to divide plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1"`
	Offset   float32     `desc:"offset from origin along direction of normal to the plane"`
}

// NewPlane returns a Plane shape with given size
func NewPlane(axis mat32.Dims, width, height float32) *Plane {
	pl := &Plane{}
	pl.Defaults()
	pl.NormAxis = axis
	pl.Size.Set(width, height)
	return pl
}

func (pl *Plane) Defaults() {
	pl.NormAxis = mat32.Y
	pl.NormNeg = false
	pl.Size.Set(1, 1)
	pl.Segs.Set(1, 1)
	pl.Offset = 0
}

// N returns number of vertex, index points in this shape element
func (pl *Plane) N() (nVtx, nIdx int) {
	nVtx, nIdx = PlaneN(int(pl.Segs.X), int(pl.Segs.Y))
	return
}

// Set sets points in given allocated arrays
func (pl *Plane) Set(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	sz := SetPlaneAxisSize(vtxAry, normAry, texAry, idxAry, pl.VtxOff, pl.IdxOff, pl.NormAxis, pl.NormNeg, pl.Size, pl.Segs, pl.Offset, pl.Pos)
	mn := pl.Pos.Sub(sz)
	mx := pl.Pos.Add(sz)
	pl.CBBox.Set(&mn, &mx)
}

// PlaneN returns the N's for a single plane's worth of
// vertex and index data with given number of segments.
// Note: In *vertex* units, not float units (i.e., x3 to get
// actual float offset in Vtx array).
// nVtx = (wsegs + 1) * (hsegs + 1)
// nIdx = wsegs * hsegs * 6
func PlaneN(wsegs, hsegs int) (nVtx, nIdx int) {
	wsegs = ints.MaxInt(wsegs, 1)
	hsegs = ints.MaxInt(hsegs, 1)
	nVtx = (wsegs + 1) * (hsegs + 1)
	nIdx = wsegs * hsegs * 6
	return
}

// SetPlaneAxisSize sets plane vertex, norm, tex, index data at
// given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Idx index.
// using Norm Axis, offset, and size params.
// wsegs, hsegs = number of segments to create in each dimension --
// more finely subdividing a plane allows for higher-quality lighting
// and texture rendering (minimum of 1 will be enforced).
// offset is the distance to place the plane along the orthogonal axis.
// pos is a 3D position offset. returns 3D size of plane.
func SetPlaneAxisSize(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32, vtxOff, idxOff int, normAxis mat32.Dims, normNeg bool, size mat32.Vec2, segs mat32.Vec2i, offset float32, pos mat32.Vec3) mat32.Vec3 {
	hSz := size.DivScalar(2)
	thin := float32(.0000001)
	sz := mat32.Vec3{}
	switch normAxis {
	case mat32.X:
		sz.Set(thin, hSz.Y, hSz.X)
		if normNeg {
			SetPlane(vtxAry, normAry, texAry, idxAry, vtxOff, idxOff, mat32.Z, mat32.Y, 1, -1, size.X, size.Y, -hSz.X, -hSz.Y, -offset, int(segs.X), int(segs.Y), pos) // nx
		} else {
			SetPlane(vtxAry, normAry, texAry, idxAry, vtxOff, idxOff, mat32.Z, mat32.Y, -1, -1, size.X, size.Y, -hSz.X, -hSz.Y, offset, int(segs.X), int(segs.Y), pos) // px
		}
	case mat32.Y:
		sz.Set(hSz.X, thin, hSz.Y)
		if normNeg {
			SetPlane(vtxAry, normAry, texAry, idxAry, vtxOff, idxOff, mat32.X, mat32.Z, 1, -1, size.X, size.Y, -hSz.X, -hSz.Y, -offset, int(segs.X), int(segs.Y), pos) // ny
		} else {
			SetPlane(vtxAry, normAry, texAry, idxAry, vtxOff, idxOff, mat32.X, mat32.Z, 1, 1, size.X, size.Y, -hSz.X, -hSz.Y, offset, int(segs.X), int(segs.Y), pos) // py
		}
	case mat32.Z:
		sz.Set(hSz.X, hSz.Y, thin)
		if normNeg {
			SetPlane(vtxAry, normAry, texAry, idxAry, vtxOff, idxOff, mat32.X, mat32.Y, -1, -1, size.X, size.Y, -hSz.X, -hSz.Y, -offset, int(segs.X), int(segs.Y), pos) // nz
		} else {
			SetPlane(vtxAry, normAry, texAry, idxAry, vtxOff, idxOff, mat32.X, mat32.Y, 1, -1, size.X, size.Y, -hSz.X, -hSz.Y, offset, int(segs.X), int(segs.Y), pos) // pz
		}
	}
	return sz
}

// SetPlane sets plane vertex, norm, tex, index data at given starting *vertex* index
// (i.e., multiply this *3 to get actual float offset in Vtx array), and starting Idx index.
// waxis, haxis = width, height axis, wdir, hdir are the directions for width
// and height dimensions.
// wsegs, hsegs = number of segments to create in each dimension --
// more finely subdividing a plane allows for higher-quality lighting
// and texture rendering (minimum of 1 will be enforced).
// offset is the distance to place the plane along the orthogonal axis.
// pos is a 3D position offset.
func SetPlane(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32, vtxOff, idxOff int, waxis, haxis mat32.Dims, wdir, hdir int, width, height, woff, hoff, zoff float32, wsegs, hsegs int, pos mat32.Vec3) {
	w := mat32.Z
	if (waxis == mat32.X && haxis == mat32.Y) || (waxis == mat32.Y && haxis == mat32.X) {
		w = mat32.Z
	} else if (waxis == mat32.X && haxis == mat32.Z) || (waxis == mat32.Z && haxis == mat32.X) {
		w = mat32.Y
	} else if (waxis == mat32.Z && haxis == mat32.Y) || (waxis == mat32.Y && haxis == mat32.Z) {
		w = mat32.X
	}
	wsegs = ints.MaxInt(wsegs, 1)
	hsegs = ints.MaxInt(hsegs, 1)

	norm := mat32.Vec3{}
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

	vtx := mat32.Vec3{}
	tex := mat32.Vec2{}
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	for iy := 0; iy < hsegs1; iy++ {
		for ix := 0; ix < wsegs1; ix++ {
			vtx.SetDim(waxis, (float32(ix)*segWidth)*fwdir+woff)
			vtx.SetDim(haxis, (float32(iy)*segHeight)*fhdir+hoff)
			vtx.SetDim(w, zoff)
			vtx.Add(pos)
			vtx.ToArray(vtxAry, vidx)
			norm.ToArray(normAry, vidx)
			tex.Set(float32(ix)/float32(wsegs), float32(1)-(float32(iy)/float32(hsegs)))
			tex.ToArray(texAry, tidx)
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
			idxAry.Set(sidx, uint32(a+vtxOff), uint32(b+vtxOff), uint32(d+vtxOff), uint32(b+vtxOff), uint32(c+vtxOff), uint32(d+vtxOff))
			sidx += 6
		}
	}
}
