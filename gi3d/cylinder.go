// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted directly from g3n: https://github.com/g3n/engine :

// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"math"

	"github.com/chewxy/math32"
	"github.com/goki/gi/mat32"
	"github.com/goki/ki/kit"
)

// Cylinder is a generalized cylinder shape, including a cone
// or truncated cone by having different size circles at either end.
type Cylinder struct {
	MeshBase
	TopRad     float32 `desc:"radius of the top -- set to 0 for a cone"`
	BotRad     float32 `desc:"radius of the bottom"`
	Height     float32 `desc:"height of the cylinder"`
	RadSegs    int     `desc:"number of radial segments"`
	HeightSegs int     `desc:"number of height segments"`
	Top        bool    `desc:"render the top disc"`
	Bottom     bool    `desc:"render the bottom disc"`
	AngStart   float32 `desc:"starting angle in radians"`
	AngLen     float32 `desc:"total angle to generate in radians (max 2*Pi)"`
}

var KiT_Cylinder = kit.Types.AddType(&Cylinder{}, nil)

// AddNewCone creates a cone mesh with the specified base radius, height,
// number of radial segments, number of height segments, and presence of a bottom cap.
func AddNewCone(sc *Scene, name string, radius, height float32, radialSegs, heightSegs int, bottom bool) *Cylinder {
	return AddNewCylinderSector(sc, name, 0, radius, height, radialSegs, heightSegs, 0, 2*math.Pi, false, bottom)
}

// AddNewCylinder creates a cylinder mesh with the specified radius, height,
// number of radial segments, number of height segments,
// and presence of a top and/or bottom cap.
func AddNewCylinder(sc *Scene, name string, radius, height float32, radialSegs, heightSegs int, top, bottom bool) *Cylinder {
	return AddNewCylinderSector(sc, name, radius, radius, height, radialSegs, heightSegs, 0, 2*math.Pi, top, bottom)
}

// AddNewCylinderSector creates a generalized cylinder (truncated cone) sector mesh
// with the specified top and bottom radii, height, number of radial segments,
// number of height segments, sector start angle in radians,
// sector size angle in radians, and presence of a top and/or bottom cap.
func AddNewCylinderSector(sc *Scene, name string, radiusTop, radiusBottom, height float32, radialSegs, heightSegs int, angStart, angLength float32, top, bottom bool) *Cylinder {
	cy := &Cylinder{}
	cy.Nm = name
	cy.TopRad = radiusTop
	cy.BotRad = radiusBottom
	cy.Height = height
	cy.RadSegs = radialSegs
	cy.HeightSegs = heightSegs
	cy.AngStart = angStart
	cy.AngLen = angLength
	cy.Top = top
	cy.Bottom = bottom
	sc.AddMesh(cy)
	return cy
}

