// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package world2d

//go:generate core generate -add-types

import (
	"fmt"
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz/physics"
)

// View connects a Virtual World with a 2D SVG Scene to visualize the world
type View struct {

	// width of lines for shape rendering, in normalized units
	LineWidth float32

	// projection matrix for converting 3D to 2D -- resulting X, Y coordinates are used from Vector3
	Projection math32.Matrix4

	// the root Group node of the virtual world
	World *physics.Group

	// the SVG rendering canvas for visualizing in 2D
	Scene *svg.SVG

	// the root Group node in the Scene under which the world is rendered
	Root *svg.Group

	// library of shapes for bodies -- name matches Body.Vis
	Library map[string]*svg.Group
}

// NewView returns a new View that links given world with given scene and root group
func NewView(world *physics.Group, sc *svg.SVG, root *svg.Group) *View {
	vw := &View{World: world, Scene: sc, Root: root}
	vw.Library = make(map[string]*svg.Group)
	vw.ProjectXZ() // more typical
	vw.LineWidth = 0.05
	return vw
}

// InitLibrary initializes Scene library with basic shapes
// based on bodies in the virtual world.  More complex visualizations
// can be configured after this.
func (vw *View) InitLibrary() {
	vw.InitLibraryBody(vw.World)
}

// Sync synchronizes the view to the world
func (vw *View) Sync() bool {
	rval := vw.SyncNode(vw.World, vw.Root)
	return rval
}

// UpdatePose updates the view pose values only from world tree.
// Essential that both trees are already synchronized.
func (vw *View) UpdatePose() {
	vw.UpdatePoseNode(vw.World, vw.Root)
}

// UpdateBodyView updates the display properties of given body name
// recurses the tree until this body name is found.
func (vw *View) UpdateBodyView(bodyNames ...string) {
	vw.UpdateBodyViewNode(bodyNames, vw.World, vw.Root)
}

// Image returns the current rendered image
func (vw *View) Image() (*image.RGBA, error) {
	img := vw.Scene.Pixels
	if img == nil {
		return nil, fmt.Errorf("eve2d.View Image: is nil")
	}
	return img, nil
}

// ProjectXY sets 2D projection to reflect 3D X,Y coords
func (vw *View) ProjectXY() {
	vw.Projection.SetIdentity()
}

// ProjectXZ sets 2D projection to reflect 3D X,Z coords
func (vw *View) ProjectXZ() {
	vw.Projection.SetIdentity()
	vw.Projection[5] = 0 // Y->Y
	vw.Projection[9] = 1 // Z->Y
}

// todo: more projections

// Projection2D projects position from 3D to 2D
func (vw *View) Projection2D(pos math32.Vector3) math32.Vector2 {
	v2 := pos.MulMatrix4(&vw.Projection)
	return math32.Vec2(v2.X, v2.Y)
}

// Transform2D returns the full 2D transform matrix for a given position and quat rotation in 3D
func (vw *View) Transform2D(phys *physics.State) math32.Matrix2 {
	pos2 := phys.Pos.MulMatrix4(&vw.Projection)
	xyaxis := math32.Vec3(1, 1, 0)
	xyaxis.SetNormal()
	inv := vw.Projection.Transpose()
	axis := xyaxis.MulMatrix4(inv)
	axis.SetNormal()
	rot := axis.MulQuat(phys.Quat)
	rot.SetNormal()
	xyrot := rot.MulMatrix4(&vw.Projection)
	xyrot.Z = 0
	xyrot.SetNormal()
	ang := xyrot.AngleTo(xyaxis)
	xf2 := math32.Translate2D(pos2.X, pos2.Y).Rotate(ang)
	return xf2
}

///////////////////////////////////////////////////////////////
// Sync, Config

// NewInLibrary adds a new item of given name in library
func (vw *View) NewInLibrary(nm string) *svg.Group {
	if vw.Library == nil {
		vw.Library = make(map[string]*svg.Group)
	}
	gp := svg.NewGroup()
	gp.SetName(nm)
	vw.Library[nm] = gp
	return gp
}

