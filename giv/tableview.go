// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"reflect"
	"sort"
	"strings"

	"github.com/chewxy/math32"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/filecat"
)

// todo:
// * search option, both as a search field and as simple type-to-search
// * popup menu option -- when user does right-mouse on item, a provided func is called
//   -- use in fileview
// * could have a native context menu for add / delete etc.
// * emit TableViewSigs

// TableView represents a slice-of-structs as a table, where the fields are
// the columns, within an overall frame.  It has two modes, determined by
// Inactive flag: if Inactive, it functions as a mutually-exclusive item
// selector, highlighting the selected row and emitting a WidgetSig
// WidgetSelected signal, and TableViewDoubleClick for double clicks (can be
// used for closing dialogs).  If !Inactive, it is a full-featured editor with
// multiple-selection, cut-and-paste, and drag-and-drop, reporting each action
// taken using the TableViewSig signals
type TableView struct {
	gi.Frame
	Slice            interface{}        `copy:"-" view:"-" json:"-" xml:"-" desc:"the slice that we are a view onto -- must be a pointer to that slice"`
	SliceValView     ValueView          `copy:"-" view:"-" json:"-" xml:"-" desc:"ValueView for the slice itself, if this was created within value view framework -- otherwise nil"`
	StyleFunc        TableViewStyleFunc `copy:"-" view:"-" json:"-" xml:"-" desc:"optional styling function"`
	ShowViewCtxtMenu bool               `desc:"if the object we're viewing has its own CtxtMenu property defined, should we also still show the view's standard context menu?"`
	Changed          bool               `desc:"has the table been edited?"`
	Values           [][]ValueView      `copy:"-" view:"-" json:"-" xml:"-" desc:"ValueView representations of the slice field values -- outer dimension is fields, inner is rows (generally more rows than fields, so this minimizes number of slices allocated)"`
	ShowIndex        bool               `xml:"index" desc:"whether to show index or not (default true) -- updated from 'index' property (bool)"`
	InactKeyNav      bool               `xml:"inact-key-nav" desc:"support key navigation when inactive (default true) -- updated from 'intact-key-nav' property (bool) -- no focus really plausible in inactive case, so it uses a low-pri capture of up / down events"`
	SelField         string             `copy:"-" view:"-" json:"-" xml:"-" desc:"current selection field -- initially select value in this field"`
	SelVal           interface{}        `copy:"-" view:"-" json:"-" xml:"-" desc:"current selection value -- initially select this value in SelField"`
	SelectedIdx      int                `copy:"-" json:"-" xml:"-" desc:"index (row) of currently-selected item (-1 if none) -- see SelectedIdxs for full set of selected rows in active editing mode"`
	SortIdx          int                `desc:"current sort index"`
	SortDesc         bool               `desc:"whether current sort order is descending"`
	SelectMode       bool               `desc:"editing-mode select rows mode"`
	SelectedIdxs     map[int]struct{}   `copy:"-" desc:"list of currently-selected indexes"`
	DraggedIdxs      []int              `copy:"-" desc:"list of currently-dragged indexes"`
	TableViewSig     ki.Signal          `copy:"-" json:"-" xml:"-" desc:"table view interactive editing signals"`
	ViewSig          ki.Signal          `copy:"-" json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	TmpSave          ValueView          `copy:"-" json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ToolbarSlice     interface{}        `copy:"-" view:"-" json:"-" xml:"-" desc:"the slice that we successfully set a toolbar for"`

	StruType     reflect.Type          `copy:"-" view:"-" json:"-" xml:"-" desc:"struct type for each row"`
	NVisFields   int                   `copy:"-" view:"-" json:"-" xml:"-" desc:"number of visible fields"`
	VisFields    []reflect.StructField `copy:"-" view:"-" json:"-" xml:"-" desc:"the visible fields"`
	SliceSize    int                   `view:"inactive" copy:"-" json:"-" xml:"-" desc:"size of slice"`
	DispRows     int                   `view:"inactive" copy:"-" json:"-" xml:"-" desc:"actual number of rows displayed = min(VisRows, SliceSize)"`
	StartIdx     int                   `view:"inactive" copy:"-" json:"-" xml:"-" desc:"starting slice index of visible rows"`
	RowHeight    float32               `view:"inactive" copy:"-" json:"-" xml:"-" desc:"height of a single row"`
	VisRows      int                   `view:"inactive" copy:"-" json:"-" xml:"-" desc:"total number of rows visible in allocated display size"`
	layoutHeight float32               `copy:"-" view:"-" json:"-" xml:"-" desc:"the height of grid from last layout -- determines when update needed"`
	renderedRows int                   `copy:"-" view:"-" json:"-" xml:"-" desc:"the number of rows rendered -- determines update"`
	inFocusGrab  bool                  `copy:"-" view:"-" json:"-" xml:"-" desc:"guard for recursive focus grabbing"`
	curIdx       int                   `copy:"-" view:"-" json:"-" xml:"-" desc:"temp idx state for e.g., dnd"`
}

var KiT_TableView = kit.Types.AddType(&TableView{}, TableViewProps)

// AddNewTableView adds a new tableview to given parent node, with given name.
func AddNewTableView(parent ki.Ki, name string) *TableView {
	return parent.AddNewChild(KiT_TableView, name).(*TableView)
}

func (tv *TableView) Disconnect() {
	tv.Frame.Disconnect()
	tv.TableViewSig.DisconnectAll()
	tv.ViewSig.DisconnectAll()
}

// TableViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view
type TableViewStyleFunc func(tv *TableView, slice interface{}, widg gi.Node2D, row, col int, vv ValueView)

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice
func (tv *TableView) SetSlice(sl interface{}, tmpSave ValueView) {
	updt := false
	if kit.IfaceIsNil(sl) {
		return
	}
	if tv.Slice != sl {
		if !tv.IsInactive() {
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
		struTyp := tv.StructType()
		if struTyp.Kind() != reflect.Struct {
			log.Printf("TableView requires that you pass a slice of struct elements -- type is not a Struct: %v\n", struTyp.String())
			return
		}
		updt = tv.UpdateStart()
		tv.SelectedIdxs = make(map[int]struct{})
		tv.SelectMode = false
		tv.SetFullReRender()
	}
	tv.ShowIndex = true
	if sidxp, err := tv.PropTry("index"); err == nil {
		tv.ShowIndex, _ = kit.ToBool(sidxp)
	}
	tv.InactKeyNav = true
	if siknp, err := tv.PropTry("inact-key-nav"); err == nil {
		tv.InactKeyNav, _ = kit.ToBool(siknp)
	}
	tv.TmpSave = tmpSave
	tv.Config()
	tv.UpdateEnd(updt)
}

var TableViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}

// TableViewSignals are signals that tableview can send, mostly for editing
// mode.  Selection events are sent on WidgetSig WidgetSelected signals in
// both modes.
type TableViewSignals int64

const (
	// TableViewDoubleClicked emitted during inactive mode when item
	// double-clicked -- can be used for accepting dialog.
	TableViewDoubleClicked TableViewSignals = iota

	// todo: add more signals as needed

	TableViewSignalsN
)

//go:generate stringer -type=TableViewSignals

// UpdateValues just updates rendered values
func (tv *TableView) UpdateValues() {
	updt := tv.UpdateStart()
	for _, vv := range tv.Values {
		for _, vvf := range vv {
			vvf.UpdateWidget()
		}
	}
	tv.UpdateEnd(updt)
}

// StructType sets the StruType and returns the type of the struct within the
// slice -- this is a non-ptr type even if slice has pointers to structs
func (tv *TableView) StructType() reflect.Type {
	tv.StruType = kit.NonPtrType(kit.SliceElType(tv.Slice))
	return tv.StruType
}

// CacheVisFields computes the number of visible fields in nVisFields and
// caches those to skip in fieldSkip
func (tv *TableView) CacheVisFields() {
	styp := tv.StructType()
	tv.VisFields = make([]reflect.StructField, 0, 20)
	kit.FlatFieldsTypeFunc(styp, func(typ reflect.Type, fld reflect.StructField) bool {
		tvtag := fld.Tag.Get("tableview")
		add := true
		if tvtag != "" {
			if tvtag == "-" {
				add = false
			} else if tvtag == "-select" && tv.IsInactive() {
				add = false
			} else if tvtag == "-edit" && !tv.IsInactive() {
				add = false
			}
		}
		if add {
			tv.VisFields = append(tv.VisFields, fld)
		}
		return true
	})
	tv.NVisFields = len(tv.VisFields)
}

