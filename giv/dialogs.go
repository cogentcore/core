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

// TextEditorDialog adds to the given dialog a display of multi-line text in a TextView,
// in which the user can copy contents to clipboard etc.
func TextEditorDialog(dlg *gi.Dialog, text []byte, filename gi.FileName, lineNumbers bool) *gi.Dialog {
	tb := texteditor.NewBuf()
	tb.Filename = filename
	tb.Opts.LineNos = lineNumbers
	tb.Stat() // update markup

	tlv := gi.NewLayout(dlg.Scene, "text-lay")
	tlv.Style(func(s *styles.Style) {
		s.Width.Ch(80)
		s.Height.Em(40)
		s.SetStretchMax()
	})
	tv := texteditor.NewEditor(tlv, "text-editor")
	// tv.Scene = dlg.Embed(gi.TypeScene).(*gi.Scene)
	tv.SetState(dlg.RdOnly, states.ReadOnly)
	tv.SetBuf(tb)
	tv.Style(func(s *styles.Style) {
		s.Font.Family = string(gi.Prefs.MonoFont)
	})

	tb.SetText(text) // triggers remarkup

	bbox := dlg.ConfigButtonBox()
	gi.NewButton(bbox, "copy-to-clip").SetText("Copy To Clipboard").SetIcon(icons.ContentCopy).
		OnClick(func(e events.Event) {
			dlg.Scene.EventMgr.ClipBoard().Write(mimedata.NewTextBytes(text))
		})
	return dlg.FullWindow(true)
}

// TextEditorDialogTextEditor returns the text view from a TextViewDialog
func TextEditorDialogTextEditor(dlg *gi.Dialog) *texteditor.Editor {
	tlv := dlg.Scene.ChildByName("text-lay", 2)
	tv := tlv.ChildByName("text-editor", 0)
	return tv.(*texteditor.Editor)
}

// StructViewDialog adds to the given dialog a display for editing fields of
// a structure using a StructView.
//
//gopy:interface=handle
func StructViewDialog(dlg *gi.Dialog, stru any, tmpSave Value) *gi.Dialog {
	sv := NewStructView(dlg.Scene, "struct-view")
	sv.SetState(dlg.RdOnly, states.ReadOnly)
	sv.ViewPath = dlg.VwPath
	sv.TmpSave = tmpSave
	sv.SetStruct(stru)
	return dlg.FullWindow(true)
}

// MapViewDialog adds to the given dialog a display for editing elements
// of a map using a MapView.
//
//gopy:interface=handle
func MapViewDialog(dlg *gi.Dialog, mp any, tmpSave Value) *gi.Dialog {
	sv := NewMapView(dlg.Scene, "map-view")
	sv.SetState(dlg.RdOnly, states.ReadOnly)
	sv.ViewPath = dlg.VwPath
	sv.TmpSave = tmpSave
	sv.SetMap(mp)
	return dlg.FullWindow(true)
}

// SliceViewDialog adds to the given dialog a display for editing elements of a slice using a SliceView.
// It also takes an optional styling function for styling elements of the slice.
//
//gopy:interface=handle
func SliceViewDialog(dlg *gi.Dialog, slice any, tmpSave Value, noAdd bool, noDelete bool, styleFunc ...SliceViewStyleFunc) *gi.Dialog {
	sv := NewSliceView(dlg.Scene, "slice-view")
	sv.SetState(dlg.RdOnly, states.ReadOnly)
	if len(styleFunc) > 0 {
		sv.StyleFunc = styleFunc[0]
	}
	sv.SetFlag(noAdd, SliceViewNoAdd)
	sv.SetFlag(noDelete, SliceViewNoDelete)
	sv.ViewPath = dlg.VwPath
	sv.TmpSave = tmpSave
	sv.SetSlice(slice)
	return dlg.FullWindow(true)
}

// SliceViewSelectDialog adds to the given dialog a display for selecting one row from the given
// slice using a SliceView. It also takes an optional styling function for styling elements of the slice.
//
//gopy:interface=handle
func SliceViewSelectDialog(dlg *gi.Dialog, slice, curVal any, styleFunc ...SliceViewStyleFunc) *gi.Dialog {
	sv := NewSliceView(dlg.Scene, "slice-view")
	sv.SetState(true, states.ReadOnly)
	if len(styleFunc) > 0 {
		sv.StyleFunc = styleFunc[0]
	}
	sv.SelVal = curVal
	sv.ViewPath = dlg.VwPath
	sv.SetSlice(slice)
	sv.OnSelect(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
	})
	sv.OnDoubleClick(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
		dlg.AcceptDialog()
	})
	return dlg.FullWindow(true)
}

