// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	// turn these on to see a traces of various stages of processing..
	// ki.SignalTrace = true
	// gi.WinEventTrace = true
	// gi3d.Update3DTrace = true
	// gi.Update2DTrace = true

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	gi.SetAppName("gi3d")
	gi.SetAppAbout(`This is a demo of the 3D graphics aspect of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://github.com/goki/gi/blob/master/examples/gi3d/README.md">README</a> page for this example app has further info.</p>`)

	win := gi.NewWindow2D("gogi-widgets-demo", "GoGi Widgets Demo\x00", width, height, true) // true = pixel sizes

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.SetProp("spacing", units.NewValue(1, units.Ex))

	trow := gi.AddNewLayout(mfr, "trow", gi.LayoutHoriz)
	trow.SetStretchMaxWidth()

	title := gi.AddNewLabel(trow, "title", `This is a demonstration of the
<a href="https://github.com/goki/gi/gi">GoGi</a> <i>3D</i> Framework<br>
See <a href="https://github.com/goki/gi/blob/master/examples/gi3d/README.md">README</a> for detailed info and things to try.`)
	title.SetProp("white-space", gi.WhiteSpaceNormal) // wrap
	title.SetProp("text-align", gi.AlignCenter)       // note: this also sets horizontal-align, which controls the "box" that the text is rendered in..
	title.SetProp("vertical-align", gi.AlignCenter)
	title.SetProp("font-size", "x-large")
	title.SetProp("line-height", 1.5)
	title.SetStretchMaxWidth()
	title.SetStretchMaxHeight()

	//////////////////////////////////////////
	//    Scene

	gi.AddNewSpace(mfr, "scspc")
	scrow := gi.AddNewLayout(mfr, "scrow", gi.LayoutHoriz)
	scrow.SetStretchMaxWidth()
	scrow.SetStretchMaxHeight()

	sc := gi3d.AddNewScene(scrow, "scene")
	sc.SetStretchMaxWidth()
	sc.SetStretchMaxHeight()

	// first, add lights, set camera
	sc.BgColor.SetUInt8(230, 230, 255, 255) // sky blue-ish
	gi3d.AddNewAmbientLight(sc, "ambient", 0.3, gi3d.DirectSun)

	dir := gi3d.AddNewDirLight(sc, "dir", 1, gi3d.DirectSun)
	dir.Pos.Set(0, 2, 0) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := gi3d.AddNewPointLight(sc, "point", 1, gi3d.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := gi3d.AddNewSpotLight(sc, "spot", 1, gi3d.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	cbm := gi3d.AddNewBox(sc, "cube1", 1, 1, 1)
	// cbm.Segs.Set(10, 10, 10) // not clear if any diff really..

	rcb := sc.AddNewObject("red-cube", cbm.Name())
	rcb.Pose.Pos.Set(-1, 0, 0)
	rcb.Mat.Color.SetString("red", nil)
	rcb.Mat.Shiny = 500

	bcb := sc.AddNewObject("blue-cube", cbm.Name())
	bcb.Pose.Pos.Set(1, 1, 0)
	bcb.Pose.Scale.X = 2
	bcb.Mat.Color.SetString("blue", nil)
	bcb.Mat.Shiny = 10

	// bcb.Mat.Specular.SetString("blue", nil) // how you get rid of specular highlights

	gcb := sc.AddNewObject("green-trans-cube", cbm.Name())
	gcb.Pose.Pos.Set(0, 0, 1)
	gcb.Mat.Color.SetUInt8(0, 255, 0, 128) // alpha = .5

	tcg := sc.AddNewGroup("TrackCamera") // automatically tracks camera -- FPS effect
	fpgun := tcg.AddNewObject("first-person-gun", cbm.Name())
	fpgun.Pose.Scale.Set(.1, .1, 1)
	fpgun.Pose.Pos.Set(.5, -.5, -2.5)          // in front of camera
	fpgun.Mat.Color.SetUInt8(255, 0, 255, 128) // alpha = .5

	grtx := gi3d.AddNewTextureFile(sc, "ground", "ground.png")

	floorp := gi3d.AddNewPlane(sc, "floor-plane", 100, 100)
	floor := sc.AddNewObject("floor", floorp.Name())
	floor.Pose.Pos.Set(0, -5, 0)
	// floor.Mat.Color.SetString("tan", nil)
	floor.Mat.Emissive.SetString("tan", nil)
	floor.Mat.SetTexture(sc, grtx.Name())

	sc.Camera.LookAt(mat32.Vec3Zero, mat32.Vec3Y)

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)

	win.MainMenuUpdated()
	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}