// Config configures the view
func (tv *TableView) Config() {
	tv.Lay = gi.LayoutVert
	tv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_Frame, "frame")
	mods, updt := tv.ConfigChildren(config, false)
	tv.ConfigSliceGrid()
	tv.ConfigToolbar()
	if mods {
		tv.SetFullReRender()
		tv.UpdateEnd(updt)
	}
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

// SliceFrame returns the outer frame widget, which contains all the header,
// fields and values
func (tv *TableView) SliceFrame() *gi.Frame {
	return tv.ChildByName("frame", 0).(*gi.Frame)
}

// GridLayout returns the SliceGrid grid-layout widget, with grid and scrollbar
func (tv *TableView) GridLayout() *gi.Layout {
	return tv.SliceFrame().ChildByName("grid-lay", 0).(*gi.Layout)
}

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values, within SliceFrame
func (tv *TableView) SliceGrid() *gi.Frame {
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
	if !tv.IsInactive() {
		nWidgPerRow += 2
	}
	idxOff = 1
	if !tv.ShowIndex {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

// SliceValueSize returns the reflect.Value and size of the slice
// sets SliceSize always to current size
func (tv *TableView) SliceValueSize() (reflect.Value, int) {
	svnp := kit.NonPtrValue(reflect.ValueOf(tv.Slice))
	sz := svnp.Len()
	tv.SliceSize = sz
	return svnp, sz
}

// ConfigSliceGrid configures the SliceGrid for the current slice
// this is only called by global Config and updates are guarded by that
func (tv *TableView) ConfigSliceGrid() {
	if kit.IfaceIsNil(tv.Slice) {
		return
	}

	tv.CacheVisFields()

	svnp, sz := tv.SliceValueSize()
	if sz == 0 {
		return
	}

	nWidgPerRow, idxOff := tv.RowWidgetNs()

	sg := tv.SliceFrame()
	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	sg.Lay = gi.LayoutVert
	sg.SetMinPrefWidth(units.NewEm(10))
	sg.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sg.SetStretchMaxWidth()  // for this to work, ALL layers above need it too

	sgcfg := kit.TypeAndNameList{}
	sgcfg.Add(gi.KiT_ToolBar, "header")
	sgcfg.Add(gi.KiT_Layout, "grid-lay")
	sg.ConfigChildren(sgcfg, true)

	sgh := tv.SliceHeader()
	sgh.Lay = gi.LayoutHoriz
	sgh.SetProp("overflow", gi.OverflowHidden) // no scrollbars!
	sgh.SetProp("spacing", 0)
	// sgh.SetStretchMaxWidth()

	gl := tv.GridLayout()
	gl.Lay = gi.LayoutHoriz
	gl.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	gl.SetStretchMaxWidth()  // for this to work, ALL layers above need it too
	gconfig := kit.TypeAndNameList{}
	gconfig.Add(gi.KiT_Frame, "grid")
	gconfig.Add(gi.KiT_ScrollBar, "scrollbar")
	gl.ConfigChildren(gconfig, true) // covered by above

	sgf := tv.SliceGrid()
	sgf.Lay = gi.LayoutGrid
	sgf.Stripes = gi.RowStripes
	sgf.SetMinPrefHeight(units.NewEm(10))
	sgf.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sgf.SetStretchMaxWidth()  // for this to work, ALL layers above need it too
	sgf.SetProp("columns", nWidgPerRow)

	// Configure Header
	hcfg := kit.TypeAndNameList{}
	if tv.ShowIndex {
		hcfg.Add(gi.KiT_Label, "head-idx")
	}
	for fli := 0; fli < tv.NVisFields; fli++ {
		fld := tv.VisFields[fli]
		labnm := fmt.Sprintf("head-%v", fld.Name)
		hcfg.Add(gi.KiT_Action, labnm)
	}
	if !tv.IsInactive() {
		hcfg.Add(gi.KiT_Label, "head-add")
		hcfg.Add(gi.KiT_Label, "head-del")
	}
	sgh.ConfigChildren(hcfg, false)

	// at this point, we make one dummy row to get size of widgets

	sgf.DeleteChildren(true)
	sgf.Kids = make(ki.Slice, nWidgPerRow)

	itxt := fmt.Sprintf("%05d", 0)
	labnm := fmt.Sprintf("index-%v", itxt)

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
				hdr.SetIcon("wedge-down")
			} else {
				hdr.SetIcon("wedge-up")
			}
		}
		hdr.Data = fli
		hdr.Tooltip = "(click to sort / toggle sort direction by this column)"
		dsc := field.Tag.Get("desc")
		if dsc != "" {
			hdr.Tooltip += ": " + dsc
		}
		hdr.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			act := send.(*gi.Action)
			fldIdx := act.Data.(int)
			tvv.SortSliceAction(fldIdx)
		})

		val := kit.OnePtrUnderlyingValue(svnp.Index(0)) // deal with pointer lists
		stru := val.Interface()
		fval := val.Elem().Field(field.Index[0])
		vv := ToValueView(fval.Interface(), "")
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetStructValue(fval.Addr(), stru, &field, tv.TmpSave)
		vtyp := vv.WidgetType()
		valnm := fmt.Sprintf("value-%v.%v", fli, itxt)
		cidx := idxOff + fli
		widg := ki.NewOfType(vtyp).(gi.Node2D)
		sgf.SetChild(widg, cidx, valnm)
		vv.ConfigWidget(widg)
	}

	if !tv.IsInactive() {
		lbl := sgh.Child(tv.NVisFields + idxOff).(*gi.Label)
		lbl.Text = "+"
		lbl.Tooltip = "insert row"
		lbl = sgh.Child(tv.NVisFields + idxOff + 1).(*gi.Label)
		lbl.Text = "-"
		lbl.Tooltip = "delete row"

		addnm := fmt.Sprintf("add-%v", itxt)
		delnm := fmt.Sprintf("del-%v", itxt)
		addact := gi.Action{}
		delact := gi.Action{}
		sgf.SetChild(&addact, idxOff+tv.NVisFields, addnm)
		sgf.SetChild(&delact, idxOff+1+tv.NVisFields, delnm)

		addact.SetIcon("plus")
		delact.SetIcon("minus")
	}

	if tv.SortIdx >= 0 {
		rawIdx := tv.VisFields[tv.SortIdx].Index
		kit.StructSliceSort(tv.Slice, rawIdx, !tv.SortDesc)
	}

	tv.ConfigScroll()
}

func (tv *TableView) ConfigScroll() {
	sb := tv.ScrollBar()
	sb.Dim = gi.Y
	sb.Defaults()
	sb.Tracking = true
	if tv.Sty.Layout.ScrollBarWidth.Dots == 0 {
		sb.SetFixedWidth(units.NewPx(16))
	} else {
		sb.SetFixedWidth(tv.Sty.Layout.ScrollBarWidth)
	}
	sb.SetStretchMaxHeight()
	sb.Min = 0
	sb.Step = 1
	tv.UpdateScroll()

	sb.SliderSig.Connect(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig != int64(gi.SliderValueChanged) {
			return
		}
		wupdt := tv.Viewport.Win.UpdateStart()
		tv.UpdateSliceGrid()
		tv.Viewport.ReRender2DNode(tv)
		tv.Viewport.Win.UpdateEnd(wupdt)
	})
}

// UpdateScroll updates grid scrollbar based on display
func (tv *TableView) UpdateScroll() {
	sb := tv.ScrollBar()
	sb.SetFullReRender()
	updt := sb.UpdateStart()
	sb.Max = float32(tv.SliceSize)
	if tv.DispRows > 0 {
		sb.PageStep = float32(tv.DispRows) * sb.Step
		sb.ThumbVal = float32(tv.DispRows)
	} else {
		sb.PageStep = 10 * sb.Step
		sb.ThumbVal = 10
	}
	sb.TrackThr = sb.Step
	// 	sb.SetValue(float32(tv.StartIdx))
	sb.Value = float32(tv.StartIdx)
	if tv.DispRows == tv.SliceSize {
		sb.Off = true
	} else {
		sb.Off = false
	}
	sb.UpdateEnd(updt)
}

func (tv *TableView) AvailHeight() float32 {
	sg := tv.SliceGrid()
	sgHt := sg.LayData.AllocSize.Y
	if sgHt == 0 {
		return 0
	}
	sgHt -= sg.ExtraSize.Y + sg.Sty.BoxSpace()*2
	return sgHt
}

