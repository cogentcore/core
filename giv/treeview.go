// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"
	"reflect"

	"github.com/chewxy/math32"
	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
)

////////////////////////////////////////////////////////////////////////////////////////
//  TreeView -- a widget that graphically represents / manipulates a Ki Tree

// TreeView provides a graphical representation of source tree structure
// (which can be any type of Ki nodes), providing full manipulation abilities
// of that source tree (move, cut, add, etc) through drag-n-drop and
// cut/copy/paste and menu actions.
type TreeView struct {
	gi.PartsWidgetBase
	SrcNode          ki.Ptr                    `desc:"Ki Node that this widget is viewing in the tree -- the source"`
	ShowViewCtxtMenu bool                      `desc:"if the object we're viewing has its own CtxtMenu property defined, should we also still show the view's own context menu?"`
	ViewIdx          int                       `desc:"linear index of this node within the entire tree -- updated on full rebuilds and may sometimes be off, but close enough for expected uses"`
	Indent           units.Value               `xml:"indent" desc:"styled amount to indent children relative to this node"`
	TreeViewSig      ki.Signal                 `json:"-" xml:"-" desc:"signal for TreeView -- all are emitted from the root tree view widget, with data = affected node -- see TreeViewSignals for the types"`
	StateStyles      [TreeViewStatesN]gi.Style `json:"-" xml:"-" desc:"styles for different states of the widget -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	WidgetSize       gi.Vec2D                  `desc:"just the size of our widget -- our alloc includes all of our children, but we only draw us"`
	Icon             gi.IconName               `json:"-" xml:"icon" view:"show-name" desc:"optional icon, displayed to the the left of the text label"`
	RootView         *TreeView                 `json:"-" xml:"-" desc:"cached root of the view"`
}

var KiT_TreeView = kit.Types.AddType(&TreeView{}, nil)

func init() {
	kit.Types.SetProps(KiT_TreeView, TreeViewProps)
}

//////////////////////////////////////////////////////////////////////////////
//    End-User API

// SetRootNode sets the root view to the root of the source node that we are
// viewing, and builds-out the view of its tree
func (tv *TreeView) SetRootNode(sk ki.Ki) {
	updt := false
	if tv.SrcNode.Ptr != sk {
		updt = tv.UpdateStart()
		tv.SrcNode.Ptr = sk
		sk.NodeSignal().Connect(tv.This(), SrcNodeSignal) // we recv signals from source
	}
	tv.RootView = tv
	tvIdx := 0
	tv.SyncToSrc(&tvIdx)
	tv.UpdateEnd(updt)
}

// SetSrcNode sets the source node that we are viewing, and builds-out the view of its tree
func (tv *TreeView) SetSrcNode(sk ki.Ki, tvIdx *int) {
	updt := false
	if tv.SrcNode.Ptr != sk {
		updt = tv.UpdateStart()
		tv.SrcNode.Ptr = sk
		sk.NodeSignal().Connect(tv.This(), SrcNodeSignal) // we recv signals from source
	}
	tv.SyncToSrc(tvIdx)
	tv.UpdateEnd(updt)
}

// SyncToSrc updates the view tree to match the source tree, using
// ConfigChildren to maximally preserve existing tree elements
func (tv *TreeView) SyncToSrc(tvIdx *int) {
	pr := prof.Start("TreeView.SyncToSrc")
	sk := tv.SrcNode.Ptr
	nm := "tv_" + sk.UniqueName()
	tv.SetNameRaw(nm) // guaranteed to be unique
	tv.SetUniqueName(nm)
	tv.ViewIdx = *tvIdx
	(*tvIdx)++
	tvPar := tv.TreeViewParent()
	if tvPar != nil {
		tv.RootView = tvPar.RootView
	}
	vcprop := "view-closed"
	skids := *sk.Children()
	tnl := make(kit.TypeAndNameList, 0, len(skids))
	typ := tv.This().Type() // always make our type
	flds := make([]ki.Ki, 0)
	fldClosed := make([]bool, 0)
	sk.FuncFields(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		flds = append(flds, k)
		tnl.Add(typ, "tv_"+k.Name())
		ft := sk.FieldTag(k.Name(), vcprop)
		cls := false
		if vc, ok := kit.ToBool(ft); ok && vc {
			cls = true
		} else {
			if vcp, ok := k.PropInherit(vcprop, false, true); ok {
				if vc, ok := kit.ToBool(vcp); vc && ok {
					cls = true
				}
			}
		}
		fldClosed = append(fldClosed, cls)
		return true
	})
	for _, skid := range skids {
		tnl.Add(typ, "tv_"+skid.UniqueName())
	}
	mods, updt := tv.ConfigChildren(tnl, false)
	if mods {
		tv.SetFullReRender()
		// fmt.Printf("got mod on %v\n", tv.PathUnique())
	}
	idx := 0
	for i, fld := range flds {
		vk := tv.Kids[idx].Embed(KiT_TreeView).(*TreeView)
		vk.SetSrcNode(fld, tvIdx)
		if mods {
			vk.SetClosedState(fldClosed[i])
		}
		idx++
	}
	for _, skid := range *sk.Children() {
		vk := tv.Kids[idx].Embed(KiT_TreeView).(*TreeView)
		vk.SetSrcNode(skid, tvIdx)
		if mods {
			if vcp, ok := skid.PropInherit(vcprop, false, true); ok {
				if vc, ok := kit.ToBool(vcp); vc && ok {
					vk.SetClosed()
				}
			}
		}
		idx++
	}
	if !sk.HasChildren() {
		tv.SetClosed()
	}
	tv.UpdateEnd(updt)
	pr.End()
}

// SrcNodeSignal is the function for receiving node signals from our SrcNode
func SrcNodeSignal(tvki, send ki.Ki, sig int64, data interface{}) {
	tv := tvki.Embed(KiT_TreeView).(*TreeView)
	if data != nil {
		dflags := data.(int64)
		if gi.Update2DTrace {
			fmt.Printf("treeview: %v got signal: %v from node: %v  data: %v  flags %v\n", tv.PathUnique(), ki.NodeSignals(sig), send.PathUnique(), kit.BitFlagsToString(dflags, ki.FlagsN), kit.BitFlagsToString(send.Flags(), ki.FlagsN))
		}
		if bitflag.HasAnyMask(dflags, int64(ki.StruUpdateFlagsMask)) {
			tvIdx := tv.ViewIdx
			tv.SyncToSrc(&tvIdx)
		} else if bitflag.HasAnyMask(dflags, int64(ki.ValUpdateFlagsMask)) {
			tv.UpdateSig()
		}
	}
}

// IsClosed returns whether this node itself closed?
func (tv *TreeView) IsClosed() bool {
	return tv.HasFlag(int(TreeViewFlagClosed))
}

// SetClosed sets the closed flag for this node -- call Close() method to
// close a node and update view
func (tv *TreeView) SetClosed() {
	tv.SetFlag(int(TreeViewFlagClosed))
}

// SetOpen clears the closed flag for this node -- call Open() method to open
// a node and update view
func (tv *TreeView) SetOpen() {
	tv.ClearFlag(int(TreeViewFlagClosed))
}

// SetClosedState sets the closed state based on arg
func (tv *TreeView) SetClosedState(closed bool) {
	tv.SetFlagState(closed, int(TreeViewFlagClosed))
}

// IsChanged returns whether this node has the changed flag set?  Only updated
// on the root note by GUI actions.
func (tv *TreeView) IsChanged() bool {
	return tv.HasFlag(int(TreeViewFlagChanged))
}

// SetChanged is called whenever a gui action updates the tree -- sets Changed
// flag on root node and emits signal
func (tv *TreeView) SetChanged() {
	if tv.RootView == nil {
		return
	}
	tv.RootView.SetFlag(int(TreeViewFlagChanged))
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewChanged), tv.This())
}

// HasClosedParent returns whether this node have a closed parent? if so, don't render!
func (tv *TreeView) HasClosedParent() bool {
	pcol := false
	tv.FuncUpParent(0, tv.This(), func(k ki.Ki, level int, d interface{}) bool {
		_, pg := gi.KiToNode2D(k)
		if pg == nil {
			return false
		}
		if pg.TypeEmbeds(KiT_TreeView) {
			// nw := pg.Embed(KiT_TreeView).(*TreeView)
			if pg.HasFlag(int(TreeViewFlagClosed)) {
				pcol = true
				return false
			}
		}
		return true
	})
	return pcol
}

// Label returns the display label for this node, satisfying the Labeler interface
func (tv *TreeView) Label() string {
	return tv.SrcNode.Ptr.Name()
}

//////////////////////////////////////////////////////////////////////////////
//    Signals etc

// TreeViewSignals are signals that treeview can send -- these are all sent
// from the root tree view widget node, with data being the relevant node
// widget
type TreeViewSignals int64

const (
	// node was selected
	TreeViewSelected TreeViewSignals = iota

	// TreeView unselected
	TreeViewUnselected

	// TreeView all items were selected
	TreeViewAllSelected

	// TreeView all items were unselected
	TreeViewAllUnselected

	// closed TreeView was opened
	TreeViewOpened

	// open TreeView was closed -- children not visible
	TreeViewClosed

	// TreeViewChanged means that some kind of edit operation has taken place
	// by the user via the gui -- we don't track the details, just that
	// changes have happened
	TreeViewChanged

	TreeViewSignalsN
)

//go:generate stringer -type=TreeViewSignals

// these extend NodeBase NodeFlags to hold TreeView state
const (
	// TreeViewFlagClosed means node is toggled closed (children not visible)
	TreeViewFlagClosed gi.NodeFlags = gi.NodeFlagsN + iota

	// TreeViewFlagChanged is updated on the root node whenever a gui edit is
	// made through the tree view on the tree -- this does not track any other
	// changes that might have occurred in the tree itself.  Also emits a TreeVi
	TreeViewFlagChanged
)

// TreeViewStates are mutually-exclusive tree view states -- determines appearance
type TreeViewStates int32

const (
	// normal state -- there but not being interacted with
	TreeViewActive TreeViewStates = iota

	// selected
	TreeViewSel

	// in focus -- will respond to keyboard input
	TreeViewFocus

	TreeViewStatesN
)

//go:generate stringer -type=TreeViewStates

// TreeViewSelectors are Style selector names for the different states:
var TreeViewSelectors = []string{":active", ":selected", ":focus"}

// These are special properties established on the RootView for maintaining
// overall tree state
const (
	// TreeViewSelProp is a slice of tree views that are currently selected
	// -- much more efficient to update the list rather than regenerate it,
	// especially for a large tree
	TreeViewSelProp = "__SelectedList"

	// TreeViewSelModeProp is a bool that, if true, automatically selects nodes
	// when nodes are moved to via keyboard actions
	TreeViewSelModeProp = "__SelectMode"
)

//////////////////////////////////////////////////////////////////////////////
//    Selection

// SelectMode returns true if keyboard movements should automatically select nodes
func (tv *TreeView) SelectMode() bool {
	smp, ok := tv.RootView.Prop(TreeViewSelModeProp)
	if !ok {
		tv.SetSelectMode(false)
		return false
	} else {
		return smp.(bool)
	}
}

// SetSelectMode updates the select mode
func (tv *TreeView) SetSelectMode(selMode bool) {
	tv.RootView.SetProp(TreeViewSelModeProp, selMode)
}

// SelectModeToggle toggles the SelectMode
func (tv *TreeView) SelectModeToggle() {
	if tv.SelectMode() {
		tv.SetSelectMode(false)
	} else {
		tv.SetSelectMode(true)
	}
}

// SelectedViews returns a slice of the currently-selected TreeViews within
// the entire tree, using a list maintained by the root node
func (tv *TreeView) SelectedViews() []*TreeView {
	if tv.RootView == nil {
		return nil
	}
	var sl []*TreeView
	slp, ok := tv.RootView.Prop(TreeViewSelProp)
	if !ok {
		sl = make([]*TreeView, 0)
		tv.SetSelectedViews(sl)
	} else {
		sl = slp.([]*TreeView)
	}
	return sl
}

// SetSelectedViews updates the selected views to given list
func (tv *TreeView) SetSelectedViews(sl []*TreeView) {
	if tv.RootView != nil {
		tv.RootView.SetProp(TreeViewSelProp, sl)
	}
}

// SelectedSrcNodes returns a slice of the currently-selected source nodes
// in the entire tree view
func (tv *TreeView) SelectedSrcNodes() ki.Slice {
	sn := make(ki.Slice, 0)
	sl := tv.SelectedViews()
	for _, v := range sl {
		sn = append(sn, v.SrcNode.Ptr)
	}
	return sn
}

// Select selects this node (if not already selected) -- must use this method
// to update global selection list
func (tv *TreeView) Select() {
	if !tv.IsSelected() {
		tv.SetSelected()
		sl := tv.SelectedViews()
		sl = append(sl, tv)
		tv.SetSelectedViews(sl)
		tv.UpdateSig()
	}
}

// Unselect unselects this node (if selected) -- must use this method
// to update global selection list
func (tv *TreeView) Unselect() {
	if tv.IsSelected() {
		tv.ClearSelected()
		sl := tv.SelectedViews()
		sz := len(sl)
		for i := 0; i < sz; i++ {
			if sl[i] == tv {
				sl = append(sl[:i], sl[i+1:]...)
				break
			}
		}
		tv.SetSelectedViews(sl)
		tv.UpdateSig()
	}
}

// UnselectAll unselects all selected items in the view
func (tv *TreeView) UnselectAll() {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	sl := tv.SelectedViews()
	tv.SetSelectedViews(nil) // clear in advance
	for _, v := range sl {
		v.ClearSelected()
		v.UpdateSig()
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewAllUnselected), tv.This())
}

// SelectAll all items in view
func (tv *TreeView) SelectAll() {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	tv.UnselectAll()
	nn := tv.RootView
	nn.Select()
	for nn != nil {
		nn = nn.MoveDown(mouse.SelectModesN) // just select
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewAllSelected), tv.This())
}

// SelectUpdate updates selection to include this node, using selectmode
// from mouse event (ExtendContinuous, ExtendOne).  Returns true if this node selected
func (tv *TreeView) SelectUpdate(mode mouse.SelectModes) bool {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	sel := false
	switch mode {
	case mouse.ExtendContinuous:
		sl := tv.SelectedViews()
		if len(sl) == 0 {
			tv.Select()
			tv.GrabFocus()
			sel = true
		} else {
			minIdx := -1
			maxIdx := 0
			for _, v := range sl {
				if minIdx < 0 {
					minIdx = v.ViewIdx
				} else {
					minIdx = ints.MinInt(minIdx, v.ViewIdx)
				}
				maxIdx = ints.MaxInt(maxIdx, v.ViewIdx)
			}
			cidx := tv.ViewIdx
			nn := tv
			tv.Select()
			if tv.ViewIdx < minIdx {
				for cidx < minIdx {
					nn = nn.MoveDown(mouse.SelectModesN) // just select
					cidx = nn.ViewIdx
				}
			} else if tv.ViewIdx > maxIdx {
				for cidx > maxIdx {
					nn = nn.MoveUp(mouse.SelectModesN) // just select
					cidx = nn.ViewIdx
				}
			}
		}
	case mouse.ExtendOne:
		if tv.IsSelected() {
			tv.UnselectAction()
		} else {
			tv.Select()
			tv.GrabFocus()
			sel = true
		}
	case mouse.NoSelectMode:
		if tv.IsSelected() {
			sl := tv.SelectedViews()
			if len(sl) > 1 {
				tv.UnselectAll()
				tv.Select()
				tv.GrabFocus()
				sel = true
			}
		} else {
			tv.UnselectAll()
			tv.Select()
			tv.GrabFocus()
			sel = true
		}
	default: // anything else
		tv.Select()
		// not sel -- no signal..
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
	return sel
}

// SelectAction updates selection to include this node, using selectmode
// from mouse event (ExtendContinuous, ExtendOne), and emits selection signal
// returns true if signal emitted
func (tv *TreeView) SelectAction(mode mouse.SelectModes) bool {
	sel := tv.SelectUpdate(mode)
	if sel {
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), tv.This())
	}
	return sel
}

// UnselectAction unselects this node (if selected) -- and emits a signal
func (tv *TreeView) UnselectAction() {
	if tv.IsSelected() {
		tv.Unselect()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewUnselected), tv.This())
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Moving

// MoveDown moves the selection down to next element in the tree, using given
// select mode (from keyboard modifiers) -- returns newly selected node
func (tv *TreeView) MoveDown(selMode mouse.SelectModes) *TreeView {
	if tv.Par == nil {
		return nil
	}
	if selMode == mouse.NoSelectMode {
		if tv.SelectMode() {
			selMode = mouse.ExtendContinuous
		}
	}
	if tv.IsClosed() || !tv.HasChildren() { // next sibling
		return tv.MoveDownSibling(selMode)
	} else {
		if tv.HasChildren() {
			nn := tv.KnownChild(0).Embed(KiT_TreeView).(*TreeView)
			if nn != nil {
				nn.SelectUpdate(selMode)
				return nn
			}
		}
	}
	return nil
}

// MoveDownAction moves the selection down to next element in the tree, using given
// select mode (from keyboard modifiers) -- and emits select event for newly selected item
func (tv *TreeView) MoveDownAction(selMode mouse.SelectModes) *TreeView {
	nn := tv.MoveDown(selMode)
	if nn != nil && nn != tv {
		nn.GrabFocus()
		nn.ScrollToMe()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), nn.This())
	}
	return nn
}

// MoveDownSibling moves down only to siblings, not down into children, using
// given select mode (from keyboard modifiers)
func (tv *TreeView) MoveDownSibling(selMode mouse.SelectModes) *TreeView {
	if tv.Par == nil {
		return nil
	}
	if tv == tv.RootView {
		return nil
	}
	myidx, ok := tv.IndexInParent()
	if ok && myidx < len(*tv.Par.Children())-1 {
		nn := tv.Par.KnownChild(myidx + 1).Embed(KiT_TreeView).(*TreeView)
		if nn != nil {
			nn.SelectUpdate(selMode)
			return nn
		}
	} else {
		return tv.Par.Embed(KiT_TreeView).(*TreeView).MoveDownSibling(selMode) // try up
	}
	return nil
}

// MoveUp moves selection up to previous element in the tree, using given
// select mode (from keyboard modifiers) -- returns newly selected node
func (tv *TreeView) MoveUp(selMode mouse.SelectModes) *TreeView {
	if tv.Par == nil || tv == tv.RootView {
		return nil
	}
	if selMode == mouse.NoSelectMode {
		if tv.SelectMode() {
			selMode = mouse.ExtendContinuous
		}
	}
	myidx, ok := tv.IndexInParent()
	if ok && myidx > 0 {
		nn := tv.Par.KnownChild(myidx - 1).Embed(KiT_TreeView).(*TreeView)
		if nn != nil {
			return nn.MoveToLastChild(selMode)
		}
	} else {
		if tv.Par != nil {
			nn := tv.Par.Embed(KiT_TreeView).(*TreeView)
			if nn != nil {
				nn.SelectUpdate(selMode)
				return nn
			}
		}
	}
	return nil
}

// MoveUpAction moves the selection up to previous element in the tree, using given
// select mode (from keyboard modifiers) -- and emits select event for newly selected item
func (tv *TreeView) MoveUpAction(selMode mouse.SelectModes) *TreeView {
	nn := tv.MoveUp(selMode)
	if nn != nil && nn != tv {
		nn.GrabFocus()
		nn.ScrollToMe()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), nn.This())
	}
	return nn
}

// TreeViewPageSteps is the number of steps to take in PageUp / Down events
var TreeViewPageSteps = 10

// MovePageUpAction moves the selection up to previous TreeViewPageSteps elements in the tree,
// using given select mode (from keyboard modifiers) -- and emits select event for newly selected item
func (tv *TreeView) MovePageUpAction(selMode mouse.SelectModes) *TreeView {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	fnn := tv.MoveUp(selMode)
	if fnn != nil && fnn != tv {
		for i := 1; i < TreeViewPageSteps; i++ {
			nn := fnn.MoveUp(selMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		fnn.GrabFocus()
		fnn.ScrollToMe()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), fnn.This())
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
	return fnn
}

// MovePageDownAction moves the selection up to previous TreeViewPageSteps elements in the tree,
// using given select mode (from keyboard modifiers) -- and emits select event for newly selected item
func (tv *TreeView) MovePageDownAction(selMode mouse.SelectModes) *TreeView {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	fnn := tv.MoveDown(selMode)
	if fnn != nil && fnn != tv {
		for i := 1; i < TreeViewPageSteps; i++ {
			nn := fnn.MoveDown(selMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		fnn.GrabFocus()
		fnn.ScrollToMe()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), fnn.This())
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
	return fnn
}

// MoveToLastChild moves to the last child under me, using given select mode
// (from keyboard modifiers)
func (tv *TreeView) MoveToLastChild(selMode mouse.SelectModes) *TreeView {
	if tv.Par == nil || tv == tv.RootView {
		return nil
	}
	if !tv.IsClosed() && tv.HasChildren() {
		nnk, ok := tv.Children().ElemFromEnd(0)
		if ok {
			nn := nnk.Embed(KiT_TreeView).(*TreeView)
			return nn.MoveToLastChild(selMode)
		}
	} else {
		tv.SelectUpdate(selMode)
		return tv
	}
	return nil
}

// Close closes the given node and updates the view accordingly (if it is not already closed)
func (tv *TreeView) Close() {
	if !tv.IsClosed() {
		updt := tv.UpdateStart()
		if tv.HasChildren() {
			tv.SetFullReRender()
		}
		tv.SetClosed()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewClosed), tv.This())
		tv.UpdateEnd(updt)
	}
}

// Open opens the given node and updates the view accordingly (if it is not already opened)
func (tv *TreeView) Open() {
	if tv.IsClosed() {
		updt := tv.UpdateStart()
		if tv.HasChildren() {
			tv.SetFullReRender()
		}
		if tv.HasChildren() {
			tv.SetClosedState(false)
		}
		// send signal in any case -- dynamic trees can open a node here!
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewOpened), tv.This())
		tv.UpdateEnd(updt)
	}
}

// ToggleClose toggles the close / open status: if closed, opens, and vice-versa
func (tv *TreeView) ToggleClose() {
	if tv.IsClosed() {
		tv.Open()
	} else {
		tv.Close()
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Modifying Source Tree

func (tv *TreeView) ContextMenuPos() (pos image.Point) {
	pos.X = tv.WinBBox.Min.X + int(tv.Indent.Dots)
	pos.Y = (tv.WinBBox.Min.Y + tv.WinBBox.Max.Y) / 2
	return
}

func (tv *TreeView) MakeContextMenu(m *gi.Menu) {
	// derived types put native menu code here
	if tv.CtxtMenuFunc != nil {
		tv.CtxtMenuFunc(tv.This().(gi.Node2D), m)
	}
	if CtxtMenuView(tv.SrcNode.Ptr, tv.IsInactive(), tv.Viewport, m) { // our viewed obj's menu
		if tv.ShowViewCtxtMenu {
			m.AddSeparator("sep-tvmenu")
			CtxtMenuView(tv.This(), tv.IsInactive(), tv.Viewport, m)
		}
	} else {
		CtxtMenuView(tv.This(), tv.IsInactive(), tv.Viewport, m)
	}
}

// IsRootOrField returns true if given node is either the root of the
// tree or a field -- various operations can not be performed on these -- if
// string is passed, then a prompt dialog is presented with that as the name
// of the operation being attempted -- otherwise it silently returns (suitable
// for context menu UpdateFunc).
func (tv *TreeView) IsRootOrField(op string) bool {
	sk := tv.SrcNode.Ptr
	if sk.IsField() {
		if op != "" {
			gi.PromptDialog(tv.Viewport, gi.DlgOpts{Title: "TreeView " + op, Prompt: fmt.Sprintf("Cannot %v fields", op)}, true, false, nil, nil)
		}
		return true
	}
	if tv.This() == tv.RootView.This() {
		if op != "" {
			gi.PromptDialog(tv.Viewport, gi.DlgOpts{Title: "TreeView " + op, Prompt: fmt.Sprintf("Cannot %v the root of the tree", op)}, true, false, nil, nil)
		}
		return true
	}
	return false
}

// SrcInsertAfter inserts a new node in the source tree after this node, at
// the same (sibling) level, propmting for the type of node to insert
func (tv *TreeView) SrcInsertAfter() {
	ttl := "Insert After"
	if tv.IsRootOrField(ttl) {
		return
	}
	sk := tv.SrcNode.Ptr
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	gi.NewKiDialog(tv.Viewport, reflect.TypeOf((*gi.Node2D)(nil)).Elem(),
		gi.DlgOpts{Title: ttl, Prompt: "Number and Type of Items to Insert:"},
		tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				tv, _ := recv.Embed(KiT_TreeView).(*TreeView)
				sk := tv.SrcNode.Ptr
				par := sk.Parent()
				dlg, _ := send.(*gi.Dialog)
				n, typ := gi.NewKiDialogValues(dlg)
				updt := par.UpdateStart()
				for i := 0; i < n; i++ {
					nm := fmt.Sprintf("New%v%v", typ.Name(), myidx+1+i)
					par.InsertNewChild(typ, myidx+1+i, nm)
				}
				tv.SetChanged()
				par.UpdateEnd(updt)
			}
		})
}

// SrcInsertBefore inserts a new node in the source tree before this node, at
// the same (sibling) level, prompting for the type of node to insert
func (tv *TreeView) SrcInsertBefore() {
	ttl := "Insert Before"
	if tv.IsRootOrField(ttl) {
		return
	}
	sk := tv.SrcNode.Ptr
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	gi.NewKiDialog(tv.Viewport, reflect.TypeOf((*gi.Node2D)(nil)).Elem(),
		gi.DlgOpts{Title: ttl, Prompt: "Number and Type of Items to Insert:"},
		tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				tv, _ := recv.Embed(KiT_TreeView).(*TreeView)
				sk := tv.SrcNode.Ptr
				par := sk.Parent()
				dlg, _ := send.(*gi.Dialog)
				n, typ := gi.NewKiDialogValues(dlg)
				updt := par.UpdateStart()
				for i := 0; i < n; i++ {
					nm := fmt.Sprintf("New%v%v", typ.Name(), myidx+i)
					par.InsertNewChild(typ, myidx+i, nm)
				}
				tv.SetChanged()
				par.UpdateEnd(updt)
			}
		})
}

// SrcAddChild adds a new child node to this one in the source tree,
// propmpting the user for the type of node to add
func (tv *TreeView) SrcAddChild() {
	ttl := "Add Child"
	gi.NewKiDialog(tv.Viewport, reflect.TypeOf((*gi.Node2D)(nil)).Elem(),
		gi.DlgOpts{Title: ttl, Prompt: "Number and Type of Items to Add:"},
		tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				tv, _ := recv.Embed(KiT_TreeView).(*TreeView)
				sk := tv.SrcNode.Ptr
				dlg, _ := send.(*gi.Dialog)
				n, typ := gi.NewKiDialogValues(dlg)
				updt := sk.UpdateStart()
				for i := 0; i < n; i++ {
					nm := fmt.Sprintf("New%v%v", typ.Name(), i)
					sk.AddNewChild(typ, nm)
				}
				tv.SetChanged()
				sk.UpdateEnd(updt)
			}
		})
}

// SrcDelete deletes the source node corresponding to this view node in the source tree
func (tv *TreeView) SrcDelete() {
	ttl := "Delete"
	if tv.IsRootOrField(ttl) {
		return
	}
	if tv.MoveDown(mouse.NoSelectMode) == nil {
		tv.MoveUp(mouse.NoSelectMode)
	}
	sk := tv.SrcNode.Ptr
	sk.Delete(true)
	tv.SetChanged()
}

// SrcDuplicate duplicates the source node corresponding to this view node in
// the source tree, and inserts the duplicate after this node (as a new
// sibling)
func (tv *TreeView) SrcDuplicate() {
	ttl := "TreeView Duplicate"
	if tv.IsRootOrField(ttl) {
		return
	}
	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	nm := fmt.Sprintf("%vCopy", sk.Name())
	nwkid := sk.Clone()
	nwkid.SetName(nm)
	par.InsertChild(nwkid, myidx+1)
	tv.SetChanged()
}

// SrcEdit pulls up a StructViewDialog window on the source object viewed by this node
func (tv *TreeView) SrcEdit() {
	tynm := kit.NonPtrType(tv.SrcNode.Ptr.Type()).Name()
	StructViewDialog(tv.Viewport, tv.SrcNode.Ptr, DlgOpts{Title: tynm}, nil, nil)
}

// SrcGoGiEditor pulls up a new GoGiEditor window on the source object viewed by this node
func (tv *TreeView) SrcGoGiEditor() {
	GoGiEditorDialog(tv.SrcNode.Ptr)
}

//////////////////////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeData adds mimedata for this node: a text/plain of the PathUnique, and
// an application/json of the source node
func (tv *TreeView) MimeData(md *mimedata.Mimes) {
	sroot := tv.RootView.SrcNode.Ptr
	src := tv.SrcNode.Ptr
	*md = append(*md, mimedata.NewTextData(src.PathFromUnique(sroot)))
	var buf bytes.Buffer
	err := src.WriteJSON(&buf, true) // true = pretty for clipboard..
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: mimedata.AppJSON, Data: buf.Bytes()})
	} else {
		log.Printf("gi.TreeView MimeData SaveJSON error: %v\n", err)
	}
}

// NodesFromMimeData creates a slice of Ki node(s) from given mime data
func (tv *TreeView) NodesFromMimeData(md mimedata.Mimes) ki.Slice {
	sl := make(ki.Slice, 0, len(md)/2)
	for _, d := range md {
		if d.Type == mimedata.AppJSON {
			nki, err := ki.ReadNewJSON(bytes.NewReader(d.Data))
			if err == nil {
				sl = append(sl, nki)
			} else {
				log.Printf("TreeView NodesFromMimeData: JSON load error: %v\n", err)
			}
		}
	}
	return sl
}

// Copy copies to clip.Board, optionally resetting the selection
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Copy(reset bool) {
	sels := tv.SelectedViews()
	nitms := ints.MaxInt(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This() != tv.This() {
				sn.MimeData(&md)
			}
		}
	}
	oswin.TheApp.ClipBoard(tv.Viewport.Win.OSWin).Write(md)
	if reset {
		tv.UnselectAll()
	}
}

// CopyAction copies to clip.Board, optionally resetting the selection-- calls Clipper copy
func (tv *TreeView) CopyAction(reset bool) {
	if cpr, ok := tv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Copy(reset)
	} else {
		tv.Copy(reset)
	}
}

// Cut copies to clip.Board and deletes selected items
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Cut() {
	if tv.IsRootOrField("Cut") {
		return
	}
	tv.Copy(false)
	sels := tv.SelectedSrcNodes()
	tv.UnselectAll()
	for _, sn := range sels {
		sn.Delete(true)
	}
	tv.SetChanged()
}

// CutAction copies to clip.Board and deletes selected items -- calls Clipper cut
func (tv *TreeView) CutAction() {
	if cpr, ok := tv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Cut()
	} else {
		tv.Cut()
	}
}

// Paste pastes clipboard at given node
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Paste() {
	md := oswin.TheApp.ClipBoard(tv.Viewport.Win.OSWin).Read([]string{mimedata.AppJSON})
	if md != nil {
		tv.PasteMenu(md)
	}
}

// PasteAction pastes clipboard at given node -- calls Clipper paste
func (tv *TreeView) PasteAction() {
	if cpr, ok := tv.This().(gi.Clipper); ok { // should always be true, but justin case..
		cpr.Paste()
	} else {
		tv.Paste()
	}
}

// MakePasteMenu makes the menu of options for paste events
func (tv *TreeView) MakePasteMenu(m *gi.Menu, data interface{}) {
	if len(*m) > 0 {
		return
	}
	m.AddAction(gi.ActOpts{Label: "Assign To", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TreeView).(*TreeView)
		tvv.PasteAssign(data.(mimedata.Mimes))
	})
	m.AddAction(gi.ActOpts{Label: "Add to Children", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TreeView).(*TreeView)
		tvv.PasteChildren(data.(mimedata.Mimes), dnd.DropCopy)
	})
	if !tv.IsRootOrField("") && tv.RootView.This() != tv.This() {
		m.AddAction(gi.ActOpts{Label: "Insert Before", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TreeView).(*TreeView)
			tvv.PasteBefore(data.(mimedata.Mimes), dnd.DropCopy)
		})
		m.AddAction(gi.ActOpts{Label: "Insert After", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TreeView).(*TreeView)
			tvv.PasteAfter(data.(mimedata.Mimes), dnd.DropCopy)
		})
	}
	m.AddAction(gi.ActOpts{Label: "Cancel", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	})
	// todo: compare, etc..
}

// PasteMenu performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (tv *TreeView) PasteMenu(md mimedata.Mimes) {
	tv.UnselectAll()
	var men gi.Menu
	tv.MakePasteMenu(&men, md)
	pos := tv.ContextMenuPos()
	gi.PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvPasteMenu")
}

// PasteAssign assigns mime data (only the first one!) to this node
func (tv *TreeView) PasteAssign(md mimedata.Mimes) {
	sl := tv.NodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	sk := tv.SrcNode.Ptr
	sk.CopyFrom(sl[0])
	tv.SetChanged()
}

// PasteBefore inserts object(s) from mime data before this node -- mod =
// DropCopy will append _Copy on the name of the inserted object
func (tv *TreeView) PasteBefore(md mimedata.Mimes, mod dnd.DropMods) {
	ttl := "Paste Before"
	sl := tv.NodesFromMimeData(md)

	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		gi.PromptDialog(tv.Viewport, gi.DlgOpts{Title: ttl, Prompt: "Cannot insert before the root of the tree"}, true, false, nil, nil)
		return
	}
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	updt := par.UpdateStart()
	for i, ns := range sl {
		if mod == dnd.DropCopy {
			ns.SetName(ns.Name() + "_Copy")
		}
		par.InsertChild(ns, myidx+i)
	}
	par.UpdateEnd(updt)
	tv.SetChanged()
}

// PasteAfter inserts object(s) from mime data after this node -- mod =
// DropCopy will append _Copy on the name of the inserted objects
func (tv *TreeView) PasteAfter(md mimedata.Mimes, mod dnd.DropMods) {
	ttl := "Paste After"
	sl := tv.NodesFromMimeData(md)

	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		gi.PromptDialog(tv.Viewport, gi.DlgOpts{Title: ttl, Prompt: "Cannot insert after the root of the tree"}, true, false, nil, nil)
		return
	}
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	updt := par.UpdateStart()
	for i, ns := range sl {
		if mod == dnd.DropCopy {
			ns.SetName(ns.Name() + "_Copy")
		}
		par.InsertChild(ns, myidx+1+i)
	}
	par.UpdateEnd(updt)
	tv.SetChanged()
}

// PasteChildren inserts object(s) from mime data at end of children of this
// node -- mod = DropCopy will append _Copy on the name of the inserted
// objects
func (tv *TreeView) PasteChildren(md mimedata.Mimes, mod dnd.DropMods) {
	sl := tv.NodesFromMimeData(md)

	sk := tv.SrcNode.Ptr
	updt := sk.UpdateStart()
	for _, ns := range sl {
		if mod == dnd.DropCopy {
			ns.SetName(ns.Name() + "_Copy")
		}
		sk.AddChild(ns)
	}
	sk.UpdateEnd(updt)
	tv.SetChanged()
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragNDropStart starts a drag-n-drop on this node -- it includes any other
// selected nodes as well, each as additional records in mimedata
func (tv *TreeView) DragNDropStart() {
	sels := tv.SelectedViews()
	nitms := ints.MaxInt(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This() != tv.This() {
				sn.MimeData(&md)
			}
		}
	}
	bi := &gi.Bitmap{}
	bi.InitName(bi, tv.UniqueName())
	bi.GrabRenderFrom(tv) // todo: show number of items?
	gi.ImageClearer(bi.Pixels, 50.0)
	tv.Viewport.Win.StartDragNDrop(tv.This(), md, bi)
}

// DragNDropTarget handles a drag-n-drop onto this node
func (tv *TreeView) DragNDropTarget(de *dnd.Event) {
	de.Target = tv.This()
	if de.Mod == dnd.DropLink {
		de.Mod = dnd.DropCopy // link not supported -- revert to copy
	}
	de.SetProcessed()
	if dpr, ok := tv.This().(gi.DragNDropper); ok {
		dpr.Drop(de.Data, de.Mod)
	} else {
		tv.Drop(de.Data, de.Mod)
	}
}

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore
func (tv *TreeView) DragNDropFinalize(mod dnd.DropMods) {
	tv.UnselectAll()
	tv.Viewport.Win.FinalizeDragNDrop(mod)
}

// Dragged is called after target accepts the drop -- we just remove
// elements that were moved
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (tv *TreeView) Dragged(de *dnd.Event) {
	if de.Mod != dnd.DropMove {
		return
	}
	sroot := tv.RootView.SrcNode.Ptr
	md := de.Data
	for _, d := range md {
		if d.Type == mimedata.TextPlain { // link
			path := string(d.Data)
			sn, ok := sroot.FindPathUnique(path)
			if ok {
				sn.Delete(true)
			}
		}
	}
}

// DragNDropSource is called after target accepts the drop -- we just remove
// elements that were moved
func (tv *TreeView) DragNDropSource(de *dnd.Event) {
	// fmt.Printf("tv src: %v\n", tv.PathUnique())
	if dpr, ok := tv.This().(gi.DragNDropper); ok {
		dpr.Dragged(de)
	} else {
		tv.Dragged(de)
	}
}

// MakeDropMenu makes the menu of options for dropping on a target
func (tv *TreeView) MakeDropMenu(m *gi.Menu, data interface{}, mod dnd.DropMods) {
	if len(*m) > 0 {
		return
	}
	switch mod {
	case dnd.DropCopy:
		m.AddLabel("Copy (Use Shift to Move):")
	case dnd.DropMove:
		m.AddLabel("Move:")
	}
	if mod == dnd.DropCopy {
		m.AddAction(gi.ActOpts{Label: "Assign To", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TreeView).(*TreeView)
			tvv.DropAssign(data.(mimedata.Mimes))
		})
	}
	m.AddAction(gi.ActOpts{Label: "Add to Children", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TreeView).(*TreeView)
		tvv.DropChildren(data.(mimedata.Mimes), mod) // captures mod
	})
	if !tv.IsRootOrField("") && tv.RootView.This() != tv.This() {
		m.AddAction(gi.ActOpts{Label: "Insert Before", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TreeView).(*TreeView)
			tvv.DropBefore(data.(mimedata.Mimes), mod) // captures mod
		})
		m.AddAction(gi.ActOpts{Label: "Insert After", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TreeView).(*TreeView)
			tvv.DropAfter(data.(mimedata.Mimes), mod) // captures mod
		})
	}
	m.AddAction(gi.ActOpts{Label: "Cancel", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TreeView).(*TreeView)
		tvv.DropCancel()
	})
	// todo: compare, etc..
}

// Drop pops up a menu to determine what specifically to do with dropped items
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (tv *TreeView) Drop(md mimedata.Mimes, mod dnd.DropMods) {
	var men gi.Menu
	tv.MakeDropMenu(&men, md, mod)
	pos := tv.ContextMenuPos()
	gi.PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvDropMenu")
}

// DropAssign assigns mime data (only the first one!) to this node
func (tv *TreeView) DropAssign(md mimedata.Mimes) {
	tv.DragNDropFinalize(dnd.DropCopy)
	tv.PasteAssign(md)
}

// DropBefore inserts object(s) from mime data before this node
func (tv *TreeView) DropBefore(md mimedata.Mimes, mod dnd.DropMods) {
	tv.DragNDropFinalize(mod)
	tv.PasteBefore(md, mod)
}

// DropAfter inserts object(s) from mime data after this node
func (tv *TreeView) DropAfter(md mimedata.Mimes, mod dnd.DropMods) {
	tv.DragNDropFinalize(mod)
	tv.PasteAfter(md, mod)
}

// DropChildren inserts object(s) from mime data at end of children of this node
func (tv *TreeView) DropChildren(md mimedata.Mimes, mod dnd.DropMods) {
	tv.DragNDropFinalize(mod)
	tv.PasteChildren(md, mod)
}

// DropCancel cancels the drop action e.g., preventing deleting of source
// items in a Move case
func (tv *TreeView) DropCancel() {
	tv.DragNDropFinalize(dnd.DropIgnore)
}

////////////////////////////////////////////////////
// Infrastructure

func (tv *TreeView) TreeViewParent() *TreeView {
	if tv.Par == nil {
		return nil
	}
	if tv.Par.TypeEmbeds(KiT_TreeView) {
		return tv.Par.Embed(KiT_TreeView).(*TreeView)
	}
	// I am rootview!
	return nil
}

// RootTreeView returns the root node of TreeView tree -- several properties
// for the overall view are stored there -- cached..
func (tv *TreeView) RootTreeView() *TreeView {
	rn := tv
	tv.FuncUp(0, tv.This(), func(k ki.Ki, level int, d interface{}) bool {
		_, pg := gi.KiToNode2D(k)
		if pg == nil {
			return false
		}
		if k.TypeEmbeds(KiT_TreeView) {
			rn = k.Embed(KiT_TreeView).(*TreeView)
			return true
		} else {
			return false
		}
	})
	return rn
}

func (tv *TreeView) KeyInput(kt *key.ChordEvent) {
	if gi.KeyEventTrace {
		fmt.Printf("TreeView KeyInput: %v\n", tv.PathUnique())
	}
	kf := gi.KeyFun(kt.Chord())
	selMode := mouse.SelectModeBits(kt.Modifiers)

	// first all the keys that work for inactive and active
	switch kf {
	case gi.KeyFunCancelSelect:
		tv.UnselectAll()
		tv.SetSelectMode(false)
		kt.SetProcessed()
	case gi.KeyFunMoveRight:
		tv.Open()
		kt.SetProcessed()
	case gi.KeyFunMoveLeft:
		tv.Close()
		kt.SetProcessed()
	case gi.KeyFunMoveDown:
		tv.MoveDownAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunMoveUp:
		tv.MoveUpAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunPageUp:
		tv.MovePageUpAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunPageDown:
		tv.MovePageDownAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunSelectMode:
		tv.SelectModeToggle()
		kt.SetProcessed()
	case gi.KeyFunSelectAll:
		tv.SelectAll()
		kt.SetProcessed()
	case gi.KeyFunEnter:
		tv.ToggleClose()
		kt.SetProcessed()
	case gi.KeyFunCopy:
		tv.CopyAction(true)
		kt.SetProcessed()
	}
	if !tv.IsInactive() && !kt.IsProcessed() {
		switch kf {
		case gi.KeyFunDelete:
			tv.SrcDelete()
			kt.SetProcessed()
		case gi.KeyFunDuplicate:
			tv.SrcDuplicate()
			kt.SetProcessed()
		case gi.KeyFunInsert:
			tv.SrcInsertBefore()
			kt.SetProcessed()
		case gi.KeyFunInsertAfter:
			tv.SrcInsertAfter()
			kt.SetProcessed()
		case gi.KeyFunCut:
			tv.CutAction()
			kt.SetProcessed()
		case gi.KeyFunPaste:
			tv.PasteAction()
			kt.SetProcessed()
		}
	}
}

func (tv *TreeView) TreeViewEvents() {
	tv.ConnectEvent(oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		tvv := recv.Embed(KiT_TreeView).(*TreeView)
		kt := d.(*key.ChordEvent)
		tvv.KeyInput(kt)
	})
	tv.ConnectEvent(oswin.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		de := d.(*dnd.Event)
		tvv := recv.Embed(KiT_TreeView).(*TreeView)
		switch de.Action {
		case dnd.Start:
			tvv.DragNDropStart()
		case dnd.DropOnTarget:
			tvv.DragNDropTarget(de)
		case dnd.DropFmSource:
			tvv.DragNDropSource(de)
		}
	})
	tv.ConnectEvent(oswin.DNDFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		de := d.(*dnd.FocusEvent)
		tvv := recv.Embed(KiT_TreeView).(*TreeView)
		switch de.Action {
		case dnd.Enter:
			tvv.Viewport.Win.DNDSetCursor(de.Mod)
		case dnd.Exit:
			tvv.Viewport.Win.DNDNotCursor()
		case dnd.Hover:
			tvv.Open()
		}
	})
	if tv.HasChildren() {
		if wb, ok := tv.BranchPart(); ok {
			wb.ButtonSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.ButtonToggled) {
					tvv, _ := recv.Embed(KiT_TreeView).(*TreeView)
					tvv.ToggleClose()
				}
			})
		}
	}
	if lbl, ok := tv.LabelPart(); ok {
		// HiPri is needed to override label's native processing
		lbl.ConnectEvent(oswin.MouseEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			lb, _ := recv.(*gi.Label)
			tvvi := lb.Parent().Parent()
			if tvvi == nil { // deleted
				return
			}
			tvv := tvvi.Embed(KiT_TreeView).(*TreeView)
			me := d.(*mouse.Event)
			switch me.Button {
			case mouse.Left:
				switch me.Action {
				case mouse.DoubleClick:
					tvv.ToggleClose()
					me.SetProcessed()
				case mouse.Release:
					tvv.SelectAction(me.SelectMode())
					me.SetProcessed()
				}
			case mouse.Right:
				if me.Action == mouse.Release {
					me.SetProcessed()
					tvv.This().(gi.Node2D).ContextMenu()
				}
			}
		})
	}
}

////////////////////////////////////////////////////
// Node2D interface

// qt calls the open / close thing a "branch"
// http://doc.qt.io/qt-5/stylesheet-examples.html#customizing-qtreeview

var TVBranchProps = ki.Props{
	"fill":   &gi.Prefs.Colors.Icon,
	"stroke": &gi.Prefs.Colors.Font,
}

// BranchPart returns the branch in parts, if it exists
func (tv *TreeView) BranchPart() (*gi.CheckBox, bool) {
	if icc, ok := tv.Parts.ChildByName("branch", 0); ok {
		return icc.(*gi.CheckBox), ok
	}
	return nil, false
}

// IconPart returns the icon in parts, if it exists
func (tv *TreeView) IconPart() (*gi.Icon, bool) {
	if icc, ok := tv.Parts.ChildByName("icon", 1); ok {
		return icc.(*gi.Icon), ok
	}
	return nil, false
}

// LabelPart returns the label in parts, if it exists
func (tv *TreeView) LabelPart() (*gi.Label, bool) {
	if lbl, ok := tv.Parts.ChildByName("label", 1); ok {
		return lbl.(*gi.Label), ok
	}
	return nil, false
}

func (tv *TreeView) ConfigParts() {
	tv.Parts.Lay = gi.LayoutHoriz
	config := kit.TypeAndNameList{}
	if tv.HasChildren() {
		config.Add(gi.KiT_CheckBox, "branch")
	}
	if tv.Icon.IsValid() {
		config.Add(gi.KiT_Icon, "icon")
	}
	config.Add(gi.KiT_Label, "label")
	mods, updt := tv.Parts.ConfigChildren(config, false) // not unique names
	// if mods {
	if tv.HasChildren() {
		if wb, ok := tv.BranchPart(); ok {
			wb.SetProp("#icon0", TVBranchProps)
			wb.SetProp("#icon1", TVBranchProps)
			wb.SetProp("no-focus", true) // note: cannot be in compiled props
			tv.StylePart(gi.Node2D(wb))
			// unfortunately StylePart only handles default Style obj -- not
			// these special styles.. todo: fix this somehow
			if bprpi, ok := tv.Prop("#branch"); ok {
				switch pr := bprpi.(type) {
				case map[string]interface{}:
					wb.SetIconProps(ki.Props(pr))
				case ki.Props:
					wb.SetIconProps(pr)
				}
			} else {
				tprops := *kit.Types.Properties(tv.Type(), true) // true = makeNew
				if bprpi, ok := kit.TypeProp(tprops, gi.WidgetDefPropsKey+"#branch"); ok {
					switch pr := bprpi.(type) {
					case map[string]interface{}:
						wb.SetIconProps(ki.Props(pr))
					case ki.Props:
						wb.SetIconProps(pr)
					}
				}
			}
		}
	}
	// }
	if tv.Icon.IsValid() {
		if ic, ok := tv.IconPart(); ok {
			if set, _ := ic.SetIcon(string(tv.Icon)); set || tv.NeedsFullReRender() || mods {
				tv.StylePart(gi.Node2D(ic))
			}
		}
	}
	if lbl, ok := tv.LabelPart(); ok {
		lbl.SetText(tv.Label())
		if mods {
			tv.StylePart(gi.Node2D(lbl))
		}
	}
	tv.Parts.UpdateEnd(updt)
}

func (tv *TreeView) ConfigPartsIfNeeded() {
	if !tv.Parts.HasChildren() {
		tv.ConfigParts()
	}
	if lbl, ok := tv.LabelPart(); ok {
		ltxt := tv.Label()
		if lbl.Text != ltxt {
			lbl.SetText(ltxt)
		}
		lbl.Sty.Font.Color = tv.Sty.Font.Color
	}
	if tv.HasChildren() {
		if wb, ok := tv.BranchPart(); ok {
			wb.SetChecked(!tv.IsClosed())
		}
	}
}

var TreeViewProps = ki.Props{
	"indent":           units.NewValue(2, units.Ch),
	"spacing":          units.NewValue(.5, units.Ch),
	"border-width":     units.NewValue(0, units.Px),
	"border-radius":    units.NewValue(0, units.Px),
	"padding":          units.NewValue(0, units.Px),
	"margin":           units.NewValue(1, units.Px),
	"text-align":       gi.AlignLeft,
	"vertical-align":   gi.AlignTop,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": "inherit",
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &gi.Prefs.Colors.Icon,
		"stroke":  &gi.Prefs.Colors.Font,
	},
	"#branch": ki.Props{
		"icon":             "widget-wedge-down",
		"icon-off":         "widget-wedge-right",
		"margin":           units.NewValue(0, units.Px),
		"padding":          units.NewValue(0, units.Px),
		"background-color": color.Transparent,
		"max-width":        units.NewValue(.8, units.Em),
		"max-height":       units.NewValue(.8, units.Em),
	},
	"#space": ki.Props{
		"width": units.NewValue(.5, units.Em),
	},
	"#label": ki.Props{
		"margin":    units.NewValue(0, units.Px),
		"padding":   units.NewValue(0, units.Px),
		"min-width": units.NewValue(16, units.Ch),
	},
	"#menu": ki.Props{
		"indicator": "none",
	},
	TreeViewSelectors[TreeViewActive]: ki.Props{},
	TreeViewSelectors[TreeViewSel]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
	TreeViewSelectors[TreeViewFocus]: ki.Props{
		"background-color": &gi.Prefs.Colors.Control,
	},
	"CtxtMenuActive": ki.PropSlice{
		{"SrcAddChild", ki.Props{
			"label": "Add Child",
		}},
		{"SrcInsertBefore", ki.Props{
			"label":    "Insert Before",
			"shortcut": gi.KeyFunInsert,
			"updtfunc": ActionUpdateFunc(func(tvi interface{}, act *gi.Action) {
				tv := tvi.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
				act.SetInactiveState(tv.IsRootOrField(""))
			}),
		}},
		{"SrcInsertAfter", ki.Props{
			"label":    "Insert After",
			"shortcut": gi.KeyFunInsertAfter,
			"updtfunc": ActionUpdateFunc(func(tvi interface{}, act *gi.Action) {
				tv := tvi.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
				act.SetInactiveState(tv.IsRootOrField(""))
			}),
		}},
		{"SrcDuplicate", ki.Props{
			"label":    "Duplicate",
			"shortcut": gi.KeyFunDuplicate,
			"updtfunc": ActionUpdateFunc(func(tvi interface{}, act *gi.Action) {
				tv := tvi.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
				act.SetInactiveState(tv.IsRootOrField(""))
			}),
		}},
		{"SrcDelete", ki.Props{
			"label":    "Delete",
			"shortcut": gi.KeyFunDelete,
			"updtfunc": ActionUpdateFunc(func(tvi interface{}, act *gi.Action) {
				tv := tvi.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
				act.SetInactiveState(tv.IsRootOrField(""))
			}),
		}},
		{"sep-edit", ki.BlankProp{}},
		{"Copy", ki.Props{
			"shortcut": gi.KeyFunCopy,
			"Args": ki.PropSlice{
				{"reset", ki.Props{
					"value": true,
				}},
			},
		}},
		{"Cut", ki.Props{
			"shortcut": gi.KeyFunCut,
			"updtfunc": ActionUpdateFunc(func(tvi interface{}, act *gi.Action) {
				tv := tvi.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
				act.SetInactiveState(tv.IsRootOrField(""))
			}),
		}},
		{"Paste", ki.Props{
			"shortcut": gi.KeyFunPaste,
		}},
		{"sep-win", ki.BlankProp{}},
		{"SrcEdit", ki.Props{
			"label": "Edit",
		}},
		{"SrcGoGiEditor", ki.Props{
			"label": "GoGi Editor",
		}},
	},
	"CtxtMenuInactive": ki.PropSlice{
		{"Copy", ki.Props{
			"shortcut": gi.KeyFunCopy,
			"Args": ki.PropSlice{
				{"reset", ki.Props{
					"value": true,
				}},
			},
		}},
		{"SrcEdit", ki.Props{
			"label": "Edit",
		}},
		{"SrcGoGiEditor", ki.Props{
			"label": "GoGi Editor",
		}},
	},
}

func (tv *TreeView) Init2D() {
	// // optimized init -- avoid tree walking
	if tv.RootView != tv {
		tv.Viewport = tv.RootView.Viewport
	} else {
		tv.Viewport = tv.ParentViewport()
	}
	tv.Sty.Defaults()
	tv.LayData.Defaults() // doesn't overwrite
	tv.ConfigParts()
	tv.ConnectToViewport()
}

func (tv *TreeView) StyleTreeView() {
	if !tv.HasChildren() {
		tv.SetClosed()
	}
	if tv.HasClosedParent() {
		tv.ClearFlag(int(gi.CanFocus))
		return
	}
	tv.SetCanFocusIfActive()
	tv.Style2DWidget() // todo: maybe don't use CSS here, for big trees?

	pst := &(tv.Par.(gi.Node2D).AsWidget().Sty)
	for i := 0; i < int(TreeViewStatesN); i++ {
		tv.StateStyles[i].CopyFrom(&tv.Sty)
		tv.StateStyles[i].SetStyleProps(pst, tv.StyleProps(TreeViewSelectors[i]), tv.Viewport)
		tv.StateStyles[i].CopyUnitContext(&tv.Sty.UnContext)
	}
	tv.Indent.SetFmInheritProp("indent", tv.This(), false, true) // no inherit, yes type defaults
	tv.Indent.ToDots(&tv.Sty.UnContext)
	if spc, ok := tv.PropInherit("spacing", false, true); ok { // no inherit, yes type
		tv.Parts.SetProp("spacing", spc) // parts is otherwise not typically styled
	}
	tv.ConfigParts()
}

func (tv *TreeView) Style2D() {
	tv.StyleTreeView()
	tv.LayData.SetFromStyle(&tv.Sty.Layout) // also does reset
}

// TreeView is tricky for alloc because it is both a layout of its children but has to
// maintain its own bbox for its own widget.

func (tv *TreeView) Size2D(iter int) {
	tv.InitLayout2D()
	if tv.HasClosedParent() {
		return // nothing
	}
	tv.SizeFromParts(iter) // get our size from parts
	tv.WidgetSize = tv.LayData.AllocSize
	h := math32.Ceil(tv.WidgetSize.Y)
	w := tv.WidgetSize.X

	if !tv.IsClosed() {
		// we layout children under us
		for _, kid := range tv.Kids {
			gis := kid.(gi.Node2D).AsWidget()
			if gis == nil {
				continue
			}
			h += math32.Ceil(gis.LayData.AllocSize.Y)
			w = gi.Max32(w, tv.Indent.Dots+gis.LayData.AllocSize.X)
		}
	}
	tv.LayData.AllocSize = gi.Vec2D{w, h}
	tv.WidgetSize.X = w // stretch
}

func (tv *TreeView) Layout2DParts(parBBox image.Rectangle, iter int) {
	spc := tv.Sty.BoxSpace()
	tv.Parts.LayData.AllocPos = tv.LayData.AllocPos.AddVal(spc)
	tv.Parts.LayData.AllocPosOrig = tv.Parts.LayData.AllocPos
	tv.Parts.LayData.AllocSize = tv.WidgetSize.AddVal(-2.0 * spc)
	tv.Parts.Layout2D(parBBox, iter)
}

func (tv *TreeView) Layout2D(parBBox image.Rectangle, iter int) bool {
	if tv.HasClosedParent() {
		tv.LayData.AllocPosRel.X = -1000000 // put it very far off screen..
	}
	tv.ConfigPartsIfNeeded()

	psize := tv.AddParentPos() // have to add our pos first before computing below:

	rn := tv.RootView
	// our alloc size is root's size minus our total indentation
	tv.LayData.AllocSize.X = rn.LayData.AllocSize.X - (tv.LayData.AllocPos.X - rn.LayData.AllocPos.X)
	tv.WidgetSize.X = tv.LayData.AllocSize.X

	tv.LayData.AllocPosOrig = tv.LayData.AllocPos
	tv.Sty.SetUnitContext(tv.Viewport, psize) // update units with final layout
	for i := 0; i < int(TreeViewStatesN); i++ {
		tv.StateStyles[i].CopyUnitContext(&tv.Sty.UnContext)
	}
	tv.BBox = tv.This().(gi.Node2D).BBox2D() // only compute once, at this point
	tv.This().(gi.Node2D).ComputeBBox2D(parBBox, image.ZP)

	if gi.Layout2DTrace {
		fmt.Printf("Layout: %v reduced X allocsize: %v rn: %v  pos: %v rn pos: %v\n", tv.PathUnique(), tv.WidgetSize.X, rn.LayData.AllocSize.X, tv.LayData.AllocPos.X, rn.LayData.AllocPos.X)
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", tv.PathUnique(), tv.LayData.AllocPos, tv.LayData.AllocSize, tv.VpBBox, tv.WinBBox)
	}

	tv.Layout2DParts(parBBox, iter) // use OUR version
	h := math32.Ceil(tv.WidgetSize.Y)
	if !tv.IsClosed() {
		for _, kid := range tv.Kids {
			ni := kid.(gi.Node2D).AsWidget()
			if ni == nil {
				continue
			}
			ni.LayData.AllocPosRel.Y = h
			ni.LayData.AllocPosRel.X = tv.Indent.Dots
			h += math32.Ceil(ni.LayData.AllocSize.Y)
		}
	}
	return tv.Layout2DChildren(iter)
}

func (tv *TreeView) BBox2D() image.Rectangle {
	// we have unusual situation of bbox != alloc
	tp := tv.LayData.AllocPosOrig.ToPointFloor()
	ts := tv.WidgetSize.ToPointCeil()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

func (tv *TreeView) ChildrenBBox2D() image.Rectangle {
	ar := tv.BBoxFromAlloc() // need to use allocated size which includes children
	if tv.Par != nil {       // use parents children bbox to determine where we can draw
		pni, _ := gi.KiToNode2D(tv.Par)
		ar = ar.Intersect(pni.ChildrenBBox2D())
	}
	return ar
}

func (tv *TreeView) Render2D() {
	if tv.HasClosedParent() {
		tv.DisconnectAllEvents(gi.AllPris)
		return // nothing
	}
	// if tv.FullReRenderIfNeeded() { // custom stuff here
	// 	return
	// }
	// fmt.Printf("tv rend: %v\n", tv.Nm)
	if tv.PushBounds() {
		if tv.IsSelected() {
			tv.Sty = tv.StateStyles[TreeViewSel]
		} else if tv.HasFocus() {
			tv.Sty = tv.StateStyles[TreeViewFocus]
		} else {
			tv.Sty = tv.StateStyles[TreeViewActive]
		}
		tv.ConfigPartsIfNeeded()
		tv.This().(gi.Node2D).ConnectEvents2D()

		// note: this is std except using WidgetSize instead of AllocSize
		rs := &tv.Viewport.Render
		rs.Lock()
		pc := &rs.Paint
		st := &tv.Sty
		pc.FontStyle = st.Font
		pc.StrokeStyle.SetColor(&st.Border.Color)
		pc.StrokeStyle.Width = st.Border.Width
		pc.FillStyle.SetColorSpec(&st.Font.BgColor)
		// tv.RenderStdBox()
		pos := tv.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
		sz := tv.WidgetSize.AddVal(-2.0 * st.Layout.Margin.Dots)
		tv.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
		rs.Unlock()
		tv.Render2DParts()
		tv.PopBounds()
	} else {
		tv.DisconnectAllEvents(gi.AllPris)
	}
	// we always have to render our kids b/c we could be out of scope but they could be in!
	tv.Render2DChildren()
}

func (tv *TreeView) ConnectEvents2D() {
	tv.TreeViewEvents()
}

func (tv *TreeView) FocusChanged2D(change gi.FocusChanges) {
	switch change {
	case gi.FocusLost:
		tv.UpdateSig()
	case gi.FocusGot:
		tv.ScrollToMe()
		tv.EmitFocusedSignal()
		tv.UpdateSig()
	case gi.FocusInactive: // don't care..
	case gi.FocusActive:
	}
}
