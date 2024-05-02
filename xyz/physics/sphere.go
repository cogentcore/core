// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package physics

import (
	"cogentcore.org/core/math32"
)

// Sphere is a spherical body shape.
type Sphere struct {
	BodyBase

	// radius
	Radius float32
}

func (sp *Sphere) SetBBox() {
	sp.BBox.SetBounds(math32.Vec3(-sp.Radius, -sp.Radius, -sp.Radius), math32.Vec3(sp.Radius, sp.Radius, sp.Radius))
	sp.BBox.XForm(sp.Abs.Quat, sp.Abs.Pos)
}

func (sp *Sphere) InitAbs(par *NodeBase) {
	sp.InitAbsBase(par)
	sp.SetBBox()
	sp.BBox.VelNilProject()
}

func (sp *Sphere) RelToAbs(par *NodeBase) {
	sp.RelToAbsBase(par)
	sp.SetBBox()
	sp.BBox.VelProject(sp.Abs.LinVel, 1)
}

func (sp *Sphere) Step(step float32) {
	sp.StepBase(step)
	sp.SetBBox()
	sp.BBox.VelProject(sp.Abs.LinVel, step)
}