// LayoutSliceGrid does the proper layout of slice grid depending on allocated size
// returns true if UpdateSliceGrid should be called after this
func (tv *TableView) LayoutSliceGrid() bool {
	sg := tv.SliceGrid()
	if kit.IfaceIsNil(tv.Slice) {
		sg.DeleteChildren(true)
		return false
	}
	_, sz := tv.SliceValueSize()
	if sz == 0 {
		sg.DeleteChildren(true)
		return false
	}

	sgHt := tv.AvailHeight()
	tv.layoutHeight = sgHt
	if sgHt == 0 {
		return false
	}

	nWidgPerRow, _ := tv.RowWidgetNs()
	tv.RowHeight = sg.GridData[gi.Row][0].AllocSize + sg.Spacing.Dots
	tv.VisRows = int(math32.Floor(sgHt / tv.RowHeight))
	tv.DispRows = ints.MinInt(tv.SliceSize, tv.VisRows)

	nWidg := nWidgPerRow * tv.DispRows

	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)
	if tv.Values == nil || sg.NumChildren() != nWidg {
		sg.DeleteChildren(true)

		tv.Values = make([][]ValueView, tv.NVisFields)
		for fli := 0; fli < tv.NVisFields; fli++ {
			tv.Values[fli] = make([]ValueView, tv.DispRows)
		}
		sg.Kids = make(ki.Slice, nWidg)
	}
	tv.ConfigScroll()
	return true
}

func (tv *TableView) SliceGridNeedsLayout() bool {
	sgHt := tv.AvailHeight()
	if sgHt != tv.layoutHeight {
		return true
	}
	return tv.renderedRows != tv.DispRows
}

// UpdateSliceGrid updates grid display -- robust to any time calling
func (tv *TableView) UpdateSliceGrid() {
	if kit.IfaceIsNil(tv.Slice) {
		return
	}
	svnp, sz := tv.SliceValueSize()
	if sz == 0 {
		return
	}
	sg := tv.SliceGrid()
	tv.DispRows = ints.MinInt(tv.SliceSize, tv.VisRows)

	nWidgPerRow, idxOff := tv.RowWidgetNs()
	nWidg := nWidgPerRow * tv.DispRows

	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	if tv.Values == nil || sg.NumChildren() != nWidg { // shouldn't happen..
		tv.LayoutSliceGrid()
		nWidg = nWidgPerRow * tv.DispRows
	}

	if sz > tv.DispRows {
		sb := tv.ScrollBar()
		tv.StartIdx = int(sb.Value)
		lastSt := sz - tv.DispRows
		tv.StartIdx = ints.MinInt(lastSt, tv.StartIdx)
		tv.StartIdx = ints.MaxInt(0, tv.StartIdx)
	} else {
		tv.StartIdx = 0
	}

	for i := 0; i < tv.DispRows; i++ {
		ridx := i * nWidgPerRow
		si := tv.StartIdx + i // slice idx
		issel := tv.IdxIsSelected(si)
		val := kit.OnePtrUnderlyingValue(svnp.Index(si)) // deal with pointer lists
		stru := val.Interface()

		itxt := fmt.Sprintf("%05d", i)
		sitxt := fmt.Sprintf("%05d", si)
		labnm := fmt.Sprintf("index-%v", itxt)
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
				idxlab.Sty.Template = "TableView.IndexLabel"
				idxlab.WidgetSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.WidgetSelected) {
						wbb := send.(gi.Node2D).AsWidget()
						row := wbb.Prop("tv-row").(int)
						tvv := recv.Embed(KiT_TableView).(*TableView)
						tvv.UpdateSelectRow(row, wbb.IsSelected())
					}
				})
			}
			idxlab.CurBgColor = gi.Prefs.Colors.Background
			idxlab.SetText(sitxt)
			idxlab.SetSelectedState(issel)
		}

		for fli := 0; fli < tv.NVisFields; fli++ {
			field := tv.VisFields[fli]
			fval := val.Elem().Field(field.Index[0])
			vv := ToValueView(fval.Interface(), "")
			if vv == nil { // shouldn't happen
				continue
			}
			vv.SetStructValue(fval.Addr(), stru, &field, tv.TmpSave)
			tv.Values[fli][i] = vv

			vtyp := vv.WidgetType()
			valnm := fmt.Sprintf("value-%v.%v", fli, itxt)
			cidx := ridx + idxOff + fli
			var widg gi.Node2D
			if sg.Kids[cidx] != nil {
				widg = sg.Kids[cidx].(gi.Node2D)
				vv.ConfigWidget(widg) // note: need config b/c vv is new
				if tv.IsInactive() {
					widg.AsNode2D().SetInactive()
				}
				widg.AsNode2D().SetSelectedState(issel)
			} else {
				widg = ki.NewOfType(vtyp).(gi.Node2D)
				sg.SetChild(widg, cidx, valnm)
				vv.ConfigWidget(widg)
				wb := widg.AsWidget()
				if wb != nil {
					// totally not worth it now:
					// wb.Sty.Template = "TableViewView.ItemWidget." + vtyp.Name()
					wb.SetProp("tv-row", i)
					wb.ClearSelected()
					wb.WidgetSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
						if sig == int64(gi.WidgetSelected) || sig == int64(gi.WidgetFocused) {
							wbb := send.(gi.Node2D).AsWidget()
							row := wbb.Prop("tv-row").(int)
							tvv := recv.Embed(KiT_TableView).(*TableView)
							if sig != int64(gi.WidgetFocused) || !tvv.inFocusGrab {
								tvv.UpdateSelectRow(row, wbb.IsSelected())
							}
						}
					})
				}
				if tv.IsInactive() {
					widg.AsNode2D().SetInactive()
				} else {
					vvb := vv.AsValueViewBase()
					vvb.ViewSig.ConnectOnly(tv.This(), // todo: do we need this?
						func(recv, send ki.Ki, sig int64, data interface{}) {
							tvv, _ := recv.Embed(KiT_TableView).(*TableView)
							tvv.SetChanged()
						})
				}
			}
			if tv.StyleFunc != nil {
				tv.StyleFunc(tv, svnp.Interface(), widg, i, fli, vv)
			}
		}

		if !tv.IsInactive() {
			addnm := fmt.Sprintf("add-%v", itxt)
			delnm := fmt.Sprintf("del-%v", itxt)
			addact := gi.Action{}
			delact := gi.Action{}
			sg.SetChild(&addact, ridx+1+tv.NVisFields, addnm)
			sg.SetChild(&delact, ridx+1+tv.NVisFields+1, delnm)

			addact.SetIcon("plus")
			addact.Tooltip = "insert a new element at this index"
			addact.Data = i
			addact.Sty.Template = "TableViewView.AddAction"
			addact.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				act := send.(*gi.Action)
				tvv := recv.Embed(KiT_TableView).(*TableView)
				tvv.SliceNewAtRow(act.Data.(int)+1, true)
			})
			delact.SetIcon("minus")
			delact.Tooltip = "delete this element"
			delact.Data = i
			delact.Sty.Template = "TableView.DelAction"
			delact.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				act := send.(*gi.Action)
				tvv := recv.Embed(KiT_TableView).(*TableView)
				tvv.SliceDeleteAtRow(act.Data.(int), true)
			})
		}
	}

	if tv.SelField != "" && tv.SelVal != nil {
		tv.SelectedIdx, _ = StructSliceIdxByValue(tv.Slice, tv.SelField, tv.SelVal)
	}
	if tv.IsInactive() && tv.SelectedIdx >= 0 {
		tv.SelectIdx(tv.SelectedIdx)
	}
	tv.UpdateScroll()
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// TableView, indicating that some kind of edit / change has taken place to
// the table data.  It isn't really practical to record all the different
// types of changes, so this is just generic.
func (tv *TableView) SetChanged() {
	tv.Changed = true
	tv.ViewSig.Emit(tv.This(), 0, nil)
	tv.ToolBar().UpdateActions() // nil safe
}

// SliceNewAtRow inserts a new blank element at given display row
func (tv *TableView) SliceNewAtRow(row int, reconfig bool) {
	tv.SliceNewAt(tv.StartIdx+row, reconfig)
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end -- reconfig means call UpdateSliceGrid to update display
func (tv *TableView) SliceNewAt(idx int, reconfig bool) {
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)

	kit.SliceNewAt(tv.Slice, idx)

	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.SetChanged()
	if reconfig {
		tv.UpdateSliceGrid()
	}
	tv.ViewSig.Emit(tv.This(), 0, nil)
}

// SliceDeleteAtRow deletes element at given display row
func (tv *TableView) SliceDeleteAtRow(row int, reconfig bool) {
	tv.SliceDeleteAt(tv.StartIdx+row, reconfig)
}

// SliceDeleteAt deletes element at given index from slice -- reconfig means
// call UpdateSliceGrid to update display
func (tv *TableView) SliceDeleteAt(idx int, reconfig bool) {
	if idx < 0 {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)

	kit.SliceDeleteAt(tv.Slice, idx)

	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.SetChanged()
	if reconfig {
		tv.UpdateSliceGrid()
	}
	tv.ViewSig.Emit(tv.This(), 0, nil)
}

