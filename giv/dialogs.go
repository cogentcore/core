// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"image/color"
	"reflect"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/textview"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/icons"
)

// DlgOpts are the basic dialog options accepted by all giv dialog methods --
// provides a named, optional way to specify these args
type DlgOpts struct {

	// generally should be provided -- used for setting name of dialog and associated window
	Title string

	// optional more detailed description of what is being requested and how it will be used -- is word-wrapped and can contain full html formatting etc.
	Prompt string

	// display the Ok button, in most View dialogs where it otherwise is not shown by default -- these views always apply edits immediately, and typically this obviates the need for Ok and Cancel, but sometimes you're giving users a temporary object to edit, and you want them to indicate if they want to proceed or not.
	Ok bool

	// display the Cancel button, in most View dialogs where it otherwise is not shown by default -- these views always apply edits immediately, and typically this obviates the need for Ok and Cancel, but sometimes you're giving users a temporary object to edit, and you want them to indicate if they want to proceed or not.
	Cancel bool

	// if non-nil, this is data that identifies what the dialog is about -- if an existing dialog for such data is already in place, then it is shown instead of making a new one
	Data any

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// if true, user cannot add elements of the slice
	NoAdd bool

	// if true, user cannot delete elements of the slice
	NoDelete bool

	// if true all fields will be inactive
	Inactive bool

	// filename, e.g., for TextView, to get highlighting
	Filename string

	// include line numbers for TextView
	LineNos bool
}

// ToGiOpts converts giv opts to gi opts
func (d *DlgOpts) ToGiOpts() gi.DlgOpts {
	// todo: temporarily enable ok, cancel until click-off etc all working
	return gi.DlgOpts{Title: d.Title, Prompt: d.Prompt, Ok: true, Cancel: true} // d.Ok, Cancel: d.Cancel}
}

// TextViewDialog opens a dialog for displaying multi-line text in a
// non-editable TextView -- user can copy contents to clipboard etc.
// there is no input from the user.
func TextViewDialog(ctx gi.Widget, opts DlgOpts, text []byte, fun func(dlg *gi.Dialog)) *textview.View {
	var dlg *gi.Dialog
	if opts.Data != nil {
		recyc := false
		dlg, recyc = gi.RecycleStdDialog(ctx, opts.ToGiOpts(), opts.Data, fun)
		if recyc {
			return TextViewDialogTextView(dlg)
		}
	} else {
		dlg = gi.NewStdDialog(ctx, opts.ToGiOpts(), fun)
	}

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	tb := textview.NewBuf()
	tb.Filename = gi.FileName(opts.Filename)
	tb.Opts.LineNos = opts.LineNos
	tb.Stat() // update markup

	tlv := frame.InsertNewChild(gi.LayoutType, prIdx+1, "text-lay").(*gi.Layout)
	tlv.Style(func(s *styles.Style) {
		s.Width.SetCh(80)
		s.Height.SetEm(40)
		s.SetStretchMax()
	})
	tv := textview.NewView(tlv, "text-view")
	// tv.Scene = dlg.Embed(gi.TypeScene).(*gi.Scene)
	tv.SetState(true, states.Disabled)
	tv.SetBuf(tb)
	tv.Style(func(s *styles.Style) {
		s.Font.Family = string(gi.Prefs.MonoFont)
	})

	tb.SetText(text) // triggers remarkup

	bbox := dlg.ConfigButtonBox()
	gi.NewButton(bbox, "copy-to-clip").
		SetText("Copy To Clipboard").
		SetIcon(icons.ContentCopy).OnClick(func(e events.Event) {
		dlg.Stage.Scene.EventMgr.ClipBoard().Write(mimedata.NewTextBytes(text))
	})
	return tv
}

// TextViewDialogTextView returns the text view from a TextViewDialog
func TextViewDialogTextView(dlg *gi.Dialog) *textview.View {
	frame := dlg.Stage.Scene
	tlv := frame.ChildByName("text-lay", 2)
	tv := tlv.ChildByName("text-view", 0)
	return tv.(*textview.View)
}

