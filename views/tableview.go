// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

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
	"cogentcore.org/core/core"
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

// TableView represents a slice of structs as a table, where the fields are
// the columns and the elements are the rows. It is a full-featured editor with
// multiple-selection, cut-and-paste, and drag-and-drop.
// Use [SliceViewBase.BindSelect] to make the table view designed for item selection.
type TableView struct {
	SliceViewBase

	// StyleFunc is an optional styling function.
	StyleFunc TableViewStyleFunc `copier:"-" view:"-" json:"-" xml:"-"`

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

// TableViewStyleFunc is a styling function for custom styling and
// configuration of elements in the table view.
type TableViewStyleFunc func(w core.Widget, s *styles.Style, row, col int)

func (tv *TableView) OnInit() {
	tv.SliceViewBase.OnInit()
	tv.AddContextMenu(tv.ContextMenu)
	tv.SortIndex = -1

	tv.Makers[0] = func(p *core.Plan) { // TODO: reduce redundancy with SliceViewBase Maker
		svi := tv.This().(SliceViewer)
		svi.UpdateSliceSize()

		tv.ViewMuLock()
		defer tv.ViewMuUnlock()

		tv.SortSlice()

		scrollTo := -1
		if tv.SelectedField != "" && tv.SelectedValue != nil {
			tv.SelectedIndex, _ = StructSliceIndexByValue(tv.Slice, tv.SelectedField, tv.SelectedValue)
			tv.SelectedField = ""
			tv.SelectedValue = nil
			tv.InitSelectedIndex = -1
			scrollTo = tv.SelectedIndex
		} else if tv.InitSelectedIndex >= 0 {
			tv.SelectedIndex = tv.InitSelectedIndex
			tv.InitSelectedIndex = -1
			scrollTo = tv.SelectedIndex
		}
		if scrollTo >= 0 {
			tv.ScrollToIndex(scrollTo)
		}
		tv.UpdateStartIndex()

		tv.MakeHeader(p)
		tv.MakeGrid(p, func(p *core.Plan) {
			svi.UpdateMaxWidths()

			for i := 0; i < tv.VisRows; i++ {
				si := tv.StartIndex + i
				svi.MakeRow(p, i, si)
			}
		})
	}
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

// SetSlice sets the source slice that we are viewing.
// Must call Update if already open.
func (tv *TableView) SetSlice(sl any) *TableView {
	if reflectx.AnyIsNil(sl) {
		tv.Slice = nil
		return tv
	}
	if tv.Slice == sl {
		tv.MakeIter = 0
		return tv
	}

	slpTyp := reflect.TypeOf(sl)
	if slpTyp.Kind() != reflect.Pointer {
		slog.Error("TableView requires that you pass a pointer to a slice of struct elements, but type is not a Ptr", "type", slpTyp)
		return tv
	}
	if slpTyp.Elem().Kind() != reflect.Slice {
		slog.Error("TableView requires that you pass a pointer to a slice of struct elements, but ptr doesn't point to a slice", "type", slpTyp.Elem())
		return tv
	}
	eltyp := reflectx.NonPointerType(reflectx.SliceElementType(sl))
	if eltyp.Kind() != reflect.Struct {
		slog.Error("TableView requires that you pass a slice of struct elements, but type is not a Struct", "type", eltyp.String())
		return tv
	}

	tv.SetSliceBase()
	tv.Slice = sl
	tv.SliceUnderlying = reflectx.Underlying(reflect.ValueOf(tv.Slice))
	tv.ElementValue = reflectx.Underlying(reflectx.SliceElementValue(sl))
	tv.cacheVisibleFields()
	return tv
}

// cacheVisibleFields caches the visible struct fields.
func (tv *TableView) cacheVisibleFields() {
	tv.visibleFields = make([]reflect.StructField, 0)
	shouldShow := func(field reflect.StructField) bool {
		tvtag := field.Tag.Get("tableview")
		switch {
		case tvtag == "+":
			return true
		case tvtag == "-":
			return false
		case tvtag == "-select" && tv.IsReadOnly():
			return false
		case tvtag == "-edit" && !tv.IsReadOnly():
			return false
		default:
			return field.Tag.Get("view") != "-"
		}
	}

	reflectx.WalkFields(tv.ElementValue,
		func(parent reflect.Value, field reflect.StructField, value reflect.Value) bool {
			return shouldShow(field)
		},
		func(parent reflect.Value, field reflect.StructField, value reflect.Value) {
			tv.visibleFields = append(tv.visibleFields, field)
		})
	tv.numVisibleFields = len(tv.visibleFields)
	tv.headerWidths = make([]int, tv.numVisibleFields)
	tv.colMaxWidths = make([]int, tv.numVisibleFields)
}

func (tv *TableView) UpdateMaxWidths() {
	if tv.SliceSize == 0 {
		return
	}
	for fli := 0; fli < tv.numVisibleFields; fli++ {
		field := tv.visibleFields[fli]
		tv.colMaxWidths[fli] = 0
		val := tv.SliceElementValue(0)
		fval := val.FieldByIndex(field.Index)
		// _, isicon := vv.(*IconValue)
		isicon := false
		isString := fval.Type().Kind() == reflect.String
		if !isString || isicon {
			continue
		}
		mxw := 0
		for rw := 0; rw < tv.SliceSize; rw++ {
			val := tv.SliceElementValue(rw)
			str := reflectx.ToString(val.FieldByIndex(field.Index).Interface())
			mxw = max(mxw, len(str))
		}
		tv.colMaxWidths[fli] = mxw
	}
}

// SliceHeader returns the Frame header for slice grid
func (tv *TableView) SliceHeader() *core.Frame {
	return tv.Child(0).(*core.Frame)
}

func (tv *TableView) MakeHeader(p *core.Plan) {
	core.AddAt(p, "header", func(w *core.Frame) {
		core.ToolbarStyles(w)
		w.Style(func(s *styles.Style) {
			s.Grow.Set(0, 0)
			s.Gap.Set(units.Em(0.5)) // matches grid default
		})
		w.Maker(func(p *core.Plan) {
			if tv.Is(SliceViewShowIndex) {
				core.AddAt(p, "head-index", func(w *core.Text) {
					w.SetType(core.TextBodyMedium)
					w.Style(func(s *styles.Style) {
						s.Align.Self = styles.Center
					})
					w.SetText("Index")
				})
			}
			for fli := 0; fli < tv.numVisibleFields; fli++ {
				field := tv.visibleFields[fli]
				core.AddAt(p, "head-"+field.Name, func(w *core.Button) {
					w.SetType(core.ButtonMenu)
					w.OnClick(func(e events.Event) {
						tv.SortSliceAction(fli)
					})
					w.Builder(func() {
						htxt := ""
						if lbl, ok := field.Tag.Lookup("label"); ok {
							htxt = lbl
						} else {
							htxt = strcase.ToSentence(field.Name)
						}
						w.SetText(htxt)
						w.Tooltip = htxt + " (click to sort by)"
						doc, ok := types.GetDoc(reflect.Value{}, tv.ElementValue, field, htxt)
						if ok && doc != "" {
							w.Tooltip += ": " + doc
						}
						tv.headerWidths[fli] = len(htxt)
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

// RowWidgetNs returns number of widgets per row and offset for index label
func (tv *TableView) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 1 + tv.numVisibleFields
	idxOff = 1
	if !tv.Is(SliceViewShowIndex) {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

func (tv *TableView) MakeRow(p *core.Plan, i, si int) {
	svi := tv.This().(SliceViewer)
	itxt := strconv.Itoa(i)
	sitxt := strconv.Itoa(si)
	invis := si >= tv.SliceSize
	val := tv.SliceElementValue(si)
	// stru := val.Interface()

	if tv.Is(SliceViewShowIndex) {
		tv.MakeGridIndex(p, i, si, itxt, invis)
	}

	vpath := tv.ViewPath + "[" + sitxt + "]"
	if si < tv.SliceSize {
		if lblr, ok := tv.Slice.(labels.SliceLabeler); ok {
			slbl := lblr.ElemLabel(si)
			if slbl != "" {
				vpath = JoinViewPath(tv.ViewPath, slbl)
			}
		}
	}
	_ = vpath

	for fli := 0; fli < tv.numVisibleFields; fli++ {
		field := tv.visibleFields[fli]
		fval := reflectx.OnePointerValue(val.FieldByIndex(field.Index))
		valnm := fmt.Sprintf("value-%v.%v", fli, itxt)
		tags := reflect.StructTag("")
		if fval.Kind() == reflect.Slice || fval.Kind() == reflect.Map {
			tags = `view:"no-inline"`
		}

		core.AddNew(p, valnm, func() core.Value {
			return core.NewValue(fval.Interface(), tags)
		}, func(w core.Value) {
			wb := w.AsWidget()
			tv.MakeValue(w, i)
			// vv.SetStructValue(fval.Addr(), stru, &field, vpath)
			w.SetProperty(SliceViewColProperty, fli)
			if !tv.IsReadOnly() {
				wb.OnChange(func(e events.Event) {
					tv.SendChange()
				})
				wb.OnInput(tv.HandleEvent)
			}
			wb.Builder(func() {
				// w.SetSliceValue(val, sv.Slice, si, sv.ViewPath)
				si := tv.StartIndex + i
				val := tv.SliceElementValue(si)
				fval := reflectx.OnePointerValue(val.FieldByIndex(field.Index))
				core.Bind(fval.Interface(), w)
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
	}
}

func (tv *TableView) HasStyleFunc() bool {
	return tv.StyleFunc != nil
}

func (tv *TableView) StyleRow(w core.Widget, idx, fidx int) {
	if tv.StyleFunc != nil {
		tv.StyleFunc(w, &w.AsWidget().Styles, idx, fidx)
	}
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (tv *TableView) SliceNewAt(idx int) {
	tv.ViewMuLock()

	tv.SliceNewAtSelect(idx)
	reflectx.SliceNewAt(tv.Slice, idx)
	if idx < 0 {
		idx = tv.SliceSize
	}

	tv.This().(SliceViewer).UpdateSliceSize()
	tv.SelectIndexAction(idx, events.SelectOne)
	tv.ViewMuUnlock()
	tv.SendChange()
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

	reflectx.SliceDeleteAt(tv.Slice, idx)

	tv.This().(SliceViewer).UpdateSliceSize()
	tv.ViewMuUnlock()
	tv.SendChange()
	tv.Update()
}

// SortSlice sorts the slice according to current settings
func (tv *TableView) SortSlice() {
	if tv.SortIndex < 0 || tv.SortIndex >= len(tv.visibleFields) {
		return
	}
	rawIndex := tv.visibleFields[tv.SortIndex].Index
	reflectx.StructSliceSort(tv.Slice, rawIndex, !tv.SortDescending)
}

// SortSliceAction sorts the slice for given field index -- toggles ascending
// vs. descending if already sorting on this dimension
func (tv *TableView) SortSliceAction(fldIndex int) {
	sgh := tv.SliceHeader()
	_, idxOff := tv.RowWidgetNs()

	ascending := true

	for fli := 0; fli < tv.numVisibleFields; fli++ {
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
	tv.SortSlice()
	tv.Update()
}

// SortFieldName returns the name of the field being sorted, along with :up or
// :down depending on descending
func (tv *TableView) SortFieldName() string {
	if tv.SortIndex >= 0 && tv.SortIndex < tv.numVisibleFields {
		nm := tv.visibleFields[tv.SortIndex].Name
		if tv.SortDescending {
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
func (tv *TableView) SetSortFieldName(nm string) {
	if nm == "" {
		return
	}
	spnm := strings.Split(nm, ":")
	got := false
	for fli := 0; fli < tv.numVisibleFields; fli++ {
		fld := tv.visibleFields[fli]
		if fld.Name == spnm[0] {
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
	if got {
		tv.SortSlice()
	}
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
	for fli := 0; fli < tv.numVisibleFields; fli++ {
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
	if !tv.IsRowInBounds(row) || tv.Is(SliceViewInFocusGrab) { // range check
		return nil
	}
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := tv.SliceGrid()
	// first check if we already have focus
	for fli := 0; fli < tv.numVisibleFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
		if w.StateIs(states.Focused) || w.ContainsFocus() {
			return w
		}
	}
	tv.SetFlag(true, SliceViewInFocusGrab)
	defer func() { tv.SetFlag(false, SliceViewInFocusGrab) }()
	for fli := 0; fli < tv.numVisibleFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(core.Widget).AsWidget()
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
func (tv *TableView) SelectFieldVal(fld, val string) bool {
	tv.SelectedField = fld
	tv.SelectedValue = val
	if tv.SelectedField != "" && tv.SelectedValue != nil {
		idx, _ := StructSliceIndexByValue(tv.Slice, tv.SelectedField, tv.SelectedValue)
		if idx >= 0 {
			tv.ScrollToIndex(idx)
			tv.UpdateSelectIndex(idx, true, events.SelectOne)
			return true
		}
	}
	return false
}

// StructSliceIndexByValue searches for first index that contains given value in field of
// given name.
func StructSliceIndexByValue(struSlice any, fldName string, fldVal any) (int, error) {
	svnp := reflectx.NonPointerValue(reflect.ValueOf(struSlice))
	sz := svnp.Len()
	struTyp := reflectx.NonPointerType(reflect.TypeOf(struSlice).Elem().Elem())
	fld, ok := struTyp.FieldByName(fldName)
	if !ok {
		err := fmt.Errorf("core.StructSliceRowByValue: field name: %v not found", fldName)
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
		if fval.Interface() == fldVal {
			return idx, nil
		}
	}
	return -1, nil
}

func (tv *TableView) EditIndex(idx int) {
	if idx < 0 || idx >= tv.SliceUnderlying.Len() {
		return
	}
	val := reflectx.UnderlyingPointer(tv.SliceUnderlying.Index(idx))
	stru := val.Interface()
	tynm := reflectx.NonPointerType(val.Type()).Name()
	lbl := labels.ToLabel(stru)
	if lbl != "" {
		tynm += ": " + lbl
	}
	d := core.NewBody().AddTitle(tynm)
	NewStructView(d).SetStruct(stru).SetReadOnly(tv.IsReadOnly())
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent)
	})
	d.RunFullDialog(tv)
}

func (tv *TableView) ContextMenu(m *core.Scene) {
	if !tv.Is(SliceViewIsArray) {
		core.NewButton(m).SetText("Edit").SetIcon(icons.Edit).
			OnClick(func(e events.Event) {
				tv.EditIndex(tv.SelectedIndex)
			})
	}
}

//////////////////////////////////////////////////////
// 	Header layout

func (tv *TableView) SizeFinal() {
	tv.SliceViewBase.SizeFinal()
	sg := tv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	sh := tv.SliceHeader()
	sh.WidgetKidsIter(func(i int, kwi core.Widget, kwb *core.WidgetBase) bool {
		_, sgb := core.AsWidget(sg.Child(i))
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
