// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorview

//go:generate core generate -add-types

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
	"cogentcore.org/core/views"
)

// TableView provides a GUI view for [table.Table] values.
type TableView struct {
	views.SliceViewBase

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
	headerWidths []int `copier:"-" view:"-" json:"-" xml:"-"`

	// colMaxWidths records maximum width in chars of string type fields
	colMaxWidths []int `set:"-" copier:"-" json:"-" xml:"-"`

	//	blank values for out-of-range rows
	BlankString string
	BlankFloat  float64
}

// check for interface impl
var _ views.SliceViewer = (*TableView)(nil)

func (tv *TableView) Init() {
	tv.SliceViewBase.Init()
	tv.SortIndex = -1
	tv.TensorDisplay.Defaults()
	tv.ColumnTensorDisplay = map[int]*TensorDisplay{}
	tv.ColumnTensorBlank = map[int]*tensor.Float64{}

	tv.Makers[0] = func(p *core.Plan) { // TODO: reduce redundancy with SliceViewBase Maker
		svi := tv.This().(views.SliceViewer)
		svi.UpdateSliceSize()

		tv.ViewMuLock()
		defer tv.ViewMuUnlock()

		scrollTo := -1
		if tv.InitSelectedIndex >= 0 {
			tv.SelectedIndex = tv.InitSelectedIndex
			tv.InitSelectedIndex = -1
			scrollTo = tv.SelectedIndex
		}
		if scrollTo >= 0 {
			tv.ScrollToIndex(scrollTo)
		}

		tv.UpdateStartIndex()
		tv.UpdateMaxWidths()

		tv.Updater(func() {
			tv.UpdateStartIndex()
			tv.UpdateMaxWidths()
		})

		tv.MakeHeader(p)
		tv.MakeGrid(p, func(p *core.Plan) {
			for i := 0; i < tv.VisRows; i++ {
				svi.MakeRow(p, i)
			}
		})
	}
}

func (tv *TableView) SliceIndex(i int) (si, vi int, invis bool) {
	si = tv.StartIndex + i
	vi = -1
	if si < len(tv.Table.Indexes) {
		vi = tv.Table.Indexes[si]
	}
	invis = vi < 0
	return
}

// StyleValue performs additional value widget styling
func (tv *TableView) StyleValue(w core.Widget, s *styles.Style, row, col int) {
	hw := float32(tv.headerWidths[col])
	if col == tv.SortIndex {
		hw += 6
	}
	if len(tv.colMaxWidths) > col {
		hw = max(float32(tv.colMaxWidths[col]), hw)
	}
	hv := units.Ch(hw)
	s.Min.X.Value = max(s.Min.X.Value, hv.Convert(s.Min.X.Unit, &s.UnitContext).Value)
	s.SetTextWrap(false)
}

// SetTable sets the source table that we are viewing, using a sequential IndexView
// and then configures the display
func (tv *TableView) SetTable(et *table.Table) *TableView {
	if et == nil {
		return nil
	}

	tv.SetSliceBase()
	tv.Table = table.NewIndexView(et)
	tv.This().(views.SliceViewer).UpdateSliceSize()
	tv.Update()
	return tv
}

// GoUpdateView updates the display for asynchronous updating from
// other goroutines.  Also updates indexview (calling Sequential).
func (tv *TableView) GoUpdateView() {
	tv.AsyncLock()
	tv.Table.Sequential()
	tv.ScrollToIndexNoUpdate(tv.SliceSize - 1)
	tv.Update()
	tv.AsyncUnlock()
}

// SetTableView sets the source IndexView of a table (using a copy so original is not modified)
// and then configures the display
func (tv *TableView) SetTableView(ix *table.IndexView) *TableView {
	if ix == nil {
		return tv
	}

	tv.Table = ix.Clone() // always copy

	tv.This().(views.SliceViewer).UpdateSliceSize()
	tv.StartIndex = 0
	tv.VisRows = tv.MinRows
	if !tv.IsReadOnly() {
		tv.SelectedIndex = -1
	}
	tv.ResetSelectedIndexes()
	tv.SetFlag(false, views.SliceViewSelectMode)
	tv.MakeIter = 0
	tv.Update()
	return tv
}

