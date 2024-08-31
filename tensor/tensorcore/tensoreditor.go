// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tensorcore provides GUI Cogent Core widgets for tensor types.
package tensorcore

import (
	"fmt"
	"image"
	"strconv"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tree"
)

// TensorEditor provides a GUI widget for representing [tensor.Tensor] values.
type TensorEditor struct {
	core.ListBase

	// the tensor that we're a view of
	Tensor tensor.Tensor `set:"-"`

	// overall layout options for tensor display
	Layout TensorLayout `set:"-"`

	// number of columns in table (as of last update)
	NCols int `edit:"-"`

	// headerWidths has number of characters in each header, per visfields
	headerWidths []int `copier:"-" display:"-" json:"-" xml:"-"`

	// colMaxWidths records maximum width in chars of string type fields
	colMaxWidths []int `set:"-" copier:"-" json:"-" xml:"-"`

	//	blank values for out-of-range rows
	BlankString string
	BlankFloat  float64
}

// check for interface impl
var _ core.Lister = (*TensorEditor)(nil)

func (tb *TensorEditor) Init() {
	tb.ListBase.Init()
	tb.Makers.Normal[0] = func(p *tree.Plan) { // TODO: reduce redundancy with ListBase Maker
		svi := tb.This.(core.Lister)
		svi.UpdateSliceSize()

		scrollTo := -1
		if tb.InitSelectedIndex >= 0 {
			tb.SelectedIndex = tb.InitSelectedIndex
			tb.InitSelectedIndex = -1
			scrollTo = tb.SelectedIndex
		}
		if scrollTo >= 0 {
			tb.ScrollToIndex(scrollTo)
		}

		tb.UpdateStartIndex()
		tb.UpdateMaxWidths()

		tb.Updater(func() {
			tb.UpdateStartIndex()
			tb.UpdateMaxWidths()
		})

		tb.MakeHeader(p)
		tb.MakeGrid(p, func(p *tree.Plan) {
			for i := 0; i < tb.VisibleRows; i++ {
				svi.MakeRow(p, i)
			}
		})
	}
}

func (tb *TensorEditor) SliceIndex(i int) (si, vi int, invis bool) {
	si = tb.StartIndex + i
	vi = si
	invis = si >= tb.SliceSize
	if !tb.Layout.TopZero {
		vi = (tb.SliceSize - 1) - si
	}
	return
}

// StyleValue performs additional value widget styling
func (tb *TensorEditor) StyleValue(w core.Widget, s *styles.Style, row, col int) {
	hw := float32(tb.headerWidths[col])
	if len(tb.colMaxWidths) > col {
		hw = max(float32(tb.colMaxWidths[col]), hw)
	}
	hv := units.Ch(hw)
	s.Min.X.Value = max(s.Min.X.Value, hv.Convert(s.Min.X.Unit, &s.UnitContext).Value)
	s.SetTextWrap(false)
}

// SetTensor sets the source tensor that we are viewing,
// and then configures the display.
func (tb *TensorEditor) SetTensor(et tensor.Tensor) *TensorEditor {
	if et == nil {
		return nil
	}

	tb.SetSliceBase()
	tb.Tensor = et
	tb.This.(core.Lister).UpdateSliceSize()
	tb.Update()
	return tb
}

func (tb *TensorEditor) UpdateSliceSize() int {
	tb.SliceSize, tb.NCols, _, _ = tensor.Projection2DShape(tb.Tensor.Shape(), tb.Layout.OddRow)
	return tb.SliceSize
}

func (tb *TensorEditor) UpdateMaxWidths() {
	if len(tb.headerWidths) != tb.NCols {
		tb.headerWidths = make([]int, tb.NCols)
		tb.colMaxWidths = make([]int, tb.NCols)
	}
	if tb.SliceSize == 0 {
		return
	}
	_, isstr := tb.Tensor.(*tensor.String)
	for fli := 0; fli < tb.NCols; fli++ {
		tb.colMaxWidths[fli] = 0
		if !isstr {
			continue
		}
		mxw := 0
		// for _, ixi := range tb.Tensor.Indexes {
		// 	if ixi >= 0 {
		// 		sval := stsr.Values[ixi]
		// 		mxw = max(mxw, len(sval))
		// 	}
		// }
		tb.colMaxWidths[fli] = mxw
	}
}

func (tb *TensorEditor) MakeHeader(p *tree.Plan) {
	tree.AddAt(p, "header", func(w *core.Frame) {
		core.ToolbarStyles(w)
		w.FinalStyler(func(s *styles.Style) {
			s.Padding.Zero()
			s.Grow.Set(0, 0)
			s.Gap.Set(units.Em(0.5)) // matches grid default
		})
		w.Maker(func(p *tree.Plan) {
			if tb.ShowIndexes {
				tree.AddAt(p, "_head-index", func(w *core.Text) { // TODO: is not working
					w.SetType(core.TextBodyMedium)
					w.Styler(func(s *styles.Style) {
						s.Align.Self = styles.Center
					})
					w.SetText("Index")
				})
			}
			for fli := 0; fli < tb.NCols; fli++ {
				hdr := tb.ColumnHeader(fli)
				tree.AddAt(p, "head-"+hdr, func(w *core.Button) {
					w.SetType(core.ButtonAction)
					w.Styler(func(s *styles.Style) {
						s.Justify.Content = styles.Start
					})
					w.Updater(func() {
						hdr := tb.ColumnHeader(fli)
						w.SetText(hdr).SetTooltip(hdr)
						tb.headerWidths[fli] = len(hdr)
					})
				})
			}
		})
	})
}