// SortSliceAction sorts the slice for given field index -- toggles ascending
// vs. descending if already sorting on this dimension
func (tv *TableView) SortSliceAction(fldIdx int) {
	oswin.TheApp.Cursor(tv.Viewport.Win.OSWin).Push(cursor.Wait)
	defer oswin.TheApp.Cursor(tv.Viewport.Win.OSWin).Pop()

	sgh := tv.SliceHeader()
	sgh.SetFullReRender()
	idxOff := 1
	if !tv.ShowIndex {
		idxOff = 0
	}

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
				hdr.SetIcon("wedge-up")
			} else {
				hdr.SetIcon("wedge-down")
			}
		} else {
			hdr.SetIcon("none")
		}
	}

	tv.SortIdx = fldIdx
	rawIdx := tv.VisFields[fldIdx].Index

	sgf := tv.SliceGrid()
	sgf.SetFullReRender()

	kit.StructSliceSort(tv.Slice, rawIdx, !tv.SortDesc)
	tv.UpdateSliceGrid()
}

// ConfigToolbar configures the toolbar actions
func (tv *TableView) ConfigToolbar() {
	if kit.IfaceIsNil(tv.Slice) || tv.IsInactive() {
		return
	}
	if tv.ToolbarSlice == tv.Slice {
		return
	}
	tb := tv.ToolBar()
	if len(*tb.Children()) == 0 {
		tb.SetStretchMaxWidth()
		tb.AddAction(gi.ActOpts{Label: "Add", Icon: "plus"},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				tvv := recv.Embed(KiT_TableView).(*TableView)
				tvv.SliceNewAt(-1, true)
			})
	}
	sz := len(*tb.Children())
	if sz > 1 {
		for i := sz - 1; i >= 1; i-- {
			tb.DeleteChildAtIndex(i, true)
		}
	}
	if HasToolBarView(tv.Slice) {
		ToolBarView(tv.Slice, tv.Viewport, tb)
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

func (tv *TableView) Style2D() {
	if !tv.IsConfiged() {
		return
	}
	if tv.IsInactive() {
		tv.SetCanFocus()
	}
	sg := tv.SliceGrid()
	sg.StartFocus() // need to call this when window is actually active
	tv.Frame.Style2D()
}

func (tv *TableView) Layout2D(parBBox image.Rectangle, iter int) bool {
	redo := tv.Frame.Layout2D(parBBox, iter)
	idxOff := 1
	if !tv.ShowIndex {
		idxOff = 0
	}
	if !tv.IsConfiged() {
		return redo
	}
	nfld := tv.NVisFields + idxOff
	sgh := tv.SliceHeader()
	sgf := tv.SliceGrid()
	if len(sgf.Kids) >= nfld {
		sumwd := float32(0)
		for fli := 0; fli < nfld; fli++ {
			lbl := sgh.Child(fli).(gi.Node2D).AsWidget()
			wd := sgf.GridData[gi.Col][fli].AllocSize
			lbl.SetMinPrefWidth(units.NewValue(wd-sgf.Spacing.Dots, units.Dot))
			sumwd += wd
		}
		if !tv.IsInactive() {
			for fli := nfld; fli < nfld+2; fli++ {
				lbl := sgh.Child(fli).(gi.Node2D).AsWidget()
				wd := sgf.GridData[gi.Col][fli].AllocSize
				lbl.SetMinPrefWidth(units.NewValue(wd-sgf.Spacing.Dots, units.Dot))
				sumwd += wd
			}
		}
		sgh.SetMinPrefWidth(units.NewValue(sumwd, units.Dot))
		sgh.Layout2D(parBBox, iter)
	}
	return redo
}

func (tv *TableView) Render2D() {
	tv.ToolBar().UpdateActions()
	if win := tv.ParentWindow(); win != nil {
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	if tv.SliceGridNeedsLayout() {
		// note: we are outside of slice grid and thus cannot do proper layout during Layout2D
		// as we don't yet know the size of grid -- so we catch it here at next step and just
		// rebuild as needed.
		tv.renderedRows = tv.DispRows
		if tv.LayoutSliceGrid() {
			tv.UpdateSliceGrid()
		}
		tv.ReRender2DTree()
		if tv.SelectedIdx > -1 {
			tv.ScrollToIdx(tv.SelectedIdx)
		}
		return
	}
	if tv.FullReRenderIfNeeded() {
		return
	}
	if !tv.IsConfiged() {
		return
	}
	if tv.PushBounds() {
		if tv.Sty.Font.Face.Metrics.Height > 0 {
			tv.VisRows = (tv.VpBBox.Max.Y - tv.VpBBox.Min.Y) / int(1.8*tv.Sty.Font.Face.Metrics.Height)
		} else {
			tv.VisRows = 10
		}
		tv.FrameStdRender()
		tv.This().(gi.Node2D).ConnectEvents2D()
		tv.RenderScrolls()
		tv.Render2DChildren()
		tv.PopBounds()
	} else {
		tv.DisconnectAllEvents(gi.AllPris)
	}
}

func (tv *TableView) ConnectEvents2D() {
	tv.TableViewEvents()
}

func (tv *TableView) HasFocus2D() bool {
	if tv.IsInactive() {
		return tv.InactKeyNav
	}
	return tv.ContainsFocus() // anyone within us gives us focus..
}

//////////////////////////////////////////////////////////////////////////////
//  Row access methods
//  NOTE: row = physical GUI display row, idx = slice index -- not the same!

// SliceStruct returns struct interface at given row
func (tv *TableView) SliceStruct(idx int) interface{} {
	svnp, sz := tv.SliceValueSize()
	if idx < 0 || idx >= sz {
		fmt.Printf("giv.TableView: slice index out of range: %v\n", idx)
		return nil
	}
	val := kit.OnePtrUnderlyingValue(svnp.Index(idx)) // deal with pointer lists
	stru := val.Interface()
	return stru
}

// IsRowInBounds returns true if disp row is in bounds
func (tv *TableView) IsRowInBounds(row int) bool {
	return row >= 0 && row < tv.DispRows
}

// IsIdxVisible returns true if slice index is currently visible
func (tv *TableView) IsIdxVisible(idx int) bool {
	return tv.IsRowInBounds(idx - tv.StartIdx)
}

// RowFirstWidget returns the first widget for given row (could be index or
// not) -- false if out of range
func (tv *TableView) RowFirstWidget(row int) (*gi.WidgetBase, bool) {
	if !tv.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, _ := tv.RowWidgetNs()
	sg := tv.SliceGrid()
	widg := sg.Kids[row*nWidgPerRow].(gi.Node2D).AsWidget()
	return widg, true
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
	if widg.VpBBox != image.ZR {
		return widg, true
	}
	ridx := nWidgPerRow * row
	for fli := 0; fli < tv.NVisFields; fli++ {
		widg := sg.Child(ridx + idxOff + fli).(gi.Node2D).AsWidget()
		if widg.VpBBox != image.ZR {
			return widg, true
		}
	}
	return nil, false
}

// RowGrabFocus grabs the focus for the first focusable widget in given row --
// returns that element or nil if not successful -- note: grid must have
// already rendered for focus to be grabbed!
func (tv *TableView) RowGrabFocus(row int) *gi.WidgetBase {
	if !tv.IsRowInBounds(row) || tv.inFocusGrab { // range check
		return nil
	}
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := tv.SliceGrid()
	// first check if we already have focus
	for fli := 0; fli < tv.NVisFields; fli++ {
		widg := sg.Child(ridx + idxOff + fli).(gi.Node2D).AsWidget()
		if widg.HasFocus() {
			return widg
		}
	}
	tv.inFocusGrab = true
	defer func() { tv.inFocusGrab = false }()
	for fli := 0; fli < tv.NVisFields; fli++ {
		widg := sg.Child(ridx + idxOff + fli).(gi.Node2D).AsWidget()
		if widg.CanFocus() {
			widg.GrabFocus()
			return widg
		}
	}
	return nil
}

// IdxGrabFocus grabs the focus for the first focusable widget in given idx --
// returns that element or nil if not successful
func (tv *TableView) IdxGrabFocus(idx int) *gi.WidgetBase {
	tv.ScrollToIdx(idx)
	return tv.RowGrabFocus(idx - tv.StartIdx)
}

// IdxPos returns center of window position of index label for row (ContextMenuPos)
func (tv *TableView) IdxPos(idx int) image.Point {
	row := idx - tv.StartIdx
	if row < 0 {
		row = 0
	}
	if row > tv.DispRows-1 {
		row = tv.DispRows - 1
	}
	var pos image.Point
	widg, ok := tv.RowFirstVisWidget(row)
	if ok {
		pos = widg.ContextMenuPos()
	}
	return pos
}

// RowFromPos returns the row that contains given vertical position, false if not found
func (tv *TableView) RowFromPos(posY int) (int, bool) {
	// todo: could optimize search to approx loc, and search up / down from there
	for rw := 0; rw < tv.DispRows; rw++ {
		widg, ok := tv.RowFirstWidget(rw)
		if ok {
			if widg.ObjBBox.Min.Y < posY && posY < widg.ObjBBox.Max.Y {
				return rw, true
			}
		}
	}
	return -1, false
}

// IdxFromPos returns the idx that contains given vertical position, false if not found
func (tv *TableView) IdxFromPos(posY int) (int, bool) {
	row, ok := tv.RowFromPos(posY)
	if !ok {
		return -1, false
	}
	return row + tv.StartIdx, true
}

// ScrollToIdx ensures that given slice idx is visible by scrolling display as needed
func (tv *TableView) ScrollToIdx(idx int) bool {
	if idx < tv.StartIdx {
		tv.StartIdx = idx
		tv.StartIdx = ints.MaxInt(0, tv.StartIdx)
		tv.UpdateScroll()
		tv.UpdateSliceGrid()
		return true
	} else if idx >= tv.StartIdx+tv.DispRows {
		tv.StartIdx = idx - (tv.DispRows - 1)
		tv.StartIdx = ints.MaxInt(0, tv.StartIdx)
		tv.UpdateScroll()
		tv.UpdateSliceGrid()
		return true
	}
	return false
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
func StructSliceIdxByValue(struSlice interface{}, fldName string, fldVal interface{}) (int, error) {
	svnp := kit.NonPtrValue(reflect.ValueOf(struSlice))
	sz := svnp.Len()
	struTyp := kit.NonPtrType(reflect.TypeOf(struSlice).Elem().Elem())
	fld, ok := struTyp.FieldByName(fldName)
	if !ok {
		err := fmt.Errorf("gi.StructSliceRowByValue: field name: %v not found\n", fldName)
		log.Println(err)
		return -1, err
	}
	fldIdx := fld.Index[0]
	for idx := 0; idx < sz; idx++ {
		rval := kit.OnePtrUnderlyingValue(svnp.Index(idx))
		fval := rval.Elem().Field(fldIdx)
		if fval.Interface() == fldVal {
			return idx, nil
		}
	}
	return -1, nil
}

/////////////////////////////////////////////////////////////////////////////
//    Moving

// MoveDown moves the selection down to next idx, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (tv *TableView) MoveDown(selMode mouse.SelectModes) int {
	if tv.SelectedIdx >= tv.SliceSize-1 {
		tv.SelectedIdx = tv.SliceSize - 1
		return -1
	}
	tv.SelectedIdx++
	tv.SelectIdxAction(tv.SelectedIdx, selMode)
	return tv.SelectedIdx
}

// MoveDownAction moves the selection down to next idx, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (tv *TableView) MoveDownAction(selMode mouse.SelectModes) int {
	nidx := tv.MoveDown(selMode)
	if nidx >= 0 {
		tv.ScrollToIdx(nidx)
		tv.WidgetSig.Emit(tv.This(), int64(gi.WidgetSelected), nidx)
	}
	return nidx
}

// MoveUp moves the selection up to previous idx, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (tv *TableView) MoveUp(selMode mouse.SelectModes) int {
	if tv.SelectedIdx <= 0 {
		tv.SelectedIdx = 0
		return -1
	}
	tv.SelectedIdx--
	tv.SelectIdxAction(tv.SelectedIdx, selMode)
	return tv.SelectedIdx
}

