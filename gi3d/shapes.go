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
// and width parameters.
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
	_ = wdz

	pi2 := float32(math.Pi / 2)

	// logic for miter joins: https://math.stackexchange.com/questions/1849784/calculate-miter-points-of-stroked-vectors-in-cartesian-plane

	for li := 0; li < np-1; li++ {
		sp := pts[li]
		ep := pts[li+1]
		spSt := !ln.Close && li == 0
		epEd := !ln.Close && li == np-2

		v := ep.Sub(sp)
		vn := v.Normal()
		xyang := math32.Atan2(vn.Y, vn.X)
		// xzang := math32.Atan2(vn.Z, vn.X)

		xyp := mat32.Vec2{wdy * math32.Cos(xyang+pi2), wdy * math32.Sin(xyang+pi2)}
		xym := mat32.Vec2{wdy * math32.Cos(xyang-pi2), wdy * math32.Sin(xyang-pi2)}

		spp := sp
		spm := sp
		epp := ep
		epm := ep

		if spSt {
			spp.X += xyp.X
			spp.Y += xyp.Y
			spm.X += xym.X
			spm.Y += xym.Y
		} else {
			pp := sp
			if ln.Close && li == 0 {
				pp = pts[np-2]
			} else {
				pp = pts[li-1]
			}
			ppd := mat32.Vec2{pp.X - sp.X, pp.Y - sp.Y}
			ppu := ppd.Normal()

			epd := mat32.Vec2{ep.X - sp.X, ep.Y - sp.Y}
			epv := epd.Normal()

			dp := ppu.Dot(epv)
			jang := mat32.Acos(dp)
			wfact := wdy / math32.Sin(jang)

			uv := ppu.MulScalar(wfact)
			vv := epv.MulScalar(wfact)
			sv := uv.Add(vv)
			spp.Y -= sv.Y
			spp.X -= sv.X
			spm.Y += sv.Y
			spm.X += sv.X
		}

		if epEd {
			epp.X += xyp.X
			epp.Y += xyp.Y
			epm.X += xym.X
			epm.Y += xym.Y
		} else {
			xp := ep
			if ln.Close && li == np-2 {
				xp = pts[1]
			} else {
				xp = pts[li+2]
			}
			npd := mat32.Vec2{xp.X - ep.X, xp.Y - ep.Y}
			npu := npd.Normal()

			epd := mat32.Vec2{sp.X - ep.X, sp.Y - ep.Y}
			epv := epd.Normal()

			dp := npu.Dot(epv)
			jang := mat32.Acos(dp)
			wfact := wdy / math32.Sin(jang)

			uv := npu.MulScalar(wfact)
			vv := epv.MulScalar(wfact)
			sv := uv.Add(vv)
			epp.X -= sv.X
			epp.Y -= sv.Y
			epm.X += sv.X
			epm.Y += sv.Y
		}

		// 1  3
		// 2  4

		ln.AddQuad([]mat32.Vec3{spp, spm, epm, epp}, mat32.Vec3{0, 0, 1}, nil, clr)

		bb.ExpandByPoints([]mat32.Vec3{spp, spm, epm, epp})
	}
	ln.BBox.BBox = bb
	ln.BBox.UpdateFmBBox()

	// if ln.Close {
	// 	ln.Points = ln.Points[:np-1]
	// }
}
