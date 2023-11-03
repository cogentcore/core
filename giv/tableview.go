// Copyright (c) 2018, The GoKi Authors. All rights reserved.
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

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
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
	StyleFunc TableViewStyleFunc `copy:"-" view:"-" json:"-" xml:"-"`

	// current selection field -- initially select value in this field
	SelField string `copy:"-" view:"-" json:"-" xml:"-"`

	// current sort index
	SortIdx int

	// whether current sort order is descending
	SortDesc bool

	// struct type for each row
	StruType reflect.Type `copy:"-" view:"-" json:"-" xml:"-"`

	// the visible fields
	VisFields []reflect.StructField `copy:"-" view:"-" json:"-" xml:"-"`

	// number of visible fields
	NVisFields int `copy:"-" view:"-" json:"-" xml:"-"`
}

// check for interface impl
var _ SliceViewer = (*TableView)(nil)

// TableViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view.  If style properties are set
// then you must call w.AsNode2dD().SetFullReRender() to trigger
// re-styling during re-render
type TableViewStyleFunc func(w gi.Widget, s *styles.Style, row, col int)

func (tv *TableView) OnInit() {
	tv.TableViewInit()
}

func (tv *TableView) TableViewInit() {
	tv.SetFlag(false, SliceViewSelectMode)
	tv.SetFlag(true, SliceViewShowIndex)
	tv.SetFlag(true, SliceViewReadOnlyKeyNav)

	tv.HandleSliceViewEvents()

	tv.Lay = gi.LayoutVert
	tv.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.FocusWithinable)
		s.SetStretchMax()
	})
	tv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(tv) {
		case "frame": // slice frame
			sf := w.(*gi.Frame)
			sf.Lay = gi.LayoutVert
			sf.Style(func(s *styles.Style) {
				s.SetMinPrefWidth(units.Ch(20))
				s.Overflow = styles.OverflowScroll // this still gives it true size during PrefSize
				s.SetStretchMax()                  // for this to work, ALL layers above need it too
				s.Border.Style.Set(styles.BorderNone)
				s.Margin.Set()
				s.Padding.Set()
			})
		case "frame/header": // slice header
			sh := w.(*gi.Frame)
			sh.Lay = gi.LayoutHoriz
			gi.ToolbarStyles(sh)
			sh.Style(func(s *styles.Style) {
				s.Spacing.Dp(0)
				s.MaxWidth.Dp(0)
				s.Overflow = styles.OverflowHidden // no scrollbars!
			})
		case "frame/grid-lay": // grid layout
			gl := w.(*gi.Layout)
			gl.Lay = gi.LayoutHoriz
			w.Style(func(s *styles.Style) {
				gl.SetStretchMax() // for this to work, ALL layers above need it too
			})
		case "frame/grid-lay/grid": // slice grid
			sg := w.(*gi.Frame)
			sg.Lay = gi.LayoutGrid
			sg.Stripes = gi.RowStripes
			sg.Style(func(s *styles.Style) {
				// this causes everything to get off, especially resizing: not taking it into account presumably:
				// sg.Spacing = gi.StdDialogVSpaceUnits

				nWidgPerRow, _ := tv.RowWidgetNs()
				s.Columns = nWidgPerRow
				s.SetMinPrefHeight(units.Em(6))
				s.SetStretchMax()                  // for this to work, ALL layers above need it too
				s.Overflow = styles.OverflowScroll // this still gives it true size during PrefSize
			})
		case "frame/grid-lay/scrollbar":
			sb := w.(*gi.Slider)
			sb.Style(func(s *styles.Style) {
				sb.Type = gi.SliderScrollbar
				s.SetFixedWidth(tv.Styles.ScrollBarWidth)
				s.SetStretchMaxHeight()
			})
			sb.OnChange(func(e events.Event) {
				updt := tv.UpdateStart()
				tv.StartIdx = int(sb.Value)
				tv.This().(SliceViewer).UpdateWidgets()
				tv.UpdateEndRender(updt)
			})

		}
		if w.Parent().PathFrom(tv) == "frame/grid-lay/grid" {
			switch {
			case strings.HasPrefix(w.Name(), "index-"):
				w.Style(func(s *styles.Style) {
					s.MinWidth.Em(1.5)
					s.Padding.Right.Dp(4)
					s.Text.Align = styles.AlignRight
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
					idx := grr.Log(strconv.Atoi(istr))
					fli := grr.Log(strconv.Atoi(fstr))
					si := tv.StartIdx + idx
					if si < tv.SliceSize {
						tv.This().(SliceViewer).StyleRow(w, si, fli)
					}
				})
			}
		}
		if w.Parent().PathFrom(tv) == "frame/header" {
			w.Style(func(s *styles.Style) {
				s.Overflow = styles.OverflowHidden // no scrollbars!
			})
		}
	})
}

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice (does Update if already viewing).
func (tv *TableView) SetSlice(sl any) *TableView {
	if laser.AnyIsNil(sl) {
		tv.Slice = nil
		return tv
	}
	if tv.Slice == sl && tv.IsConfiged() {
		tv.Update()
		return tv
	}
	if !tv.IsReadOnly() {
		tv.SelIdx = -1
	}
	tv.StartIdx = 0
	tv.SortIdx = -1
	tv.SortDesc = false
	slpTyp := reflect.TypeOf(sl)
	if slpTyp.Kind() != reflect.Ptr {
		slog.Error("TableView requires that you pass a pointer to a slice of struct elements, but type is not a Ptr", "type", slpTyp)
		return tv
	}
	if slpTyp.Elem().Kind() != reflect.Slice {
		slog.Error("TableView requires that you pass a pointer to a slice of struct elements, but ptr doesn't point to a slice", "type", slpTyp.Elem())
		return tv
	}
	tv.Slice = sl
	tv.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(tv.Slice))
	struTyp := tv.StructType()
	if struTyp.Kind() != reflect.Struct {
		slog.Error("TableView requires that you pass a slice of struct elements, but type is not a Struct", "type", struTyp.String())
		return tv
	}
	tv.ElVal = laser.OnePtrValue(laser.SliceElValue(sl))
	updt := tv.UpdateStart()
	tv.ResetSelectedIdxs()
	tv.SetFlag(false, SliceViewSelectMode)
	tv.UpdateEnd(updt)
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
	laser.FlatFieldsTypeFunc(styp, func(typ reflect.Type, fld reflect.StructField) bool {
		if !fld.IsExported() {
			return true
		}
		tvtag := fld.Tag.Get("tableview")
		add := true
		if tvtag != "" {
			if tvtag == "-" {
				add = false
			} else if tvtag == "-select" && tv.IsReadOnly() {
				add = false
			} else if tvtag == "-edit" && !tv.IsReadOnly() {
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
					tv.VisFields = append(tv.VisFields, rfld)
				} else {
					fmt.Printf("TableView: Field name: %v is ambiguous from base struct type: %v, cannot be used in view!\n", fld.Name, styp.String())
				}
			} else {
				tv.VisFields = append(tv.VisFields, fld)
			}
		}
		return true
	})
	tv.NVisFields = len(tv.VisFields)
}

