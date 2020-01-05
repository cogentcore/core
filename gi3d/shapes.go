// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"log"
	"math"

	"github.com/chewxy/math32"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/ki/kit"
)

// shapes define different standard mesh shapes

///////////////////////////////////////////////////////////////////////////
//   Plane

// Plane is a flat 2D plane, which can be oriented along any
// axis facing either positive or negative
type Plane struct {
	MeshBase
	NormAxis mat32.Dims  `desc:"axis along which the normal perpendicular to the plane points.  E.g., if the Y axis is specified, then it is a standard X-Z ground plane -- see also NormNeg for whether it is facing in the positive or negative of the given axis."`
	NormNeg  bool        `desc:"if false, the plane normal facing in the positive direction along specified NormAxis, otherwise it faces in the negative if true"`
	Size     mat32.Vec2  `desc:"2D size of plane"`
	Segs     mat32.Vec2i `desc:"number of segments to divide plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1"`
	Offset   float32     `desc:"offset from origin along direction of normal to the plane"`
}

var KiT_Plane = kit.Types.AddType(&Plane{}, nil)

// AddNewPlane adds Plane mesh to given scene,
// with given name and size, with its normal pointing
// by default in the positive Y axis (i.e., a "ground" plane).
// Offset is 0.
func AddNewPlane(sc *Scene, name string, width, height float32) *Plane {
	pl := &Plane{}
	pl.Nm = name
	pl.NormAxis = mat32.Y
	pl.NormNeg = false
	pl.Size.Set(width, height)
	pl.Segs.Set(1, 1)
	pl.Offset = 0
	sc.AddMesh(pl)
	return pl
}

