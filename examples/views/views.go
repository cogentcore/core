// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"

	"github.com/goki/gi/gi"
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

type TableStruct struct {
	Icon       gi.IconName `desc:"an icon"`
	IntField   int         `desc:"an integer field"`
	FloatField float32     `desc:"a float field"`
	StrField   string      `desc:"a string field"`
	File       gi.FileName `desc:"a file"`
}

func mainrun() {

	tstslice := make([]string, 40)

	for i := 0; i < len(tstslice); i++ {
		tstslice[i] = fmt.Sprintf("el: %v", i)
	}

	tstmap := make(map[string]string)

	tstmap["mapkey1"] = "whatever"
	tstmap["mapkey2"] = "testing"
	tstmap["mapkey3"] = "boring"

	tsttable := make([]*TableStruct, 100)

	for i := range tsttable {
		ts := &TableStruct{Icon: "go", IntField: i, FloatField: float32(i) / 10.0}
		tsttable[i] = ts
	}

	// turn this on to see a trace of the rendering
	// gi.WinEventTrace = true
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true

	gi.SetAppName("views")
	gi.SetAppAbout(`This is a demo of the MapView and SliceView views in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	width := 1024
	height := 768
	win := gi.NewMainWindow("gogi-views-test", "GoGi Views Test", width, height)

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
	but.SetText("SliceDialog")
	but.Tooltip = "open a SliceViewDialog slice view with a lot of elments, for performance testing"
	but.ButtonSig.Connect(win, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			sl := make([]float32, 2880)
			gi.ProfileToggle()
			gi.WindowOpenTimer = time.Now()
			giv.SliceViewDialog(vp, &sl, giv.DlgOpts{Title: "SliceView Test", Prompt: "It should open quickly."}, nil, nil, nil)
		}
	})
	but = gi.AddNewButton(trow, "table-test")
	but.SetText("TableDialog")
	but.Tooltip = "open a TableViewDialog view "
	but.ButtonSig.Connect(win, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			giv.TableViewDialog(vp, &tsttable, giv.DlgOpts{Title: "TableView Test", Prompt: "how does it resize."}, nil, nil, nil)
		}
	})

	lab1 := gi.AddNewLabel(trow, "lab1", "<large>This is a test of the <tt>Slice</tt> and <tt>Map</tt> Views reflect-ive GUI</large>")
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	gi.AddNewStretch(trow, "str2")

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = mat32.X

	mv := giv.AddNewMapView(split, "mv")
	mv.SetMap(&tstmap)
	mv.SetStretchMaxWidth()
	mv.SetStretchMaxHeight()

	sv := giv.AddNewSliceView(split, "sv")
	// sv.SetInactive()
	sv.SetSlice(&tstslice)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	tv := giv.AddNewTableView(split, "tv")
	// sv.SetInactive()
	tv.SetSlice(&tsttable)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	split.SetSplits(.2, .2, .6)

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
