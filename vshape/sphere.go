// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import (
	"math"

	"goki.dev/mat32/v2"
)

// Sphere is a sphere shape (can be a partial sphere too)
type Sphere struct {
	ShapeBase

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

// NewSphere returns a Sphere shape with given size
func NewSphere(radius float32, segs int) *Sphere {
	sp := &Sphere{}
	sp.Defaults()
	sp.Radius = radius
	sp.WidthSegs = segs
	sp.HeightSegs = segs
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

func (sp *Sphere) N() (nVtx, nIdx int) {
	nVtx, nIdx = SphereSectorN(sp.WidthSegs, sp.HeightSegs, sp.ElevStart, sp.ElevLen)
	return
}

// SetSphereSector sets points in given allocated arrays
func (sp *Sphere) Set(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	sp.CBBox = SetSphereSector(vtxAry, normAry, texAry, idxAry, sp.VtxOff, sp.IdxOff, sp.Radius, sp.WidthSegs, sp.HeightSegs, sp.AngStart, sp.AngLen, sp.ElevStart, sp.ElevLen, sp.Pos)
}

// SphereSectorN returns the N's for a sphere sector's
// vertex and index data with given number of segments.
// Note: In *vertex* units, not float units (i.e., x3 to get
// actual float offset in Vtx array).
func SphereSectorN(widthSegs, heightSegs int, elevStart, elevLen float32) (nVtx, nIdx int) {
	nVtx = (widthSegs + 1) * (heightSegs + 1)

	elevStRad := mat32.DegToRad(elevStart)
	elevLenRad := mat32.DegToRad(elevLen)
	elevEndRad := elevStRad + elevLenRad

	h1idx := heightSegs - 1
	if elevStRad > 0 {
		h1idx++
	}
	h2idx := heightSegs - 1
	if elevEndRad < math.Pi {
		h2idx++
	}
	nIdx = 3*h1idx*widthSegs + 3*h2idx*widthSegs
	return
}

// SetSphereSector sets a sphere sector vertex, norm, tex, index data at
// given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Idx index,
// with the specified radius, number of radial segments in each
// dimension (min 3), radial sector start
// angle and length in degrees (0 - 360), start = -1,0,0,
// elevation start angle and length in degrees (0 - 180), top = 0, bot = 180.
// pos is an arbitrary offset (for composing shapes),
// returns bounding box.
func SetSphereSector(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32, vtxOff, idxOff int, radius float32, widthSegs, heightSegs int, angStart, angLen, elevStart, elevLen float32, pos mat32.Vec3) mat32.Box3 {
	angStRad := mat32.DegToRad(angStart)
	angLenRad := mat32.DegToRad(angLen)
	elevStRad := mat32.DegToRad(elevStart)
	elevLenRad := mat32.DegToRad(elevLen)
	elevEndRad := elevStRad + elevLenRad

	if widthSegs < 3 || heightSegs < 3 {
		panic("Invalid argument: segments. The number of segments needs to be greater or equal to 3.")
	}

	bb := mat32.Box3{}
	bb.SetEmpty()

	idx := 0
	vidx := vtxOff * 3
	tidx := vtxOff * 2
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
			pt.SetAdd(pos)
			norm.Set(px, py, pz)
			norm.Normalize()

			vtxAry.SetVec3(vidx+idx*3, pt)
			normAry.SetVec3(vidx+idx*3, norm)
			texAry.Set(tidx+idx*2, u, v)
			vtxsRow = append(vtxsRow, uint32(idx))
			bb.ExpandByPoint(pt)
			idx++
		}
		vtxs = append(vtxs, vtxsRow)
	}

	vOff := uint32(vtxOff)
	ii := idxOff
	for y := 0; y < heightSegs; y++ {
		for x := 0; x < widthSegs; x++ {
			v1 := vtxs[y][x+1]
			v2 := vtxs[y][x]
			v3 := vtxs[y+1][x]
			v4 := vtxs[y+1][x+1]
			if y != 0 || elevStRad > 0 {
				idxAry.Set(ii, vOff+v1, vOff+v2, vOff+v4)
				ii += 3
			}
			if y != heightSegs-1 || elevEndRad < math.Pi {
				idxAry.Set(ii, vOff+v2, vOff+v3, vOff+v4)
				ii += 3
			}
		}
	}
	return bb
}

// DiskSectorN returns the N's for a disk sector's
// vertex and index data with given number of segments.
// Note: In *vertex* units, not float units (i.e., x3 to get
// actual float offset in Vtx array).
func DiskSectorN(segs int) (nVtx, nIdx int) {
	nVtx = segs + 2
	nIdx = 2 * (segs - 1)
	return
}

// SetDiskSector sets a disk sector vertex, norm, tex, index data at
// given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Idx index,
// with the specified radius, number of radial segments (minimum 3),
// sector start angle and angle length in degrees.
// The center of the disk is at the origin,
// and angle runs counter-clockwise on the XY plane, starting at (x,y,z)=(1,0,0).
// pos is an arbitrary offset (for composing shapes),
// returns bounding box.
func SetDiskSector(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32, vtxOff, idxOff int, radius float32, segs int, angStart, angLen float32, pos mat32.Vec3) mat32.Box3 {
	// Validate arguments
	if segs < 3 {
		panic("Invalid argument: segments. The number of segments needs to be greater or equal to 3.")
	}
	angStRad := mat32.DegToRad(angStart)
	angLenRad := mat32.DegToRad(angLen)

	idx := 0
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	bb := mat32.Box3{}
	bb.SetEmpty()

	// center position
	center := pos
	vtxAry.SetVec3(vidx, center)
	var norm mat32.Vec3
	norm.Z = 1
	normAry.SetVec3(vidx, norm)
	centerUV := mat32.NewVec2(0.5, 0.5)
	texAry.SetVec2(tidx, centerUV)
	idx++

	var pt mat32.Vec3
	// Generate the segments
	for i := 0; i <= segs; i++ {
		segment := angStRad + float32(i)/float32(segs)*angLenRad
		vx := float32(radius * mat32.Cos(segment))
		vy := float32(radius * mat32.Sin(segment))
		pt.Set(vx, vy, 0)
		pt.SetAdd(pos)

		// Appends vertex position, norm and uv coordinates
		vtxAry.Set(vidx+idx*3, vx, vy, 0)
		normAry.SetVec3(vidx+idx*3, norm)
		texAry.Set(tidx+idx*2, (vx/radius+1)/2, (vy/radius+1)/2)
		bb.ExpandByPoint(pt)
		idx++
	}

	vOff := uint32(vtxOff)
	ii := idxOff
	for i := 1; i <= segs; i++ {
		idxAry.Set(ii, vOff+uint32(i), vOff+uint32(i)+1, vOff) // ctr = last
	}
	return bb
}
