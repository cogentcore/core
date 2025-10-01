// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
)

const (
	// TrackCameraName is a reserved top-level Group name -- this group
	// will have its Pose updated to match that of the camera automatically.
	TrackCameraName = "TrackCamera"

	// Plane2DMeshName is the reserved name for the 2D plane mesh
	// used for Text2D and Embed2D
	Plane2DMeshName = "__Plane2D"

	// LineMeshName is the reserved name for a unit-sized Line segment
	LineMeshName = "__UnitLine"

	// ConeMeshName is the reserved name for a unit-sized Cone segment.
	// Has the number of segments appended.
	ConeMeshName = "__UnitCone"
)

////////   Plane

// Plane is a flat 2D plane, which can be oriented along any
// axis facing either positive or negative
type Plane struct { //types:add -setters
	MeshBase

	// axis along which the normal perpendicular to the plane points.  E.g., if the Y axis is specified, then it is a standard X-Z ground plane -- see also NormalNeg for whether it is facing in the positive or negative of the given axis.
	NormAxis math32.Dims

	// if false, the plane normal facing in the positive direction along specified NormAxis, otherwise it faces in the negative if true
	NormalNeg bool

	// 2D size of plane
	Size math32.Vector2

	// number of segments to divide plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1
	Segs math32.Vector2i

	// offset from origin along direction of normal to the plane
	Offset float32
}

// NewPlane adds Plane mesh to given scene,
// with given name and size, with its normal pointing
// by default in the positive Y axis (i.e., a "ground" plane).
// Offset is 0.
func NewPlane(sc *Scene, name string, width, height float32) *Plane {
	pl := &Plane{}
	pl.Name = name
	pl.NormAxis = math32.Y
	pl.NormalNeg = false
	pl.Size.Set(width, height)
	pl.Segs.Set(1, 1)
	pl.Offset = 0
	sc.SetMesh(pl)
	return pl
}

func (pl *Plane) MeshSize() (numVertex, nIndex int, hasColor bool) {
	pl.NumVertex, pl.NumIndex = shape.PlaneN(int(pl.Segs.X), int(pl.Segs.Y))
	pl.HasColor = false
	return pl.NumVertex, pl.NumIndex, pl.HasColor
}

func (pl *Plane) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	pos := math32.Vector3{}
	sz := shape.SetPlaneAxisSize(vertex, normal, texcoord, index, 0, 0, pl.NormAxis, pl.NormalNeg, pl.Size, pl.Segs, pl.Offset, pos)
	mn := pos.Sub(sz)
	mx := pos.Add(sz)
	pl.BBox.SetBounds(mn, mx)
}

////////   Box

// Box is a rectangular-shaped solid (cuboid)
type Box struct { //types:add -setters
	MeshBase

	// size along each dimension
	Size math32.Vector3

	// number of segments to divide each plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1
	Segs math32.Vector3i
}

// NewBox adds Box mesh to given scene, with given name and size
func NewBox(sc *Scene, name string, width, height, depth float32) *Box {
	bx := &Box{}
	bx.Name = name
	bx.Size.Set(width, height, depth)
	bx.Segs.Set(1, 1, 1)
	sc.SetMesh(bx)
	return bx
}

func (bx *Box) MeshSize() (numVertex, nIndex int, hasColor bool) {
	bx.NumVertex, bx.NumIndex = shape.BoxN(bx.Segs)
	bx.HasColor = false
	return bx.NumVertex, bx.NumIndex, bx.HasColor
}

func (bx *Box) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	pos := math32.Vector3{}
	hSz := shape.SetBox(vertex, normal, texcoord, index, 0, 0, bx.Size, bx.Segs, pos)
	mn := pos.Sub(hSz)
	mx := pos.Add(hSz)
	bx.BBox.SetBounds(mn, mx)
}

////////   Sphere

