// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"image"
	"log"
	"reflect"
	"strconv"
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/cursor"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// todo:
// * search option, both as a search field and as simple type-to-search

// TableView represents a slice-of-structs as a table, where the fields are
// the columns, within an overall frame.  It has two modes, determined by
// Inactive flag: if Inactive, it functions as a mutually-exclusive item
// selector, highlighting the selected row and emitting a WidgetSig
// WidgetSelected signal, and TableViewDoubleClick for double clicks (can be
// used for closing dialogs).  If !Inactive, it is a full-featured editor with
// multiple-selection, cut-and-paste, and drag-and-drop, reporting each action
// taken using the TableViewSig signals
// Automatically has a toolbar with Slice ToolBar props if defined
// set prop toolbar = false to turn off
type TableView struct {
	SliceViewBase

	// [view: -] optional styling function
	StyleFunc TableViewStyleFunc `copy:"-" view:"-" json:"-" xml:"-" desc:"optional styling function"`

	// [view: -] current selection field -- initially select value in this field
	SelField string `copy:"-" view:"-" json:"-" xml:"-" desc:"current selection field -- initially select value in this field"`

	// current sort index
	SortIdx int `desc:"current sort index"`

	// whether current sort order is descending
	SortDesc bool `desc:"whether current sort order is descending"`

	// [view: -] struct type for each row
	StruType reflect.Type `copy:"-" view:"-" json:"-" xml:"-" desc:"struct type for each row"`

	// [view: -] the visible fields
	VisFields []reflect.StructField `copy:"-" view:"-" json:"-" xml:"-" desc:"the visible fields"`

	// [view: -] number of visible fields
	NVisFields int `copy:"-" view:"-" json:"-" xml:"-" desc:"number of visible fields"`
}

// check for interface impl
var _ SliceViewer = (*TableView)(nil)

// TableViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view.  If style properties are set
// then you must call widg.AsNode2dD().SetFullReRender() to trigger
// re-styling during re-render
type TableViewStyleFunc func(tv *TableView, slice any, widg gi.Node2D, row, col int, vv ValueView)

func (tv *TableView) OnInit() {
	tv.Lay = gi.LayoutVert
	tv.AddStyler(func(w *gi.WidgetBase, s *styles.Style) {
		tv.Spacing = gi.StdDialogVSpaceUnits
		s.SetStretchMax()
	})
}

