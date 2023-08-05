// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"math"

	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/vgpu/vshape"
)

// Note: Raw line rendering via OpenGL is not very effective
// -- lines are often very thin and appearance is hardware dependent.
//
// The approach below produces consistent results across platforms,
// is very fast, and is "good enough" for most purposes.
// For high-quality vector rendering, use Embed2D with SVG etc.

// Lines are lines rendered as long thin boxes defined by points
// and width parameters.  The Mesh must be drawn in the XY plane (i.e., use Z = 0
// or a constant unless specifically relevant to have full 3D variation).
// Rotate the solid to put into other planes.
type Lines struct {
	MeshBase
	Points []mat32.Vec3 `desc:"line points (must be 2 or more)"`                                       // line points (must be 2 or more)
	Width  mat32.Vec2   `desc:"line width, Y = height perpendicular to line direction, and X = depth"` // line width, Y = height perpendicular to line direction, and X = depth
	Colors []gist.Color `desc:"optional colors for each point -- actual color interpolates between"`   // optional colors for each point -- actual color interpolates between
	Closed bool         `desc:"if true, connect the first and last points to form a closed shape"`     // if true, connect the first and last points to form a closed shape
}

var TypeLines = kit.Types.AddType(&Lines{}, nil)

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
	ln.Closed = closed
	sc.AddMesh(ln)
	return ln
}

func (ln *Lines) Sizes() (nVtx, nIdx int, hasColor bool) {
	ln.NVtx, ln.NIdx = vshape.LinesN(len(ln.Points), ln.Closed)
	ln.Color = len(ln.Colors) > 0
	return ln.NVtx, ln.NIdx, ln.Color
}

func (ln *Lines) Set(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	pos := mat32.Vec3{}
	bb := vshape.SetLines(vtxAry, normAry, texAry, idxAry, 0, 0, ln.Points, ln.Width, ln.Closed, pos)
	ln.BBox.SetBounds(bb.Min, bb.Max)
	// todo: colors!
	ln.SetMod(sc)
}

// UnitLineMesh returns the unit-sized line mesh, of name LineMeshName
func UnitLineMesh(sc *Scene) *Lines {
	lm := sc.MeshByName(LineMeshName)
	if lm != nil {
		return lm.(*Lines)
	}
	lmm := AddNewLines(sc, LineMeshName, []mat32.Vec3{{-.5, 0, 0}, {.5, 0, 0}}, mat32.Vec2{1, 1}, OpenLines)
	return lmm
}

// UnitConeMesh returns the unit-sized cone mesh, of name ConeMeshName-segs
func UnitConeMesh(sc *Scene, segs int) *Cylinder {
	nm := fmt.Sprintf("%s-%d", ConeMeshName, segs)
	cm := sc.MeshByName(nm)
	if cm != nil {
		return cm.(*Cylinder)
	}
	cmm := AddNewCone(sc, nm, 1, 1, segs, 1, true)
	return cmm
}

// AddNewLine adds a new line between two specified points, using a shared
// mesh unit line, which is rotated and positioned to go between the designated points.
func AddNewLine(sc *Scene, parent ki.Ki, name string, st, ed mat32.Vec3, width float32, clr gist.Color) *Solid {
	lm := UnitLineMesh(sc)
	ln := AddNewSolid(sc, parent, name, lm.Name())
	ln.Pose.Scale.Set(1, width, width)
	SetLineStartEnd(ln, st, ed)
	ln.Mat.Color = clr
	return ln
}

// SetLineStartEnd sets line Pose such that it starts / ends at given poitns.
func SetLineStartEnd(ln *Solid, st, ed mat32.Vec3) {
	wd := ln.Pose.Scale.Y
	d := ed.Sub(st)
	midp := st.Add(d.DivScalar(2))
	ln.Pose.Pos = midp
	dst := st.DistTo(ed)
	ln.Pose.Scale.Set(dst, wd, wd)
	dn := d.Normal()
	ln.Pose.Quat.SetFromUnitVectors(mat32.Vec3{1, 0, 0}, dn)
}

const (
	// StartArrow specifies to add a starting arrow
	StartArrow = true

	// NoStartArrow specifies not to add a starting arrow
	NoStartArrow = false

	// EndArrow specifies to add a ending arrow
	EndArrow = true

	// EndArrow specifies not to add a ending arrow
	NoEndArrow = false
)