// IsConfiged returns true if the widget is fully configured
func (tv *TableView) IsConfiged() bool {
	if len(tv.Kids) == 0 {
		return false
	}
	sf := tv.SliceFrame()
	if len(sf.Kids) == 0 {
		return false
	}
	return true
}

// Config configures the view
func (tv *TableView) ConfigWidget(sc *gi.Scene) {
	tv.ConfigTableView(sc)
}

func (tv *TableView) ConfigTableView(sc *gi.Scene) {
	if tv.Is(SliceViewConfiged) {
		if tv.NeedsConfigRows() {
			tv.This().(SliceViewer).ConfigRows(sc)
		} else {
			tv.This().(SliceViewer).UpdateWidgets()
		}
		return
	}
	updt := tv.UpdateStart()
	tv.ConfigFrame(sc)
	tv.This().(SliceViewer).ConfigOneRow(sc)
	tv.ConfigScroll()
	tv.ApplyStyleTree(sc)
	tv.UpdateEndLayout(updt)
}

func (tv *TableView) ConfigFrame(sc *gi.Scene) {
	tv.SetFlag(true, SliceViewConfiged)
	sf := gi.NewFrame(tv, "frame").SetLayout(gi.LayoutVert)
	sf.SetFlag(true, gi.LayoutNoKeys)
	gi.NewFrame(sf, "header").SetLayout(gi.LayoutHoriz)
	gl := gi.NewLayout(sf, "grid-lay").SetLayout(gi.LayoutHoriz)
	gl.SetFlag(true, gi.LayoutNoKeys)
	gi.NewFrame(gl, "grid").SetLayout(gi.LayoutGrid)
	gi.NewSlider(gl, "scrollbar")
	tv.ConfigHeader(sc)
}