// MoveUpAction moves the selection up to previous idx, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (tv *TableView) MoveUpAction(selMode mouse.SelectModes) int {
	nidx := tv.MoveUp(selMode)
	if nidx >= 0 {
		tv.ScrollToIdx(nidx)
		tv.WidgetSig.Emit(tv.This(), int64(gi.WidgetSelected), nidx)
	}
	return nidx
}

// MovePageDown moves the selection down to next page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (tv *TableView) MovePageDown(selMode mouse.SelectModes) int {
	if tv.SelectedIdx >= tv.SliceSize-1 {
		tv.SelectedIdx = tv.SliceSize - 1
		return -1
	}
	tv.SelectedIdx += tv.VisRows
	tv.SelectedIdx = ints.MinInt(tv.SelectedIdx, tv.SliceSize-1)
	tv.SelectIdxAction(tv.SelectedIdx, selMode)
	return tv.SelectedIdx
}

// MovePageDownAction moves the selection down to next page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (tv *TableView) MovePageDownAction(selMode mouse.SelectModes) int {
	nidx := tv.MovePageDown(selMode)
	if nidx >= 0 {
		tv.ScrollToIdx(nidx)
		tv.WidgetSig.Emit(tv.This(), int64(gi.WidgetSelected), nidx)
	}
	return nidx
}

// MovePageUp moves the selection up to previous page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (tv *TableView) MovePageUp(selMode mouse.SelectModes) int {
	if tv.SelectedIdx <= 0 {
		tv.SelectedIdx = 0
		return -1
	}
	tv.SelectedIdx -= tv.VisRows
	tv.SelectedIdx = ints.MaxInt(0, tv.SelectedIdx)
	tv.SelectIdxAction(tv.SelectedIdx, selMode)
	return tv.SelectedIdx
}

// MovePageUpAction moves the selection up to previous page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (tv *TableView) MovePageUpAction(selMode mouse.SelectModes) int {
	nidx := tv.MovePageUp(selMode)
	if nidx >= 0 {
		tv.ScrollToIdx(nidx)
		tv.WidgetSig.Emit(tv.This(), int64(gi.WidgetSelected), nidx)
	}
	return nidx
}

//////////////////////////////////////////////////////////////////////////////
//    Selection: user operates on the index labels

// SelectRowWidgets sets the selection state of given row of widgets
func (tv *TableView) SelectRowWidgets(row int, sel bool) {
	if row < 0 {
		return
	}
	var win *gi.Window
	if tv.Viewport != nil {
		win = tv.Viewport.Win
	}
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	sg := tv.SliceGrid()
	nWidgPerRow, idxOff := tv.RowWidgetNs()
	ridx := row * nWidgPerRow
	for fli := 0; fli < tv.NVisFields; fli++ {
		seldx := ridx + idxOff + fli
		if sg.Kids.IsValidIndex(seldx) == nil {
			widg := sg.Child(seldx).(gi.Node2D).AsNode2D()
			widg.SetSelectedState(sel)
			widg.UpdateSig()
		}
	}
	if tv.ShowIndex {
		if sg.Kids.IsValidIndex(ridx) == nil {
			widg := sg.Child(ridx).(gi.Node2D).AsNode2D()
			widg.SetSelectedState(sel)
			widg.UpdateSig()
		}
	}

	if win != nil {
		win.UpdateEnd(updt)
	}
}

// SelectIdxWidgets sets the selection state of given slice index
// returns false if index is not visible
func (tv *TableView) SelectIdxWidgets(idx int, sel bool) bool {
	if !tv.IsIdxVisible(idx) {
		return false
	}
	tv.SelectRowWidgets(idx-tv.StartIdx, sel)
	return true
}

// UpdateSelectRow updates the selection for the given row
// callback from widgetsig select
func (tv *TableView) UpdateSelectRow(row int, sel bool) {
	idx := row + tv.StartIdx
	tv.UpdateSelectIdx(idx, sel)
}

// UpdateSelectIdx updates the selection for the given index
func (tv *TableView) UpdateSelectIdx(idx int, sel bool) {
	if tv.IsInactive() {
		if tv.SelectedIdx == idx { // never unselect
			tv.SelectIdxWidgets(tv.SelectedIdx, true)
			return
		}
		if tv.SelectedIdx >= 0 { // unselect current
			tv.SelectIdxWidgets(tv.SelectedIdx, false)
		}
		if sel {
			tv.SelectedIdx = idx
			tv.SelectIdxWidgets(tv.SelectedIdx, true)
		}
		tv.WidgetSig.Emit(tv.This(), int64(gi.WidgetSelected), tv.SelectedIdx)
	} else {
		selMode := mouse.SelectOne
		win := tv.Viewport.Win
		if win != nil {
			selMode = win.LastSelMode
		}
		tv.SelectIdxAction(idx, selMode)
	}
}

// IdxIsSelected returns the selected status of given slice index
func (tv *TableView) IdxIsSelected(idx int) bool {
	if _, ok := tv.SelectedIdxs[idx]; ok {
		return true
	}
	return false
}

