// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tensorcore provides GUI Cogent Core widgets for tensor types.
package tensorcore

//go:generate core generate

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"image"
	"log"
	"strconv"
	"strings"

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

// Table provides a GUI widget for representing [table.Table] values.
type Table struct {
	core.ListBase

	// the idx view of the table that we're a view of
	Table *table.IndexView `set:"-"`

	// overall display options for tensor display
	TensorDisplay TensorDisplay `set:"-"`

	// per column tensor display params
	ColumnTensorDisplay map[int]*TensorDisplay `set:"-"`

	// per column blank tensor values
	ColumnTensorBlank map[int]*tensor.Float64 `set:"-"`

	// number of columns in table (as of last update)
	NCols int `edit:"-"`

	// current sort index
	SortIndex int

	// whether current sort order is descending
	SortDescending bool

	// headerWidths has number of characters in each header, per visfields
	headerWidths []int `copier:"-" display:"-" json:"-" xml:"-"`

	// colMaxWidths records maximum width in chars of string type fields
	colMaxWidths []int `set:"-" copier:"-" json:"-" xml:"-"`

	//	blank values for out-of-range rows
	BlankString string
	BlankFloat  float64
}

// check for interface impl
var _ core.Lister = (*Table)(nil)

func (tb *Table) Init() {
	tb.ListBase.Init()
	tb.SortIndex = -1
	tb.TensorDisplay.Defaults()
	tb.ColumnTensorDisplay = map[int]*TensorDisplay{}
	tb.ColumnTensorBlank = map[int]*tensor.Float64{}

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

func (tb *Table) SliceIndex(i int) (si, vi int, invis bool) {
	si = tb.StartIndex + i
	vi = -1
	if si < len(tb.Table.Indexes) {
		vi = tb.Table.Indexes[si]
	}
	invis = vi < 0
	return
}

// StyleValue performs additional value widget styling
func (tb *Table) StyleValue(w core.Widget, s *styles.Style, row, col int) {
	hw := float32(tb.headerWidths[col])
	if col == tb.SortIndex {
		hw += 6
	}
	if len(tb.colMaxWidths) > col {
		hw = max(float32(tb.colMaxWidths[col]), hw)
	}
	hv := units.Ch(hw)
	s.Min.X.Value = max(s.Min.X.Value, hv.Convert(s.Min.X.Unit, &s.UnitContext).Value)
	s.SetTextWrap(false)
}

// SetTable sets the source table that we are viewing, using a sequential IndexView
// and then configures the display
func (tb *Table) SetTable(et *table.Table) *Table {
	if et == nil {
		return nil
	}

	tb.SetSliceBase()
	tb.Table = table.NewIndexView(et)
	tb.This.(core.Lister).UpdateSliceSize()
	tb.Update()
	return tb
}

// AsyncUpdateTable updates the display for asynchronous updating from
// other goroutines. Also updates indexview (calling Sequential).
func (tb *Table) AsyncUpdateTable() {
	tb.AsyncLock()
	tb.Table.Sequential()
	tb.ScrollToIndexNoUpdate(tb.SliceSize - 1)
	tb.Update()
	tb.AsyncUnlock()
}

// SetIndexView sets the source IndexView of a table (using a copy so original is not modified)
// and then configures the display
func (tb *Table) SetIndexView(ix *table.IndexView) *Table {
	if ix == nil {
		return tb
	}

	tb.Table = ix.Clone() // always copy

	tb.This.(core.Lister).UpdateSliceSize()
	tb.StartIndex = 0
	tb.VisibleRows = tb.MinRows
	if !tb.IsReadOnly() {
		tb.SelectedIndex = -1
	}
	tb.ResetSelectedIndexes()
	tb.SelectMode = false
	tb.MakeIter = 0
	tb.Update()
	return tb
}

func (tb *Table) UpdateSliceSize() int {
	tb.Table.DeleteInvalid() // table could have changed
	if tb.Table.Len() == 0 {
		tb.Table.Sequential()
	}
	tb.SliceSize = tb.Table.Len()
	tb.NCols = tb.Table.Table.NumColumns()
	return tb.SliceSize
}

func (tb *Table) UpdateMaxWidths() {
	if len(tb.headerWidths) != tb.NCols {
		tb.headerWidths = make([]int, tb.NCols)
		tb.colMaxWidths = make([]int, tb.NCols)
	}

	if tb.SliceSize == 0 {
		return
	}
	for fli := 0; fli < tb.NCols; fli++ {
		tb.colMaxWidths[fli] = 0
		col := tb.Table.Table.Columns[fli]
		stsr, isstr := col.(*tensor.String)

		if !isstr {
			continue
		}
		mxw := 0
		for _, ixi := range tb.Table.Indexes {
			if ixi >= 0 {
				sval := stsr.Values[ixi]
				mxw = max(mxw, len(sval))
			}
		}
		tb.colMaxWidths[fli] = mxw
	}
}

func (tb *Table) MakeHeader(p *tree.Plan) {
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
				field := tb.Table.Table.ColumnNames[fli]
				tree.AddAt(p, "head-"+field, func(w *core.Button) {
					w.SetType(core.ButtonAction)
					w.Styler(func(s *styles.Style) {
						s.Justify.Content = styles.Start
					})
					w.OnClick(func(e events.Event) {
						tb.SortSliceAction(fli)
					})
					w.Updater(func() {
						field := tb.Table.Table.ColumnNames[fli]
						w.SetText(field).SetTooltip(field + " (tap to sort by)")
						tb.headerWidths[fli] = len(field)
						if fli == tb.SortIndex {
							if tb.SortDescending {
								w.SetIndicator(icons.KeyboardArrowDown)
							} else {
								w.SetIndicator(icons.KeyboardArrowUp)
							}
						} else {
							w.SetIndicator(icons.Blank)
						}
					})
				})
			}
		})
	})
}