// StructViewDialog is for editing fields of a structure using a StructView.
// Optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore)
// gopy:interface=handle
func StructViewDialog(ctx gi.Widget, opts DlgOpts, stru any, fun func(dlg *gi.Dialog)) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(ctx, opts.ToGiOpts(), stru, fun)
	if recyc {
		return dlg
	}

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(StructViewType, prIdx+1, "struct-view").(*StructView)
	// if opts.Inactive {
	// 	sv.SetState(true, states.Disabled)
	// }
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetStruct(stru)
	return dlg
}

// MapViewDialog is for editing elements of a map using a MapView.
// Optionally connects to given signal receiving object and function for dialog signals
// (nil to ignore)
// gopy:interface=handle
func MapViewDialog(ctx gi.Widget, opts DlgOpts, mp any, fun func(dlg *gi.Dialog)) *gi.Dialog {
	// note: map is not directly comparable, so we have to use the pointer here..
	mptr := reflect.ValueOf(mp).Pointer()
	dlg, recyc := gi.RecycleStdDialog(ctx, opts.ToGiOpts(), mptr, fun)
	if recyc {
		return dlg
	}
	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()
	sv := frame.InsertNewChild(MapViewType, prIdx+1, "map-view").(*MapView)
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetMap(mp)
	return dlg
}

// SliceViewDialog for editing elements of a slice using a SliceView --
// optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore).    Also has an optional styling
// function for styling elements of the table.
// gopy:interface=handle
func SliceViewDialog(ctx gi.Widget, opts DlgOpts, slice any, styleFunc SliceViewStyleFunc, fun func(dlg *gi.Dialog)) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(ctx, opts.ToGiOpts(), slice, fun)
	if recyc {
		return dlg
	}

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(SliceViewType, prIdx+1, "slice-view").(*SliceView)
	sv.SetState(false, states.Disabled)
	sv.StyleFunc = styleFunc
	sv.NoAdd = opts.NoAdd
	sv.NoDelete = opts.NoDelete
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetSlice(slice)
	return dlg
}

// SliceViewDialogNoStyle for editing elements of a slice using a SliceView --
// optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore).  This version does not have the style function.
// gopy:interface=handle
func SliceViewDialogNoStyle(ctx gi.Widget, opts DlgOpts, slice any, fun func(dlg *gi.Dialog)) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(ctx, opts.ToGiOpts(), slice, fun)
	if recyc {
		return dlg
	}

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(SliceViewType, prIdx+1, "slice-view").(*SliceView)
	sv.SetState(false, states.Disabled)
	sv.NoAdd = opts.NoAdd
	sv.NoDelete = opts.NoDelete
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetSlice(slice)
	return dlg
}

// SliceViewSelectDialog for selecting one row from given slice -- connections
// functions available for both the widget signal reporting selection events,
// and the overall dialog signal.  Also has an optional styling function for
// styling elements of the table.
// gopy:interface=handle
func SliceViewSelectDialog(ctx gi.Widget, opts DlgOpts, slice, curVal any, styleFunc SliceViewStyleFunc, fun func(dlg *gi.Dialog)) *gi.Dialog {
	// if opts.CSS == nil {
	// 	opts.CSS = ki.Props{
	// 		"textfield": ki.Props{
	// 			":inactive": ki.Props{
	// 				"background-color": &gi.Prefs.Colors.Control,
	// 			},
	// 		},
	// 	}
	// }
	dlg, recyc := gi.RecycleStdDialog(ctx, opts.ToGiOpts(), slice, fun)
	if recyc {
		return dlg
	}

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(SliceViewType, prIdx+1, "slice-view").(*SliceView)
	sv.SetState(true, states.Disabled)
	sv.StyleFunc = styleFunc
	sv.SelVal = curVal
	sv.ViewPath = opts.ViewPath
	sv.SetSlice(slice)
	sv.OnChange(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
	})
	sv.OnDoubleClick(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
		dlg.AcceptDialog()
	})
	return dlg
}