// SelectedIdxsList returns list of selected idxs, sorted either ascending or descending
func (tv *TableView) SelectedIdxsList(descendingSort bool) []int {
	rws := make([]int, len(tv.SelectedIdxs))
	i := 0
	for r, _ := range tv.SelectedIdxs {
		rws[i] = r
		i++
	}
	if descendingSort {
		sort.Slice(rws, func(i, j int) bool {
			return rws[i] > rws[j]
		})
	} else {
		sort.Slice(rws, func(i, j int) bool {
			return rws[i] < rws[j]
		})
	}
	return rws
}

// SelectIdx selects given idx (if not already selected) -- updates select
// status of index label
func (tv *TableView) SelectIdx(idx int) {
	tv.SelectedIdxs[idx] = struct{}{}
	tv.SelectIdxWidgets(idx, true)
}

// UnselectIdx unselects given idx (if selected)
func (tv *TableView) UnselectIdx(idx int) {
	if tv.IdxIsSelected(idx) {
		delete(tv.SelectedIdxs, idx)
	}
	tv.SelectIdxWidgets(idx, false)
}

// UnselectAllIdxs unselects all selected idxs
func (tv *TableView) UnselectAllIdxs() {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	for r, _ := range tv.SelectedIdxs {
		tv.SelectIdxWidgets(r, false)
	}
	tv.SelectedIdxs = make(map[int]struct{})
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// SelectAllIdxs selects all idxs
func (tv *TableView) SelectAllIdxs() {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	tv.UnselectAllIdxs()
	tv.SelectedIdxs = make(map[int]struct{}, tv.SliceSize)
	for row := 0; row < tv.SliceSize; row++ {
		tv.SelectedIdxs[row] = struct{}{}
		tv.SelectIdxWidgets(row, true)
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// SelectIdxAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (tv *TableView) SelectIdxAction(idx int, mode mouse.SelectModes) {
	if mode == mouse.NoSelect {
		return
	}
	idx = ints.MinInt(idx, tv.SliceSize-1)
	if idx < 0 {
		idx = 0
	}
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	switch mode {
	case mouse.SelectOne:
		if tv.IdxIsSelected(idx) {
			if len(tv.SelectedIdxs) > 1 {
				tv.UnselectAllIdxs()
			}
			tv.SelectedIdx = idx
			tv.SelectIdx(idx)
			tv.IdxGrabFocus(idx)
		} else {
			tv.UnselectAllIdxs()
			tv.SelectedIdx = idx
			tv.SelectIdx(idx)
			tv.IdxGrabFocus(idx)
		}
		tv.WidgetSig.Emit(tv.This(), int64(gi.WidgetSelected), tv.SelectedIdx)
	case mouse.ExtendContinuous:
		if len(tv.SelectedIdxs) == 0 {
			tv.SelectedIdx = idx
			tv.SelectIdx(idx)
			tv.IdxGrabFocus(idx)
			tv.WidgetSig.Emit(tv.This(), int64(gi.WidgetSelected), tv.SelectedIdx)
		} else {
			minIdx := -1
			maxIdx := 0
			for r, _ := range tv.SelectedIdxs {
				if minIdx < 0 {
					minIdx = r
				} else {
					minIdx = ints.MinInt(minIdx, r)
				}
				maxIdx = ints.MaxInt(maxIdx, r)
			}
			cidx := idx
			tv.SelectedIdx = idx
			tv.SelectIdx(idx)
			if idx < minIdx {
				for cidx < minIdx {
					r := tv.MoveDown(mouse.SelectQuiet) // just select
					cidx = r
				}
			} else if idx > maxIdx {
				for cidx > maxIdx {
					r := tv.MoveUp(mouse.SelectQuiet) // just select
					cidx = r
				}
			}
			tv.IdxGrabFocus(idx)
			tv.WidgetSig.Emit(tv.This(), int64(gi.WidgetSelected), tv.SelectedIdx)
		}
	case mouse.ExtendOne:
		if tv.IdxIsSelected(idx) {
			tv.UnselectIdxAction(idx)
		} else {
			tv.SelectedIdx = idx
			tv.SelectIdx(idx)
			tv.IdxGrabFocus(idx)
			tv.WidgetSig.Emit(tv.This(), int64(gi.WidgetSelected), tv.SelectedIdx)
		}
	case mouse.Unselect:
		tv.SelectedIdx = idx
		tv.UnselectIdxAction(idx)
	case mouse.SelectQuiet:
		tv.SelectedIdx = idx
		tv.SelectIdx(idx)
	case mouse.UnselectQuiet:
		tv.SelectedIdx = idx
		tv.UnselectIdx(idx)
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// UnselectIdxAction unselects this idx (if selected) -- and emits a signal
func (tv *TableView) UnselectIdxAction(idx int) {
	if tv.IdxIsSelected(idx) {
		tv.UnselectIdx(idx)
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeDataIdx adds mimedata for given idx: an application/json of the struct
func (tv *TableView) MimeDataIdx(md *mimedata.Mimes, idx int) {
	stru := tv.SliceStruct(idx)
	b, err := json.MarshalIndent(stru, "", "  ")
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: filecat.DataJson, Data: b})
	} else {
		log.Printf("gi.TableView MimeData JSON Marshall error: %v\n", err)
	}
}

// FromMimeData creates a slice of structs from mime data
func (tv *TableView) FromMimeData(md mimedata.Mimes) []interface{} {
	svnp, _ := tv.SliceValueSize()
	svtyp := svnp.Type()
	sl := make([]interface{}, 0, len(md))
	for _, d := range md {
		if d.Type == filecat.DataJson {
			nval := reflect.New(svtyp.Elem()).Interface()
			err := json.Unmarshal(d.Data, nval)
			if err == nil {
				sl = append(sl, nval)
			} else {
				log.Printf("gi.TableView FromMimeData: JSON load error: %v\n", err)
			}
		}
	}
	return sl
}

// Copy copies selected idxs to clip.Board, optionally resetting the selection
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TableView) Copy(reset bool) {
	nitms := len(tv.SelectedIdxs)
	if nitms == 0 {
		return
	}
	ixs := tv.SelectedIdxsList(false) // ascending
	md := make(mimedata.Mimes, 0, nitms)
	for _, i := range ixs {
		tv.MimeDataIdx(&md, i)
	}
	oswin.TheApp.ClipBoard(tv.Viewport.Win.OSWin).Write(md)
	if reset {
		tv.UnselectAllIdxs()
	}
}

// CopyIdxs copies selected indexes to clip.Board, optionally resetting the selection
func (tv *TableView) CopyIdxs(reset bool) {
	if cpr, ok := tv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Copy(reset)
	} else {
		tv.Copy(reset)
	}
}

// DeleteIdxs deletes all selected indexes
func (tv *TableView) DeleteIdxs() {
	if len(tv.SelectedIdxs) == 0 {
		return
	}
	updt := tv.UpdateStart()
	ixs := tv.SelectedIdxsList(true) // descending sort
	for _, i := range ixs {
		tv.SliceDeleteAt(i, false)
	}
	tv.SetChanged()
	tv.UpdateSliceGrid()
	tv.UpdateEnd(updt)
}

// Cut copies selected indexes to clip.Board and deletes selected indexes
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TableView) Cut() {
	if len(tv.SelectedIdxs) == 0 {
		return
	}
	updt := tv.UpdateStart()
	tv.CopyIdxs(false)
	ixs := tv.SelectedIdxsList(true) // descending sort
	idx := ixs[0]
	tv.UnselectAllIdxs()
	for _, i := range ixs {
		tv.SliceDeleteAt(i, false)
	}
	tv.SetChanged()
	tv.UpdateSliceGrid()
	tv.UpdateEnd(updt)
	tv.SelectIdxAction(idx, mouse.SelectOne)
}

// CutIdxs copies selected indexes to clip.Board and deletes selected indexes
func (tv *TableView) CutIdxs() {
	if cpr, ok := tv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Cut()
	} else {
		tv.Cut()
	}
}

// Paste pastes clipboard at curIdx
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TableView) Paste() {
	md := oswin.TheApp.ClipBoard(tv.Viewport.Win.OSWin).Read([]string{filecat.DataJson})
	if md != nil {
		tv.PasteMenu(md, tv.curIdx)
	}
}

// PasteIdx pastes clipboard at given idx
func (tv *TableView) PasteIdx(idx int) {
	tv.curIdx = idx
	if cpr, ok := tv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Paste()
	} else {
		tv.Paste()
	}
}

