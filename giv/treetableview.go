// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/glop/sentence"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"fmt"
	"image"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
)

type (
	TableRowData interface {
		OnInit()
		SetStyles()
		SetSlice(sl any) *TreeTableView
		StructType() reflect.Type
		CacheVisFields()
		ConfigWidget()
		ConfigTreeTableView()
		ConfigFrame()
		ConfigHeader()
		SliceGrid() *SliceViewGrid
		SliceHeader() *gi.Frame
		RowWidgetNs() (nWidgPerRow int, idxOff int)
		ConfigRows()
		UpdateWidgets()
		StyleRow(w gi.Widget, idx int, fidx int)
		SliceNewAt(idx int)
		SliceDeleteAt(idx int)
		SortSlice()
		SortSliceAction(fldIdx int)
		SortFieldName() string
		SetSortFieldName(nm string)
		RowFirstVisWidget(row int) (*gi.WidgetBase, bool)
		RowGrabFocus(row int) *gi.WidgetBase
		SelectRowWidgets(row int, sel bool)
		SelectFieldVal(fld string, val string) bool
		EditIdx(idx int)
		ContextMenu(m *gi.Scene)
		SizeFinal()
	}
)

type TreeTableView struct { //gti:add

	TreeHeaderFrame *gi.Frame
	TreeView        *TreeView

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
	StruType reflect.Type `copier:"-" view:"-" json:"-" xml:"-"`

	// the visible fields
	VisFields []reflect.StructField `copier:"-" view:"-" json:"-" xml:"-"`

	// number of visible fields
	NVisFields int `copier:"-" view:"-" json:"-" xml:"-"`

	// HeaderWidths has number of characters in each header, per visfields
	HeaderWidths []int `copier:"-" view:"-" json:"-" xml:"-"`
}

//go:generate core generate

var _ SliceViewer = (*TreeTableView)(nil)

func (t *TreeTableView) OnInit() {
	t.Frame.OnInit()
	t.SliceViewBase.HandleEvents()
	t.SetStyles()
	t.AddContextMenu(t.SliceViewBase.ContextMenu)
	t.AddContextMenu(t.ContextMenu)
}

