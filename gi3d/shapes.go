// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/vgpu/vshape"
)

// shapes define different standard mesh shapes

///////////////////////////////////////////////////////////////////////////
//   Plane

// Plane is a flat 2D plane, which can be oriented along any
// axis facing either positive or negative
type Plane struct {
	MeshBase

	// axis along which the normal perpendicular to the plane points.  E.g., if the Y axis is specified, then it is a standard X-Z ground plane -- see also NormNeg for whether it is facing in the positive or negative of the given axis.
	NormAxis mat32.Dims `desc:"axis along which the normal perpendicular to the plane points.  E.g., if the Y axis is specified, then it is a standard X-Z ground plane -- see also NormNeg for whether it is facing in the positive or negative of the given axis."`

	// if false, the plane normal facing in the positive direction along specified NormAxis, otherwise it faces in the negative if true
	NormNeg bool `desc:"if false, the plane normal facing in the positive direction along specified NormAxis, otherwise it faces in the negative if true"`

	// 2D size of plane
	Size mat32.Vec2 `desc:"2D size of plane"`

	// number of segments to divide plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1
	Segs mat32.Vec2i `desc:"number of segments to divide plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1"`

	// offset from origin along direction of normal to the plane
	Offset float32 `desc:"offset from origin along direction of normal to the plane"`
}

var TypePlane = kit.Types.AddType(&Plane{}, nil)

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

func (pl *Plane) Sizes() (nVtx, nIdx int, hasColor bool) {
	pl.NVtx, pl.NIdx = vshape.PlaneN(int(pl.Segs.X), int(pl.Segs.Y))
	pl.Color = false
	return pl.NVtx, pl.NIdx, pl.Color
}

func (pl *Plane) Set(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	pos := mat32.Vec3{}
	sz := vshape.SetPlaneAxisSize(vtxAry, normAry, texAry, idxAry, 0, 0, pl.NormAxis, pl.NormNeg, pl.Size, pl.Segs, pl.Offset, pos)
	mn := pos.Sub(sz)
	mx := pos.Add(sz)
	pl.BBoxMu.Lock()
	pl.BBox.SetBounds(mn, mx)
	pl.BBoxMu.Unlock()
}

///////////////////////////////////////////////////////////////////////////
//   Box

// Box is a rectangular-shaped solid (cuboid)
type Box struct {
	MeshBase

	// size along each dimension
	Size mat32.Vec3 `desc:"size along each dimension"`

	// number of segments to divide each plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1
	Segs mat32.Vec3i `desc:"number of segments to divide each plane into (enforced to be at least 1) -- may potentially increase rendering quality to have > 1"`
}

var TypeBox = kit.Types.AddType(&Box{}, nil)

// AddNewBox adds Box mesh to given scene, with given name and size
func AddNewBox(sc *Scene, name string, width, height, depth float32) *Box {
	bx := &Box{}
	bx.Nm = name
	bx.Size.Set(width, height, depth)
	bx.Segs.Set(1, 1, 1)
	sc.AddMesh(bx)
	return bx
}

func (bx *Box) Sizes() (nVtx, nIdx int, hasColor bool) {
	bx.NVtx, bx.NIdx = vshape.BoxN(bx.Segs)
	bx.Color = false
	return bx.NVtx, bx.NIdx, bx.Color
}

func (bx *Box) Set(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	pos := mat32.Vec3{}
	hSz := vshape.SetBox(vtxAry, normAry, texAry, idxAry, 0, 0, bx.Size, bx.Segs, pos)
	mn := pos.Sub(hSz)
	mx := pos.Add(hSz)
	bx.BBoxMu.Lock()
	bx.BBox.SetBounds(mn, mx)
	bx.BBoxMu.Unlock()
}

///////////////////////////////////////////////////////////////////////////
//   Sphere

