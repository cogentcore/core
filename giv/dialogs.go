// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
)

// DlgOpts are the basic dialog options accepted by all giv dialog methods --
// provides a named, optional way to specify these args
type DlgOpts struct {
	Title    string      `desc:"generally should be provided -- used for setting name of dialog and associated window"`
	Prompt   string      `desc:"optional more detailed description of what is being requested and how it will be used -- is word-wrapped and can contain full html formatting etc."`
	CSS      ki.Props    `desc:"optional style properties applied to dialog -- can be used to customize any aspect of existing dialogs"`
	TmpSave  ValueView   `desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewPath string      `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
	Ok       bool        `desc:"display the Ok button, in most View dialogs where it otherwise is not shown by default -- these views always apply edits immediately, and typically this obviates the need for Ok and Cancel, but sometimes you're giving users a temporary object to edit, and you want them to indicate if they want to proceed or not."`
	Cancel   bool        `desc:"display the Cancel button, in most View dialogs where it otherwise is not shown by default -- these views always apply edits immediately, and typically this obviates the need for Ok and Cancel, but sometimes you're giving users a temporary object to edit, and you want them to indicate if they want to proceed or not."`
	NoAdd    bool        `desc:"if true, user cannot add elements of the slice"`
	NoDelete bool        `desc:"if true, user cannot delete elements of the slice"`
	Inactive bool        `desc:"if true all fields will be inactive"`
	Data     interface{} `desc:"if non-nil, this is data that identifies what the dialog is about -- if an existing dialog for such data is already in place, then it is shown instead of making a new one"`
	Filename string      `desc:"filename, e.g., for TextView, to get highlighting"`
	LineNos  bool        `desc:"include line numbers for TextView"`
}

// ToGiOpts converts giv opts to gi opts
func (d *DlgOpts) ToGiOpts() gi.DlgOpts {
	return gi.DlgOpts{Title: d.Title, Prompt: d.Prompt, CSS: d.CSS}
}

// TextViewDialog opens a dialog for displaying multi-line text in a
// non-editable TextView -- user can copy contents to clipboard etc.
// there is no input from the user.
func TextViewDialog(avp *gi.Viewport2D, text []byte, opts DlgOpts) *TextView {
	var dlg *gi.Dialog
	if opts.Data != nil {
		recyc := false
		dlg, recyc = gi.RecycleStdDialog(opts.Data, opts.ToGiOpts(), opts.Ok, opts.Cancel)
		if recyc {
			return TextViewDialogTextView(dlg)
		}
	} else {
		dlg = gi.NewStdDialog(opts.ToGiOpts(), opts.Ok, opts.Cancel)
	}

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	tb := &TextBuf{}
	tb.InitName(tb, "text-view-dialog-buf")
	tb.Filename = gi.FileName(opts.Filename)
	tb.Opts.LineNos = opts.LineNos
	tb.Stat() // update markup

	tlv := frame.InsertNewChild(gi.KiT_Layout, prIdx+1, "text-lay").(*gi.Layout)
	tlv.SetProp("width", units.NewCh(80))
	tlv.SetProp("height", units.NewEm(40))
	tlv.SetProp("line-nos", opts.LineNos)
	tlv.SetStretchMax()
	tv := AddNewTextView(tlv, "text-view")
	tv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	tv.SetInactive()
	tv.SetProp("font-family", gi.Prefs.MonoFont)
	tv.SetBuf(tb)

	tb.SetText(text) // triggers remarkup

	bbox, _ := dlg.ButtonBox(frame)
	if bbox == nil {
		bbox = dlg.AddButtonBox(frame)
	}
	cpb := gi.AddNewButton(bbox, "copy-to-clip")
	cpb.SetText("Copy To Clipboard")
	cpb.SetIcon("copy")
	cpb.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
			oswin.TheApp.ClipBoard(ddlg.Win.OSWin).Write(mimedata.NewTextBytes(text))
		}
	})

	dlg.UpdateEndNoSig(true) // going to be shown
	dlg.Open(0, 0, avp, nil)
	return tv
}

// TextViewDialogTextView returns the text view from a TextViewDialog
func TextViewDialogTextView(dlg *gi.Dialog) *TextView {
	frame := dlg.Frame()
	tlv := frame.ChildByName("text-lay", 2)
	tv := tlv.ChildByName("text-view", 0)
	return tv.(*TextView)
}

