// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"image/color"
	"math"

	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
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

	// line points (must be 2 or more)
	Points []math32.Vector3

	// line width, Y = height perpendicular to line direction, and X = depth
	Width math32.Vector2

	// optional colors for each point -- actual color interpolates between
	Colors []color.RGBA

	// if true, connect the first and last points to form a closed shape
	Closed bool
}

const (
	// CloseLines is used for the closed arg in NewLines:
	// connect first and last
	CloseLines = true

	// OpenLines is used for the closed arg in NewLines:
	// don't connect first and last
	OpenLines = false
)

// NewLines adds Lines mesh to given scene, with given start, end, and width
func NewLines(sc *Scene, name string, points []math32.Vector3, width math32.Vector2, closed bool) *Lines {
	ln := &Lines{}
	ln.Name = name
	ln.Points = points
	ln.Width = width
	ln.Closed = closed
	sc.SetMesh(ln)
	return ln
}

func (ln *Lines) MeshSize() (numVertex, nIndex int, hasColor bool) {
	ln.NumVertex, ln.NumIndex = shape.LinesN(len(ln.Points), ln.Closed)
	ln.HasColor = len(ln.Colors) > 0
	return ln.NumVertex, ln.NumIndex, ln.HasColor
}

func (ln *Lines) Set(vertex, normal, texcoord, clrs math32.ArrayF32, indexArray math32.ArrayU32) {
	pos := math32.Vector3{}
	bb := shape.SetLines(vertex, normal, texcoord, indexArray, 0, 0, ln.Points, ln.Width, ln.Closed, pos)
	ln.BBox.SetBounds(bb.Min, bb.Max)
	// todo: colors!
}

// UnitLineMesh returns the unit-sized line mesh, of name LineMeshName
func UnitLineMesh(sc *Scene) *Lines {
	lm, _ := sc.MeshByName(LineMeshName)
	if lm != nil {
		return lm.(*Lines)
	}
	lmm := NewLines(sc, LineMeshName, []math32.Vector3{{-.5, 0, 0}, {.5, 0, 0}}, math32.Vec2(1, 1), OpenLines)
	return lmm
}

// UnitConeMesh returns the unit-sized cone mesh, of name ConeMeshName-segs
func UnitConeMesh(sc *Scene, segs int) *Cylinder {
	nm := fmt.Sprintf("%s-%d", ConeMeshName, segs)
	cm, _ := sc.MeshByName(nm)
	if cm != nil {
		return cm.(*Cylinder)
	}
	cmm := NewCone(sc, nm, 1, 1, segs, 1, true)
	return cmm
}