// Sphere is a sphere mesh
type Sphere struct { //types:add -setters
	MeshBase

	// radius of the sphere
	Radius float32

	// number of segments around the width of the sphere (32 is reasonable default for full circle)
	WidthSegs int `min:"3"`

	// number of height segments (32 is reasonable default for full height)
	HeightSegs int `min:"3"`

	// starting radial angle in degrees, relative to -1,0,0 left side starting point
	AngStart float32 `min:"0" max:"360" step:"5"`

	// total radial angle to generate in degrees (max = 360)
	AngLen float32 `min:"0" max:"360" step:"5"`

	// starting elevation (height) angle in degrees - 0 = top of sphere, and Pi is bottom
	ElevStart float32 `min:"0" max:"180" step:"5"`

	// total angle to generate in degrees (max = 180)
	ElevLen float32 `min:"0" max:"180" step:"5"`
}

// NewSphere creates a sphere mesh with the specified radius,
// number of segments (resolution).
func NewSphere(sc *Scene, name string, radius float32, segs int) *Sphere {
	sp := &Sphere{}
	sp.Name = name
	sp.Radius = radius
	sp.WidthSegs = segs
	sp.HeightSegs = segs
	sp.AngStart = 0
	sp.AngLen = 360
	sp.ElevStart = 0
	sp.ElevLen = 180
	sc.SetMesh(sp)
	return sp
}

func (sp *Sphere) Defaults() {
	sp.Radius = 1
	sp.WidthSegs = 32
	sp.HeightSegs = 32
	sp.AngStart = 0
	sp.AngLen = 360
	sp.ElevStart = 0
	sp.ElevLen = 180
}

func (sp *Sphere) MeshSize() (numVertex, nIndex int, hasColor bool) {
	sp.NumVertex, sp.NumIndex = shape.SphereSectorN(sp.WidthSegs, sp.HeightSegs, sp.ElevStart, sp.ElevLen)
	sp.HasColor = false
	return sp.NumVertex, sp.NumIndex, sp.HasColor
}

func (sp *Sphere) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	pos := math32.Vector3{}
	bb := shape.SetSphereSector(vertex, normal, texcoord, index, 0, 0, sp.Radius, sp.WidthSegs, sp.HeightSegs, sp.AngStart, sp.AngLen, sp.ElevStart, sp.ElevLen, pos)
	sp.BBox.SetBounds(bb.Min, bb.Max)
}

////////   Cylinder / Cone

// Cylinder is a generalized cylinder shape, including a cone
// or truncated cone by having different size circles at either end.
// Height is up along the Y axis.
type Cylinder struct { //types:add -setters
	MeshBase

	// height of the cylinder
	Height float32

	// radius of the top -- set to 0 for a cone
	TopRad float32

	// radius of the bottom
	BotRad float32

	// number of radial segments (32 is a reasonable default for full circle)
	RadialSegs int `min:"1"`

	// number of height segments
	HeightSegs int

	// render the top disc
	Top bool

	// render the bottom disc
	Bottom bool

	// starting angle in degrees, relative to -1,0,0 left side starting point
	AngStart float32 `min:"0" max:"360" step:"5"`

	// total angle to generate in degrees (max 360)
	AngLen float32 `min:"0" max:"360" step:"5"`
}

// NewCone creates a cone mesh with the specified base radius, height,
// number of radial segments, number of height segments, and presence of a bottom cap.
// Height is along the Y axis.
func NewCone(sc *Scene, name string, height, radius float32, radialSegs, heightSegs int, bottom bool) *Cylinder {
	return NewCylinderSector(sc, name, height, 0, radius, radialSegs, heightSegs, 0, 360, false, bottom)
}

// NewCylinder creates a cylinder mesh with the specified radius, height,
// number of radial segments, number of height segments,
// and presence of a top and/or bottom cap.
// Height is along the Y axis.
func NewCylinder(sc *Scene, name string, height, radius float32, radialSegs, heightSegs int, top, bottom bool) *Cylinder {
	return NewCylinderSector(sc, name, height, radius, radius, radialSegs, heightSegs, 0, 360, top, bottom)
}