func (pl *Plane) Make(sc *Scene) {
	pl.Reset()

	hSz := pl.Size.DivScalar(2)

	clr := gi.Color{}

	thin := float32(.0000001)
	sz := mat32.Vec3{}

	switch pl.NormAxis {
	case mat32.X:
		sz.Set(thin, hSz.Y, hSz.X)
		if pl.NormNeg {
			pl.AddPlane(mat32.Z, mat32.Y, 1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, -pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // nx
		} else {
			pl.AddPlane(mat32.Z, mat32.Y, -1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // px
		}
	case mat32.Y:
		sz.Set(hSz.X, thin, hSz.Y)
		if pl.NormNeg {
			pl.AddPlane(mat32.X, mat32.Z, 1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, -pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // ny
		} else {
			pl.AddPlane(mat32.X, mat32.Z, 1, 1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // py
		}
	case mat32.Z:
		sz.Set(hSz.X, hSz.Y, thin)
		if pl.NormNeg {
			pl.AddPlane(mat32.X, mat32.Y, -1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, -pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // nz
		} else {
			pl.AddPlane(mat32.X, mat32.Y, 1, -1, pl.Size.X, pl.Size.Y, -hSz.X, -hSz.Y, pl.Offset, int(pl.Segs.X), int(pl.Segs.Y), clr) // pz
		}
	}

	pl.BBox.SetBounds(sz.Negate(), sz)
}

///////////////////////////////////////////////////////////////////////////
//   Box

// Box is a rectangular-shaped solid (cuboid)
type Box struct {
	MeshBase
	Size mat32.Vec3  `desc:"size along each dimension"`
	Segs mat32.Vec3i `desc:"number of segments to divide each plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1"`
}

var KiT_Box = kit.Types.AddType(&Box{}, nil)

// AddNewBox adds Box mesh to given scene, with given name and size
func AddNewBox(sc *Scene, name string, width, height, depth float32) *Box {
	bx := &Box{}
	bx.Nm = name
	bx.Size.Set(width, height, depth)
	bx.Segs.Set(1, 1, 1)
	sc.AddMesh(bx)
	return bx
}

func (bx *Box) Make(sc *Scene) {
	bx.Reset()

	hSz := bx.Size.DivScalar(2)

	clr := gi.Color{}

	// start with neg z as typically back
	bx.AddPlane(mat32.X, mat32.Y, -1, -1, bx.Size.X, bx.Size.Y, -hSz.X, -hSz.Y, -hSz.Z, int(bx.Segs.X), int(bx.Segs.Y), clr) // nz
	bx.AddPlane(mat32.X, mat32.Z, 1, -1, bx.Size.X, bx.Size.Z, -hSz.X, -hSz.Z, -hSz.Y, int(bx.Segs.X), int(bx.Segs.Z), clr)  // ny
	bx.AddPlane(mat32.Z, mat32.Y, -1, -1, bx.Size.Z, bx.Size.Y, -hSz.Z, -hSz.Y, hSz.X, int(bx.Segs.Z), int(bx.Segs.Y), clr)  // px
	bx.AddPlane(mat32.Z, mat32.Y, 1, -1, bx.Size.Z, bx.Size.Y, -hSz.Z, -hSz.Y, -hSz.X, int(bx.Segs.Z), int(bx.Segs.Y), clr)  // nx
	bx.AddPlane(mat32.X, mat32.Z, 1, 1, bx.Size.X, bx.Size.Z, -hSz.X, -hSz.Z, hSz.Y, int(bx.Segs.X), int(bx.Segs.Z), clr)    // py
	bx.AddPlane(mat32.X, mat32.Y, 1, -1, bx.Size.X, bx.Size.Y, -hSz.X, -hSz.Y, hSz.Z, int(bx.Segs.X), int(bx.Segs.Y), clr)   // pz

	bx.BBox.SetBounds(hSz.Negate(), hSz)
}

///////////////////////////////////////////////////////////////////////////
//   Lines

// Note: Raw line rendering via OpenGL is not very effective
// -- lines are often very thin and appearance is hardware dependent.
// This approach produces consistent results across platforms,
// is very fast, and is "good enough" for most purposes.
// For high-quality vector rendering, render to a Viewport2D
// and use that as a texture.

// Lines are lines rendered as long thin boxes defined by points
// and width parameters.  The Mesh must be drawn in the XY plane (i.e., use Z = 0
// or a constant unless specifically relevant to have full 3D variation).
// Rotate the solid to put into other planes.
type Lines struct {
	MeshBase
	Points []mat32.Vec3 `desc:"line points (must be 2 or more)"`
	Width  mat32.Vec2   `desc:"line width, Y = height perpendicular to line direction, and X = depth"`
	Close  bool         `desc:"if true, connect the first and last points to form a closed shape"`
}

var KiT_Lines = kit.Types.AddType(&Lines{}, nil)

// AddNewLines adds Lines mesh to given scene, with given start, end, and width
func AddNewLines(sc *Scene, name string, points []mat32.Vec3, width mat32.Vec2) *Lines {
	ln := &Lines{}
	ln.Nm = name
	ln.Points = points
	ln.Width = width
	sc.AddMesh(ln)
	return ln
}

func MiterPts(ax, ay, bx, by, cx, cy, w2 float32) mat32.Vec2 {
	ppd := mat32.Vec2{ax - bx, ay - by}
	ppu := ppd.Normal()

	epd := mat32.Vec2{cx - bx, cy - by}
	epv := epd.Normal()

	dp := ppu.Dot(epv)
	jang := mat32.Acos(dp)
	wfact := w2 / math32.Sin(jang)

	uv := ppu.MulScalar(-wfact)
	vv := epv.MulScalar(-wfact)
	sv := uv.Add(vv)
	return sv
}

func (ln *Lines) Make(sc *Scene) {
	ln.Reset()

	np := len(ln.Points)
	if np < 2 {
		log.Printf("gi3d.Lines: %v -- need 2 or more Points\n", ln.Name())
		return
	}

	pts := ln.Points
	if ln.Close {
		pts = append(pts, ln.Points[0])
		np++
	}

	clr := gi.Color{}

	bb := mat32.Box3{}
	bb.SetEmpty()

	wdy := ln.Width.Y / 2
	wdz := ln.Width.X / 2

	pi2 := float32(math.Pi / 2)

	// logic for miter joins: https://math.stackexchange.com/questions/1849784/calculate-miter-points-of-stroked-vectors-in-cartesian-plane

	for li := 0; li < np-1; li++ {
		sp := pts[li]
		ep := pts[li+1]
		spSt := !ln.Close && li == 0
		epEd := !ln.Close && li == np-2

		swap := false
		if ep.X < sp.X {
			sp, ep = ep, sp
			spSt, epEd = epEd, spSt
			swap = true
		}

		v := ep.Sub(sp)
		vn := v.Normal()
		xyang := math32.Atan2(vn.Y, vn.X)
		xy := mat32.Vec2{wdy * math32.Cos(xyang+pi2), wdy * math32.Sin(xyang+pi2)}

		//   sypzm --- eypzm
		//   / |        / |
		// sypzp -- eypzp |
		//  | symzm --| eymzm
		//  | /       | /
		// symzp -- eymzp

		sypzp, sypzm, symzp, symzm := sp, sp, sp, sp
		eypzp, eypzm, eymzp, eymzm := ep, ep, ep, ep

		if !spSt {
			pp := sp
			if swap {
				if ln.Close && li == np-2 {
					pp = pts[1]
				} else {
					pp = pts[li+2]
				}
			} else {
				if ln.Close && li == 0 {
					pp = pts[np-2]
				} else {
					pp = pts[li-1]
				}
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
				if ln.Close && li == 0 {
					xp = pts[np-2]
				} else {
					xp = pts[li-1]
				}
			} else {
				if ln.Close && li == np-2 {
					xp = pts[1]
				} else {
					xp = pts[li+2]
				}
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
			ln.AddQuad([]mat32.Vec3{sypzm, symzm, eymzm, eypzm}, nil, clr) // back (zm)
			ln.AddQuad([]mat32.Vec3{sypzp, sypzm, eypzm, eypzp}, nil, clr) // bottom (yp, upside down)
			ln.AddQuad([]mat32.Vec3{symzm, symzp, eymzp, eymzm}, nil, clr) // top (ym)
			ln.AddQuad([]mat32.Vec3{symzp, sypzp, eypzp, eymzp}, nil, clr) // front (zp)
		} else {
			ln.AddQuad([]mat32.Vec3{symzm, sypzm, eypzm, eymzm}, nil, clr) // back (zm)
			ln.AddQuad([]mat32.Vec3{symzp, symzm, eymzm, eymzp}, nil, clr) // bottom (ym)
			ln.AddQuad([]mat32.Vec3{sypzm, sypzp, eypzp, eypzm}, nil, clr) // top (yp)
			ln.AddQuad([]mat32.Vec3{sypzp, symzp, eymzp, eypzp}, nil, clr) // front (zp)
		}

		bb.ExpandByPoints([]mat32.Vec3{sypzp, symzp, eypzp, eymzp})
		bb.ExpandByPoints([]mat32.Vec3{sypzm, symzm, eypzm, eymzm})
	}
	ln.BBox.BBox = bb
	ln.BBox.UpdateFmBBox()
}