func (t *TreeTableView) SetStyles() {
	t.SortIdx = -1
	t.MinRows = 4
	t.SetFlag(false, SliceViewSelectMode)
	t.SetFlag(true, SliceViewShowIndex)
	t.SetFlag(true, SliceViewReadOnlyKeyNav)

	t.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.DoubleClickable)
		s.Direction = styles.Column
		// absorb horizontal here, vertical in view
		s.Overflow.X = styles.OverflowAuto
		s.Grow.Set(1, 1)
	})
	t.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(t) {
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
		case "grid": // slice grid
			sg := w.(*SliceViewGrid)
			sg.Stripes = gi.RowStripes
			sg.Style(func(s *styles.Style) {
				sg.MinRows = t.MinRows
				s.Display = styles.Grid
				nWidgPerRow, _ := t.RowWidgetNs()
				s.Columns = nWidgPerRow
				s.Grow.Set(1, 1)
				s.Overflow.Y = styles.OverflowAuto
				s.Gap.Set(units.Em(0.5)) // note: match header
				// baseline mins:
				s.Min.X.Ch(20)
				s.Min.Y.Em(6)
			})
		}
		if w.Parent().PathFrom(t) == "grid" {
			switch {
			case strings.HasPrefix(w.Name(), "index-"):
				w.Style(func(s *styles.Style) {
					s.Min.X.Ch(5)
					s.Padding.Right.Dp(4)
					s.Text.Align = styles.End
					s.Min.Y.Em(1)
					s.GrowWrap = false
				})
			case strings.HasPrefix(w.Name(), "add-"):
				w.Style(func(s *styles.Style) {
					w.(*gi.Button).SetType(gi.ButtonAction)
					s.Color = colors.Scheme.Success.Base
				})
			case strings.HasPrefix(w.Name(), "del-"):
				w.Style(func(s *styles.Style) {
					w.(*gi.Button).SetType(gi.ButtonAction)
					s.Color = colors.Scheme.Error.Base
				})
			case strings.HasPrefix(w.Name(), "value-"):
				w.Style(func(s *styles.Style) {
					fstr := strings.TrimPrefix(w.Name(), "value-")
					dp := strings.Index(fstr, ".")
					istr := fstr[dp+1:] // index is after .
					fstr = fstr[:dp]    // field idx is -X.
					idx := grr.Log1(strconv.Atoi(istr))
					fli := grr.Log1(strconv.Atoi(fstr))
					hw := float32(t.HeaderWidths[fli])
					if fli == t.SortIdx {
						hw += 6
					}
					hv := units.Ch(hw)
					s.Min.X.Val = max(s.Min.X.Val, hv.Convert(s.Min.X.Un, &s.UnContext).Val)
					s.Max.X.Val = max(s.Max.X.Val, hv.Convert(s.Max.X.Un, &s.UnContext).Val)
					si := t.StartIdx + idx
					if si < t.SliceSize {
						t.This().(SliceViewer).StyleRow(w, si, fli)
					}
				})
			}
		}
		if w.Parent().PathFrom(t) == "header" {
			w.Style(func(s *styles.Style) {
				if hdr, ok := w.(*gi.Button); ok {
					fli := hdr.Prop("field-index").(int)
					if fli == t.SortIdx {
						if t.SortDesc {
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

func (t *TreeTableView) SetSlice(sl any) *TreeTableView {
	if laser.AnyIsNil(sl) {
		t.Slice = nil
		return t
	}
	if t.Slice == sl && t.Is(SliceViewConfigured) {
		t.Update()
		return t
	}
	updt := t.UpdateStart()
	defer t.UpdateEndLayout(updt)

	t.SetFlag(false, SliceViewConfigured)
	t.StartIdx = 0
	t.VisRows = t.MinRows
	slpTyp := reflect.TypeOf(sl)
	if slpTyp.Kind() != reflect.Ptr {
		slog.Error("TableView requires that you pass a pointer to a slice of struct elements, but type is not a Ptr", "type", slpTyp)
		return t
	}
	if slpTyp.Elem().Kind() != reflect.Slice {
		slog.Error("TableView requires that you pass a pointer to a slice of struct elements, but ptr doesn't point to a slice", "type", slpTyp.Elem())
		return t
	}
	t.Slice = sl
	t.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(t.Slice))
	struTyp := t.StructType()
	if struTyp.Kind() != reflect.Struct {
		slog.Error("TableView requires that you pass a slice of struct elements, but type is not a Struct", "type", struTyp.String())
		return t
	}
	t.ElVal = laser.OnePtrValue(laser.SliceElValue(sl))
	t.CacheVisFields()
	if !t.IsReadOnly() {
		t.SelIdx = -1
	}
	t.ResetSelectedIdxs()
	t.SetFlag(false, SliceViewSelectMode)
	t.ConfigIter = 0
	t.Update()
	return t
}

func (t *TreeTableView) StructType() reflect.Type {
	t.StruType = laser.NonPtrType(laser.SliceElType(t.Slice))
	return t.StruType
}

func (t *TreeTableView) CacheVisFields() {
	styp := t.StructType()
	t.VisFields = make([]reflect.StructField, 0)
	laser.FlatFieldsTypeFuncIf(styp,
		func(typ reflect.Type, fld reflect.StructField) bool {
			if !fld.IsExported() {
				return false
			}
			tvtag := fld.Tag.Get("tableview")
			if tvtag != "" {
				if tvtag == "-" {
					return false
				} else if tvtag == "-select" && t.IsReadOnly() {
					return false
				} else if tvtag == "-edit" && !t.IsReadOnly() {
					return false
				}
			}
			vtag := fld.Tag.Get("view")
			return vtag != "-"
		},
		func(typ reflect.Type, fld reflect.StructField) bool {
			if !fld.IsExported() {
				return true
			}
			tvtag := fld.Tag.Get("tableview")
			add := true
			if tvtag != "" {
				if tvtag == "-" {
					add = false
				} else if tvtag == "-select" && t.IsReadOnly() {
					add = false
				} else if tvtag == "-edit" && !t.IsReadOnly() {
					add = false
				}
			}
			vtag := fld.Tag.Get("view")
			if vtag == "-" {
				add = false
			}
			if add {
				if typ != styp {
					rfld, has := styp.FieldByName(fld.Name)
					if has {
						t.VisFields = append(t.VisFields, rfld)
					} else {
						fmt.Printf("TableView: Field name: %v is ambiguous from base struct type: %v, cannot be used in view!\n", fld.Name, styp.String())
					}
				} else {
					t.VisFields = append(t.VisFields, fld)
				}
			}
			return true
		})
	t.NVisFields = len(t.VisFields)
}

func (t *TreeTableView) ConfigWidget() { t.ConfigTreeTableView() }
func (t *TreeTableView) ConfigTreeTableView() {
	if t.Is(SliceViewConfigured) {
		t.This().(SliceViewer).UpdateWidgets()
		return
	}
	updt := t.UpdateStart()
	t.SortSlice()
	t.ConfigFrame()
	t.This().(SliceViewer).ConfigRows()
	t.This().(SliceViewer).UpdateWidgets()
	t.ApplyStyleTree()
	t.UpdateEndLayout(updt)

	hSplits := NewHSplits(t)          //todo
	treeFrame := gi.NewFrame(hSplits) //left
	//tableFrame := gi.NewFrame(hSplits) //right
	hSplits.SetSplits(.2, .8)

	treeFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	treeHeaderFrame := gi.NewFrame(treeFrame) //treeHeader for align table header
	treeHeaderFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Row
	})
	gi.NewTextField(treeHeaderFrame).SetPlaceholder("filter content")
	gi.NewButton(treeHeaderFrame).SetIcon("hierarchy")
	gi.NewButton(treeHeaderFrame).SetIcon("circled_add")
	gi.NewButton(treeHeaderFrame).SetIcon("trash")
	gi.NewButton(treeHeaderFrame).SetIcon("star")

	treeView := NewTreeView(treeFrame)
	treeView.IconOpen = icons.ExpandCircleDown
	treeView.IconClosed = icons.ExpandCircleRight
	treeView.IconLeaf = icons.Blank
}

// util
func NewHSplits(parent ki.Ki) *gi.Splits { return newSplits(parent, true) }
func NewVSplits(parent ki.Ki) *gi.Splits { return newSplits(parent, false) }

func newSplits(parent ki.Ki, isHorizontal bool) *gi.Splits { // Horizontal and vertical
	splits := gi.NewSplits(parent)
	splits.Style(func(s *styles.Style) {
		if isHorizontal {
			s.Direction = styles.Row
		} else {
			s.Direction = styles.Column
		}
		s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
	})
	return splits
}

func (t *TreeTableView) ConfigFrame() {
	if t.HasChildren() {
		return
	}
	t.SetFlag(true, SliceViewConfigured)
	gi.NewFrame(t, "header")
	NewSliceViewGrid(t, "grid")
	t.ConfigHeader()
}

func (t *TreeTableView) ConfigHeader() {
	sgh := t.SliceHeader()
	if sgh.HasChildren() || t.NVisFields == 0 {
		return
	}
	hcfg := ki.Config{}
	if t.Is(SliceViewShowIndex) {
		hcfg.Add(gi.LabelType, "head-idx")
	}
	t.HeaderWidths = make([]int, t.NVisFields)
	for fli := 0; fli < t.NVisFields; fli++ {
		fld := t.VisFields[fli]
		labnm := "head-" + fld.Name
		hcfg.Add(gi.ButtonType, labnm)
	}
	if !t.IsReadOnly() {
		hcfg.Add(gi.LabelType, "head-add")
		hcfg.Add(gi.LabelType, "head-del")
	}
	sgh.ConfigChildren(hcfg) // headers SHOULD be unique, but with labels..
	_, idxOff := t.RowWidgetNs()
	nfld := t.NVisFields
	for fli := 0; fli < nfld; fli++ {
		fli := fli
		field := t.VisFields[fli]
		hdr := sgh.Child(idxOff + fli).(*gi.Button)
		hdr.SetType(gi.ButtonMenu)
		htxt := ""
		if lbl, ok := field.Tag.Lookup("label"); ok {
			htxt = lbl
		} else {
			htxt = sentence.Case(field.Name)
		}
		hdr.SetText(htxt)
		t.HeaderWidths[fli] = len(htxt)
		hdr.SetProp("field-index", fli)
		if fli == t.SortIdx {
			if t.SortDesc {
				hdr.SetIcon(icons.KeyboardArrowDown)
			} else {
				hdr.SetIcon(icons.KeyboardArrowUp)
			}
		}
		hdr.Tooltip = hdr.Text + " (tap to sort by)"
		doc, ok := gti.GetDoc(reflect.Value{}, t.ElVal, &field, hdr.Text)
		if ok && doc != "" {
			hdr.Tooltip += ": " + doc
		}
		hdr.OnClick(func(e events.Event) {
			t.SortSliceAction(fli)
		})
	}
	if !t.IsReadOnly() {
		cidx := t.NVisFields + idxOff
		if !t.Is(SliceViewNoAdd) {
			lbl := sgh.Child(cidx).(*gi.Label)
			lbl.Text = "+"
			lbl.Tooltip = "insert row"
			cidx++
		}
		if !t.Is(SliceViewNoDelete) {
			lbl := sgh.Child(cidx).(*gi.Label)
			lbl.Text = "-"
			lbl.Tooltip = "delete row"
		}
	}
}

func (t *TreeTableView) SliceGrid() *SliceViewGrid { return t.Child(1).(*SliceViewGrid) }

func (t *TreeTableView) SliceHeader() *gi.Frame { return t.Child(0).(*gi.Frame) }

func (t *TreeTableView) RowWidgetNs() (nWidgPerRow int, idxOff int) {
	nWidgPerRow = 1 + t.NVisFields
	if !t.IsReadOnly() {
		if !t.Is(SliceViewNoAdd) {
			nWidgPerRow += 1
		}
		if !t.Is(SliceViewNoDelete) {
			nWidgPerRow += 1
		}
	}
	idxOff = 1
	if !t.Is(SliceViewShowIndex) {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

func (t *TreeTableView) ConfigRows() {
	sg := t.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	t.SetFlag(true, SliceViewConfigured)
	sg.SetFlag(true, gi.LayoutNoKeys)

	t.ViewMuLock()
	defer t.ViewMuUnlock()

	sg.DeleteChildren(ki.DestroyKids)
	t.Values = nil

	t.This().(SliceViewer).UpdtSliceSize()

	if t.IsNil() {
		return
	}

	nWidgPerRow, idxOff := t.RowWidgetNs()
	nWidg := nWidgPerRow * t.VisRows
	sg.Styles.Columns = nWidgPerRow

	t.Values = make([]Value, t.NVisFields*t.VisRows)
	sg.Kids = make(ki.Slice, nWidg)

	for i := 0; i < t.VisRows; i++ {
		i := i
		si := i
		ridx := i * nWidgPerRow
		var val reflect.Value
		if si < t.SliceSize {
			val = laser.OnePtrUnderlyingValue(t.SliceNPVal.Index(si)) // deal with pointer lists
		} else {
			val = t.ElVal
		}
		if val.IsZero() {
			val = t.ElVal
		}
		stru := val.Interface()

		idxlab := &gi.Label{}
		itxt := strconv.Itoa(i)
		sitxt := strconv.Itoa(si)
		labnm := "index-" + itxt
		if t.Is(SliceViewShowIndex) {
			idxlab = &gi.Label{}
			sg.SetChild(idxlab, ridx, labnm)
			idxlab.OnSelect(func(e events.Event) {
				e.SetHandled()
				t.UpdateSelectRow(i)
			})
			idxlab.SetText(sitxt)
			idxlab.ContextMenus = t.ContextMenus
		}

		vpath := t.ViewPath + "[" + sitxt + "]"
		for fli := 0; fli < t.NVisFields; fli++ {
			fli := fli
			field := t.VisFields[fli]
			fval := val.Elem().FieldByIndex(field.Index)
			vvi := i*t.NVisFields + fli
			tags := ""
			if fval.Kind() == reflect.Slice || fval.Kind() == reflect.Map {
				tags = `view:"no-inline"`
			}
			vv := ToValue(fval.Interface(), tags)
			t.Values[vvi] = vv
			vv.SetStructValue(fval.Addr(), stru, &field, t.TmpSave, vpath)
			vv.SetReadOnly(t.IsReadOnly())

			vtyp := vv.WidgetType()
			valnm := fmt.Sprintf("value-%v.%v", fli, itxt)
			cidx := ridx + idxOff + fli
			w := ki.NewOfType(vtyp).(gi.Widget)
			sg.SetChild(w, cidx, valnm)
			vv.ConfigWidget(w)
			wb := w.AsWidget()
			wb.OnSelect(func(e events.Event) {
				e.SetHandled()
				t.UpdateSelectRow(i)
			})
			wb.ContextMenus = t.ContextMenus

			if t.IsReadOnly() {
				w.AsWidget().SetReadOnly(true)
			} else {
				vvb := vv.AsValueBase()
				vvb.OnChange(func(e events.Event) {
					t.SetChanged()
				})
			}
		}

		if !t.IsReadOnly() {
			cidx := ridx + t.NVisFields + idxOff
			if !t.Is(SliceViewNoAdd) {
				addnm := fmt.Sprintf("add-%v", itxt)
				addact := gi.Button{}
				sg.SetChild(&addact, cidx, addnm)
				addact.SetType(gi.ButtonAction).SetIcon(icons.Add).
					SetTooltip("insert a new element at this index").OnClick(func(e events.Event) {
					t.SliceNewAtRow(i + 1)
				})
				cidx++
			}
			if !t.Is(SliceViewNoDelete) {
				delnm := fmt.Sprintf("del-%v", itxt)
				delact := gi.Button{}
				sg.SetChild(&delact, cidx, delnm)
				delact.SetType(gi.ButtonAction).SetIcon(icons.Delete).
					SetTooltip("delete this element").OnClick(func(e events.Event) {
					t.SliceDeleteAtRow(i)
				})
				cidx++
			}
		}
	}
	t.ConfigTree()
	t.ApplyStyleTree()
}

func (t *TreeTableView) UpdateWidgets() {
	sg := t.This().(SliceViewer).SliceGrid()
	if sg == nil || t.VisRows == 0 || sg.VisRows == 0 || !sg.HasChildren() {
		return
	}
	// sc := t.Sc

	updt := sg.UpdateStart()
	defer sg.UpdateEndRender(updt)

	t.ViewMuLock()
	defer t.ViewMuUnlock()

	t.This().(SliceViewer).UpdtSliceSize()

	nWidgPerRow, idxOff := t.RowWidgetNs()

	t.UpdateStartIdx()
	for i := 0; i < t.VisRows; i++ {
		i := i
		ridx := i * nWidgPerRow
		si := t.StartIdx + i // slice idx
		invis := si >= t.SliceSize

		var idxlab *gi.Label
		if t.Is(SliceViewShowIndex) {
			idxlab = sg.Kids[ridx].(*gi.Label)
			idxlab.SetTextUpdate(strconv.Itoa(si))
			idxlab.SetState(invis, states.Invisible)
		}

		sitxt := strconv.Itoa(si)
		vpath := t.ViewPath + "[" + sitxt + "]"
		if si < t.SliceSize {
			if lblr, ok := t.Slice.(gi.SliceLabeler); ok {
				slbl := lblr.ElemLabel(si)
				if slbl != "" {
					vpath = t.ViewPath + "[" + slbl + "]"
				}
			}
		}
		for fli := 0; fli < t.NVisFields; fli++ {
			fli := fli
			field := t.VisFields[fli]
			cidx := ridx + idxOff + fli
			w := sg.Kids[cidx].(gi.Widget)
			wb := w.AsWidget()

			var val reflect.Value
			if si < t.SliceSize {
				val = laser.OnePtrUnderlyingValue(t.SliceNPVal.Index(si)) // deal with pointer lists
				if val.IsZero() {
					val = t.ElVal
				}
			} else {
				val = t.ElVal
			}
			stru := val.Interface()
			fval := val.Elem().FieldByIndex(field.Index)
			vvi := i*t.NVisFields + fli
			vv := t.Values[vvi]
			vv.SetStructValue(fval.Addr(), stru, &field, t.TmpSave, vpath)
			vv.SetReadOnly(t.IsReadOnly())
			vv.UpdateWidget()

			w.SetState(invis, states.Invisible)
			if !invis {
				issel := t.IdxIsSelected(si)
				if t.IsReadOnly() {
					wb.SetReadOnly(true)
				}
				wb.SetSelected(issel)
			} else {
				wb.SetSelected(false)
				if t.Is(SliceViewShowIndex) {
					idxlab.SetSelected(false)
				}
			}
		}
		if !t.IsReadOnly() {
			cidx := ridx + t.NVisFields + idxOff
			if !t.Is(SliceViewNoAdd) {
				addact := sg.Kids[cidx].(*gi.Button)
				addact.SetState(invis, states.Invisible)
				cidx++
			}
			if !t.Is(SliceViewNoDelete) {
				delact := sg.Kids[cidx].(*gi.Button)
				delact.SetState(invis, states.Invisible)
				cidx++
			}
		}
	}

	if t.SelField != "" && t.SelVal != nil {
		t.SelIdx, _ = StructSliceIdxByValue(t.Slice, t.SelField, t.SelVal)
		t.SelField = ""
		t.SelVal = nil
		t.ScrollToIdx(t.SelIdx)
		// t.SetFocusEvent() // todo:
	} else if t.InitSelIdx >= 0 {
		t.SelIdx = t.InitSelIdx
		t.InitSelIdx = -1
		t.ScrollToIdx(t.SelIdx)
		// t.SetFocusEvent()
	}

	if t.IsReadOnly() && t.SelIdx >= 0 {
		t.SelectIdx(t.SelIdx)
	}
}

func (t *TreeTableView) StyleRow(w gi.Widget, idx, fidx int) {
	if t.StyleFunc != nil {
		t.StyleFunc(w, &w.AsWidget().Styles, idx, fidx)
	}
}

func (t *TreeTableView) SliceNewAt(idx int) {
	t.ViewMuLock()
	updt := t.UpdateStart()
	defer t.UpdateEndLayout(updt)

	t.SliceNewAtSel(idx)
	laser.SliceNewAt(t.Slice, idx)
	if idx < 0 {
		idx = t.SliceSize
	}

	t.This().(SliceViewer).UpdtSliceSize()
	if t.TmpSave != nil {
		t.TmpSave.SaveTmp()
	}
	t.ViewMuUnlock()
	t.SetChanged()
	t.This().(SliceViewer).UpdateWidgets()
}

func (t *TreeTableView) SliceDeleteAt(idx int) {
	if idx < 0 || idx >= t.SliceSize {
		return
	}
	t.ViewMuLock()
	updt := t.UpdateStart()
	defer t.UpdateEndLayout(updt)

	t.SliceDeleteAtSel(idx)

	laser.SliceDeleteAt(t.Slice, idx)

	t.This().(SliceViewer).UpdtSliceSize()

	if t.TmpSave != nil {
		t.TmpSave.SaveTmp()
	}
	t.ViewMuUnlock()
	t.SetChanged()
	t.This().(SliceViewer).UpdateWidgets()
}

func (t *TreeTableView) SortSlice() {
	if t.SortIdx < 0 || t.SortIdx >= len(t.VisFields) {
		return
	}
	rawIdx := t.VisFields[t.SortIdx].Index
	laser.StructSliceSort(t.Slice, rawIdx, !t.SortDesc)
}

func (t *TreeTableView) SortSliceAction(fldIdx int) {
	updt := t.UpdateStart()
	defer t.UpdateEndLayout(updt)

	sgh := t.SliceHeader()
	_, idxOff := t.RowWidgetNs()

	ascending := true

	for fli := 0; fli < t.NVisFields; fli++ {
		hdr := sgh.Child(idxOff + fli).(*gi.Button)
		hdr.SetType(gi.ButtonAction)
		if fli == fldIdx {
			if t.SortIdx == fli {
				t.SortDesc = !t.SortDesc
				ascending = !t.SortDesc
			} else {
				t.SortDesc = false
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

	t.SortIdx = fldIdx
	t.SortSlice()
	sgh.Update() // requires full update due to sort button icon
	t.UpdateWidgets()
}

func (t *TreeTableView) SortFieldName() string {
	if t.SortIdx >= 0 && t.SortIdx < t.NVisFields {
		nm := t.VisFields[t.SortIdx].Name
		if t.SortDesc {
			nm += ":down"
		} else {
			nm += ":up"
		}
		return nm
	}
	return ""
}

func (t *TreeTableView) SetSortFieldName(nm string) {
	if nm == "" {
		return
	}
	spnm := strings.Split(nm, ":")
	got := false
	for fli := 0; fli < t.NVisFields; fli++ {
		fld := t.VisFields[fli]
		if fld.Name == spnm[0] {
			got = true
			// fmt.Println("sorting on:", fld.Name, fli, "from:", nm)
			t.SortIdx = fli
		}
	}
	if len(spnm) == 2 {
		if spnm[1] == "down" {
			t.SortDesc = true
		} else {
			t.SortDesc = false
		}
	}
	if got {
		t.SortSlice()
	}
}

func (t *TreeTableView) RowFirstVisWidget(row int) (*gi.WidgetBase, bool) {
	if !t.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, idxOff := t.RowWidgetNs()
	sg := t.SliceGrid()
	w := sg.Kids[row*nWidgPerRow].(gi.Widget).AsWidget()
	if w.Geom.TotalBBox != (image.Rectangle{}) {
		return w, true
	}
	ridx := nWidgPerRow * row
	for fli := 0; fli < t.NVisFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(gi.Widget).AsWidget()
		if w.Geom.TotalBBox != (image.Rectangle{}) {
			return w, true
		}
	}
	return nil, false
}

func (t *TreeTableView) RowGrabFocus(row int) *gi.WidgetBase {
	if !t.IsRowInBounds(row) || t.Is(SliceViewInFocusGrab) { // range check
		return nil
	}
	nWidgPerRow, idxOff := t.RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := t.SliceGrid()
	// first check if we already have focus
	for fli := 0; fli < t.NVisFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(gi.Widget).AsWidget()
		if w.StateIs(states.Focused) || w.ContainsFocus() {
			return w
		}
	}
	t.SetFlag(true, SliceViewInFocusGrab)
	defer func() { t.SetFlag(false, SliceViewInFocusGrab) }()
	for fli := 0; fli < t.NVisFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(gi.Widget).AsWidget()
		if w.CanFocus() {
			w.SetFocusEvent()
			return w
		}
	}
	return nil
}

func (t *TreeTableView) SelectRowWidgets(row int, sel bool) {
	if row < 0 {
		return
	}
	updt := t.UpdateStart()
	defer t.UpdateEndRender(updt)

	sg := t.SliceGrid()
	nWidgPerRow, idxOff := t.RowWidgetNs()
	ridx := row * nWidgPerRow
	for fli := 0; fli < t.NVisFields; fli++ {
		seldx := ridx + idxOff + fli
		if sg.Kids.IsValidIndex(seldx) == nil {
			w := sg.Child(seldx).(gi.Widget).AsWidget()
			w.SetSelected(sel)
		}
	}
	if t.Is(SliceViewShowIndex) {
		if sg.Kids.IsValidIndex(ridx) == nil {
			w := sg.Child(ridx).(gi.Widget).AsWidget()
			w.SetSelected(sel)
		}
	}
}

func (t *TreeTableView) SelectFieldVal(fld, val string) bool {
	t.SelField = fld
	t.SelVal = val
	if t.SelField != "" && t.SelVal != nil {
		idx, _ := StructSliceIdxByValue(t.Slice, t.SelField, t.SelVal)
		if idx >= 0 {
			t.ScrollToIdx(idx)
			t.UpdateSelectIdx(idx, true)
			return true
		}
	}
	return false
}

//func StructSliceIdxByValue(struSlice any, fldName string, fldVal any) (int, error) {
//	svnp := laser.NonPtrValue(reflect.ValueOf(struSlice))
//	sz := svnp.Len()
//	struTyp := laser.NonPtrType(reflect.TypeOf(struSlice).Elem().Elem())
//	fld, ok := struTyp.FieldByName(fldName)
//	if !ok {
//		err := fmt.Errorf("gi.StructSliceRowByValue: field name: %v not found\n", fldName)
//		slog.Error(err.Error())
//		return -1, err
//	}
//	fldIdx := fld.Index
//	for idx := 0; idx < sz; idx++ {
//		rval := laser.OnePtrUnderlyingValue(svnp.Index(idx))
//		fval := rval.Elem().FieldByIndex(fldIdx)
//		if !fval.IsValid() {
//			continue
//		}
//		if fval.Interface() == fldVal {
//			return idx, nil
//		}
//	}
//	return -1, nil
//}

func (t *TreeTableView) EditIdx(idx int) {
	val := laser.OnePtrUnderlyingValue(t.SliceNPVal.Index(idx))
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
	d.NewFullDialog(t).Run()
}

func (t *TreeTableView) ContextMenu(m *gi.Scene) {
	if !t.Is(SliceViewIsArray) {
		gi.NewButton(m).SetText("Edit").SetIcon(icons.Edit).
			OnClick(func(e events.Event) {
				t.EditIdx(t.SelIdx)
			})
		gi.NewSeparator(m)
	}
}

func (t *TreeTableView) SizeFinal() {
	t.SliceViewBase.SizeFinal()
	sg := t.This().(SliceViewer).SliceGrid()
	sh := t.SliceHeader()
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
