// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/units"
	"goki.dev/icons"

	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

var (
	Ticker *time.Ticker
	Frame  *gi.Frame
)

func main() {
	gimain.Main(mainrun)
}

type TableStruct struct {

	// an icon
	Icon icons.Icon `desc:"an icon"`

	// an integer field
	IntField int `desc:"an integer field"`

	// a float field
	FloatField float32 `desc:"a float field"`

	// a string field
	StrField string `desc:"a string field"`

	// a file
	File gi.FileName `desc:"a file"`
}

type ILStruct struct {

	// click to show next
	On bool `desc:"click to show next"`

	// [viewif: On] can u see me?
	ShowMe string `viewif:"On" desc:"can u see me?"`

	// [viewif: On] a conditional
	Cond int `viewif:"On" desc:"a conditional"`

	// [viewif: On&&Cond==0] On and Cond=0 -- note that slbool as bool cannot be used directly..
	Cond1 string `viewif:"On&&Cond==0" desc:"On and Cond=0 -- note that slbool as bool cannot be used directly.."`

	// [viewif: On&&Cond<=1] if Cond=0
	Cond2 TableStruct `viewif:"On&&Cond<=1" desc:"if Cond=0"`

	// a value
	Val float32 `desc:"a value"`
}

type Struct struct {

	// an enum
	Stripes gi.Stripes `desc:"an enum"`

	// [viewif: !(Stripes==[RowStripes,ColStripes])] a string
	Name string `viewif:"!(Stripes==[RowStripes,ColStripes])" desc:"a string"`

	// click to show next
	ShowNext bool `desc:"click to show next"`

	// [viewif: ShowNext] can u see me?
	ShowMe string `viewif:"ShowNext" desc:"can u see me?"`

	// [view: inline] how about that
	Inline ILStruct `view:"inline" desc:"how about that"`

	// a conditional
	Cond int `desc:"a conditional"`

	// [viewif: Cond==0] if Cond=0
	Cond1 string `viewif:"Cond==0" desc:"if Cond=0"`

	// [viewif: Cond>=0] if Cond=0
	Cond2 TableStruct `viewif:"Cond>=0" desc:"if Cond=0"`

	// a value
	Val float32 `desc:"a value"`
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

	var stru Struct

	// turn this on to see a trace of the rendering
	// gi.WinEventTrace = true
	// gi.RenderTrace = true
	// gi.LayoutTrace = true

	gi.SetAppName("views")
	gi.SetAppAbout(`This is a demo of the MapView and SliceView views in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	width := 1024
	height := 768
	win := gi.NewMainWindow("gogi-views-test", "GoGi Views Test", width, height)

	vp := win.WinScene()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	Frame = mfr

	trow := gi.NewLayout(mfr, "trow", gi.LayoutHoriz)
	trow.SetProp("horizontal-align", "center")
	trow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	trow.SetStretchMaxWidth()

	spc := gi.NewSpace(mfr, "spc1")
	spc.SetFixedHeight(units.Em(2))

	gi.NewStretch(trow, "str1")
	but := gi.NewButton(trow, "slice-test")
	but.SetText("SliceDialog")
	but.Tooltip = "open a SliceViewDialog slice view with a lot of elments, for performance testing"
	but.ButtonSig.Connect(win, func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonClicked) {
			sl := make([]float32, 2880)
			gi.ProfileToggle()
			gi.WindowOpenTimer = time.Now()
			giv.SliceViewDialog(vp, &sl, giv.DlgOpts{Title: "SliceView Test", Prompt: "It should open quickly."}, nil, nil, nil)
		}
	})
	but = gi.NewButton(trow, "table-test")
	but.SetText("TableDialog")
	but.Tooltip = "open a TableViewDialog view "
	but.ButtonSig.Connect(win, func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonClicked) {
			giv.TableViewDialog(vp, &tsttable, giv.DlgOpts{Title: "TableView Test", Prompt: "how does it resize."}, nil, nil, nil)
		}
	})

	lab1 := gi.NewLabel(trow, "lab1", "<large>This is a test of the <tt>Slice</tt> and <tt>Map</tt> Views reflect-ive GUI</large>")
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	gi.NewStretch(trow, "str2")

	split := gi.NewSplitView(mfr, "split")
	split.Dim = mat32.X

	strv := giv.NewStructView(split, "strv")
	strv.SetStruct(&stru)
	strv.SetStretchMax()

	mv := giv.NewMapView(split, "mv")
	mv.SetMap(&tstmap)
	mv.SetStretchMaxWidth()
	mv.SetStretchMaxHeight()

	sv := giv.NewSliceView(split, "sv")
	// sv.SetInactive()
	sv.SetSlice(&tstslice)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	tv := giv.NewTableView(split, "tv")
	// sv.SetInactive()
	tv.SetSlice(&tsttable)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	split.SetSplits(.3, .2, .2, .3)

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

	// Ticker = time.NewTicker(1 * time.Second)
	// go Animate()

	win.StartEventLoop()
}

// Animate
func Animate() {
	for {
		<-Ticker.C // wait for tick

		updt := Frame.UpdateStart()
		// fmt.Printf("updt\n")
		Frame.UpdateEnd(updt)
	}

}
