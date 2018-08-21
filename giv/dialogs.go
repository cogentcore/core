// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/iancoleman/strcase"
)

// StructViewDialog is for editing fields of a structure using a StructView --
// optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore)
func StructViewDialog(avp *gi.Viewport2D, stru interface{}, tmpSave ValueView, title, prompt string, css ki.Props, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	winm := strcase.ToKebab(title)
	dlg := gi.NewStdDialog(winm, title, prompt, true, true, css)

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
func MapViewDialog(avp *gi.Viewport2D, mp interface{}, tmpSave ValueView, title, prompt string, css ki.Props, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	winm := strcase.ToKebab(title)
	dlg := gi.NewStdDialog(winm, title, prompt, true, true, css)

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
// dialog signals (nil to ignore).    Also has an optional styling
// function for styling elements of the table.
func SliceViewDialog(avp *gi.Viewport2D, slice interface{}, tmpSave ValueView, title, prompt string, css ki.Props, recv ki.Ki, fun ki.RecvFunc, stylefun SliceViewStyleFunc) *gi.Dialog {
	winm := strcase.ToKebab(title)
	dlg := gi.NewStdDialog(winm, title, prompt, true, true, css)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	sv := frame.InsertNewChild(KiT_SliceView, prIdx+2, "slice-view").(*SliceView)
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetInactiveState(false)
	sv.StyleFunc = stylefun
	sv.SetSlice(slice, tmpSave)

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.SetProp("min-width", units.NewValue(50, units.Em))
	dlg.SetProp("min-height", units.NewValue(30, units.Em))
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// SliceViewSelectDialog for selecting one row from given slice -- connections
// functions available for both the widget signal reporting selection events,
// and the overall dialog signal.  Also has an optional styling function for
// styling elements of the table.
func SliceViewSelectDialog(avp *gi.Viewport2D, slice, curVal interface{}, title, prompt string, initRow int, css ki.Props, recv ki.Ki, selFun ki.RecvFunc, dlgFun ki.RecvFunc, stylefun SliceViewStyleFunc) *gi.Dialog {
	if css == nil {
		css = ki.Props{
			"textfield": ki.Props{
				":inactive": ki.Props{
					"background-color": &gi.Prefs.Colors.Control,
				},
			},
		}
	}
	winm := strcase.ToKebab(title)
	dlg := gi.NewStdDialog(winm, title, prompt, true, true, css)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	sv := frame.InsertNewChild(KiT_SliceView, prIdx+2, "slice-view").(*SliceView)
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetInactiveState(true)
	sv.SelectedIdx = initRow
	sv.StyleFunc = stylefun
	sv.SelVal = curVal
	sv.SetSlice(slice, nil)

	sv.SliceViewSig.Connect(dlg.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(SliceViewDoubleClicked) {
			ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
			ddlg.Accept()
		}
	})

	if recv != nil {
		if selFun != nil {
			sv.WidgetSig.Connect(recv, selFun)
		}
		if dlgFun != nil {
			dlg.DialogSig.Connect(recv, dlgFun)
		}
	}
	dlg.SetProp("min-width", units.NewValue(50, units.Em))
	dlg.SetProp("min-height", units.NewValue(30, units.Em))
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// TableViewDialog is for editing fields of a slice-of-struct using a
// TableView -- optionally connects to given signal receiving object and
// function for dialog signals (nil to ignore).  Also has an optional styling
// function for styling elements of the table.
func TableViewDialog(avp *gi.Viewport2D, slcOfStru interface{}, tmpSave ValueView, title, prompt string, css ki.Props, recv ki.Ki, fun ki.RecvFunc, stylefun TableViewStyleFunc) *gi.Dialog {
	winm := strcase.ToKebab(title)
	dlg := gi.NewStdDialog(winm, title, prompt, true, true, css)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	sv := frame.InsertNewChild(KiT_TableView, prIdx+2, "tableview").(*TableView)
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetInactiveState(false)
	sv.StyleFunc = stylefun
	sv.SetSlice(slcOfStru, tmpSave)

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.SetProp("min-width", units.NewValue(50, units.Em))
	dlg.SetProp("min-height", units.NewValue(30, units.Em))
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// TableViewSelectDialog is for selecting a row from a slice-of-struct using a
// TableView -- optionally connects to given signal receiving object and
// functions for signals (nil to ignore): selFun for the widget signal
// reporting selection events, and dlgFun for the overall dialog signals.
// Also has an optional styling function for styling elements of the table.
func TableViewSelectDialog(avp *gi.Viewport2D, slcOfStru interface{}, title, prompt string, initRow int, css ki.Props, recv ki.Ki, selFun ki.RecvFunc, dlgFun ki.RecvFunc, styleFun TableViewStyleFunc) *gi.Dialog {
	if css == nil {
		css = ki.Props{
			"textfield": ki.Props{
				":inactive": ki.Props{
					"background-color": &gi.Prefs.Colors.Control,
				},
			},
		}
	}
	winm := strcase.ToKebab(title)
	dlg := gi.NewStdDialog(winm, title, prompt, true, true, css)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	sv := frame.InsertNewChild(KiT_TableView, prIdx+2, "tableview").(*TableView)
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetInactiveState(true)
	sv.StyleFunc = styleFun
	sv.SelectedIdx = initRow
	sv.SetSlice(slcOfStru, nil)

	sv.TableViewSig.Connect(dlg.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(TableViewDoubleClicked) {
			ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
			ddlg.Accept()
		}
	})

	if recv != nil {
		if selFun != nil {
			sv.WidgetSig.Connect(recv, selFun)
		}
		if dlgFun != nil {
			dlg.DialogSig.Connect(recv, dlgFun)
		}
	}
	dlg.SetProp("min-width", units.NewValue(50, units.Em))
	dlg.SetProp("min-height", units.NewValue(30, units.Em))
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp)
	return dlg
}

// show fonts in a bigger size so you can actually see the differences
var FontChooserSize = 18
var FontChooserSizeDots = 18

// FontChooserDialog for choosing a font -- the recv and fun signal receivers
// if non-nil are connected to the selection signal for the struct table view,
// so they are updated with that
func FontChooserDialog(avp *gi.Viewport2D, title, prompt string, css ki.Props, recv ki.Ki, selFun ki.RecvFunc, dlgFun ki.RecvFunc) *gi.Dialog {
	FontChooserSizeDots = int(avp.Sty.UnContext.ToDots(float32(FontChooserSize), units.Pt))
	gi.FontLibrary.LoadAllFonts(FontChooserSizeDots)
	dlg := TableViewSelectDialog(avp, &gi.FontLibrary.FontInfo, title, prompt, -1, css, recv, selFun, dlgFun, FontInfoStyleFunc)
	return dlg
}

func FontInfoStyleFunc(tv *TableView, slice interface{}, widg gi.Node2D, row, col int, vv ValueView) {
	if col == 4 {
		finf, ok := slice.([]gi.FontInfo)
		if ok {
			widg.SetProp("font-family", (finf)[row].Name)
			widg.SetProp("font-stretch", (finf)[row].Stretch)
			widg.SetProp("font-weight", (finf)[row].Weight)
			widg.SetProp("font-style", (finf)[row].Style)
			widg.SetProp("font-size", units.NewValue(float32(FontChooserSize), units.Pt))
		}
	}
}

// IconChooserDialog for choosing an Icon -- the recv and fun signal receivers
// if non-nil are connected to the selection signal for the slice view
func IconChooserDialog(avp *gi.Viewport2D, curIc gi.IconName, title, prompt string, css ki.Props, recv ki.Ki, selFun ki.RecvFunc, dlgFun ki.RecvFunc) *gi.Dialog {
	if css == nil {
		css = ki.Props{
			"icon": ki.Props{
				"width":  units.NewValue(2, units.Em),
				"height": units.NewValue(2, units.Em),
			},
		}
	}
	dlg := SliceViewSelectDialog(avp, &gi.CurIconList, curIc, title, prompt, -1, css, recv, selFun, dlgFun, IconChooserStyleFunc)
	return dlg
}

func IconChooserStyleFunc(sv *SliceView, slice interface{}, widg gi.Node2D, row int, vv ValueView) {
	ic, ok := slice.([]gi.IconName)
	if ok {
		widg.(*gi.Action).SetText(string(ic[row]))
		widg.SetProp("max-width", -1)
	}
}

// ColorViewDialog for editing a color using a ColorView -- optionally
// connects to given signal receiving object and function for dialog signals
// (nil to ignore)
func ColorViewDialog(avp *gi.Viewport2D, clr *gi.Color, tmpSave ValueView, title, prompt string, css ki.Props, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	dlg := gi.NewStdDialog("color-view", title, prompt, true, true, css)

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

// FileViewDialog is for selecting / manipulating files -- ext is one or more
// (comma separated) extensions -- files with those will be highighted
// (include the . at the start of the extension).  recv and fun connect to the
// dialog signal: if signal value is gi.DialogAccepted use FileViewDialogValue
// to get the resulting selected file.  The optional filterFunc can filter
// files shown in the view -- e.g., FileViewDirOnlyFilter (for only showing
// directories) and FileViewExtOnlyFilter (for only showing directories).
func FileViewDialog(avp *gi.Viewport2D, path, file, ext string, title, prompt string, css ki.Props, filterFunc FileViewFilterFunc, recv ki.Ki, fun ki.RecvFunc) *gi.Dialog {
	dlg := gi.NewStdDialog("file-view", title, prompt, true, true, css)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nspc := frame.InsertNewChild(gi.KiT_Space, prIdx+1, "view-space").(*gi.Space)
	nspc.SetFixedHeight(gi.StdDialogVSpaceUnits)

	fv := frame.InsertNewChild(KiT_FileView, prIdx+2, "file-view").(*FileView)
	fv.SetStretchMaxHeight()
	fv.SetStretchMaxWidth()
	fv.FilterFunc = filterFunc
	fv.SetPathFile(path, file, ext)

	fv.FileSig.Connect(dlg.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(FileViewDoubleClicked) {
			ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
			ddlg.Accept()
		}
	})

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
	fvk, ok := frame.ChildByName("file-view", 0)
	if ok {
		fv := fvk.(*FileView)
		return fv.SelectedFile()
	}
	return ""
}
