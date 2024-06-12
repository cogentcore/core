// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image/color"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/examples/assets"
)

func main() {
	sc := xyz.NewOffscreenScene()

	// options - must be set here
	// sc.MultiSample = 1
	// sc.Wireframe = true
	// sc.NoNav = true

	// first, add lights, set camera
	sc.BackgroundColor = colors.FromRGB(230, 230, 255) // sky blue-ish
	xyz.NewAmbientLight(sc, "ambient", 0.3, xyz.DirectSun)

	// sc.Camera.Pose.Pos.Set(-2, 9, 3)
	sc.Camera.Pose.Pos.Set(-2, 2, 10)
	sc.Camera.LookAt(math32.Vector3{}, math32.Vec3(0, 1, 0)) // defaults to looking at origin

	dir := xyz.NewDirLight(sc, "dir", 1, xyz.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := xyz.NewPointLight(sc, "point", 1, xyz.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := xyz.NewSpotLight(sc, "spot", 1, xyz.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	grtx := xyz.NewTextureFileFS(assets.Content, sc, "ground", "ground.png")
	// _ = grtx
	// wdtx := xyz.NewTextureFile(sc, "wood", "wood.png")

	cbm := xyz.NewBox(sc, "cube1", 1, 1, 1)
	cbm.Segs.Set(10, 10, 10) // not clear if any diff really..

	rbgp := xyz.NewGroup(sc)

	rcb := xyz.NewSolid(rbgp).SetMesh(cbm)
	rcb.Pose.Pos.Set(-1, 0, 0)
	rcb.Material.SetColor(colors.Red).SetShiny(500)

	bcb := xyz.NewSolid(rbgp).SetMesh(cbm)
	bcb.Pose.Pos.Set(1, 1, 0)
	bcb.Pose.Scale.X = 2 // somehow causing to not render
	bcb.Material.SetColor(colors.Blue).SetShiny(10).SetReflective(0.2)

	gcb := xyz.NewSolid(rbgp).SetMesh(cbm)
	gcb.Pose.Pos.Set(0, 0, 1)
	gcb.Material.SetColor(color.RGBA{0, 255, 0, 128}).SetShiny(20) // alpha = .5 -- note: colors are NOT premultiplied here: will become so when rendered!

	floorp := xyz.NewPlane(sc, "floor", 100, 100)
	floor := xyz.NewSolid(sc).SetMesh(floorp)
	floor.Pose.Pos.Set(0, -5, 0)
	floor.Material.Color = colors.Tan
	// floor.Mat.Emissive.SetName("brown")
	// floor.Mat.Bright = 2 // .5 for wood / brown
	floor.Material.SetTexture(grtx)
	floor.Material.Tiling.Repeat.Set(40, 40)
	// floor.SetDisabled() // not selectable

	img := errors.Must1(sc.ImageUpdate())
	imagex.Save(img, "render.png")
	sc.ImageDone()
}