func (tv *TableView) OnChildAdded(child ki.Ki) {
	if w := gi.AsWidget(child); w != nil {
		switch w.Name() {
		case "frame": // slice frame
			sf := child.(*gi.Frame)
			sf.Lay = gi.LayoutVert
			sf.AddStyler(func(w *gi.WidgetBase, s *styles.Style) {
				s.SetMinPrefWidth(units.Ch(20))
				s.Overflow = styles.OverflowScroll // this still gives it true size during PrefSize
				s.SetStretchMax()                  // for this to work, ALL layers above need it too
				s.Border.Style.Set(styles.BorderNone)
				s.Margin.Set()
				s.Padding.Set()
			})
		case "header": // slice header
			sh := child.(*gi.ToolBar)
			sh.Lay = gi.LayoutHoriz
			sh.AddStyler(func(w *gi.WidgetBase, s *styles.Style) {
				sh.Spacing.SetPx(0)
				s.Overflow = styles.OverflowHidden // no scrollbars!
			})
		case "grid-lay": // grid layout
			gl := child.(*gi.Layout)
			gl.Lay = gi.LayoutHoriz
			w.AddStyler(func(w *gi.WidgetBase, s *styles.Style) {
				gl.SetStretchMax() // for this to work, ALL layers above need it too
			})
		case "grid": // slice grid
			sg := child.(*gi.Frame)
			sg.Lay = gi.LayoutGrid
			sg.Stripes = gi.RowStripes
			sg.AddStyler(func(w *gi.WidgetBase, s *styles.Style) {
				// this causes everything to get off, especially resizing: not taking it into account presumably:
				// sg.Spacing = gi.StdDialogVSpaceUnits

				nWidgPerRow, _ := tv.RowWidgetNs()
				s.Columns = nWidgPerRow
				s.SetMinPrefHeight(units.Em(6))
				s.SetStretchMax()                  // for this to work, ALL layers above need it too
				s.Overflow = styles.OverflowScroll // this still gives it true size during PrefSize
			})
		}
		// STYTODO: set header sizes here (see LayoutHeader)
		// if _, ok := child.(*gi.Label); ok && w.Parent().Name() == "header" {
		// 	w.AddStyler(func(w *gi.WidgetBase, s *styles.Style) {
		// 		spc := tv.SliceHeader().Spacing.Dots
		// 		ip, _ := w.IndexInParent()
		// 		s.SetMinPrefWidth(units.Dot())
		// 	})
		// }
		if w.Parent().Name() == "grid" && strings.HasPrefix(w.Name(), "index-") {
			w.AddStyler(func(w *gi.WidgetBase, s *styles.Style) {
				s.MinWidth.SetEm(1.5)
				s.Padding.Right.SetPx(4 * gi.Prefs.DensityMul())
				s.Text.Align = styles.AlignRight
			})
		}
	}
}

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice (does Update if already viewing).
func (tv *TableView) SetSlice(sl any) {
	if laser.IfaceIsNil(sl) {
		tv.Slice = nil
		return
	}
	if tv.Slice == sl && tv.IsConfiged() {
		tv.Update()
		return
	}
	if !tv.IsDisabled() {
		tv.SelectedIdx = -1
	}
	tv.StartIdx = 0
	tv.SortIdx = -1
	tv.SortDesc = false
	slpTyp := reflect.TypeOf(sl)
	if slpTyp.Kind() != reflect.Ptr {
		log.Printf("TableView requires that you pass a pointer to a slice of struct elements -- type is not a Ptr: %v\n", slpTyp.String())
		return
	}
	if slpTyp.Elem().Kind() != reflect.Slice {
		log.Printf("TableView requires that you pass a pointer to a slice of struct elements -- ptr doesn't point to a slice: %v\n", slpTyp.Elem().String())
		return
	}
	tv.Slice = sl
	tv.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(tv.Slice))
	struTyp := tv.StructType()
	if struTyp.Kind() != reflect.Struct {
		log.Printf("TableView requires that you pass a slice of struct elements -- type is not a Struct: %v\n", struTyp.String())
		return
	}
	updt := tv.UpdateStart()
	tv.ResetSelectedIdxs()
	tv.SelectMode = false
	tv.SetFullReRender()

	tv.Config()
	tv.UpdateEnd(updt)
}

var TableViewProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
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
	tv.VisFields = make([]reflect.StructField, 0, 20)
	laser.FlatFieldsTypeFunc(styp, func(typ reflect.Type, fld reflect.StructField) bool {
		if !fld.IsExported() {
			return true
		}
		tvtag := fld.Tag.Get("tableview")
		add := true
		if tvtag != "" {
			if tvtag == "-" {
				add = false
			} else if tvtag == "-select" && tv.IsDisabled() {
				add = false
			} else if tvtag == "-edit" && !tv.IsDisabled() {
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
func (tv *TableView) ConfigWidget(vp *Scene) {
	config := ki.Config{}
	config.Add(gi.TypeToolBar, "toolbar")
	config.Add(gi.FrameType, "frame")
	mods, updt := tv.ConfigChildren(config)
	tv.ConfigSliceGrid()
	tv.ConfigToolbar()
	if mods {
		tv.SetFullReRender()
		tv.UpdateEnd(updt)
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
func (tv *TableView) ScrollBar() *gi.ScrollBar {
	return tv.GridLayout().ChildByName("scrollbar", 1).(*gi.ScrollBar)
}

// SliceHeader returns the Toolbar header for slice grid
func (tv *TableView) SliceHeader() *gi.ToolBar {
	return tv.SliceFrame().Child(0).(*gi.ToolBar)
}

// ToolBar returns the toolbar widget
func (tv *TableView) ToolBar() *gi.ToolBar {
	return tv.ChildByName("toolbar", 0).(*gi.ToolBar)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (tv *TableView) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 1 + tv.NVisFields
	if !tv.IsDisabled() {
		if !tv.NoAdd {
			nWidgPerRow += 1
		}
		if !tv.NoDelete {
			nWidgPerRow += 1
		}
	}
	idxOff = 1
	if !tv.ShowIndex {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

// ConfigSliceGrid configures the SliceGrid for the current slice
// this is only called by global Config and updates are guarded by that
func (tv *TableView) ConfigSliceGrid() {
	sg := tv.SliceFrame()
	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	sgf := tv.This().(SliceViewer).SliceGrid()
	if sgf != nil {
		sgf.DeleteChildren(ki.DestroyKids)
	}

	if laser.IfaceIsNil(tv.Slice) {
		return
	}

	tv.CacheVisFields()

	sz := tv.This().(SliceViewer).UpdtSliceSize()
	if sz == 0 {
		return
	}

	nWidgPerRow, idxOff := tv.RowWidgetNs()

	sgcfg := ki.Config{}
	sgcfg.Add(gi.TypeToolBar, "header")
	sgcfg.Add(gi.LayoutType, "grid-lay")
	sg.ConfigChildren(sgcfg)

	sgh := tv.SliceHeader()

	gl := tv.GridLayout()
	gconfig := ki.Config{}
	gconfig.Add(gi.FrameType, "grid")
	gconfig.Add(gi.TypeScrollBar, "scrollbar")
	gl.ConfigChildren(gconfig) // covered by above

	sgf = tv.This().(SliceViewer).SliceGrid()

	// Configure Header
	hcfg := ki.Config{}
	if tv.ShowIndex {
		hcfg.Add(gi.LabelType, "head-idx")
	}
	for fli := 0; fli < tv.NVisFields; fli++ {
		fld := tv.VisFields[fli]
		labnm := "head-" + fld.Name
		hcfg.Add(gi.ActionType, labnm)
	}
	if !tv.IsDisabled() {
		hcfg.Add(gi.LabelType, "head-add")
		hcfg.Add(gi.LabelType, "head-del")
	}
	sgh.ConfigChildren(hcfg) // headers SHOULD be unique, but with labels..

	// at this point, we make one dummy row to get size of widgets

	sgf.Kids = make(ki.Slice, nWidgPerRow)

	itxt := "0"
	labnm := "index-" + itxt

	if tv.ShowIndex {
		lbl := sgh.Child(0).(*gi.Label)
		lbl.Text = "Index"

		idxlab := &gi.Label{}
		sgf.SetChild(idxlab, 0, labnm)
		idxlab.Text = itxt
	}

	for fli := 0; fli < tv.NVisFields; fli++ {
		field := tv.VisFields[fli]
		hdr := sgh.Child(idxOff + fli).(*gi.Action)
		hdr.SetText(field.Name)
		if fli == tv.SortIdx {
			if tv.SortDesc {
				hdr.SetIcon(icons.KeyboardArrowDown)
			} else {
				hdr.SetIcon(icons.KeyboardArrowUp)
			}
		}
		hdr.Data = fli
		hdr.Tooltip = field.Name + " (click to sort by)"
		dsc := field.Tag.Get("desc")
		if dsc != "" {
			hdr.Tooltip += ": " + dsc
		}
		hdr.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tvv := recv.Embed(TypeTableView).(*TableView)
			act := send.(*gi.Action)
			fldIdx := act.Data.(int)
			tvv.SortSliceAction(fldIdx)
		})

		val := laser.OnePtrUnderlyingValue(tv.SliceNPVal.Index(0)) // deal with pointer lists
		stru := val.Interface()
		fval := val.Elem().FieldByIndex(field.Index)
		vv := ToValueView(fval.Interface(), "")
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetStructValue(fval.Addr(), stru, &field, tv.TmpSave, tv.ViewPath)
		vtyp := vv.WidgetType()
		valnm := fmt.Sprintf("value-%v.%v", fli, itxt)
		cidx := idxOff + fli
		widg := ki.NewOfType(vtyp).(gi.Node2D)
		sgf.SetChild(widg, cidx, valnm)
		vv.ConfigWidget(widg)
	}

	if !tv.IsDisabled() {
		cidx := tv.NVisFields + idxOff
		if !tv.NoAdd {
			lbl := sgh.Child(cidx).(*gi.Label)
			lbl.Text = "+"
			lbl.Tooltip = "insert row"
			addnm := "add-" + itxt
			addact := gi.Action{}
			sgf.SetChild(&addact, cidx, addnm)
			addact.SetIcon(icons.Add)
			cidx++
		}
		if !tv.NoDelete {
			lbl := sgh.Child(cidx).(*gi.Label)
			lbl.Text = "-"
			lbl.Tooltip = "delete row"
			delnm := "del-" + itxt
			delact := gi.Action{}
			sgf.SetChild(&delact, cidx, delnm)
			delact.SetIcon(icons.Delete)
			cidx++
		}
	}

	if tv.SortIdx >= 0 {
		tv.SortSlice()
	}
	tv.ConfigScroll()
}

// LayoutSliceGrid does the proper layout of slice grid depending on allocated size
// returns true if UpdateSliceGrid should be called after this
func (tv *TableView) LayoutSliceGrid() bool {
	sg := tv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return false
	}

	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	if laser.IfaceIsNil(tv.Slice) {
		sg.DeleteChildren(ki.DestroyKids)
		return false
	}

	tv.ViewMuLock()
	defer tv.ViewMuUnlock()

	sz := tv.This().(SliceViewer).UpdtSliceSize()
	if sz == 0 {
		sg.DeleteChildren(ki.DestroyKids)
		return false
	}

	nWidgPerRow, _ := tv.RowWidgetNs()
	if len(sg.GridData) > 0 && len(sg.GridData[gi.Row]) > 0 {
		tv.RowHeight = sg.GridData[gi.Row][0].AllocSize + sg.Spacing.Dots
	}
	if tv.Style.Font.Face == nil {
		tv.Style.Font = paint.OpenFont(tv.Style.FontRender(), &tv.Style.UnContext)
	}
	tv.RowHeight = mat32.Max(tv.RowHeight, tv.Style.Font.Face.Metrics.Height)

	mvp := tv.Sc
	if mvp != nil && mvp.HasFlag(int(gi.ScFlagPrefSizing)) {
		tv.VisRows = min(gi.LayoutPrefMaxRows, tv.SliceSize)
		tv.LayoutHeight = float32(tv.VisRows) * tv.RowHeight
	} else {
		sgHt := tv.AvailHeight()
		tv.LayoutHeight = sgHt
		if sgHt == 0 {
			return false
		}
		tv.VisRows = int(mat32.Floor(sgHt / tv.RowHeight))
	}
	tv.DispRows = min(tv.SliceSize, tv.VisRows)

	nWidg := nWidgPerRow * tv.DispRows

	if tv.Values == nil || sg.NumChildren() != nWidg {
		sg.DeleteChildren(ki.DestroyKids)

		tv.Values = make([]ValueView, tv.NVisFields*tv.DispRows)
		sg.Kids = make(ki.Slice, nWidg)
	}
	tv.ConfigScroll()
	tv.LayoutHeader()
	return true
}

// LayoutHeader updates the header layout based on field widths
func (tv *TableView) LayoutHeader() {
	// STYTODO: set these styles in stylers
	_, idxOff := tv.RowWidgetNs()
	nfld := tv.NVisFields + idxOff
	sgh := tv.SliceHeader()
	sgf := tv.SliceGrid()
	spc := sgh.Spacing.Dots
	gd := sgf.GridData[gi.Col]
	if gd == nil {
		return
	}
	sumwd := float32(0)
	for fli := 0; fli < nfld; fli++ {
		lbl := sgh.Child(fli).(gi.Node2D).AsWidget()
		wd := gd[fli].AllocSize - spc
		if fli == 0 {
			wd += spc
		}
		lbl.SetFixedWidth(units.Dot(wd))
		sumwd += wd
	}
	if !tv.IsDisabled() {
		mx := len(sgf.GridData[gi.Col])
		for fli := nfld; fli < mx; fli++ {
			lbl := sgh.Child(fli).(gi.Node2D).AsWidget()
			wd := gd[fli].AllocSize - spc
			lbl.SetFixedWidth(units.Dot(wd))
			sumwd += wd
		}
	}
	sgh.SetMinPrefWidth(units.Dot(sumwd + spc))
}

// UpdateSliceGrid updates grid display -- robust to any time calling
func (tv *TableView) UpdateSliceGrid() {
	sg := tv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}

	wupdt := tv.TopUpdateStart()
	defer tv.TopUpdateEnd(wupdt)

	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	if laser.IfaceIsNil(tv.Slice) {
		sg.DeleteChildren(ki.DestroyKids)
		return
	}

	tv.ViewMuLock()
	defer tv.ViewMuUnlock()

	sz := tv.This().(SliceViewer).UpdtSliceSize()
	if sz == 0 {
		sg.DeleteChildren(ki.DestroyKids)
		return
	}
	tv.DispRows = min(tv.SliceSize, tv.VisRows)

	nWidgPerRow, idxOff := tv.RowWidgetNs()
	nWidg := nWidgPerRow * tv.DispRows

	if tv.Values == nil || sg.NumChildren() != nWidg { // shouldn't happen..
		tv.ViewMuUnlock()
		tv.LayoutSliceGrid()
		tv.ViewMuLock()
		nWidg = nWidgPerRow * tv.DispRows
	}
	if sg.NumChildren() != nWidg || sg.NumChildren() == 0 {
		return
	}

	tv.UpdateStartIdx()

	for i := 0; i < tv.DispRows; i++ {
		ridx := i * nWidgPerRow
		si := tv.StartIdx + i // slice idx
		issel := tv.IdxIsSelected(si)
		val := laser.OnePtrUnderlyingValue(tv.SliceNPVal.Index(si)) // deal with pointer lists
		stru := val.Interface()

		itxt := strconv.Itoa(i)
		sitxt := strconv.Itoa(si)
		labnm := "index-" + itxt
		if tv.ShowIndex {
			var idxlab *gi.Label
			if sg.Kids[ridx] != nil {
				idxlab = sg.Kids[ridx].(*gi.Label)
			} else {
				idxlab = &gi.Label{}
				sg.SetChild(idxlab, ridx, labnm)
				idxlab.SetProp("tv-row", i)
				idxlab.Selectable = true
				idxlab.Redrawable = true
				idxlab.Style.Template = "giv.TableView.IndexLabel"
				idxlab.WidgetSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
					if sig == int64(gi.WidgetSelected) {
						wbb := send.(gi.Node2D).AsWidget()
						row := wbb.Prop("tv-row").(int)
						tvv := recv.Embed(TypeTableView).(*TableView)
						tvv.UpdateSelectRow(row, wbb.StateIs(states.Selected))
					}
				})
			}
			idxlab.SetSelected(issel)
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
			field := tv.VisFields[fli]
			fval := val.Elem().FieldByIndex(field.Index)
			vvi := i*tv.NVisFields + fli
			var vv ValueView
			if tv.Values[vvi] == nil {
				tags := ""
				if fval.Kind() == reflect.Slice || fval.Kind() == reflect.Map {
					tags = `view:"no-inline"`
				}
				vv = ToValueView(fval.Interface(), tags)
				tv.Values[vvi] = vv
			} else {
				vv = tv.Values[vvi]
			}
			if vv == nil {
				fmt.Printf("field: %v %v has nil valueview: %v -- should not happen -- fix ToValueView\n", fli, field.Name, fval.String())
				continue
			}
			vv.SetStructValue(fval.Addr(), stru, &field, tv.TmpSave, vpath)

			vtyp := vv.WidgetType()
			valnm := fmt.Sprintf("value-%v.%v", fli, itxt)
			cidx := ridx + idxOff + fli
			var widg gi.Node2D
			if sg.Kids[cidx] != nil {
				widg = sg.Kids[cidx].(gi.Node2D)
				vv.UpdateWidget()
				if tv.IsDisabled() {
					widg.AsNode2D().SetDisabled()
				}
				widg.AsNode2D().SetSelected(issel)
			} else {
				widg = ki.NewOfType(vtyp).(gi.Node2D)
				sg.SetChild(widg, cidx, valnm)
				vv.ConfigWidget(widg)
				wb := widg.AsWidget()
				if wb != nil {
					// totally not worth it now:
					// wb.Sty.Template = "giv.TableViewView.ItemWidget." + vtyp.Name()
					wb.SetProp("tv-row", i)
					wb.ClearSelected()
					wb.WidgetSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
						if sig == int64(gi.WidgetSelected) { // || sig == int64(gi.WidgetFocused) {
							wbb := send.(gi.Node2D).AsWidget()
							row := wbb.Prop("tv-row").(int)
							tvv := recv.Embed(TypeTableView).(*TableView)
							// if sig != int64(gi.WidgetFocused) || !tvv.InFocusGrab {
							tvv.UpdateSelectRow(row, wbb.StateIs(states.Selected))
							// }
						}
					})
				}
				if tv.IsDisabled() {
					widg.AsNode2D().SetDisabled()
				} else {
					vvb := vv.AsValueViewBase()
					vvb.ViewSig.ConnectOnly(tv.This(), // todo: do we need this?
						func(recv, send ki.Ki, sig int64, data any) {
							tvv, _ := recv.Embed(TypeTableView).(*TableView)
							tvv.SetChanged()
						})
				}
			}
			tv.This().(SliceViewer).StyleRow(tv.SliceNPVal, widg, si, fli, vv)
		}

		if !tv.IsDisabled() {
			cidx := ridx + tv.NVisFields + idxOff
			if !tv.NoAdd {
				if sg.Kids[cidx] == nil {
					addnm := fmt.Sprintf("add-%v", itxt)
					addact := gi.Action{}
					sg.SetChild(&addact, cidx, addnm)
					addact.SetIcon(icons.Add)
					addact.Tooltip = "insert a new element at this index"
					addact.Data = i
					addact.Style.Template = "giv.TableView.AddAction"
					addact.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
						act := send.(*gi.Action)
						tvv := recv.Embed(TypeTableView).(*TableView)
						tvv.SliceNewAtRow(act.Data.(int) + 1)
					})
				}
				cidx++
			}
			if !tv.NoDelete {
				if sg.Kids[cidx] == nil {
					delnm := fmt.Sprintf("del-%v", itxt)
					delact := gi.Action{}
					sg.SetChild(&delact, cidx, delnm)
					delact.SetIcon(icons.Delete)
					delact.Tooltip = "delete this element"
					delact.Data = i
					delact.Style.Template = "giv.TableView.DelAction"
					delact.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
						act := send.(*gi.Action)
						tvv := recv.Embed(TypeTableView).(*TableView)
						tvv.SliceDeleteAtRow(act.Data.(int), true)
					})
				}
				cidx++
			}
		}
	}

	if tv.SelField != "" && tv.SelVal != nil {
		tv.SelectedIdx, _ = StructSliceIdxByValue(tv.Slice, tv.SelField, tv.SelVal)
	}
	if tv.IsDisabled() && tv.SelectedIdx >= 0 {
		tv.SelectIdx(tv.SelectedIdx)
	}
	tv.UpdateScroll()
}

