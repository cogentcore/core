// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import (
	"math"

	"cogentcore.org/core/math32"
)

// Sphere is a sphere shape (can be a partial sphere too)
type Sphere struct {
	ShapeBase

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

func (sp *Sphere) N() (numVertex, nIndex int) {
	numVertex, nIndex = SphereSectorN(sp.WidthSegs, sp.HeightSegs, sp.ElevStart, sp.ElevLen)
	return
}

// SetSphereSector sets points in given allocated arrays
func (sp *Sphere) Set(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32) {
	sp.CBBox = SetSphereSector(vertexArray, normArray, textureArray, indexArray, sp.VtxOff, sp.IndexOff, sp.Radius, sp.WidthSegs, sp.HeightSegs, sp.AngStart, sp.AngLen, sp.ElevStart, sp.ElevLen, sp.Pos)
}

// SphereSectorN returns the N's for a sphere sector's
// vertex and index data with given number of segments.
// Note: In *vertex* units, not float units (i.e., x3 to get
// actual float offset in Vtx array).
func SphereSectorN(widthSegs, heightSegs int, elevStart, elevLen float32) (numVertex, nIndex int) {
	numVertex = (widthSegs + 1) * (heightSegs + 1)

	elevStRad := math32.DegToRad(elevStart)
	elevLenRad := math32.DegToRad(elevLen)
	elevEndRad := elevStRad + elevLenRad

	h1idx := heightSegs - 1
	if elevStRad > 0 {
		h1idx++
	}
	h2idx := heightSegs - 1
	if elevEndRad < math.Pi {
		h2idx++
	}
	nIndex = 3*h1idx*widthSegs + 3*h2idx*widthSegs
	return
}

// SetSphereSector sets a sphere sector vertex, norm, tex, index data at
// given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Index index,
// with the specified radius, number of radial segments in each
// dimension (min 3), radial sector start
// angle and length in degrees (0 - 360), start = -1,0,0,
// elevation start angle and length in degrees (0 - 180), top = 0, bot = 180.
// pos is an arbitrary offset (for composing shapes),
// returns bounding box.
func SetSphereSector(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32, vtxOff, idxOff int, radius float32, widthSegs, heightSegs int, angStart, angLen, elevStart, elevLen float32, pos math32.Vector3) math32.Box3 {
	angStRad := math32.DegToRad(angStart)
	angLenRad := math32.DegToRad(angLen)
	elevStRad := math32.DegToRad(elevStart)
	elevLenRad := math32.DegToRad(elevLen)
	elevEndRad := elevStRad + elevLenRad

	if widthSegs < 3 || heightSegs < 3 {
		panic("Invalid argument: segments. The number of segments needs to be greater or equal to 3.")
	}

	bb := math32.Box3{}
	bb.SetEmpty()

	idx := 0
	vidx := vtxOff * 3
	tidx := vtxOff * 2
	vtxs := make([][]uint32, 0)
	var pt, norm math32.Vector3

	for y := 0; y <= heightSegs; y++ {
		vtxsRow := make([]uint32, 0)
		v := float32(y) / float32(heightSegs)
		for x := 0; x <= widthSegs; x++ {
			u := float32(x) / float32(widthSegs)
			px := -radius * math32.Cos(angStRad+u*angLenRad) * math32.Sin(elevStRad+v*elevLenRad)
			py := radius * math32.Cos(elevStRad+v*elevLenRad)
			pz := radius * math32.Sin(angStRad+u*angLenRad) * math32.Sin(elevStRad+v*elevLenRad)
			pt.Set(px, py, pz)
			pt.SetAdd(pos)
			norm.Set(px, py, pz)
			norm.SetNormal()

			vertexArray.SetVector3(vidx+idx*3, pt)
			normArray.SetVector3(vidx+idx*3, norm)
			textureArray.Set(tidx+idx*2, u, v)
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
				indexArray.Set(ii, vOff+v1, vOff+v2, vOff+v4)
				ii += 3
			}
			if y != heightSegs-1 || elevEndRad < math.Pi {
				indexArray.Set(ii, vOff+v2, vOff+v3, vOff+v4)
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
func DiskSectorN(segs int) (numVertex, nIndex int) {
	numVertex = segs + 2
	nIndex = 2 * (segs - 1)
	return
}

// SetDiskSector sets a disk sector vertex, norm, tex, index data at
// given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Index index,
// with the specified radius, number of radial segments (minimum 3),
// sector start angle and angle length in degrees.
// The center of the disk is at the origin,
// and angle runs counter-clockwise on the XY plane, starting at (x,y,z)=(1,0,0).
// pos is an arbitrary offset (for composing shapes),
// returns bounding box.
func SetDiskSector(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32, vtxOff, idxOff int, radius float32, segs int, angStart, angLen float32, pos math32.Vector3) math32.Box3 {
	// Validate arguments
	if segs < 3 {
		panic("Invalid argument: segments. The number of segments needs to be greater or equal to 3.")
	}
	angStRad := math32.DegToRad(angStart)
	angLenRad := math32.DegToRad(angLen)

	idx := 0
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	bb := math32.Box3{}
	bb.SetEmpty()

	// center position
	center := pos
	vertexArray.SetVector3(vidx, center)
	var norm math32.Vector3
	norm.Z = 1
	normArray.SetVector3(vidx, norm)
	centerUV := math32.Vec2(0.5, 0.5)
	textureArray.SetVector2(tidx, centerUV)
	idx++

	var pt math32.Vector3
	// Generate the segments
	for i := 0; i <= segs; i++ {
		segment := angStRad + float32(i)/float32(segs)*angLenRad
		vx := float32(radius * math32.Cos(segment))
		vy := float32(radius * math32.Sin(segment))
		pt.Set(vx, vy, 0)
		pt.SetAdd(pos)

		// Appends vertex position, norm and uv coordinates
		vertexArray.Set(vidx+idx*3, vx, vy, 0)
		normArray.SetVector3(vidx+idx*3, norm)
		textureArray.Set(tidx+idx*2, (vx/radius+1)/2, (vy/radius+1)/2)
		bb.ExpandByPoint(pt)
		idx++
	}

	vOff := uint32(vtxOff)
	ii := idxOff
	for i := 1; i <= segs; i++ {
		indexArray.Set(ii, vOff+uint32(i), vOff+uint32(i)+1, vOff) // ctr = last
	}
	return bb
}