// StructViewDialog is for editing fields of a structure using a StructView --
// optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore)
// gopy:interface=handle
func StructViewDialog(avp *gi.Viewport2D, stru interface{}, opts DlgOpts, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(stru, opts.ToGiOpts(), opts.Ok, opts.Cancel)
	if recyc {
		return dlg
	}

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	sv := frame.InsertNewChild(KiT_StructView, prIdx+1, "struct-view").(*StructView)
	sv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	if opts.Inactive {
		sv.SetInactive()
	}
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetStruct(stru)
	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, func() {
		MainMenuView(stru, dlg.Win, dlg.Win.MainMenu)
	})
	return dlg
}

// MapViewDialog is for editing elements of a map using a MapView -- optionally
// connects to given signal receiving object and function for dialog signals
// (nil to ignore)
// gopy:interface=handle
func MapViewDialog(avp *gi.Viewport2D, mp interface{}, opts DlgOpts, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	// note: map is not directly comparable, so we have to use the pointer here..
	mptr := reflect.ValueOf(mp).Pointer()
	dlg, recyc := gi.RecycleStdDialog(mptr, opts.ToGiOpts(), opts.Ok, opts.Cancel)
	if recyc {
		return dlg
	}

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	sv := frame.InsertNewChild(KiT_MapView, prIdx+1, "map-view").(*MapView)
	sv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetMap(mp)

	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, func() {
		MainMenuView(mp, dlg.Win, dlg.Win.MainMenu)
	})
	return dlg
}

// SliceViewDialog for editing elements of a slice using a SliceView --
// optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore).    Also has an optional styling
// function for styling elements of the table.
// gopy:interface=handle
func SliceViewDialog(avp *gi.Viewport2D, slice interface{}, opts DlgOpts, styleFunc SliceViewStyleFunc, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(slice, opts.ToGiOpts(), opts.Ok, opts.Cancel)
	if recyc {
		return dlg
	}

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	sv := frame.InsertNewChild(KiT_SliceView, prIdx+1, "slice-view").(*SliceView)
	sv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	sv.SetInactiveState(false)
	sv.StyleFunc = styleFunc
	sv.NoAdd = opts.NoAdd
	sv.NoDelete = opts.NoDelete
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetSlice(slice)

	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, func() {
		MainMenuView(slice, dlg.Win, dlg.Win.MainMenu)
	})
	return dlg
}

// SliceViewDialogNoStyle for editing elements of a slice using a SliceView --
// optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore).  This version does not have the style function.
// gopy:interface=handle
func SliceViewDialogNoStyle(avp *gi.Viewport2D, slice interface{}, opts DlgOpts, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(slice, opts.ToGiOpts(), opts.Ok, opts.Cancel)
	if recyc {
		return dlg
	}

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	sv := frame.InsertNewChild(KiT_SliceView, prIdx+1, "slice-view").(*SliceView)
	sv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	sv.SetInactiveState(false)
	sv.NoAdd = opts.NoAdd
	sv.NoDelete = opts.NoDelete
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetSlice(slice)

	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, func() {
		MainMenuView(slice, dlg.Win, dlg.Win.MainMenu)
	})
	return dlg
}

// SliceViewSelectDialog for selecting one row from given slice -- connections
// functions available for both the widget signal reporting selection events,
// and the overall dialog signal.  Also has an optional styling function for
// styling elements of the table.
// gopy:interface=handle
func SliceViewSelectDialog(avp *gi.Viewport2D, slice, curVal interface{}, opts DlgOpts, styleFunc SliceViewStyleFunc, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	if opts.CSS == nil {
		opts.CSS = ki.Props{
			"textfield": ki.Props{
				":inactive": ki.Props{
					"background-color": &gi.Prefs.Colors.Control,
				},
			},
		}
	}
	dlg, recyc := gi.RecycleStdDialog(slice, opts.ToGiOpts(), gi.AddOk, gi.AddCancel)
	if recyc {
		return dlg
	}

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	sv := frame.InsertNewChild(KiT_SliceView, prIdx+1, "slice-view").(*SliceView)
	sv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	sv.SetInactiveState(true)
	sv.StyleFunc = styleFunc
	sv.SelVal = curVal
	sv.ViewPath = opts.ViewPath
	sv.SetSlice(slice)

	sv.SliceViewSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(SliceViewDoubleClicked) {
			ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
			ddlg.Accept()
		}
	})

	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, nil)
	return dlg
}

