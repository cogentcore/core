// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted directly from g3n: https://github.com/g3n/engine :

// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Cylinder is a generalized cylinder shape, including a cone
// or truncated cone by having different size circles at either end.
// Height is up along the Y axis.
type Cylinder struct {
	MeshBase
	Height     float32 `desc:"height of the cylinder"`
	TopRad     float32 `desc:"radius of the top -- set to 0 for a cone"`
	BotRad     float32 `desc:"radius of the bottom"`
	RadialSegs int     `min:"1" desc:"number of radial segments (32 is a reasonable default for full circle)"`
	HeightSegs int     `desc:"number of height segments"`
	Top        bool    `desc:"render the top disc"`
	Bottom     bool    `desc:"render the bottom disc"`
	AngStart   float32 `min:"0" max:"360" step:"5" desc:"starting angle in degrees, relative to -1,0,0 left side starting point"`
	AngLen     float32 `min:"0" max:"360" step:"5" desc:"total angle to generate in degrees (max 360)"`
}

var KiT_Cylinder = kit.Types.AddType(&Cylinder{}, nil)

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

func (cy *Cylinder) Make(sc *Scene) {
	cy.Reset()
	cy.AddCylinderSector(cy.Height, cy.TopRad, cy.BotRad, cy.RadialSegs, cy.HeightSegs, cy.AngStart, cy.AngLen, cy.Top, cy.Bottom, mat32.Vec3{})
	cy.BBox.UpdateFmBBox()
}

//////////////////////////////////////////////////////////
//  Capsule

// Capsule is a generalized capsule shape: a cylinder with hemisphere end caps.
// Supports different radii on each end.
// Height is along the Y axis -- total height is Height + TopRad + BotRad.
type Capsule struct {
	MeshBase
	Height     float32 `desc:"height of the cylinder portion"`
	TopRad     float32 `desc:"radius of the top -- set to 0 for a cone"`
	BotRad     float32 `desc:"radius of the bottom"`
	RadialSegs int     `min:"1" desc:"number of radial segments (32 is a reasonable default for full circle)"`
	HeightSegs int     `desc:"number of height segments"`
	CapSegs    int     `desc:"number of segments in the hemisphere cap ends (16 is a reasonable default)"`
	AngStart   float32 `min:"0" max:"360" step:"5" desc:"starting angle in degrees, relative to -1,0,0 left side starting point"`
	AngLen     float32 `min:"0" max:"360" step:"5" desc:"total angle to generate in degrees (max 360)"`
}

var KiT_Capsule = kit.Types.AddType(&Capsule{}, nil)

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

func (cp *Capsule) Make(sc *Scene) {
	cp.Reset()
	if cp.BotRad > 0 {
		cp.AddSphereSector(cp.BotRad, cp.RadialSegs, cp.CapSegs, cp.AngStart, cp.AngLen, 90, 90, mat32.Vec3{0, -cp.Height / 2, 0})
	}
	cp.AddCylinderSector(cp.Height, cp.TopRad, cp.BotRad, cp.RadialSegs, cp.HeightSegs, cp.AngStart, cp.AngLen, false, false, mat32.Vec3{})
	if cp.TopRad > 0 {
		cp.AddSphereSector(cp.TopRad, cp.RadialSegs, cp.CapSegs, cp.AngStart, cp.AngLen, 0, 90, mat32.Vec3{0, cp.Height / 2, 0})
	}
	cp.BBox.UpdateFmBBox()
}

//////////////////////////////////////////////////////////
//  Mesh code