// TableViewDialog adds to the given dialog a display for editing fields of a slice-of-structs using a
// TableView. It also takes an optional styling function for styling elements of the table.
//
//gopy:interface=handle
func TableViewDialog(dlg *gi.Dialog, slcOfStru any, tmpSave Value, noAdd bool, noDelete bool, styleFunc ...TableViewStyleFunc) *gi.Dialog {
	sv := NewTableView(dlg.Scene, "tableview")
	if len(styleFunc) > 0 {
		sv.StyleFunc = styleFunc[0]
	}
	sv.SetFlag(noAdd, SliceViewNoAdd)
	sv.SetFlag(noDelete, SliceViewNoDelete)
	sv.ViewPath = dlg.VwPath
	sv.TmpSave = tmpSave
	sv.SetState(dlg.RdOnly, states.ReadOnly)
	sv.SetSlice(slcOfStru)
	return dlg.FullWindow(true)
}

// TableViewSelectDialog adds to the given dialog a display for selecting a row from a slice-of-structs using a
// TableView. It also takes an optional styling function for styling elements of the table.
//
//gopy:interface=handle
func TableViewSelectDialog(dlg *gi.Dialog, slcOfStru any, initRow int, styleFunc ...TableViewStyleFunc) *gi.Dialog {
	sv := NewTableView(dlg.Scene, "tableview")
	sv.SetState(true, states.ReadOnly)
	if len(styleFunc) > 0 {
		sv.StyleFunc = styleFunc[0]
	}
	sv.SelectedIdx = initRow
	sv.ViewPath = dlg.VwPath
	sv.SetSlice(slcOfStru)
	sv.OnSelect(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
	})
	sv.OnDoubleClick(func(e events.Event) {
		dlg.Data = sv.SelectedIdx
		dlg.AcceptDialog()
	})
	return dlg.FullWindow(true)
}

// show fonts in a bigger size so you can actually see the differences
var FontChooserSize = 18
var FontChooserSizeDots = 18

// FontChooserDialog adds to the given dialog a display for choosing a font.
func FontChooserDialog(dlg *gi.Dialog) *gi.Dialog {
	wb := dlg.Stage.CtxWidget.AsWidget()
	FontChooserSizeDots = int(wb.Styles.UnContext.ToDots(float32(FontChooserSize), units.UnitPt))
	paint.FontLibrary.OpenAllFonts(FontChooserSizeDots)
	fi := paint.FontLibrary.FontInfo
	return TableViewSelectDialog(dlg, &fi, -1,
		func(w gi.Widget, s *styles.Style, row, col int) {
			if col != 4 {
				return
			}
			s.Font.Family = fi[row].Name
			s.Font.Stretch = fi[row].Stretch
			s.Font.Weight = fi[row].Weight
			s.Font.Style = fi[row].Style
			s.Font.Size.Pt(float32(FontChooserSize))
		}).FullWindow(true)
}

// IconChooserDialog adds to the given dialog a display for choosing an icon.
func IconChooserDialog(dlg *gi.Dialog, curIc icons.Icon) *gi.Dialog {
	ics := icons.All()
	return SliceViewSelectDialog(dlg, &ics, curIc,
		func(w gi.Widget, s *styles.Style, row int) {
			w.(*gi.Button).SetText(string(ics[row]))
			s.SetStretchMaxWidth()
		}).FullWindow(true)
}

// ColorViewDialog adds to the given dialog a display for editing a color using a ColorView.
func ColorViewDialog(dlg *gi.Dialog, clr color.RGBA, tmpSave Value) *gi.Dialog {
	dlg.Stage.ClickOff = true

	sv := NewColorView(dlg.Scene, "color-view")
	sv.SetState(dlg.RdOnly, states.ReadOnly)
	sv.ViewPath = dlg.VwPath
	sv.TmpSave = tmpSave
	sv.SetColor(clr)
	sv.OnChange(func(e events.Event) {
		dlg.Data = sv.Color
	})
	return dlg.FullWindow(true)
}

// FileViewDialog adds to the given dialog a display for selecting / manipulating files.
// Ext is one or more (comma separated) extensions; files with those will be highlighted
// (include the . at the start of the extension). The optional filterFunc can filter
// files shown in the view -- e.g., FileViewDirOnlyFilter (for only showing
// directories) and FileViewExtOnlyFilter (for only showing files with certain extensions).
func FileViewDialog(dlg *gi.Dialog, filename, ext string, filterFunc ...FileViewFilterFunc) *gi.Dialog {
	dlg.Data = filename
	dlg.Scene.SetName("file-view") // use a consistent name for consistent sizing / placement
	dlg.NewWindow(true)

	fv := NewFileView(dlg.Scene, "file-view")
	if len(filterFunc) > 0 {
		fv.FilterFunc = filterFunc[0]
	}
	fv.SetFilename(filename, ext)
	fv.OnSelect(func(e events.Event) {
		dlg.Data = fv.SelectedFile()
	})
	fv.OnDoubleClick(func(e events.Event) {
		dlg.Data = fv.SelectedFile()
		dlg.AcceptDialog()
	})

	dlg.Cancel().Ok("Open")
	return dlg.FullWindow(true)
}

// ArgViewDialog adds to the given dialog a display for editing args for a method call
// in the FuncButton system.
func ArgViewDialog(dlg *gi.Dialog, args []Value) *gi.Dialog {
	sv := NewArgView(dlg.Scene, "arg-view")
	sv.SetState(dlg.RdOnly, states.ReadOnly)
	sv.SetArgs(args)
	return dlg
}
