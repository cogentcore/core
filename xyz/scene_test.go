// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
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
	sc.Update()

	sc.AssertImage(t, "scene.png")
}
