// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
)

func gieditSaveGUI(vp *gi.Viewport2D, obj ki.Ki) {
	FileViewDialog(vp, "./", obj.Name()+".json", "Save GUI to JSON", "", nil, obj, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.DialogAccepted) {
			dlg, _ := send.(*gi.Dialog)
			fnm := FileViewDialogValue(dlg)
			recv.SaveJSON(fnm)
		}
	})
}

func gieditLoadGUI(vp *gi.Viewport2D, obj ki.Ki) {
	FileViewDialog(vp, "./", obj.Name()+".json", "Load GUI from JSON", "", nil, obj, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.DialogAccepted) {
			dlg, _ := send.(*gi.Dialog)
			fnm := FileViewDialogValue(dlg)
			recv.LoadJSON(fnm)
		}
	})
}

// GoGiEditor opens an interactive editor of the given Ki tree, at its root
func GoGiEditor(obj ki.Ki) {
	width := 1280
	height := 920
	win := gi.NewWindow2D("gogi-editor", "GoGi Editor", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	tbar := mfr.AddNewChild(gi.KiT_ToolBar, "tbar").(*gi.ToolBar)
	tbar.Lay = gi.LayoutHoriz
	tbar.SetStretchMaxWidth()

	trow := mfr.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutHoriz
	trow.SetStretchMaxWidth()

	spc := mfr.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.Text = fmt.Sprintf("GoGi Editor of Ki Node Tree: %v", obj.Name())
	title.SetStretchMaxWidth()
	trow.AddNewChild(gi.KiT_Stretch, "str2")

	split := mfr.AddNewChild(gi.KiT_SplitView, "split").(*gi.SplitView)
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
		tvn, _ := data.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
		svr, _ := recv.Embed(KiT_StructView).(*StructView)
		if sig == int64(TreeViewSelected) {
			svr.SetStruct(tvn.SrcNode.Ptr, nil)
		}
	})

	bspc := mfr.AddNewChild(gi.KiT_Space, "ButSpc").(*gi.Space)
	bspc.SetFixedHeight(units.NewValue(1.0, units.Em))

	updtobj := tbar.AddNewChild(gi.KiT_Action, "updtobj").(*gi.Action)
	updtobj.SetText("Update")
	updtobj.Tooltip = "Updates the source window based on changes made in the editor"
	updtobj.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		obj.UpdateSig()
	})

	savej := tbar.AddNewChild(gi.KiT_Action, "savejson").(*gi.Action)
	savej.SetText("Save JSON")
	savej.Tooltip = "Save current scenegraph as a JSON-formatted file that can then be Loaded and will re-create the GUI display as it currently is (signal connections are not saved)"
	savej.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		gieditSaveGUI(vp, obj)
	})

	loadj := tbar.AddNewChild(gi.KiT_Action, "loadjson").(*gi.Action)
	loadj.SetText("Load JSON")
	loadj.Tooltip = "Load a previously-saved JSON-formatted scenegraph"
	loadj.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		gieditLoadGUI(vp, obj)
	})

	fontsel := tbar.AddNewChild(gi.KiT_Action, "fontsel").(*gi.Action)
	fontnm := tbar.AddNewChild(gi.KiT_TextField, "selfont").(*gi.TextField)
	fontnm.SetMinPrefWidth(units.NewValue(20, units.Em))
	fontnm.Tooltip = "shows the font name selected via Select Font"

	fontsel.SetText("Select Font")
	fontsel.Tooltip = "pulls up a font selection dialog -- font name will be copied to clipboard so you can paste it into any relevant fields"
	fontsel.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		FontChooserDialog(vp, "Select a Font", "", nil, win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			sv, _ := send.(*TableView)
			si := sv.SelectedIdx
			if si >= 0 {
				fi := gi.FontLibrary.FontInfo[si]
				fontnm.SetText(fi.Name)
				fontnm.SelectAll()
				fontnm.Copy(false) // don't reset
			}
		}, nil)
	})

	// main menu
	appnm := oswin.TheApp.Name()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.KnownChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	fmen := win.MainMenu.KnownChildByName("File", 0).(*gi.Action)
	fmen.Menu = make(gi.Menu, 0, 10)
	fmen.Menu.AddMenuText("Update", "Command+U", win.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		obj.UpdateSig()
	})
	fmen.Menu.AddMenuText("Load", "Command+O", win.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		gieditLoadGUI(vp, obj)
	})
	fmen.Menu.AddMenuText("Save", "Command+S", win.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		gieditSaveGUI(vp, obj)
	})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddMenuText("Close Window", "Command+W", win.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		win.OSWin.CloseReq()
	})

	emen := win.MainMenu.KnownChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win, false)

	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		gi.ChoiceDialog(vp, "Save JSON Before Closing?", "Do you want to save to JSON before closing?", []string{"Close Without Saving", "Save JSON", "Cancel"}, nil, win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			switch sig {
			case 0:
				w.Close()
			case 1:
				gieditSaveGUI(vp, obj)
			case 2:
				// default is to do nothing, i.e., cancel
			}
		})
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop() // in a separate goroutine
}
