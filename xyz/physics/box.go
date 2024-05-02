// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package physics

import (
	"cogentcore.org/core/math32"
)

// Box is a box body shape
type Box struct {
	BodyBase

	// size of box in each dimension (units arbitrary, as long as they are all consistent -- meters is typical)
	Size math32.Vector3
}

func (bx *Box) SetBBox() {
	bx.BBox.SetBounds(bx.Size.MulScalar(-.5), bx.Size.MulScalar(.5))
	bx.BBox.XForm(bx.Abs.Quat, bx.Abs.Pos)
}

func (bx *Box) InitAbs(par *NodeBase) {
	bx.InitAbsBase(par)
	bx.SetBBox()
	bx.BBox.VelNilProject()
}

func (bx *Box) RelToAbs(par *NodeBase) {
	bx.RelToAbsBase(par)
	bx.SetBBox()
	bx.BBox.VelProject(bx.Abs.LinVel, 1)
}

func (bx *Box) Step(step float32) {
	bx.StepBase(step)
	bx.SetBBox()
	bx.BBox.VelProject(bx.Abs.LinVel, step)
}
