// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package world

//go:generate core generate -add-types

import (
	"fmt"
	"image"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/physics"
)

// View connects a Virtual World with a Xyz Scene to visualize the world,
// including ability to render offscreen
type View struct {

	// the root Group node of the virtual world
	World *physics.Group

	// the scene object for visualizing
	Scene *xyz.Scene

	// the root Group node in the Scene under which the world is rendered
	Root *xyz.Group
}

// NewView returns a new View that links given world with given scene and root group
func NewView(world *physics.Group, sc *xyz.Scene, root *xyz.Group) *View {
	vw := &View{World: world, Scene: sc, Root: root}
	return vw
}

// InitLibrary initializes Scene library with basic Solid shapes
// based on bodies in the virtual world.  More complex visualizations
// can be configured after this.
func (vw *View) InitLibrary() {
	vw.InitLibraryBody(vw.World, vw.Scene)
}

// Sync synchronizes the view to the world
func (vw *View) Sync() bool {
	rval := vw.SyncNode(vw.World, vw.Root, vw.Scene)
	return rval
}

// UpdatePose updates the view pose values only from world tree.
// Essential that both trees are already synchronized.
func (vw *View) UpdatePose() {
	vw.UpdatePoseNode(vw.World, vw.Root)
	vw.Scene.NeedsUpdate()
}

// UpdateBodyView updates the display properties of given body name
// recurses the tree until this body name is found.
func (vw *View) UpdateBodyView(bodyNames ...string) {
	vw.UpdateBodyViewNode(bodyNames, vw.World, vw.Root)
	vw.Scene.NeedsUpdate()
}

// RenderOffNode does an offscreen render using given node
// for the camera position and orientation.
// Current scene camera is saved and restored
func (vw *View) RenderOffNode(node physics.Node, cam *Camera) error {
	sc := vw.Scene
	sc.UpdateNodesIfNeeded()
	camnm := "eve-view-renderoff-save"
	sc.SaveCamera(camnm)
	defer sc.SetCamera(camnm)
	sc.Camera.FOV = cam.FOV
	sc.Camera.Near = cam.Near
	sc.Camera.Far = cam.Far
	nb := node.AsNodeBase()
	sc.Camera.Pose.Pos = nb.Abs.Pos
	sc.Camera.Pose.Quat = nb.Abs.Quat
	sc.Camera.Pose.Scale.Set(1, 1, 1)
	sz := sc.Geom.Size
	sc.Geom.Size = cam.Size
	sc.Frame.SetSize(sc.Geom.Size) // nop if same
	ok := sc.Render()
	sc.Geom.Size = sz
	if !ok {
		return fmt.Errorf("could not render scene")
	}
	return nil
}

// Image returns the current rendered image
func (vw *View) Image() (*image.RGBA, error) {
	return vw.Scene.ImageCopy()
}

// DepthImage returns the current rendered depth image
func (vw *View) DepthImage() ([]float32, error) {
	return vw.Scene.DepthImage()
}

///////////////////////////////////////////////////////////////
// Sync, Config

// InitLibraryBody initializes Scene library with basic Solid shapes
// based on bodies in the virtual world.  More complex visualizations
// can be configured after this.
func (vw *View) InitLibraryBody(wn physics.Node, sc *xyz.Scene) {
	bod := wn.AsBody()
	if bod != nil {
		vw.InitLibSolid(bod, sc)
	}
	for idx := range wn.AsTree().Children {
		wk := wn.Child(idx).(physics.Node)
		vw.InitLibraryBody(wk, sc)
	}
}

// InitLibSolid initializes Scene library with Solid for given body
func (vw *View) InitLibSolid(bod physics.Body, sc *xyz.Scene) {
	nm := bod.Name()
	bb := bod.AsBodyBase()
	if bb.Vis == "" {
		bb.Vis = nm
	}
	if _, has := sc.Library[nm]; has {
		return
	}
	lgp := sc.NewInLibrary(nm)
	sld := xyz.NewSolid(lgp)
	sld.SetName(nm)
	wt := bod.NodeType().ShortName()
	switch wt {
	case "physics.Box":
		mnm := "eveBox"
		bm := sc.MeshByName(mnm)
		if bm == nil {
			bm = xyz.NewBox(sc, mnm, 1, 1, 1)
		}
		sld.SetMeshName(mnm)
	case "physics.Cylinder":
		mnm := "eveCylinder"
		cm := sc.MeshByName(mnm)
		if cm == nil {
			cm = xyz.NewCylinder(sc, mnm, 1, 1, 32, 1, true, true)
		}
		sld.SetMeshName(mnm)
	case "physics.Capsule":
		mnm := "eveCapsule"
		cm := sc.MeshByName(mnm)
		if cm == nil {
			cm = xyz.NewCapsule(sc, mnm, 1, .2, 32, 1)
		}
		sld.SetMeshName(mnm)
	case "physics.Sphere":
		mnm := "eveSphere"
		sm := sc.MeshByName(mnm)
		if sm == nil {
			sm = xyz.NewSphere(sc, mnm, 1, 32)
		}
		sld.SetMeshName(mnm)
	}
}