func (tb *TensorEditor) ColumnHeader(col int) string {
	_, cc := tensor.Projection2DCoords(tb.Tensor.Shape(), tb.Layout.OddRow, 0, col)
	sitxt := ""
	for i, ccc := range cc {
		sitxt += fmt.Sprintf("%03d", ccc)
		if i < len(cc)-1 {
			sitxt += ","
		}
	}
	return sitxt
}

// SliceHeader returns the Frame header for slice grid
func (tb *TensorEditor) SliceHeader() *core.Frame {
	return tb.Child(0).(*core.Frame)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (tb *TensorEditor) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 1 + tb.NCols
	idxOff = 1
	if !tb.ShowIndexes {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

func (tb *TensorEditor) MakeRow(p *tree.Plan, i int) {
	svi := tb.This.(core.Lister)
	si, _, invis := svi.SliceIndex(i)
	itxt := strconv.Itoa(i)

	if tb.ShowIndexes {
		tb.MakeGridIndex(p, i, si, itxt, invis)
	}

	_, isstr := tb.Tensor.(*tensor.String)
	for fli := 0; fli < tb.NCols; fli++ {
		valnm := fmt.Sprintf("value-%v.%v", fli, itxt)

		fval := float64(0)
		str := ""
		tree.AddNew(p, valnm, func() core.Value {
			if isstr {
				return core.NewValue(&str, "")
			} else {
				return core.NewValue(&fval, "")
			}
		}, func(w core.Value) {
			wb := w.AsWidget()
			tb.MakeValue(w, i)
			w.AsTree().SetProperty(core.ListColProperty, fli)
			if !tb.IsReadOnly() {
				wb.OnChange(func(e events.Event) {
					_, vi, invis := svi.SliceIndex(i)
					if !invis {
						if isstr {
							tensor.Projection2DSetString(tb.Tensor, tb.Layout.OddRow, vi, fli, str)
						} else {
							tensor.Projection2DSet(tb.Tensor, tb.Layout.OddRow, vi, fli, fval)
						}
					}
					tb.SendChange()
				})
			}
			wb.Updater(func() {
				_, vi, invis := svi.SliceIndex(i)
				if !invis {
					if isstr {
						str = tensor.Projection2DString(tb.Tensor, tb.Layout.OddRow, vi, fli)
						core.Bind(&str, w)
					} else {
						fval = tensor.Projection2DValue(tb.Tensor, tb.Layout.OddRow, vi, fli)
						core.Bind(&fval, w)
					}
				} else {
					if isstr {
						core.Bind(tb.BlankString, w)
					} else {
						core.Bind(tb.BlankFloat, w)
					}
				}
				wb.SetReadOnly(tb.IsReadOnly())
				wb.SetState(invis, states.Invisible)
				if svi.HasStyler() {
					w.Style()
				}
				if invis {
					wb.SetSelected(false)
				}
			})
		})
	}
}

func (tb *TensorEditor) HasStyler() bool { return false }

func (tb *TensorEditor) StyleRow(w core.Widget, idx, fidx int) {}

// RowFirstVisWidget returns the first visible widget for given row (could be
// index or not) -- false if out of range
func (tb *TensorEditor) RowFirstVisWidget(row int) (*core.WidgetBase, bool) {
	if !tb.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, idxOff := tb.RowWidgetNs()
	lg := tb.ListGrid
	w := lg.Children[row*nWidgPerRow].(core.Widget).AsWidget()
	if w.Geom.TotalBBox != (image.Rectangle{}) {
		return w, true
	}
	ridx := nWidgPerRow * row
	for fli := 0; fli < tb.NCols; fli++ {
		w := lg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
		if w.Geom.TotalBBox != (image.Rectangle{}) {
			return w, true
		}
	}
	return nil, false
}

// RowGrabFocus grabs the focus for the first focusable widget in given row --
// returns that element or nil if not successful -- note: grid must have
// already rendered for focus to be grabbed!
func (tb *TensorEditor) RowGrabFocus(row int) *core.WidgetBase {
	if !tb.IsRowInBounds(row) || tb.InFocusGrab { // range check
		return nil
	}
	nWidgPerRow, idxOff := tb.RowWidgetNs()
	ridx := nWidgPerRow * row
	lg := tb.ListGrid
	// first check if we already have focus
	for fli := 0; fli < tb.NCols; fli++ {
		w := lg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
		if w.StateIs(states.Focused) || w.ContainsFocus() {
			return w
		}
	}
	tb.InFocusGrab = true
	defer func() { tb.InFocusGrab = false }()
	for fli := 0; fli < tb.NCols; fli++ {
		w := lg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
		if w.CanFocus() {
			w.SetFocusEvent()
			return w
		}
	}
	return nil
}

//////////////////////////////////////////////////////
// 	Header layout

func (tb *TensorEditor) SizeFinal() {
	tb.ListBase.SizeFinal()
	lg := tb.ListGrid
	sh := tb.SliceHeader()
	sh.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
		sgb := core.AsWidget(lg.Child(i))
		gsz := &sgb.Geom.Size
		if gsz.Actual.Total.X == 0 {
			return tree.Continue
		}
		ksz := &cwb.Geom.Size
		ksz.Actual.Total.X = gsz.Actual.Total.X
		ksz.Actual.Content.X = gsz.Actual.Content.X
		ksz.Alloc.Total.X = gsz.Alloc.Total.X
		ksz.Alloc.Content.X = gsz.Alloc.Content.X
		return tree.Continue
	})
	gsz := &lg.Geom.Size
	ksz := &sh.Geom.Size
	if gsz.Actual.Total.X > 0 {
		ksz.Actual.Total.X = gsz.Actual.Total.X
		ksz.Actual.Content.X = gsz.Actual.Content.X
		ksz.Alloc.Total.X = gsz.Alloc.Total.X
		ksz.Alloc.Content.X = gsz.Alloc.Content.X
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// SaveTSV writes a tensor to a tab-separated-values (TSV) file.
// Outer-most dims are rows in the file, and inner-most is column --
// Reading just grabs all values and doesn't care about shape.
func (tb *TensorEditor) SaveCSV(filename core.Filename) error { //types:add
	return tensor.SaveCSV(tb.Tensor, filename, table.Tab.Rune())
}

// OpenTSV reads a tensor from a tab-separated-values (TSV) file.
// using the Go standard encoding/csv reader conforming
// to the official CSV standard.
// Reads all values and assigns as many as fit.
func (tb *TensorEditor) OpenCSV(filename core.Filename) error { //types:add
	return tensor.OpenCSV(tb.Tensor, filename, table.Tab.Rune())
}

func (tb *TensorEditor) MakeToolbar(p *tree.Plan) {
	if tb.Tensor == nil {
		return
	}
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.OpenCSV).SetIcon(icons.Open)
		w.SetAfterFunc(func() { tb.Update() })
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.SaveCSV).SetIcon(icons.Save)
		w.SetAfterFunc(func() { tb.Update() })
	})
}