// SetLineStartEnd sets line Pose such that it starts / ends at given poitns.
func SetLineStartEnd(pose *Pose, st, ed math32.Vector3) {
	wd := pose.Scale.Y
	d := ed.Sub(st)
	midp := st.Add(d.DivScalar(2))
	pose.Pos = midp
	dst := st.DistanceTo(ed)
	pose.Scale.Set(dst, wd, wd)
	dn := d.Normal()
	pose.Quat.SetFromUnitVectors(math32.Vec3(1, 0, 0), dn)
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

// MakeLine returns a Maker function for making a line
// between two specified points, using a shared
// mesh unit line, which is rotated and positioned
// to go between the designated points.
func MakeLine(sc *Scene, st, ed math32.Vector3, width float32, clr color.RGBA) func(ln *Solid) {
	lm := UnitLineMesh(sc)
	return func(ln *Solid) {
		ln.SetMesh(lm)
		ln.isLinear = true
		ln.Pose.Scale.Set(1, width, width)
		SetLineStartEnd(&ln.Pose, st, ed)
		ln.Material.Color = clr
	}
}

// MakeArrow returns a Maker function for making a group with a new line + cone
// between two specified points, using shared mesh unit line and arrow heads,
// which are rotated and positioned to go between the designated points.
// The arrowSize is a multiplier on the width for the radius and length
// of the arrow head, with width providing an additional multiplicative
// factor for width to achieve "fat" vs. "thin" arrows.
// arrowSegs determines how many faces there are on the arrowhead
// 4 = a 4-sided pyramid, etc.
func MakeArrow(sc *Scene, st, ed math32.Vector3, width float32, clr color.RGBA, startArrow, endArrow bool, arrowSize, arrowWidth float32, arrowSegs int) func(g *Group) {
	cm := UnitConeMesh(sc, arrowSegs)
	return func(g *Group) {
		g.isLinear = true
		d := ed.Sub(st)
		dst := d.Length()
		awd := arrowSize * arrowWidth
		asz := (arrowSize * width) / dst
		hasz := 0.5 * asz

		g.Maker(func(p *tree.Plan) {
			tree.Add(p, func(ln *Solid) {
				MakeLine(sc, st, ed, width, clr)(ln)
				ln.Pose.SetIdentity()
				switch {
				case startArrow && endArrow:
					ln.Pose.Scale.X -= 2 * asz
				case startArrow:
					ln.Pose.Scale.X -= asz
					ln.Pose.Pos.X += hasz
				case endArrow:
					ln.Pose.Scale.X -= asz
					ln.Pose.Pos.X -= hasz
				}
			})

			g.Pose.Scale.Set(1, width, width) // group does everything
			SetLineStartEnd(&g.Pose, st, ed)

			if startArrow {
				tree.Add(p, func(ar *Solid) {
					ar.SetMesh(cm)
					ar.Pose.Scale.Set(awd, asz, awd)                               // Y is up
					ar.Pose.Quat.SetFromAxisAngle(math32.Vec3(0, 0, 1), math.Pi/2) // rotate from XY up to -X
					ar.Pose.Pos = math32.Vec3(-0.5+hasz, 0, 0)
					ar.Material.Color = clr
				})
			}
			if endArrow {
				tree.Add(p, func(ar *Solid) {
					ar.SetMesh(cm)
					ar.Pose.Scale.Set(awd, asz, awd)
					ar.Pose.Quat.SetFromAxisAngle(math32.Vec3(0, 0, 1), -math.Pi/2) // rotate from XY up to +X
					ar.Pose.Pos = math32.Vec3(0.5-hasz, 0, 0)
					ar.Material.Color = clr
				})
			}
		})
	}
}

// NewLineBoxMeshes adds two Meshes defining the edges of a Box.
// Meshes are named meshNm+"-front" and meshNm+"-side" -- need to be
// initialized, e.g., using sc.InitMesh()
func NewLineBoxMeshes(sc *Scene, meshNm string, bbox math32.Box3, width float32) (front, side *Lines) {
	wd := math32.Vec2(width, width)
	sz := bbox.Size()
	hSz := sz.MulScalar(0.5)

	// front mesh
	fbl := math32.Vec3(-hSz.X, -hSz.Y, 0)
	ftl := math32.Vec3(-hSz.X, hSz.Y, 0)
	ftr := math32.Vec3(hSz.X, hSz.Y, 0)
	fbr := math32.Vec3(hSz.X, -hSz.Y, 0)
	front = NewLines(sc, meshNm+"-front", []math32.Vector3{fbl, ftl, ftr, fbr}, wd, CloseLines)

	// side mesh in XY plane, Z -> X
	sbl := math32.Vec3(-hSz.Z, -hSz.Y, 0)
	stl := math32.Vec3(-hSz.Z, hSz.Y, 0)
	str := math32.Vec3(hSz.Z, hSz.Y, 0)
	sbr := math32.Vec3(hSz.Z, -hSz.Y, 0)
	side = NewLines(sc, meshNm+"-side", []math32.Vector3{sbl, stl, str, sbr}, wd, CloseLines)
	return
}

const (
	// Inactive is used for args indicating if node should be inactive
	Inactive = true

	// Active is used for args indicating if node should be inactive or not
	Active = false
)

// MakeLineBox returns a Maker function that adds a new Group with Solids
// and two Meshes defining the edges of a Box.
// This can be used for drawing a selection box around a Node in the scene,
// for example.
// offset is an arbitrary offset (for composing shapes).
// Meshes are named meshNm+"-front" and meshNm+"-side" -- need to be
// initialized, e.g., using sc.InitMesh()
// inactive indicates whether the box and solids should be flagged as inactive
// (not selectable).
func MakeLineBox(sc *Scene, meshNm string, bbox math32.Box3, width float32, clr color.RGBA, inactive bool) func(g *Group) {
	sz := bbox.Size()
	hSz := sz.MulScalar(0.5)
	front, side := NewLineBoxMeshes(sc, meshNm, bbox, width)
	ctr := bbox.Min.Add(hSz)

	return func(g *Group) {
		g.Pose.Pos = ctr

		bs := NewSolid(g).SetMesh(front).SetColor(clr)
		bs.SetName("back")
		bs.Pose.Pos.Set(0, 0, -hSz.Z)

		ls := NewSolid(g).SetMesh(side).SetColor(clr)
		ls.SetName("left")
		ls.Pose.Pos.Set(-hSz.X, 0, 0)
		ls.Pose.SetAxisRotation(0, 1, 0, 90)

		rs := NewSolid(g).SetMesh(side).SetColor(clr)
		rs.SetName("right")
		rs.Pose.Pos.Set(hSz.X, 0, 0)
		rs.Pose.SetAxisRotation(0, 1, 0, -90)

		fs := NewSolid(g).SetMesh(front).SetColor(clr)
		fs.SetName("front")

		fs.Pose.Pos.Set(0, 0, hSz.Z)

		// todo:
		// if inactive {
		// 	g.SetDisabled()
		// 	bs.SetDisabled()
		// 	ls.SetDisabled()
		// 	rs.SetDisabled()
		// 	fs.SetDisabled()
		// }
	}
}