// ConfigBodySolid configures a solid for a body with current values
func (vw *View) ConfigBodySolid(bod physics.Body, sld *xyz.Solid) {
	wt := bod.NodeType().ShortName()
	switch wt {
	case "physics.Box":
		bx := bod.(*physics.Box)
		sld.Pose.Scale = bx.Size
		if bx.Color != "" {
			sld.Mat.Color = errors.Log1(colors.FromString(bx.Color))
		}
	case "physics.Cylinder":
		cy := bod.(*physics.Cylinder)
		sld.Pose.Scale.Set(cy.BotRad, cy.Height, cy.BotRad)
		if cy.Color != "" {
			sld.Mat.Color = errors.Log1(colors.FromString(cy.Color))
		}
	case "physics.Capsule":
		cp := bod.(*physics.Capsule)
		sld.Pose.Scale.Set(cp.BotRad/.2, cp.Height/1.4, cp.BotRad/.2)
		if cp.Color != "" {
			sld.Mat.Color = errors.Log1(colors.FromString(cp.Color))
		}
	case "physics.Sphere":
		sp := bod.(*physics.Sphere)
		sld.Pose.Scale.SetScalar(sp.Radius)
		if sp.Color != "" {
			sld.Mat.Color = errors.Log1(colors.FromString(sp.Color))
		}
	}
}

// ConfigView configures the view node to properly display world node
func (vw *View) ConfigView(wn physics.Node, vn xyz.Node, sc *xyz.Scene) {
	wb := wn.AsNodeBase()
	vb := vn.(*xyz.Group)
	vb.Pose.Pos = wb.Rel.Pos
	vb.Pose.Quat = wb.Rel.Quat
	bod := wn.AsBody()
	if bod == nil {
		return
	}
	if !vb.HasChildren() {
		sc.AddFromLibrary(bod.AsBodyBase().Vis, vb)
	}
	bgp := vb.Child(0)
	if bgp.HasChildren() {
		sld, has := bgp.Child(0).(*xyz.Solid)
		if has {
			vw.ConfigBodySolid(bod, sld)
		}
	}
}

// SyncNode updates the view tree to match the world tree, using
// efficient plan-based Build to maximally preserve existing tree elements
// returns true if view tree was modified (elements added / removed etc)
func (vw *View) SyncNode(wn physics.Node, vn xyz.Node, sc *xyz.Scene) bool {
	nm := wn.Name()
	vn.SetName(nm) // guaranteed to be unique
	skids := wn.AsTree().Children
	p := make(tree.TypePlan, 0, len(skids))
	for _, skid := range skids {
		p.Add(xyz.GroupType, skid.Name())
	}
	mod := tree.Update(vn, p)
	modall := mod
	for idx := range skids {
		wk := wn.Child(idx).(physics.Node)
		vk := vn.Child(idx).(xyz.Node)
		vw.ConfigView(wk, vk, sc)
		if wk.HasChildren() {
			kmod := vw.SyncNode(wk, vk, sc)
			if kmod {
				modall = true
			}
		}
	}
	if modall {
		sc.NeedsUpdate()
	}
	return modall
}

///////////////////////////////////////////////////////////////
// UpdatePose

// UpdatePoseNode updates the view pose values only from world tree.
// Essential that both trees are already synchronized.
func (vw *View) UpdatePoseNode(wn physics.Node, vn xyz.Node) {
	skids := wn.AsTree().Children
	for idx := range skids {
		wk := wn.Child(idx).(physics.Node)
		vk := vn.Child(idx).(xyz.Node)
		wb := wk.AsNodeBase()
		vb := vk.AsNode()
		vb.Pose.Pos = wb.Rel.Pos
		vb.Pose.Quat = wb.Rel.Quat
		vw.UpdatePoseNode(wk, vk)
	}
}

// UpdateBodyViewNode updates the body view info for given name(s)
// Essential that both trees are already synchronized.
func (vw *View) UpdateBodyViewNode(bodyNames []string, wn physics.Node, vn xyz.Node) {
	skids := wn.AsTree().Children
	for idx := range skids {
		wk := wn.Child(idx).(physics.Node)
		vk := vn.Child(idx).(xyz.Node)
		match := false
		if _, isBod := wk.(physics.Body); isBod {
			for _, nm := range bodyNames {
				if wk.Name() == nm {
					match = true
					break
				}
			}
		}
		if match {
			wb := wk.(physics.Body)
			bgp := vk.Child(0)
			if bgp.HasChildren() {
				sld, has := bgp.Child(0).(*xyz.Solid)
				if has {
					vw.ConfigBodySolid(wb, sld)
				}
			}
		}
		vw.UpdateBodyViewNode(bodyNames, wk, vk)
	}
}