// ConfigOneRow configures one row for initial row height measurement
func (tv *TableView) ConfigOneRow(sc *gi.Scene) {
	sg := tv.This().(SliceViewer).SliceGrid()
	sg.SetFlag(true, gi.LayoutNoKeys)
	if sg.HasChildren() {
		return
	}
	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	tv.VisRows = 0
	if tv.IsNil() {
		return
	}

	tv.CacheVisFields()

	nWidgPerRow, idxOff := tv.RowWidgetNs()
	sg.Kids = make(ki.Slice, nWidgPerRow)

	tv.ConfigHeader(sc)

	itxt := "0"
	if tv.Is(SliceViewShowIndex) {
		labnm := "index-" + itxt
		idxlab := &gi.Label{}
		sg.SetChild(idxlab, 0, labnm)
		idxlab.Text = itxt
	}

	val := tv.ElVal
	stru := val.Interface()

	for fli := 0; fli < tv.NVisFields; fli++ {
		field := tv.VisFields[fli]
		fval := val.Elem().FieldByIndex(field.Index)
		vv := ToValue(fval.Interface(), "")
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetStructValue(fval.Addr(), stru, &field, tv.TmpSave, tv.ViewPath)
		vtyp := vv.WidgetType()
		valnm := fmt.Sprintf("value-%v.%v", fli, itxt)
		cidx := idxOff + fli
		w := ki.NewOfType(vtyp).(gi.Widget)
		sg.SetChild(w, cidx, valnm)
		vv.ConfigWidget(w, sc)
	}

	if !tv.IsReadOnly() {
		cidx := tv.NVisFields + idxOff
		if !tv.Is(SliceViewNoAdd) {
			addnm := "add-" + itxt
			addbt := gi.Button{}
			sg.SetChild(&addbt, cidx, addnm)
			addbt.SetType(gi.ButtonAction)
			addbt.SetIcon(icons.Add)
			cidx++
		}
		if !tv.Is(SliceViewNoDelete) {
			delnm := "del-" + itxt
			delbt := gi.Button{}
			sg.SetChild(&delbt, cidx, delnm)
			delbt.SetType(gi.ButtonAction)
			delbt.SetIcon(icons.Delete)
			delbt.Style(func(s *styles.Style) {
				s.Color = colors.Scheme.Error.Base
			})
			cidx++
		}
	}
	if tv.SortIdx >= 0 {
		tv.SortSlice()
	}
}

func (tv *TableView) ConfigHeaderStyleWidth(w *gi.WidgetBase, sg *gi.Frame, spc float32, idx int) {
	if w.Parts != nil {
		w.Parts.Style(func(s *styles.Style) {
			s.Overflow = styles.OverflowHidden // no scrollbars!
		})
	}
	w.Style(func(s *styles.Style) {
		s.Overflow = styles.OverflowHidden // no scrollbars!
		gd := sg.GridData[gi.Col]
		if gd == nil {
			return
		}
		if len(gd) > idx {
			wd := gd[idx].AllocSize - spc
			s.SetFixedWidth(units.Dot(wd))
		}
	})
}