// AddNewArrow adds a group with a new line + cone between two specified points, using shared
// mesh unit line and arrow heads, which are rotated and positioned to go between the designated points.
// The arrowSize is a multiplier on the width for the radius and length of the arrow head, with width
// providing an additional multiplicative factor for width to achieve "fat" vs. "thin" arrows.
// arrowSegs determines how many faces there are on the arrowhead -- 4 = a 4-sided pyramid, etc.
func AddNewArrow(sc *Scene, parent ki.Ki, name string, st, ed mat32.Vec3, width float32, clr gist.Color, startArrow, endArrow bool, arrowSize, arrowWidth float32, arrowSegs int) *Group {
	cm := UnitConeMesh(sc, arrowSegs)
	gp := AddNewGroup(sc, parent, name)

	asz := arrowSize * width
	awd := arrowWidth * asz
	d := ed.Sub(st)
	dn := d.Normal()

	lst := st
	led := ed
	if startArrow {
		lst.SetAdd(dn.MulScalar(asz))
	}
	if endArrow {
		led.SetAdd(dn.MulScalar(-asz))
	}
	ln := AddNewLine(sc, gp, name+"-line", lst, led, width, clr)

	if startArrow {
		ar := AddNewSolid(sc, gp, name+"-start-arrow", cm.Name())
		ar.Pose.Scale.Set(awd, asz, awd)                              // Y is up
		ar.Pose.Quat.SetFromAxisAngle(mat32.Vec3{0, 0, 1}, math.Pi/2) // rotate from XY up to -X
		ar.Pose.Quat.SetMul(ln.Pose.Quat)
		ar.Pose.Pos = st.Add(dn.MulScalar(.5 * asz))
		ar.Mat.Color = clr
	}
	if endArrow {
		ar := AddNewSolid(sc, gp, name+"-end-arrow", cm.Name())
		ar.Pose.Scale.Set(awd, asz, awd)
		ar.Pose.Quat.SetFromAxisAngle(mat32.Vec3{0, 0, 1}, -math.Pi/2) // rotate from XY up to +X
		ar.Pose.Quat.SetMul(ln.Pose.Quat)
		ar.Pose.Pos = ed.Add(dn.MulScalar(-.5 * asz))
		ar.Mat.Color = clr
	}
	return gp
}

// AddNewLineBoxMeshes adds two Meshes defining the edges of a Box.
// Meshes are named meshNm+"-front" and meshNm+"-side" -- need to be
// initialized, e.g., using sc.InitMesh()
func AddNewLineBoxMeshes(sc *Scene, meshNm string, bbox mat32.Box3, width float32) {
	wd := mat32.Vec2{width, width}
	sz := bbox.Size()
	hSz := sz.MulScalar(0.5)

	// front mesh
	fbl := mat32.Vec3{-hSz.X, -hSz.Y, 0}
	ftl := mat32.Vec3{-hSz.X, hSz.Y, 0}
	ftr := mat32.Vec3{hSz.X, hSz.Y, 0}
	fbr := mat32.Vec3{hSz.X, -hSz.Y, 0}
	AddNewLines(sc, meshNm+"-front", []mat32.Vec3{fbl, ftl, ftr, fbr}, wd, CloseLines)

	// side mesh in XY plane, Z -> X
	sbl := mat32.Vec3{-hSz.Z, -hSz.Y, 0}
	stl := mat32.Vec3{-hSz.Z, hSz.Y, 0}
	str := mat32.Vec3{hSz.Z, hSz.Y, 0}
	sbr := mat32.Vec3{hSz.Z, -hSz.Y, 0}
	AddNewLines(sc, meshNm+"-side", []mat32.Vec3{sbl, stl, str, sbr}, wd, CloseLines)
}

const (
	// Inactive is used for args indicating if node should be inactive
	Inactive = true

	// Active is used for args indicating if node should be inactive or not
	Active = false
)

// AddNewLineBox adds a new Group with Solid's and two Meshes defining the edges of a Box.
// This can be used for drawing a selection box around a Node in the scene, for example.
// offset is an arbitrary offset (for composing shapes).
// Meshes are named meshNm+"-front" and meshNm+"-side" -- need to be
// initialized, e.g., using sc.InitMesh()
// inactive indicates whether the box and solids should be flagged as inactive
// (not selectable).
func AddNewLineBox(sc *Scene, parent ki.Ki, meshNm, boxNm string, bbox mat32.Box3, width float32, clr gist.Color, inactive bool) *Group {
	sz := bbox.Size()
	hSz := sz.MulScalar(0.5)

	AddNewLineBoxMeshes(sc, meshNm, bbox, width)
	frmnm := meshNm + "-front"
	sdmnm := meshNm + "-side"

	ctr := bbox.Min.Add(hSz)
	bgp := AddNewGroup(sc, parent, boxNm)
	bgp.Pose.Pos = ctr

	bs := AddNewSolid(sc, bgp, boxNm+"-back", frmnm)
	bs.Mat.Color = clr
	bs.Pose.Pos.Set(0, 0, -hSz.Z)

	ls := AddNewSolid(sc, bgp, boxNm+"-left", sdmnm)
	ls.Mat.Color = clr
	ls.Pose.Pos.Set(-hSz.X, 0, 0)
	ls.Pose.SetAxisRotation(0, 1, 0, 90)

	rs := AddNewSolid(sc, bgp, boxNm+"-right", sdmnm)
	rs.Mat.Color = clr
	rs.Pose.Pos.Set(hSz.X, 0, 0)
	rs.Pose.SetAxisRotation(0, 1, 0, -90)

	fs := AddNewSolid(sc, bgp, boxNm+"-front", frmnm)
	fs.Mat.Color = clr
	fs.Pose.Pos.Set(0, 0, hSz.Z)

	if inactive {
		bgp.SetInactive()
		bs.SetInactive()
		ls.SetInactive()
		rs.SetInactive()
		fs.SetInactive()
	}

	return bgp
}