// SliceHeader returns the Frame header for slice grid
func (tb *Table) SliceHeader() *core.Frame {
	return tb.Child(0).(*core.Frame)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (tb *Table) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 1 + tb.NCols
	idxOff = 1
	if !tb.ShowIndexes {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

func (tb *Table) MakeRow(p *tree.Plan, i int) {
	svi := tb.This.(core.Lister)
	si, _, invis := svi.SliceIndex(i)
	itxt := strconv.Itoa(i)

	if tb.ShowIndexes {
		tb.MakeGridIndex(p, i, si, itxt, invis)
	}

	for fli := 0; fli < tb.NCols; fli++ {
		col := tb.Table.Table.Columns[fli]
		valnm := fmt.Sprintf("value-%v.%v", fli, itxt)

		_, isstr := col.(*tensor.String)
		if col.NumDims() == 1 {
			str := ""
			fval := float64(0)
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
						if si < len(tb.Table.Indexes) {
							if isstr {
								tb.Table.Table.SetStringIndex(fli, tb.Table.Indexes[si], str)
							} else {
								tb.Table.Table.SetFloatIndex(fli, tb.Table.Indexes[si], fval)
							}
						}
						tb.SendChange()
					})
				}
				wb.Updater(func() {
					_, vi, invis := svi.SliceIndex(i)
					if !invis {
						if isstr {
							str = tb.Table.Table.StringIndex(fli, vi)
							core.Bind(&str, w)
						} else {
							fval = tb.Table.Table.FloatIndex(fli, vi)
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
		} else {
			tree.AddAt(p, valnm, func(w *TensorGrid) {
				w.SetReadOnly(tb.IsReadOnly())
				wb := w.AsWidget()
				w.SetProperty(core.ListRowProperty, i)
				w.SetProperty(core.ListColProperty, fli)
				w.Styler(func(s *styles.Style) {
					s.Grow.Set(0, 0)
				})
				wb.Updater(func() {
					si, vi, invis := svi.SliceIndex(i)
					var cell tensor.Tensor
					if invis {
						cell = tb.ColTensorBlank(fli, col)
					} else {
						cell = tb.Table.Table.TensorIndex(fli, vi)
					}
					wb.ValueTitle = tb.ValueTitle + "[" + strconv.Itoa(si) + "]"
					w.SetState(invis, states.Invisible)
					w.SetTensor(cell)
					w.Display = *tb.GetColumnTensorDisplay(fli)
				})
			})
		}
	}
}

// ColTensorBlank returns tensor blanks for given tensor col
func (tb *Table) ColTensorBlank(cidx int, col tensor.Tensor) *tensor.Float64 {
	if ctb, has := tb.ColumnTensorBlank[cidx]; has {
		return ctb
	}
	ctb := tensor.New[float64](col.Shape().Sizes, col.Shape().Names...).(*tensor.Float64)
	tb.ColumnTensorBlank[cidx] = ctb
	return ctb
}

// GetColumnTensorDisplay returns tensor display parameters for this column
// either the overall defaults or the per-column if set
func (tb *Table) GetColumnTensorDisplay(col int) *TensorDisplay {
	if ctd, has := tb.ColumnTensorDisplay[col]; has {
		return ctd
	}
	if tb.Table != nil {
		cl := tb.Table.Table.Columns[col]
		if len(cl.MetaDataMap()) > 0 {
			return tb.SetColumnTensorDisplay(col)
		}
	}
	return &tb.TensorDisplay
}

// SetColumnTensorDisplay sets per-column tensor display params and returns them
// if already set, just returns them
func (tb *Table) SetColumnTensorDisplay(col int) *TensorDisplay {
	if ctd, has := tb.ColumnTensorDisplay[col]; has {
		return ctd
	}
	ctd := &TensorDisplay{}
	*ctd = tb.TensorDisplay
	if tb.Table != nil {
		cl := tb.Table.Table.Columns[col]
		ctd.FromMeta(cl)
	}
	tb.ColumnTensorDisplay[col] = ctd
	return ctd
}

// NewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (tb *Table) NewAt(idx int) {
	tb.NewAtSelect(idx)

	tb.Table.InsertRows(idx, 1)

	tb.SelectIndexEvent(idx, events.SelectOne)
	tb.Update()
	tb.IndexGrabFocus(idx)
}

// DeleteAt deletes element at given index from slice
func (tb *Table) DeleteAt(idx int) {
	if idx < 0 || idx >= tb.SliceSize {
		return
	}
	tb.DeleteAtSelect(idx)
	tb.Table.DeleteRows(idx, 1)
	tb.Update()
}

// SortSliceAction sorts the slice for given field index -- toggles ascending
// vs. descending if already sorting on this dimension
func (tb *Table) SortSliceAction(fldIndex int) {
	sgh := tb.SliceHeader()
	_, idxOff := tb.RowWidgetNs()

	for fli := 0; fli < tb.NCols; fli++ {
		hdr := sgh.Child(idxOff + fli).(*core.Button)
		hdr.SetType(core.ButtonAction)
		if fli == fldIndex {
			if tb.SortIndex == fli {
				tb.SortDescending = !tb.SortDescending
			} else {
				tb.SortDescending = false
			}
		}
	}

	tb.SortIndex = fldIndex
	if fldIndex == -1 {
		tb.Table.SortIndexes()
	} else {
		tb.Table.SortColumn(tb.SortIndex, !tb.SortDescending)
	}
	tb.Update() // requires full update due to sort button icon
}

// TensorDisplayAction allows user to select tensor display options for column
// pass -1 for global params for the entire table
func (tb *Table) TensorDisplayAction(fldIndex int) {
	ctd := &tb.TensorDisplay
	if fldIndex >= 0 {
		ctd = tb.SetColumnTensorDisplay(fldIndex)
	}
	d := core.NewBody().AddTitle("Tensor grid display options")
	core.NewForm(d).SetStruct(ctd)
	d.RunFullDialog(tb)
	// tv.UpdateSliceGrid()
	tb.NeedsRender()
}

func (tb *Table) HasStyler() bool { return false }

func (tb *Table) StyleRow(w core.Widget, idx, fidx int) {}

// SortFieldName returns the name of the field being sorted, along with :up or
// :down depending on descending
func (tb *Table) SortFieldName() string {
	if tb.SortIndex >= 0 && tb.SortIndex < tb.NCols {
		nm := tb.Table.Table.ColumnNames[tb.SortIndex]
		if tb.SortDescending {
			nm += ":down"
		} else {
			nm += ":up"
		}
		return nm
	}
	return ""
}

// SetSortField sets sorting to happen on given field and direction -- see
// SortFieldName for details
func (tb *Table) SetSortFieldName(nm string) {
	if nm == "" {
		return
	}
	spnm := strings.Split(nm, ":")
	got := false
	for fli := 0; fli < tb.NCols; fli++ {
		fld := tb.Table.Table.ColumnNames[fli]
		if fld == spnm[0] {
			got = true
			// fmt.Println("sorting on:", fld.Name, fli, "from:", nm)
			tb.SortIndex = fli
		}
	}
	if len(spnm) == 2 {
		if spnm[1] == "down" {
			tb.SortDescending = true
		} else {
			tb.SortDescending = false
		}
	}
	_ = got
	// if got {
	// 	tv.SortSlice()
	// }
}

// RowFirstVisWidget returns the first visible widget for given row (could be
// index or not) -- false if out of range
func (tb *Table) RowFirstVisWidget(row int) (*core.WidgetBase, bool) {
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
func (tb *Table) RowGrabFocus(row int) *core.WidgetBase {
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

func (tb *Table) SizeFinal() {
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

// SelectedColumnStrings returns the string values of given column name.
func (tb *Table) SelectedColumnStrings(colName string) []string {
	dt := tb.Table.Table
	jis := tb.SelectedIndexesList(false)
	if len(jis) == 0 || dt == nil {
		return nil
	}
	var s []string
	for _, i := range jis {
		v := dt.StringValue(colName, i)
		s = append(s, v)
	}
	return s
}

//////////////////////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

func (tb *Table) MakeToolbar(p *tree.Plan) {
	if tb.Table == nil || tb.Table.Table == nil {
		return
	}
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.Table.AddRows).SetIcon(icons.Add)
		w.SetAfterFunc(func() { tb.Update() })
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.Table.SortColumnName).SetText("Sort").SetIcon(icons.Sort)
		w.SetAfterFunc(func() { tb.Update() })
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.Table.FilterColumnName).SetText("Filter").SetIcon(icons.FilterAlt)
		w.SetAfterFunc(func() { tb.Update() })
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.Table.Sequential).SetText("Unfilter").SetIcon(icons.FilterAltOff)
		w.SetAfterFunc(func() { tb.Update() })
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.Table.OpenCSV).SetIcon(icons.Open)
		w.SetAfterFunc(func() { tb.Update() })
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.Table.SaveCSV).SetIcon(icons.Save)
		w.SetAfterFunc(func() { tb.Update() })
	})
}

