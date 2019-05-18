// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {

	tstslice := make([]string, 10)

	for i := 0; i < len(tstslice); i++ {
		tstslice[i] = fmt.Sprintf("this is element: %v", i)
	}

	tstmap := make(map[string]string)

	tstmap["mapkey1"] = "whatever"
	tstmap["mapkey2"] = "testing"
	tstmap["mapkey3"] = "boring"

	// turn this on to see a trace of the rendering
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true

	gi.SetAppName("views")
	gi.SetAppAbout(`This is a demo of the MapView and SliceView views in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	width := 1024
	height := 768
	win := gi.NewWindow2D("gogi-views-test", "GoGi Views Test", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	trow := gi.AddNewLayout(mfr, "trow", gi.LayoutHoriz)
	trow.SetProp("horizontal-align", "center")
	trow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	trow.SetStretchMaxWidth()

	spc := gi.AddNewSpace(mfr, "spc1")
	spc.SetFixedHeight(units.NewEm(2))

	gi.AddNewStretch(trow, "str1")
	but := gi.AddNewButton(trow, "slice-test")
	but.SetText("SliceTest")
	but.Tooltip = "open a window of a slice view with a lot of elments, for performance testing"
	but.ButtonSig.Connect(win, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			sl := make([]float32, 2880)
			gi.ProfileToggle()
			giv.SliceViewDialog(vp, &sl, giv.DlgOpts{Title: "SliceView Test", Prompt: "It should open quickly."}, nil, nil, nil)
		}
	})

	lab1 := gi.AddNewLabel(trow, "lab1", "<large>This is a test of the <tt>Slice</tt> and <tt>Map</tt> Views reflect-ive GUI</large>")
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	gi.AddNewStretch(trow, "str2")

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = gi.X

	mvfr := gi.AddNewFrame(split, "mvfr", gi.LayoutHoriz)
	svfr := gi.AddNewFrame(split, "svfr", gi.LayoutHoriz)
	split.SetSplits(.5, .5)

	mv := giv.AddNewMapView(mvfr, "mv")
	mv.SetMap(&tstmap, nil)
	mv.SetStretchMaxWidth()
	mv.SetStretchMaxHeight()

	sv := giv.AddNewSliceView(svfr, "sv")
	sv.SetSlice(&tstslice, nil)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	win.SetCloseCleanFunc(func(w *gi.Window) {
		go gi.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()
	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}
