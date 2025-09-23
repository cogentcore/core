// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package world implements visualization of [physics] using [xyz]
// 3D graphics.
package world

//go:generate core generate -add-types

import (
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/physics"
)

// World connects a Virtual World with an [xyz.Scene] to visualize the world,
// including ability to render offscreen.
type World struct {

	// World is the root Group node of the virtual world
	World *physics.Group

	// Scene is the [xyz.Scene] object for visualizing.
	Scene *xyz.Scene

	// Root is the root Group node in the Scene under which the world is rendered.
	Root *xyz.Group
}

// NewWorld returns a new World that links given [physics] world
// (top level Group) with given [xyz.Scene], making a
// top-level Root group in the scene.
func NewWorld(world *physics.Group, sc *xyz.Scene) *World {
	rgp := xyz.NewGroup(sc)
	rgp.SetName("world")
	wr := &World{World: world, Scene: sc, Root: rgp}
	GroupInit(wr.World, rgp)
	wr.Update()
	return wr
}

// Init initializes the physics world, e.g., at start of a new sim run.
func (wr *World) Init() {
	wr.World.WorldInit()
	wr.Update()
}

// Update updates the view from current physics node state.
func (wr *World) Update() {
	if wr.Scene != nil {
		wr.Scene.Update()
	}
}

// RenderFromNode does an offscreen render using given node
// for the camera position and orientation, returning the render image.
// Current scene camera is saved and restored.
func (wr *World) RenderFromNode(node physics.Node, cam *Camera) image.Image {
	sc := wr.Scene
	camnm := "physics-view-rendernode-save"
	sc.SaveCamera(camnm)
	defer func() {
		sc.SetCamera(camnm)
		sc.UseMainFrame()
	}()

	sc.Camera.FOV = cam.FOV
	sc.Camera.Near = cam.Near
	sc.Camera.Far = cam.Far
	nb := node.AsNodeBase()
	sc.Camera.Pose.Pos = nb.Abs.Pos
	sc.Camera.Pose.Quat = nb.Abs.Quat
	sc.Camera.Pose.Scale.Set(1, 1, 1)

	if sc.UseAltFrame(cam.Size) {
		return sc.RenderGrabImage()
	}
	return nil
}

// DepthImage returns the current rendered depth image
// func (vw *World) DepthImage() ([]float32, error) {
// 	return vw.Scene.DepthImage()
// }

// MakeStateToolbar returns a toolbar function for physics state updates,
// calling the given updt function after making the change.
func MakeStateToolbar(ps *physics.State, updt func()) func(p *tree.Plan) {
	return func(p *tree.Plan) {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(ps.SetEulerRotation).SetAfterFunc(updt).SetIcon(icons.Rotate90DegreesCcw)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(ps.SetAxisRotation).SetAfterFunc(updt).SetIcon(icons.Rotate90DegreesCcw)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(ps.RotateEuler).SetAfterFunc(updt).SetIcon(icons.Rotate90DegreesCcw)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(ps.RotateOnAxis).SetAfterFunc(updt).SetIcon(icons.Rotate90DegreesCcw)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(ps.EulerRotation).SetAfterFunc(updt).SetShowReturn(true).SetIcon(icons.Rotate90DegreesCcw)
		})
		tree.Add(p, func(w *core.Separator) {})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(ps.MoveOnAxis).SetAfterFunc(updt).SetIcon(icons.MoveItem)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(ps.MoveOnAxisAbs).SetAfterFunc(updt).SetIcon(icons.MoveItem)
		})

	}
}
