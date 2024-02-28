// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"image"
	"log/slog"
	"reflect"
	"strconv"
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/states"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// todo:
// * search option, both as a search field and as simple type-to-search

// TableView represents a slice-of-structs as a table, where the fields are
// the columns, within an overall frame.  It is a full-featured editor with
// multiple-selection, cut-and-paste, and drag-and-drop.
// If ReadOnly, it functions as a mutually-exclusive item
// selector, highlighting the selected row and emitting a Selected action.
type TableView struct {
	SliceViewBase

	// optional styling function
	StyleFunc TableViewStyleFunc `copier:"-" view:"-" json:"-" xml:"-"`

	// current selection field -- initially select value in this field
	SelField string `copier:"-" view:"-" json:"-" xml:"-"`

	// current sort index
	SortIdx int

	// whether current sort order is descending
	SortDesc bool

	// struct type for each row
	StruType reflect.Type `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// the visible fields
	VisFields []reflect.StructField `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// number of visible fields
	NVisFields int `set:"-" copier:"-" view:"-" json:"-" xml:"-"`

	// HeaderWidths has number of characters in each header, per visfields
	HeaderWidths []int `set:"-" copier:"-" json:"-" xml:"-"`

	// ColMaxWidths records maximum width in chars of string type fields
	ColMaxWidths []int `set:"-" copier:"-" json:"-" xml:"-"`
}

// check for interface impl
var _ SliceViewer = (*TableView)(nil)

// TableViewStyleFunc is a styling function for custom styling and
// configuration of elements in the table view.
type TableViewStyleFunc func(w gi.Widget, s *styles.Style, row, col int)

func (tv *TableView) OnInit() {
	tv.Frame.OnInit()
	tv.SliceViewBase.HandleEvents()
	tv.SetStyles()
	tv.AddContextMenu(tv.SliceViewBase.ContextMenu)
	tv.AddContextMenu(tv.ContextMenu)
}

func (tv *TableView) SetStyles() {
	tv.SliceViewBase.SetStyles() // handles all the basics
	tv.SortIdx = -1

	// we only have to handle the header
	tv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(tv) {
		case "header": // slice header
			sh := w.(*gi.Frame)
			gi.ToolbarStyles(sh)
			sh.Style(func(s *styles.Style) {
				s.Grow.Set(0, 0)
				s.Gap.Set(units.Em(0.5)) // matches grid default
			})
		case "header/head-idx": // index header
			lbl := w.(*gi.Label)
			lbl.SetText("Index").SetType(gi.LabelBodyMedium)
			w.Style(func(s *styles.Style) {
				s.Align.Self = styles.Center
			})
		}
		if w.Parent().PathFrom(tv) == "header" {
			w.Style(func(s *styles.Style) {
				if hdr, ok := w.(*gi.Button); ok {
					fli := hdr.Prop("field-index").(int)
					if fli == tv.SortIdx {
						if tv.SortDesc {
							hdr.SetIcon(icons.KeyboardArrowDown)
						} else {
							hdr.SetIcon(icons.KeyboardArrowUp)
						}
					}
				}
			})
		}
	})
}

// StyleValueWidget performs additional value widget styling
func (tv *TableView) StyleValueWidget(w gi.Widget, s *styles.Style, row, col int) {
	hw := float32(tv.HeaderWidths[col])
	if col == tv.SortIdx {
		hw += 6
	}
	if len(tv.ColMaxWidths) > col {
		hw = max(float32(tv.ColMaxWidths[col]), hw)
	}
	hv := units.Ch(hw)
	s.Min.X.Val = max(s.Min.X.Val, hv.Convert(s.Min.X.Un, &s.UnitContext).Val)
	s.SetTextWrap(false)
}

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice (does Update if already viewing).
func (tv *TableView) SetSlice(sl any) *TableView {
	if laser.AnyIsNil(sl) {
		tv.Slice = nil
		return tv
	}
	if tv.Slice == sl && tv.Is(SliceViewConfigured) {
		tv.Update()
		return tv
	}

	slpTyp := reflect.TypeOf(sl)
	if slpTyp.Kind() != reflect.Ptr {
		slog.Error("TableView requires that you pass a pointer to a slice of struct elements, but type is not a Ptr", "type", slpTyp)
		return tv
	}
	if slpTyp.Elem().Kind() != reflect.Slice {
		slog.Error("TableView requires that you pass a pointer to a slice of struct elements, but ptr doesn't point to a slice", "type", slpTyp.Elem())
		return tv
	}
	eltyp := laser.NonPtrType(laser.SliceElType(sl))
	if eltyp.Kind() != reflect.Struct {
		slog.Error("TableView requires that you pass a slice of struct elements, but type is not a Struct", "type", eltyp.String())
		return tv
	}

	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	tv.SetSliceBase()
	tv.Slice = sl
	tv.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(tv.Slice))
	tv.ElVal = laser.OnePtrValue(laser.SliceElValue(sl))
	tv.CacheVisFields()
	tv.Update()
	return tv
}