// TableViewDialog is for editing fields of a slice-of-struct using a
// TableView -- optionally connects to given signal receiving object and
// function for dialog signals (nil to ignore).  Also has an optional styling
// function for styling elements of the table.
// gopy:interface=handle
func TableViewDialog(ctx gi.Widget, opts DlgOpts, slcOfStru any, styleFunc TableViewStyleFunc, fun func(dlg *gi.Dialog)) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(ctx, opts.ToGiOpts(), slcOfStru, fun)
	if recyc {
		return dlg
	}

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(TableViewType, prIdx+1, "tableview").(*TableView)
	sv.SetState(false, states.Disabled)
	sv.StyleFunc = styleFunc
	sv.NoAdd = opts.NoAdd
	sv.NoDelete = opts.NoDelete
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	if opts.Inactive {
		sv.SetState(true, states.Disabled)
	}
	sv.SetSlice(slcOfStru)
	return dlg
}

// TableViewSelectDialog is for selecting a row from a slice-of-struct using a
// TableView -- optionally connects to given signal receiving object and
// functions for signals (nil to ignore): selFunc for the widget signal
// reporting selection events, and dlgFunc for the overall dialog signals.
// Also has an optional styling function for styling elements of the table.
// gopy:interface=handle
func TableViewSelectDialog(ctx gi.Widget, opts DlgOpts, slcOfStru any, initRow int, styleFunc TableViewStyleFunc, fun func(dlg *gi.Dialog)) *gi.Dialog {
	// if opts.CSS == nil {
	// 	opts.CSS = ki.Props{
	// 		"textfield": ki.Props{
	// 			":inactive": ki.Props{
	// 				"background-color": &gi.Prefs.Colors.Control,
	// 			},
	// 		},
	// 	}
	// }
	dlg, recyc := gi.RecycleStdDialog(ctx, opts.ToGiOpts(), slcOfStru, fun)
	if recyc {
		return dlg
	}

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(TableViewType, prIdx+1, "tableview").(*TableView)
	sv.SetState(true, states.Disabled)
	sv.StyleFunc = styleFunc
	sv.SelectedIdx = initRow
	sv.ViewPath = opts.ViewPath
	sv.SetSlice(slcOfStru)
	sv.OnChange(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
	})
	sv.OnDoubleClick(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
		dlg.AcceptDialog()
	})
	return dlg
}

// show fonts in a bigger size so you can actually see the differences
var FontChooserSize = 18
var FontChooserSizeDots = 18

// FontChooserDialog for choosing a font -- the recv and func signal receivers
// if non-nil are connected to the selection signal for the struct table view,
// so they are updated with that
func FontChooserDialog(ctx gi.Widget, opts DlgOpts, fun func(dlg *gi.Dialog)) *gi.Dialog {
	wb := ctx.AsWidget()
	FontChooserSizeDots = int(wb.Styles.UnContext.ToDots(float32(FontChooserSize), units.UnitPt))
	paint.FontLibrary.OpenAllFonts(FontChooserSizeDots)
	dlg := TableViewSelectDialog(ctx, opts, &paint.FontLibrary.FontInfo, -1, FontInfoStyleFunc, fun)
	return dlg
}

func FontInfoStyleFunc(tv *TableView, slice any, widg gi.Widget, row, col int, vv Value) {
	if col != 4 {
		return
	}
	finf, ok := slice.([]paint.FontInfo)
	if ok {
		widg.SetProp("font-family", (finf)[row].Name)
		widg.SetProp("font-stretch", (finf)[row].Stretch)
		widg.SetProp("font-weight", (finf)[row].Weight)
		widg.SetProp("font-style", (finf)[row].Style)
		widg.SetProp("font-size", units.Pt(float32(FontChooserSize)))
	}
}

