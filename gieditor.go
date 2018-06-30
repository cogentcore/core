// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
)

// open an interactive editor of the given Ki tree, at its root
func GoGiEditorOf(obj ki.Ki) {
	width := 1280
	height := 920
	win := NewWindow2D("gogi-editor", "GoGi Editor", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	vp.Fill = true

	vlay := vp.AddNewChild(KiT_Frame, "vlay").(*Frame)
	vlay.Lay = LayoutCol

	trow := vlay.AddNewChild(KiT_Layout, "trow").(*Layout)
	trow.Lay = LayoutRow
	trow.SetStretchMaxWidth()

	spc := vlay.AddNewChild(KiT_Space, "spc1").(*Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(KiT_Stretch, "str1")
	title := trow.AddNewChild(KiT_Label, "title").(*Label)
	title.Text = fmt.Sprintf("GoGi Editor of Ki Node Tree: %v", obj.Name())
	title.SetStretchMaxWidth()
	trow.AddNewChild(KiT_Stretch, "str2")

	split := vlay.AddNewChild(KiT_SplitView, "split").(*SplitView)
	split.Dim = X

	tvfr := split.AddNewChild(KiT_Frame, "tvfr").(*Frame)
	svfr := split.AddNewChild(KiT_Frame, "svfr").(*Frame)
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

	bspc := vlay.AddNewChild(KiT_Space, "ButSpc").(*Space)
	bspc.SetFixedHeight(units.NewValue(1.0, units.Em))

	brow := vlay.AddNewChild(KiT_Layout, "brow").(*Layout)
	brow.Lay = LayoutRow
	brow.SetStretchMaxWidth()

	updtobj := brow.AddNewChild(KiT_Button, "updtobj").(*Button)
	updtobj.SetText("Update")
	updtobj.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			obj.UpdateSig()
		}
	})

	savej := brow.AddNewChild(KiT_Button, "savejson").(*Button)
	savej.SetText("Save JSON")
	savej.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			FileViewDialog(vp, "./", obj.Name()+".json", "Save GUI to JSON", "", obj, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(DialogAccepted) {
					dlg, _ := send.(*Dialog)
					fnm := FileViewDialogValue(dlg)
					recv.SaveJSON(fnm)
				}
			})
		}
	})

	loadj := brow.AddNewChild(KiT_Button, "loadjson").(*Button)
	loadj.SetText("Load JSON")
	loadj.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			FileViewDialog(vp, "./", obj.Name()+".json", "Load GUI from JSON", "", obj, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(DialogAccepted) {
					dlg, _ := send.(*Dialog)
					fnm := FileViewDialogValue(dlg)
					recv.LoadJSON(fnm)
				}
			})
		}
	})

	fontsel := brow.AddNewChild(KiT_Button, "fontsel").(*Button)
	fontnm := brow.AddNewChild(KiT_TextField, "selfont").(*TextField)
	fontnm.SetMinPrefWidth(units.NewValue(20, units.Em))

	fontsel.SetText("Select Font")
	fontsel.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			FontChooserDialog(vp, "Select a Font", "", win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sv, _ := send.(*StructTableView)
				si := sv.SelectedIdx
				if si >= 0 {
					fi := FontLibrary.FontInfo[si]
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
