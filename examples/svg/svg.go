// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

// TODO: fix
/*
func main() { gimain.Run(app) }

var CurFilename = ""
var TheSVG *svg.Editor
var TheZoom *gi.Spinner
var TheTransX *gi.Spinner
var TheTransY *gi.Spinner
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
	SetZoom(TheSVG.ParentRenderWin().LogicalDPI() / 96.0)
	SetTrans(0, 0)
	TheSVG.UpdateEnd(updt)
}

func FileViewOpenSVG(ctx gi.Widget) {
	giv.FileViewDialog(ctx, giv.DlgOpts{Title: "Open SVG"}, CurFilename, func(dlg *gi.Dialog) {
		if dlg.Accepted {
			OpenSVG(dlg.Data.(string))
		})
}

func app() {
	width := 1600
	height := 1200

	gi.SetAppName("svg")
	gi.SetAppAbout(`This is a demo of the SVG rendering (and start on editing) in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>
<p>You can drag the image around and use the scroll wheel to zoom.</p>`)

	win := gi.NewMainRenderWin("gogi-svg-viewer", "GoGi SVG Viewer", width, height)

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
	svge.SetProp("width", units.Dp(float32(width-20)))
	svge.SetProp("height", units.Dp(float32(height-100)))
	svge.SetStretchMaxWidth()
	svge.SetStretchMaxHeight()

	loads := tbar.AddButton(gi.ActOpts{Label: "Open SVG", Icon: icons.FileOpen}, func(act *gi.Button) {
		FileViewOpenSVG(act)
	})
	loads.StartFocus()

	fnm := gi.NewTextField(tbar, "cur-fname")
	TheFile = fnm
	fnm.SetMinPrefWidth(units.Ch(60))

	zmlb := gi.NewLabel(tbar, "zmlb", "Zoom: ")
	zmlb.SetProp("vertical-align", styles.AlignMiddle)
	zmlb.Tooltip = "zoom scaling factor -- can use mouse scrollwheel to zoom as well"

	zoomout := tbar.AddButton(gi.ActOpts{Icon: icons.ZoomOut, Name: "zoomout", Tooltip: "zoom out"},
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
		sp := send.(*gi.Spinner)
		SetZoom(sp.Value)
		win.FullReRender()
	})

	zoomin := tbar.AddButton(gi.ActOpts{Icon: icons.ZoomIn, Name: "zoomin", Tooltip: " zoom in"},
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
	trlb.SetProp("vertical-align", styles.AlignMiddle)

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
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "RenderWin"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Button)
	amen.Menu = make(gi.MenuStage, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Button)
	emen.Menu = make(gi.MenuStage, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	// note: Command in shortcuts is automatically translated into Control for
	// Linux, RenderWins or Meta for MacOS
	fmen := win.MainMenu.ChildByName("File", 0).(*gi.Button)
	fmen.Menu = make(gi.MenuStage, 0, 10)
	fmen.Menu.AddButton(gi.ActOpts{Label: "Open", Shortcut: "Command+O"}, func(act *gi.Button) {
		FileViewOpenSVG(act)
	})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddButton(gi.ActOpts{Label: "Close RenderWin", Shortcut: "Command+W"},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			win.RenderWin.Close()
		})

	win.SetCloseCleanFunc(func(w *gi.RenderWin) {
		go gi.Quit() // once main window is closed, quit
	})

	// todo: when saving works, add option to save, and change above to CloseReq

	win.MainMenuUpdated()

	win.StartEventLoop()
}
*/