func (tv *TableView) StyleRow(svnp reflect.Value, widg gi.Node2D, idx, fidx int, vv ValueView) {
	if tv.StyleFunc != nil {
		tv.StyleFunc(tv, svnp.Interface(), widg, idx, fidx, vv)
	}
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (tv *TableView) SliceNewAt(idx int) {
	wupdt := tv.TopUpdateStart()
	defer tv.TopUpdateEnd(wupdt)

	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)

	tv.SliceNewAtSel(idx)
	laser.SliceNewAt(tv.Slice, idx)
	if idx < 0 {
		idx = tv.SliceSize
	}

	tv.This().(SliceViewer).UpdtSliceSize()

	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.SetChanged()
	tv.SetFullReRender()
	tv.ScrollBar().SetFullReRender()
	tv.This().(SliceViewer).LayoutSliceGrid()
	tv.This().(SliceViewer).UpdateSliceGrid()
	tv.ViewSig.Emit(tv.This(), 0, nil)
	tv.SliceViewSig.Emit(tv.This(), int64(SliceViewInserted), idx)
}

// SliceDeleteAt deletes element at given index from slice -- doupdt means
// call UpdateSliceGrid to update display
func (tv *TableView) SliceDeleteAt(idx int, doupdt bool) {
	if idx < 0 || idx >= tv.SliceSize {
		return
	}
	wupdt := tv.TopUpdateStart()
	defer tv.TopUpdateEnd(wupdt)

	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)

	tv.SliceDeleteAtSel(idx)

	laser.SliceDeleteAt(tv.Slice, idx)

	tv.This().(SliceViewer).UpdtSliceSize()

	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.SetChanged()
	if doupdt {
		tv.SetFullReRender()
		tv.ScrollBar().SetFullReRender()
		tv.This().(SliceViewer).LayoutSliceGrid()
		tv.This().(SliceViewer).UpdateSliceGrid()
	}
	tv.ViewSig.Emit(tv.This(), 0, nil)
	tv.SliceViewSig.Emit(tv.This(), int64(SliceViewDeleted), idx)
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
	goosi.TheApp.Cursor(tv.ParentRenderWin().RenderWin).Push(cursor.Wait)
	defer goosi.TheApp.Cursor(tv.ParentRenderWin().RenderWin).Pop()

	wupdt := tv.TopUpdateStart()
	defer tv.TopUpdateEnd(wupdt)

	updt := tv.UpdateStart()
	sgh := tv.SliceHeader()
	sgh.SetFullReRender()
	_, idxOff := tv.RowWidgetNs()

	ascending := true

	for fli := 0; fli < tv.NVisFields; fli++ {
		hdr := sgh.Child(idxOff + fli).(*gi.Action)
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
	tv.UpdateSliceGrid()
	tv.UpdateEnd(updt)
}