// NewCylinderSector creates a generalized cylinder (truncated cone) sector mesh
// with the specified top and bottom radii, height, number of radial segments,
// number of height segments, sector start angle in degrees,
// sector size angle in degrees, and presence of a top and/or bottom cap.
// Height is along the Y axis.
func NewCylinderSector(sc *Scene, name string, height, topRad, botRad float32, radialSegs, heightSegs int, angStart, angLen float32, top, bottom bool) *Cylinder {
	cy := &Cylinder{}
	cy.Name = name
	cy.Height = height
	cy.TopRad = topRad
	cy.BotRad = botRad
	cy.RadialSegs = radialSegs
	cy.HeightSegs = heightSegs
	cy.AngStart = angStart
	cy.AngLen = angLen
	cy.Top = top
	cy.Bottom = bottom
	sc.SetMesh(cy)
	return cy
}

func (cy *Cylinder) Defaults() {
	cy.Height = 1
	cy.TopRad = 0.5
	cy.BotRad = 0.5
	cy.RadialSegs = 32
	cy.HeightSegs = 32
	cy.Top = true
	cy.Bottom = true
	cy.AngStart = 0
	cy.AngLen = 360
}

func (cy *Cylinder) MeshSize() (numVertex, nIndex int, hasColor bool) {
	cy.NumVertex, cy.NumIndex = shape.CylinderSectorN(cy.RadialSegs, cy.HeightSegs, cy.Top, cy.Bottom)
	cy.HasColor = false
	return cy.NumVertex, cy.NumIndex, cy.HasColor
}

func (cy *Cylinder) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	pos := math32.Vector3{}
	bb := shape.SetCylinderSector(vertex, normal, texcoord, index, 0, 0, cy.Height, cy.TopRad, cy.BotRad, cy.RadialSegs, cy.HeightSegs, cy.AngStart, cy.AngLen, cy.Top, cy.Bottom, pos)
	cy.BBox.SetBounds(bb.Min, bb.Max)
}

////////  Capsule

// Capsule is a generalized capsule shape: a cylinder with hemisphere end caps.
// Supports different radii on each end.
// Height is along the Y axis -- total height is Height + TopRad + BotRad.
type Capsule struct { //types:add -setters
	MeshBase

	// height of the cylinder portion
	Height float32

	// radius of the top -- set to 0 for a cone
	TopRad float32

	// radius of the bottom
	BotRad float32

	// number of radial segments (32 is a reasonable default for full circle)
	RadialSegs int `min:"1"`

	// number of height segments
	HeightSegs int

	// number of segments in the hemisphere cap ends (16 is a reasonable default)
	CapSegs int

	// starting angle in degrees, relative to -1,0,0 left side starting point
	AngStart float32 `min:"0" max:"360" step:"5"`

	// total angle to generate in degrees (max 360)
	AngLen float32 `min:"0" max:"360" step:"5"`
}

