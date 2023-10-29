// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"image/color"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/icons"
)

/*
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

	// if true all fields will be ReadOnly
	ReadOnly bool

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
*/

// TextEditorDialog adds to the given dialog a display of multi-line text in a
// non-editable TextView -- user can copy contents to clipboard etc.
// there is no input from the user.
func TextEditorDialog(dlg *gi.Dialog, text []byte, filename gi.FileName, lineNumbers bool) *gi.Dialog {
	frame := dlg.Scene
	prIdx := dlg.PromptWidgetIdx()

	tb := texteditor.NewBuf()
	tb.Filename = filename
	tb.Opts.LineNos = lineNumbers
	tb.Stat() // update markup

	tlv := frame.InsertNewChild(gi.LayoutType, prIdx+1, "text-lay").(*gi.Layout)
	tlv.Style(func(s *styles.Style) {
		s.Width.Ch(80)
		s.Height.Em(40)
		s.SetStretchMax()
	})
	tv := texteditor.NewEditor(tlv, "text-editor")
	// tv.Scene = dlg.Embed(gi.TypeScene).(*gi.Scene)
	tv.SetState(true, states.ReadOnly)
	tv.SetBuf(tb)
	tv.Style(func(s *styles.Style) {
		s.Font.Family = string(gi.Prefs.MonoFont)
	})

	tb.SetText(text) // triggers remarkup

	bbox := dlg.ConfigButtonBox()
	gi.NewButton(bbox, "copy-to-clip").SetText("Copy To Clipboard").SetIcon(icons.ContentCopy).
		OnClick(func(e events.Event) {
			dlg.Stage.Scene.EventMgr.ClipBoard().Write(mimedata.NewTextBytes(text))
		})
	return dlg
}

// TextEditorDialogTextEditor returns the text view from a TextViewDialog
func TextEditorDialogTextEditor(dlg *gi.Dialog) *texteditor.Editor {
	frame := dlg.Stage.Scene
	tlv := frame.ChildByName("text-lay", 2)
	tv := tlv.ChildByName("text-editor", 0)
	return tv.(*texteditor.Editor)
}

// StructViewDialog adds to the given dialog a display for editing fields of
// a structure using a StructView.
//
//gopy:interface=handle
func StructViewDialog(dlg *gi.Dialog, stru any, viewPath string, tmpSave Value) *gi.Dialog {
	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(StructViewType, prIdx+1, "struct-view").(*StructView)
	if dlg.RdOnly {
		sv.SetState(true, states.ReadOnly)
	}
	sv.ViewPath = viewPath
	sv.TmpSave = tmpSave
	sv.SetStruct(stru)
	return dlg
}

// MapViewDialog adds to the given dialog a display for editing elements
// of a map using a MapView.
//
//gopy:interface=handle
func MapViewDialog(dlg *gi.Dialog, mp any, viewPath string, tmpSave Value) *gi.Dialog {
	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()
	sv := frame.InsertNewChild(MapViewType, prIdx+1, "map-view").(*MapView)
	if dlg.RdOnly {
		sv.SetState(true, states.ReadOnly)
	}
	sv.ViewPath = viewPath
	sv.TmpSave = tmpSave
	sv.SetMap(mp)
	return dlg
}

// SliceViewDialog adds to the given dialog a display for editing elements of a slice using a SliceView.
// It also takes an optional styling function for styling elements of the slice.
//
//gopy:interface=handle
func SliceViewDialog(dlg *gi.Dialog, slice any, viewPath string, tmpSave Value, noAdd bool, noDelete bool, styleFunc ...SliceViewStyleFunc) *gi.Dialog {
	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(SliceViewType, prIdx+1, "slice-view").(*SliceView)
	if dlg.RdOnly {
		sv.SetState(true, states.ReadOnly)
	}
	if len(styleFunc) > 0 {
		sv.StyleFunc = styleFunc[0]
	}
	sv.SetFlag(noAdd, SliceViewNoAdd)
	sv.SetFlag(noDelete, SliceViewNoDelete)
	sv.ViewPath = viewPath
	sv.TmpSave = tmpSave
	sv.SetSlice(slice)
	return dlg
}

// SliceViewSelectDialog adds to the given dialog a display for selecting one row from the given
// slice using a SliceView. It also takes an optional styling function for styling elements of the slice.
//
//gopy:interface=handle
func SliceViewSelectDialog(dlg *gi.Dialog, slice, curVal any, viewPath string, styleFunc ...SliceViewStyleFunc) *gi.Dialog {
	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(SliceViewType, prIdx+1, "slice-view").(*SliceView)
	sv.SetState(true, states.ReadOnly)
	if len(styleFunc) > 0 {
		sv.StyleFunc = styleFunc[0]
	}
	sv.SelVal = curVal
	sv.ViewPath = viewPath
	sv.SetSlice(slice)
	sv.OnSelect(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
	})
	sv.OnDoubleClick(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
		dlg.AcceptDialog()
	})
	return dlg
}

