// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	_ "cogentcore.org/core/iox/imagex"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/views"
	"cogentcore.org/core/xyz"
	_ "cogentcore.org/core/xyz/io/obj"
	"cogentcore.org/core/xyzview"
)

func main() {
	b := core.NewBody("XYZ Object Viewer")

	sv := xyzview.NewSceneView(b)
	sv.Config()
	sc := sv.SceneXYZ()

	// first, add lights, set camera
	sc.BackgroundColor = colors.FromRGB(230, 230, 255) // sky blue-ish
	xyz.NewAmbientLight(sc, "ambient", 0.3, xyz.DirectSun)

	dir := xyz.NewDirLight(sc, "dir", 1, xyz.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := xyz.NewPointLight(sc, "point", 1, xyz.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := xyz.NewSpotLight(sc, "spot", 1, xyz.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	sc.Camera.LookAt(mat32.Vec3{}, mat32.V3(0, 1, 0)) // defaults to looking at origin

	objgp := xyz.NewGroup(sc, "obj-gp")

	curFn := "objs/airplane_prop_001.obj"
	// curFn := "objs/piano_005.obj"
	exts := ".obj,.dae,.gltf"

	errors.Log1(sc.OpenNewObj(curFn, objgp))

	b.AddAppBar(func(tb *core.Toolbar) {
		core.NewButton(tb).SetText("Open").SetIcon(icons.Open).
			SetTooltip("Open a 3D object file for viewing").
			OnClick(func(e events.Event) {
				views.FileViewDialog(tb, curFn, exts, "Open 3D Object", func(selFile string) {
					curFn = selFile
					objgp.DeleteChildren()
					sc.DeleteMeshes()
					sc.DeleteTextures()
					errors.Log1(sc.OpenNewObj(selFile, objgp))
					sc.SetCamera("default")
					sc.NeedsUpdate()
					sv.NeedsRender()
				})
			})
	})

	sc.NeedsConfig()
	b.RunMainWindow()
}
