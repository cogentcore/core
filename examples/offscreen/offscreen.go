// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"goki.dev/colors"
	"goki.dev/gi3d"
	"goki.dev/gi3d/examples/assets"
	"goki.dev/grows/images"
	"goki.dev/mat32/v2"
	"goki.dev/vgpu/v2/vgpu"
)

func main() {
	gp, dev, err := vgpu.NoDisplayGPU("offscreen")
	if err != nil {
		log.Println(err)
		return
	}
	sc := gi3d.NewScene("scene").SetSize(image.Point{640, 480})
	sc.ConfigFrame(gp, dev)

	// options - must be set here
	// sc.MultiSample = 1
	sc.Wireframe = true
	// sc.NoNav = true

	// first, add lights, set camera
	sc.BackgroundColor = colors.FromRGB(230, 230, 255) // sky blue-ish
	gi3d.NewAmbientLight(sc, "ambient", 0.3, gi3d.DirectSun)

	sc.Camera.Pose.Pos.Set(0, 5, 10)              // default position
	sc.Camera.LookAt(mat32.Vec3Zero, mat32.Vec3Y) // defaults to looking at origin

	dir := gi3d.NewDirLight(sc, "dir", 1, gi3d.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := gi3d.NewPointLight(sc, "point", 1, gi3d.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := gi3d.NewSpotLight(sc, "spot", 1, gi3d.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	grtx := gi3d.NewTextureFileFS(assets.Content, sc, "ground", "ground.png")
	// _ = grtx
	// wdtx := gi3d.NewTextureFile(sc, "wood", "wood.png")

	cbm := gi3d.NewBox(sc, "cube1", 1, 1, 1)
	// cbm.Segs.Set(10, 10, 10) // not clear if any diff really..

	rbgp := gi3d.NewGroup(sc, "r-b-group")

	rcb := gi3d.NewSolid(rbgp, "red-cube").SetMesh(cbm)
	rcb.Pose.Pos.Set(-1, 0, 0)
	rcb.Mat.Color = colors.Red
	rcb.Mat.Shiny = 500

	bcb := gi3d.NewSolid(rbgp, "blue-cube").SetMesh(cbm)
	bcb.Pose.Pos.Set(1, 1, 0)
	bcb.Pose.Scale.X = 2 // somehow causing to not render
	bcb.Mat.Color = colors.Blue
	bcb.Mat.Shiny = 10
	// bcb.Mat.Reflective = 0.2

	gcb := gi3d.NewSolid(rbgp, "green-trans-cube").SetMesh(cbm)
	gcb.Pose.Pos.Set(0, 0, 1)
	gcb.Mat.Color = color.RGBA{0, 255, 0, 128} // alpha = .5 -- note: colors are NOT premultiplied here: will become so when rendered!

	floorp := gi3d.NewPlane(sc, "floor-plane", 100, 100)
	floor := gi3d.NewSolid(sc, "floor").SetMesh(floorp)
	floor.Pose.Pos.Set(0, -5, 0)
	floor.Mat.Color = colors.Tan
	// floor.Mat.Emissive.SetName("brown")
	floor.Mat.Bright = 2 // .5 for wood / brown
	floor.Mat.SetTexture(grtx)
	floor.Mat.Tiling.Repeat.Set(40, 40)
	// floor.SetDisabled() // not selectable

	sc.Config()
	sc.UpdateNodes()
	if !sc.Render() {
		log.Println("no render")
	}

	img, err := sc.Image()
	if err != nil {
		fmt.Println(err)
		return
	}
	images.Save(img, "render.png")
}