// ConfigToolbar configures the toolbar actions
func (tv *TableView) ConfigToolbar() {
	if laser.IfaceIsNil(tv.Slice) {
		return
	}
	if tv.ToolbarSlice == tv.Slice {
		return
	}
	if !tv.ShowToolBar {
		tv.ToolbarSlice = tv.Slice
		return
	}
	tb := tv.ToolBar()
	ndef := 2 // number of default actions
	if tv.isArray || tv.IsDisabled() || tv.NoAdd {
		ndef = 1
	}
	if len(*tb.Children()) < ndef {
		tb.SetStretchMaxWidth()
		tb.AddAction(gi.ActOpts{Label: "UpdtView", Icon: icons.Refresh, Tooltip: "update this TableView to reflect current state of table"},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				tvv := recv.Embed(TypeTableView).(*TableView)
				tvv.UpdateSliceGrid()
			})
		if ndef > 1 {
			tb.AddAction(gi.ActOpts{Label: "Add", Icon: icons.Add, Tooltip: "add a new element to the table"},
				tv.This(), func(recv, send ki.Ki, sig int64, data any) {
					tvv := recv.Embed(TypeTableView).(*TableView)
					tvv.SliceNewAt(-1)
				})
		}
	}
	sz := len(*tb.Children())
	if sz > ndef {
		for i := sz - 1; i >= ndef; i-- {
			tb.DeleteChildAtIndex(i, ki.DestroyKids)
		}
	}
	if HasToolBarView(tv.Slice) {
		ToolBarView(tv.Slice, tv.Sc, tb)
		tb.SetFullReRender()
	}
	tv.ToolbarSlice = tv.Slice
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