// SliceViewSelectDialogValue gets the index of the selected item (-1 if nothing selected)
func SliceViewSelectDialogValue(dlg *gi.Dialog) int {
	frame := dlg.Frame()
	sv := frame.ChildByName("slice-view", 0)
	if sv != nil {
		svv := sv.(*SliceView)
		return svv.SelectedIdx
	}
	return -1
}

// TableViewDialog is for editing fields of a slice-of-struct using a
// TableView -- optionally connects to given signal receiving object and
// function for dialog signals (nil to ignore).  Also has an optional styling
// function for styling elements of the table.
// gopy:interface=handle
func TableViewDialog(avp *gi.Viewport2D, slcOfStru interface{}, opts DlgOpts, styleFunc TableViewStyleFunc, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(slcOfStru, opts.ToGiOpts(), opts.Ok, opts.Cancel)
	if recyc {
		return dlg
	}

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	sv := frame.InsertNewChild(KiT_TableView, prIdx+1, "tableview").(*TableView)
	sv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	sv.SetInactiveState(false)
	sv.StyleFunc = styleFunc
	sv.NoAdd = opts.NoAdd
	sv.NoDelete = opts.NoDelete
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	if opts.Inactive {
		sv.SetInactive()
	}
	sv.SetSlice(slcOfStru)

	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, func() {
		MainMenuView(slcOfStru, dlg.Win, dlg.Win.MainMenu)
	})
	return dlg
}

// TableViewSelectDialog is for selecting a row from a slice-of-struct using a
// TableView -- optionally connects to given signal receiving object and
// functions for signals (nil to ignore): selFunc for the widget signal
// reporting selection events, and dlgFunc for the overall dialog signals.
// Also has an optional styling function for styling elements of the table.
// gopy:interface=handle
func TableViewSelectDialog(avp *gi.Viewport2D, slcOfStru interface{}, opts DlgOpts, initRow int, styleFunc TableViewStyleFunc, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	if opts.CSS == nil {
		opts.CSS = ki.Props{
			"textfield": ki.Props{
				":inactive": ki.Props{
					"background-color": &gi.Prefs.Colors.Control,
				},
			},
		}
	}
	dlg, recyc := gi.RecycleStdDialog(slcOfStru, opts.ToGiOpts(), gi.AddOk, gi.AddCancel)
	if recyc {
		return dlg
	}

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	sv := frame.InsertNewChild(KiT_TableView, prIdx+1, "tableview").(*TableView)
	sv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	sv.SetInactiveState(true)
	sv.StyleFunc = styleFunc
	sv.SelectedIdx = initRow
	sv.ViewPath = opts.ViewPath
	sv.SetSlice(slcOfStru)

	sv.SliceViewSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(SliceViewDoubleClicked) {
			ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
			ddlg.Accept()
		}
	})

	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, nil)
	return dlg
}

// TableViewSelectDialogValue gets the index of the selected item (-1 if nothing selected)
func TableViewSelectDialogValue(dlg *gi.Dialog) int {
	frame := dlg.Frame()
	sv := frame.ChildByName("tableview", 0)
	if sv != nil {
		svv := sv.(*TableView)
		rval := svv.SelectedIdx
		return rval
	}
	return -1
}

// show fonts in a bigger size so you can actually see the differences
var FontChooserSize = 18
var FontChooserSizeDots = 18

// FontChooserDialog for choosing a font -- the recv and func signal receivers
// if non-nil are connected to the selection signal for the struct table view,
// so they are updated with that
func FontChooserDialog(avp *gi.Viewport2D, opts DlgOpts, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	FontChooserSizeDots = int(avp.Sty.UnContext.ToDots(float32(FontChooserSize), units.Pt))
	girl.FontLibrary.OpenAllFonts(FontChooserSizeDots)
	dlg := TableViewSelectDialog(avp, &girl.FontLibrary.FontInfo, opts, -1, FontInfoStyleFunc, recv, dlgFunc)
	return dlg
}