// StructType sets the StruType and returns the type of the struct within the
// slice -- this is a non-ptr type even if slice has pointers to structs
func (tv *TableView) StructType() reflect.Type {
	tv.StruType = laser.NonPtrType(laser.SliceElType(tv.Slice))
	return tv.StruType
}

// CacheVisFields computes the number of visible fields in nVisFields and
// caches those to skip in fieldSkip
func (tv *TableView) CacheVisFields() {
	styp := tv.StructType()
	tv.VisFields = make([]reflect.StructField, 0)
	shouldShow := func(fld reflect.StructField) bool {
		if !fld.IsExported() {
			return false
		}
		tvtag := fld.Tag.Get("tableview")
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
			return fld.Tag.Get("view") != "-"
		}
	}
	laser.FlatFieldsTypeFuncIf(styp,
		func(typ reflect.Type, fld reflect.StructField) bool {
			return shouldShow(fld)
		},
		func(typ reflect.Type, fld reflect.StructField) bool {
			if !shouldShow(fld) {
				return true
			}
			if typ != styp {
				rfld, has := styp.FieldByName(fld.Name)
				if has {
					tv.VisFields = append(tv.VisFields, rfld)
				} else {
					fmt.Printf("TableView: Field name: %v is ambiguous from base struct type: %v, cannot be used in view!\n", fld.Name, styp.String())
				}
			} else {
				tv.VisFields = append(tv.VisFields, fld)
			}
			return true
		})
	tv.NVisFields = len(tv.VisFields)
}

// ConfigWidget configures the view
func (tv *TableView) ConfigWidget() {
	tv.ConfigTableView()
}

func (tv *TableView) ConfigTableView() {
	if tv.Is(SliceViewConfigured) {
		tv.This().(SliceViewer).UpdateWidgets()
		return
	}
	updt := tv.UpdateStart()
	tv.SortSlice()
	tv.ConfigFrame()
	tv.This().(SliceViewer).ConfigRows()
	tv.This().(SliceViewer).UpdateWidgets()
	tv.ApplyStyleTree()
	tv.UpdateEndLayout(updt)
}

func (tv *TableView) ConfigFrame() {
	if tv.HasChildren() {
		return
	}
	tv.SetFlag(true, SliceViewConfigured)
	gi.NewFrame(tv, "header")
	NewSliceViewGrid(tv, "grid")
	tv.ConfigHeader()
}

func (tv *TableView) ConfigHeader() {
	sgh := tv.SliceHeader()
	if sgh.HasChildren() || tv.NVisFields == 0 {
		return
	}
	hcfg := ki.Config{}
	if tv.Is(SliceViewShowIndex) {
		hcfg.Add(gi.LabelType, "head-idx")
	}
	tv.HeaderWidths = make([]int, tv.NVisFields)
	tv.ColMaxWidths = make([]int, tv.NVisFields)
	for fli := 0; fli < tv.NVisFields; fli++ {
		fld := tv.VisFields[fli]
		labnm := "head-" + fld.Name
		hcfg.Add(gi.ButtonType, labnm)
	}
	sgh.ConfigChildren(hcfg) // headers SHOULD be unique, but with labels..
	_, idxOff := tv.RowWidgetNs()
	nfld := tv.NVisFields
	for fli := 0; fli < nfld; fli++ {
		fli := fli
		field := tv.VisFields[fli]
		hdr := sgh.Child(idxOff + fli).(*gi.Button)
		hdr.SetType(gi.ButtonMenu)
		htxt := ""
		if lbl, ok := field.Tag.Lookup("label"); ok {
			htxt = lbl
		} else {
			htxt = strcase.ToSentence(field.Name)
		}
		hdr.SetText(htxt)
		tv.HeaderWidths[fli] = len(htxt)
		hdr.SetProp("field-index", fli)
		if fli == tv.SortIdx {
			if tv.SortDesc {
				hdr.SetIcon(icons.KeyboardArrowDown)
			} else {
				hdr.SetIcon(icons.KeyboardArrowUp)
			}
		}
		hdr.Tooltip = hdr.Text + " (tap to sort by)"
		doc, ok := gti.GetDoc(reflect.Value{}, tv.ElVal, &field, hdr.Text)
		if ok && doc != "" {
			hdr.Tooltip += ": " + doc
		}
		hdr.OnClick(func(e events.Event) {
			tv.SortSliceAction(fli)
		})
	}
}

