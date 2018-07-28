// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
)

// StructViewDialog is for editing fields of a structure using a StructView --
// optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore)
func StructViewDialog(avp *gi.Viewport2D, stru interface{}, tmpSave ValueView, title, prompt string, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	dlg := gi.NewStdDialog("struct-view", title, prompt, true, true)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	sv := frame.InsertNewChild(KiT_StructView, prIdx+2, "struct-view").(*StructView)
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetStruct(stru, tmpSave)

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.SetProp("min-width", units.NewValue(60, units.Em))
	dlg.SetProp("min-height", units.NewValue(30, units.Em))
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// MapViewDialog is for editing elements of a map using a MapView -- optionally
// connects to given signal receiving object and function for dialog signals
// (nil to ignore)
func MapViewDialog(avp *gi.Viewport2D, mp interface{}, tmpSave ValueView, title, prompt string, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	dlg := gi.NewStdDialog("map-view", title, prompt, true, true)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	sv := frame.InsertNewChild(KiT_MapView, prIdx+2, "map-view").(*MapView)
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetMap(mp, tmpSave)

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.SetProp("min-width", units.NewValue(60, units.Em))
	dlg.SetProp("min-height", units.NewValue(30, units.Em))
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// SliceViewDialog for editing elements of a slice using a SliceView --
// optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore). selectOnly turns it into a selector with no
// editing of fields, and signal connection is to the selection signal, not
// the overall dialog signal
func SliceViewDialog(avp *gi.Viewport2D, mp interface{}, selectOnly bool, tmpSave ValueView, title, prompt string, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	dlg := gi.NewStdDialog("slice-view", title, prompt, true, true)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	sv := frame.InsertNewChild(KiT_SliceView, prIdx+2, "slice-view").(*SliceView)
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetInactiveState(selectOnly)
	sv.SetSlice(mp, tmpSave)

	if recv != nil && fun != nil {
		if selectOnly {
			sv.SelectSig.Connect(recv, fun)
		} else {
			dlg.DialogSig.Connect(recv, fun)
		}
	}
	dlg.SetProp("min-width", units.NewValue(50, units.Em))
	dlg.SetProp("min-height", units.NewValue(30, units.Em))
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// StructTableViewDialog is for editing / selecting fields of a
// slice-of-struct using a StructTableView -- optionally connects to given
// signal receiving object and function for signals (nil to ignore).
// selectOnly turns it into a selector with no editing of fields, and signal
// connection is to the selection signal, not the overall dialog signal
func StructTableViewDialog(avp *gi.Viewport2D, slcOfStru interface{}, selectOnly bool, tmpSave ValueView, title, prompt string, recv ki.Ki, fun ki.RecvFunc, stylefun StructTableViewStyleFunc) *gi.Dialog {
	dlg := gi.NewStdDialog("struct-table-view", title, prompt, true, true)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	sv := frame.InsertNewChild(KiT_StructTableView, prIdx+2, "struct-view").(*StructTableView)
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetInactiveState(selectOnly)
	sv.StyleFunc = stylefun
	sv.SetSlice(slcOfStru, tmpSave)

	if recv != nil && fun != nil {
		if selectOnly {
			sv.SelectSig.Connect(recv, fun)
		} else {
			dlg.DialogSig.Connect(recv, fun)
		}
	}
	dlg.SetProp("min-width", units.NewValue(50, units.Em))
	dlg.SetProp("min-height", units.NewValue(30, units.Em))
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// FontChooserDialog for choosing a font -- the recv and fun signal receivers
// if non-nil are connected to the selection signal for the struct table view,
// so they are updated with that
func FontChooserDialog(avp *gi.Viewport2D, title, prompt string, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	dlg := StructTableViewDialog(avp, &gi.FontLibrary.FontInfo, true, nil, title, prompt, recv, fun, FontInfoStyleFunc)
	return dlg
}

func FontInfoStyleFunc(slice interface{}, widg gi.Node2D, row, col int, vv ValueView) {
	if col == 3 {
		finf, ok := slice.([]gi.FontInfo)
		if ok {
			gi := widg.AsNode2D()
			gi.SetProp("font-family", (finf)[row].Name)
			gi.SetProp("font-style", (finf)[row].Style)
			gi.SetProp("font-weight", (finf)[row].Weight)
		}
	}
}

// IconChooserDialog for choosing an Icon -- the recv and fun signal receivers
// if non-nil are connected to the selection signal for the slice view
func IconChooserDialog(avp *gi.Viewport2D, title, prompt string, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	dlg := SliceViewDialog(avp, &gi.CurIconList, true, nil, title, prompt, recv, fun)
	return dlg
}

// ColorViewDialog for editing a color using a ColorView -- optionally
// connects to given signal receiving object and function for dialog signals
// (nil to ignore)
func ColorViewDialog(avp *gi.Viewport2D, clr *gi.Color, tmpSave ValueView, title, prompt string, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	dlg := gi.NewStdDialog("color-view", title, prompt, true, true)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	sv := frame.InsertNewChild(KiT_ColorView, prIdx+2, "color-view").(*ColorView)
	sv.SetColor(clr, tmpSave)

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// FileViewDialog is for selecting / manipulating files -- if recv and fun are
// non-nil, they connect to the dialog signals
func FileViewDialog(avp *gi.Viewport2D, path, file string, title, prompt string, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	dlg := gi.NewStdDialog("file-view", title, prompt, true, true)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	fv := frame.InsertNewChild(KiT_FileView, prIdx+2, "file-view").(*FileView)
	fv.SetStretchMaxHeight()
	fv.SetStretchMaxWidth()
	fv.SetPathFile(path, file)

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	// dlg.SetMinPrefWidth(units.NewValue(40, units.Em))
	// dlg.SetMinPrefHeight(units.NewValue(35, units.Em))
	dlg.SetProp("min-width", units.NewValue(60, units.Em))
	dlg.SetProp("min-height", units.NewValue(35, units.Em))
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// FileViewDialogValue gets the full path of selected file
func FileViewDialogValue(dlg *gi.Dialog) string {
	frame := dlg.Frame()
	fv := frame.ChildByName("file-view", 0).(*FileView)
	return fv.SelectedFile()
}
