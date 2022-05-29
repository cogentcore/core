// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vshape

import (
	"math"

	"github.com/goki/mat32"
)

// Torus is a torus mesh, defined by the radius of the solid tube and the
// larger radius of the ring.
type Torus struct {
	ShapeBase
	Radius     float32 `desc:"larger radius of the torus ring"`
	TubeRadius float32 `desc:"radius of the solid tube"`
	RadialSegs int     `min:"1" desc:"number of segments around the radius of the torus (32 is reasonable default for full circle)"`
	TubeSegs   int     `min:"1" desc:"number of segments for the tube itself (32 is reasonable default for full height)"`
	AngStart   float32 `min:"0" max:"360" step:"5" desc:"starting radial angle in degrees relative to 1,0,0 starting point"`
	AngLen     float32 `min:"0" max:"360" step:"5" desc:"total radial angle to generate in degrees (max = 360)"`
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

func (tr *Torus) N() (nVtx, nIdx int) {
	nVtx, nIdx = TorusSectorN(tr.RadialSegs, tr.TubeSegs)
	return
}

// Set sets points for torus in given allocated arrays
func (tr *Torus) Set(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32) {
	tr.CBBox = SetTorusSector(vtxAry, normAry, texAry, idxAry, tr.VtxOff, tr.IdxOff, tr.Radius, tr.TubeRadius, tr.RadialSegs, tr.TubeSegs, tr.AngStart, tr.AngLen, tr.Pos)
}

// TorusSectorN returns N's for a torus geometry with
// number of radial segments, number of tubular segments,
// radial sector start angle and length in degrees (0 - 360)
func TorusSectorN(radialSegs, tubeSegs int) (nVtx, nIdx int) {
	nVtx = (radialSegs + 1) * (tubeSegs + 1)
	nIdx = radialSegs * tubeSegs * 6
	return
}

// SetTorusSector sets torus sector vertex, norm, tex, index data
// at given starting *vertex* index (i.e., multiply this *3 to get
// actual float offset in Vtx array), and starting Idx index,
// with the specified revolution radius, tube radius,
// number of radial segments, number of tubular segments,
// radial sector start angle and length in degrees (0 - 360)
// pos is an arbitrary offset (for composing shapes),
// returns bounding box.
func SetTorusSector(vtxAry, normAry, texAry mat32.ArrayF32, idxAry mat32.ArrayU32, vtxOff, idxOff int, radius, tubeRadius float32, radialSegs, tubeSegs int, angStart, angLen float32, pos mat32.Vec3) mat32.Box3 {
	angStRad := mat32.DegToRad(angStart)
	angLenRad := mat32.DegToRad(angLen)

	idx := 0
	vidx := vtxOff * 3
	tidx := vtxOff * 2

	bb := mat32.Box3{}
	bb.SetEmpty()

	var center mat32.Vec3
	for j := 0; j <= radialSegs; j++ {
		for i := 0; i <= tubeSegs; i++ {
			u := angStRad + float32(i)/float32(tubeSegs)*angLenRad
			v := float32(j) / float32(radialSegs) * math.Pi * 2

			center.X = radius * mat32.Cos(u)
			center.Y = radius * mat32.Sin(u)

			var pt mat32.Vec3
			pt.X = (radius + tubeRadius*mat32.Cos(v)) * mat32.Cos(u)
			pt.Y = (radius + tubeRadius*mat32.Cos(v)) * mat32.Sin(u)
			pt.Z = tubeRadius * mat32.Sin(v)
			pt.SetAdd(pos)
			vtxAry.SetVec3(vidx+idx*3, pt)
			texAry.Set(tidx+idx*2, float32(i)/float32(tubeSegs), float32(j)/float32(radialSegs))
			normAry.SetVec3(vidx+idx*3, pt.Sub(center).Normal())
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
			idxAry.Set(ii, vOff+uint32(a), vOff+uint32(b), vOff+uint32(d), vOff+uint32(b), vOff+uint32(c), vOff+uint32(d))
			ii += 6
		}
	}
	return bb
}
