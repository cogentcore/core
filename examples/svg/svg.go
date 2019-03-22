// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/mitchellh/go-homedir"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

var CurFilename = ""
var TheSVG *svg.Editor
var TheZoom *gi.SpinBox
var TheTransX *gi.SpinBox
var TheTransY *gi.SpinBox
var TheFile *gi.TextField

func SetZoom(zf float32) {
	TheSVG.Scale = zf
	TheZoom.SetValue(zf)
	TheSVG.SetTransform()
}

func SetTrans(xt, yt float32) {
	TheSVG.Trans.Set(xt, yt)
	TheTransX.SetValue(xt)
	TheTransY.SetValue(yt)
	TheSVG.SetTransform()
}

func OpenSVG(fnm string) {
	CurFilename = fnm
	TheFile.SetText(CurFilename)
	updt := TheSVG.UpdateStart()
	TheSVG.SetFullReRender()
	fmt.Printf("Opening: %v\n", CurFilename)
	TheSVG.OpenXML(CurFilename)
	SetZoom(TheSVG.Viewport.Win.LogicalDPI() / 96.0)
	SetTrans(0, 0)
	TheSVG.UpdateEnd(updt)
}

func FileViewOpenSVG(vp *gi.Viewport2D) {
	giv.FileViewDialog(vp, CurFilename, ".svg", giv.DlgOpts{Title: "Open SVG"}, nil,
		vp.Win, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				dlg, _ := send.(*gi.Dialog)
				OpenSVG(giv.FileViewDialogValue(dlg))
			}
		})
}

func mainrun() {
	width := 1600
	height := 1200

	gi.SetAppName("svg")
	gi.SetAppAbout(`This is a demo of the SVG rendering (and start on editing) in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>
<p>You can drag the image around and use the scroll wheel to zoom.</p>`)

	win := gi.NewWindow2D("gogi-svg-viewer", "GoGi SVG Viewer", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := gi.AddNewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()

	svgrow := gi.AddNewLayout(mfr, "svgrow", gi.LayoutHoriz)
	svgrow.SetProp("horizontal-align", "center")
	svgrow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	svgrow.SetStretchMaxWidth()
	svgrow.SetStretchMaxHeight()

	svge := svg.AddNewEditor(svgrow, "svg")
	TheSVG = svge
	svge.InitScale()
	svge.Fill = true
	svge.SetProp("background-color", "white")
	svge.SetProp("width", units.NewValue(float32(width-20), units.Px))
	svge.SetProp("height", units.NewValue(float32(height-100), units.Px))
	svge.SetStretchMaxWidth()
	svge.SetStretchMaxHeight()

	loads := tbar.AddAction(gi.ActOpts{Label: "Open SVG", Icon: "file-open"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			FileViewOpenSVG(vp)
		})
	loads.StartFocus()

	fnm := gi.AddNewTextField(tbar, "cur-fname")
	TheFile = fnm
	fnm.SetMinPrefWidth(units.NewValue(60, units.Ch))

	zmlb := gi.AddNewLabel(tbar, "zmlb", "Zoom: ")
	zmlb.SetProp("vertical-align", gi.AlignMiddle)
	zmlb.Tooltip = "zoom scaling factor -- can use mouse scrollwheel to zoom as well"

	zoomout := tbar.AddAction(gi.ActOpts{Icon: "zoom-out", Name: "zoomout", Tooltip: "zoom out"},
		win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			SetZoom(svge.Scale * 0.9)
			win.FullReRender()
		})
	zoomout.SetProp("margin", 0)
	zoomout.SetProp("padding", 0)
	zoomout.SetProp("#icon", ki.Props{
		"width":  units.NewValue(1.5, units.Em),
		"height": units.NewValue(1.5, units.Em),
	})
	zoom := gi.AddNewSpinBox(tbar, "zoom")
	// zoom.SetMinPrefWidth(units.NewValue(10, units.Em))
	zoom.SetValue(svge.Scale)
	zoom.Tooltip = "zoom scaling factor -- can use mouse scrollwheel to zoom as well"
	TheZoom = zoom
	zoom.SpinBoxSig.Connect(win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sp := send.(*gi.SpinBox)
		SetZoom(sp.Value)
		win.FullReRender()
	})

	zoomin := tbar.AddAction(gi.ActOpts{Icon: "zoom-in", Name: "zoomin", Tooltip: " zoom in"},
		win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			SetZoom(svge.Scale * 1.1)
			win.FullReRender()
		})
	zoomin.SetProp("margin", 0)
	zoomin.SetProp("padding", 0)
	zoomin.SetProp("#icon", ki.Props{
		"width":  units.NewValue(1.5, units.Em),
		"height": units.NewValue(1.5, units.Em),
	})

	gi.AddNewSpace(tbar, "spctr")
	trlb := gi.AddNewLabel(tbar, "trlb", "Translate: ")
	trlb.Tooltip = "Translation of overall image -- can use mouse drag to move as well"
	trlb.SetProp("vertical-align", gi.AlignMiddle)

	trx := gi.AddNewSpinBox(tbar, "trx")
	// zoom.SetMinPrefWidth(units.NewValue(10, units.Em))
	trx.SetValue(svge.Trans.X)
	TheTransX = trx

	try := gi.AddNewSpinBox(tbar, "try")
	// zoom.SetMinPrefWidth(units.NewValue(10, units.Em))
	try.SetValue(svge.Trans.Y)
	TheTransY = try

	fnm.TextFieldSig.Connect(win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			tf := send.(*gi.TextField)
			fn, _ := homedir.Expand(tf.Text())
			OpenSVG(fn)
		}
	})

	svge.NodeSig.Connect(win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		ssvg := send.Embed(svg.KiT_Editor).(*svg.Editor)
		SetZoom(ssvg.Scale)
		SetTrans(ssvg.Trans.X, ssvg.Trans.Y)
	})

	vp.UpdateEndNoSig(updt)

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	// note: Command in shortcuts is automatically translated into Control for
	// Linux, Windows or Meta for MacOS
	fmen := win.MainMenu.ChildByName("File", 0).(*gi.Action)
	fmen.Menu = make(gi.Menu, 0, 10)
	fmen.Menu.AddAction(gi.ActOpts{Label: "Open", Shortcut: "Command+O"},
		win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			FileViewOpenSVG(vp)
		})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddAction(gi.ActOpts{Label: "Close Window", Shortcut: "Command+W"},
		win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			win.OSWin.Close()
		})

	win.SetCloseCleanFunc(func(w *gi.Window) {
		go gi.Quit() // once main window is closed, quit
	})

	// todo: when saving works, add option to save, and change above to CloseReq

	win.MainMenuUpdated()

	win.StartEventLoop()
}