// NewCapsule creates a generalized capsule mesh (cylinder + hemisphere caps)
// with the specified height and radius, number of radial, sphere segments,
// and number of height segments
// Height is along the Y axis.
func NewCapsule(sc *Scene, name string, height, radius float32, segs, heightSegs int) *Capsule {
	cp := &Capsule{}
	cp.Name = name
	cp.Height = height
	cp.TopRad = radius
	cp.BotRad = radius
	cp.RadialSegs = segs
	cp.HeightSegs = heightSegs
	cp.CapSegs = segs
	cp.AngStart = 0
	cp.AngLen = 360
	sc.SetMesh(cp)
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

func (cp *Capsule) MeshSize() (numVertex, nIndex int, hasColor bool) {
	numVertex, nIndex = shape.CylinderSectorN(cp.RadialSegs, cp.HeightSegs, false, false)
	if cp.BotRad > 0 {
		nv, ni := shape.SphereSectorN(cp.RadialSegs, cp.CapSegs, 90, 90)
		numVertex += nv
		nIndex += ni
	}
	if cp.TopRad > 0 {
		nv, ni := shape.SphereSectorN(cp.RadialSegs, cp.CapSegs, 0, 90)
		numVertex += nv
		nIndex += ni
	}
	return
}

func (cp *Capsule) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	pos := math32.Vector3{}
	voff := 0
	ioff := 0
	bb := shape.SetCylinderSector(vertex, normal, texcoord, index, voff, ioff, cp.Height, cp.TopRad, cp.BotRad, cp.RadialSegs, cp.HeightSegs, cp.AngStart, cp.AngLen, false, false, pos)
	nv, ni := shape.CylinderSectorN(cp.RadialSegs, cp.HeightSegs, false, false)
	voff += nv
	ioff += ni

	if cp.BotRad > 0 {
		ps := pos
		ps.Y -= cp.Height / 2
		cbb := shape.SetSphereSector(vertex, normal, texcoord, index, voff, ioff, cp.BotRad, cp.RadialSegs, cp.CapSegs, cp.AngStart, cp.AngLen, 90, 90, ps)
		bb.ExpandByBox(cbb)
		nv, ni = shape.SphereSectorN(cp.RadialSegs, cp.CapSegs, 90, 90)
		voff += nv
		ioff += ni
	}
	if cp.TopRad > 0 {
		ps := pos
		ps.Y += cp.Height / 2
		cbb := shape.SetSphereSector(vertex, normal, texcoord, index, voff, ioff, cp.TopRad, cp.RadialSegs, cp.CapSegs, cp.AngStart, cp.AngLen, 0, 90, ps)
		bb.ExpandByBox(cbb)
	}
	cp.BBox.SetBounds(bb.Min, bb.Max)
}

////////  Torus

// Torus is a torus mesh, defined by the radius of the solid tube and the
// larger radius of the ring.
type Torus struct { //types:add -setters
	MeshBase

	// larger radius of the torus ring
	Radius float32

	// radius of the solid tube
	TubeRadius float32

	// number of segments around the radius of the torus (32 is reasonable default for full circle)
	RadialSegs int `min:"1"`

	// number of segments for the tube itself (32 is reasonable default for full height)
	TubeSegs int `min:"1"`

	// starting radial angle in degrees relative to 1,0,0 starting point
	AngStart float32 `min:"0" max:"360" step:"5"`

	// total radial angle to generate in degrees (max = 360)
	AngLen float32 `min:"0" max:"360" step:"5"`
}

// NewTorus creates a sphere mesh with the specified outer ring radius,
// solid tube radius, and number of segments (resolution).
func NewTorus(sc *Scene, name string, radius, tubeRadius float32, segs int) *Torus {
	sp := &Torus{}
	sp.Name = name
	sp.Radius = radius
	sp.TubeRadius = tubeRadius
	sp.RadialSegs = segs
	sp.TubeSegs = segs
	sp.AngStart = 0
	sp.AngLen = 360
	sc.SetMesh(sp)
	return sp
}

func (tr *Torus) Defaults() {
	tr.Radius = 1
	tr.TubeRadius = .1
	tr.RadialSegs = 32
	tr.TubeSegs = 32
	tr.AngStart = 0
	tr.AngLen = 360
}

func (tr *Torus) MeshSize() (numVertex, nIndex int, hasColor bool) {
	numVertex, nIndex = shape.TorusSectorN(tr.RadialSegs, tr.TubeSegs)
	return
}

// Set sets points for torus in given allocated arrays
func (tr *Torus) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	pos := math32.Vector3{}
	bb := shape.SetTorusSector(vertex, normal, texcoord, index, 0, 0, tr.Radius, tr.TubeRadius, tr.RadialSegs, tr.TubeSegs, tr.AngStart, tr.AngLen, pos)
	tr.BBox.SetBounds(bb.Min, bb.Max)
}