func FontInfoStyleFunc(tv *TableView, slice interface{}, widg gi.Node2D, row, col int, vv ValueView) {
	if col == 4 {
		finf, ok := slice.([]girl.FontInfo)
		if ok {
			widg.SetProp("font-family", (finf)[row].Name)
			widg.SetProp("font-stretch", (finf)[row].Stretch)
			widg.SetProp("font-weight", (finf)[row].Weight)
			widg.SetProp("font-style", (finf)[row].Style)
			widg.SetProp("font-size", units.NewPt(float32(FontChooserSize)))
			widg.AsNode2D().SetFullReRender()
		}
	}
}

// IconChooserDialog for choosing an Icon -- the recv and fun signal receivers
// if non-nil are connected to the selection signal for the slice view, and
// the dialog signal.
func IconChooserDialog(avp *gi.Viewport2D, curIc gi.IconName, opts DlgOpts, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	if opts.CSS == nil {
		opts.CSS = ki.Props{
			"icon": ki.Props{
				"width":  units.NewEm(2),
				"height": units.NewEm(2),
			},
		}
	}
	dlg := SliceViewSelectDialog(avp, &gi.CurIconList, curIc, opts, IconChooserStyleFunc, recv, dlgFunc)
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
func ColorViewDialog(avp *gi.Viewport2D, clr gist.Color, opts DlgOpts, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(clr, opts.ToGiOpts(), gi.AddOk, gi.AddCancel)
	if recyc {
		return dlg
	}

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	sv := frame.InsertNewChild(KiT_ColorView, prIdx+1, "color-view").(*ColorView)
	sv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetColor(clr)

	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, nil)
	return dlg
}

// ColorViewDialogValue gets the color from the dialog
func ColorViewDialogValue(dlg *gi.Dialog) gist.Color {
	frame := dlg.Frame()
	cvvvk := frame.ChildByType(KiT_ColorView, ki.Embeds, 2)
	if cvvvk != nil {
		cvvv := cvvvk.(*ColorView)
		return cvvv.Color
	}
	return gist.Color{}
}

// FileViewDialog is for selecting / manipulating files -- ext is one or more
// (comma separated) extensions -- files with those will be highlighted
// (include the . at the start of the extension).  recv and dlgFunc connect to the
// dialog signal: if signal value is gi.DialogAccepted use FileViewDialogValue
// to get the resulting selected file.  The optional filterFunc can filter
// files shown in the view -- e.g., FileViewDirOnlyFilter (for only showing
// directories) and FileViewExtOnlyFilter (for only showing directories).
func FileViewDialog(avp *gi.Viewport2D, filename, ext string, opts DlgOpts, filterFunc FileViewFilterFunc, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	dlg := gi.NewStdDialog(opts.ToGiOpts(), gi.AddOk, gi.AddCancel)
	dlg.SetName("file-view") // use a consistent name for consistent sizing / placement

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	fv := frame.InsertNewChild(KiT_FileView, prIdx+1, "file-view").(*FileView)
	fv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	fv.FilterFunc = filterFunc
	fv.SetFilename(filename, ext)

	fv.FileSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(FileViewDoubleClicked) {
			ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
			ddlg.Accept()
		}
	})

	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, nil)
	return dlg
}

// FileViewDialogValue gets the full path of selected file
func FileViewDialogValue(dlg *gi.Dialog) string {
	frame := dlg.Frame()
	fvk := frame.ChildByName("file-view", 0)
	if fvk != nil {
		fv := fvk.(*FileView)
		fn := fv.SelectedFile()
		return fn
	}
	return ""
}

// ArgViewDialog for editing args for a method call in the MethView system
func ArgViewDialog(avp *gi.Viewport2D, args []ArgData, opts DlgOpts, recv ki.Ki, dlgFunc ki.RecvFunc) *gi.Dialog {
	dlg := gi.NewStdDialog(opts.ToGiOpts(), gi.AddOk, gi.AddCancel)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	sv := frame.InsertNewChild(KiT_ArgView, prIdx+1, "arg-view").(*ArgView)
	sv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	sv.SetInactiveState(false)
	sv.SetArgs(args)

	if recv != nil && dlgFunc != nil {
		dlg.DialogSig.Connect(recv, dlgFunc)
	}

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, nil)
	return dlg
}
