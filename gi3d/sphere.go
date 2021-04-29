// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"math"

	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Sphere is a sphere mesh
type Sphere struct {
	MeshBase
	Radius     float32 `desc:"radius of the sphere"`
	WidthSegs  int     `min:"3" desc:"number of segments around the width of the sphere (32 is reasonable default for full circle)"`
	HeightSegs int     `min:"3" desc:"number of height segments (32 is reasonable default for full height)"`
	AngStart   float32 `min:"0" max:"360" step:"5" desc:"starting radial angle in degrees, relative to -1,0,0 left side starting point"`
	AngLen     float32 `min:"0" max:"360" step:"5" desc:"total radial angle to generate in degrees (max = 360)"`
	ElevStart  float32 `min:"0" max:"180" step:"5" desc:"starting elevation (height) angle in degrees - 0 = top of sphere, and Pi is bottom"`
	ElevLen    float32 `min:"0" max:"180" step:"5" desc:"total angle to generate in degrees (max = 180)"`
}

var KiT_Sphere = kit.Types.AddType(&Sphere{}, nil)

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

func (sp *Sphere) Make(sc *Scene) {
	sp.Reset()
	sp.AddSphereSector(sp.Radius, sp.WidthSegs, sp.HeightSegs, sp.AngStart, sp.AngLen, sp.ElevStart, sp.ElevLen, mat32.Vec3{})
	sp.BBox.UpdateFmBBox()
}

// AddSphereSector creates a sphere sector mesh
// with the specified radius, number of radial segments in each dimension,
// radial sector start angle and length in degrees (0 - 360), start = -1,0,0,
// elevation start angle and length in degrees (0 - 180), top = 0, bot = 180,
// offset is an arbitrary offset (for composing shapes).
func (ms *MeshBase) AddSphereSector(radius float32, widthSegs, heightSegs int, angStart, angLen, elevStart, elevLen float32, offset mat32.Vec3) {
	nVtx := (widthSegs + 1) * (heightSegs + 1)

	angStRad := mat32.DegToRad(angStart)
	angLenRad := mat32.DegToRad(angLen)
	elevStRad := mat32.DegToRad(elevStart)
	elevLenRad := mat32.DegToRad(elevLen)
	elevEndRad := elevStRad + elevLenRad

	// Create buffers
	pos := mat32.NewArrayF32(nVtx*3, nVtx*3)
	norms := mat32.NewArrayF32(nVtx*3, nVtx*3)
	uvs := mat32.NewArrayF32(nVtx*2, nVtx*2)
	idxs := mat32.NewArrayU32(0, nVtx)
	stidx := uint32(ms.Vtx.Len() / 3)

	bb := mat32.Box3{}
	bb.SetEmpty()

	idx := 0
	vtxs := make([][]uint32, 0)
	var pt, norm mat32.Vec3

	for y := 0; y <= heightSegs; y++ {
		vtxsRow := make([]uint32, 0)
		v := float32(y) / float32(heightSegs)
		for x := 0; x <= widthSegs; x++ {
			u := float32(x) / float32(widthSegs)
			px := -radius * mat32.Cos(angStRad+u*angLenRad) * mat32.Sin(elevStRad+v*elevLenRad)
			py := radius * mat32.Cos(elevStRad+v*elevLenRad)
			pz := radius * mat32.Sin(angStRad+u*angLenRad) * mat32.Sin(elevStRad+v*elevLenRad)
			pt.Set(px, py, pz)
			pt.SetAdd(offset)
			norm.Set(px, py, pz)
			norm.Normalize()

			pos.SetVec3(idx*3, pt)
			norms.SetVec3(idx*3, norm)
			uvs.Set(idx*2, u, v)
			vtxsRow = append(vtxsRow, uint32(idx))
			bb.ExpandByPoint(pt)
			idx++
		}
		vtxs = append(vtxs, vtxsRow)
	}

	for y := 0; y < heightSegs; y++ {
		for x := 0; x < widthSegs; x++ {
			v1 := vtxs[y][x+1]
			v2 := vtxs[y][x]
			v3 := vtxs[y+1][x]
			v4 := vtxs[y+1][x+1]
			if y != 0 || elevStRad > 0 {
				idxs.Append(stidx+v1, stidx+v2, stidx+v4)
			}
			if y != heightSegs-1 || elevEndRad < math.Pi {
				idxs.Append(stidx+v2, stidx+v3, stidx+v4)
			}
		}
	}

	ms.Vtx = append(ms.Vtx, pos...)
	ms.Idx = append(ms.Idx, idxs...)
	ms.Norm = append(ms.Norm, norms...)
	ms.Tex = append(ms.Tex, uvs...)

	ms.BBox.BBox.ExpandByBox(bb)
}

// AddDiskSector creates a disk (filled circle) or disk sector mesh
// with the specified radius, number of radial segments (minimum 3),
// sector start angle and angle length in degrees.
// The center of the disk is at the origin,
// and angle runs counter-clockwise on the XY plane, starting at (x,y,z)=(1,0,0).
func (ms *MeshBase) AddDiskSector(radius float32, segs int, angStart, angLen float32, offset mat32.Vec3) {
	// Validate arguments
	if segs < 3 {
		panic("Invalid argument: segments. The number of segments needs to be greater or equal to 3.")
	}
	angStRad := mat32.DegToRad(angStart)
	angLenRad := mat32.DegToRad(angLen)

	// Create buffers
	pos := mat32.NewArrayF32(0, 16)
	norms := mat32.NewArrayF32(0, 16)
	uvs := mat32.NewArrayF32(0, 16)
	idxs := mat32.NewArrayU32(0, 16)
	stidx := uint32(ms.Vtx.Len() / 3)

	bb := mat32.Box3{}
	bb.SetEmpty()

	// Append circle center position
	center := mat32.NewVec3(0, 0, 0)
	pos.AppendVec3(center)

	// Append circle center norm
	var norm mat32.Vec3
	norm.Z = 1
	norms.AppendVec3(norm)

	// Append circle center uv coordinate
	centerUV := mat32.NewVec2(0.5, 0.5)
	uvs.AppendVec2(centerUV)

	var pt mat32.Vec3
	// Generate the segments
	for i := 0; i <= segs; i++ {
		segment := angStRad + float32(i)/float32(segs)*angLenRad
		vx := float32(radius * mat32.Cos(segment))
		vy := float32(radius * mat32.Sin(segment))
		pt.Set(vx, vy, 0)
		pt.SetAdd(offset)

		// Appends vertex position, norm and uv coordinates
		pos.Append(vx, vy, 0)
		norms.AppendVec3(norm)
		uvs.Append((vx/radius+1)/2, (vy/radius+1)/2)
		bb.ExpandByPoint(pt)
	}

	for i := 1; i <= segs; i++ {
		idxs.Append(stidx+uint32(i), stidx+uint32(i)+1, 0)
	}

	ms.Vtx = append(ms.Vtx, pos...)
	ms.Idx = append(ms.Idx, idxs...)
	ms.Norm = append(ms.Norm, norms...)
	ms.Tex = append(ms.Tex, uvs...)

	ms.BBox.BBox.ExpandByBox(bb)
}
