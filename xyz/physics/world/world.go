// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package world

//go:generate core generate -add-types

import (
	"image"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/physics"
)

// View connects a Virtual World with an [xyz.Scene] to visualize the world,
// including ability to render offscreen.
type View struct {

	// World is the root Group node of the virtual world
	World *physics.Group

	// Scene is the [xyz.Scene] object for visualizing.
	Scene *xyz.Scene

	// Root is the root Group node in the Scene under which the world is rendered.
	Root *xyz.Group
}

// NewView returns a new View that links given world with given scene and root group
func NewView(world *physics.Group, sc *xyz.Scene, root *xyz.Group) *View {
	vw := &View{World: world, Scene: sc, Root: root}
	vw.Init()
	return vw
}

// Init performs initialization at New.
func (vw *View) Init() {
	GroupInit(vw.World, vw.Root)
}

// Update updates the view from current physics node state.
func (vw *View) Update() {
	vw.Scene.Update()
}

// RenderFromNode does an offscreen render using given node
// for the camera position and orientation, returning the render image.
// Current scene camera is saved and restored.
func (vw *View) RenderFromNode(node physics.Node, cam *Camera) image.Image {
	sc := vw.Scene
	camnm := "physics-view-rendernode-save"
	sc.SaveCamera(camnm)
	defer sc.SetCamera(camnm)

	sc.Camera.FOV = cam.FOV
	sc.Camera.Near = cam.Near
	sc.Camera.Far = cam.Far
	nb := node.AsNodeBase()
	sc.Camera.Pose.Pos = nb.Abs.Pos
	sc.Camera.Pose.Quat = nb.Abs.Quat
	sc.Camera.Pose.Scale.Set(1, 1, 1)

	img := sc.RenderGrabImage()
	if img != nil {
		return imagex.Resize(img, cam.Size)
	}
	return nil
}

// DepthImage returns the current rendered depth image
// func (vw *View) DepthImage() ([]float32, error) {
// 	return vw.Scene.DepthImage()
// }
