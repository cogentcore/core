// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import (
	"image/color"
	"log"
	"math"

	"goki.dev/mat32/v2"
)

// Lines are lines rendered as long thin boxes defined by points
// and width parameters.  The Mesh must be drawn in the XY plane (i.e., use Z = 0
// or a constant unless specifically relevant to have full 3D variation).
// Rotate the solid to put into other planes.
type Lines struct {
	ShapeBase

	// line points (must be 2 or more)
	Points []mat32.Vec3 `desc:"line points (must be 2 or more)"`

	// line width, Y = height perpendicular to line direction, and X = depth
	Width mat32.Vec2 `desc:"line width, Y = height perpendicular to line direction, and X = depth"`

	// optional colors for each point -- actual color interpolates between
	Colors []color.Color `desc:"optional colors for each point -- actual color interpolates between"`

	// if true, connect the first and last points to form a closed shape
	Closed bool `desc:"if true, connect the first and last points to form a closed shape"`
}

// NewLines returns a Lines shape with given size
func NewLines(points []mat32.Vec3, width mat32.Vec2, closed bool) *Lines {
	ln := &Lines{}
	ln.Defaults()
	ln.Points = points
	ln.Width = width
	ln.Closed = closed
	return ln
}

func (ln *Lines) Defaults() {
	ln.Width.Set(.1, .1)
}

func (ln *Lines) N() (nVtx, nIdx int) {
	nVtx, nIdx = LinesN(len(ln.Points), ln.Closed)
	return
}

// Set sets points in given allocated arrays
func (ln *Lines) Set(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	ln.CBBox = SetLines(vtxAry, normAry, texAry, idxAry, ln.VtxOff, ln.IdxOff, ln.Points, ln.Width, ln.Closed, ln.Pos)
	// todo: colors!
}

/////////////////////////////////////////////////////////////////////

// LinesN returns number of vertex and idx points
func LinesN(npoints int, closed bool) (nVtx, nIdx int) {
	qvn, qin := QuadN()
	nv, ni := 4*(npoints-1)*qvn, 4*(npoints-1)*qin
	if closed {
		nv += 4 * qvn
		ni += 4 * qin
	} else {
		nv += 2 * qvn
		ni += 2 * qin
	}
	return nv, ni
}