// AddFromLibrary adds shape from library to given group
func (vw *View) AddFromLibrary(nm string, gp *svg.Group) {
	lgp, has := vw.Library[nm]
	if !has {
		return
	}
	gp.AddChild(lgp.Clone())
}

// InitLibraryBody initializes Scene library with basic shapes
// based on bodies in the virtual world.  More complex visualizations
// can be configured after this.
func (vw *View) InitLibraryBody(wn physics.Node) {
	bod := wn.AsBody()
	if bod != nil {
		vw.InitLibShape(bod)
	}
	for idx := range *wn.Children() {
		wk := wn.Child(idx).(physics.Node)
		vw.InitLibraryBody(wk)
	}
}

// InitLibShape initializes Scene library with basic shape for given body
func (vw *View) InitLibShape(bod physics.Body) {
	nm := bod.Name()
	bb := bod.AsBodyBase()
	if bb.Vis == "" {
		bb.Vis = nm
	}
	if _, has := vw.Library[nm]; has {
		return
	}
	lgp := vw.NewInLibrary(nm)
	wt := bod.NodeType().ShortName()
	switch wt {
	case "physics.Box":
		mnm := "eveBox"
		svg.NewRect(lgp).SetPos(math32.Vec2(0, 0)).SetSize(math32.Vec2(1, 1)).SetName(mnm)
	case "physics.Cylinder":
		mnm := "eveCylinder"
		svg.NewEllipse(lgp).SetPos(math32.Vec2(0, 0)).SetRadii(math32.Vec2(.1, .1)).SetName(mnm)
	case "physics.Capsule":
		mnm := "eveCapsule"
		svg.NewEllipse(lgp).SetPos(math32.Vec2(0, 0)).SetRadii(math32.Vec2(.1, .1)).SetName(mnm)
	case "physics.Sphere":
		mnm := "eveSphere"
		svg.NewCircle(lgp).SetPos(math32.Vec2(0, 0)).SetRadius(.1).SetName(mnm)
	}
}

// ConfigBodyShape configures a shape for a body with current values
func (vw *View) ConfigBodyShape(bod physics.Body, shp svg.Node) {
	wt := bod.NodeType().ShortName()
	sb := shp.AsNodeBase()
	sb.Nm = bod.Name()
	switch wt {
	case "physics.Box":
		bx := bod.(*physics.Box)
		sz := vw.Projection2D(bx.Size)
		shp.(*svg.Rect).SetSize(sz)
		sb.Paint.Transform = math32.Translate2D(-sz.X/2, -sz.Y/2)
		shp.SetProperty("transform", sb.Paint.Transform.String())
		shp.SetProperty("stroke-width", vw.LineWidth)
		shp.SetProperty("fill", "none")
		if bx.Color != "" {
			shp.SetProperty("stroke", bx.Color)
		}
	case "physics.Cylinder":
		cy := bod.(*physics.Cylinder)
		sz3 := math32.Vec3(cy.BotRad*2, cy.Height, cy.TopRad*2)
		sz := vw.Projection2D(sz3)
		shp.(*svg.Ellipse).SetRadii(sz)
		sb.Paint.Transform = math32.Translate2D(-sz.X/2, -sz.Y/2)
		shp.SetProperty("transform", sb.Paint.Transform.String())
		shp.SetProperty("stroke-width", vw.LineWidth)
		shp.SetProperty("fill", "none")
		if cy.Color != "" {
			shp.SetProperty("stroke", cy.Color)
		}
	case "physics.Capsule":
		cp := bod.(*physics.Capsule)
		sz3 := math32.Vec3(cp.BotRad*2, cp.Height, cp.TopRad*2)
		sz := vw.Projection2D(sz3)
		shp.(*svg.Ellipse).SetRadii(sz)
		sb.Paint.Transform = math32.Translate2D(-sz.X/2, -sz.Y/2)
		shp.SetProperty("transform", sb.Paint.Transform.String())
		shp.SetProperty("stroke-width", vw.LineWidth)
		shp.SetProperty("fill", "none")
		if cp.Color != "" {
			shp.SetProperty("stroke", cp.Color)
		}
	case "physics.Sphere":
		sp := bod.(*physics.Sphere)
		sz3 := math32.Vec3(sp.Radius*2, sp.Radius*2, sp.Radius*2)
		sz := vw.Projection2D(sz3)
		shp.(*svg.Circle).SetRadius(sz.X) // should be same as Y
		sb.Paint.Transform = math32.Translate2D(-sz.X/2, -sz.Y/2)
		shp.SetProperty("transform", sb.Paint.Transform.String())
		shp.SetProperty("stroke-width", vw.LineWidth)
		shp.SetProperty("fill", "none")
		if sp.Color != "" {
			shp.SetProperty("stroke", sp.Color)
		}
	}
}