func (tb *Table) MimeDataType() string {
	return fileinfo.DataCsv
}

// CopySelectToMime copies selected rows to mime data
func (tb *Table) CopySelectToMime() mimedata.Mimes {
	nitms := len(tb.SelectedIndexes)
	if nitms == 0 {
		return nil
	}
	ix := &table.IndexView{}
	ix.Table = tb.Table.Table
	idx := tb.SelectedIndexesList(false) // ascending
	iidx := make([]int, len(idx))
	for i, di := range idx {
		iidx[i] = tb.Table.Indexes[di]
	}
	ix.Indexes = iidx
	var b bytes.Buffer
	ix.WriteCSV(&b, table.Tab, table.Headers)
	md := mimedata.NewTextBytes(b.Bytes())
	md[0].Type = fileinfo.DataCsv
	return md
}

// FromMimeData returns records from csv of mime data
func (tb *Table) FromMimeData(md mimedata.Mimes) [][]string {
	var recs [][]string
	for _, d := range md {
		if d.Type == fileinfo.DataCsv {
			b := bytes.NewBuffer(d.Data)
			cr := csv.NewReader(b)
			cr.Comma = table.Tab.Rune()
			rec, err := cr.ReadAll()
			if err != nil || len(rec) == 0 {
				log.Printf("Error reading CSV from clipboard: %s\n", err)
				return nil
			}
			recs = append(recs, rec...)
		}
	}
	return recs
}

// PasteAssign assigns mime data (only the first one!) to this idx
func (tb *Table) PasteAssign(md mimedata.Mimes, idx int) {
	recs := tb.FromMimeData(md)
	if len(recs) == 0 {
		return
	}
	tb.Table.Table.ReadCSVRow(recs[1], tb.Table.Indexes[idx])
	tb.UpdateChange()
}

// PasteAtIndex inserts object(s) from mime data at (before) given slice index
// adds to end of table
func (tb *Table) PasteAtIndex(md mimedata.Mimes, idx int) {
	recs := tb.FromMimeData(md)
	nr := len(recs) - 1
	if nr <= 0 {
		return
	}
	tb.Table.InsertRows(idx, nr)
	for ri := 0; ri < nr; ri++ {
		rec := recs[1+ri]
		rw := tb.Table.Indexes[idx+ri]
		tb.Table.Table.ReadCSVRow(rec, rw)
	}
	tb.SendChange()
	tb.SelectIndexEvent(idx, events.SelectOne)
	tb.Update()
}