// SliceHeader returns the Frame header for slice grid
func (tv *TableView) SliceHeader() *gi.Frame {
	return tv.Child(0).(*gi.Frame)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (tv *TableView) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 1 + tv.NVisFields
	idxOff = 1
	if !tv.Is(SliceViewShowIndex) {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

// ConfigRows configures VisRows worth of widgets
// to display slice data.  It should only be called
// when NeedsConfigRows is true: when VisRows changes.
func (tv *TableView) ConfigRows() {
	sg := tv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	tv.SetFlag(true, SliceViewConfigured)
	sg.SetFlag(true, gi.LayoutNoKeys)

	tv.ViewMuLock()
	defer tv.ViewMuUnlock()

	sg.DeleteChildren(ki.DestroyKids)
	tv.Values = nil

	tv.This().(SliceViewer).UpdtSliceSize()

	if tv.IsNil() {
		return
	}

	nWidgPerRow, idxOff := tv.RowWidgetNs()
	nWidg := nWidgPerRow * tv.VisRows
	sg.Styles.Columns = nWidgPerRow

	tv.Values = make([]Value, tv.NVisFields*tv.VisRows)
	sg.Kids = make(ki.Slice, nWidg)

	for i := 0; i < tv.VisRows; i++ {
		i := i
		si := i
		ridx := i * nWidgPerRow
		var val reflect.Value
		if si < tv.SliceSize {
			if tv.SliceNPVal.Index(si).IsValid() {
				val = laser.OnePtrUnderlyingValue(tv.SliceNPVal.Index(si)) // deal with pointer lists
			}
		} else {
			val = tv.ElVal
		}
		if !val.IsValid() {
			continue
		}
		val = tv.ElVal

		stru := val.Interface()

		idxlab := &gi.Label{}
		itxt := strconv.Itoa(i)
		sitxt := strconv.Itoa(si)
		labnm := "index-" + itxt
		if tv.Is(SliceViewShowIndex) {
			idxlab = &gi.Label{}
			sg.SetChild(idxlab, ridx, labnm)
			idxlab.SetText(sitxt)
			idxlab.OnSelect(func(e events.Event) {
				e.SetHandled()
				tv.UpdateSelectRow(i, e.SelectMode())
			})
			idxlab.SetProp(SliceViewRowProp, i)
		}

		vpath := tv.ViewPath + "[" + sitxt + "]"
		for fli := 0; fli < tv.NVisFields; fli++ {
			fli := fli
			field := tv.VisFields[fli]
			fval := val.Elem().FieldByIndex(field.Index)
			vvi := i*tv.NVisFields + fli
			tags := ""
			if fval.Kind() == reflect.Slice || fval.Kind() == reflect.Map {
				tags = `view:"no-inline"`
			}
			vv := ToValue(fval.Interface(), tags)
			tv.Values[vvi] = vv
			vv.SetStructValue(fval.Addr(), stru, &field, tv.TmpSave, vpath)
			vv.SetReadOnly(tv.IsReadOnly())

			vtyp := vv.WidgetType()
			valnm := fmt.Sprintf("value-%v.%v", fli, itxt)
			cidx := ridx + idxOff + fli
			w := ki.NewOfType(vtyp).(gi.Widget)
			sg.SetChild(w, cidx, valnm)
			vv.ConfigWidget(w)
			w.SetProp(SliceViewRowProp, i)
			w.SetProp(SliceViewColProp, fli)

			if !tv.IsReadOnly() {
				vvb := vv.AsValueBase()
				vvb.OnChange(func(e events.Event) {
					tv.SetChanged()
				})
				vvb.AsWidgetBase().OnInput(tv.HandleEvent)
			}
			if i == 0 && tv.SliceSize > 0 {
				tv.ColMaxWidths[fli] = 0
				_, isicon := vv.(*IconValue)
				if !isicon && fval.Kind() == reflect.String {
					mxw := 0
					for rw := 0; rw < tv.SliceSize; rw++ {
						sval := laser.OnePtrUnderlyingValue(tv.SliceNPVal.Index(rw))
						fval := sval.Elem().FieldByIndex(field.Index)
						str := fval.String()
						mxw = max(mxw, len(str))
					}
					tv.ColMaxWidths[fli] = mxw
				}
			}
		}
	}
	tv.ConfigTree()
	tv.ApplyStyleTree()
}

// UpdateWidgets updates the row widget display to
// represent the current state of the slice data,
// including which range of data is being displayed.
// This is called for scrolling, navigation etc.
func (tv *TableView) UpdateWidgets() {
	sg := tv.This().(SliceViewer).SliceGrid()
	if sg == nil || tv.VisRows == 0 || sg.VisRows == 0 || !sg.HasChildren() {
		return
	}

	updt := sg.UpdateStart()
	defer sg.UpdateEndRender(updt)

	tv.ViewMuLock()
	defer tv.ViewMuUnlock()

	tv.This().(SliceViewer).UpdtSliceSize()

	nWidgPerRow, idxOff := tv.RowWidgetNs()

	scrollTo := -1
	if tv.SelField != "" && tv.SelVal != nil {
		tv.SelIdx, _ = StructSliceIdxByValue(tv.Slice, tv.SelField, tv.SelVal)
		tv.SelField = ""
		tv.SelVal = nil
		tv.InitSelIdx = -1
		scrollTo = tv.SelIdx
	} else if tv.InitSelIdx >= 0 {
		tv.SelIdx = tv.InitSelIdx
		tv.InitSelIdx = -1
		scrollTo = tv.SelIdx
	}
	if scrollTo >= 0 {
		tv.ScrollToIdx(scrollTo)
	}
	tv.UpdateStartIdx()

	for i := 0; i < tv.VisRows; i++ {
		i := i
		ridx := i * nWidgPerRow
		si := tv.StartIdx + i // slice idx
		invis := si >= tv.SliceSize

		var idxlab *gi.Label
		if tv.Is(SliceViewShowIndex) {
			if len(sg.Kids) == 0 {
				break
			}
			if sg.Kids[ridx] == nil {
				continue
			}
			idxlab = sg.Kids[ridx].(*gi.Label)
			idxlab.SetTextUpdate(strconv.Itoa(si))
			idxlab.SetState(invis, states.Invisible)
		}

		sitxt := strconv.Itoa(si)
		vpath := tv.ViewPath + "[" + sitxt + "]"
		if si < tv.SliceSize {
			if lblr, ok := tv.Slice.(gi.SliceLabeler); ok {
				slbl := lblr.ElemLabel(si)
				if slbl != "" {
					vpath = JoinViewPath(tv.ViewPath, slbl)
				}
			}
		}
		for fli := 0; fli < tv.NVisFields; fli++ {
			fli := fli
			field := tv.VisFields[fli]
			cidx := ridx + idxOff + fli
			if len(sg.Kids) < cidx {
				break
			}
			if sg.Kids[cidx] == nil {
				return
			}
			w := sg.Kids[cidx].(gi.Widget)
			wb := w.AsWidget()

			var val reflect.Value
			if !invis {
				val = laser.OnePtrUnderlyingValue(tv.SliceNPVal.Index(si)) // deal with pointer lists
				if val.IsZero() {
					val = tv.ElVal
				}
			} else {
				val = tv.ElVal
			}
			stru := val.Interface()
			fval := val.Elem().FieldByIndex(field.Index)
			vvi := i*tv.NVisFields + fli
			vv := tv.Values[vvi]
			vv.SetStructValue(fval.Addr(), stru, &field, tv.TmpSave, vpath)
			vv.SetReadOnly(tv.IsReadOnly())
			vv.UpdateWidget()
			w.SetState(invis, states.Invisible)
			if !invis {
				if tv.IsReadOnly() {
					wb.SetReadOnly(true)
				}
			} else {
				wb.SetSelected(false)
				if tv.Is(SliceViewShowIndex) {
					idxlab.SetSelected(false)
				}
			}
			if tv.This().(SliceViewer).HasStyleFunc() {
				w.ApplyStyle()
			}
		}
	}
}

func (tv *TableView) HasStyleFunc() bool {
	return tv.StyleFunc != nil
}

func (tv *TableView) StyleRow(w gi.Widget, idx, fidx int) {
	if tv.StyleFunc != nil {
		tv.StyleFunc(w, &w.AsWidget().Styles, idx, fidx)
	}
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (tv *TableView) SliceNewAt(idx int) {
	tv.ViewMuLock()
	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	tv.SliceNewAtSel(idx)
	laser.SliceNewAt(tv.Slice, idx)
	if idx < 0 {
		idx = tv.SliceSize
	}

	tv.This().(SliceViewer).UpdtSliceSize()
	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.SelectIdxAction(idx, events.SelectOne)
	tv.ViewMuUnlock()
	tv.SetChanged()
	tv.This().(SliceViewer).UpdateWidgets()
	tv.IdxGrabFocus(idx)
}

// SliceDeleteAt deletes element at given index from slice
func (tv *TableView) SliceDeleteAt(idx int) {
	if idx < 0 || idx >= tv.SliceSize {
		return
	}
	tv.ViewMuLock()
	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	tv.SliceDeleteAtSel(idx)

	laser.SliceDeleteAt(tv.Slice, idx)

	tv.This().(SliceViewer).UpdtSliceSize()

	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.ViewMuUnlock()
	tv.SetChanged()
	tv.This().(SliceViewer).UpdateWidgets()
}

// SortSlice sorts the slice according to current settings
func (tv *TableView) SortSlice() {
	if tv.SortIdx < 0 || tv.SortIdx >= len(tv.VisFields) {
		return
	}
	rawIdx := tv.VisFields[tv.SortIdx].Index
	laser.StructSliceSort(tv.Slice, rawIdx, !tv.SortDesc)
}

// SortSliceAction sorts the slice for given field index -- toggles ascending
// vs. descending if already sorting on this dimension
func (tv *TableView) SortSliceAction(fldIdx int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	sgh := tv.SliceHeader()
	_, idxOff := tv.RowWidgetNs()

	ascending := true

	for fli := 0; fli < tv.NVisFields; fli++ {
		hdr := sgh.Child(idxOff + fli).(*gi.Button)
		hdr.SetType(gi.ButtonAction)
		if fli == fldIdx {
			if tv.SortIdx == fli {
				tv.SortDesc = !tv.SortDesc
				ascending = !tv.SortDesc
			} else {
				tv.SortDesc = false
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

	tv.SortIdx = fldIdx
	tv.SortSlice()
	sgh.Update() // requires full update due to sort button icon
	tv.UpdateWidgets()
}

// SortFieldName returns the name of the field being sorted, along with :up or
// :down depending on descending
func (tv *TableView) SortFieldName() string {
	if tv.SortIdx >= 0 && tv.SortIdx < tv.NVisFields {
		nm := tv.VisFields[tv.SortIdx].Name
		if tv.SortDesc {
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
	for fli := 0; fli < tv.NVisFields; fli++ {
		fld := tv.VisFields[fli]
		if fld.Name == spnm[0] {
			got = true
			// fmt.Println("sorting on:", fld.Name, fli, "from:", nm)
			tv.SortIdx = fli
		}
	}
	if len(spnm) == 2 {
		if spnm[1] == "down" {
			tv.SortDesc = true
		} else {
			tv.SortDesc = false
		}
	}
	if got {
		tv.SortSlice()
	}
}

// RowFirstVisWidget returns the first visible widget for given row (could be
// index or not) -- false if out of range
func (tv *TableView) RowFirstVisWidget(row int) (*gi.WidgetBase, bool) {
	if !tv.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	sg := tv.SliceGrid()
	w := sg.Kids[row*nWidgPerRow].(gi.Widget).AsWidget()
	if w.Geom.TotalBBox != (image.Rectangle{}) {
		return w, true
	}
	ridx := nWidgPerRow * row
	for fli := 0; fli < tv.NVisFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(gi.Widget).AsWidget()
		if w.Geom.TotalBBox != (image.Rectangle{}) {
			return w, true
		}
	}
	return nil, false
}

// RowGrabFocus grabs the focus for the first focusable widget in given row --
// returns that element or nil if not successful -- note: grid must have
// already rendered for focus to be grabbed!
func (tv *TableView) RowGrabFocus(row int) *gi.WidgetBase {
	if !tv.IsRowInBounds(row) || tv.Is(SliceViewInFocusGrab) { // range check
		return nil
	}
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := tv.SliceGrid()
	// first check if we already have focus
	for fli := 0; fli < tv.NVisFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(gi.Widget).AsWidget()
		if w.StateIs(states.Focused) || w.ContainsFocus() {
			return w
		}
	}
	tv.SetFlag(true, SliceViewInFocusGrab)
	defer func() { tv.SetFlag(false, SliceViewInFocusGrab) }()
	for fli := 0; fli < tv.NVisFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(gi.Widget).AsWidget()
		if w.CanFocus() {
			w.SetFocusEvent()
			return w
		}
	}
	return nil
}

// SelectFieldVal sets SelField and SelVal and attempts to find corresponding
// row, setting SelectedIdx and selecting row if found -- returns true if
// found, false otherwise
func (tv *TableView) SelectFieldVal(fld, val string) bool {
	tv.SelField = fld
	tv.SelVal = val
	if tv.SelField != "" && tv.SelVal != nil {
		idx, _ := StructSliceIdxByValue(tv.Slice, tv.SelField, tv.SelVal)
		if idx >= 0 {
			tv.ScrollToIdx(idx)
			tv.UpdateSelectIdx(idx, true, events.SelectOne)
			return true
		}
	}
	return false
}

// StructSliceIdxByValue searches for first index that contains given value in field of
// given name.
func StructSliceIdxByValue(struSlice any, fldName string, fldVal any) (int, error) {
	svnp := laser.NonPtrValue(reflect.ValueOf(struSlice))
	sz := svnp.Len()
	struTyp := laser.NonPtrType(reflect.TypeOf(struSlice).Elem().Elem())
	fld, ok := struTyp.FieldByName(fldName)
	if !ok {
		err := fmt.Errorf("gi.StructSliceRowByValue: field name: %v not found", fldName)
		slog.Error(err.Error())
		return -1, err
	}
	fldIdx := fld.Index
	for idx := 0; idx < sz; idx++ {
		rval := laser.OnePtrUnderlyingValue(svnp.Index(idx))
		fval := rval.Elem().FieldByIndex(fldIdx)
		if !fval.IsValid() {
			continue
		}
		if fval.Interface() == fldVal {
			return idx, nil
		}
	}
	return -1, nil
}

func (tv *TableView) EditIdx(idx int) {
	if idx < 0 || idx >= tv.SliceNPVal.Len() {
		return
	}
	val := laser.OnePtrUnderlyingValue(tv.SliceNPVal.Index(idx))
	stru := val.Interface()
	tynm := laser.NonPtrType(val.Type()).Name()
	lbl := gi.ToLabel(stru)
	if lbl != "" {
		tynm += ": " + lbl
	}
	d := gi.NewBody().AddTitle(tynm)
	NewStructView(d).SetStruct(stru)
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw)
	})
	d.NewFullDialog(tv).Run()
}

func (tv *TableView) ContextMenu(m *gi.Scene) {
	if !tv.Is(SliceViewIsArray) {
		gi.NewButton(m).SetText("Edit").SetIcon(icons.Edit).
			OnClick(func(e events.Event) {
				tv.EditIdx(tv.SelIdx)
			})
	}
}

//////////////////////////////////////////////////////
// 	Header layout

func (tv *TableView) SizeFinal() {
	tv.SliceViewBase.SizeFinal()
	sg := tv.This().(SliceViewer).SliceGrid()
	sh := tv.SliceHeader()
	sh.WidgetKidsIter(func(i int, kwi gi.Widget, kwb *gi.WidgetBase) bool {
		_, sgb := gi.AsWidget(sg.Child(i))
		gsz := &sgb.Geom.Size
		ksz := &kwb.Geom.Size
		ksz.Actual.Total.X = gsz.Actual.Total.X
		ksz.Actual.Content.X = gsz.Actual.Content.X
		ksz.Alloc.Total.X = gsz.Alloc.Total.X
		ksz.Alloc.Content.X = gsz.Alloc.Content.X
		return ki.Continue
	})
	gsz := &sg.Geom.Size
	ksz := &sh.Geom.Size
	ksz.Actual.Total.X = gsz.Actual.Total.X
	ksz.Actual.Content.X = gsz.Actual.Content.X
	ksz.Alloc.Total.X = gsz.Alloc.Total.X
	ksz.Alloc.Content.X = gsz.Alloc.Content.X
}
