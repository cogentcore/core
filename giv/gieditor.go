// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
)

// GoGiEditor opens an interactive editor of the given Ki tree, at its root
func GoGiEditor(obj ki.Ki) {
	width := 1280
	height := 920
	win := gi.NewWindow2D("gogi-editor", "GoGi Editor", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	trow := vlay.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutRow
	trow.SetStretchMaxWidth()

	spc := vlay.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.Text = fmt.Sprintf("GoGi Editor of Ki Node Tree: %v", obj.Name())
	title.SetStretchMaxWidth()
	trow.AddNewChild(gi.KiT_Stretch, "str2")

	split := vlay.AddNewChild(KiT_SplitView, "split").(*SplitView)
	split.Dim = gi.X

	tvfr := split.AddNewChild(gi.KiT_Frame, "tvfr").(*gi.Frame)
	svfr := split.AddNewChild(gi.KiT_Frame, "svfr").(*gi.Frame)
	split.SetSplits(.3, .7)

	tv := tvfr.AddNewChild(KiT_TreeView, "tv").(*TreeView)
	tv.SetRootNode(obj)

	sv := svfr.AddNewChild(KiT_StructView, "sv").(*StructView)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()
	sv.SetStruct(obj, nil)

	tv.TreeViewSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if data == nil {
			return
		}
		tvn, _ := data.(ki.Ki).EmbeddedStruct(KiT_TreeView).(*TreeView)
		svr, _ := recv.EmbeddedStruct(KiT_StructView).(*StructView)
		if sig == int64(TreeViewSelected) {
			svr.SetStruct(tvn.SrcNode.Ptr, nil)
		}
	})

	bspc := vlay.AddNewChild(gi.KiT_Space, "ButSpc").(*gi.Space)
	bspc.SetFixedHeight(units.NewValue(1.0, units.Em))

	brow := vlay.AddNewChild(gi.KiT_Layout, "brow").(*gi.Layout)
	brow.Lay = gi.LayoutRow
	brow.SetStretchMaxWidth()

	updtobj := brow.AddNewChild(gi.KiT_Button, "updtobj").(*gi.Button)
	updtobj.SetText("Update")
	updtobj.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			obj.UpdateSig()
		}
	})

	savej := brow.AddNewChild(gi.KiT_Button, "savejson").(*gi.Button)
	savej.SetText("Save JSON")
	savej.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			FileViewDialog(vp, "./", obj.Name()+".json", "Save GUI to JSON", "", obj, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					dlg, _ := send.(*gi.Dialog)
					fnm := FileViewDialogValue(dlg)
					recv.SaveJSON(fnm)
				}
			})
		}
	})

	loadj := brow.AddNewChild(gi.KiT_Button, "loadjson").(*gi.Button)
	loadj.SetText("Load JSON")
	loadj.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			FileViewDialog(vp, "./", obj.Name()+".json", "Load GUI from JSON", "", obj, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					dlg, _ := send.(*gi.Dialog)
					fnm := FileViewDialogValue(dlg)
					recv.LoadJSON(fnm)
				}
			})
		}
	})

	fontsel := brow.AddNewChild(gi.KiT_Button, "fontsel").(*gi.Button)
	fontnm := brow.AddNewChild(gi.KiT_TextField, "selfont").(*gi.TextField)
	fontnm.SetMinPrefWidth(units.NewValue(20, units.Em))

	fontsel.SetText("Select Font")
	fontsel.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			FontChooserDialog(vp, "Select a Font", "", win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sv, _ := send.(*StructTableView)
				si := sv.SelectedIdx
				if si >= 0 {
					fi := gi.FontLibrary.FontInfo[si]
					fontnm.SetText(fi.Name)
					fontnm.SelectAll()
					fontnm.Copy(false) // don't reset
				}
			})
		}
	})

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop() // in a separate goroutine
}