// TableViewDialog adds to the given dialog a display for editing fields of a slice-of-structs using a
// TableView. It also takes an optional styling function for styling elements of the table.
//
//gopy:interface=handle
func TableViewDialog(dlg *gi.Dialog, slcOfStru any, viewPath string, tmpSave Value, noAdd bool, noDelete bool, styleFunc ...TableViewStyleFunc) *gi.Dialog {
	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(TableViewType, prIdx+1, "tableview").(*TableView)
	if len(styleFunc) > 0 {
		sv.StyleFunc = styleFunc[0]
	}
	sv.SetFlag(noAdd, SliceViewNoAdd)
	sv.SetFlag(noDelete, SliceViewNoDelete)
	sv.ViewPath = viewPath
	sv.TmpSave = tmpSave
	if dlg.RdOnly {
		sv.SetState(true, states.ReadOnly)
	}
	sv.SetSlice(slcOfStru)
	return dlg
}

// TableViewSelectDialog adds to the given dialog a display for selecting a row from a slice-of-structs using a
// TableView. It also takes an optional styling function for styling elements of the table.
//
//gopy:interface=handle
func TableViewSelectDialog(dlg *gi.Dialog, slcOfStru any, initRow int, viewPath string, styleFunc ...TableViewStyleFunc) *gi.Dialog {
	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(TableViewType, prIdx+1, "tableview").(*TableView)
	sv.SetState(true, states.ReadOnly)
	if len(styleFunc) > 0 {
		sv.StyleFunc = styleFunc[0]
	}
	sv.SelectedIdx = initRow
	sv.ViewPath = viewPath
	sv.SetSlice(slcOfStru)
	sv.OnSelect(func(e events.Event) {
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

// FontChooserDialog adds to the given dialog a display for choosing a font.
func FontChooserDialog(dlg *gi.Dialog, viewPath string) *gi.Dialog {
	wb := dlg.Stage.CtxWidget.AsWidget()
	FontChooserSizeDots = int(wb.Styles.UnContext.ToDots(float32(FontChooserSize), units.UnitPt))
	paint.FontLibrary.OpenAllFonts(FontChooserSizeDots)
	fi := paint.FontLibrary.FontInfo
	return TableViewSelectDialog(dlg, &fi, -1, viewPath,
		func(w gi.Widget, s *styles.Style, row, col int) {
			if col != 4 {
				return
			}
			s.Font.Family = fi[row].Name
			s.Font.Stretch = fi[row].Stretch
			s.Font.Weight = fi[row].Weight
			s.Font.Style = fi[row].Style
			s.Font.Size.Pt(float32(FontChooserSize))
		})
}

// IconChooserDialog adds to the given dialog a display for choosing an icon.
func IconChooserDialog(dlg *gi.Dialog, curIc icons.Icon, viewPath string) *gi.Dialog {
	ics := icons.All()
	return SliceViewSelectDialog(dlg, &ics, curIc, viewPath,
		func(w gi.Widget, s *styles.Style, row int) {
			w.(*gi.Button).SetText(string(ics[row]))
			s.SetStretchMaxWidth()
		})
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

	var fv *FileView
	// we need to wrap the function to ensure the data has the selected file
	f := func(dlg *gi.Dialog) {
		dlg.Data = fv.SelectedFile()
		fun(dlg)
	}
	dlg := gi.NewStdDialog(ctx, gopts, f)
	dlg.Stage.Scene.SetName("file-view") // use a consistent name for consistent sizing / placement
	dlg.NewWindow(true)

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	fv = frame.InsertNewChild(FileViewType, prIdx+1, "file-view").(*FileView)
	fv.FilterFunc = filterFunc
	fv.SetFilename(filename, ext)
	fv.OnSelect(func(e events.Event) {
		dlg.Data = fv.SelectedFile()
	})
	fv.OnDoubleClick(func(e events.Event) {
		dlg.Data = fv.SelectedFile()
		dlg.AcceptDialog()
	})
	return dlg
}

// ArgViewDialog for editing args for a method call in the MethodView system
func ArgViewDialog(ctx gi.Widget, opts DlgOpts, args []Value, fun func(dlg *gi.Dialog)) *gi.Dialog {
	dlg := gi.NewStdDialog(ctx, opts.ToGiOpts(), fun)

	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	sv := frame.InsertNewChild(ArgViewType, prIdx+1, "arg-view").(*ArgView)
	sv.SetState(false, states.ReadOnly)
	sv.SetArgs(args)

	return dlg
}
