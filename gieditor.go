// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"

	_ "github.com/rcoreilly/goki/gi/oswin/init"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
)

// open an interactive editor of the given Ki tree, at its root
func GoGiEditorOf(obj ki.Ki) *Window {
	width := 1280
	height := 1024
	win := NewWindow2D("GoGi Editor Window", width, height)
	win.UpdateStart()

	vp := win.WinViewport2D()
	vp.SetProp("background-color", "#FFF")
	vp.Fill = true

	vlay := vp.AddNewChild(KiT_Frame, "vlay").(*Frame)
	vlay.Lay = LayoutCol

	row1 := vlay.AddNewChild(KiT_Layout, "row1").(*Layout)
	row1.Lay = LayoutRow
	row1.SetProp("align-vert", AlignMiddle)
	row1.SetProp("align-horiz", "center")
	row1.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	row1.SetStretchMaxWidth()

	spc := vlay.AddNewChild(KiT_Space, "spc1").(*Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	row1.AddNewChild(KiT_Stretch, "str1")
	lab1 := row1.AddNewChild(KiT_Label, "lab1").(*Label)
	lab1.Text = fmt.Sprintf("GoGi Editor of Ki Node Tree: %v", obj.Name())
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	row1.AddNewChild(KiT_Stretch, "str2")

	split := vlay.AddNewChild(KiT_SplitView, "split").(*SplitView)
	split.Dim = X

	tvfr := split.AddNewChild(KiT_Frame, "tvfr").(*Frame)
	svfr := split.AddNewChild(KiT_Frame, "svfr").(*Frame)
	split.SetSplits(.2, .8)

	tv1 := tvfr.AddNewChild(KiT_TreeView, "tv").(*TreeView)
	tv1.SetSrcNode(obj)

	sv1 := svfr.AddNewChild(KiT_StructView, "sv").(*StructView)
	sv1.SetStruct(obj)

	tv1.TreeViewSig.Connect(sv1.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if data == nil {
			return
		}
		tvn, _ := data.(ki.Ki).EmbeddedStruct(KiT_TreeView).(*TreeView)
		svr, _ := recv.EmbeddedStruct(KiT_StructView).(*StructView)
		if sig == int64(NodeSelected) {
			svr.SetStruct(tvn.SrcNode.Ptr)
		}
	})

	bspc := vlay.AddNewChild(KiT_Space, "ButSpc").(*Space)
	bspc.SetFixedHeight(units.NewValue(1.0, units.Em))

	rowb := vlay.AddNewChild(KiT_Layout, "rowb").(*Layout)
	rowb.Lay = LayoutRow
	rowb.SetProp("align-vert", AlignMiddle)
	rowb.SetProp("align-horiz", "center")
	rowb.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	rowb.SetStretchMaxWidth()

	updtobj := rowb.AddNewChild(KiT_Button, "updtobj").(*Button)
	updtobj.SetText("Update")
	updtobj.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			obj.UpdateStart()
			obj.UpdateEnd()
		}
	})

	savej := rowb.AddNewChild(KiT_Button, "savejson").(*Button)
	savej.SetText("Save JSON")
	savej.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			obj.SaveJSONToFile("GoGiEditorOut.json") // todo: first a string prompt, then a file dialog
		}
	})

	loadj := rowb.AddNewChild(KiT_Button, "loadjson").(*Button)
	loadj.SetText("Load JSON")
	loadj.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			obj.LoadJSONFromFile("GoGiEditorOut.json") // todo: first a string prompt, then a file dialog
		}
	})

	win.UpdateEnd()
	win.StartEventLoopNoWait()
	return win
}