func (tv *TableView) DoLayout(vp *Scene, parBBox image.Rectangle, iter int) bool {
	redo := tv.Frame.DoLayout(vp, parBBox, iter)
	if !tv.IsConfiged() {
		return redo
	}
	tv.LayoutHeader()
	tv.SliceHeader().DoLayout(vp, parBBox, iter)
	return redo
}

// RowFirstVisWidget returns the first visible widget for given row (could be
// index or not) -- false if out of range
func (tv *TableView) RowFirstVisWidget(row int) (*gi.WidgetBase, bool) {
	if !tv.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	sg := tv.SliceGrid()
	widg := sg.Kids[row*nWidgPerRow].(gi.Node2D).AsWidget()
	if widg.ScBBox != (image.Rectangle{}) {
		return widg, true
	}
	ridx := nWidgPerRow * row
	for fli := 0; fli < tv.NVisFields; fli++ {
		widg := sg.Child(ridx + idxOff + fli).(gi.Node2D).AsWidget()
		if widg.ScBBox != (image.Rectangle{}) {
			return widg, true
		}
	}
	return nil, false
}

// RowGrabFocus grabs the focus for the first focusable widget in given row --
// returns that element or nil if not successful -- note: grid must have
// already rendered for focus to be grabbed!
func (tv *TableView) RowGrabFocus(row int) *gi.WidgetBase {
	if !tv.IsRowInBounds(row) || tv.InFocusGrab { // range check
		return nil
	}
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := tv.SliceGrid()
	// first check if we already have focus
	for fli := 0; fli < tv.NVisFields; fli++ {
		widg := sg.Child(ridx + idxOff + fli).(gi.Node2D).AsWidget()
		if widg.StateIs(states.Focused) || widg.ContainsFocus() {
			return widg
		}
	}
	tv.InFocusGrab = true
	defer func() { tv.InFocusGrab = false }()
	for fli := 0; fli < tv.NVisFields; fli++ {
		widg := sg.Child(ridx + idxOff + fli).(gi.Node2D).AsWidget()
		if widg.CanFocus() {
			widg.GrabFocus()
			return widg
		}
	}
	return nil
}

