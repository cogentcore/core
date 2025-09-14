// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package world

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/physics"
)

func (vw *View) nodePlanAdd(p *tree.Plan, nb *physics.NodeBase) {
	newFunc := nb.NewViewNode
	initFunc := nb.InitViewNode
	if newFunc == nil {
		if _, ok := nb.This.(*physics.Group); ok {
			newFunc = func() tree.Node {
				return any(tree.New[xyz.Group]()).(tree.Node)
			}
		} else if _, ok := nb.This.(physics.Body); ok {
			newFunc = func() tree.Node {
				return any(tree.New[xyz.Solid]()).(tree.Node)
			}
		} // todo: joint
	}
	if initFunc == nil {
		switch x := nb.This.(type) {
		case *physics.Group:
			initFunc = func(n tree.Node) {
				vw.groupViewInit(x, n.(*xyz.Group))
			}
		case *physics.Box:
			initFunc = func(n tree.Node) {
				vw.boxViewInit(x, n.(*xyz.Solid))
			}
		case *physics.Cylinder:
			initFunc = func(n tree.Node) {
				vw.cylinderViewInit(x, n.(*xyz.Solid))
			}
		case *physics.Capsule:
			initFunc = func(n tree.Node) {
				vw.capsuleViewInit(x, n.(*xyz.Solid))
			}
		case *physics.Sphere:
			initFunc = func(n tree.Node) {
				vw.sphereViewInit(x, n.(*xyz.Solid))
			}
		}
	}
	p.Add(nb.Name, newFunc, initFunc)
}

// groupViewInit is the default InitViewNode function for groups.
func (vw *View) groupViewInit(gp *physics.Group, vgp *xyz.Group) {
	vgp.Maker(func(p *tree.Plan) {
		for _, c := range gp.Children {
			nb := c.(physics.Node).AsNodeBase()
			vw.nodePlanAdd(p, nb)
		}
	})
	vgp.Updater(func() {
		vw.nodeUpdateViewPose(gp.AsNodeBase(), vgp.AsNodeBase())
	})
}

// nodeUpdateViewPose updates the view node pose parameters
func (vw *View) nodeUpdateViewPose(nd *physics.NodeBase, vn *xyz.NodeBase) {
	vn.Pose.Pos = nd.Rel.Pos
	vn.Pose.Quat = nd.Rel.Quat
}

// bodyViewUpdate is the default Updater function for [physics.BodyBase].
func (vw *View) bodyViewUpdate(bd *physics.BodyBase, sld *xyz.Solid) {
	vw.nodeUpdateViewPose(bd.AsNodeBase(), sld.AsNodeBase())
	if bd.Color != "" {
		sld.Material.Color = errors.Log1(colors.FromString(bd.Color))
	}
}

// boxViewInit is the default InitViewNode function for [physics.Box].
func (vw *View) boxViewInit(bx *physics.Box, sld *xyz.Solid) {
	mnm := "physics.Box"
	ms, _ := sld.Scene.MeshByName(mnm)
	if ms == nil {
		ms = xyz.NewBox(sld.Scene, mnm, 1, 1, 1)
	}
	sld.SetMeshName(mnm)
	sld.Updater(func() {
		sld.Pose.Scale = bx.Size
		vw.bodyViewUpdate(bx.AsBodyBase(), sld)
	})
}

// cylinderViewInit is the default InitViewNode function for [physics.Cylinder].
func (vw *View) cylinderViewInit(cy *physics.Cylinder, sld *xyz.Solid) {
	mnm := "physics.Cylinder"
	ms, _ := sld.Scene.MeshByName(mnm)
	if ms == nil {
		ms = xyz.NewCylinder(sld.Scene, mnm, 1, 1, 32, 1, true, true)
	}
	sld.SetMeshName(mnm)
	sld.Updater(func() {
		sld.Pose.Scale.Set(cy.BotRad, cy.Height, cy.BotRad)
		vw.bodyViewUpdate(cy.AsBodyBase(), sld)
	})
}

// capsuleViewInit is the default InitViewNode function for [physics.Capsule].
func (vw *View) capsuleViewInit(cp *physics.Capsule, sld *xyz.Solid) {
	mnm := "physics.Capsule"
	ms, _ := sld.Scene.MeshByName(mnm)
	if ms == nil {
		ms = xyz.NewCapsule(sld.Scene, mnm, 1, .2, 32, 1)
	}
	sld.SetMeshName(mnm)
	sld.Updater(func() {
		sld.Pose.Scale.Set(cp.BotRad/.2, cp.Height/1.4, cp.BotRad/.2)
		vw.bodyViewUpdate(cp.AsBodyBase(), sld)
	})
}

// sphereViewInit is the default InitViewNode function for [physics.Sphere].
func (vw *View) sphereViewInit(sp *physics.Sphere, sld *xyz.Solid) {
	mnm := "physics.Sphere"
	ms, _ := sld.Scene.MeshByName(mnm)
	if ms == nil {
		ms = xyz.NewSphere(sld.Scene, mnm, 1, 32)
	}
	sld.SetMeshName(mnm)
	sld.Updater(func() {
		sld.Pose.Scale.SetScalar(sp.Radius)
		vw.bodyViewUpdate(sp.AsBodyBase(), sld)
	})
}
