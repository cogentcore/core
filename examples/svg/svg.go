// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func main() {}

// TODO: fix
/*
var CurFilename = ""
var TheSVG *svg.Editor
var TheZoom *core.Spinner
var TheTransX *core.Spinner
var TheTransY *core.Spinner
var TheFile *core.TextField

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
	update := TheSVG.UpdateStart()
	TheSVG.SetFullReRender()
	fmt.Printf("Opening: %v\n", CurFilename)
	TheSVG.OpenXML(core.Filename(CurFilename))
	SetZoom(TheSVG.ParentRenderWindow().LogicalDPI() / 96.0)
	SetTrans(0, 0)
	TheSVG.UpdateEnd(update)
}

func FilePickerOpenSVG(ctx core.Widget) {
	core.FilePickerDialog(ctx, core.DlgOpts{Title: "Open SVG"}, CurFilename, func(dlg *core.Dialog) {
		if dlg.Accepted {
			OpenSVG(dlg.Data.(string))
		})
}

func main() {
	width := 1600
	height := 1200

	core.SetAppName("svg")

	win := core.NewMainRenderWindow("core-svg-viewer", "Cogent Core SVG Viewer", width, height)

	vp := win.WinScene()
	update := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := core.NewToolbar(mfr, "tbar")
	tbar.SetStretchMaxWidth()

	svgrow := core.NewFrame(mfr, "svgrow", core.LayoutHoriz)
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

	loads := tbar.AddButton(core.ActOpts{Label: "Open SVG", Icon: icons.Open}, func(act *core.Button) {
		FilePickerOpenSVG(act)
	})
	loads.StartFocus()

	fnm := core.NewTextField(tbar, "cur-fname")
	TheFile = fnm
	fnm.SetMinPrefWidth(units.Ch(60))

	zmlb := core.NewText(tbar, "zmlb", "Zoom: ")
	zmlb.SetProp("vertical-align", styles.Middle)
	zmlb.Tooltip = "zoom scaling factor -- can use mouse scrollwheel to zoom as well"

	zoomout := tbar.AddButton(core.ActOpts{Icon: icons.ZoomOut, Name: "zoomout", Tooltip: "zoom out"},
		win.This, func(recv, send tree.Node, sig int64, data any) {
			SetZoom(svge.Scale * 0.9)
			win.FullReRender()
		})
	zoomout.SetProp("margin", 0)
	zoomout.SetProp("padding", 0)
	zoomout.SetProp("#icon", tree.Properties{
		"width":  units.Em(1.5),
		"height": units.Em(1.5),
	})
	zoom := core.NewSpinBox(tbar, "zoom")
	// zoom.SetMinPrefWidth(units.NewEm(10))
	zoom.SetValue(svge.Scale)
	zoom.Tooltip = "zoom scaling factor -- can use mouse scrollwheel to zoom as well"
	TheZoom = zoom
	zoom.SpinBoxSig.Connect(win.This, func(recv, send tree.Node, sig int64, data any) {
		sp := send.(*core.Spinner)
		SetZoom(sp.Value)
		win.FullReRender()
	})

	zoomin := tbar.AddButton(core.ActOpts{Icon: icons.ZoomIn, Name: "zoomin", Tooltip: " zoom in"},
		win.This, func(recv, send tree.Node, sig int64, data any) {
			SetZoom(svge.Scale * 1.1)
			win.FullReRender()
		})
	zoomin.SetProp("margin", 0)
	zoomin.SetProp("padding", 0)
	zoomin.SetProp("#icon", tree.Properties{
		"width":  units.Em(1.5),
		"height": units.Em(1.5),
	})

	core.NewSpace(tbar, "spctr")
	trlb := core.NewText(tbar, "trlb", "Translate: ")
	trlb.Tooltip = "Translation of overall image -- can use mouse drag to move as well"
	trlb.SetProp("vertical-align", styles.Middle)

	trx := core.NewSpinBox(tbar, "trx")
	// zoom.SetMinPrefWidth(units.NewEm(10))
	trx.SetValue(svge.Trans.X)
	TheTransX = trx
	trx.SpinBoxSig.Connect(win.This, func(recv, send tree.Node, sig int64, data any) {
		SetTrans(trx.Value, svge.Trans.Y)
		win.FullReRender()
	})

	try := core.NewSpinBox(tbar, "try")
	// zoom.SetMinPrefWidth(units.NewEm(10))
	try.SetValue(svge.Trans.Y)
	TheTransY = try
	try.SpinBoxSig.Connect(win.This, func(recv, send tree.Node, sig int64, data any) {
		SetTrans(svge.Trans.X, try.Value)
		win.FullReRender()
	})

	fnm.TextFieldSig.Connect(win.This, func(recv, send tree.Node, sig int64, data any) {
		if sig == int64(core.TextFieldDone) {
			tf := send.(*core.TextField)
			fn, _ := homedir.Expand(tf.Text())
			OpenSVG(fn)
		}
	})

	svge.NodeSig.Connect(win.This, func(recv, send tree.Node, sig int64, data any) {
		ssvg := send.Embed(svg.TypeEditor).(*svg.Editor)
		SetZoom(ssvg.Scale)
		SetTrans(ssvg.Trans.X, ssvg.Trans.Y)
	})

	vp.UpdateEndNoSig(update)

	// main menu
	appnm := core.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "RenderWin"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*core.Button)
	amen.Menu = make(core.MenuStage, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*core.Button)
	emen.Menu = make(core.MenuStage, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	// note: Command in shortcuts is automatically translated into Control for
	// Linux, Windows or Meta for MacOS
	fmen := win.MainMenu.ChildByName("File", 0).(*core.Button)
	fmen.Menu = make(core.MenuStage, 0, 10)
	fmen.Menu.AddButton(core.ActOpts{Label: "Open", Shortcut: "Command+O"}, func(act *core.Button) {
		FilePickerOpenSVG(act)
	})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddButton(core.ActOpts{Label: "Close RenderWin", Shortcut: "Command+W"},
		win.This, func(recv, send tree.Node, sig int64, data any) {
			win.RenderWindow.Close()
		})

	win.SetCloseCleanFunc(func(w *core.RenderWindow) {
		go core.Quit() // once main window is closed, quit
	})

	// todo: when saving works, add option to save, and change above to CloseReq

	win.MainMenuUpdated()

	win.StartEventLoop()
}
*/
