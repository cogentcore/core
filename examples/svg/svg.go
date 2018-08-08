// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"

	"github.com/goki/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
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

func LoadSVG(fnm string) {
	CurFilename = fnm
	TheFile.SetText(CurFilename)
	updt := TheSVG.UpdateStart()
	TheSVG.SetFullReRender()
	fmt.Printf("Loading: %v\n", CurFilename)
	TheSVG.LoadXML(CurFilename)
	SetZoom(TheSVG.Viewport.Win.LogicalDPI() / 96.0)
	SetTrans(0, 0)
	TheSVG.UpdateEnd(updt)
}

func mainrun() {
	width := 1600
	height := 1200

	win := gi.NewWindow2D("gogi-svg-viewer", "GoGi SVG Viewer", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := mfr.AddNewChild(gi.KiT_ToolBar, "tbar").(*gi.ToolBar)
	tbar.Lay = gi.LayoutHoriz
	tbar.SetStretchMaxWidth()

	svgrow := mfr.AddNewChild(gi.KiT_Layout, "svgrow").(*gi.Layout)
	svgrow.Lay = gi.LayoutHoriz
	svgrow.SetProp("horizontal-align", "center")
	svgrow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	svgrow.SetStretchMaxWidth()
	svgrow.SetStretchMaxHeight()

	svge := svgrow.AddNewChild(svg.KiT_Editor, "svg").(*svg.Editor)
	TheSVG = svge
	svge.InitScale()
	svge.Fill = true
	svge.SetProp("background-color", "white")
	svge.SetProp("width", units.NewValue(float32(width-20), units.Px))
	svge.SetProp("height", units.NewValue(float32(height-100), units.Px))
	svge.SetStretchMaxWidth()
	svge.SetStretchMaxHeight()

	loads := tbar.AddNewChild(gi.KiT_Action, "loadsvg").(*gi.Action)
	loads.SetText("Load SVG")

	fnm := tbar.AddNewChild(gi.KiT_TextField, "cur-fname").(*gi.TextField)
	TheFile = fnm
	fnm.SetMinPrefWidth(units.NewValue(40, units.Em))

	zmlb := tbar.AddNewChild(gi.KiT_Label, "zmlb").(*gi.Label)
	zmlb.Text = "Zoom: "
	zmlb.SetProp("vertical-align", gi.AlignMiddle)
	zmlb.Tooltip = "zoom scaling factor -- can use mouse scrollwheel to zoom as well"
	zoomout := tbar.AddNewChild(gi.KiT_Action, "zoomout").(*gi.Action)
	zoomout.SetProp("margin", 0)
	zoomout.SetProp("padding", 0)
	zoomout.SetProp("#icon", ki.Props{
		"width":  units.NewValue(1.5, units.Em),
		"height": units.NewValue(1.5, units.Em),
	})
	zoomout.SetIcon("zoom-out")
	zoomout.Tooltip = "zoom out"

	zoom := tbar.AddNewChild(gi.KiT_SpinBox, "zoom").(*gi.SpinBox)
	// zoom.SetMinPrefWidth(units.NewValue(10, units.Em))
	zoom.SetValue(svge.Scale)
	zoom.Tooltip = "zoom scaling factor -- can use mouse scrollwheel to zoom as well"
	TheZoom = zoom

	zoomin := tbar.AddNewChild(gi.KiT_Action, "zoomin").(*gi.Action)
	zoomin.Tooltip = "zoom in"
	zoomin.SetProp("margin", 0)
	zoomin.SetProp("padding", 0)
	zoomin.SetProp("#icon", ki.Props{
		"width":  units.NewValue(1.5, units.Em),
		"height": units.NewValue(1.5, units.Em),
	})
	zoomin.SetIcon("zoom-in")

	tbar.AddNewChild(gi.KiT_Space, "spctr")
	trlb := tbar.AddNewChild(gi.KiT_Label, "trlb").(*gi.Label)
	trlb.Text = "Translate: "
	trlb.Tooltip = "Translation of overall image -- can use mouse drag to move as well"
	trlb.SetProp("vertical-align", gi.AlignMiddle)

	trx := tbar.AddNewChild(gi.KiT_SpinBox, "trx").(*gi.SpinBox)
	// zoom.SetMinPrefWidth(units.NewValue(10, units.Em))
	trx.SetValue(svge.Trans.X)
	TheTransX = trx

	try := tbar.AddNewChild(gi.KiT_SpinBox, "try").(*gi.SpinBox)
	// zoom.SetMinPrefWidth(units.NewValue(10, units.Em))
	try.SetValue(svge.Trans.Y)
	TheTransY = try

	loads.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		path, fn := filepath.Split(CurFilename)
		path, _ = homedir.Expand(path)
		giv.FileViewDialog(vp, path, fn, "Load SVG", "", nil, win, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				dlg, _ := send.(*gi.Dialog)
				LoadSVG(giv.FileViewDialogValue(dlg))
			}
		})
	})

	fnm.TextFieldSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			tf := send.(*gi.TextField)
			fn, _ := homedir.Expand(tf.Text())
			LoadSVG(fn)
		}
	})

	zoomin.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		SetZoom(svge.Scale * 1.1)
		win.FullReRender()
	})

	zoomout.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		SetZoom(svge.Scale * 0.9)
		win.FullReRender()
	})

	zoom.SpinBoxSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		sp := send.(*gi.SpinBox)
		SetZoom(sp.Value)
		win.FullReRender()
	})

	svge.NodeSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		ssvg := send.EmbeddedStruct(svg.KiT_Editor).(*svg.Editor)
		SetZoom(ssvg.Scale)
		SetTrans(ssvg.Trans.X, ssvg.Trans.Y)
	})

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