func (tv *TableView) UpdateSliceSize() int {
	tv.Table.DeleteInvalid() // table could have changed
	if tv.Table.Len() == 0 {
		tv.Table.Sequential()
	}
	tv.SliceSize = tv.Table.Len()
	tv.NCols = tv.Table.Table.NumColumns()
	return tv.SliceSize
}

func (tv *TableView) UpdateMaxWidths() {
	if len(tv.headerWidths) != tv.NCols {
		tv.headerWidths = make([]int, tv.NCols)
		tv.colMaxWidths = make([]int, tv.NCols)
	}

	if tv.SliceSize == 0 {
		return
	}
	for fli := 0; fli < tv.NCols; fli++ {
		tv.colMaxWidths[fli] = 0
		col := tv.Table.Table.Columns[fli]
		stsr, isstr := col.(*tensor.String)

		if !isstr {
			continue
		}
		mxw := 0
		for _, ixi := range tv.Table.Indexes {
			if ixi >= 0 {
				sval := stsr.Values[ixi]
				mxw = max(mxw, len(sval))
			}
		}
		tv.colMaxWidths[fli] = mxw
	}
}

func (tv *TableView) MakeHeader(p *core.Plan) {
	core.AddAt(p, "header", func(w *core.Frame) {
		core.ToolbarStyles(w)
		w.Style(func(s *styles.Style) {
			s.Grow.Set(0, 0)
			s.Gap.Set(units.Em(0.5)) // matches grid default
		})
		w.Maker(func(p *core.Plan) {
			if tv.Is(views.SliceViewShowIndex) {
				core.AddAt(p, "head-index", func(w *core.Text) { // TODO: is not working
					w.SetType(core.TextBodyMedium)
					w.Style(func(s *styles.Style) {
						s.Align.Self = styles.Center
					})
					w.SetText("Index")
				})
			}
			for fli := 0; fli < tv.NCols; fli++ {
				field := tv.Table.Table.ColumnNames[fli]
				core.AddAt(p, "head-"+field, func(w *core.Button) {
					w.SetType(core.ButtonMenu)
					w.SetText(field)
					w.OnClick(func(e events.Event) {
						tv.SortSliceAction(fli)
					})
					w.Updater(func() {
						field := tv.Table.Table.ColumnNames[fli]
						w.SetText(field).SetTooltip(field + " (tap to sort by)")
						tv.headerWidths[fli] = len(field)
						if fli == tv.SortIndex {
							if tv.SortDescending {
								w.SetIcon(icons.KeyboardArrowDown)
							} else {
								w.SetIcon(icons.KeyboardArrowUp)
							}
						}
					})
				})
			}
		})
	})
}

