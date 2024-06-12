// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import (
	"math"

	"cogentcore.org/core/math32"
)

// Torus is a torus mesh, defined by the radius of the solid tube and the
// larger radius of the ring.
type Torus struct {
	ShapeBase

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

// NewTorus returns a Torus mesh with the specified outer ring radius,
// solid tube radius, and number of segments (resolution).
func NewTorus(radius, tubeRadius float32, segs int) *Torus {
	tr := &Torus{}
	tr.Defaults()
	tr.Radius = radius
	tr.TubeRadius = tubeRadius
	tr.RadialSegs = segs
	tr.TubeSegs = segs
	return tr
}

func (tr *Torus) Defaults() {
	tr.Radius = 1
	tr.TubeRadius = .1
	tr.RadialSegs = 32
	tr.TubeSegs = 32
	tr.AngStart = 0
	tr.AngLen = 360
}

func (tr *Torus) N() (numVertex, nIndex int) {
	numVertex, nIndex = TorusSectorN(tr.RadialSegs, tr.TubeSegs)
	return
}

// Set sets points for torus in given allocated arrays
func (tr *Torus) Set(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32) {
	tr.CBBox = SetTorusSector(vertexArray, normArray, textureArray, indexArray, tr.VtxOff, tr.IndexOff, tr.Radius, tr.TubeRadius, tr.RadialSegs, tr.TubeSegs, tr.AngStart, tr.AngLen, tr.Pos)
}

// TorusSectorN returns N's for a torus geometry with
// number of radial segments, number of tubular segments,
// radial sector start angle and length in degrees (0 - 360)
func TorusSectorN(radialSegs, tubeSegs int) (numVertex, nIndex int) {
	numVertex = (radialSegs + 1) * (tubeSegs + 1)
	nIndex = radialSegs * tubeSegs * 6
	return
}

// SetTorusSector sets torus sector vertex, norm, tex, index data
// at given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Index index,
// with the specified revolution radius, tube radius,
// number of radial segments, number of tubular segments,
// radial sector start angle and length in degrees (0 - 360)
// pos is an arbitrary offset (for composing shapes),
// returns bounding box.
func SetTorusSector(vertexArray, normArray, textureArray math32.ArrayF32, indexArray math32.ArrayU32, vtxOff, idxOff int, radius, tubeRadius float32, radialSegs, tubeSegs int, angStart, angLen float32, pos math32.Vector3) math32.Box3 {
	angStRad := math32.DegToRad(angStart)
	angLenRad := math32.DegToRad(angLen)

	idx := 0
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	bb := math32.Box3{}
	bb.SetEmpty()

	var center math32.Vector3
	for j := 0; j <= radialSegs; j++ {
		for i := 0; i <= tubeSegs; i++ {
			u := angStRad + float32(i)/float32(tubeSegs)*angLenRad
			v := float32(j) / float32(radialSegs) * math.Pi * 2

			center.X = radius * math32.Cos(u)
			center.Y = radius * math32.Sin(u)

			var pt math32.Vector3
			pt.X = (radius + tubeRadius*math32.Cos(v)) * math32.Cos(u)
			pt.Y = (radius + tubeRadius*math32.Cos(v)) * math32.Sin(u)
			pt.Z = tubeRadius * math32.Sin(v)
			pt.SetAdd(pos)
			vertexArray.SetVector3(vidx+idx*3, pt)
			textureArray.Set(tidx+idx*2, float32(i)/float32(tubeSegs), float32(j)/float32(radialSegs))
			normArray.SetVector3(vidx+idx*3, pt.Sub(center).Normal())
			bb.ExpandByPoint(pt)
			idx++
		}
	}

	vOff := uint32(vtxOff)
	ii := idxOff
	for j := 1; j <= radialSegs; j++ {
		for i := 1; i <= tubeSegs; i++ {
			a := (tubeSegs+1)*j + i - 1
			b := (tubeSegs+1)*(j-1) + i - 1
			c := (tubeSegs+1)*(j-1) + i
			d := (tubeSegs+1)*j + i
			indexArray.Set(ii, vOff+uint32(a), vOff+uint32(b), vOff+uint32(d), vOff+uint32(b), vOff+uint32(c), vOff+uint32(d))
			ii += 6
		}
	}
	return bb
}
