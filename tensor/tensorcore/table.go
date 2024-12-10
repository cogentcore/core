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

	"cogentcore.org/core/base/errors"
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

	// Table is the table that we're a view of.
	Table *table.Table `set:"-"`

	// GridStyle has global grid display styles. GridStylers on the Table
	// are applied to this on top of defaults.
	GridStyle GridStyle `set:"-"`

	// ColumnGridStyle has per column grid display styles.
	ColumnGridStyle map[int]*GridStyle `set:"-"`

	// current sort index.
	SortIndex int

	// whether current sort order is descending.
	SortDescending bool

	// number of columns in table (as of last update).
	nCols int `edit:"-"`

	// headerWidths has number of characters in each header, per visfields.
	headerWidths []int `copier:"-" display:"-" json:"-" xml:"-"`

	// colMaxWidths records maximum width in chars of string type fields.
	colMaxWidths []int `set:"-" copier:"-" json:"-" xml:"-"`

	//	blank values for out-of-range rows.
	blankString string
	blankFloat  float64

	// blankCells has per column blank tensor cells.
	blankCells map[int]*tensor.Float64 `set:"-"`
}

// check for interface impl
var _ core.Lister = (*Table)(nil)

func (tb *Table) Init() {
	tb.ListBase.Init()
	tb.SortIndex = -1
	tb.GridStyle.Defaults()
	tb.ColumnGridStyle = map[int]*GridStyle{}
	tb.blankCells = map[int]*tensor.Float64{}

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
	if si < tb.Table.NumRows() {
		vi = tb.Table.RowIndex(si)
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

// SetTable sets the source table that we are viewing, using a sequential view,
// and then configures the display
func (tb *Table) SetTable(dt *table.Table) *Table {
	if dt == nil {
		tb.Table = nil
	} else {
		tb.Table = table.NewView(dt)
		tb.GridStyle.ApplyStylersFrom(tb.Table)
	}
	tb.This.(core.Lister).UpdateSliceSize()
	tb.SetSliceBase()
	tb.Update()
	return tb
}

// SetSlice sets the source table to a [table.NewSliceTable]
// from the given slice.
func (tb *Table) SetSlice(sl any) *Table {
	return tb.SetTable(errors.Log1(table.NewSliceTable(sl)))
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

func (tb *Table) UpdateSliceSize() int {
	tb.Table.ValidIndexes() // table could have changed
	if tb.Table.NumRows() == 0 {
		tb.Table.Sequential()
	}
	tb.SliceSize = tb.Table.NumRows()
	tb.nCols = tb.Table.NumColumns()
	return tb.SliceSize
}

func (tb *Table) UpdateMaxWidths() {
	if len(tb.headerWidths) != tb.nCols {
		tb.headerWidths = make([]int, tb.nCols)
		tb.colMaxWidths = make([]int, tb.nCols)
	}

	if tb.SliceSize == 0 {
		return
	}
	for fli := 0; fli < tb.nCols; fli++ {
		tb.colMaxWidths[fli] = 0
		col := tb.Table.Columns.Values[fli]
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
			for fli := 0; fli < tb.nCols; fli++ {
				field := tb.Table.Columns.Keys[fli]
				tree.AddAt(p, "head-"+field, func(w *core.Button) {
					w.SetType(core.ButtonAction)
					w.Styler(func(s *styles.Style) {
						s.Justify.Content = styles.Start
					})
					w.OnClick(func(e events.Event) {
						tb.SortColumn(fli)
					})
					if tb.Table.Columns.Values[fli].NumDims() > 1 {
						w.AddContextMenu(func(m *core.Scene) {
							core.NewButton(m).SetText("Edit grid style").SetIcon(icons.Edit).
								OnClick(func(e events.Event) {
									tb.EditGridStyle(fli)
								})
						})
					}
					w.Updater(func() {
						field := tb.Table.Columns.Keys[fli]
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
	nWidgPerRow = 1 + tb.nCols
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

	for fli := 0; fli < tb.nCols; fli++ {
		col := tb.Table.Columns.Values[fli]
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
						_, vi, invis := svi.SliceIndex(i)
						if !invis {
							if isstr {
								col.SetString1D(str, vi)
							} else {
								col.SetFloat1D(fval, vi)
							}
						}
						tb.This.(core.Lister).UpdateMaxWidths()
						tb.SendChange()
					})
				}
				wb.Updater(func() {
					_, vi, invis := svi.SliceIndex(i)
					if !invis {
						if isstr {
							str = col.String1D(vi)
							core.Bind(&str, w)
						} else {
							fval = col.Float1D(vi)
							core.Bind(&fval, w)
						}
					} else {
						if isstr {
							core.Bind(tb.blankString, w)
						} else {
							core.Bind(tb.blankFloat, w)
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
						cell = tb.blankCell(fli, col)
					} else {
						cell = col.RowTensor(vi)
					}
					wb.ValueTitle = tb.ValueTitle + "[" + strconv.Itoa(si) + "]"
					w.SetState(invis, states.Invisible)
					w.SetTensor(cell)
					w.GridStyle = *tb.GetColumnGridStyle(fli)
				})
			})
		}
	}
}

// blankCell returns tensor blanks for given tensor col
func (tb *Table) blankCell(cidx int, col tensor.Tensor) *tensor.Float64 {
	if ctb, has := tb.blankCells[cidx]; has {
		return ctb
	}
	ctb := tensor.New[float64](col.ShapeSizes()...).(*tensor.Float64)
	tb.blankCells[cidx] = ctb
	return ctb
}

// GetColumnGridStyle gets grid style for given column.
func (tb *Table) GetColumnGridStyle(col int) *GridStyle {
	if ctd, has := tb.ColumnGridStyle[col]; has {
		return ctd
	}
	ctd := &GridStyle{}
	*ctd = tb.GridStyle
	if tb.Table != nil {
		cl := tb.Table.Columns.Values[col]
		ctd.ApplyStylersFrom(cl)
	}
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

// SortColumn sorts the slice for given column index.
// Toggles ascending vs. descending if already sorting on this dimension.
func (tb *Table) SortColumn(fldIndex int) {
	sgh := tb.SliceHeader()
	_, idxOff := tb.RowWidgetNs()

	for fli := 0; fli < tb.nCols; fli++ {
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
		tb.Table.IndexesNeeded()
		col := tb.Table.ColumnByIndex(tb.SortIndex)
		col.Sort(!tb.SortDescending)
		tb.Table.IndexesFromTensor(col)
	}
	tb.Update() // requires full update due to sort button icon
}

// EditGridStyle shows an editor dialog for grid style for given column index.
func (tb *Table) EditGridStyle(col int) {
	ctd := tb.GetColumnGridStyle(col)
	d := core.NewBody("Tensor grid style")
	core.NewForm(d).SetStruct(ctd).
		OnChange(func(e events.Event) {
			tb.ColumnGridStyle[col] = ctd
			tb.Update()
		})
	core.NewButton(d).SetText("Edit global style").SetIcon(icons.Edit).
		OnClick(func(e events.Event) {
			tb.EditGlobalGridStyle()
		})
	d.RunWindowDialog(tb)
}

// EditGlobalGridStyle shows an editor dialog for global grid styles.
func (tb *Table) EditGlobalGridStyle() {
	d := core.NewBody("Tensor grid style")
	core.NewForm(d).SetStruct(&tb.GridStyle).
		OnChange(func(e events.Event) {
			tb.Update()
		})
	d.RunWindowDialog(tb)
}

func (tb *Table) HasStyler() bool { return false }

func (tb *Table) StyleRow(w core.Widget, idx, fidx int) {}

// SortFieldName returns the name of the field being sorted, along with :up or
// :down depending on descending
func (tb *Table) SortFieldName() string {
	if tb.SortIndex >= 0 && tb.SortIndex < tb.nCols {
		nm := tb.Table.Columns.Keys[tb.SortIndex]
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
	for fli := 0; fli < tb.nCols; fli++ {
		fld := tb.Table.Columns.Keys[fli]
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
	for fli := 0; fli < tb.nCols; fli++ {
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
	for fli := 0; fli < tb.nCols; fli++ {
		w := lg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
		if w.StateIs(states.Focused) || w.ContainsFocus() {
			return w
		}
	}
	tb.InFocusGrab = true
	defer func() { tb.InFocusGrab = false }()
	for fli := 0; fli < tb.nCols; fli++ {
		w := lg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
		if w.CanFocus() {
			w.SetFocus()
			return w
		}
	}
	return nil
}

//////// Header layout

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
	dt := tb.Table
	jis := tb.SelectedIndexesList(false)
	if len(jis) == 0 || dt == nil {
		return nil
	}
	var s []string
	col := dt.Column(colName)
	for _, i := range jis {
		v := col.StringRow(i, 0)
		s = append(s, v)
	}
	return s
}

////////  Copy / Cut / Paste

func (tb *Table) MakeToolbar(p *tree.Plan) {
	if tb.Table == nil {
		return
	}
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.Table.AddRows).SetIcon(icons.Add)
		w.SetAfterFunc(func() { tb.Update() })
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.Table.SortColumns).SetText("Sort").SetIcon(icons.Sort)
		w.SetAfterFunc(func() { tb.Update() })
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(tb.Table.FilterString).SetText("Filter").SetIcon(icons.FilterAlt)
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
	ix := table.NewView(tb.Table)
	idx := tb.SelectedIndexesList(false) // ascending
	iidx := make([]int, len(idx))
	for i, di := range idx {
		iidx[i] = tb.Table.RowIndex(di)
	}
	ix.Indexes = iidx
	var b bytes.Buffer
	ix.WriteCSV(&b, tensor.Tab, table.Headers)
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
			cr.Comma = tensor.Tab.Rune()
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
	tb.Table.ReadCSVRow(recs[1], tb.Table.RowIndex(idx))
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
		rw := tb.Table.RowIndex(idx + ri)
		tb.Table.ReadCSVRow(rec, rw)
	}
	tb.SendChange()
	tb.SelectIndexEvent(idx, events.SelectOne)
	tb.Update()
}