// MakePasteMenu makes the menu of options for paste events
func (tv *TableView) MakePasteMenu(m *gi.Menu, data interface{}, idx int) {
	if len(*m) > 0 {
		return
	}
	m.AddAction(gi.ActOpts{Label: "Assign To", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.PasteAssign(data.(mimedata.Mimes), idx)
	})
	m.AddAction(gi.ActOpts{Label: "Insert Before", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.PasteAtIdx(data.(mimedata.Mimes), idx)
	})
	m.AddAction(gi.ActOpts{Label: "Insert After", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.PasteAtIdx(data.(mimedata.Mimes), idx+1)
	})
	m.AddAction(gi.ActOpts{Label: "Cancel", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	})
}

// PasteMenu performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (tv *TableView) PasteMenu(md mimedata.Mimes, idx int) {
	tv.UnselectAllIdxs()
	var men gi.Menu
	tv.MakePasteMenu(&men, md, idx)
	pos := tv.IdxPos(idx)
	gi.PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvPasteMenu")
}

// PasteAssign assigns mime data (only the first one!) to this idx
func (tv *TableView) PasteAssign(md mimedata.Mimes, idx int) {
	tvl := reflect.ValueOf(tv.Slice)
	tvnp := kit.NonPtrValue(tvl)

	sl := tv.FromMimeData(md)
	updt := tv.UpdateStart()
	if len(sl) == 0 {
		return
	}
	ns := sl[0]
	tvnp.Index(idx).Set(reflect.ValueOf(ns).Elem())
	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.SetChanged()
	tv.UpdateSliceGrid()
	tv.UpdateEnd(updt)
}

// PasteAtIdx inserts object(s) from mime data at (before) given idx
func (tv *TableView) PasteAtIdx(md mimedata.Mimes, idx int) {
	tvl := reflect.ValueOf(tv.Slice)
	tvnp := kit.NonPtrValue(tvl)

	sl := tv.FromMimeData(md)
	updt := tv.UpdateStart()
	for _, ns := range sl {
		sz := tvnp.Len()
		tvnp = reflect.Append(tvnp, reflect.ValueOf(ns).Elem())
		tvl.Elem().Set(tvnp)
		if idx >= 0 && idx < sz {
			reflect.Copy(tvnp.Slice(idx+1, sz+1), tvnp.Slice(idx, sz))
			tvnp.Index(idx).Set(reflect.ValueOf(ns).Elem())
			tvl.Elem().Set(tvnp)
		}
		idx++
	}
	if tv.TmpSave != nil {
		tv.TmpSave.SaveTmp()
	}
	tv.SetChanged()
	tv.UpdateSliceGrid()
	tv.UpdateEnd(updt)
	tv.SelectIdxAction(idx, mouse.SelectOne)
}

// Duplicate copies selected items and inserts them after current selection --
// return idx of start of duplicates if successful, else -1
func (tv *TableView) Duplicate() int {
	nitms := len(tv.SelectedIdxs)
	if nitms == 0 {
		return -1
	}
	ixs := tv.SelectedIdxsList(true) // descending sort -- last first
	pasteAt := ixs[0]
	tv.CopyIdxs(true)
	md := oswin.TheApp.ClipBoard(tv.Viewport.Win.OSWin).Read([]string{filecat.DataJson})
	tv.PasteAtIdx(md, pasteAt)
	return pasteAt
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragNDropStart starts a drag-n-drop
func (tv *TableView) DragNDropStart() {
	nitms := len(tv.SelectedIdxs)
	if nitms == 0 {
		return
	}
	md := make(mimedata.Mimes, 0, nitms)
	for i, _ := range tv.SelectedIdxs {
		tv.MimeDataIdx(&md, i)
	}
	ixs := tv.SelectedIdxsList(true) // descending sort
	widg, ok := tv.RowFirstVisWidget(ixs[0])
	if ok {
		bi := &gi.Bitmap{}
		bi.InitName(bi, tv.UniqueName())
		if !bi.GrabRenderFrom(widg) { // offscreen!
			log.Printf("giv.TableView: unexpected failure in getting widget pixels -- cannot start DND\n")
			return
		}
		gi.ImageClearer(bi.Pixels, 50.0)
		tv.Viewport.Win.StartDragNDrop(tv.This(), md, bi)
	}
}

// DragNDropTarget handles a drag-n-drop drop
func (tv *TableView) DragNDropTarget(de *dnd.Event) {
	de.Target = tv.This()
	if de.Mod == dnd.DropLink {
		de.Mod = dnd.DropCopy // link not supported -- revert to copy
	}
	idx, ok := tv.IdxFromPos(de.Where.Y)
	if ok {
		de.SetProcessed()
		tv.curIdx = idx
		if dpr, ok := tv.This().(gi.DragNDropper); ok {
			dpr.Drop(de.Data, de.Mod)
		} else {
			tv.Drop(de.Data, de.Mod)
		}
	}
}

// MakeDropMenu makes the menu of options for dropping on a target
func (tv *TableView) MakeDropMenu(m *gi.Menu, data interface{}, mod dnd.DropMods, idx int) {
	if len(*m) > 0 {
		return
	}
	switch mod {
	case dnd.DropCopy:
		m.AddLabel("Copy (Shift=Move):")
	case dnd.DropMove:
		m.AddLabel("Move:")
	}
	if mod == dnd.DropCopy {
		m.AddAction(gi.ActOpts{Label: "Assign To", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			tvv.DropAssign(data.(mimedata.Mimes), idx)
		})
	}
	m.AddAction(gi.ActOpts{Label: "Insert Before", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.DropBefore(data.(mimedata.Mimes), mod, idx) // captures mod
	})
	m.AddAction(gi.ActOpts{Label: "Insert After", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.DropAfter(data.(mimedata.Mimes), mod, idx) // captures mod
	})
	m.AddAction(gi.ActOpts{Label: "Cancel", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TableView).(*TableView)
		tvv.DropCancel()
	})
}

// Drop pops up a menu to determine what specifically to do with dropped items
// this satisfies gi.DragNDropper interface, and can be overwritten in subtypes
func (tv *TableView) Drop(md mimedata.Mimes, mod dnd.DropMods) {
	var men gi.Menu
	tv.MakeDropMenu(&men, md, mod, tv.curIdx)
	pos := tv.IdxPos(tv.curIdx)
	gi.PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvDropMenu")
}

// DropAssign assigns mime data (only the first one!) to this node
func (tv *TableView) DropAssign(md mimedata.Mimes, idx int) {
	tv.DraggedIdxs = nil
	tv.PasteAssign(md, idx)
	tv.DragNDropFinalize(dnd.DropCopy)
}

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore -- ends up calling DragNDropSource if us..
func (tv *TableView) DragNDropFinalize(mod dnd.DropMods) {
	tv.UnselectAllIdxs()
	tv.Viewport.Win.FinalizeDragNDrop(mod)
}

// DragNDropSource is called after target accepts the drop -- we just remove
// elements that were moved
func (tv *TableView) DragNDropSource(de *dnd.Event) {
	if de.Mod != dnd.DropMove || len(tv.DraggedIdxs) == 0 {
		return
	}
	updt := tv.UpdateStart()
	sort.Slice(tv.DraggedIdxs, func(i, j int) bool {
		return tv.DraggedIdxs[i] > tv.DraggedIdxs[j]
	})
	idx := tv.DraggedIdxs[0]
	for _, i := range tv.DraggedIdxs {
		tv.SliceDeleteAt(i, false)
	}
	tv.DraggedIdxs = nil
	tv.UpdateSliceGrid()
	tv.UpdateEnd(updt)
	tv.SelectIdxAction(idx, mouse.SelectOne)
}

// SaveDraggedIdxs saves selectedindexes into dragged indexes taking into account insertion at indexes
func (tv *TableView) SaveDraggedIdxs(idx int) {
	sz := len(tv.SelectedIdxs)
	if sz == 0 {
		tv.DraggedIdxs = nil
		return
	}
	tv.DraggedIdxs = make([]int, len(tv.SelectedIdxs))
	idx = 0
	for i, _ := range tv.SelectedIdxs {
		if i > idx {
			tv.DraggedIdxs[idx] = i + sz // make room for insertion
		} else {
			tv.DraggedIdxs[idx] = i
		}
		idx++
	}
}

// DropBefore inserts object(s) from mime data before this node
func (tv *TableView) DropBefore(md mimedata.Mimes, mod dnd.DropMods, idx int) {
	tv.SaveDraggedIdxs(idx)
	tv.PasteAtIdx(md, idx)
	tv.DragNDropFinalize(mod)
}

// DropAfter inserts object(s) from mime data after this node
func (tv *TableView) DropAfter(md mimedata.Mimes, mod dnd.DropMods, idx int) {
	tv.SaveDraggedIdxs(idx + 1)
	tv.PasteAtIdx(md, idx+1)
	tv.DragNDropFinalize(mod)
}

