// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"math"

	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Torus is a torus mesh, defined by the radius of the solid tube and the
// larger radius of the ring.
type Torus struct {
	MeshBase
	Radius     float32 `desc:"larger radius of the torus ring"`
	TubeRadius float32 `desc:"radius of the solid tube"`
	RadialSegs int     `min:"1" desc:"number of segments around the radius of the torus (32 is reasonable default for full circle)"`
	TubeSegs   int     `min:"1" desc:"number of segments for the tube itself (32 is reasonable default for full height)"`
	AngStart   float32 `min:"0" max:"360" step:"5" desc:"starting radial angle in degrees relative to 1,0,0 starting point"`
	AngLen     float32 `min:"0" max:"360" step:"5" desc:"total radial angle to generate in degrees (max = 360)"`
}

var KiT_Torus = kit.Types.AddType(&Torus{}, nil)

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

func (sp *Torus) Make(sc *Scene) {
	sp.Reset()
	sp.AddTorusSector(sp.Radius, sp.TubeRadius, sp.RadialSegs, sp.TubeSegs, sp.AngStart, sp.AngLen, mat32.Vec3{})
	sp.BBox.UpdateFmBBox()
}

// NewTorus creates a torus geometry with the specified revolution radius, tube radius,
// number of radial segments, number of tubular segments,
// radial sector start angle and length in degrees (0 - 360)
func (ms *MeshBase) AddTorusSector(radius, tubeRadius float32, radialSegs, tubeSegs int, angStart, angLen float32, offset mat32.Vec3) {
	angStRad := mat32.DegToRad(angStart)
	angLenRad := mat32.DegToRad(angLen)

	pos := mat32.NewArrayF32(0, 0)
	norms := mat32.NewArrayF32(0, 0)
	uvs := mat32.NewArrayF32(0, 0)
	idxs := mat32.NewArrayU32(0, 0)
	stidx := uint32(ms.Vtx.Len() / 3)

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
			pt.SetAdd(offset)
			pos.AppendVec3(pt)
			bb.ExpandByPoint(pt)

			uvs.Append(float32(i)/float32(tubeSegs), float32(j)/float32(radialSegs))
			norms.AppendVec3(pt.Sub(center).Normal())
		}
	}

	for j := 1; j <= radialSegs; j++ {
		for i := 1; i <= tubeSegs; i++ {
			a := (tubeSegs+1)*j + i - 1
			b := (tubeSegs+1)*(j-1) + i - 1
			c := (tubeSegs+1)*(j-1) + i
			d := (tubeSegs+1)*j + i
			idxs.Append(stidx+uint32(a), stidx+uint32(b), stidx+uint32(d), stidx+uint32(b), stidx+uint32(c), stidx+uint32(d))
		}
	}

	ms.Vtx = append(ms.Vtx, pos...)
	ms.Idx = append(ms.Idx, idxs...)
	ms.Norm = append(ms.Norm, norms...)
	ms.Tex = append(ms.Tex, uvs...)

	ms.BBox.BBox.ExpandByBox(bb)
}