func (tv *TableView) ConfigHeader(sc *gi.Scene) {
	sgh := tv.SliceHeader()
	if sgh.HasChildren() || tv.NVisFields == 0 {
		return
	}
	hcfg := ki.Config{}
	if tv.Is(SliceViewShowIndex) {
		hcfg.Add(gi.LabelType, "head-idx")
	}
	for fli := 0; fli < tv.NVisFields; fli++ {
		fld := tv.VisFields[fli]
		labnm := "head-" + fld.Name
		hcfg.Add(gi.ButtonType, labnm)
	}
	if !tv.IsReadOnly() {
		hcfg.Add(gi.LabelType, "head-add")
		hcfg.Add(gi.LabelType, "head-del")
	}
	sgh.ConfigChildren(hcfg) // headers SHOULD be unique, but with labels..
	sg := tv.SliceGrid()
	spc := sgh.Styles.Spacing.Dots
	_, idxOff := tv.RowWidgetNs()
	nfld := tv.NVisFields
	if tv.Is(SliceViewShowIndex) {
		lbl := sgh.Child(0).(*gi.Label)
		lbl.Text = "Index"
		tv.ConfigHeaderStyleWidth(lbl.AsWidget(), sg, spc, 0)
	}
	for fli := 0; fli < nfld; fli++ {
		fli := fli
		field := tv.VisFields[fli]
		hdr := sgh.Child(idxOff + fli).(*gi.Button)
		hdr.SetType(gi.ButtonAction)
		hdr.SetText(field.Name)
		if fli == tv.SortIdx {
			if tv.SortDesc {
				hdr.SetIcon(icons.KeyboardArrowDown)
			} else {
				hdr.SetIcon(icons.KeyboardArrowUp)
			}
		}
		hdr.Tooltip = field.Name + " (click to sort by)"
		dsc := field.Tag.Get("desc")
		if dsc != "" {
			hdr.Tooltip += ": " + dsc
		}
		hdr.OnClick(func(e events.Event) {
			tv.SortSliceAction(fli)
		})
		tv.ConfigHeaderStyleWidth(hdr.AsWidget(), sg, spc, fli+idxOff)
	}
	if !tv.IsReadOnly() {
		cidx := tv.NVisFields + idxOff
		if !tv.Is(SliceViewNoAdd) {
			lbl := sgh.Child(cidx).(*gi.Label)
			lbl.Text = "+"
			lbl.Tooltip = "insert row"
			tv.ConfigHeaderStyleWidth(lbl.AsWidget(), sg, spc, cidx)
			cidx++
		}
		if !tv.Is(SliceViewNoDelete) {
			lbl := sgh.Child(cidx).(*gi.Label)
			lbl.Text = "-"
			lbl.Tooltip = "delete row"
			tv.ConfigHeaderStyleWidth(lbl.AsWidget(), sg, spc, cidx)
		}
	}
}

// SliceFrame returns the outer frame widget, which contains all the header,
// fields and values
func (tv *TableView) SliceFrame() *gi.Frame {
	return tv.ChildByName("frame", 0).(*gi.Frame)
}

// GridLayout returns the SliceGrid grid-layout widget, with grid and scrollbar
func (tv *TableView) GridLayout() *gi.Layout {
	if !tv.IsConfiged() {
		return nil
	}
	return tv.SliceFrame().ChildByName("grid-lay", 0).(*gi.Layout)
}

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values, within SliceFrame
func (tv *TableView) SliceGrid() *gi.Frame {
	if !tv.IsConfiged() {
		return nil
	}
	return tv.GridLayout().ChildByName("grid", 0).(*gi.Frame)
}

// ScrollBar returns the SliceGrid scrollbar
func (tv *TableView) ScrollBar() *gi.Slider {
	return tv.GridLayout().ChildByName("scrollbar", 1).(*gi.Slider)
}

