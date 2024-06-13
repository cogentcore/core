// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log/slog"
	"reflect"
	"strconv"
	"strings"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// todo:
// * search option, both as a search field and as simple type-to-search

// Table represents a slice of structs as a table, where the fields are
// the columns and the elements are the rows. It is a full-featured editor with
// multiple-selection, cut-and-paste, and drag-and-drop.
// Use [ListBase.BindSelect] to make the table designed for item selection.
type Table struct {
	ListBase

	// StyleFunc is an optional styling function.
	StyleFunc TableStyleFunc `copier:"-" view:"-" json:"-" xml:"-"`

	// SelectedField is the current selection field; initially select value in this field.
	SelectedField string `copier:"-" view:"-" json:"-" xml:"-"`

	// SortIndex is the current sort index.
	SortIndex int

	// SortDescending is whether the current sort order is descending.
	SortDescending bool

	// visibleFields are the visible fields.
	visibleFields []reflect.StructField

	// numVisibleFields is the number of visible fields.
	numVisibleFields int

	// headerWidths has the number of characters in each header, per visibleFields.
	headerWidths []int

	// colMaxWidths records maximum width in chars of string type fields.
	colMaxWidths []int
}

// TableStyleFunc is a styling function for custom styling and
// configuration of elements in the table.
type TableStyleFunc func(w Widget, s *styles.Style, row, col int)

func (tb *Table) Init() {
	tb.ListBase.Init()
	tb.AddContextMenu(tb.ContextMenu)
	tb.SortIndex = -1

	tb.Makers[0] = func(p *Plan) { // TODO: reduce redundancy with ListBase Maker
		svi := tb.This.(Lister)
		svi.UpdateSliceSize()

		tb.SortSlice()

		scrollTo := -1
		if tb.SelectedField != "" && tb.SelectedValue != nil {
			tb.SelectedIndex, _ = StructSliceIndexByValue(tb.Slice, tb.SelectedField, tb.SelectedValue)
			tb.SelectedField = ""
			tb.SelectedValue = nil
			tb.InitSelectedIndex = -1
			scrollTo = tb.SelectedIndex
		} else if tb.InitSelectedIndex >= 0 {
			tb.SelectedIndex = tb.InitSelectedIndex
			tb.InitSelectedIndex = -1
			scrollTo = tb.SelectedIndex
		}
		if scrollTo >= 0 {
			tb.ScrollToIndex(scrollTo)
		}
		tb.UpdateStartIndex()

		tb.Updater(func() {
			tb.UpdateStartIndex()
			svi.UpdateMaxWidths()
		})

		tb.MakeHeader(p)
		tb.MakeGrid(p, func(p *Plan) {
			for i := 0; i < tb.VisRows; i++ {
				svi.MakeRow(p, i)
			}
		})
	}
}