// SliceHeader returns the Frame header for slice grid
func (tv *TableView) SliceHeader() *core.Frame {
	return tv.Child(0).(*core.Frame)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (tv *TableView) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 1 + tv.NCols
	idxOff = 1
	if !tv.Is(views.SliceViewShowIndex) {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

func (tv *TableView) MakeRow(p *core.Plan, i int) {
	svi := tv.This().(views.SliceViewer)
	si, _, invis := svi.SliceIndex(i)
	itxt := strconv.Itoa(i)

	if tv.Is(views.SliceViewShowIndex) {
		tv.MakeGridIndex(p, i, si, itxt, invis)
	}

	for fli := 0; fli < tv.NCols; fli++ {
		col := tv.Table.Table.Columns[fli]
		valnm := fmt.Sprintf("value-%v.%v", fli, itxt)

		_, isstr := col.(*tensor.String)
		if col.NumDims() == 1 {
			str := ""
			fval := float64(0)
			core.AddNew(p, valnm, func() core.Value {
				if isstr {
					return core.NewValue(&str, "")
				} else {
					return core.NewValue(&fval, "")
				}
			}, func(w core.Value) {
				wb := w.AsWidget()
				tv.MakeValue(w, i)
				w.AsTree().SetProperty(views.SliceViewColProperty, fli)
				if !tv.IsReadOnly() {
					wb.OnChange(func(e events.Event) {
						if si < len(tv.Table.Indexes) {
							if isstr {
								tv.Table.Table.SetStringIndex(fli, tv.Table.Indexes[si], str)
							} else {
								tv.Table.Table.SetFloatIndex(fli, tv.Table.Indexes[si], fval)
							}
						}
						tv.SendChange()
					})
				}
				wb.Updater(func() {
					_, vi, invis := svi.SliceIndex(i)
					if !invis {
						if isstr {
							str = tv.Table.Table.StringIndex(fli, vi)
							core.Bind(&str, w)
						} else {
							fval = tv.Table.Table.FloatIndex(fli, vi)
							core.Bind(&fval, w)
						}
					} else {
						if isstr {
							core.Bind(tv.BlankString, w)
						} else {
							core.Bind(tv.BlankFloat, w)
						}
					}
					wb.SetReadOnly(tv.IsReadOnly())
					w.SetState(invis, states.Invisible)
					if svi.HasStyleFunc() {
						w.ApplyStyle()
					}
					if invis {
						wb.SetSelected(false)
					}
				})
			})
		} else {
			core.AddAt(p, valnm, func(w *TensorGrid) {
				w.SetReadOnly(tv.IsReadOnly())
				wb := w.AsWidget()
				w.SetProperty(views.SliceViewRowProperty, i)
				w.SetProperty(views.SliceViewColProperty, fli)
				w.Style(func(s *styles.Style) {
					s.Grow.Set(0, 0)
				})
				wb.Updater(func() {
					si, vi, invis := svi.SliceIndex(i)
					var cell tensor.Tensor
					if invis {
						cell = tv.ColTensorBlank(fli, col)
					} else {
						cell = tv.Table.Table.TensorIndex(fli, vi)
					}
					wb.ValueTitle = tv.ValueTitle + "[" + strconv.Itoa(si) + "]"
					w.SetState(invis, states.Invisible)
					w.SetTensor(cell)
					w.Display = *tv.GetColumnTensorDisplay(fli)
				})
			})
		}
	}
}

// ColTensorBlank returns tensor blanks for given tensor col
func (tv *TableView) ColTensorBlank(cidx int, col tensor.Tensor) *tensor.Float64 {
	if ctb, has := tv.ColumnTensorBlank[cidx]; has {
		return ctb
	}
	ctb := tensor.New[float64](col.Shape().Sizes, col.Shape().Names...).(*tensor.Float64)
	tv.ColumnTensorBlank[cidx] = ctb
	return ctb
}

// GetColumnTensorDisplay returns tensor display parameters for this column
// either the overall defaults or the per-column if set
func (tv *TableView) GetColumnTensorDisplay(col int) *TensorDisplay {
	if ctd, has := tv.ColumnTensorDisplay[col]; has {
		return ctd
	}
	if tv.Table != nil {
		cl := tv.Table.Table.Columns[col]
		if len(cl.MetaDataMap()) > 0 {
			return tv.SetColumnTensorDisplay(col)
		}
	}
	return &tv.TensorDisplay
}

// SetColumnTensorDisplay sets per-column tensor display params and returns them
// if already set, just returns them
func (tv *TableView) SetColumnTensorDisplay(col int) *TensorDisplay {
	if ctd, has := tv.ColumnTensorDisplay[col]; has {
		return ctd
	}
	ctd := &TensorDisplay{}
	*ctd = tv.TensorDisplay
	if tv.Table != nil {
		cl := tv.Table.Table.Columns[col]
		ctd.FromMeta(cl)
	}
	tv.ColumnTensorDisplay[col] = ctd
	return ctd
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (tv *TableView) SliceNewAt(idx int) {
	tv.ViewMuLock()

	tv.SliceNewAtSelect(idx)

	tv.Table.InsertRows(idx, 1)

	tv.SelectIndexAction(idx, events.SelectOne)
	tv.ViewMuUnlock()
	tv.Update()
	tv.IndexGrabFocus(idx)
}

// SliceDeleteAt deletes element at given index from slice
func (tv *TableView) SliceDeleteAt(idx int) {
	if idx < 0 || idx >= tv.SliceSize {
		return
	}
	tv.ViewMuLock()

	tv.SliceDeleteAtSelect(idx)

	tv.Table.DeleteRows(idx, 1)

	tv.ViewMuUnlock()
	tv.Update()
}

// SortSliceAction sorts the slice for given field index -- toggles ascending
// vs. descending if already sorting on this dimension
func (tv *TableView) SortSliceAction(fldIndex int) {
	sgh := tv.SliceHeader()
	_, idxOff := tv.RowWidgetNs()

	ascending := true

	for fli := 0; fli < tv.NCols; fli++ {
		hdr := sgh.Child(idxOff + fli).(*core.Button)
		hdr.SetType(core.ButtonAction)
		if fli == fldIndex {
			if tv.SortIndex == fli {
				tv.SortDescending = !tv.SortDescending
				ascending = !tv.SortDescending
			} else {
				tv.SortDescending = false
			}
			if ascending {
				hdr.SetIcon(icons.KeyboardArrowUp)
			} else {
				hdr.SetIcon(icons.KeyboardArrowDown)
			}
		} else {
			hdr.SetIcon("none")
		}
	}

	tv.SortIndex = fldIndex
	if fldIndex == -1 {
		tv.Table.SortIndexes()
	} else {
		tv.Table.SortColumn(tv.SortIndex, !tv.SortDescending)
	}
	tv.Update() // requires full update due to sort button icon
}

// TensorDisplayAction allows user to select tensor display options for column
// pass -1 for global params for the entire table
func (tv *TableView) TensorDisplayAction(fldIndex int) {
	ctd := &tv.TensorDisplay
	if fldIndex >= 0 {
		ctd = tv.SetColumnTensorDisplay(fldIndex)
	}
	d := core.NewBody().AddTitle("Tensor grid display options")
	views.NewStructView(d).SetStruct(ctd)
	d.RunFullDialog(tv)
	// tv.UpdateSliceGrid()
	tv.NeedsRender()
}

func (tv *TableView) HasStyleFunc() bool {
	return false
}

func (tv *TableView) StyleRow(w core.Widget, idx, fidx int) {
}

// SortFieldName returns the name of the field being sorted, along with :up or
// :down depending on descending
func (tv *TableView) SortFieldName() string {
	if tv.SortIndex >= 0 && tv.SortIndex < tv.NCols {
		nm := tv.Table.Table.ColumnNames[tv.SortIndex]
		if tv.SortDescending {
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
func (tv *TableView) SetSortFieldName(nm string) {
	if nm == "" {
		return
	}
	spnm := strings.Split(nm, ":")
	got := false
	for fli := 0; fli < tv.NCols; fli++ {
		fld := tv.Table.Table.ColumnNames[fli]
		if fld == spnm[0] {
			got = true
			// fmt.Println("sorting on:", fld.Name, fli, "from:", nm)
			tv.SortIndex = fli
		}
	}
	if len(spnm) == 2 {
		if spnm[1] == "down" {
			tv.SortDescending = true
		} else {
			tv.SortDescending = false
		}
	}
	_ = got
	// if got {
	// 	tv.SortSlice()
	// }
}

// RowFirstVisWidget returns the first visible widget for given row (could be
// index or not) -- false if out of range
func (tv *TableView) RowFirstVisWidget(row int) (*core.WidgetBase, bool) {
	if !tv.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	sg := tv.SliceGrid()
	w := sg.Kids[row*nWidgPerRow].(core.Widget).AsWidget()
	if w.Geom.TotalBBox != (image.Rectangle{}) {
		return w, true
	}
	ridx := nWidgPerRow * row
	for fli := 0; fli < tv.NCols; fli++ {
		w := sg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
		if w.Geom.TotalBBox != (image.Rectangle{}) {
			return w, true
		}
	}
	return nil, false
}

// RowGrabFocus grabs the focus for the first focusable widget in given row --
// returns that element or nil if not successful -- note: grid must have
// already rendered for focus to be grabbed!
func (tv *TableView) RowGrabFocus(row int) *core.WidgetBase {
	if !tv.IsRowInBounds(row) || tv.Is(views.SliceViewInFocusGrab) { // range check
		return nil
	}
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := tv.SliceGrid()
	// first check if we already have focus
	for fli := 0; fli < tv.NCols; fli++ {
		w := sg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
		if w.StateIs(states.Focused) || w.ContainsFocus() {
			return w
		}
	}
	tv.SetFlag(true, views.SliceViewInFocusGrab)
	defer func() { tv.SetFlag(false, views.SliceViewInFocusGrab) }()
	for fli := 0; fli < tv.NCols; fli++ {
		w := sg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
		if w.CanFocus() {
			w.SetFocusEvent()
			return w
		}
	}
	return nil
}

//////////////////////////////////////////////////////
// 	Header layout

func (tv *TableView) SizeFinal() {
	tv.SliceViewBase.SizeFinal()
	sg := tv.This().(views.SliceViewer).SliceGrid()
	sh := tv.SliceHeader()
	sh.WidgetKidsIter(func(i int, kwi core.Widget, kwb *core.WidgetBase) bool {
		_, sgb := core.AsWidget(sg.Child(i))
		gsz := &sgb.Geom.Size
		if gsz.Actual.Total.X == 0 {
			return tree.Continue
		}
		ksz := &kwb.Geom.Size
		ksz.Actual.Total.X = gsz.Actual.Total.X
		ksz.Actual.Content.X = gsz.Actual.Content.X
		ksz.Alloc.Total.X = gsz.Alloc.Total.X
		ksz.Alloc.Content.X = gsz.Alloc.Content.X
		return tree.Continue
	})
	gsz := &sg.Geom.Size
	ksz := &sh.Geom.Size
	if gsz.Actual.Total.X > 0 {
		ksz.Actual.Total.X = gsz.Actual.Total.X
		ksz.Actual.Content.X = gsz.Actual.Content.X
		ksz.Alloc.Total.X = gsz.Alloc.Total.X
		ksz.Alloc.Content.X = gsz.Alloc.Content.X
	}
}

// SelectedColumnStrings returns the string values of given column name.
func (tv *TableView) SelectedColumnStrings(colName string) []string {
	dt := tv.Table.Table
	jis := tv.SelectedIndexesList(false)
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

func (tv *TableView) MakeToolbar(p *core.Plan) {
	if tv.Table == nil || tv.Table.Table == nil {
		return
	}
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(tv.Table.AddRows).SetIcon(icons.Add)
		w.SetAfterFunc(func() { tv.Update() })
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(tv.Table.SortColumnName).SetText("Sort").SetIcon(icons.Sort)
		w.SetAfterFunc(func() { tv.Update() })
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(tv.Table.FilterColumnName).SetText("Filter").SetIcon(icons.FilterAlt)
		w.SetAfterFunc(func() { tv.Update() })
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(tv.Table.Sequential).SetText("Unfilter").SetIcon(icons.FilterAltOff)
		w.SetAfterFunc(func() { tv.Update() })
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(tv.Table.OpenCSV).SetIcon(icons.Open)
		w.SetAfterFunc(func() { tv.Update() })
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(tv.Table.SaveCSV).SetIcon(icons.Save)
		w.SetAfterFunc(func() { tv.Update() })
	})
}

func (tv *TableView) MimeDataType() string {
	return fileinfo.DataCsv
}

// CopySelectToMime copies selected rows to mime data
func (tv *TableView) CopySelectToMime() mimedata.Mimes {
	nitms := len(tv.SelectedIndexes)
	if nitms == 0 {
		return nil
	}
	ix := &table.IndexView{}
	ix.Table = tv.Table.Table
	idx := tv.SelectedIndexesList(false) // ascending
	iidx := make([]int, len(idx))
	for i, di := range idx {
		iidx[i] = tv.Table.Indexes[di]
	}
	ix.Indexes = iidx
	var b bytes.Buffer
	ix.WriteCSV(&b, table.Tab, table.Headers)
	md := mimedata.NewTextBytes(b.Bytes())
	md[0].Type = fileinfo.DataCsv
	return md
}

// FromMimeData returns records from csv of mime data
func (tv *TableView) FromMimeData(md mimedata.Mimes) [][]string {
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
func (tv *TableView) PasteAssign(md mimedata.Mimes, idx int) {
	recs := tv.FromMimeData(md)
	if len(recs) == 0 {
		return
	}
	tv.Table.Table.ReadCSVRow(recs[1], tv.Table.Indexes[idx])
	tv.SendChange()
	tv.Update()
}

// PasteAtIndex inserts object(s) from mime data at (before) given slice index
// adds to end of table
func (tv *TableView) PasteAtIndex(md mimedata.Mimes, idx int) {
	recs := tv.FromMimeData(md)
	nr := len(recs) - 1
	if nr <= 0 {
		return
	}
	tv.Table.InsertRows(idx, nr)
	for ri := 0; ri < nr; ri++ {
		rec := recs[1+ri]
		rw := tv.Table.Indexes[idx+ri]
		tv.Table.Table.ReadCSVRow(rec, rw)
	}
	tv.SendChange()
	tv.SelectIndexAction(idx, events.SelectOne)
	tv.Update()
}