// IconChooserDialog for choosing an Icon -- the recv and fun signal receivers
// if non-nil are connected to the selection signal for the slice view, and
// the dialog signal.
func IconChooserDialog(ctx gi.Widget, opts DlgOpts, curIc icons.Icon, fun func(dlg *gi.Dialog)) *gi.Dialog {
	// if opts.CSS == nil {
	// 	opts.CSS = ki.Props{
	// 		"icon": ki.Props{
	// 			"width":  units.Em(2),
	// 			"height": units.Em(2),
	// 		},
	// 	}
	// }
	dlg := SliceViewSelectDialog(ctx, opts, &gi.CurIconList, curIc, IconChooserStyleFunc, fun)
	return dlg
}

func IconChooserStyleFunc(sv *SliceView, slice any, widg gi.Widget, row int, vv Value) {
	ic, ok := slice.([]icons.Icon)
	if ok {
		widg.(*gi.Button).SetText(string(ic[row]))
		widg.SetStretchMaxWidth()
	}
}

// ColorViewDialog for editing a color using a ColorView -- optionally
// connects to given signal receiving object and function for dialog signals
// (nil to ignore)
func ColorViewDialog(ctx gi.Widget, opts DlgOpts, clr color.RGBA, fun func(dlg *gi.Dialog)) *gi.Dialog {
	dlg, recyc := gi.RecycleStdDialog(ctx, opts.ToGiOpts(), clr, fun)
	if recyc {
		return dlg
	}

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()
	dlg.Stage.ClickOff = true

	sv := frame.InsertNewChild(ColorViewType, prIdx+1, "color-view").(*ColorView)
	sv.ViewPath = opts.ViewPath
	sv.TmpSave = opts.TmpSave
	sv.SetColor(clr)
	sv.OnChange(func(e events.Event) {
		dlg.Data = sv.Color
	})
	return dlg
}

// FileViewDialog is for selecting / manipulating files -- ext is one or more
// (comma separated) extensions -- files with those will be highlighted
// (include the . at the start of the extension).  recv and dlgFunc connect to the
// dialog signal: if signal value is gi.DialogAccepted use FileViewDialogValue
// to get the resulting selected file.  The optional filterFunc can filter
// files shown in the view -- e.g., FileViewDirOnlyFilter (for only showing
// directories) and FileViewExtOnlyFilter (for only showing directories).
func FileViewDialog(ctx gi.Widget, opts DlgOpts, filename, ext string, filterFunc FileViewFilterFunc, fun func(dlg *gi.Dialog)) *gi.Dialog {
	gopts := opts.ToGiOpts()
	gopts.Ok = true
	gopts.Cancel = true
	dlg := gi.NewStdDialog(ctx, gopts, fun)
	dlg.Stage.Scene.SetName("file-view") // use a consistent name for consistent sizing / placement

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	fv := frame.InsertNewChild(FileViewType, prIdx+1, "file-view").(*FileView)
	fv.FilterFunc = filterFunc
	fv.SetFilename(filename, ext)
	fv.OnChange(func(e events.Event) {
		dlg.Data = fv.SelectedFile()
	})
	fv.OnDoubleClick(func(e events.Event) {
		dlg.Data = fv.SelectedFile()
		dlg.AcceptDialog()
	})
	return dlg
}

// ArgViewDialog for editing args for a method call in the MethView system
func ArgViewDialog(ctx gi.Widget, opts DlgOpts, args []ArgData, fun func(dlg *gi.Dialog)) *gi.Dialog {
	dlg := gi.NewStdDialog(ctx, opts.ToGiOpts(), fun)

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(ArgViewType, prIdx+1, "arg-view").(*ArgView)
	sv.SetState(false, states.Disabled)
	sv.SetArgs(args)

	return dlg
}
