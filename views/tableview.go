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

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"cogentcore.org/core/units"
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

	// structType is the non-pointer struct type for each row.
	structType reflect.Type

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
	tv.Frame.OnInit()
	tv.SliceViewBase.HandleEvents()
	tv.SetStyles()
	tv.AddContextMenu(tv.SliceViewBase.ContextMenu)
	tv.AddContextMenu(tv.ContextMenu)
}

func (tv *TableView) SetStyles() {
	tv.SliceViewBase.SetStyles() // handles all the basics
	tv.SortIndex = -1

	// we only have to handle the header
	tv.OnWidgetAdded(func(w core.Widget) {
		switch w.PathFrom(tv) {
		case "header": // slice header
			sh := w.(*core.Frame)
			core.ToolbarStyles(sh)
			sh.Style(func(s *styles.Style) {
				s.Grow.Set(0, 0)
				s.Gap.Set(units.Em(0.5)) // matches grid default
			})
		case "header/head-idx": // index header
			lbl := w.(*core.Text)
			lbl.SetText("Index").SetType(core.TextBodyMedium)
			w.Style(func(s *styles.Style) {
				s.Align.Self = styles.Center
			})
		}
		if w.Parent().PathFrom(tv) == "header" {
			w.Style(func(s *styles.Style) {
				if hdr, ok := w.(*core.Button); ok {
					fli := hdr.Property("field-index").(int)
					if fli == tv.SortIndex {
						if tv.SortDescending {
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
func (tv *TableView) StyleValueWidget(w core.Widget, s *styles.Style, row, col int) {
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

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice (does Update if already viewing).
func (tv *TableView) SetSlice(sl any) *TableView {
	if reflectx.AnyIsNil(sl) {
		tv.Slice = nil
		return tv
	}
	if tv.Slice == sl && tv.Is(SliceViewConfigured) {
		tv.Update()
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
	tv.SliceNPVal = reflectx.NonPointerValue(reflect.ValueOf(tv.Slice))
	tv.ElVal = reflectx.OnePointerValue(reflectx.SliceElementValue(sl))
	tv.CacheVisFields()
	tv.Update()
	return tv
}

// getStructType sets and returns the structType, removing any pointers.
func (tv *TableView) getStructType() reflect.Type {
	tv.structType = reflectx.NonPointerType(reflectx.SliceElementType(tv.Slice))
	return tv.structType
}

// CacheVisFields computes the number of visible fields in nVisFields and
// caches those to skip in fieldSkip
func (tv *TableView) CacheVisFields() {
	styp := tv.getStructType()
	tv.visibleFields = make([]reflect.StructField, 0)
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
	reflectx.WalkTypeFlatFieldsIf(styp,
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
					tv.visibleFields = append(tv.visibleFields, rfld)
				} else {
					fmt.Printf("TableView: Field name: %v is ambiguous from base struct type: %v, cannot be used in view!\n", fld.Name, styp.String())
				}
			} else {
				tv.visibleFields = append(tv.visibleFields, fld)
			}
			return true
		})
	tv.numVisibleFields = len(tv.visibleFields)
}

// Config configures the view
func (tv *TableView) Config() {
	tv.ConfigTableView()
}

func (tv *TableView) ConfigTableView() {
	if tv.Is(SliceViewConfigured) {
		tv.This().(SliceViewer).UpdateWidgets()
		return
	}
	tv.SortSlice()
	tv.ConfigFrame()
	tv.This().(SliceViewer).ConfigRows()
	tv.This().(SliceViewer).UpdateWidgets()
	tv.ApplyStyleTree()
	tv.NeedsLayout()
}

func (tv *TableView) ConfigFrame() {
	if tv.HasChildren() {
		return
	}
	tv.SetFlag(true, SliceViewConfigured)
	core.NewFrame(tv, "header")
	NewSliceViewGrid(tv, "grid")
	tv.ConfigHeader()
}

func (tv *TableView) ConfigHeader() {
	sgh := tv.SliceHeader()
	if sgh.HasChildren() || tv.numVisibleFields == 0 {
		return
	}
	hcfg := tree.Config{}
	if tv.Is(SliceViewShowIndex) {
		hcfg.Add(core.TextType, "head-idx")
	}
	tv.headerWidths = make([]int, tv.numVisibleFields)
	tv.colMaxWidths = make([]int, tv.numVisibleFields)
	for fli := 0; fli < tv.numVisibleFields; fli++ {
		fld := tv.visibleFields[fli]
		labnm := "head-" + fld.Name
		hcfg.Add(core.ButtonType, labnm)
	}
	sgh.ConfigChildren(hcfg) // headers SHOULD be unique, but with labels..
	_, idxOff := tv.RowWidgetNs()
	nfld := tv.numVisibleFields
	for fli := 0; fli < nfld; fli++ {
		field := tv.visibleFields[fli]
		hdr := sgh.Child(idxOff + fli).(*core.Button)
		hdr.SetType(core.ButtonMenu)
		htxt := ""
		if lbl, ok := field.Tag.Lookup("label"); ok {
			htxt = lbl
		} else {
			htxt = strcase.ToSentence(field.Name)
		}
		hdr.SetText(htxt)
		tv.headerWidths[fli] = len(htxt)
		hdr.SetProperty("field-index", fli)
		if fli == tv.SortIndex {
			if tv.SortDescending {
				hdr.SetIcon(icons.KeyboardArrowDown)
			} else {
				hdr.SetIcon(icons.KeyboardArrowUp)
			}
		}
		hdr.Tooltip = hdr.Text + " (tap to sort by)"
		doc, ok := types.GetDoc(reflect.Value{}, tv.ElVal, &field, hdr.Text)
		if ok && doc != "" {
			hdr.Tooltip += ": " + doc
		}
		hdr.OnClick(func(e events.Event) {
			tv.SortSliceAction(fli)
		})
	}
}

// SliceHeader returns the Frame header for slice grid
func (tv *TableView) SliceHeader() *core.Frame {
	return tv.Child(0).(*core.Frame)
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

// ConfigRows configures VisRows worth of widgets
// to display slice data.  It should only be called
// when NeedsConfigRows is true: when VisRows changes.
func (tv *TableView) ConfigRows() {
	sg := tv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	tv.SetFlag(true, SliceViewConfigured)
	sg.SetFlag(true, core.LayoutNoKeys)

	tv.ViewMuLock()
	defer tv.ViewMuUnlock()

	sg.DeleteChildren()
	tv.Values = nil

	tv.This().(SliceViewer).UpdateSliceSize()

	if tv.IsNil() {
		return
	}

	nWidgPerRow, idxOff := tv.RowWidgetNs()
	nWidg := nWidgPerRow * tv.VisRows
	sg.Styles.Columns = nWidgPerRow

	tv.Values = make([]Value, tv.numVisibleFields*tv.VisRows)
	sg.Kids = make(tree.Slice, nWidg)

	for i := 0; i < tv.VisRows; i++ {
		si := i
		ridx := i * nWidgPerRow
		var val reflect.Value
		if si < tv.SliceSize {
			val = reflectx.OnePointerUnderlyingValue(tv.SliceNPVal.Index(si)) // deal with pointer lists
		} else {
			val = tv.ElVal
		}
		if val.IsZero() {
			val = tv.ElVal
		}
		stru := val.Interface()

		idxlab := &core.Text{}
		itxt := strconv.Itoa(i)
		sitxt := strconv.Itoa(si)
		labnm := "index-" + itxt
		if tv.Is(SliceViewShowIndex) {
			idxlab = &core.Text{}
			sg.SetChild(idxlab, ridx, labnm)
			idxlab.SetText(sitxt)
			idxlab.OnSelect(func(e events.Event) {
				e.SetHandled()
				tv.UpdateSelectRow(i, e.SelectMode())
			})
			idxlab.SetProperty(SliceViewRowProperty, i)
		}

		vpath := tv.ViewPath + "[" + sitxt + "]"
		for fli := 0; fli < tv.numVisibleFields; fli++ {
			field := tv.visibleFields[fli]
			fval := val.Elem().FieldByIndex(field.Index)
			vvi := i*tv.numVisibleFields + fli
			tags := ""
			if fval.Kind() == reflect.Slice || fval.Kind() == reflect.Map {
				tags = `view:"no-inline"`
			}
			vv := ToValue(fval.Interface(), tags)
			tv.Values[vvi] = vv
			vv.SetStructValue(fval.Addr(), stru, &field, vpath)
			vv.SetReadOnly(tv.IsReadOnly())

			vtyp := vv.WidgetType()
			valnm := fmt.Sprintf("value-%v.%v", fli, itxt)
			cidx := ridx + idxOff + fli
			w := tree.NewOfType(vtyp).(core.Widget)
			sg.SetChild(w, cidx, valnm)
			Config(vv, w)
			w.SetProperty(SliceViewRowProperty, i)
			w.SetProperty(SliceViewColProperty, fli)

			if !tv.IsReadOnly() {
				vv.OnChange(func(e events.Event) {
					tv.SendChange()
				})
				vv.AsWidgetBase().OnInput(tv.HandleEvent)
			}
			if i == 0 && tv.SliceSize > 0 {
				tv.colMaxWidths[fli] = 0
				_, isicon := vv.(*IconValue)
				if !isicon && fval.Kind() == reflect.String {
					mxw := 0
					for rw := 0; rw < tv.SliceSize; rw++ {
						sval := reflectx.OnePointerUnderlyingValue(tv.SliceNPVal.Index(rw))
						elem := sval.Elem()
						if !elem.IsValid() {
							continue
						}
						fval := elem.FieldByIndex(field.Index)
						str := fval.String()
						mxw = max(mxw, len(str))
					}
					tv.colMaxWidths[fli] = mxw
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

	tv.ViewMuLock()
	defer tv.ViewMuUnlock()

	tv.This().(SliceViewer).UpdateSliceSize()

	nWidgPerRow, idxOff := tv.RowWidgetNs()

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

	for i := 0; i < tv.VisRows; i++ {
		ridx := i * nWidgPerRow
		si := tv.StartIndex + i // slice idx
		invis := si >= tv.SliceSize

		var idxlab *core.Text
		if tv.Is(SliceViewShowIndex) {
			if len(sg.Kids) == 0 {
				break
			}
			idxlab = sg.Kids[ridx].(*core.Text)
			idxlab.SetText(strconv.Itoa(si)).Config()
			idxlab.SetState(invis, states.Invisible)
		}

		sitxt := strconv.Itoa(si)
		vpath := tv.ViewPath + "[" + sitxt + "]"
		if si < tv.SliceSize {
			if lblr, ok := tv.Slice.(core.SliceLabeler); ok {
				slbl := lblr.ElemLabel(si)
				if slbl != "" {
					vpath = JoinViewPath(tv.ViewPath, slbl)
				}
			}
		}
		for fli := 0; fli < tv.numVisibleFields; fli++ {
			field := tv.visibleFields[fli]
			cidx := ridx + idxOff + fli
			if len(sg.Kids) < cidx {
				break
			}
			if sg.Kids[cidx] == nil {
				return
			}
			w := sg.Kids[cidx].(core.Widget)
			wb := w.AsWidget()

			var val reflect.Value
			if !invis {
				val = reflectx.OnePointerUnderlyingValue(tv.SliceNPVal.Index(si)) // deal with pointer lists
				if val.IsZero() {
					val = tv.ElVal
				}
			} else {
				val = tv.ElVal
			}
			stru := val.Interface()
			fval := val.Elem().FieldByIndex(field.Index)
			vvi := i*tv.numVisibleFields + fli
			vv := tv.Values[vvi]
			vv.SetStructValue(fval.Addr(), stru, &field, vpath)
			vv.SetReadOnly(tv.IsReadOnly())
			vv.Update()
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
	sg.NeedsRender()
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
	tv.This().(SliceViewer).UpdateWidgets()
	tv.IndexGrabFocus(idx)
	tv.NeedsLayout()
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
	tv.This().(SliceViewer).UpdateWidgets()
	tv.NeedsLayout()
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
	sgh.Update() // requires full update due to sort button icon
	tv.UpdateWidgets()
	tv.NeedsLayout()
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
		rval := reflectx.OnePointerUnderlyingValue(svnp.Index(idx))
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
	if idx < 0 || idx >= tv.SliceNPVal.Len() {
		return
	}
	val := reflectx.OnePointerUnderlyingValue(tv.SliceNPVal.Index(idx))
	stru := val.Interface()
	tynm := reflectx.NonPointerType(val.Type()).Name()
	lbl := core.ToLabel(stru)
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