// Sphere is a sphere mesh
type Sphere struct {
	MeshBase

	// radius of the sphere
	Radius float32 `desc:"radius of the sphere"`

	// [min: 3] number of segments around the width of the sphere (32 is reasonable default for full circle)
	WidthSegs int `min:"3" desc:"number of segments around the width of the sphere (32 is reasonable default for full circle)"`

	// [min: 3] number of height segments (32 is reasonable default for full height)
	HeightSegs int `min:"3" desc:"number of height segments (32 is reasonable default for full height)"`

	// [min: 0] [max: 360] [step: 5] starting radial angle in degrees, relative to -1,0,0 left side starting point
	AngStart float32 `min:"0" max:"360" step:"5" desc:"starting radial angle in degrees, relative to -1,0,0 left side starting point"`

	// [min: 0] [max: 360] [step: 5] total radial angle to generate in degrees (max = 360)
	AngLen float32 `min:"0" max:"360" step:"5" desc:"total radial angle to generate in degrees (max = 360)"`

	// [min: 0] [max: 180] [step: 5] starting elevation (height) angle in degrees - 0 = top of sphere, and Pi is bottom
	ElevStart float32 `min:"0" max:"180" step:"5" desc:"starting elevation (height) angle in degrees - 0 = top of sphere, and Pi is bottom"`

	// [min: 0] [max: 180] [step: 5] total angle to generate in degrees (max = 180)
	ElevLen float32 `min:"0" max:"180" step:"5" desc:"total angle to generate in degrees (max = 180)"`
}

var TypeSphere = kit.Types.AddType(&Sphere{}, nil)