// DropCancel cancels the drop action e.g., preventing deleting of source
// items in a Move case
func (tv *TableView) DropCancel() {
	tv.DragNDropFinalize(dnd.DropIgnore)
}

//////////////////////////////////////////////////////////////////////////////
//    Events

func (tv *TableView) StdCtxtMenu(m *gi.Menu, idx int) {
	m.AddAction(gi.ActOpts{Label: "Copy", Data: idx},
		tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			tvv.CopyIdxs(true)
		})
	m.AddAction(gi.ActOpts{Label: "Cut", Data: idx},
		tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			tvv.CutIdxs()
		})
	m.AddAction(gi.ActOpts{Label: "Paste", Data: idx},
		tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			tvv.PasteIdx(data.(int))
		})
	m.AddAction(gi.ActOpts{Label: "Duplicate", Data: idx},
		tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			tvv.Duplicate()
		})
}

func (tv *TableView) ItemCtxtMenu(idx int) {
	stru := tv.SliceStruct(idx)
	if stru == nil {
		return
	}
	var men gi.Menu

	if CtxtMenuView(stru, tv.IsInactive(), tv.Viewport, &men) {
		if tv.ShowViewCtxtMenu {
			men.AddSeparator("sep-tvmenu")
			tv.StdCtxtMenu(&men, idx)
		}
	} else {
		tv.StdCtxtMenu(&men, idx)
	}
	if len(men) > 0 {
		pos := tv.IdxPos(idx)
		gi.PopupMenu(men, pos.X, pos.Y, tv.Viewport, tv.Nm+"-menu")
	}
}

func (tv *TableView) KeyInputActive(kt *key.ChordEvent) {
	if gi.KeyEventTrace {
		fmt.Printf("TableView KeyInput: %v\n", tv.PathUnique())
	}
	kf := gi.KeyFun(kt.Chord())
	selMode := mouse.SelectModeBits(kt.Modifiers)
	if selMode == mouse.SelectOne {
		if tv.SelectMode {
			selMode = mouse.ExtendContinuous
		}
	}
	idx := tv.SelectedIdx
	switch kf {
	case gi.KeyFunCancelSelect:
		tv.UnselectAllIdxs()
		tv.SelectMode = false
		kt.SetProcessed()
	case gi.KeyFunMoveDown:
		tv.MoveDownAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunMoveUp:
		tv.MoveUpAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunPageDown:
		tv.MovePageDownAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunPageUp:
		tv.MovePageUpAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunSelectMode:
		tv.SelectMode = !tv.SelectMode
		kt.SetProcessed()
	case gi.KeyFunSelectAll:
		tv.SelectAllIdxs()
		tv.SelectMode = false
		kt.SetProcessed()
	// case gi.KeyFunDelete: // too dangerous
	// 	tv.SliceDelete(tv.SelectedIdx, true)
	// 	tv.SelectMode = false
	// 	tv.SelectIdxAction(idx, mouse.SelectOne)
	// 	kt.SetProcessed()
	case gi.KeyFunDuplicate:
		nidx := tv.Duplicate()
		tv.SelectMode = false
		if nidx >= 0 {
			tv.SelectIdxAction(nidx, mouse.SelectOne)
		}
		kt.SetProcessed()
	case gi.KeyFunInsert:
		tv.SliceNewAt(idx, true)
		tv.SelectMode = false
		tv.SelectIdxAction(idx+1, mouse.SelectOne) // todo: somehow nidx not working
		kt.SetProcessed()
	case gi.KeyFunInsertAfter:
		tv.SliceNewAt(idx+1, true)
		tv.SelectMode = false
		tv.SelectIdxAction(idx+1, mouse.SelectOne)
		kt.SetProcessed()
	case gi.KeyFunCopy:
		tv.CopyIdxs(true)
		tv.SelectMode = false
		tv.SelectIdxAction(idx, mouse.SelectOne)
		kt.SetProcessed()
	case gi.KeyFunCut:
		tv.CutIdxs()
		tv.SelectMode = false
		kt.SetProcessed()
	case gi.KeyFunPaste:
		tv.PasteIdx(tv.SelectedIdx)
		tv.SelectMode = false
		kt.SetProcessed()
	}
}

func (tv *TableView) KeyInputInactive(kt *key.ChordEvent) {
	if gi.KeyEventTrace {
		fmt.Printf("TableView Inactive KeyInput: %v\n", tv.PathUnique())
	}
	kf := gi.KeyFun(kt.Chord())
	idx := tv.SelectedIdx
	switch {
	case kf == gi.KeyFunMoveDown:
		ni := idx + 1
		if ni < tv.SliceSize {
			tv.ScrollToIdx(ni)
			tv.UpdateSelectIdx(ni, true)
			kt.SetProcessed()
		}
	case kf == gi.KeyFunMoveUp:
		ni := idx - 1
		if ni >= 0 {
			tv.ScrollToIdx(ni)
			tv.UpdateSelectIdx(ni, true)
			kt.SetProcessed()
		}
	case kf == gi.KeyFunPageDown:
		ni := ints.MinInt(idx+tv.VisRows, tv.SliceSize-1)
		tv.ScrollToIdx(ni)
		tv.UpdateSelectIdx(ni, true)
		kt.SetProcessed()
	case kf == gi.KeyFunPageUp:
		ni := ints.MaxInt(idx-tv.VisRows, 0)
		tv.ScrollToIdx(ni)
		tv.UpdateSelectIdx(ni, true)
		kt.SetProcessed()
	case kf == gi.KeyFunEnter || kf == gi.KeyFunAccept || kt.Rune == ' ':
		tv.TableViewSig.Emit(tv.This(), int64(TableViewDoubleClicked), tv.SelectedIdx)
		kt.SetProcessed()
	}
}

func (tv *TableView) TableViewEvents() {
	// LowPri to allow other focal widgets to capture
	tv.ConnectEvent(oswin.MouseScrollEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.ScrollEvent)
		tvv := recv.Embed(KiT_TableView).(*TableView)
		me.SetProcessed()
		sbb := tvv.ScrollBar()
		cur := float32(sbb.Pos)
		sbb.SliderMoved(cur, cur-float32(me.NonZeroDelta(false))) // preferY
	})
	if tv.IsInactive() {
		if tv.InactKeyNav {
			tv.ConnectEvent(oswin.KeyChordEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
				tvv := recv.Embed(KiT_TableView).(*TableView)
				kt := d.(*key.ChordEvent)
				tvv.KeyInputInactive(kt)
			})
		}
		tv.ConnectEvent(oswin.MouseEvent, gi.LowRawPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			me := d.(*mouse.Event)
			tvv := recv.Embed(KiT_TableView).(*TableView)
			if !tvv.HasFocus() {
				tvv.GrabFocus()
			}
			if me.Button == mouse.Left && me.Action == mouse.DoubleClick {
				tvv.TableViewSig.Emit(tvv.This(), int64(TableViewDoubleClicked), tvv.SelectedIdx)
				me.SetProcessed()
			}
			if me.Button == mouse.Right && me.Action == mouse.Release {
				tvv.ItemCtxtMenu(tvv.SelectedIdx)
				me.SetProcessed()
			}
		})
	} else {
		tv.ConnectEvent(oswin.MouseEvent, gi.LowRawPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			me := d.(*mouse.Event)
			tvv := recv.Embed(KiT_TableView).(*TableView)
			if me.Button == mouse.Right && me.Action == mouse.Release {
				tvv.ItemCtxtMenu(tvv.SelectedIdx)
				me.SetProcessed()
			}
		})
		tv.ConnectEvent(oswin.KeyChordEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			tvv := recv.Embed(KiT_TableView).(*TableView)
			kt := d.(*key.ChordEvent)
			tvv.KeyInputActive(kt)
		})
		tv.ConnectEvent(oswin.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			de := d.(*dnd.Event)
			tvv := recv.Embed(KiT_TableView).(*TableView)
			switch de.Action {
			case dnd.Start:
				tvv.DragNDropStart()
			case dnd.DropOnTarget:
				tvv.DragNDropTarget(de)
			case dnd.DropFmSource:
				tvv.DragNDropSource(de)
			}
		})
		sg := tv.SliceGrid()
		if sg != nil {
			sg.ConnectEvent(oswin.DNDFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
				de := d.(*dnd.FocusEvent)
				switch de.Action {
				case dnd.Enter:
					tv.Viewport.Win.DNDSetCursor(de.Mod)
				case dnd.Exit:
					tv.Viewport.Win.DNDNotCursor()
				case dnd.Hover:
					// nothing here?
				}
			})
		}
	}
}
