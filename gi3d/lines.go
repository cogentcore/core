// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"log"
	"math"

	"github.com/chewxy/math32"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

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

const (
	// CloseLines is used for the closed arg in AddNewLines:
	// connect first and last
	CloseLines = true

	// OpenLines is used for the closed arg in AddNewLines:
	// don't connect first and last
	OpenLines = false
)

// AddNewLines adds Lines mesh to given scene, with given start, end, and width
func AddNewLines(sc *Scene, name string, points []mat32.Vec3, width mat32.Vec2, closed bool) *Lines {
	ln := &Lines{}
	ln.Nm = name
	ln.Points = points
	ln.Width = width
	ln.Close = closed
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

// AddNewLineBox adds a new Group with Solid's and two Meshes defining the edges of a Box.
// This can be used for drawing a selection box around a Node in the scene, for example.
func AddNewLineBox(sc *Scene, parent ki.Ki, meshNm, boxNm string, bbox mat32.Box3, width float32, clr gi.Color) *Group {
	wd := mat32.Vec2{width, width}
	sz := bbox.Size()
	hSz := sz.MulScalar(0.5)

	// front mesh
	fbl := mat32.Vec3{-hSz.X, -hSz.Y, 0}
	ftl := mat32.Vec3{-hSz.X, hSz.Y, 0}
	ftr := mat32.Vec3{hSz.X, hSz.Y, 0}
	fbr := mat32.Vec3{hSz.X, -hSz.Y, 0}
	frm := AddNewLines(sc, meshNm+"-front", []mat32.Vec3{fbl, ftl, ftr, fbr}, wd, CloseLines)

	// side mesh in XY plane, Z -> X
	sbl := mat32.Vec3{-hSz.Z, -hSz.Y, 0}
	stl := mat32.Vec3{-hSz.Z, hSz.Y, 0}
	str := mat32.Vec3{hSz.Z, hSz.Y, 0}
	sbr := mat32.Vec3{hSz.Z, -hSz.Y, 0}

	sdm := AddNewLines(sc, meshNm+"-side", []mat32.Vec3{sbl, stl, str, sbr}, wd, CloseLines)

	ctr := bbox.Min.Add(hSz)
	bgp := AddNewGroup(sc, parent, boxNm)
	bgp.Pose.Pos = ctr

	bs := AddNewSolid(sc, bgp, boxNm+"-back", frm.Name())
	bs.Mat.Color = clr
	bs.Pose.Pos.Set(0, 0, -hSz.Z)

	ls := AddNewSolid(sc, bgp, boxNm+"-left", sdm.Name())
	ls.Mat.Color = clr
	ls.Pose.Pos.Set(-hSz.X, 0, 0)
	ls.Pose.SetAxisRotation(0, 1, 0, 90)

	rs := AddNewSolid(sc, bgp, boxNm+"-right", sdm.Name())
	rs.Mat.Color = clr
	rs.Pose.Pos.Set(hSz.X, 0, 0)
	rs.Pose.SetAxisRotation(0, 1, 0, -90)

	fs := AddNewSolid(sc, bgp, boxNm+"-front", frm.Name())
	fs.Mat.Color = clr
	fs.Pose.Pos.Set(0, 0, hSz.Z)

	return bgp
}
