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

func SetZoom(zf float32) {
	TheSVG.Scale = zf
	TheZoom.SetValue(zf)
	TheSVG.SetTransform()
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
	fnm.SetMinPrefWidth(units.NewValue(20, units.Em))

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

	loads.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			path, fn := filepath.Split(CurFilename)
			path, _ = homedir.Expand(path)
			gi.FileViewDialog(vp, path, fn, "Load SVG", "", win, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					dlg, _ := send.(*gi.Dialog)
					CurFilename = gi.FileViewDialogValue(dlg)
					fnm.SetText(CurFilename)
					updt := svg.UpdateStart()
					fmt.Printf("Loading: %v\n", CurFilename)
					svg.LoadXML(CurFilename)
					svg.SetFullReRender()
					SetZoom(svg.Viewport.Win.LogicalDPI() / 96.0)
					svg.UpdateEnd(updt)
				}
			})
		}
	})

	fnm.TextFieldSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			tf := send.(*gi.TextField)
			CurFilename, _ = homedir.Expand(tf.Text())
			updt := svg.UpdateStart()
			fmt.Printf("Loading: %v\n", CurFilename)
			svg.LoadXML(CurFilename)
			svg.SetFullReRender()
			SetZoom(svg.Viewport.Win.LogicalDPI() / 96.0)
			svg.UpdateEnd(updt)
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

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
