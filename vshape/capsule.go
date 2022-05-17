// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import "github.com/goki/mat32"

// Capsule is a generalized capsule shape: a cylinder with hemisphere end caps.
// Supports different radii on each end.
// Height is along the Y axis -- total height is Height + TopRad + BotRad.
type Capsule struct {
	ShapeBase
	Height     float32 `desc:"height of the cylinder portion"`
	TopRad     float32 `desc:"radius of the top -- set to 0 to omit top cap"`
	BotRad     float32 `desc:"radius of the bottom -- set to 0 to omit bottom cap"`
	RadialSegs int     `min:"1" desc:"number of radial segments (32 is a reasonable default for full circle)"`
	HeightSegs int     `desc:"number of height segments"`
	CapSegs    int     `desc:"number of segments in the hemisphere cap ends (16 is a reasonable default)"`
	AngStart   float32 `min:"0" max:"360" step:"5" desc:"starting angle in degrees, relative to -1,0,0 left side starting point"`
	AngLen     float32 `min:"0" max:"360" step:"5" desc:"total angle to generate in degrees (max 360)"`
}

// NewCapsule returns a Capsule shape with given radius, height,
// number of radial segments, number of height segments,
// and presence of a top and/or bottom cap.
// Height is along the Y axis.
func NewCapsule(height, radius float32, segs, heightSegs int) *Capsule {
	cp := &Capsule{}
	cp.Defaults()

	cp.Height = height
	cp.TopRad = radius
	cp.BotRad = radius
	cp.RadialSegs = segs
	cp.CapSegs = segs
	cp.HeightSegs = heightSegs
	return cp
}

func (cp *Capsule) Defaults() {
	cp.Height = 1
	cp.TopRad = 0.5
	cp.BotRad = 0.5
	cp.RadialSegs = 32
	cp.HeightSegs = 32
	cp.CapSegs = 32
	cp.AngStart = 0
	cp.AngLen = 360
}

func (cp *Capsule) N() (nVtx, nIdx int) {
	nVtx, nIdx = CylinderSectorN(cp.RadialSegs, cp.HeightSegs, false, false)
	if cp.BotRad > 0 {
		nv, ni := SphereSectorN(cp.RadialSegs, cp.CapSegs, 90, 90)
		nVtx += nv
		nIdx += ni
	}
	if cp.TopRad > 0 {
		nv, ni := SphereSectorN(cp.RadialSegs, cp.CapSegs, 0, 90)
		nVtx += nv
		nIdx += ni
	}
	return
}

// SetCapsuleSector sets points in given allocated arrays
func (cp *Capsule) Set(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	voff := cp.VtxOff
	ioff := cp.IdxOff
	cp.CBBox = SetCylinderSector(vtxAry, normAry, texAry, idxAry, voff, ioff, cp.Height, cp.TopRad, cp.BotRad, cp.RadialSegs, cp.HeightSegs, cp.AngStart, cp.AngLen, false, false, cp.Pos)
	nv, ni := CylinderSectorN(cp.RadialSegs, cp.HeightSegs, false, false)
	voff += nv
	ioff += ni

	if cp.BotRad > 0 {
		ps := cp.Pos
		ps.Y -= cp.Height / 2
		cbb := SetSphereSector(vtxAry, normAry, texAry, idxAry, voff, ioff, cp.BotRad, cp.RadialSegs, cp.CapSegs, cp.AngStart, cp.AngLen, 90, 90, ps)
		cp.CBBox.ExpandByBox(cbb)
		nv, ni = SphereSectorN(cp.RadialSegs, cp.CapSegs, 90, 90)
		voff += nv
		ioff += ni
	}
	if cp.TopRad > 0 {
		ps := cp.Pos
		ps.Y += cp.Height / 2
		cbb := SetSphereSector(vtxAry, normAry, texAry, idxAry, voff, ioff, cp.TopRad, cp.RadialSegs, cp.CapSegs, cp.AngStart, cp.AngLen, 0, 90, ps)
		cp.CBBox.ExpandByBox(cbb)
	}
}