// AddNewSphere creates a sphere mesh with the specified radius,
// number of segments (resolution).
func AddNewSphere(sc *Scene, name string, radius float32, segs int) *Sphere {
	sp := &Sphere{}
	sp.Nm = name
	sp.Radius = radius
	sp.WidthSegs = segs
	sp.HeightSegs = segs
	sp.AngStart = 0
	sp.AngLen = 360
	sp.ElevStart = 0
	sp.ElevLen = 180
	sc.AddMesh(sp)
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

func (sp *Sphere) Sizes() (nVtx, nIdx int, hasColor bool) {
	sp.NVtx, sp.NIdx = vshape.SphereSectorN(sp.WidthSegs, sp.HeightSegs, sp.ElevStart, sp.ElevLen)
	sp.Color = false
	return sp.NVtx, sp.NIdx, sp.Color
}

func (sp *Sphere) Set(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	pos := mat32.Vec3{}
	bb := vshape.SetSphereSector(vtxAry, normAry, texAry, idxAry, 0, 0, sp.Radius, sp.WidthSegs, sp.HeightSegs, sp.AngStart, sp.AngLen, sp.ElevStart, sp.ElevLen, pos)
	sp.BBoxMu.Lock()
	sp.BBox.SetBounds(bb.Min, bb.Max)
	sp.BBoxMu.Unlock()
}

///////////////////////////////////////////////////////////////////////////
//   Cylinder / Cone

// Cylinder is a generalized cylinder shape, including a cone
// or truncated cone by having different size circles at either end.
// Height is up along the Y axis.
type Cylinder struct {
	MeshBase

	// height of the cylinder
	Height float32 `desc:"height of the cylinder"`

	// radius of the top -- set to 0 for a cone
	TopRad float32 `desc:"radius of the top -- set to 0 for a cone"`

	// radius of the bottom
	BotRad float32 `desc:"radius of the bottom"`

	// [min: 1] number of radial segments (32 is a reasonable default for full circle)
	RadialSegs int `min:"1" desc:"number of radial segments (32 is a reasonable default for full circle)"`

	// number of height segments
	HeightSegs int `desc:"number of height segments"`

	// render the top disc
	Top bool `desc:"render the top disc"`

	// render the bottom disc
	Bottom bool `desc:"render the bottom disc"`

	// [min: 0] [max: 360] [step: 5] starting angle in degrees, relative to -1,0,0 left side starting point
	AngStart float32 `min:"0" max:"360" step:"5" desc:"starting angle in degrees, relative to -1,0,0 left side starting point"`

	// [min: 0] [max: 360] [step: 5] total angle to generate in degrees (max 360)
	AngLen float32 `min:"0" max:"360" step:"5" desc:"total angle to generate in degrees (max 360)"`
}

var TypeCylinder = kit.Types.AddType(&Cylinder{}, nil)

// AddNewCone creates a cone mesh with the specified base radius, height,
// number of radial segments, number of height segments, and presence of a bottom cap.
// Height is along the Y axis.
func AddNewCone(sc *Scene, name string, height, radius float32, radialSegs, heightSegs int, bottom bool) *Cylinder {
	return AddNewCylinderSector(sc, name, height, 0, radius, radialSegs, heightSegs, 0, 360, false, bottom)
}

// AddNewCylinder creates a cylinder mesh with the specified radius, height,
// number of radial segments, number of height segments,
// and presence of a top and/or bottom cap.
// Height is along the Y axis.
func AddNewCylinder(sc *Scene, name string, height, radius float32, radialSegs, heightSegs int, top, bottom bool) *Cylinder {
	return AddNewCylinderSector(sc, name, height, radius, radius, radialSegs, heightSegs, 0, 360, top, bottom)
}

// AddNewCylinderSector creates a generalized cylinder (truncated cone) sector mesh
// with the specified top and bottom radii, height, number of radial segments,
// number of height segments, sector start angle in degrees,
// sector size angle in degrees, and presence of a top and/or bottom cap.
// Height is along the Y axis.
func AddNewCylinderSector(sc *Scene, name string, height, topRad, botRad float32, radialSegs, heightSegs int, angStart, angLen float32, top, bottom bool) *Cylinder {
	cy := &Cylinder{}
	cy.Nm = name
	cy.Height = height
	cy.TopRad = topRad
	cy.BotRad = botRad
	cy.RadialSegs = radialSegs
	cy.HeightSegs = heightSegs
	cy.AngStart = angStart
	cy.AngLen = angLen
	cy.Top = top
	cy.Bottom = bottom
	sc.AddMesh(cy)
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

func (cy *Cylinder) Sizes() (nVtx, nIdx int, hasColor bool) {
	cy.NVtx, cy.NIdx = vshape.CylinderSectorN(cy.RadialSegs, cy.HeightSegs, cy.Top, cy.Bottom)
	cy.Color = false
	return cy.NVtx, cy.NIdx, cy.Color
}

func (cy *Cylinder) Set(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	pos := mat32.Vec3{}
	bb := vshape.SetCylinderSector(vtxAry, normAry, texAry, idxAry, 0, 0, cy.Height, cy.TopRad, cy.BotRad, cy.RadialSegs, cy.HeightSegs, cy.AngStart, cy.AngLen, cy.Top, cy.Bottom, pos)
	cy.BBoxMu.Lock()
	cy.BBox.SetBounds(bb.Min, bb.Max)
	cy.BBoxMu.Unlock()
}

//////////////////////////////////////////////////////////////////////////
//  Capsule

// Capsule is a generalized capsule shape: a cylinder with hemisphere end caps.
// Supports different radii on each end.
// Height is along the Y axis -- total height is Height + TopRad + BotRad.
type Capsule struct {
	MeshBase

	// height of the cylinder portion
	Height float32 `desc:"height of the cylinder portion"`

	// radius of the top -- set to 0 for a cone
	TopRad float32 `desc:"radius of the top -- set to 0 for a cone"`

	// radius of the bottom
	BotRad float32 `desc:"radius of the bottom"`

	// [min: 1] number of radial segments (32 is a reasonable default for full circle)
	RadialSegs int `min:"1" desc:"number of radial segments (32 is a reasonable default for full circle)"`

	// number of height segments
	HeightSegs int `desc:"number of height segments"`

	// number of segments in the hemisphere cap ends (16 is a reasonable default)
	CapSegs int `desc:"number of segments in the hemisphere cap ends (16 is a reasonable default)"`

	// [min: 0] [max: 360] [step: 5] starting angle in degrees, relative to -1,0,0 left side starting point
	AngStart float32 `min:"0" max:"360" step:"5" desc:"starting angle in degrees, relative to -1,0,0 left side starting point"`

	// [min: 0] [max: 360] [step: 5] total angle to generate in degrees (max 360)
	AngLen float32 `min:"0" max:"360" step:"5" desc:"total angle to generate in degrees (max 360)"`
}

var TypeCapsule = kit.Types.AddType(&Capsule{}, nil)

// AddNewCapsule creates a generalized capsule mesh (cylinder + hemisphere caps)
// with the specified height and radius, number of radial, sphere segments,
// and number of height segments
// Height is along the Y axis.
func AddNewCapsule(sc *Scene, name string, height, radius float32, segs, heightSegs int) *Capsule {
	cp := &Capsule{}
	cp.Nm = name
	cp.Height = height
	cp.TopRad = radius
	cp.BotRad = radius
	cp.RadialSegs = segs
	cp.HeightSegs = heightSegs
	cp.CapSegs = segs
	cp.AngStart = 0
	cp.AngLen = 360
	sc.AddMesh(cp)
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

func (cp *Capsule) Sizes() (nVtx, nIdx int, hasColor bool) {
	nVtx, nIdx = vshape.CylinderSectorN(cp.RadialSegs, cp.HeightSegs, false, false)
	if cp.BotRad > 0 {
		nv, ni := vshape.SphereSectorN(cp.RadialSegs, cp.CapSegs, 90, 90)
		nVtx += nv
		nIdx += ni
	}
	if cp.TopRad > 0 {
		nv, ni := vshape.SphereSectorN(cp.RadialSegs, cp.CapSegs, 0, 90)
		nVtx += nv
		nIdx += ni
	}
	return
}

func (cp *Capsule) Set(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	pos := mat32.Vec3{}
	voff := 0
	ioff := 0
	bb := vshape.SetCylinderSector(vtxAry, normAry, texAry, idxAry, voff, ioff, cp.Height, cp.TopRad, cp.BotRad, cp.RadialSegs, cp.HeightSegs, cp.AngStart, cp.AngLen, false, false, pos)
	nv, ni := vshape.CylinderSectorN(cp.RadialSegs, cp.HeightSegs, false, false)
	voff += nv
	ioff += ni

	if cp.BotRad > 0 {
		ps := pos
		ps.Y -= cp.Height / 2
		cbb := vshape.SetSphereSector(vtxAry, normAry, texAry, idxAry, voff, ioff, cp.BotRad, cp.RadialSegs, cp.CapSegs, cp.AngStart, cp.AngLen, 90, 90, ps)
		bb.ExpandByBox(cbb)
		nv, ni = vshape.SphereSectorN(cp.RadialSegs, cp.CapSegs, 90, 90)
		voff += nv
		ioff += ni
	}
	if cp.TopRad > 0 {
		ps := pos
		ps.Y += cp.Height / 2
		cbb := vshape.SetSphereSector(vtxAry, normAry, texAry, idxAry, voff, ioff, cp.TopRad, cp.RadialSegs, cp.CapSegs, cp.AngStart, cp.AngLen, 0, 90, ps)
		bb.ExpandByBox(cbb)
	}
	cp.BBoxMu.Lock()
	cp.BBox.SetBounds(bb.Min, bb.Max)
	cp.BBoxMu.Unlock()
}

//////////////////////////////////////////////////////////////////////////
//  Torus

// Torus is a torus mesh, defined by the radius of the solid tube and the
// larger radius of the ring.
type Torus struct {
	MeshBase

	// larger radius of the torus ring
	Radius float32 `desc:"larger radius of the torus ring"`

	// radius of the solid tube
	TubeRadius float32 `desc:"radius of the solid tube"`

	// [min: 1] number of segments around the radius of the torus (32 is reasonable default for full circle)
	RadialSegs int `min:"1" desc:"number of segments around the radius of the torus (32 is reasonable default for full circle)"`

	// [min: 1] number of segments for the tube itself (32 is reasonable default for full height)
	TubeSegs int `min:"1" desc:"number of segments for the tube itself (32 is reasonable default for full height)"`

	// [min: 0] [max: 360] [step: 5] starting radial angle in degrees relative to 1,0,0 starting point
	AngStart float32 `min:"0" max:"360" step:"5" desc:"starting radial angle in degrees relative to 1,0,0 starting point"`

	// [min: 0] [max: 360] [step: 5] total radial angle to generate in degrees (max = 360)
	AngLen float32 `min:"0" max:"360" step:"5" desc:"total radial angle to generate in degrees (max = 360)"`
}

var TypeTorus = kit.Types.AddType(&Torus{}, nil)

// AddNewTorus creates a sphere mesh with the specified outer ring radius,
// solid tube radius, and number of segments (resolution).
func AddNewTorus(sc *Scene, name string, radius, tubeRadius float32, segs int) *Torus {
	sp := &Torus{}
	sp.Nm = name
	sp.Radius = radius
	sp.TubeRadius = tubeRadius
	sp.RadialSegs = segs
	sp.TubeSegs = segs
	sp.AngStart = 0
	sp.AngLen = 360
	sc.AddMesh(sp)
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

func (tr *Torus) Sizes() (nVtx, nIdx int, hasColor bool) {
	nVtx, nIdx = vshape.TorusSectorN(tr.RadialSegs, tr.TubeSegs)
	return
}

// Set sets points for torus in given allocated arrays
func (tr *Torus) Set(sc *Scene, vtxAry, normAry, texAry, clrAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	pos := mat32.Vec3{}
	bb := vshape.SetTorusSector(vtxAry, normAry, texAry, idxAry, 0, 0, tr.Radius, tr.TubeRadius, tr.RadialSegs, tr.TubeSegs, tr.AngStart, tr.AngLen, pos)
	tr.BBoxMu.Lock()
	tr.BBox.SetBounds(bb.Min, bb.Max)
	tr.BBoxMu.Unlock()
}