// SelectRowWidgets sets the selection state of given row of widgets
func (tv *TableView) SelectRowWidgets(row int, sel bool) {
	if row < 0 {
		return
	}
	wupdt := tv.TopUpdateStart()
	defer tv.TopUpdateEnd(wupdt)

	sg := tv.SliceGrid()
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	ridx := row * nWidgPerRow
	for fli := 0; fli < tv.NVisFields; fli++ {
		seldx := ridx + idxOff + fli
		if sg.Kids.IsValidIndex(seldx) == nil {
			widg := sg.Child(seldx).(gi.Node2D).AsNode2D()
			widg.SetSelected(sel)
			widg.UpdateSig()
		}
	}
	if tv.ShowIndex {
		if sg.Kids.IsValidIndex(ridx) == nil {
			widg := sg.Child(ridx).(gi.Node2D).AsNode2D()
			widg.SetSelected(sel)
			widg.UpdateSig()
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
		log.Println(err)
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
	StructViewDialog(tv.Scene, stru, DlgOpts{Title: tynm}, nil, nil)
}

func (tv *TableView) StdCtxtMenu(m *gi.Menu, idx int) {
	if tv.isArray {
		return
	}
	tv.SliceViewBase.StdCtxtMenu(m, idx)
	m.AddSeparator("sep-edit")
	m.AddAction(gi.ActOpts{Label: "Edit", Data: idx},
		tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tvv := recv.Embed(TypeTableView).(*TableView)
			tvv.EditIdx(data.(int))
		})
}
