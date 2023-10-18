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
	"strconv"
	"strings"
	"sync"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/filecat"
)

////////////////////////////////////////////////////////////////////////////////////////
//  SliceViewer

// SliceViewer is the interface used by SliceViewBase to support any abstractions
// needed for different types of slice views.
type SliceViewer interface {
	// AsSliceViewBase returns the base for direct access to relevant fields etc
	AsSliceViewBase() *SliceViewBase

	// IsConfiged returns true if is fully configured for display
	IsConfiged() bool

	// SliceGrid returns the SliceGrid grid frame widget, which contains all the
	// fields and values
	SliceGrid() *gi.Frame

	// ScrollBar returns the SliceGrid scrollbar
	ScrollBar() *gi.Slider

	// RowWidgetNs returns number of widgets per row and offset for index label
	RowWidgetNs() (nWidgPerRow, idxOff int)

	// SliceSize returns the current size of the slice and sets SliceSize
	UpdtSliceSize() int

	// LayoutSliceGrid does the proper layout of slice grid depending on allocated size
	// returns true if UpdateSliceGrid should be called after this
	LayoutSliceGrid() bool

	// UpdateSliceGrid updates grid display -- robust to any time calling
	UpdateSliceGrid()

	// SliceGridNeedsUpdate returns true when slice grid needs to be updated.
	// this should be true if the underlying size has changed, or other
	// indication that the data might have changed.
	SliceGridNeedsUpdate() bool

	// StyleRow calls a custom style function on given row (and field)
	StyleRow(svnp reflect.Value, widg gi.Widget, idx, fidx int, vv Value)

	// RowFirstWidget returns the first widget for given row (could be index or
	// not) -- false if out of range
	RowFirstWidget(row int) (*gi.WidgetBase, bool)

	// RowGrabFocus grabs the focus for the first focusable widget in given row --
	// returns that element or nil if not successful -- note: grid must have
	// already rendered for focus to be grabbed!
	RowGrabFocus(row int) *gi.WidgetBase

	// SelectRowWidgets sets the selection state of given row of widgets
	SelectRowWidgets(row int, sel bool)

	// SliceNewAt inserts a new blank element at given index in the slice -- -1
	// means the end
	SliceNewAt(idx int)

	// SliceDeleteAt deletes element at given index from slice
	// if updt is true, then update the grid after
	SliceDeleteAt(idx int)

	// MimeDataType returns the data type for mime clipboard (copy / paste) data
	// e.g., filecat.DataJson
	MimeDataType() string

	// CopySelToMime copies selected rows to mime data
	CopySelToMime() mimedata.Mimes

	// PasteAssign assigns mime data (only the first one!) to this idx
	PasteAssign(md mimedata.Mimes, idx int)

	// PasteAtIdx inserts object(s) from mime data at (before) given slice index
	PasteAtIdx(md mimedata.Mimes, idx int)

	// ItemCtxtMenu pulls up the context menu for given slice index
	ItemCtxtMenu(idx int)

	// StdCtxtMenu generates the standard context menu for this view
	StdCtxtMenu(m *gi.Menu, idx int)

	// NeedsDoubleReRender returns true if initial render requires a 2nd pass
	NeedsDoubleReRender() bool
}

////////////////////////////////////////////////////////////////////////////////////////
//  SliceViewBase

