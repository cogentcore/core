// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/colors"
	"goki.dev/events"
	"goki.dev/gi"
	"goki.dev/giv"
	_ "goki.dev/grows/images"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ki"
	"goki.dev/mat32"
	"goki.dev/xyz"
	_ "goki.dev/xyz/io/obj"
	"goki.dev/xyzv"
)

func main() {
	b := gi.NewAppBody("XYZ Object Viewer")
	b.App().About = `This is a viewer for the 3D graphics aspect of the <b>GoGi</b> graphical interface system, within the <b>Goki</b> tree framework.  See <a href="https://github.com/goki">Goki on GitHub</a>. <p>The <a href="https://goki.dev/gi/v2/blob/master/examples/xyzviewer/README.md">README</a> page for this example app has further info.</p>`

	sv := xyzv.NewSceneView(b)
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

	grr.Log1(sc.OpenNewObj(curFn, objgp))

	b.AddAppBar(func(tb *gi.Toolbar) {
		gi.NewButton(tb).SetText("Open").SetIcon(icons.Open).
			SetTooltip("Open a 3D object file for viewing").
			OnClick(func(e events.Event) {
				giv.FileViewDialog(tb, curFn, exts, "Open 3D Object", func(selFile string) {
					curFn = selFile
					updt := sc.UpdateStart()
					objgp.DeleteChildren(true)
					sc.DeleteMeshes()
					sc.DeleteTextures()
					ki.DelMgr.DestroyDeleted() // this is actually essential to prevent leaking memory!
					grr.Log1(sc.OpenNewObj(selFile, objgp))
					sc.SetCamera("default")
					sc.UpdateEndConfig(updt)
					sv.SetNeedsRender(true)
				})
			})
	})

	sc.SetNeedsConfig()
	b.NewWindow().Run().Wait()
}
