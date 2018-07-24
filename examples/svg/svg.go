// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/mitchellh/go-homedir"
)

func main() {
	driver.Main(func(app oswin.App) {
		mainrun()
	})
}

var CurFilename = ""
var TheSVG *gi.SVGEdit
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

	// turn this on to see a trace of the rendering
	// gi.Update2DTrace = true
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	win := gi.NewWindow2D("gogi-svg-viewer", "GoGi SVG Viewer", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	brow := vlay.AddNewChild(gi.KiT_Layout, "brow").(*gi.Layout)
	brow.Lay = gi.LayoutRow
	brow.SetStretchMaxWidth()

	svgrow := vlay.AddNewChild(gi.KiT_Layout, "svgrow").(*gi.Layout)
	svgrow.Lay = gi.LayoutRow
	svgrow.SetProp("align-horiz", "center")
	svgrow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	svgrow.SetStretchMaxWidth()
	svgrow.SetStretchMaxHeight()

	svg := svgrow.AddNewChild(gi.KiT_SVGEdit, "svg").(*gi.SVGEdit)
	TheSVG = svg
	svg.InitScale()
	svg.Fill = true
	svg.SetProp("background-color", "white")
	svg.SetProp("width", units.NewValue(float32(width-20), units.Px))
	svg.SetProp("height", units.NewValue(float32(height-100), units.Px))
	svg.SetStretchMaxWidth()
	svg.SetStretchMaxHeight()

	loads := brow.AddNewChild(gi.KiT_Button, "loadsvg").(*gi.Button)
	loads.SetText("Load SVG")

	fnm := brow.AddNewChild(gi.KiT_TextField, "cur-fname").(*gi.TextField)
	TheFile = fnm
	fnm.SetMinPrefWidth(units.NewValue(40, units.Em))

	zmlb := brow.AddNewChild(gi.KiT_Label, "zmlb").(*gi.Label)
	zmlb.Text = "Zoom: "
	zmlb.SetProp("align-vert", gi.AlignMiddle)
	zoomout := brow.AddNewChild(gi.KiT_Button, "zoomout").(*gi.Button)
	zoomout.SetIcon("zoom-out")

	zoom := brow.AddNewChild(gi.KiT_SpinBox, "zoom").(*gi.SpinBox)
	// zoom.SetMinPrefWidth(units.NewValue(10, units.Em))
	zoom.SetValue(svg.Scale)
	TheZoom = zoom

	zoomin := brow.AddNewChild(gi.KiT_Button, "zoomin").(*gi.Button)
	zoomin.SetProp("margin", 0)
	zoomin.SetProp("padding", 0)
	zoomin.SetIcon("zoom-in")
	zoomin.SetProp("#icon", ki.Props{ // todo: not working
		"width":  units.NewValue(2, units.Em),
		"height": units.NewValue(2, units.Em),
	})

	brow.AddNewChild(gi.KiT_Space, "spctr")
	trlb := brow.AddNewChild(gi.KiT_Label, "trlb").(*gi.Label)
	trlb.Text = "Translate: "
	trlb.SetProp("align-vert", gi.AlignMiddle)

	trx := brow.AddNewChild(gi.KiT_SpinBox, "trx").(*gi.SpinBox)
	// zoom.SetMinPrefWidth(units.NewValue(10, units.Em))
	trx.SetValue(svg.Trans.X)
	TheTransX = trx

	try := brow.AddNewChild(gi.KiT_SpinBox, "try").(*gi.SpinBox)
	// zoom.SetMinPrefWidth(units.NewValue(10, units.Em))
	try.SetValue(svg.Trans.Y)
	TheTransY = try

	loads.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			path, fn := filepath.Split(CurFilename)
			path, _ = homedir.Expand(path)
			gi.FileViewDialog(vp, path, fn, "Load SVG", "", win, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					dlg, _ := send.(*gi.Dialog)
					LoadSVG(gi.FileViewDialogValue(dlg))
				}
			})
		}
	})

	fnm.TextFieldSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			tf := send.(*gi.TextField)
			fn, _ := homedir.Expand(tf.Text())
			LoadSVG(fn)
		}
	})

	zoomin.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			SetZoom(svg.Scale * 1.1)
			win.FullReRender()
		}
	})

	zoomout.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			SetZoom(svg.Scale * 0.9)
			win.FullReRender()
		}
	})

	zoom.SpinBoxSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		sp := send.(*gi.SpinBox)
		SetZoom(sp.Value)
		win.FullReRender()
	})

	svg.NodeSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		ssvg := send.EmbeddedStruct(gi.KiT_SVGEdit).(*gi.SVGEdit)
		SetZoom(ssvg.Scale)
		SetTrans(ssvg.Trans.X, ssvg.Trans.Y)
	})

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