// ConfigView configures the view node to properly display world node
func (vw *View) ConfigView(wn physics.Node, vn svg.Node) {
	wb := wn.AsNodeBase()
	vb := vn.(*svg.Group)
	vb.Paint.Transform = vw.Transform2D(&wb.Rel)
	vb.SetProperty("transform", vb.Paint.Transform.String())
	bod := wn.AsBody()
	if bod == nil {
		return
	}
	if !vb.HasChildren() {
		vw.AddFromLibrary(bod.AsBodyBase().Vis, vb)
	}
	bgp := vb.Child(0)
	if bgp.HasChildren() {
		shp, has := bgp.Child(0).(svg.Node)
		if has {
			vw.ConfigBodyShape(bod, shp)
		}
	}
	sz := vw.Scene.Geom.Size
	vw.Scene.Config(sz.X, sz.Y)
}

// SyncNode updates the view tree to match the world tree, using
// ConfigChildren to maximally preserve existing tree elements
// returns true if view tree was modified (elements added / removed etc)
func (vw *View) SyncNode(wn physics.Node, vn svg.Node) bool {
	nm := wn.Name()
	vn.SetName(nm) // guaranteed to be unique
	skids := *wn.Children()
	p := make(tree.TypePlan, 0, len(skids))
	for _, skid := range skids {
		p.Add(svg.GroupType, skid.Name())
	}
	mod := tree.Build(vn, p)
	modall := mod
	for idx := range skids {
		wk := wn.Child(idx).(physics.Node)
		vk := vn.Child(idx).(svg.Node)
		vw.ConfigView(wk, vk)
		if wk.HasChildren() {
			kmod := vw.SyncNode(wk, vk)
			if kmod {
				modall = true
			}
		}
	}
	return modall
}

///////////////////////////////////////////////////////////////
// UpdatePose

// UpdatePoseNode updates the view pose values only from world tree.
// Essential that both trees are already synchronized.
func (vw *View) UpdatePoseNode(wn physics.Node, vn svg.Node) {
	skids := *wn.Children()
	for idx := range skids {
		wk := wn.Child(idx).(physics.Node)
		vk := vn.Child(idx).(svg.Node).(*svg.Group)
		wb := wk.AsNodeBase()
		vk.Paint.Transform = vw.Transform2D(&wb.Rel)
		vk.SetProperty("transform", vk.Paint.Transform.String())
		// fmt.Printf("wk: %s  pos: %v  vk: %s\n", wk.Name(), ps, vk.Child(0).Name())
		vw.UpdatePoseNode(wk, vk)
	}
}

// UpdateBodyViewNode updates the body view info for given name(s)
// Essential that both trees are already synchronized.
func (vw *View) UpdateBodyViewNode(bodyNames []string, wn physics.Node, vn svg.Node) {
	skids := *wn.Children()
	for idx := range skids {
		wk := wn.Child(idx).(physics.Node)
		vk := vn.Child(idx).(svg.Node)
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
			bgp := vk.Child(0)
			if bgp.HasChildren() {
				shp, has := bgp.Child(0).(svg.Node)
				if has {
					vw.ConfigBodyShape(wk.AsBody(), shp)
				}
			}
		}
		vw.UpdateBodyViewNode(bodyNames, wk, vk)
	}
}