func (cy *Cylinder) Make(sc *Scene) {
	cy.Reset()

	hHt := cy.Height / 2
	vertices := [][]int{}
	uvsOrig := [][]mat32.Vec2{}

	// Create buffer for vertex positions
	pos := mat32.NewArrayF32(0, 0)

	for y := 0; y <= cy.HeightSegs; y++ {
		var verticesRow = []int{}
		var uvsRow = []mat32.Vec2{}
		v := float32(y) / float32(cy.HeightSegs)
		radius := v*(cy.BotRad-cy.TopRad) + cy.TopRad
		for x := 0; x <= cy.RadSegs; x++ {
			u := float32(x) / float32(cy.RadSegs)
			var vtx mat32.Vec3
			vtx.X = float32(radius * mat32.Sin(u*cy.AngLen+cy.AngStart))
			vtx.Y = float32(-v*cy.Height + hHt)
			vtx.Z = float32(radius * mat32.Cos(u*cy.AngLen+cy.AngStart))
			pos.AppendVec3(vtx)
			verticesRow = append(verticesRow, pos.Size()/3-1)
			uvsRow = append(uvsRow, mat32.Vec2{float32(u), 1.0 - float32(v)})
		}
		vertices = append(vertices, verticesRow)
		uvsOrig = append(uvsOrig, uvsRow)
	}

	tanTheta := (cy.BotRad - cy.TopRad) / cy.Height
	var na, nb mat32.Vec3

	// Create preallocated buffers for normals and uvs and buffer for indices
	npos := pos.Size()
	norms := mat32.NewArrayF32(npos, npos)
	uvs := mat32.NewArrayF32(2*npos/3, 2*npos/3)
	idxs := mat32.NewArrayU32(0, 0)

	for x := 0; x < cy.RadSegs; x++ {
		if cy.TopRad != 0 {
			pos.GetVec3(3*vertices[0][x], &na)
			pos.GetVec3(3*vertices[0][x+1], &nb)
		} else {
			pos.GetVec3(3*vertices[1][x], &na)
			pos.GetVec3(3*vertices[1][x+1], &nb)
		}

		na.Y = math32.Sqrt(float32(na.X*na.X+na.Z*na.Z)) * tanTheta
		na.Normalize()
		nb.Y = math32.Sqrt(float32(nb.X*nb.X+nb.Z*nb.Z)) * tanTheta
		nb.Normalize()

		for y := 0; y < cy.HeightSegs; y++ {
			v1 := vertices[y][x]
			v2 := vertices[y+1][x]
			v3 := vertices[y+1][x+1]
			v4 := vertices[y][x+1]

			n1 := na
			n2 := na
			n3 := nb
			n4 := nb

			uv1 := uvsOrig[y][x]
			uv2 := uvsOrig[y+1][x]
			uv3 := uvsOrig[y+1][x+1]
			uv4 := uvsOrig[y][x+1]

			idxs.Append(uint32(v1), uint32(v2), uint32(v4))
			norms.SetVec3(3*v1, n1)
			norms.SetVec3(3*v2, n2)
			norms.SetVec3(3*v4, n4)

			idxs.Append(uint32(v2), uint32(v3), uint32(v4))
			norms.SetVec3(3*v2, n2)
			norms.SetVec3(3*v3, n3)
			norms.SetVec3(3*v4, n4)

			uvs.SetVec2(2*v1, uv1)
			uvs.SetVec2(2*v2, uv2)
			uvs.SetVec2(2*v3, uv3)
			uvs.SetVec2(2*v4, uv4)
		}
	}
	// First group is the body of the cylinder
	// without the caps
	// c.AddGroup(0, idxs.Size(), 0)
	// nextGroup := idxs.Size()

	// Top cap
	if cy.Top && cy.TopRad > 0 {

		// Array of vertex indicesOrig to build used to build the faces.
		idxsOrig := []uint32{}
		nextidx := pos.Size() / 3

		// Appends top segments vertices and builds array of its idxsOrig
		var uv1, uv2, uv3 mat32.Vec2
		for x := 0; x < cy.RadSegs; x++ {
			uv1 = uvsOrig[0][x]
			uv2 = uvsOrig[0][x+1]
			uv3 = mat32.Vec2{uv2.X, 0}
			// Appends CENTER with its own UV.
			pos.Append(0, float32(hHt), 0)
			norms.Append(0, 1, 0)
			uvs.AppendVec2(uv3)
			idxsOrig = append(idxsOrig, uint32(nextidx))
			nextidx++
			// Appends vertex
			v := mat32.Vec3{}
			vi := vertices[0][x]
			pos.GetVec3(3*vi, &v)
			pos.AppendVec3(v)
			norms.Append(0, 1, 0)
			uvs.AppendVec2(uv1)
			idxsOrig = append(idxsOrig, uint32(nextidx))
			nextidx++
		}
		// Appends copy of first vertex (center)
		var vtx, normal mat32.Vec3
		var uv mat32.Vec2
		pos.GetVec3(3*int(idxsOrig[0]), &vtx)
		norms.GetVec3(3*int(idxsOrig[0]), &normal)
		uvs.GetVec2(2*int(idxsOrig[0]), &uv)
		pos.AppendVec3(vtx)
		norms.AppendVec3(normal)
		uvs.AppendVec2(uv)
		idxsOrig = append(idxsOrig, uint32(nextidx))
		nextidx++

		// Appends copy of second vertex (v1) USING LAST UV2
		pos.GetVec3(3*int(idxsOrig[1]), &vtx)
		norms.GetVec3(3*int(idxsOrig[1]), &normal)
		pos.AppendVec3(vtx)
		norms.AppendVec3(normal)
		uvs.AppendVec2(uv2)
		idxsOrig = append(idxsOrig, uint32(nextidx))
		nextidx++

		// Append faces idxsOrig
		for x := 0; x < cy.RadSegs; x++ {
			pos := 2 * x
			i1 := idxsOrig[pos]
			i2 := idxsOrig[pos+1]
			i3 := idxsOrig[pos+3]
			idxs.Append(uint32(i1), uint32(i2), uint32(i3))
		}
		// Second group is optional top cap of the cylinder
		// c.AddGroup(nextGroup, idxs.Size()-nextGroup, 1)
		// nextGroup = idxs.Size()
	}

	// Bottom cap
	if cy.Bottom && cy.BotRad > 0 {

		// Array of vertex idxsOrig to build used to build the faces.
		idxsOrig := []uint32{}
		nextidx := pos.Size() / 3

		// Appends top segments vertices and builds array of its idxsOrig
		var uv1, uv2, uv3 mat32.Vec2
		for x := 0; x < cy.RadSegs; x++ {
			uv1 = uvsOrig[cy.HeightSegs][x]
			uv2 = uvsOrig[cy.HeightSegs][x+1]
			uv3 = mat32.Vec2{uv2.X, 1}
			// Appends CENTER with its own UV.
			pos.Append(0, float32(-hHt), 0)
			norms.Append(0, -1, 0)
			uvs.AppendVec2(uv3)
			idxsOrig = append(idxsOrig, uint32(nextidx))
			nextidx++
			// Appends vertex
			v := mat32.Vec3{}
			vi := vertices[cy.HeightSegs][x]
			pos.GetVec3(3*vi, &v)
			pos.AppendVec3(v)
			norms.Append(0, -1, 0)
			uvs.AppendVec2(uv1)
			idxsOrig = append(idxsOrig, uint32(nextidx))
			nextidx++
		}

		// Appends copy of first vertex (center)
		var vtx, normal mat32.Vec3
		var uv mat32.Vec2
		pos.GetVec3(3*int(idxsOrig[0]), &vtx)
		norms.GetVec3(3*int(idxsOrig[0]), &normal)
		uvs.GetVec2(2*int(idxsOrig[0]), &uv)
		pos.AppendVec3(vtx)
		norms.AppendVec3(normal)
		uvs.AppendVec2(uv)
		idxsOrig = append(idxsOrig, uint32(nextidx))
		nextidx++

		// Appends copy of second vertex (v1) USING LAST UV2
		pos.GetVec3(3*int(idxsOrig[1]), &vtx)
		norms.GetVec3(3*int(idxsOrig[1]), &normal)
		pos.AppendVec3(vtx)
		norms.AppendVec3(normal)
		uvs.AppendVec2(uv2)
		idxsOrig = append(idxsOrig, uint32(nextidx))
		nextidx++

		// Appends faces idxsOrig
		for x := 0; x < cy.RadSegs; x++ {
			pos := 2 * x
			i1 := idxsOrig[pos]
			i2 := idxsOrig[pos+3]
			i3 := idxsOrig[pos+1]
			idxs.Append(uint32(i1), uint32(i2), uint32(i3))
		}
		// Third group is optional bottom cap of the cylinder
		// c.AddGroup(nextGroup, idxs.Size()-nextGroup, 2)
	}

	cy.Vtx = pos
	cy.Idx = idxs
	cy.Norm = norms
	cy.Tex = uvs
}
