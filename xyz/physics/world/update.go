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

// Add adds given node to the [tree.Plan], using NewView
// and InitView functions on the node, or default ones.
func Add(p *tree.Plan, nb *physics.NodeBase) {
	newFunc := nb.NewView
	initFunc := nb.InitView
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
				GroupInit(x, n.(*xyz.Group))
			}
		case *physics.Box:
			initFunc = func(n tree.Node) {
				BoxInit(x, n.(*xyz.Solid))
			}
		case *physics.Cylinder:
			initFunc = func(n tree.Node) {
				CylinderInit(x, n.(*xyz.Solid))
			}
		case *physics.Capsule:
			initFunc = func(n tree.Node) {
				CapsuleInit(x, n.(*xyz.Solid))
			}
		case *physics.Sphere:
			initFunc = func(n tree.Node) {
				SphereInit(x, n.(*xyz.Solid))
			}
		}
	}
	p.Add(nb.Name, newFunc, initFunc)
}

// GroupInit is the default InitView function for groups.
func GroupInit(gp *physics.Group, vgp *xyz.Group) {
	vgp.Maker(func(p *tree.Plan) {
		for _, c := range gp.Children {
			Add(p, c.(physics.Node).AsNodeBase())
		}
	})
	vgp.Updater(func() {
		UpdatePose(gp.AsNodeBase(), vgp.AsNodeBase())
	})
}

// UpdatePose updates the view node pose from physics node state.
func UpdatePose(nd *physics.NodeBase, vn *xyz.NodeBase) {
	vn.Pose.Pos = nd.Rel.Pos
	vn.Pose.Quat = nd.Rel.Quat
}

// UpdateColor updates the view color for [physics.BodyBase] nodes.
func UpdateColor(bd *physics.BodyBase, sld *xyz.Solid) {
	if bd.Color != "" {
		sld.Material.Color = errors.Log1(colors.FromString(bd.Color))
	}
}

// BoxInit is the default InitView function for [physics.Box].
// Only updates Pose in Updater: if node will change size or color,
// add updaters for that.
func BoxInit(bx *physics.Box, sld *xyz.Solid) {
	mnm := "physics.Box"
	if ms, _ := sld.Scene.MeshByName(mnm); ms == nil {
		xyz.NewBox(sld.Scene, mnm, 1, 1, 1)
	}
	sld.SetMeshName(mnm)
	sld.Pose.Scale = bx.Size
	UpdateColor(bx.AsBodyBase(), sld)
	sld.Updater(func() {
		UpdatePose(bx.AsNodeBase(), sld.AsNodeBase())
	})
}

// CylinderInit is the default InitView function for [physics.Cylinder].
// Only updates Pose in Updater: if node will change size or color,
// add updaters for that.
func CylinderInit(cy *physics.Cylinder, sld *xyz.Solid) {
	mnm := "physics.Cylinder"
	if ms, _ := sld.Scene.MeshByName(mnm); ms == nil {
		xyz.NewCylinder(sld.Scene, mnm, 1, 1, 32, 1, true, true)
	}
	sld.SetMeshName(mnm)
	sld.Pose.Scale.Set(cy.BotRad, cy.Height, cy.BotRad)
	UpdateColor(cy.AsBodyBase(), sld)
	sld.Updater(func() {
		UpdatePose(cy.AsNodeBase(), sld.AsNodeBase())
	})
}

// CapsuleInit is the default InitView function for [physics.Capsule].
// Only updates Pose in Updater: if node will change size or color,
// add updaters for that.
func CapsuleInit(cp *physics.Capsule, sld *xyz.Solid) {
	mnm := "physics.Capsule"
	if ms, _ := sld.Scene.MeshByName(mnm); ms == nil {
		ms = xyz.NewCapsule(sld.Scene, mnm, 1, .2, 32, 1)
	}
	sld.SetMeshName(mnm)
	sld.Pose.Scale.Set(cp.BotRad/.2, cp.Height/1.4, cp.BotRad/.2)
	UpdateColor(cp.AsBodyBase(), sld)
	sld.Updater(func() {
		UpdatePose(cp.AsNodeBase(), sld.AsNodeBase())
	})
}

// SphereInit is the default InitView function for [physics.Sphere].
// Only updates Pose in Updater: if node will change size or color,
// add updaters for that.
func SphereInit(sp *physics.Sphere, sld *xyz.Solid) {
	mnm := "physics.Sphere"
	if ms, _ := sld.Scene.MeshByName(mnm); ms == nil {
		ms = xyz.NewSphere(sld.Scene, mnm, 1, 32)
	}
	sld.SetMeshName(mnm)
	sld.Pose.Scale.SetScalar(sp.Radius)
	UpdateColor(sp.AsBodyBase(), sld)
	sld.Updater(func() {
		UpdatePose(sp.AsNodeBase(), sld.AsNodeBase())
	})
}
