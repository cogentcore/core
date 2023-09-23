// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/mitchellh/go-homedir"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/svg"
)

func main() {
	gimain.Main(mainrun)
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
	TheSVG.OpenXML(gi.FileName(CurFilename))
	SetZoom(TheSVG.ParentOSWin().LogicalDPI() / 96.0)
	SetTrans(0, 0)
	TheSVG.UpdateEnd(updt)
}

func FileViewOpenSVG(vp *gi.Scene) {
	giv.FileViewDialog(vp, CurFilename, ".svg", giv.DlgOpts{Title: "Open SVG"}, nil,
		vp.Win, func(recv, send ki.Ki, sig int64, data any) {
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

	win := gi.NewMainOSWin("gogi-svg-viewer", "GoGi SVG Viewer", width, height)

	vp := win.WinScene()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := gi.NewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()

	svgrow := gi.NewLayout(mfr, "svgrow", gi.LayoutHoriz)
	svgrow.SetProp("horizontal-align", "center")
	svgrow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	svgrow.SetStretchMaxWidth()
	svgrow.SetStretchMaxHeight()

	svge := svg.NewEditor(svgrow, "svg")
	TheSVG = svge
	svge.InitScale()
	svge.Fill = true
	svge.SetProp("background-color", "white")
	svge.SetProp("width", units.Px(float32(width-20)))
	svge.SetProp("height", units.Px(float32(height-100)))
	svge.SetStretchMaxWidth()
	svge.SetStretchMaxHeight()

	loads := tbar.AddAction(gi.ActOpts{Label: "Open SVG", Icon: icons.FileOpen}, win.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			FileViewOpenSVG(vp)
		})
	loads.StartFocus()

	fnm := gi.NewTextField(tbar, "cur-fname")
	TheFile = fnm
	fnm.SetMinPrefWidth(units.Ch(60))

	zmlb := gi.NewLabel(tbar, "zmlb", "Zoom: ")
	zmlb.SetProp("vertical-align", gist.AlignMiddle)
	zmlb.Tooltip = "zoom scaling factor -- can use mouse scrollwheel to zoom as well"

	zoomout := tbar.AddAction(gi.ActOpts{Icon: icons.ZoomOut, Name: "zoomout", Tooltip: "zoom out"},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			SetZoom(svge.Scale * 0.9)
			win.FullReRender()
		})
	zoomout.SetProp("margin", 0)
	zoomout.SetProp("padding", 0)
	zoomout.SetProp("#icon", ki.Props{
		"width":  units.Em(1.5),
		"height": units.Em(1.5),
	})
	zoom := gi.NewSpinBox(tbar, "zoom")
	// zoom.SetMinPrefWidth(units.NewEm(10))
	zoom.SetValue(svge.Scale)
	zoom.Tooltip = "zoom scaling factor -- can use mouse scrollwheel to zoom as well"
	TheZoom = zoom
	zoom.SpinBoxSig.Connect(win.This(), func(recv, send ki.Ki, sig int64, data any) {
		sp := send.(*gi.SpinBox)
		SetZoom(sp.Value)
		win.FullReRender()
	})

	zoomin := tbar.AddAction(gi.ActOpts{Icon: icons.ZoomIn, Name: "zoomin", Tooltip: " zoom in"},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			SetZoom(svge.Scale * 1.1)
			win.FullReRender()
		})
	zoomin.SetProp("margin", 0)
	zoomin.SetProp("padding", 0)
	zoomin.SetProp("#icon", ki.Props{
		"width":  units.Em(1.5),
		"height": units.Em(1.5),
	})

	gi.NewSpace(tbar, "spctr")
	trlb := gi.NewLabel(tbar, "trlb", "Translate: ")
	trlb.Tooltip = "Translation of overall image -- can use mouse drag to move as well"
	trlb.SetProp("vertical-align", gist.AlignMiddle)

	trx := gi.NewSpinBox(tbar, "trx")
	// zoom.SetMinPrefWidth(units.NewEm(10))
	trx.SetValue(svge.Trans.X)
	TheTransX = trx
	trx.SpinBoxSig.Connect(win.This(), func(recv, send ki.Ki, sig int64, data any) {
		SetTrans(trx.Value, svge.Trans.Y)
		win.FullReRender()
	})

	try := gi.NewSpinBox(tbar, "try")
	// zoom.SetMinPrefWidth(units.NewEm(10))
	try.SetValue(svge.Trans.Y)
	TheTransY = try
	try.SpinBoxSig.Connect(win.This(), func(recv, send ki.Ki, sig int64, data any) {
		SetTrans(svge.Trans.X, try.Value)
		win.FullReRender()
	})

	fnm.TextFieldSig.Connect(win.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.TextFieldDone) {
			tf := send.(*gi.TextField)
			fn, _ := homedir.Expand(tf.Text())
			OpenSVG(fn)
		}
	})

	svge.NodeSig.Connect(win.This(), func(recv, send ki.Ki, sig int64, data any) {
		ssvg := send.Embed(svg.TypeEditor).(*svg.Editor)
		SetZoom(ssvg.Scale)
		SetTrans(ssvg.Trans.X, ssvg.Trans.Y)
	})

	vp.UpdateEndNoSig(updt)

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "OSWin"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	// note: Command in shortcuts is automatically translated into Control for
	// Linux, OSWins or Meta for MacOS
	fmen := win.MainMenu.ChildByName("File", 0).(*gi.Action)
	fmen.Menu = make(gi.Menu, 0, 10)
	fmen.Menu.AddAction(gi.ActOpts{Label: "Open", Shortcut: "Command+O"},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			FileViewOpenSVG(vp)
		})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddAction(gi.ActOpts{Label: "Close OSWin", Shortcut: "Command+W"},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			win.OSWin.Close()
		})

	win.SetCloseCleanFunc(func(w *gi.OSWin) {
		go gi.Quit() // once main window is closed, quit
	})

	// todo: when saving works, add option to save, and change above to CloseReq

	win.MainMenuUpdated()

	win.StartEventLoop()
}