// SliceViewBase is the base for SliceView and TableView and any other viewers
// of array-like data.  It automatically computes the number of rows that fit
// within its allocated space, and manages the offset view window into the full
// list of items, and supports row selection, copy / paste, Drag-n-Drop, etc.
// Set to Inactive for select-only mode, which emits WidgetSig WidgetSelected
// signals when selection is updated.
// Automatically has a toolbar with Slice Toolbar props if defined
// set prop toolbar = false to turn off
type SliceViewBase struct {
	gi.Frame

	// the slice that we are a view onto -- must be a pointer to that slice
	Slice any `copy:"-" view:"-" json:"-" xml:"-"`

	// optional mutex that, if non-nil, will be used around any updates that read / modify the underlying Slice data -- can be used to protect against random updating if your code has specific update points that can be likewise protected with this same mutex
	ViewMu *sync.Mutex `copy:"-" view:"-" json:"-" xml:"-"`

	// non-ptr reflect.Value of the slice
	SliceNPVal reflect.Value `copy:"-" view:"-" json:"-" xml:"-"`

	// Value for the slice itself, if this was created within value view framework -- otherwise nil
	SliceValView Value `copy:"-" view:"-" json:"-" xml:"-"`

	// whether the slice is actually an array -- no modifications -- set by SetSlice
	isArray bool `copy:"-" view:"-" json:"-" xml:"-"`

	// if true, user cannot add elements to the slice
	NoAdd bool

	// if true, user cannot delete elements from the slice
	NoDelete bool

	// if the type we're viewing has its own CtxtMenu property defined, should we also still show the view's standard context menu?
	ShowViewCtxtMenu bool

	// has the slice been edited?
	Changed bool

	// Value representations of the slice values
	Values []Value `copy:"-" view:"-" json:"-" xml:"-"`

	// whether to show index or not
	ShowIndex bool

	// whether to show the toolbar or not
	ShowToolbar bool

	// support key navigation when inactive (default true) -- no focus really plausible in inactive case, so it uses a low-pri capture of up / down events
	InactKeyNav bool

	// current selection value -- initially select this value if set
	SelVal any `copy:"-" view:"-" json:"-" xml:"-"`

	// index of currently-selected item, in Inactive mode only
	SelectedIdx int `copy:"-" json:"-" xml:"-"`

	// editing-mode select rows mode
	SelectMode bool `copy:"-"`

	// if view is inactive, default selection mode is to choose one row only -- if this is true, standard multiple selection logic with modifier keys is instead supported
	InactMultiSel bool

	// list of currently-selected slice indexes
	SelectedIdxs map[int]struct{} `copy:"-"`

	// list of currently-dragged indexes
	DraggedIdxs []int `copy:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `copy:"-" json:"-" xml:"-"`

	// the slice that we successfully set a toolbar for
	ToolbarSlice any `copy:"-" view:"-" json:"-" xml:"-"`

	// size of slice
	SliceSize int `inactive:"+" copy:"-" json:"-" xml:"-"`

	// actual number of rows displayed = min(VisRows, SliceSize)
	DispRows int `inactive:"+" copy:"-" json:"-" xml:"-"`

	// starting slice index of visible rows
	StartIdx int `inactive:"+" copy:"-" json:"-" xml:"-"`

	// height of a single row
	RowHeight float32 `inactive:"+" copy:"-" json:"-" xml:"-"`

	// total number of rows visible in allocated display size
	VisRows int `inactive:"+" copy:"-" json:"-" xml:"-"`

	// the height of grid from last layout -- determines when update needed
	LayoutHeight float32 `copy:"-" view:"-" json:"-" xml:"-"`

	// the number of rows rendered -- determines update
	RenderedRows int `copy:"-" view:"-" json:"-" xml:"-"`

	// guard for recursive focus grabbing
	InFocusGrab bool `copy:"-" view:"-" json:"-" xml:"-"`

	// guard for recursive rebuild
	InFullRebuild bool `copy:"-" view:"-" json:"-" xml:"-"`

	// temp idx state for e.g., dnd
	CurIdx int `copy:"-" view:"-" json:"-" xml:"-"`
}

func (sv *SliceViewBase) OnInit() {
	sv.SelectMode = false
	sv.ShowIndex = true
	sv.ShowToolbar = true
	sv.InactKeyNav = true

	sv.HandleSliceViewEvents()

	sv.Lay = gi.LayoutVert
	sv.Style(func(s *styles.Style) {
		sv.Spacing = gi.StdDialogVSpaceUnits
		s.SetStretchMax()
	})
}

func (sv *SliceViewBase) OnChildAdded(child ki.Ki) {
	w, _ := gi.AsWidget(child)
	switch w.PathFrom(sv.This()) {
	case "grid-lay": // grid layout
		gl := w.(*gi.Layout)
		gl.Lay = gi.LayoutHoriz
		w.Style(func(s *styles.Style) {
			gl.SetStretchMax() // for this to work, ALL layers above need it too
		})
	case "grid-lay/grid": // slice grid
		sg := w.(*gi.Frame)
		sg.Lay = gi.LayoutGrid
		sg.Stripes = gi.RowStripes
		sg.Style(func(s *styles.Style) {
			nWidgPerRow, _ := sv.RowWidgetNs()
			s.Columns = nWidgPerRow
			// setting a pref here is key for giving it a scrollbar in larger context
			s.SetMinPrefHeight(units.Em(6))
			s.SetMinPrefWidth(units.Ch(20))
			s.SetStretchMax()                  // for this to work, ALL layers above need it too
			s.Overflow = styles.OverflowScroll // this still gives it true size during PrefSize
		})
	}
	if w.Parent().Name() == "grid" && strings.HasPrefix(w.Name(), "index-") {
		w.Style(func(s *styles.Style) {
			s.MinWidth.SetEm(1.5)
			s.Padding.Right.SetDp(4)
			s.Text.Align = styles.AlignRight
		})
	}
	if w.Parent().Name() == "grid" && (strings.HasPrefix(w.Name(), "add-") || strings.HasPrefix(w.Name(), "del-")) {
		w.Style(func(s *styles.Style) {
			w.(*gi.Button).SetType(gi.ButtonAction)
		})
	}
}

func (sv *SliceViewBase) AsSliceViewBase() *SliceViewBase {
	return sv
}

// Each SliceView must implement its own SetSlice, Config, etc pipeline.
// This one is for a basic SliceView
// the only interface call is UpdateSliceGrid()

// SetSlice sets the source slice that we are viewing -- rebuilds the children
// to represent this slice
func (sv *SliceViewBase) SetSlice(sl any) {
	if laser.AnyIsNil(sl) {
		sv.Slice = nil
		return
	}
	newslc := false
	if reflect.TypeOf(sl).Kind() != reflect.Pointer { // prevent crash on non-comparable
		newslc = true
	} else {
		newslc = (sv.Slice != sl)
	}
	if !newslc && sv.IsConfiged() {
		sv.Update()
		return
	}
	updt := sv.UpdateStart()
	sv.StartIdx = 0
	sv.Slice = sl
	sv.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(sv.Slice))
	sv.isArray = laser.NonPtrType(reflect.TypeOf(sl)).Kind() == reflect.Array
	// make sure elements aren't nil to prevent later panics
	for i := 0; i < sv.SliceNPVal.Len(); i++ {
		val := sv.SliceNPVal.Index(i)
		k := val.Kind()
		if (k == reflect.Chan || k == reflect.Func || k == reflect.Interface || k == reflect.Map || k == reflect.Pointer || k == reflect.Slice) && val.IsNil() {
			val.Set(reflect.New(laser.NonPtrType(val.Type())))
		}
	}
	if !sv.IsDisabled() {
		sv.SelectedIdx = -1
	}
	sv.ResetSelectedIdxs()
	sv.Config(sv.Sc)
	sv.UpdateEndLayout(updt)
}

// Update is the high-level update display call -- robust to any changes
func (sv *SliceViewBase) Update() {
	if !sv.This().(gi.Widget).IsVisible() {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)
}

// UpdateValues updates the widget display of slice values, assuming same slice config
func (sv *SliceViewBase) UpdateValues() {
	updt := sv.UpdateStart()
	for _, vv := range sv.Values {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

// Config configures a standard setup of the overall Frame
func (sv *SliceViewBase) ConfigWidget(sc *gi.Scene) {
	config := ki.Config{}
	config.Add(gi.ToolbarType, "toolbar")
	config.Add(gi.LayoutType, "grid-lay")
	mods, updt := sv.ConfigChildren(config)

	gl := sv.GridLayout()
	gconfig := ki.Config{}
	gconfig.Add(gi.FrameType, "grid")
	gconfig.Add(gi.SliderType, "scrollbar")
	gl.ConfigChildren(gconfig) // covered by above

	sv.ConfigSliceGrid()
	sv.ConfigToolbar()

	if sv.Sc != nil && sv.Sc.MainStageMgr() != nil {
		sv.ApplyStyleTree(sc)
	}
	if mods {
		sv.UpdateEndLayout(updt)
	}
}

// IsConfiged returns true if the widget is fully configured
func (sv *SliceViewBase) IsConfiged() bool {
	if len(sv.Kids) == 0 {
		return false
	}
	return true
}

// GridLayout returns the Layout containing the Grid and the scrollbar
func (sv *SliceViewBase) GridLayout() *gi.Layout {
	return sv.ChildByName("grid-lay", 0).(*gi.Layout)
}

// SliceGrid returns the SliceGrid grid frame widget, which contains all the
// fields and values
func (sv *SliceViewBase) SliceGrid() *gi.Frame {
	return sv.GridLayout().ChildByName("grid", 0).(*gi.Frame)
}

// ScrollBar returns the SliceGrid scrollbar
func (sv *SliceViewBase) ScrollBar() *gi.Slider {
	return sv.GridLayout().ChildByName("scrollbar", 1).(*gi.Slider)
}

// Toolbar returns the toolbar widget
func (sv *SliceViewBase) Toolbar() *gi.Toolbar {
	tbi := sv.ChildByName("toolbar", 1)
	if tbi == nil {
		return nil
	}
	return tbi.(*gi.Toolbar)
}

// RowWidgetNs returns number of widgets per row and offset for index label
func (sv *SliceViewBase) RowWidgetNs() (nWidgPerRow, idxOff int) {
	nWidgPerRow = 2
	if !sv.IsDisabled() && !sv.isArray {
		if !sv.NoAdd {
			nWidgPerRow += 1
		}
		if !sv.NoDelete {
			nWidgPerRow += 1
		}
	}
	idxOff = 1
	if !sv.ShowIndex {
		nWidgPerRow -= 1
		idxOff = 0
	}
	return
}

// UpdtSliceSize updates and returns the size of the slice and sets SliceSize
func (sv *SliceViewBase) UpdtSliceSize() int {
	sz := sv.SliceNPVal.Len()
	sv.SliceSize = sz
	return sz
}

// SliceGridNeedsUpdate returns true when slice grid needs to be updated.
// this should be true if the underlying size has changed, or other
// indication that the data might have changed.
func (sv *SliceViewBase) SliceGridNeedsUpdate() bool {
	csz := sv.SliceSize
	nsz := sv.This().(SliceViewer).UpdtSliceSize()
	return csz != nsz
}

// ViewMuLock locks the ViewMu if non-nil
func (sv *SliceViewBase) ViewMuLock() {
	if sv.ViewMu == nil {
		return
	}
	sv.ViewMu.Lock()
}

// ViewMuUnlock Unlocks the ViewMu if non-nil
func (sv *SliceViewBase) ViewMuUnlock() {
	if sv.ViewMu == nil {
		return
	}
	sv.ViewMu.Unlock()
}

// ConfigSliceGrid configures the SliceGrid for the current slice
// it is only called once at start, under overall Config
func (sv *SliceViewBase) ConfigSliceGrid() {
	sg := sv.This().(SliceViewer).SliceGrid()
	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	nWidgPerRow, idxOff := sv.RowWidgetNs()
	sc := sv.Sc

	sg.DeleteChildren(ki.DestroyKids)

	if laser.AnyIsNil(sv.Slice) {
		return
	}
	sz := sv.This().(SliceViewer).UpdtSliceSize()
	if sz == 0 {
		return
	}

	sg.Kids = make(ki.Slice, nWidgPerRow)

	// at this point, we make one dummy row to get size of widgets
	val := laser.OnePtrUnderlyingValue(sv.SliceNPVal.Index(0)) // deal with pointer lists
	vv := ToValue(val.Interface(), "")
	if vv == nil { // shouldn't happen
		return
	}
	vv.SetSliceValue(val, sv.Slice, 0, sv.TmpSave, sv.ViewPath)
	vtyp := vv.WidgetType()
	itxt := fmt.Sprintf("%05d", 0)
	labnm := fmt.Sprintf("index-%v", itxt)
	valnm := fmt.Sprintf("value-%v", itxt)

	if sv.ShowIndex {
		idxlab := &gi.Label{}
		sg.SetChild(idxlab, 0, labnm)
		idxlab.Text = itxt
	}

	widg := ki.NewOfType(vtyp).(gi.Widget)
	sg.SetChild(widg, idxOff, valnm)
	vv.ConfigWidget(widg, sc)

	if !sv.IsDisabled() && !sv.isArray {
		cidx := idxOff
		if !sv.NoAdd {
			cidx++
			addnm := fmt.Sprintf("add-%v", itxt)
			addbt := gi.Button{}
			sg.SetChild(&addbt, cidx, addnm)
			addbt.SetType(gi.ButtonAction)
			addbt.SetIcon(icons.Add)
		}
		if !sv.NoDelete {
			cidx++
			delnm := fmt.Sprintf("del-%v", itxt)
			delbt := gi.Button{}
			sg.SetChild(&delbt, cidx, delnm)
			delbt.SetType(gi.ButtonAction)
			delbt.SetIcon(icons.Delete)
			delbt.Style(func(s *styles.Style) {
				s.Color = colors.Scheme.Error.Base
			})
		}
	}
	sv.ConfigScroll()
}

// ConfigScroll configures the scrollbar
func (sv *SliceViewBase) ConfigScroll() {
	sb := sv.This().(SliceViewer).ScrollBar()
	sb.Dim = mat32.Y
	sb.Tracking = true
	if sv.Styles.ScrollBarWidth.Dots == 0 {
		sb.SetFixedWidth(units.Dp(16))
	} else {
		sb.SetFixedWidth(sv.Styles.ScrollBarWidth)
	}
	sb.SetStretchMaxHeight()
	sb.Min = 0
	sb.Step = 1
	sv.UpdateScroll()
	sb.OnChange(func(e events.Event) {
		updt := sv.UpdateStart()
		sv.StartIdx = int(sb.Value)
		sv.This().(SliceViewer).UpdateSliceGrid()
		sv.UpdateEndRender(updt)
	})
}

func (sv *SliceViewBase) DoLayout(sc *gi.Scene, parBBox image.Rectangle, iter int) bool {
	sv.This().(SliceViewer).LayoutSliceGrid()
	sv.This().(SliceViewer).UpdateSliceGrid()
	return sv.Frame.DoLayout(sc, parBBox, iter)
}

// UpdateStartIdx updates StartIdx to fit current view
func (sv *SliceViewBase) UpdateStartIdx() {
	sz := sv.This().(SliceViewer).UpdtSliceSize()
	if sz > sv.DispRows {
		lastSt := sz - sv.DispRows
		sv.StartIdx = min(lastSt, sv.StartIdx)
		sv.StartIdx = max(0, sv.StartIdx)
	} else {
		sv.StartIdx = 0
	}
}

// UpdateScroll updates grid scrollbar based on display
func (sv *SliceViewBase) UpdateScroll() {
	sb := sv.This().(SliceViewer).ScrollBar()
	updt := sb.UpdateStart()
	sb.Max = float32(sv.SliceSize) + 0.01 // bit of extra to ensure last line always shows up
	if sv.DispRows > 0 {
		sb.PageStep = float32(sv.DispRows) * sb.Step
		sb.ThumbVal = float32(sv.DispRows)
	} else {
		sb.PageStep = 10 * sb.Step
		sb.ThumbVal = 10
	}
	sb.TrackThr = sb.Step
	sb.SetValue(float32(sv.StartIdx)) // essential for updating pos from value
	if sv.DispRows == sv.SliceSize {
		sb.Off = true
	} else {
		sb.Off = false
	}
	sb.UpdateEnd(updt)
}

func (sv *SliceViewBase) AvailHeight() float32 {
	sg := sv.This().(SliceViewer).SliceGrid()
	sgHt := sg.LayState.Alloc.Size.Y
	if sgHt == 0 {
		return 0
	}
	sgHt -= sg.ExtraSize.Y + sg.Styles.BoxSpace().Size().Y
	return sgHt
}

// LayoutSliceGrid does the proper layout of slice grid depending on allocated size
// returns true if UpdateSliceGrid should be called after this
func (sv *SliceViewBase) LayoutSliceGrid() bool {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return false
	}

	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	if laser.AnyIsNil(sv.Slice) {
		sg.DeleteChildren(ki.DestroyKids)
		return false
	}

	sv.ViewMuLock()
	defer sv.ViewMuUnlock()

	sz := sv.This().(SliceViewer).UpdtSliceSize()
	if sz == 0 {
		sg.DeleteChildren(ki.DestroyKids)
		return false
	}

	nWidgPerRow, _ := sv.RowWidgetNs()
	if len(sg.GridData) > 0 && len(sg.GridData[gi.Row]) > 0 {
		sv.RowHeight = sg.GridData[gi.Row][0].AllocSize + 4*sg.Spacing.Dots
	}
	if sv.Styles.Font.Face == nil {
		sv.Styles.Font = paint.OpenFont(sv.Styles.FontRender(), &sv.Styles.UnContext)
	}
	sv.RowHeight = mat32.Max(sv.RowHeight, sv.Styles.Font.Face.Metrics.Height)

	sc := sv.Sc
	if sc != nil && sc.Is(gi.ScPrefSizing) {
		sv.VisRows = gi.LayoutPrefMaxRows
		sv.LayoutHeight = float32(sv.VisRows) * sv.RowHeight
	} else {
		sgHt := sv.AvailHeight()
		sv.LayoutHeight = sgHt
		if sgHt == 0 {
			return false
		}
		sv.VisRows = int(mat32.Floor(sgHt / sv.RowHeight))
	}
	sv.DispRows = min(sv.SliceSize, sv.VisRows)
	// fmt.Println(sv.DispRows, sv.SliceSize, sv.VisRows, sv.RowHeight)

	nWidg := nWidgPerRow * sv.DispRows

	if sv.Values == nil || sg.NumChildren() != nWidg {
		sg.DeleteChildren(ki.DestroyKids)

		sv.Values = make([]Value, sv.DispRows)
		sg.Kids = make(ki.Slice, nWidg)
	}
	sv.ConfigScroll()
	return true
}

func (sv *SliceViewBase) SliceGridNeedsLayout() bool {
	sgHt := sv.AvailHeight()
	if sgHt != sv.LayoutHeight {
		return true
	}
	return sv.RenderedRows != sv.DispRows
}

// UpdateSliceGrid updates grid display -- robust to any time calling
func (sv *SliceViewBase) UpdateSliceGrid() {
	sg := sv.This().(SliceViewer).SliceGrid()
	if sg == nil {
		return
	}
	updt := sg.UpdateStart()
	defer sg.UpdateEnd(updt)

	if laser.AnyIsNil(sv.Slice) {
		sg.DeleteChildren(ki.DestroyKids)
		return
	}

	sv.ViewMuLock()
	defer sv.ViewMuUnlock()

	sz := sv.This().(SliceViewer).UpdtSliceSize()
	if sz == 0 {
		sg.DeleteChildren(ki.DestroyKids)
		return
	}
	sv.DispRows = min(sv.SliceSize, sv.VisRows)

	nWidgPerRow, idxOff := sv.RowWidgetNs()
	nWidg := nWidgPerRow * sv.DispRows
	sc := sv.Sc

	if sv.Values == nil || sg.NumChildren() != nWidg { // shouldn't happen..
		sv.ViewMuUnlock()
		sv.LayoutSliceGrid()
		sv.ViewMuLock()
		nWidg = nWidgPerRow * sv.DispRows
	}

	sv.UpdateStartIdx()

	for i := 0; i < sv.DispRows; i++ {
		i := i
		ridx := i * nWidgPerRow
		si := sv.StartIdx + i // slice idx
		issel := sv.IdxIsSelected(si)
		val := laser.OnePtrUnderlyingValue(sv.SliceNPVal.Index(si)) // deal with pointer lists
		var vv Value
		if sv.Values[i] == nil {
			vv = ToValue(val.Interface(), "")
			sv.Values[i] = vv
		} else {
			vv = sv.Values[i]
		}
		vv.SetSliceValue(val, sv.Slice, si, sv.TmpSave, sv.ViewPath)

		vtyp := vv.WidgetType()
		itxt := strconv.Itoa(i)
		sitxt := strconv.Itoa(si)
		labnm := "index-" + itxt
		valnm := "value-" + itxt

		if sv.ShowIndex {
			var idxlab *gi.Label
			if sg.Kids[ridx] != nil {
				idxlab = sg.Kids[ridx].(*gi.Label)
			} else {
				idxlab = &gi.Label{}
				sg.SetChild(idxlab, ridx, labnm)
				// todo:
				// idxlab.Selectable = true
				// idxlab.Redrawable = true
				idxlab.OnSelect(func(e events.Event) {
					sv.UpdateSelectRow(i, sg.StateIs(states.Selected))
				})
			}
			idxlab.SetText(sitxt)
			idxlab.SetSelected(issel)
		}

		var widg gi.Widget
		if sg.Kids[ridx+idxOff] != nil {
			widg = sg.Kids[ridx+idxOff].(gi.Widget)
			vv.UpdateWidget()
			if sv.IsDisabled() {
				widg.AsWidget().SetState(true, states.Disabled)
			}
			widg.AsWidget().SetSelected(issel)
		} else {
			widg = ki.NewOfType(vtyp).(gi.Widget)
			sg.SetChild(widg, ridx+idxOff, valnm)
			vv.ConfigWidget(widg, sc)
			wb := widg.AsWidget()
			// wb.Sty.Template = "giv.SliceViewBase.ItemWidget." + vtyp.Name()

			if sv.IsDisabled() {
				widg.AsWidget().SetState(true, states.Disabled)
				if wb != nil {
					wb.SetProp("slv-row", i)
					wb.SetState(false, states.Selected)
					wb.OnSelect(func(e events.Event) {
						sv.UpdateSelectRow(i, wb.StateIs(states.Selected))
					})
				}
			} else {
				vvb := vv.AsValueBase()
				vvb.OnChange(func(e events.Event) {
					sv.SendChange()
				})
				if !sv.isArray {
					cidx := ridx + idxOff
					if !sv.NoAdd {
						cidx++
						addnm := fmt.Sprintf("add-%v", itxt)
						addact := gi.Button{}
						sg.SetChild(&addact, cidx, addnm)

						addact.SetIcon(icons.Add)
						addact.Tooltip = "insert a new element at this index"
						addact.Data = i
						addact.OnClick(func(e events.Event) {
							sv.SliceNewAtRow(i + 1)
						})
					}

					if !sv.NoDelete {
						cidx++
						delnm := fmt.Sprintf("del-%v", itxt)
						delact := gi.Button{}
						sg.SetChild(&delact, cidx, delnm)

						delact.SetIcon(icons.Delete)
						delact.Tooltip = "delete this element"
						delact.Data = i
						delact.OnClick(func(e events.Event) {
							sv.SliceDeleteAtRow(i)
						})
					}
				}
			}
		}
		sv.This().(SliceViewer).StyleRow(sv.SliceNPVal, widg, si, 0, vv)
	}
	if sv.SelVal != nil {
		sv.SelectedIdx, _ = SliceIdxByValue(sv.Slice, sv.SelVal)
	}
	if sv.IsDisabled() && sv.SelectedIdx >= 0 {
		sv.SelectIdxWidgets(sv.SelectedIdx, true)
	}
	sv.UpdateScroll()
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// SliceViewBase, indicating that some kind of edit / change has taken place to
// the table data.  It isn't really practical to record all the different
// types of changes, so this is just generic.
func (sv *SliceViewBase) SetChanged() {
	sv.Changed = true
	sv.SendChange()
	sv.Toolbar().UpdateButtons() // nil safe
}

// SliceNewAtRow inserts a new blank element at given display row
func (sv *SliceViewBase) SliceNewAtRow(row int) {
	sv.This().(SliceViewer).SliceNewAt(sv.StartIdx + row)
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (sv *SliceViewBase) SliceNewAt(idx int) {
	if sv.isArray {
		return
	}

	sv.ViewMuLock() // no return!  must unlock before return below

	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	sv.SliceNewAtSel(idx)

	sltyp := laser.SliceElType(sv.Slice) // has pointer if it is there
	iski := ki.IsKi(sltyp)
	slptr := sltyp.Kind() == reflect.Ptr

	svl := reflect.ValueOf(sv.Slice)
	sz := sv.SliceSize

	svnp := sv.SliceNPVal

	if iski && sv.SliceValView != nil {
		vvb := sv.SliceValView.AsValueBase()
		if vvb.Owner != nil {
			if ownki, ok := vvb.Owner.(ki.Ki); ok {
				gi.NewKiDialog(sv, gi.DlgOpts{Title: "Slice New", Prompt: "Number and Type of Items to Insert:", Ok: true, Cancel: true},
					ownki.BaseType(), func(dlg *gi.Dialog) {
						if dlg.Accepted {
							typ := dlg.Data.(*gti.Type) // todo: n!
							n := 1
							updt := ownki.UpdateStart()
							for i := 0; i < n; i++ {
								nm := fmt.Sprintf("New%v%v", typ.Name, idx+1+i)
								ownki.InsertNewChild(typ, idx+1+i, nm)
							}
							sv.SetChanged()
							ownki.UpdateEnd(updt)
						}
					})
			}
		}
	} else {
		nval := reflect.New(laser.NonPtrType(sltyp)) // make the concrete el
		if !slptr {
			nval = nval.Elem() // use concrete value
		}
		svnp = reflect.Append(svnp, nval)
		if idx >= 0 && idx < sz {
			reflect.Copy(svnp.Slice(idx+1, sz+1), svnp.Slice(idx, sz))
			svnp.Index(idx).Set(nval)
		}
		svl.Elem().Set(svnp)
	}
	if idx < 0 {
		idx = sz
	}

	sv.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(sv.Slice)) // need to update after changes

	sv.This().(SliceViewer).UpdtSliceSize()

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetChanged()
	sv.ViewMuUnlock()
}

// SliceDeleteAtRow deletes element at given display row
// if updt is true, then update the grid after
func (sv *SliceViewBase) SliceDeleteAtRow(row int) {
	sv.This().(SliceViewer).SliceDeleteAt(sv.StartIdx + row)
}

// SliceNewAtSel updates selected rows based on
// inserting new element at given index.
// must be called with successful SliceNewAt
func (sv *SliceViewBase) SliceNewAtSel(idx int) {
	sl := sv.SelectedIdxsList(false) // ascending
	sv.ResetSelectedIdxs()
	for _, ix := range sl {
		if ix >= idx {
			ix++
		}
		sv.SelectedIdxs[ix] = struct{}{}
	}
}

// SliceDeleteAtSel updates selected rows based on
// deleting element at given index
// must be called with successful SliceDeleteAt
func (sv *SliceViewBase) SliceDeleteAtSel(idx int) {
	sl := sv.SelectedIdxsList(true) // desscending
	sv.ResetSelectedIdxs()
	for _, ix := range sl {
		switch {
		case ix == idx:
			continue
		case ix > idx:
			ix--
		}
		sv.SelectedIdxs[ix] = struct{}{}
	}
}

// SliceDeleteAt deletes element at given index from slice
func (sv *SliceViewBase) SliceDeleteAt(idx int) {
	if sv.isArray {
		return
	}
	if idx < 0 || idx >= sv.SliceSize {
		return
	}
	sv.ViewMuLock()
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	sv.SliceDeleteAtSel(idx)

	laser.SliceDeleteAt(sv.Slice, idx)

	sv.This().(SliceViewer).UpdtSliceSize()

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}

	sv.ViewMuUnlock()
	sv.SetChanged()
}

// ConfigToolbar configures the toolbar actions
func (sv *SliceViewBase) ConfigToolbar() {
	if laser.AnyIsNil(sv.Slice) {
		return
	}
	if sv.ToolbarSlice == sv.Slice {
		return
	}
	if !sv.ShowToolbar {
		sv.ToolbarSlice = sv.Slice
		return
	}
	tb := sv.Toolbar()
	ndef := 2 // number of default actions
	if sv.isArray || sv.IsDisabled() || sv.NoAdd {
		ndef = 1
	}
	if len(*tb.Children()) < ndef {
		tb.SetStretchMaxWidth()
		tb.AddButton(gi.ActOpts{Name: "UpdateView", Label: "Update view", Icon: icons.Refresh, Tooltip: "update this SliceView to reflect current state of slice"}, func(act *gi.Button) {
			sv.SetNeedsLayout()

		})
		if ndef > 1 {
			tb.AddButton(gi.ActOpts{Label: "Add", Icon: icons.Add, Tooltip: "add a new element to the slice"},
				func(act *gi.Button) {
					sv.This().(SliceViewer).SliceNewAt(-1)
				})
		}
	}
	sz := len(*tb.Children())
	if sz > ndef {
		for i := sz - 1; i >= ndef; i-- {
			tb.DeleteChildAtIndex(i, ki.DestroyKids)
		}
	}
	ToolbarView(sv.Slice, tb)
	sv.ToolbarSlice = sv.Slice
}

func (sv *SliceViewBase) ApplyStyle(sc *gi.Scene) {
	sv.Frame.ApplyStyle(sc)
	// if sv.IsDisabled() {
	// 	sv.SetCanFocus()
	// }
	// // sg := sv.This().(SliceViewer).SliceGrid()
	// // sg.StartFocus() // need to call this when window is actually active
}

func (sv *SliceViewBase) Render(sc *gi.Scene) {
	// sv.Toolbar().UpdateButtons()
	if sv.PushBounds(sc) {
		sv.FrameStdRender(sc)
		sv.RenderScrolls(sc)
		sv.RenderChildren(sc)
		sv.PopBounds(sc)
	}
}

func (sv *SliceViewBase) NeedsDoubleReRender() bool {
	return false
}

// func (sv *SliceViewBase) StateIs(states.Focused) bool {
// 	if !sv.ContainsFocus() {
// 		return false
// 	}
// 	if sv.IsDisabled() {
// 		return sv.InactKeyNav
// 	}
// 	return true
// }

//////////////////////////////////////////////////////////////////////////////
//  Row access methods
//  NOTE: row = physical GUI display row, idx = slice index -- not the same!

// SliceVal returns value interface at given slice index
// must be protected by mutex
func (sv *SliceViewBase) SliceVal(idx int) any {
	if idx < 0 || idx >= sv.SliceSize {
		fmt.Printf("giv.SliceViewBase: slice index out of range: %v\n", idx)
		return nil
	}
	val := laser.OnePtrUnderlyingValue(sv.SliceNPVal.Index(idx)) // deal with pointer lists
	vali := val.Interface()
	return vali
}

// IsRowInBounds returns true if disp row is in bounds
func (sv *SliceViewBase) IsRowInBounds(row int) bool {
	return row >= 0 && row < sv.DispRows
}

// IsIdxVisible returns true if slice index is currently visible
func (sv *SliceViewBase) IsIdxVisible(idx int) bool {
	return sv.IsRowInBounds(idx - sv.StartIdx)
}

// RowFirstWidget returns the first widget for given row (could be index or
// not) -- false if out of range
func (sv *SliceViewBase) RowFirstWidget(row int) (*gi.WidgetBase, bool) {
	if !sv.ShowIndex {
		return nil, false
	}
	if !sv.IsRowInBounds(row) {
		return nil, false
	}
	nWidgPerRow, _ := sv.This().(SliceViewer).RowWidgetNs()
	sg := sv.This().(SliceViewer).SliceGrid()
	widg := sg.Kids[row*nWidgPerRow].(gi.Widget).AsWidget()
	return widg, true
}

// RowGrabFocus grabs the focus for the first focusable widget in given row --
// returns that element or nil if not successful -- note: grid must have
// already rendered for focus to be grabbed!
func (sv *SliceViewBase) RowGrabFocus(row int) *gi.WidgetBase {
	if !sv.IsRowInBounds(row) || sv.InFocusGrab { // range check
		return nil
	}
	nWidgPerRow, idxOff := sv.This().(SliceViewer).RowWidgetNs()
	ridx := nWidgPerRow * row
	sg := sv.This().(SliceViewer).SliceGrid()
	widg := sg.Child(ridx + idxOff).(gi.Widget).AsWidget()
	if widg.StateIs(states.Focused) {
		return widg
	}
	sv.InFocusGrab = true
	widg.GrabFocus()
	sv.InFocusGrab = false
	return widg
}

// IdxGrabFocus grabs the focus for the first focusable widget in given idx --
// returns that element or nil if not successful
func (sv *SliceViewBase) IdxGrabFocus(idx int) *gi.WidgetBase {
	sv.ScrollToIdx(idx)
	return sv.This().(SliceViewer).RowGrabFocus(idx - sv.StartIdx)
}

// IdxPos returns center of window position of index label for idx (ContextMenuPos)
func (sv *SliceViewBase) IdxPos(idx int) image.Point {
	row := idx - sv.StartIdx
	if row < 0 {
		row = 0
	}
	if row > sv.DispRows-1 {
		row = sv.DispRows - 1
	}
	var pos image.Point
	widg, ok := sv.This().(SliceViewer).RowFirstWidget(row)
	if ok {
		pos = widg.ContextMenuPos(nil)
	}
	return pos
}

// RowFromPos returns the row that contains given vertical position, false if not found
func (sv *SliceViewBase) RowFromPos(posY int) (int, bool) {
	// todo: could optimize search to approx loc, and search up / down from there
	for rw := 0; rw < sv.DispRows; rw++ {
		widg, ok := sv.This().(SliceViewer).RowFirstWidget(rw)
		if ok {
			if widg.ScBBox.Min.Y < posY && posY < widg.ScBBox.Max.Y {
				return rw, true
			}
		}
	}
	return -1, false
}

// IdxFromPos returns the idx that contains given vertical position, false if not found
func (sv *SliceViewBase) IdxFromPos(posY int) (int, bool) {
	row, ok := sv.RowFromPos(posY)
	if !ok {
		return -1, false
	}
	return row + sv.StartIdx, true
}

// ScrollToIdxNoUpdt ensures that given slice idx is visible by scrolling display as needed
// This version does not update the slicegrid -- just computes the StartIdx and updates the scrollbar
func (sv *SliceViewBase) ScrollToIdxNoUpdt(idx int) bool {
	if sv.DispRows == 0 {
		return false
	}
	if idx < sv.StartIdx {
		sv.StartIdx = idx
		sv.StartIdx = max(0, sv.StartIdx)
		sv.UpdateScroll()
		return true
	} else if idx >= sv.StartIdx+sv.DispRows {
		sv.StartIdx = idx - (sv.DispRows - 1)
		sv.StartIdx = max(0, sv.StartIdx)
		sv.UpdateScroll()
		return true
	}
	return false
}

// ScrollToIdx ensures that given slice idx is visible by scrolling display as needed
func (sv *SliceViewBase) ScrollToIdx(idx int) bool {
	updt := sv.ScrollToIdxNoUpdt(idx)
	if updt {
		sv.SetNeedsLayout()
	}
	return updt
}

// SelectVal sets SelVal and attempts to find corresponding row, setting
// SelectedIdx and selecting row if found -- returns true if found, false
// otherwise.
func (sv *SliceViewBase) SelectVal(val string) bool {
	sv.SelVal = val
	if sv.SelVal != nil {
		sv.ViewMuLock()
		idx, _ := SliceIdxByValue(sv.Slice, sv.SelVal)
		sv.ViewMuUnlock()
		if idx >= 0 {
			sv.ScrollToIdx(idx)
			sv.UpdateSelectIdx(idx, true)
			return true
		}
	}
	return false
}

// SliceIdxByValue searches for first index that contains given value in slice
// -- returns false if not found
func SliceIdxByValue(slc any, fldVal any) (int, bool) {
	svnp := laser.NonPtrValue(reflect.ValueOf(slc))
	sz := svnp.Len()
	for idx := 0; idx < sz; idx++ {
		rval := laser.NonPtrValue(svnp.Index(idx))
		if rval.Interface() == fldVal {
			return idx, true
		}
	}
	return -1, false
}

/////////////////////////////////////////////////////////////////////////////
//    Moving

// MoveDown moves the selection down to next row, using given select mode
// (from keyboard modifiers) -- returns newly selected row or -1 if failed
func (sv *SliceViewBase) MoveDown(selMode events.SelectModes) int {
	if sv.SelectedIdx >= sv.SliceSize-1 {
		sv.SelectedIdx = sv.SliceSize - 1
		return -1
	}
	sv.SelectedIdx++
	sv.SelectIdxAction(sv.SelectedIdx, selMode)
	return sv.SelectedIdx
}

// MoveDownAction moves the selection down to next row, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected
// row
func (sv *SliceViewBase) MoveDownAction(selMode events.SelectModes) int {
	nidx := sv.MoveDown(selMode)
	if nidx >= 0 {
		sv.ScrollToIdx(nidx)
		sv.Send(events.Select) // todo: need to do this for the item?
	}
	return nidx
}

// MoveUp moves the selection up to previous idx, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (sv *SliceViewBase) MoveUp(selMode events.SelectModes) int {
	if sv.SelectedIdx <= 0 {
		sv.SelectedIdx = 0
		return -1
	}
	sv.SelectedIdx--
	sv.SelectIdxAction(sv.SelectedIdx, selMode)
	return sv.SelectedIdx
}

// MoveUpAction moves the selection up to previous idx, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (sv *SliceViewBase) MoveUpAction(selMode events.SelectModes) int {
	nidx := sv.MoveUp(selMode)
	if nidx >= 0 {
		sv.ScrollToIdx(nidx)
		sv.Send(events.Select)
	}
	return nidx
}

// MovePageDown moves the selection down to next page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (sv *SliceViewBase) MovePageDown(selMode events.SelectModes) int {
	if sv.SelectedIdx >= sv.SliceSize-1 {
		sv.SelectedIdx = sv.SliceSize - 1
		return -1
	}
	sv.SelectedIdx += sv.VisRows
	sv.SelectedIdx = min(sv.SelectedIdx, sv.SliceSize-1)
	sv.SelectIdxAction(sv.SelectedIdx, selMode)
	return sv.SelectedIdx
}

// MovePageDownAction moves the selection down to next page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (sv *SliceViewBase) MovePageDownAction(selMode events.SelectModes) int {
	nidx := sv.MovePageDown(selMode)
	if nidx >= 0 {
		sv.ScrollToIdx(nidx)
		sv.Send(events.Select)
	}
	return nidx
}

// MovePageUp moves the selection up to previous page, using given select mode
// (from keyboard modifiers) -- returns newly selected idx or -1 if failed
func (sv *SliceViewBase) MovePageUp(selMode events.SelectModes) int {
	if sv.SelectedIdx <= 0 {
		sv.SelectedIdx = 0
		return -1
	}
	sv.SelectedIdx -= sv.VisRows
	sv.SelectedIdx = max(0, sv.SelectedIdx)
	sv.SelectIdxAction(sv.SelectedIdx, selMode)
	return sv.SelectedIdx
}

// MovePageUpAction moves the selection up to previous page, using given select
// mode (from keyboard modifiers) -- and emits select event for newly selected idx
func (sv *SliceViewBase) MovePageUpAction(selMode events.SelectModes) int {
	nidx := sv.MovePageUp(selMode)
	if nidx >= 0 {
		sv.ScrollToIdx(nidx)
		sv.Send(events.Select)
	}
	return nidx
}

//////////////////////////////////////////////////////////////////////////////
//    Selection: user operates on the index labels

// SelectRowWidgets sets the selection state of given row of widgets
func (sv *SliceViewBase) SelectRowWidgets(row int, sel bool) {
	if row < 0 {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndRender(updt)

	sg := sv.This().(SliceViewer).SliceGrid()
	nWidgPerRow, idxOff := sv.This().(SliceViewer).RowWidgetNs()
	rowidx := row * nWidgPerRow
	if sv.ShowIndex {
		if sg.Kids.IsValidIndex(rowidx) == nil {
			widg := sg.Child(rowidx).(gi.Widget).AsWidget()
			widg.SetSelected(sel)
			widg.SetNeedsRender()
		}
	}
	if sg.Kids.IsValidIndex(rowidx+idxOff) == nil {
		widg := sg.Child(rowidx + idxOff).(gi.Widget).AsWidget()
		widg.SetSelected(sel)
		widg.SetNeedsRender()
	}
}

// SelectIdxWidgets sets the selection state of given slice index
// returns false if index is not visible
func (sv *SliceViewBase) SelectIdxWidgets(idx int, sel bool) bool {
	if !sv.IsIdxVisible(idx) {
		return false
	}
	sv.This().(SliceViewer).SelectRowWidgets(idx-sv.StartIdx, sel)
	return true
}

// UpdateSelectRow updates the selection for the given row
// callback from widgetsig select
func (sv *SliceViewBase) UpdateSelectRow(row int, sel bool) {
	idx := row + sv.StartIdx
	sv.UpdateSelectIdx(idx, sel)
}

// UpdateSelectIdx updates the selection for the given index
func (sv *SliceViewBase) UpdateSelectIdx(idx int, sel bool) {
	if sv.IsDisabled() && !sv.InactMultiSel {
		updt := sv.UpdateStart()
		defer sv.UpdateEndRender(updt)

		sv.UnselectAllIdxs()
		if sel || sv.SelectedIdx == idx {
			sv.SelectedIdx = idx
			sv.SelectIdx(idx)
		}
		sv.Send(events.Select)
	} else {
		selMode := events.SelectOne
		em := sv.EventMgr()
		if em != nil {
			selMode = em.LastSelMode
		}
		sv.SelectIdxAction(idx, selMode)
	}
}

// IdxIsSelected returns the selected status of given slice index
func (sv *SliceViewBase) IdxIsSelected(idx int) bool {
	if _, ok := sv.SelectedIdxs[idx]; ok {
		return true
	}
	return false
}

func (sv *SliceViewBase) ResetSelectedIdxs() {
	sv.SelectedIdxs = make(map[int]struct{})
}

// SelectedIdxsList returns list of selected indexes,
// sorted either ascending or descending
func (sv *SliceViewBase) SelectedIdxsList(descendingSort bool) []int {
	rws := make([]int, len(sv.SelectedIdxs))
	i := 0
	for r := range sv.SelectedIdxs {
		if r >= sv.SliceSize { // double safety check at this point
			delete(sv.SelectedIdxs, r)
			rws = rws[:len(rws)-1]
			continue
		}
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
func (sv *SliceViewBase) SelectIdx(idx int) {
	sv.SelectedIdxs[idx] = struct{}{}
	sv.SelectIdxWidgets(idx, true)
}

// UnselectIdx unselects given idx (if selected)
func (sv *SliceViewBase) UnselectIdx(idx int) {
	if sv.IdxIsSelected(idx) {
		delete(sv.SelectedIdxs, idx)
	}
	sv.SelectIdxWidgets(idx, false)
}

// UnselectAllIdxs unselects all selected idxs
func (sv *SliceViewBase) UnselectAllIdxs() {
	for r := range sv.SelectedIdxs {
		sv.SelectIdxWidgets(r, false)
	}
	sv.ResetSelectedIdxs()
}

// SelectAllIdxs selects all idxs
func (sv *SliceViewBase) SelectAllIdxs() {
	updt := sv.UpdateStart()
	defer sv.UpdateEndRender(updt)

	sv.UnselectAllIdxs()
	sv.SelectedIdxs = make(map[int]struct{}, sv.SliceSize)
	for idx := 0; idx < sv.SliceSize; idx++ {
		sv.SelectedIdxs[idx] = struct{}{}
		sv.SelectIdxWidgets(idx, true)
	}
}

// SelectIdxAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (sv *SliceViewBase) SelectIdxAction(idx int, mode events.SelectModes) {
	if mode == events.NoSelect {
		return
	}
	idx = min(idx, sv.SliceSize-1)
	if idx < 0 {
		sv.ResetSelectedIdxs()
		return
	}
	// row := idx - sv.StartIdx // note: could be out of bounds
	updt := sv.UpdateStart()
	defer sv.UpdateEndRender(updt)

	switch mode {
	case events.SelectOne:
		if sv.IdxIsSelected(idx) {
			if len(sv.SelectedIdxs) > 1 {
				sv.UnselectAllIdxs()
			}
			sv.SelectedIdx = idx
			sv.SelectIdx(idx)
			sv.IdxGrabFocus(idx)
		} else {
			sv.UnselectAllIdxs()
			sv.SelectedIdx = idx
			sv.SelectIdx(idx)
			sv.IdxGrabFocus(idx)
		}
		sv.Send(events.Select) //  sv.SelectedIdx)
	case events.ExtendContinuous:
		if len(sv.SelectedIdxs) == 0 {
			sv.SelectedIdx = idx
			sv.SelectIdx(idx)
			sv.IdxGrabFocus(idx)
			sv.Send(events.Select) //  sv.SelectedIdx)
		} else {
			minIdx := -1
			maxIdx := 0
			for r := range sv.SelectedIdxs {
				if minIdx < 0 {
					minIdx = r
				} else {
					minIdx = min(minIdx, r)
				}
				maxIdx = max(maxIdx, r)
			}
			cidx := idx
			sv.SelectedIdx = idx
			sv.SelectIdx(idx)
			if idx < minIdx {
				for cidx < minIdx {
					r := sv.MoveDown(events.SelectQuiet) // just select
					cidx = r
				}
			} else if idx > maxIdx {
				for cidx > maxIdx {
					r := sv.MoveUp(events.SelectQuiet) // just select
					cidx = r
				}
			}
			sv.IdxGrabFocus(idx)
			sv.Send(events.Select) //  sv.SelectedIdx)
		}
	case events.ExtendOne:
		if sv.IdxIsSelected(idx) {
			sv.UnselectIdxAction(idx)
			sv.Send(events.Select) //  sv.SelectedIdx)
		} else {
			sv.SelectedIdx = idx
			sv.SelectIdx(idx)
			sv.IdxGrabFocus(idx)
			sv.Send(events.Select) //  sv.SelectedIdx)
		}
	case events.Unselect:
		sv.SelectedIdx = idx
		sv.UnselectIdxAction(idx)
	case events.SelectQuiet:
		sv.SelectedIdx = idx
		sv.SelectIdx(idx)
	case events.UnselectQuiet:
		sv.SelectedIdx = idx
		sv.UnselectIdx(idx)
	}
}

// UnselectIdxAction unselects this idx (if selected) -- and emits a signal
func (sv *SliceViewBase) UnselectIdxAction(idx int) {
	if sv.IdxIsSelected(idx) {
		sv.UnselectIdx(idx)
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeDataIdx adds mimedata for given idx: an application/json of the struct
func (sv *SliceViewBase) MimeDataIdx(md *mimedata.Mimes, idx int) {
	sv.ViewMuLock()
	val := sv.SliceVal(idx)
	b, err := json.MarshalIndent(val, "", "  ")
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: filecat.DataJson, Data: b})
	} else {
		log.Printf("gi.SliceViewBase MimeData JSON Marshall error: %v\n", err)
	}
	sv.ViewMuUnlock()
}

// FromMimeData creates a slice of structs from mime data
func (sv *SliceViewBase) FromMimeData(md mimedata.Mimes) []any {
	svtyp := sv.SliceNPVal.Type()
	sl := make([]any, 0, len(md))
	for _, d := range md {
		if d.Type == filecat.DataJson {
			nval := reflect.New(svtyp.Elem()).Interface()
			err := json.Unmarshal(d.Data, nval)
			if err == nil {
				sl = append(sl, nval)
			} else {
				log.Printf("gi.SliceViewBase FromMimeData: JSON load error: %v\n", err)
			}
		}
	}
	return sl
}

// MimeDataType returns the data type for mime clipboard (copy / paste) data
// e.g., filecat.DataJson
func (sv *SliceViewBase) MimeDataType() string {
	return filecat.DataJson
}

// CopySelToMime copies selected rows to mime data
func (sv *SliceViewBase) CopySelToMime() mimedata.Mimes {
	nitms := len(sv.SelectedIdxs)
	if nitms == 0 {
		return nil
	}
	ixs := sv.SelectedIdxsList(false) // ascending
	md := make(mimedata.Mimes, 0, nitms)
	for _, i := range ixs {
		sv.MimeDataIdx(&md, i)
	}
	return md
}

// Copy copies selected rows to clip.Board, optionally resetting the selection
// satisfies gi.Clipper interface and can be overridden by subtypes
func (sv *SliceViewBase) Copy(reset bool) {
	nitms := len(sv.SelectedIdxs)
	if nitms == 0 {
		return
	}
	md := sv.This().(SliceViewer).CopySelToMime()
	if md != nil {
		sv.EventMgr().ClipBoard().Write(md)
	}
	if reset {
		sv.UnselectAllIdxs()
	}
}

// CopyIdxs copies selected idxs to clip.Board, optionally resetting the selection
func (sv *SliceViewBase) CopyIdxs(reset bool) {
	if cpr, ok := sv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Copy(reset)
	} else {
		sv.Copy(reset)
	}
}

// DeleteIdxs deletes all selected indexes
func (sv *SliceViewBase) DeleteIdxs() {
	if len(sv.SelectedIdxs) == 0 {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	ixs := sv.SelectedIdxsList(true) // descending sort
	for _, i := range ixs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.SetChanged()
}

// Cut copies selected indexes to clip.Board and deletes selected indexes
// satisfies gi.Clipper interface and can be overridden by subtypes
func (sv *SliceViewBase) Cut() {
	if len(sv.SelectedIdxs) == 0 {
		return
	}
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	sv.CopyIdxs(false)
	ixs := sv.SelectedIdxsList(true) // descending sort
	idx := ixs[0]
	sv.UnselectAllIdxs()
	for _, i := range ixs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.SetChanged()
	sv.SelectIdxAction(idx, events.SelectOne)
}

// CutIdxs copies selected indexes to clip.Board and deletes selected indexes
func (sv *SliceViewBase) CutIdxs() {
	if cpr, ok := sv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Cut()
	} else {
		sv.Cut()
	}
}

// Paste pastes clipboard at CurIdx
// satisfies gi.Clipper interface and can be overridden by subtypes
func (sv *SliceViewBase) Paste() {
	dt := sv.This().(SliceViewer).MimeDataType()
	md := sv.EventMgr().ClipBoard().Read([]string{dt})
	if md != nil {
		sv.PasteMenu(md, sv.CurIdx)
	}
}

// PasteIdx pastes clipboard at given idx
func (sv *SliceViewBase) PasteIdx(idx int) {
	sv.CurIdx = idx
	if cpr, ok := sv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Paste()
	} else {
		sv.Paste()
	}
}

// MakePasteMenu makes the menu of options for paste events
func (sv *SliceViewBase) MakePasteMenu(m *gi.Menu, data any, idx int) {
	if len(*m) > 0 {
		return
	}
	m.AddButton(gi.ActOpts{Label: "Assign To", Data: data}, func(act *gi.Button) {
		sv.This().(SliceViewer).PasteAssign(data.(mimedata.Mimes), idx)
	})
	m.AddButton(gi.ActOpts{Label: "Insert Before", Data: data}, func(act *gi.Button) {
		sv.This().(SliceViewer).PasteAtIdx(data.(mimedata.Mimes), idx)
	})
	m.AddButton(gi.ActOpts{Label: "Insert After", Data: data}, func(act *gi.Button) {
		sv.This().(SliceViewer).PasteAtIdx(data.(mimedata.Mimes), idx+1)
	})
	m.AddButton(gi.ActOpts{Label: "Cancel", Data: data}, func(act *gi.Button) {
	})
}

// PasteMenu performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (sv *SliceViewBase) PasteMenu(md mimedata.Mimes, idx int) {
	sv.UnselectAllIdxs()
	var menu gi.Menu
	sv.MakePasteMenu(&menu, md, idx)
	pos := sv.IdxPos(idx)
	gi.NewMenu(menu, sv.This().(gi.Widget), pos).Run()
}

// PasteAssign assigns mime data (only the first one!) to this idx
func (sv *SliceViewBase) PasteAssign(md mimedata.Mimes, idx int) {
	sl := sv.FromMimeData(md)
	if len(sl) == 0 {
		return
	}
	updt := sv.UpdateStart()
	ns := sl[0]
	sv.SliceNPVal.Index(idx).Set(reflect.ValueOf(ns).Elem())
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetChanged()
	sv.UpdateEndRender(updt)
}

// PasteAtIdx inserts object(s) from mime data at (before) given slice index
func (sv *SliceViewBase) PasteAtIdx(md mimedata.Mimes, idx int) {
	sl := sv.FromMimeData(md)
	if len(sl) == 0 {
		return
	}
	svl := reflect.ValueOf(sv.Slice)
	svnp := sv.SliceNPVal
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	for _, ns := range sl {
		sz := svnp.Len()
		svnp = reflect.Append(svnp, reflect.ValueOf(ns).Elem())
		svl.Elem().Set(svnp)
		if idx >= 0 && idx < sz {
			reflect.Copy(svnp.Slice(idx+1, sz+1), svnp.Slice(idx, sz))
			svnp.Index(idx).Set(reflect.ValueOf(ns).Elem())
			svl.Elem().Set(svnp)
		}
		idx++
	}

	sv.SliceNPVal = laser.NonPtrValue(reflect.ValueOf(sv.Slice)) // need to update after changes

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetChanged()
	sv.SelectIdxAction(idx, events.SelectOne)
}

// Duplicate copies selected items and inserts them after current selection --
// return idx of start of duplicates if successful, else -1
func (sv *SliceViewBase) Duplicate() int {
	nitms := len(sv.SelectedIdxs)
	if nitms == 0 {
		return -1
	}
	ixs := sv.SelectedIdxsList(true) // descending sort -- last first
	pasteAt := ixs[0]
	sv.CopyIdxs(true)
	dt := sv.This().(SliceViewer).MimeDataType()
	md := sv.EventMgr().ClipBoard().Read([]string{dt})
	sv.This().(SliceViewer).PasteAtIdx(md, pasteAt)
	return pasteAt
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragNDropStart starts a drag-n-drop
func (sv *SliceViewBase) DragNDropStart() {
	nitms := len(sv.SelectedIdxs)
	if nitms == 0 {
		return
	}
	md := sv.This().(SliceViewer).CopySelToMime()
	_ = md
	ixs := sv.SelectedIdxsList(false) // ascending
	widg, ok := sv.This().(SliceViewer).RowFirstWidget(ixs[0])
	if ok {
		sp := &gi.Sprite{}
		sp.GrabRenderFrom(widg)
		gi.ImageClearer(sp.Pixels, 50.0)
		// todo:
		// sv.ParentRenderWin().StartDragNDrop(sv.This(), md, sp)
	}
}

// DragNDropTarget handles a drag-n-drop drop
func (sv *SliceViewBase) DragNDropTarget(de events.Event) {
	// de.Target = sv.This()
	// if de.Mod == events.DropLink {
	// 	de.Mod = events.DropCopy // link not supported -- revert to copy
	// }
	// idx, ok := sv.IdxFromPos(de.LocalPos().Y)
	// if ok {
	// 	de.SetHandled()
	// 	sv.CurIdx = idx
	// 	if dpr, ok := sv.This().(gi.DragNDropper); ok {
	// 		dpr.Drop(de.Data, de.Mod)
	// 	} else {
	// 		sv.Drop(de.Data, de.Mod)
	// 	}
	// }
}

// MakeDropMenu makes the menu of options for dropping on a target
func (sv *SliceViewBase) MakeDropMenu(m *gi.Menu, data any, mod events.DropMods, idx int) {
	if len(*m) > 0 {
		return
	}
	switch mod {
	case events.DropCopy:
		m.AddLabel("Copy (Shift=Move):")
	case events.DropMove:
		m.AddLabel("Move:")
	}
	if mod == events.DropCopy {
		m.AddButton(gi.ActOpts{Label: "Assign To", Data: data}, func(act *gi.Button) {
			sv.DropAssign(data.(mimedata.Mimes), idx)
		})
	}
	m.AddButton(gi.ActOpts{Label: "Insert Before", Data: data}, func(act *gi.Button) {
		sv.DropBefore(data.(mimedata.Mimes), mod, idx) // captures mod
	})
	m.AddButton(gi.ActOpts{Label: "Insert After", Data: data}, func(act *gi.Button) {
		sv.DropAfter(data.(mimedata.Mimes), mod, idx) // captures mod
	})
	m.AddButton(gi.ActOpts{Label: "Cancel", Data: data}, func(act *gi.Button) {
		sv.DropCancel()
	})
}

// Drop pops up a menu to determine what specifically to do with dropped items
// this satisfies gi.DragNDropper interface, and can be overwritten in subtypes
func (sv *SliceViewBase) Drop(md mimedata.Mimes, mod events.DropMods) {
	var menu gi.Menu
	sv.MakeDropMenu(&menu, md, mod, sv.CurIdx)
	pos := sv.IdxPos(sv.CurIdx)
	gi.NewMenu(menu, sv.This().(gi.Widget), pos).Run()
}

// DropAssign assigns mime data (only the first one!) to this node
func (sv *SliceViewBase) DropAssign(md mimedata.Mimes, idx int) {
	sv.DraggedIdxs = nil
	sv.This().(SliceViewer).PasteAssign(md, idx)
	sv.DragNDropFinalize(events.DropCopy)
}

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore -- ends up calling DragNDropSource if us..
func (sv *SliceViewBase) DragNDropFinalize(mod events.DropMods) {
	sv.UnselectAllIdxs()
	// sv.ParentRenderWin().FinalizeDragNDrop(mod)
}

// DragNDropSource is called after target accepts the drop -- we just remove
// elements that were moved
func (sv *SliceViewBase) DragNDropSource(de events.Event) {
	// if de.Mod != events.DropMove || len(sv.DraggedIdxs) == 0 {
	// 	return
	// }

	updt := sv.UpdateStart()
	defer sv.UpdateEnd(updt)

	sort.Slice(sv.DraggedIdxs, func(i, j int) bool {
		return sv.DraggedIdxs[i] > sv.DraggedIdxs[j]
	})
	idx := sv.DraggedIdxs[0]
	for _, i := range sv.DraggedIdxs {
		sv.This().(SliceViewer).SliceDeleteAt(i)
	}
	sv.DraggedIdxs = nil
	sv.SelectIdxAction(idx, events.SelectOne)
}

// SaveDraggedIdxs saves selectedindexes into dragged indexes
// taking into account insertion at idx
func (sv *SliceViewBase) SaveDraggedIdxs(idx int) {
	sz := len(sv.SelectedIdxs)
	if sz == 0 {
		sv.DraggedIdxs = nil
		return
	}
	ixs := sv.SelectedIdxsList(false) // ascending
	sv.DraggedIdxs = make([]int, len(ixs))
	for i, ix := range ixs {
		if ix > idx {
			sv.DraggedIdxs[i] = ix + sz // make room for insertion
		} else {
			sv.DraggedIdxs[i] = ix
		}
	}
}

// DropBefore inserts object(s) from mime data before this node
func (sv *SliceViewBase) DropBefore(md mimedata.Mimes, mod events.DropMods, idx int) {
	sv.SaveDraggedIdxs(idx)
	sv.This().(SliceViewer).PasteAtIdx(md, idx)
	sv.DragNDropFinalize(mod)
}

// DropAfter inserts object(s) from mime data after this node
func (sv *SliceViewBase) DropAfter(md mimedata.Mimes, mod events.DropMods, idx int) {
	sv.SaveDraggedIdxs(idx + 1)
	sv.This().(SliceViewer).PasteAtIdx(md, idx+1)
	sv.DragNDropFinalize(mod)
}

// DropCancel cancels the drop action e.g., preventing deleting of source
// items in a Move case
func (sv *SliceViewBase) DropCancel() {
	sv.DragNDropFinalize(events.DropIgnore)
}

//////////////////////////////////////////////////////////////////////////////
//    Events

func (sv *SliceViewBase) StdCtxtMenu(m *gi.Menu, idx int) {
	if sv.isArray {
		return
	}
	m.AddButton(gi.ActOpts{Label: "Copy", Data: idx}, func(act *gi.Button) {
		sv.CopyIdxs(true)
	})
	m.AddButton(gi.ActOpts{Label: "Cut", Data: idx}, func(act *gi.Button) {
		sv.CutIdxs()
	})
	m.AddButton(gi.ActOpts{Label: "Paste", Data: idx}, func(act *gi.Button) {
		sv.PasteIdx(idx)
	})
	m.AddButton(gi.ActOpts{Label: "Duplicate", Data: idx}, func(act *gi.Button) {
		sv.Duplicate()
	})
}

func (sv *SliceViewBase) ItemCtxtMenu(idx int) {
	val := sv.SliceVal(idx)
	if val == nil {
		return
	}
	var menu gi.Menu

	// todo: CtxtMenuView
	// if CtxtMenuView(val, sv.IsDisabled(), sv.Sc, &menu) {
	// 	if sv.ShowViewCtxtMenu {
	// 		menu.AddSeparator("sep-svmenu")
	// 		sv.This().(SliceViewer).StdCtxtMenu(&menu, idx)
	// 	}
	// } else {
	sv.This().(SliceViewer).StdCtxtMenu(&menu, idx)
	// }
	if len(menu) > 0 {
		pos := sv.IdxPos(idx)
		// if pos == (image.Point{}) {
		// 	em := sv.EventMgr()
		// 	if em != nil {
		// 		pos = em.LastMousePos
		// 	}
		// }
		gi.NewMenu(menu, sv.This().(gi.Widget), pos).Run()
	}
}

// KeyInputNav supports multiple selection navigation keys
func (sv *SliceViewBase) KeyInputNav(kt events.Event) {
	kf := gi.KeyFun(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())
	if selMode == events.SelectOne {
		if sv.SelectMode {
			selMode = events.ExtendContinuous
		}
	}
	switch kf {
	case gi.KeyFunCancelSelect:
		sv.UnselectAllIdxs()
		sv.SelectMode = false
		kt.SetHandled()
	case gi.KeyFunMoveDown:
		sv.MoveDownAction(selMode)
		kt.SetHandled()
	case gi.KeyFunMoveUp:
		sv.MoveUpAction(selMode)
		kt.SetHandled()
	case gi.KeyFunPageDown:
		sv.MovePageDownAction(selMode)
		kt.SetHandled()
	case gi.KeyFunPageUp:
		sv.MovePageUpAction(selMode)
		kt.SetHandled()
	case gi.KeyFunSelectMode:
		sv.SelectMode = !sv.SelectMode
		kt.SetHandled()
	case gi.KeyFunSelectAll:
		sv.SelectAllIdxs()
		sv.SelectMode = false
		kt.SetHandled()
	}
}

func (sv *SliceViewBase) KeyInputActive(kt events.Event) {
	if gi.KeyEventTrace {
		fmt.Printf("SliceViewBase KeyInput: %v\n", sv.Path())
	}
	sv.KeyInputNav(kt)
	if kt.IsHandled() {
		return
	}
	idx := sv.SelectedIdx
	kf := gi.KeyFun(kt.KeyChord())
	switch kf {
	// case gi.KeyFunDelete: // too dangerous
	// 	sv.This().(SliceViewer).SliceDeleteAt(sv.SelectedIdx)
	// 	sv.SelectMode = false
	// 	sv.SelectIdxAction(idx, events.SelectOne)
	// 	kt.SetHandled()
	case gi.KeyFunDuplicate:
		nidx := sv.Duplicate()
		sv.SelectMode = false
		if nidx >= 0 {
			sv.SelectIdxAction(nidx, events.SelectOne)
		}
		kt.SetHandled()
	case gi.KeyFunInsert:
		sv.This().(SliceViewer).SliceNewAt(idx)
		sv.SelectMode = false
		sv.SelectIdxAction(idx+1, events.SelectOne) // todo: somehow nidx not working
		kt.SetHandled()
	case gi.KeyFunInsertAfter:
		sv.This().(SliceViewer).SliceNewAt(idx + 1)
		sv.SelectMode = false
		sv.SelectIdxAction(idx+1, events.SelectOne)
		kt.SetHandled()
	case gi.KeyFunCopy:
		sv.CopyIdxs(true)
		sv.SelectMode = false
		sv.SelectIdxAction(idx, events.SelectOne)
		kt.SetHandled()
	case gi.KeyFunCut:
		sv.CutIdxs()
		sv.SelectMode = false
		kt.SetHandled()
	case gi.KeyFunPaste:
		sv.PasteIdx(sv.SelectedIdx)
		sv.SelectMode = false
		kt.SetHandled()
	}
}

func (sv *SliceViewBase) KeyInputInactive(kt events.Event) {
	if gi.KeyEventTrace {
		fmt.Printf("SliceViewBase Inactive KeyInput: %v\n", sv.Path())
	}
	if sv.InactMultiSel {
		sv.KeyInputNav(kt)
		if kt.IsHandled() {
			return
		}
	}
	kf := gi.KeyFun(kt.KeyChord())
	idx := sv.SelectedIdx
	switch {
	case kf == gi.KeyFunMoveDown:
		ni := idx + 1
		if ni < sv.SliceSize {
			sv.ScrollToIdx(ni)
			sv.UpdateSelectIdx(ni, true)
			kt.SetHandled()
		}
	case kf == gi.KeyFunMoveUp:
		ni := idx - 1
		if ni >= 0 {
			sv.ScrollToIdx(ni)
			sv.UpdateSelectIdx(ni, true)
			kt.SetHandled()
		}
	case kf == gi.KeyFunPageDown:
		ni := min(idx+sv.VisRows-1, sv.SliceSize-1)
		sv.ScrollToIdx(ni)
		sv.UpdateSelectIdx(ni, true)
		kt.SetHandled()
	case kf == gi.KeyFunPageUp:
		ni := max(idx-(sv.VisRows-1), 0)
		sv.ScrollToIdx(ni)
		sv.UpdateSelectIdx(ni, true)
		kt.SetHandled()
	case kf == gi.KeyFunEnter || kf == gi.KeyFunAccept || kt.KeyRune() == ' ':
		sv.Send(events.DoubleClick, kt)
		kt.SetHandled()
	}
}

func (sv *SliceViewBase) HandleSliceViewEvents() {
	// LowPri to allow other focal widgets to capture
	sv.On(events.Scroll, func(e events.Event) {
		e.SetHandled()
		se := e.(*events.MouseScroll)
		sbb := sv.This().(SliceViewer).ScrollBar()
		cur := float32(sbb.Pos)
		sbb.SetSliderPosAction(cur - float32(se.DimDelta(mat32.Y)))
	})
	sv.OnDoubleClick(func(e events.Event) {
		si := sv.SelectedIdx
		sv.UnselectAllIdxs()
		sv.SelectIdx(si)
		sv.Send(events.DoubleClick, e)
		e.SetHandled()
	})
	// todo ctxmenu
	// sv.Onwe.AddFunc(events.MouseUp, gi.LowRawPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	me := d.(events.Event)
	// 	svv := recv.Embed(TypeSliceViewBase).(*SliceViewBase)
	// 	// if !svv.StateIs(states.Focused) {
	// 	// 	svv.GrabFocus()
	// 	// }
	// 	if me.Button == events.Right && me.Action == events.Release {
	// 		svv.This().(SliceViewer).ItemCtxtMenu(svv.SelectedIdx)
	// 		me.SetHandled()
	// 	}
	// })
	if sv.IsDisabled() {
		if sv.InactKeyNav {
			sv.OnKeyChord(func(e events.Event) {
				sv.KeyInputInactive(e)
			})
		}
	} else {
		sv.OnKeyChord(func(e events.Event) {
			sv.KeyInputActive(e)
		})
		// svwe.AddFunc(goosi.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		// 	de := d.(events.Event)
		// 	svv := recv.Embed(TypeSliceViewBase).(*SliceViewBase)
		// 	switch de.Action {
		// 	case events.Start:
		// 		svv.DragNDropStart()
		// 	case events.DropOnTarget:
		// 		svv.DragNDropTarget(de)
		// 	case events.DropFmSource:
		// 		svv.DragNDropSource(de)
		// 	}
		// })
		// sg := sv.This().(SliceViewer).SliceGrid()
		// if sg != nil {
		// 	sgwe.AddFunc(goosi.DNDFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		// 		de := d.(*events.FocusEvent)
		// 		sgg := recv.Embed(gi.FrameType).(*gi.Frame)
		// 		switch de.Action {
		// 		case events.Enter:
		// 			sgg.ParentRenderWin().DNDSetCursor(de.Mod)
		// 		case events.Exit:
		// 			sgg.ParentRenderWin().DNDNotCursor()
		// 		case events.Hover:
		// 			// nothing here?
		// 		}
		// 	})
		// }
	}
}