// SliceHeader returns the Frame header for slice grid
func (tv *TableView) SliceHeader() *gi.Frame {
	return tv.SliceFrame().Child(0).(*gi.Frame)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (tv *TableView) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 1 + tv.NVisFields
	if !tv.IsReadOnly() {
		if !tv.Is(SliceViewNoAdd) {
			nWidgPerRow += 1
		}
		if !tv.Is(SliceViewNoDelete) {
			nWidgPerRow += 1
		}
	}
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
func (tv *TableView) ConfigRows(sc *gi.Scene) {
	sg := tv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	sg.SetFlag(true, gi.LayoutNoKeys)

	updt := sg.UpdateStart()
	defer sg.UpdateEndLayout(updt)

	tv.ViewMuLock()
	defer tv.ViewMuUnlock()

	sg.DeleteChildren(ki.DestroyKids)
	tv.Values = nil
	tv.VisRows = 0

	if tv.IsNil() {
		return
	}

	tv.VisRows, tv.RowHeight, tv.LayoutHeight = tv.VisRowsAvail()

	nWidgPerRow, idxOff := tv.RowWidgetNs()
	nWidg := nWidgPerRow * tv.VisRows

	tv.Values = make([]Value, tv.NVisFields*tv.VisRows)
	sg.Kids = make(ki.Slice, nWidg)

	for i := 0; i < tv.VisRows; i++ {
		i := i
		si := i
		ridx := i * nWidgPerRow
		var val reflect.Value
		if si < tv.SliceSize {
			val = laser.OnePtrUnderlyingValue(tv.SliceNPVal.Index(si)) // deal with pointer lists
		} else {
			val = tv.ElVal
		}
		if laser.ValueIsZero(val) {
			val = tv.ElVal
		}
		stru := val.Interface()

		idxlab := &gi.Label{}
		itxt := strconv.Itoa(i)
		sitxt := strconv.Itoa(si)
		labnm := "index-" + itxt
		if tv.Is(SliceViewShowIndex) {
			idxlab = &gi.Label{}
			sg.SetChild(idxlab, ridx, labnm)
			idxlab.OnSelect(func(e events.Event) {
				e.SetHandled()
				tv.UpdateSelectRow(i)
			})
			idxlab.SetText(sitxt)
		}

		vpath := tv.ViewPath + "[" + sitxt + "]"
		if lblr, ok := tv.Slice.(gi.SliceLabeler); ok {
			slbl := lblr.ElemLabel(si)
			if slbl != "" {
				vpath = tv.ViewPath + "[" + slbl + "]"
			}
		}
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
			vv.ConfigWidget(w, sc)
			wb := w.AsWidget()
			wb.OnSelect(func(e events.Event) {
				e.SetHandled()
				tv.UpdateSelectRow(i)
			})

			if tv.IsReadOnly() {
				w.AsWidget().SetReadOnly(true)
			} else {
				vvb := vv.AsValueBase()
				vvb.OnChange(func(e events.Event) {
					tv.SetChanged()
				})
			}
		}

		if !tv.IsReadOnly() {
			cidx := ridx + tv.NVisFields + idxOff
			if !tv.Is(SliceViewNoAdd) {
				addnm := fmt.Sprintf("add-%v", itxt)
				addact := gi.Button{}
				sg.SetChild(&addact, cidx, addnm)
				addact.SetType(gi.ButtonAction)
				addact.SetIcon(icons.Add)
				addact.Tooltip = "insert a new element at this index"
				addact.OnClick(func(e events.Event) {
					tv.SliceNewAtRow(i + 1)
				})
				cidx++
			}
			if !tv.Is(SliceViewNoDelete) {
				delnm := fmt.Sprintf("del-%v", itxt)
				delact := gi.Button{}
				sg.SetChild(&delact, cidx, delnm)
				delact.SetType(gi.ButtonAction)
				delact.SetIcon(icons.Delete)
				delact.Tooltip = "delete this element"
				delact.OnClick(func(e events.Event) {
					tv.SliceDeleteAtRow(i)
				})
				cidx++
			}
		}
	}
	tv.This().(SliceViewer).UpdateWidgets()
}

