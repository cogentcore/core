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

// todo: add type that manages a world, view etc -- no need to replicate that
// and change name of world package to something better -- confusing, although
// the new type will be world so actually that is ok.

// Add adds given physics node to the [tree.Plan], using NewView
// function on the node, or default.
func Add(p *tree.Plan, nb *physics.NodeBase) {
	newFunc := nb.NewView
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
	p.Add(nb.Name, newFunc, func(n tree.Node) { Init(nb, n) })
}

// Init is the physics node initialization function,
// which calls InitView if set on the node, or the default.
func Init(nb *physics.NodeBase, n tree.Node) {
	initFunc := nb.InitView
	if initFunc != nil {
		initFunc(n)
		return
	}
	switch x := nb.This.(type) {
	case *physics.Group:
		GroupInit(x, n.(*xyz.Group))
	case *physics.Box:
		BoxInit(x, n.(*xyz.Solid))
	case *physics.Cylinder:
		CylinderInit(x, n.(*xyz.Solid))
	case *physics.Capsule:
		CapsuleInit(x, n.(*xyz.Solid))
	case *physics.Sphere:
		SphereInit(x, n.(*xyz.Solid))
	}
}

// GroupInit is the default InitView function for groups.
func GroupInit(gp *physics.Group, vgp *xyz.Group) {
	gp.View = vgp.This
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

// UpdateColor updates the view color to given color.
func UpdateColor(clr string, sld *xyz.Solid) {
	if clr != "" {
		sld.Material.Color = errors.Log1(colors.FromString(clr))
	}
}

// BoxInit is the default InitView function for [physics.Box].
// Only updates Pose in Updater: if node will change size or color,
// add updaters for that.
func BoxInit(bx *physics.Box, sld *xyz.Solid) {
	bx.View = sld.This
	mnm := "physics.Box"
	if ms, _ := sld.Scene.MeshByName(mnm); ms == nil {
		xyz.NewBox(sld.Scene, mnm, 1, 1, 1)
	}
	sld.SetMeshName(mnm)
	sld.Pose.Scale = bx.Size
	UpdateColor(bx.Color, sld)
	sld.Updater(func() {
		UpdatePose(bx.AsNodeBase(), sld.AsNodeBase())
	})
}

// CylinderInit is the default InitView function for [physics.Cylinder].
// Only updates Pose in Updater: if node will change size or color,
// add updaters for that.
func CylinderInit(cy *physics.Cylinder, sld *xyz.Solid) {
	cy.View = sld.This
	mnm := "physics.Cylinder"
	if ms, _ := sld.Scene.MeshByName(mnm); ms == nil {
		xyz.NewCylinder(sld.Scene, mnm, 1, 1, 32, 1, true, true)
	}
	sld.SetMeshName(mnm)
	sld.Pose.Scale.Set(cy.BotRad, cy.Height, cy.BotRad)
	UpdateColor(cy.Color, sld)
	sld.Updater(func() {
		UpdatePose(cy.AsNodeBase(), sld.AsNodeBase())
	})
}

// CapsuleInit is the default InitView function for [physics.Capsule].
// Only updates Pose in Updater: if node will change size or color,
// add updaters for that.
func CapsuleInit(cp *physics.Capsule, sld *xyz.Solid) {
	cp.View = sld.This
	mnm := "physics.Capsule"
	if ms, _ := sld.Scene.MeshByName(mnm); ms == nil {
		ms = xyz.NewCapsule(sld.Scene, mnm, 1, .2, 32, 1)
	}
	sld.SetMeshName(mnm)
	sld.Pose.Scale.Set(cp.BotRad/.2, cp.Height/1.4, cp.BotRad/.2)
	UpdateColor(cp.Color, sld)
	sld.Updater(func() {
		UpdatePose(cp.AsNodeBase(), sld.AsNodeBase())
	})
}

// SphereInit is the default InitView function for [physics.Sphere].
// Only updates Pose in Updater: if node will change size or color,
// add updaters for that.
func SphereInit(sp *physics.Sphere, sld *xyz.Solid) {
	sp.View = sld.This
	mnm := "physics.Sphere"
	if ms, _ := sld.Scene.MeshByName(mnm); ms == nil {
		ms = xyz.NewSphere(sld.Scene, mnm, 1, 32)
	}
	sld.SetMeshName(mnm)
	sld.Pose.Scale.SetScalar(sp.Radius)
	UpdateColor(sp.Color, sld)
	sld.Updater(func() {
		UpdatePose(sp.AsNodeBase(), sld.AsNodeBase())
	})
}