func (tb *TensorEditor) MimeDataType() string {
	return fileinfo.DataCsv
}

// CopySelectToMime copies selected rows to mime data
func (tb *TensorEditor) CopySelectToMime() mimedata.Mimes {
	nitms := len(tb.SelectedIndexes)
	if nitms == 0 {
		return nil
	}
	// idx := tb.SelectedIndexesList(false) // ascending
	// var b bytes.Buffer
	// ix.WriteCSV(&b, table.Tab, table.Headers)
	// md := mimedata.NewTextBytes(b.Bytes())
	// md[0].Type = fileinfo.DataCsv
	// return md
	return nil
}

// FromMimeData returns records from csv of mime data
func (tb *TensorEditor) FromMimeData(md mimedata.Mimes) [][]string {
	var recs [][]string
	for _, d := range md {
		if d.Type == fileinfo.DataCsv {
			// b := bytes.NewBuffer(d.Data)
			// cr := csv.NewReader(b)
			// cr.Comma = table.Tab.Rune()
			// rec, err := cr.ReadAll()
			// if err != nil || len(rec) == 0 {
			// 	log.Printf("Error reading CSV from clipboard: %s\n", err)
			// 	return nil
			// }
			// recs = append(recs, rec...)
		}
	}
	return recs
}

// PasteAssign assigns mime data (only the first one!) to this idx
func (tb *TensorEditor) PasteAssign(md mimedata.Mimes, idx int) {
	// recs := tb.FromMimeData(md)
	// if len(recs) == 0 {
	// 	return
	// }
	// tb.Tensor.ReadCSVRow(recs[1], tb.Tensor.Indexes[idx])
	// tb.UpdateChange()
}

// PasteAtIndex inserts object(s) from mime data at (before) given slice index
// adds to end of table
func (tb *TensorEditor) PasteAtIndex(md mimedata.Mimes, idx int) {
	// recs := tb.FromMimeData(md)
	// nr := len(recs) - 1
	// if nr <= 0 {
	// 	return
	// }
	// tb.Tensor.InsertRows(idx, nr)
	// for ri := 0; ri < nr; ri++ {
	// 	rec := recs[1+ri]
	// 	rw := tb.Tensor.Indexes[idx+ri]
	// 	tb.Tensor.ReadCSVRow(rec, rw)
	// }
	// tb.SendChange()
	// tb.SelectIndexEvent(idx, events.SelectOne)
	// tb.Update()
}