// StyleValue performs additional value widget styling
func (tb *Table) StyleValue(w Widget, s *styles.Style, row, col int) {
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

// SetSlice sets the source slice that we are viewing.
// Must call Update if already open.
func (tb *Table) SetSlice(sl any) *Table {
	if reflectx.AnyIsNil(sl) {
		tb.Slice = nil
		return tb
	}
	if tb.Slice == sl {
		tb.MakeIter = 0
		return tb
	}

	slpTyp := reflect.TypeOf(sl)
	if slpTyp.Kind() != reflect.Pointer {
		slog.Error("Table requires that you pass a pointer to a slice of struct elements, but type is not a Ptr", "type", slpTyp)
		return tb
	}
	if slpTyp.Elem().Kind() != reflect.Slice {
		slog.Error("Table requires that you pass a pointer to a slice of struct elements, but ptr doesn't point to a slice", "type", slpTyp.Elem())
		return tb
	}
	eltyp := reflectx.NonPointerType(reflectx.SliceElementType(sl))
	if eltyp.Kind() != reflect.Struct {
		slog.Error("Table requires that you pass a slice of struct elements, but type is not a Struct", "type", eltyp.String())
		return tb
	}

	tb.SetSliceBase()
	tb.Slice = sl
	tb.SliceUnderlying = reflectx.Underlying(reflect.ValueOf(tb.Slice))
	tb.ElementValue = reflectx.Underlying(reflectx.SliceElementValue(sl))
	tb.cacheVisibleFields()
	return tb
}

// cacheVisibleFields caches the visible struct fields.
func (tb *Table) cacheVisibleFields() {
	tb.visibleFields = make([]reflect.StructField, 0)
	shouldShow := func(field reflect.StructField) bool {
		tvtag := field.Tag.Get("table")
		switch {
		case tvtag == "+":
			return true
		case tvtag == "-":
			return false
		case tvtag == "-select" && tb.IsReadOnly():
			return false
		case tvtag == "-edit" && !tb.IsReadOnly():
			return false
		default:
			return field.Tag.Get("view") != "-"
		}
	}

	reflectx.WalkFields(tb.ElementValue,
		func(parent reflect.Value, field reflect.StructField, value reflect.Value) bool {
			return shouldShow(field)
		},
		func(parent reflect.Value, field reflect.StructField, value reflect.Value) {
			tb.visibleFields = append(tb.visibleFields, field)
		})
	tb.numVisibleFields = len(tb.visibleFields)
	tb.headerWidths = make([]int, tb.numVisibleFields)
	tb.colMaxWidths = make([]int, tb.numVisibleFields)
}

func (tb *Table) UpdateMaxWidths() {
	if tb.SliceSize == 0 {
		return
	}
	for fli := 0; fli < tb.numVisibleFields; fli++ {
		field := tb.visibleFields[fli]
		tb.colMaxWidths[fli] = 0
		val := tb.SliceElementValue(0)
		fval := val.FieldByIndex(field.Index)
		_, isicon := fval.Interface().(icons.Icon)
		isString := fval.Type().Kind() == reflect.String
		if !isString || isicon {
			continue
		}
		mxw := 0
		for rw := 0; rw < tb.SliceSize; rw++ {
			val := tb.SliceElementValue(rw)
			str := reflectx.ToString(val.FieldByIndex(field.Index).Interface())
			mxw = max(mxw, len(str))
		}
		tb.colMaxWidths[fli] = mxw
	}
}

// SliceHeader returns the Frame header for slice grid
func (tb *Table) SliceHeader() *Frame {
	return tb.Child(0).(*Frame)
}

func (tb *Table) MakeHeader(p *Plan) {
	AddAt(p, "header", func(w *Frame) {
		ToolbarStyles(w)
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(0, 0)
			s.Gap.Set(units.Em(0.5)) // matches grid default
		})
		w.Maker(func(p *Plan) {
			if tb.ShowIndexes {
				AddAt(p, "head-index", func(w *Text) {
					w.SetType(TextBodyMedium)
					w.Styler(func(s *styles.Style) {
						s.Align.Self = styles.Center
					})
					w.SetText("Index")
				})
			}
			for fli := 0; fli < tb.numVisibleFields; fli++ {
				field := tb.visibleFields[fli]
				AddAt(p, "head-"+field.Name, func(w *Button) {
					w.SetType(ButtonMenu)
					w.OnClick(func(e events.Event) {
						tb.SortSliceAction(fli)
					})
					w.Updater(func() {
						htxt := ""
						if lbl, ok := field.Tag.Lookup("label"); ok {
							htxt = lbl
						} else {
							htxt = strcase.ToSentence(field.Name)
						}
						w.SetText(htxt)
						w.Tooltip = htxt + " (click to sort by)"
						doc, ok := types.GetDoc(reflect.Value{}, tb.ElementValue, field, htxt)
						if ok && doc != "" {
							w.Tooltip += ": " + doc
						}
						tb.headerWidths[fli] = len(htxt)
						if fli == tb.SortIndex {
							if tb.SortDescending {
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

// RowWidgetNs returns number of widgets per row and offset for index label
func (tb *Table) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 1 + tb.numVisibleFields
	idxOff = 1
	if !tb.ShowIndexes {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

func (tb *Table) MakeRow(p *Plan, i int) {
	svi := tb.This.(Lister)
	si, _, invis := svi.SliceIndex(i)
	itxt := strconv.Itoa(i)
	val := tb.SliceElementValue(si)
	// stru := val.Interface()

	if tb.ShowIndexes {
		tb.MakeGridIndex(p, i, si, itxt, invis)
	}

	for fli := 0; fli < tb.numVisibleFields; fli++ {
		field := tb.visibleFields[fli]
		fval := reflectx.OnePointerValue(val.FieldByIndex(field.Index))
		valnm := fmt.Sprintf("value-%d-%s-%s", fli, itxt, reflectx.ShortTypeName(field.Type))
		tags := field.Tag
		if fval.Kind() == reflect.Slice || fval.Kind() == reflect.Map {
			ni := reflect.StructTag(`view:"no-inline"`)
			if tags == "" {
				tags += " " + ni
			} else {
				tags = ni
			}
		}
		readOnlyTag := tags.Get("edit") == "-"

		AddNew(p, valnm, func() Value {
			return NewValue(fval.Interface(), tags)
		}, func(w Value) {
			wb := w.AsWidget()
			tb.MakeValue(w, i)
			w.AsTree().SetProperty(ListColProperty, fli)
			if !tb.IsReadOnly() && !readOnlyTag {
				wb.OnChange(func(e events.Event) {
					tb.SendChange()
				})
				wb.OnInput(tb.HandleEvent)
			}
			wb.Updater(func() {
				si, vi, invis := svi.SliceIndex(i)
				val := tb.SliceElementValue(vi)
				fval := reflectx.OnePointerValue(val.FieldByIndex(field.Index))
				Bind(fval.Interface(), w)

				vc := tb.ValueTitle + "[" + strconv.Itoa(si) + "]"
				if !invis {
					if lblr, ok := tb.Slice.(labels.SliceLabeler); ok {
						slbl := lblr.ElemLabel(si)
						if slbl != "" {
							vc = JoinValueTitle(tb.ValueTitle, slbl)
						}
					}
				}
				wb.ValueTitle = vc + " (" + wb.ValueTitle + ")"
				wb.SetReadOnly(tb.IsReadOnly() || readOnlyTag)
				wb.SetState(invis, states.Invisible)
				if svi.HasStyleFunc() {
					w.Style()
				}
				if invis {
					wb.SetSelected(false)
				}
			})
		})
	}
}

func (tb *Table) HasStyleFunc() bool {
	return tb.StyleFunc != nil
}

func (tb *Table) StyleRow(w Widget, idx, fidx int) {
	if tb.StyleFunc != nil {
		tb.StyleFunc(w, &w.AsWidget().Styles, idx, fidx)
	}
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (tb *Table) SliceNewAt(idx int) {
	tb.SliceNewAtSelect(idx)
	reflectx.SliceNewAt(tb.Slice, idx)
	if idx < 0 {
		idx = tb.SliceSize
	}

	tb.This.(Lister).UpdateSliceSize()
	tb.SelectIndexAction(idx, events.SelectOne)
	tb.SendChange()
	tb.Update()
	tb.IndexGrabFocus(idx)
}

// SliceDeleteAt deletes element at given index from slice
func (tb *Table) SliceDeleteAt(idx int) {
	if idx < 0 || idx >= tb.SliceSize {
		return
	}

	tb.SliceDeleteAtSelect(idx)

	reflectx.SliceDeleteAt(tb.Slice, idx)

	tb.This.(Lister).UpdateSliceSize()
	tb.SendChange()
	tb.Update()
}

// SortSlice sorts the slice according to current settings
func (tb *Table) SortSlice() {
	if tb.SortIndex < 0 || tb.SortIndex >= len(tb.visibleFields) {
		return
	}
	rawIndex := tb.visibleFields[tb.SortIndex].Index
	reflectx.StructSliceSort(tb.Slice, rawIndex, !tb.SortDescending)
}

// SortSliceAction sorts the slice for given field index -- toggles ascending
// vs. descending if already sorting on this dimension
func (tb *Table) SortSliceAction(fldIndex int) {
	sgh := tb.SliceHeader()
	_, idxOff := tb.RowWidgetNs()

	ascending := true

	for fli := 0; fli < tb.numVisibleFields; fli++ {
		hdr := sgh.Child(idxOff + fli).(*Button)
		hdr.SetType(ButtonAction)
		if fli == fldIndex {
			if tb.SortIndex == fli {
				tb.SortDescending = !tb.SortDescending
				ascending = !tb.SortDescending
			} else {
				tb.SortDescending = false
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

	tb.SortIndex = fldIndex
	tb.SortSlice()
	tb.Update()
}

// SortFieldName returns the name of the field being sorted, along with :up or
// :down depending on descending
func (tb *Table) SortFieldName() string {
	if tb.SortIndex >= 0 && tb.SortIndex < tb.numVisibleFields {
		nm := tb.visibleFields[tb.SortIndex].Name
		if tb.SortDescending {
			nm += ":down"
		} else {
			nm += ":up"
		}
		return nm
	}
	return ""
}

// SetSortFieldName sets sorting to happen on given field and direction -- see
// SortFieldName for details
func (tb *Table) SetSortFieldName(nm string) {
	if nm == "" {
		return
	}
	spnm := strings.Split(nm, ":")
	got := false
	for fli := 0; fli < tb.numVisibleFields; fli++ {
		fld := tb.visibleFields[fli]
		if fld.Name == spnm[0] {
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
	if got {
		tb.SortSlice()
	}
}

// RowFirstVisWidget returns the first visible widget for given row (could be
// index or not) -- false if out of range
func (tb *Table) RowFirstVisWidget(row int) (*WidgetBase, bool) {
	if !tb.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, idxOff := tb.RowWidgetNs()
	sg := tb.SliceGrid()
	w := sg.Children[row*nWidgPerRow].(Widget).AsWidget()
	if w.Geom.TotalBBox != (image.Rectangle{}) {
		return w, true
	}
	ridx := nWidgPerRow * row
	for fli := 0; fli < tb.numVisibleFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(Widget).AsWidget()
		if w.Geom.TotalBBox != (image.Rectangle{}) {
			return w, true
		}
	}
	return nil, false
}

// RowGrabFocus grabs the focus for the first focusable widget in given row --
// returns that element or nil if not successful -- note: grid must have
// already rendered for focus to be grabbed!
func (tb *Table) RowGrabFocus(row int) *WidgetBase {
	if !tb.IsRowInBounds(row) || tb.InFocusGrab { // range check
		return nil
	}
	nWidgPerRow, idxOff := tb.RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := tb.SliceGrid()
	// first check if we already have focus
	for fli := 0; fli < tb.numVisibleFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(Widget).AsWidget()
		if w.StateIs(states.Focused) || w.ContainsFocus() {
			return w
		}
	}
	tb.InFocusGrab = true
	defer func() { tb.InFocusGrab = false }()
	for fli := 0; fli < tb.numVisibleFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(Widget).AsWidget()
		if w.CanFocus() {
			w.SetFocusEvent()
			return w
		}
	}
	return nil
}

// SelectFieldVal sets SelField and SelVal and attempts to find corresponding
// row, setting SelectedIndex and selecting row if found -- returns true if
// found, false otherwise
func (tb *Table) SelectFieldVal(fld, val string) bool {
	tb.SelectedField = fld
	tb.SelectedValue = val
	if tb.SelectedField != "" && tb.SelectedValue != nil {
		idx, _ := StructSliceIndexByValue(tb.Slice, tb.SelectedField, tb.SelectedValue)
		if idx >= 0 {
			tb.ScrollToIndex(idx)
			tb.UpdateSelectIndex(idx, true, events.SelectOne)
			return true
		}
	}
	return false
}

// StructSliceIndexByValue searches for first index that contains given value in field of
// given name.
func StructSliceIndexByValue(structSlice any, fieldName string, fieldValue any) (int, error) {
	svnp := reflectx.NonPointerValue(reflect.ValueOf(structSlice))
	sz := svnp.Len()
	struTyp := reflectx.NonPointerType(reflect.TypeOf(structSlice).Elem().Elem())
	fld, ok := struTyp.FieldByName(fieldName)
	if !ok {
		err := fmt.Errorf("StructSliceRowByValue: field name: %v not found", fieldName)
		slog.Error(err.Error())
		return -1, err
	}
	fldIndex := fld.Index
	for idx := 0; idx < sz; idx++ {
		rval := reflectx.UnderlyingPointer(svnp.Index(idx))
		fval := rval.Elem().FieldByIndex(fldIndex)
		if !fval.IsValid() {
			continue
		}
		if fval.Interface() == fieldValue {
			return idx, nil
		}
	}
	return -1, nil
}

func (tb *Table) EditIndex(idx int) {
	if idx < 0 || idx >= tb.SliceUnderlying.Len() {
		return
	}
	val := reflectx.UnderlyingPointer(tb.SliceUnderlying.Index(idx))
	stru := val.Interface()
	tynm := reflectx.NonPointerType(val.Type()).Name()
	lbl := labels.ToLabel(stru)
	if lbl != "" {
		tynm += ": " + lbl
	}
	d := NewBody().AddTitle(tynm)
	NewForm(d).SetStruct(stru).SetReadOnly(tb.IsReadOnly())
	d.AddBottomBar(func(parent Widget) {
		d.AddCancel(parent)
		d.AddOK(parent)
	})
	d.RunFullDialog(tb)
}

func (tb *Table) ContextMenu(m *Scene) {
	if !tb.isArray {
		NewButton(m).SetText("Edit").SetIcon(icons.Edit).
			OnClick(func(e events.Event) {
				tb.EditIndex(tb.SelectedIndex)
			})
	}
}

//////////////////////////////////////////////////////
// 	Header layout

func (tb *Table) SizeFinal() {
	tb.ListBase.SizeFinal()
	sg := tb.This.(Lister).SliceGrid()
	if sg == nil {
		return
	}
	sh := tb.SliceHeader()
	sh.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		_, sgb := AsWidget(sg.Child(i))
		gsz := &sgb.Geom.Size
		ksz := &kwb.Geom.Size
		ksz.Actual.Total.X = gsz.Actual.Total.X
		ksz.Actual.Content.X = gsz.Actual.Content.X
		ksz.Alloc.Total.X = gsz.Alloc.Total.X
		ksz.Alloc.Content.X = gsz.Alloc.Content.X
		return tree.Continue
	})
	gsz := &sg.Geom.Size
	ksz := &sh.Geom.Size
	ksz.Actual.Total.X = gsz.Actual.Total.X
	ksz.Actual.Content.X = gsz.Actual.Content.X
	ksz.Alloc.Total.X = gsz.Alloc.Total.X
	ksz.Alloc.Content.X = gsz.Alloc.Content.X
}