// AddNewCylinderSector creates a generalized cylinder (truncated cone) sector mesh
// with the specified top and bottom radii, height, number of radial segments,
// number of height segments, sector start angle in degrees (start = -1,0,0)
// sector size angle in degrees, and presence of a top and/or bottom cap.
// Height is along the Y axis.
// offset is an arbitrary offset (for composing shapes).
func (ms *MeshBase) AddCylinderSector(height, topRad, botRad float32, radialSegs, heightSegs int, angStart, angLen float32, top, bottom bool, offset mat32.Vec3) {
	hHt := height / 2
	vtxs := [][]int{}
	uvsOrig := [][]mat32.Vec2{}

	angStRad := mat32.DegToRad(angStart)
	angLenRad := mat32.DegToRad(angLen)

	// Create buffer for vertex positions
	pos := mat32.NewArrayF32(0, 0)
	stidx := uint32(ms.Vtx.Len() / 3)

	bb := mat32.Box3{}
	bb.SetEmpty()

	var pt mat32.Vec3
	for y := 0; y <= heightSegs; y++ {
		var vtxsRow = []int{}
		var uvsRow = []mat32.Vec2{}
		v := float32(y) / float32(heightSegs)
		radius := v*(botRad-topRad) + topRad
		for x := 0; x <= radialSegs; x++ {
			u := float32(x) / float32(radialSegs)
			pt.X = -radius * mat32.Cos(u*angLenRad+angStRad)
			pt.Y = -v*height + hHt
			pt.Z = radius * mat32.Sin(u*angLenRad+angStRad)
			pt.SetAdd(offset)
			pos.AppendVec3(pt)
			bb.ExpandByPoint(pt)
			vtxsRow = append(vtxsRow, pos.Size()/3-1)
			uvsRow = append(uvsRow, mat32.Vec2{u, 1.0 - v})
		}
		vtxs = append(vtxs, vtxsRow)
		uvsOrig = append(uvsOrig, uvsRow)
	}

	tanTheta := (botRad - topRad) / height
	var na, nb mat32.Vec3

	// Create preallocated buffers for normals and uvs and buffer for indices
	npos := pos.Size()
	norms := mat32.NewArrayF32(npos, npos)
	uvs := mat32.NewArrayF32(2*npos/3, 2*npos/3)
	idxs := mat32.NewArrayU32(0, 0)

	for x := 0; x < radialSegs; x++ {
		if topRad != 0 {
			pos.GetVec3(3*vtxs[0][x], &na)
			pos.GetVec3(3*vtxs[0][x+1], &nb)
		} else {
			pos.GetVec3(3*vtxs[1][x], &na)
			pos.GetVec3(3*vtxs[1][x+1], &nb)
		}

		na.Y = mat32.Sqrt(na.X*na.X+na.Z*na.Z) * tanTheta
		na.Normalize()
		nb.Y = mat32.Sqrt(nb.X*nb.X+nb.Z*nb.Z) * tanTheta
		nb.Normalize()

		for y := 0; y < heightSegs; y++ {
			v1 := vtxs[y][x]
			v2 := vtxs[y+1][x]
			v3 := vtxs[y+1][x+1]
			v4 := vtxs[y][x+1]

			n1 := na
			n2 := na
			n3 := nb
			n4 := nb

			uv1 := uvsOrig[y][x]
			uv2 := uvsOrig[y+1][x]
			uv3 := uvsOrig[y+1][x+1]
			uv4 := uvsOrig[y][x+1]

			idxs.Append(stidx+uint32(v1), stidx+uint32(v2), stidx+uint32(v4))
			norms.SetVec3(3*v1, n1)
			norms.SetVec3(3*v2, n2)
			norms.SetVec3(3*v4, n4)

			idxs.Append(stidx+uint32(v2), stidx+uint32(v3), stidx+uint32(v4))
			norms.SetVec3(3*v2, n2)
			norms.SetVec3(3*v3, n3)
			norms.SetVec3(3*v4, n4)

			uvs.SetVec2(2*v1, uv1)
			uvs.SetVec2(2*v2, uv2)
			uvs.SetVec2(2*v3, uv3)
			uvs.SetVec2(2*v4, uv4)
		}
	}

	// Top cap
	if top && topRad > 0 {
		// Array of vertex indicesOrig to build used to build the faces.
		idxsOrig := []uint32{}
		nextidx := pos.Size() / 3

		// Appends top segments vtxs and builds array of its idxsOrig
		var uv1, uv2, uv3 mat32.Vec2
		for x := 0; x < radialSegs; x++ {
			uv1 = uvsOrig[0][x]
			uv2 = uvsOrig[0][x+1]
			uv3 = mat32.Vec2{uv2.X, 0}
			// Appends CENTER with its own UV.
			pos.Append(0, hHt, 0)
			norms.Append(0, 1, 0)
			uvs.AppendVec2(uv3)
			idxsOrig = append(idxsOrig, uint32(nextidx))
			nextidx++
			// Appends vertex
			v := mat32.Vec3{}
			vi := vtxs[0][x]
			pos.GetVec3(3*vi, &v)
			pos.AppendVec3(v)
			norms.Append(0, 1, 0)
			uvs.AppendVec2(uv1)
			idxsOrig = append(idxsOrig, uint32(nextidx))
			nextidx++
		}
		// Appends copy of first vertex (center)
		var pt, norm mat32.Vec3
		var uv mat32.Vec2
		pos.GetVec3(3*int(idxsOrig[0]), &pt)
		norms.GetVec3(3*int(idxsOrig[0]), &norm)
		uvs.GetVec2(2*int(idxsOrig[0]), &uv)
		pos.AppendVec3(pt)
		norms.AppendVec3(norm)
		uvs.AppendVec2(uv)
		idxsOrig = append(idxsOrig, uint32(nextidx))
		nextidx++

		// Appends copy of second vertex (v1) USING LAST UV2
		pos.GetVec3(3*int(idxsOrig[1]), &pt)
		norms.GetVec3(3*int(idxsOrig[1]), &norm)
		pos.AppendVec3(pt)
		norms.AppendVec3(norm)
		uvs.AppendVec2(uv2)
		idxsOrig = append(idxsOrig, uint32(nextidx))
		nextidx++

		// Append faces idxsOrig
		for x := 0; x < radialSegs; x++ {
			pos := 2 * x
			i1 := idxsOrig[pos]
			i2 := idxsOrig[pos+1]
			i3 := idxsOrig[pos+3]
			idxs.Append(uint32(stidx+i1), uint32(stidx+i2), uint32(stidx+i3))
		}
	}

	// Bottom cap
	if bottom && botRad > 0 {
		// Array of vertex idxsOrig to build used to build the faces.
		idxsOrig := []uint32{}
		nextidx := pos.Size() / 3

		// Appends top segments vtxs and builds array of its idxsOrig
		var uv1, uv2, uv3 mat32.Vec2
		for x := 0; x < radialSegs; x++ {
			uv1 = uvsOrig[heightSegs][x]
			uv2 = uvsOrig[heightSegs][x+1]
			uv3 = mat32.Vec2{uv2.X, 1}
			// Appends CENTER with its own UV.
			pos.Append(0, -hHt, 0)
			norms.Append(0, -1, 0)
			uvs.AppendVec2(uv3)
			idxsOrig = append(idxsOrig, uint32(nextidx))
			nextidx++
			// Appends vertex
			v := mat32.Vec3{}
			vi := vtxs[heightSegs][x]
			pos.GetVec3(3*vi, &v)
			pos.AppendVec3(v)
			norms.Append(0, -1, 0)
			uvs.AppendVec2(uv1)
			idxsOrig = append(idxsOrig, uint32(nextidx))
			nextidx++
		}

		// Appends copy of first vertex (center)
		var pt, norm mat32.Vec3
		var uv mat32.Vec2
		pos.GetVec3(3*int(idxsOrig[0]), &pt)
		norms.GetVec3(3*int(idxsOrig[0]), &norm)
		uvs.GetVec2(2*int(idxsOrig[0]), &uv)
		pos.AppendVec3(pt)
		norms.AppendVec3(norm)
		uvs.AppendVec2(uv)
		idxsOrig = append(idxsOrig, uint32(nextidx))
		nextidx++

		// Appends copy of second vertex (v1) USING LAST UV2
		pos.GetVec3(3*int(idxsOrig[1]), &pt)
		norms.GetVec3(3*int(idxsOrig[1]), &norm)
		pos.AppendVec3(pt)
		norms.AppendVec3(norm)
		uvs.AppendVec2(uv2)
		idxsOrig = append(idxsOrig, uint32(nextidx))
		nextidx++

		// Appends faces idxsOrig
		for x := 0; x < radialSegs; x++ {
			pos := 2 * x
			i1 := idxsOrig[pos]
			i2 := idxsOrig[pos+3]
			i3 := idxsOrig[pos+1]
			idxs.Append(uint32(stidx+i1), uint32(stidx+i2), uint32(stidx+i3))
		}
	}

	ms.Vtx = append(ms.Vtx, pos...)
	ms.Idx = append(ms.Idx, idxs...)
	ms.Norm = append(ms.Norm, norms...)
	ms.Tex = append(ms.Tex, uvs...)

	ms.BBox.BBox.ExpandByBox(bb)
}