// UpdateWidgets updates the row widget display to
// represent the current state of the slice data,
// including which range of data is being displayed.
// This is called for scrolling, navigation etc.
func (tv *TableView) UpdateWidgets() {
	sg := tv.This().(SliceViewer).SliceGrid()
	if sg == nil || tv.VisRows == 0 || !sg.HasChildren() {
		return
	}
	// sc := tv.Sc

	updt := sg.UpdateStart()
	defer sg.UpdateEndLayout(updt)

	tv.ViewMuLock()
	defer tv.ViewMuUnlock()

	nWidgPerRow, idxOff := tv.RowWidgetNs()

	tv.UpdateStartIdx()
	for i := 0; i < tv.VisRows; i++ {
		i := i
		ridx := i * nWidgPerRow
		si := tv.StartIdx + i // slice idx

		var idxlab *gi.Label
		if tv.Is(SliceViewShowIndex) {
			idxlab = sg.Kids[ridx].(*gi.Label)
			idxlab.SetText(strconv.Itoa(si))
			idxlab.SetNeedsRender()
		}

		sitxt := strconv.Itoa(si)
		vpath := tv.ViewPath + "[" + sitxt + "]"
		if lblr, ok := tv.Slice.(gi.SliceLabeler); ok {
			slbl := lblr.ElemLabel(si)
			if slbl != "" {
				vpath = tv.ViewPath + "[" + slbl + "]"
			}
		}
		for fli := 0; fli < tv.NVisFields; fli++ {
			fli := fli
			field := tv.VisFields[fli]
			cidx := ridx + idxOff + fli
			w := sg.Kids[cidx].(gi.Widget)

			var val reflect.Value
			if si < tv.SliceSize {
				val = laser.OnePtrUnderlyingValue(tv.SliceNPVal.Index(si)) // deal with pointer lists
				if laser.ValueIsZero(val) {
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

			if si < tv.SliceSize {
				w.SetState(false, states.Invisible)
				issel := tv.IdxIsSelected(si)
				if tv.IsReadOnly() {
					w.AsWidget().SetReadOnly(true)
				}
				w.AsWidget().SetSelected(issel)
			} else {
				w.SetState(true, states.Invisible)
				w.AsWidget().SetSelected(false)
				if tv.Is(SliceViewShowIndex) {
					idxlab.SetState(true, states.Invisible)
					idxlab.SetSelected(false)
				}
			}
		}

		if !tv.IsReadOnly() {
			cidx := ridx + tv.NVisFields + idxOff
			invis := true
			if si < tv.SliceSize {
				invis = false
			}
			if !tv.Is(SliceViewNoAdd) {
				addact := sg.Kids[cidx].(*gi.Button)
				addact.SetState(invis, states.Invisible)
				cidx++
			}
			if !tv.Is(SliceViewNoDelete) {
				delact := sg.Kids[cidx].(*gi.Button)
				delact.SetState(invis, states.Invisible)
				cidx++
			}
		}
	}

	if tv.SelField != "" && tv.SelVal != nil {
		tv.SelIdx, _ = StructSliceIdxByValue(tv.Slice, tv.SelField, tv.SelVal)
	}
	if tv.IsReadOnly() && tv.SelIdx >= 0 {
		tv.SelectIdx(tv.SelIdx)
	}
	tv.UpdateScroll()
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
	tv.ViewMuUnlock()
	tv.SetChanged()
	tv.Update()
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
	tv.Update()
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
	tv.Update()
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

// SetSortField sets sorting to happen on given field and direction -- see
// SortFieldName for details
func (tv *TableView) SetSortFieldName(nm string) {
	if nm == "" {
		return
	}
	spnm := strings.Split(nm, ":")
	for fli := 0; fli < tv.NVisFields; fli++ {
		fld := tv.VisFields[fli]
		if fld.Name == spnm[0] {
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
	if w.ScBBox != (image.Rectangle{}) {
		return w, true
	}
	ridx := nWidgPerRow * row
	for fli := 0; fli < tv.NVisFields; fli++ {
		w := sg.Child(ridx + idxOff + fli).(gi.Widget).AsWidget()
		if w.ScBBox != (image.Rectangle{}) {
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
			w.GrabFocus()
			return w
		}
	}
	return nil
}

// SelectRowWidgets sets the selection state of given row of widgets
func (tv *TableView) SelectRowWidgets(row int, sel bool) {
	if row < 0 {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	sg := tv.SliceGrid()
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	ridx := row * nWidgPerRow
	for fli := 0; fli < tv.NVisFields; fli++ {
		seldx := ridx + idxOff + fli
		if sg.Kids.IsValidIndex(seldx) == nil {
			w := sg.Child(seldx).(gi.Widget).AsWidget()
			w.SetSelected(sel)
		}
	}
	if tv.Is(SliceViewShowIndex) {
		if sg.Kids.IsValidIndex(ridx) == nil {
			w := sg.Child(ridx).(gi.Widget).AsWidget()
			w.SetSelected(sel)
		}
	}
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
			tv.UpdateSelectIdx(idx, true)
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
		err := fmt.Errorf("gi.StructSliceRowByValue: field name: %v not found\n", fldName)
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
	val := laser.OnePtrUnderlyingValue(tv.SliceNPVal.Index(idx))
	stru := val.Interface()
	tynm := laser.NonPtrType(val.Type()).Name()
	lbl := gi.ToLabel(stru)
	if lbl != "" {
		tynm += ": " + lbl
	}
	d := gi.NewDialog(tv).Title(tynm).FullWindow(true)
	NewStructView(d).SetStruct(stru)
	d.Run()
}

func (tv *TableView) StdCtxtMenu(m *gi.Scene, idx int) {
	if tv.Is(SliceViewIsArray) {
		return
	}
	tv.SliceViewBase.StdCtxtMenu(m, idx)
	gi.NewSeparator(m, "sep-edit")
	gi.NewButton(m, "edit").SetText("Edit").SetData(idx).
		OnClick(func(e events.Event) {
			tv.EditIdx(idx)
		})
}
