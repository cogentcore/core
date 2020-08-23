// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1600
	height := 1200

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	gi.SetAppName("gi3dview")
	gi.SetAppAbout(`This is a viewer for the 3D graphics aspect of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://github.com/goki/gi/blob/master/examples/gi3dviewer/README.md">README</a> page for this example app has further info.</p>`)

	win := gi.NewMainWindow("gi3d-viewer", "GoGi 3D Viewer", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.SetProp("spacing", units.NewEx(1))

	tbar := gi.AddNewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()

	//////////////////////////////////////////
	//    Scene

	gi.AddNewSpace(mfr, "scspc")
	scvw := gi3d.AddNewSceneView(mfr, "sceneview")
	scvw.SetStretchMax()
	scvw.Config()
	sc := scvw.Scene()

	// first, add lights, set camera
	sc.BgColor.SetUInt8(230, 230, 255, 255) // sky blue-ish
	gi3d.AddNewAmbientLight(sc, "ambient", 0.3, gi3d.DirectSun)

	dir := gi3d.AddNewDirLight(sc, "dir", 1, gi3d.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := gi3d.AddNewPointLight(sc, "point", 1, gi3d.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := gi3d.AddNewSpotLight(sc, "spot", 1, gi3d.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	sc.Camera.LookAt(mat32.Vec3Zero, mat32.Vec3Y) // defaults to looking at origin

	objgp := gi3d.AddNewGroup(sc, sc, "obj-gp")

	curFn := ""
	exts := ".obj,.dae,.gltf"

	tbar.AddAction(gi.ActOpts{Label: "Open...", Icon: "file-open", Tooltip: "Open a 3D object file for viewing."}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		giv.FileViewDialog(vp, curFn, exts, giv.DlgOpts{Title: "Open 3D Object", Prompt: "Open a 3D object file for viewing."}, nil,
			win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					dlg, _ := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
					fn := giv.FileViewDialogValue(dlg)
					curFn = fn
					updt := sc.UpdateStart()
					objgp.DeleteChildren(true)
					sc.DeleteMeshes()
					sc.DeleteTextures()
					ki.DelMgr.DestroyDeleted() // this is actually essential to prevent leaking memory!
					_, err := sc.OpenNewObj(fn, objgp)
					if err != nil {
						log.Println(err)
					}
					sc.SetCamera("default")
					sc.UpdateEnd(updt)
				}
			})
	})

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)

	win.MainMenuUpdated()
	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}