// SetLines sets lines rendered as long thin boxes defined by points
// and width parameters.  The Mesh must be drawn in the XY plane (i.e., use Z = 0
// or a constant unless specifically relevant to have full 3D variation).
// Rotate to put into other planes.
func SetLines(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32, vtxOff, idxOff int, points []mat32.Vec3, width mat32.Vec2, closed bool, pos mat32.Vec3) mat32.Box3 {
	np := len(points)
	if np < 2 {
		log.Printf("vshape.SetLines: need 2 or more Points\n")
		return mat32.Box3{}
	}

	pts := points
	if closed {
		pts = append(pts, points[0])
		np++
	}

	vidx := vtxOff * 3

	voff := vidx
	ioff := idxOff
	qvn, qin := QuadN()

	wdy := width.Y / 2
	wdz := width.X / 2

	pi2 := float32(math.Pi / 2)

	// logic for miter joins: https://math.stackexchange.com/questions/1849784/calculate-miter-points-of-stroked-vectors-in-cartesian-plane

	for li := 0; li < np-1; li++ {
		sp := pts[li]
		sp.SetAdd(pos)
		ep := pts[li+1]
		ep.SetAdd(pos)
		spSt := !closed && li == 0
		epEd := !closed && li == np-2

		swap := false
		if ep.X < sp.X {
			sp, ep = ep, sp
			spSt, epEd = epEd, spSt
			swap = true
		}

		v := ep.Sub(sp)
		vn := v.Normal()
		xyang := mat32.Atan2(vn.Y, vn.X)
		xy := mat32.Vec2{wdy * mat32.Cos(xyang+pi2), wdy * mat32.Sin(xyang+pi2)}

		//   sypzm --- eypzm
		//   / |        / |
		// sypzp -- eypzp |// ToAlphaPreMult converts a non-alpha-premultiplied color to a premultiplied one.
		//  | symzm --| eymzm
		//  | /       | /
		// symzp -- eymzp

		sypzp, sypzm, symzp, symzm := sp, sp, sp, sp
		eypzp, eypzm, eymzp, eymzm := ep, ep, ep, ep

		if !spSt {
			pp := sp
			if swap {
				if closed && li == np-2 {
					pp = pts[1]
				} else {
					pp = pts[li+2]
				}
				pp.SetAdd(pos)
			} else {
				if closed && li == 0 {
					pp = pts[np-2]
				} else {
					pp = pts[li-1]
				}
				pp.SetAdd(pos)
			}
			xy = MiterPts(pp.X, pp.Y, sp.X, sp.Y, ep.X, ep.Y, wdy)
		}

		sypzp.X += xy.X
		sypzp.Y += xy.Y
		sypzp.Z += wdz

		sypzm.X += xy.X
		sypzm.Y += xy.Y
		sypzm.Z += -wdz

		symzp.X += -xy.X
		symzp.Y += -xy.Y
		symzp.Z += wdz

		symzm.X += -xy.X
		symzm.Y += -xy.Y
		symzm.Z += -wdz

		if !epEd {
			xp := ep
			if swap {
				if closed && li == 0 {
					xp = pts[np-2]
				} else {
					xp = pts[li-1]
				}
				xp.SetAdd(pos)
			} else {
				if closed && li == np-2 {
					xp = pts[1]
				} else {
					xp = pts[li+2]
				}
				xp.SetAdd(pos)
			}
			xy = MiterPts(xp.X, xp.Y, ep.X, ep.Y, sp.X, sp.Y, wdy)
		}

		eypzp.X += xy.X
		eypzp.Y += xy.Y
		eypzp.Z += wdz

		eypzm.X += xy.X
		eypzm.Y += xy.Y
		eypzm.Z += -wdz

		eymzp.X += -xy.X
		eymzp.Y += -xy.Y
		eymzp.Z += wdz

		eymzm.X += -xy.X
		eymzm.Y += -xy.Y
		eymzm.Z += -wdz

		// front     back
		// 0  3      1  2
		// 1  2      0  3
		// two triangles are: 0,1,2;  0,2,3

		if swap {
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{sypzm, symzm, eymzm, eypzm}, nil, pos)
			voff += qvn
			ioff += qin
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{sypzp, sypzm, eypzm, eypzp}, nil, pos) // bottom (yp, upside down)
			voff += qvn
			ioff += qin
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{symzm, symzp, eymzp, eymzm}, nil, pos) // top (ym)
			voff += qvn
			ioff += qin
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{symzp, sypzp, eypzp, eymzp}, nil, pos) // front (zp)
			voff += qvn
			ioff += qin
		} else {
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{symzm, sypzm, eypzm, eymzm}, nil, pos) // back (zm)
			voff += qvn
			ioff += qin
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{symzp, symzm, eymzm, eymzp}, nil, pos) // bottom (ym)
			voff += qvn
			ioff += qin
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{sypzm, sypzp, eypzp, eypzm}, nil, pos) // top (yp)
			voff += qvn
			ioff += qin
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{sypzp, symzp, eymzp, eypzp}, nil, pos) // front (zp)
			voff += qvn
			ioff += qin
		}

		if spSt { // do cap
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{sypzm, symzm, symzp, sypzp}, nil, pos)
			voff += qvn
			ioff += qin
		}
		if epEd {
			SetQuad(vtxAry, normAry, texAry, idxAry, voff, ioff, []mat32.Vec3{eypzp, eymzp, eymzm, eypzm}, nil, pos)
			voff += qvn
			ioff += qin
		}
	}
	vn := voff - vtxOff
	return BBoxFromVtxs(vtxAry, vtxOff, vn)
}

// MiterPts returns the miter points
func MiterPts(ax, ay, bx, by, cx, cy, w2 float32) mat32.Vec2 {
	ppd := mat32.Vec2{ax - bx, ay - by}
	ppu := ppd.Normal()

	epd := mat32.Vec2{cx - bx, cy - by}
	epv := epd.Normal()

	dp := ppu.Dot(epv)
	jang := mat32.Acos(dp)
	wfact := w2 / mat32.Sin(jang)

	uv := ppu.MulScalar(-wfact)
	vv := epv.MulScalar(-wfact)
	sv := uv.Add(vv)
	return sv
}
