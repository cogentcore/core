// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz_test

import (
	"image/color"
	"testing"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	_ "cogentcore.org/core/paint/renderers"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	. "cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/examples/assets"
	_ "cogentcore.org/core/xyz/io/obj"
)

func TestScene(t *testing.T) {
	sc := NewOffscreenScene()

	NewAmbient(sc, "ambient", 0.3, DirectSun)
	NewDirectional(sc, "directional", 1, DirectSun).Pos.Set(0, 2, 1)
	sc.Camera.Pose.Pos.Set(0, 2, 10)
	sc.Camera.LookAt(math32.Vector3{}, math32.Vec3(0, 1, 0)) // defaults to looking at origin

	cube := NewBox(sc, "cube1", 1, 1, 1)
	cube.Segs.Set(10, 10, 10) // not clear if any diff really..

	tree.AddChild(sc, func(g *Group) {
		tree.AddChild(g, func(n *Solid) {
			n.SetMesh(cube).SetColor(colors.Red).SetShiny(500).SetPos(-1, 0, 0)
		})
		tree.AddChild(g, func(n *Solid) {
			n.SetMesh(cube).SetColor(colors.Blue).SetShiny(10).SetReflective(0.2).
				SetPos(1, 1, 0)
			n.Pose.Scale.X = 2
		})
		tree.AddChild(g, func(n *Solid) {
			n.SetMesh(cube).SetColor(color.RGBA{0, 255, 0, 128}).SetShiny(20).SetPos(0, 0, 1)
		})
	})

	grtx := NewTextureFileFS(assets.Content, sc, "ground", "ground.png")
	floorp := NewPlane(sc, "floor-plane", 100, 100)
	tree.AddChild(sc, func(n *Solid) {
		n.SetMesh(floorp).SetColor(colors.Tan).SetTexture(grtx).SetPos(0, -5, 0)
		n.Material.Tiling.Repeat.Set(40, 40)
	})

	lines := NewLines(sc, "Lines", []math32.Vector3{{-3, -1, 0}, {-2, 1, 0}, {2, 1, 0}, {3, -1, 0}}, math32.Vec2(.2, .1), CloseLines)
	tree.AddChild(sc, func(n *Solid) {
		n.SetMesh(lines).SetColor(color.RGBA{255, 255, 0, 128}).SetPos(0, 0, 1)
	})

	// this line should go from lower left front of red cube to upper vertex of above hi-line
	tree.AddChild(sc, func(g *xyz.Group) {
		InitArrow(g, math32.Vec3(-1.5, -.5, .5), math32.Vec3(2, 1, 1), .05, colors.Cyan, StartArrow, EndArrow, 4, .5, 8)
	})

	cylinder := NewCylinder(sc, "cylinder", 1.5, .5, 32, 1, true, true)
	tree.AddChild(sc, func(n *Solid) {
		n.SetMesh(cylinder).SetPos(-2.25, 0, 0)
	})

	capsule := NewCapsule(sc, "capsule", 1.5, .5, 32, 1)
	tree.AddChild(sc, func(n *Solid) {
		n.SetMesh(capsule).SetColor(colors.Tan).SetPos(3.25, 0, 0)
	})

	sphere := NewSphere(sc, "sphere", .75, 32)
	tree.AddChild(sc, func(n *Solid) {
		n.SetMesh(sphere).SetColor(colors.Orange).SetPos(0, -2, 0)
		n.Material.Color.A = 200
	})

	// Good strategy for objects if used in multiple places is to load
	// into library, then add from there.
	lgo := errors.Log1(sc.OpenToLibraryFS(assets.Content, "gopher.obj", ""))
	lgo.Pose.SetAxisRotation(0, 1, 0, -90) // for all cases

	tree.AddChild(sc, func(g *Group) {
		g.Maker(func(p *tree.Plan) {
			tree.Add(p, func(n *Object) {
				n.SetObjectName("gopher").SetScale(.5, .5, .5).SetPos(1.4, -2.5, 0)
				n.SetAxisRotation(0, 1, 0, -60)
			})
			tree.Add(p, func(n *Object) {
				n.SetObjectName("gopher").SetPos(-1.5, -2, 0).SetScale(.2, .2, .2)
			})
		})
	})

	torus := NewTorus(sc, "torus", .75, .1, 32)
	tree.AddChild(sc, func(n *Solid) {
		n.SetMesh(torus).SetColor(colors.White).SetPos(-1.6, -1.6, -.2).SetAxisRotation(1, 0, 0, 90)
		n.Material.Color.A = 200
	})

	tree.AddChild(sc, func(n *Text2D) {
		n.SetText("Text2D can put <b>HTML</b> formatted Text anywhere you might <i>want</i>").SetPos(0, 2.2, 0)
		n.Styles.Text.Align = text.Center
		n.Pose.Scale.SetScalar(0.2)
	})

	// automatically tracks camera -- FPS effect
	tree.AddChildAt(sc, TrackCameraName, func(g *Group) {
		tree.AddChild(g, func(n *Solid) {
			// in front of camera
			n.SetMesh(cube).SetScale(.1, .1, 1).SetPos(.5, -.5, -2.5).
				SetColor(color.RGBA{255, 0, 255, 128})
		})
	})

	sc.AssertImage(t, "scene.png")
}
