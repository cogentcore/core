// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"reflect"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
)

////////////////////////////////////////////////////////////////////////////////////////
//  TreeView -- a widget that graphically represents / manipulates a Ki Tree

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

	TreeViewSignalsN
)

//go:generate stringer -type=TreeViewSignals

// these extend NodeBase NodeFlags to hold TreeView state
const (
	// node is closed
	TreeViewFlagClosed NodeFlags = NodeFlagsN + iota
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

// internal indexes for accessing elements of the widget -- todo: icon!
const (
	tvBranchIdx = iota
	tvSpaceIdx
	tvLabelIdx
)

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

// TreeView provides a graphical representation of source tree structure
// (which can be any type of Ki nodes), providing full manipulation abilities
// of that source tree (move, cut, add, etc) through drag-n-drop and
// cut/copy/paste and menu actions.
type TreeView struct {
	WidgetBase
	SrcNode     ki.Ptr                 `desc:"Ki Node that this widget is viewing in the tree -- the source"`
	ViewIdx     int                    `desc:"linear index of this node within the entire tree -- updated on full rebuilds and may sometimes be off, but close enough for expected uses"`
	Indent      units.Value            `xml:"indent" desc:"styled amount to indent children relative to this node"`
	TreeViewSig ki.Signal              `json:"-" xml:"-" desc:"signal for TreeView -- all are emitted from the root tree view widget, with data = affected node -- see TreeViewSignals for the types"`
	StateStyles [TreeViewStatesN]Style `json:"-" xml:"-" desc:"styles for different states of the widget -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	WidgetSize  Vec2D                  `desc:"just the size of our widget -- our alloc includes all of our children, but we only draw us"`
	Icon        *Icon                  `json:"-" xml:"-" desc:"optional icon, displayed to the the left of the text label"`
	RootView    *TreeView              `json:"-" xml:"-" desc:"cached root of the view"`
}

var KiT_TreeView = kit.Types.AddType(&TreeView{}, TreeViewProps)

//////////////////////////////////////////////////////////////////////////////
//    End-User API

// SetRootNode sets the root view to the root of the source node that we are
// viewing, and builds-out the view of its tree
func (tv *TreeView) SetRootNode(sk ki.Ki) {
	updt := false
	if tv.SrcNode.Ptr != sk {
		updt = tv.UpdateStart()
		tv.SrcNode.Ptr = sk
		sk.NodeSignal().Connect(tv.This, SrcNodeSignal) // we recv signals from source
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
		sk.NodeSignal().Connect(tv.This, SrcNodeSignal) // we recv signals from source
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
	skids := sk.Children()
	tnl := make(kit.TypeAndNameList, 0, len(skids))
	typ := tv.This.Type() // always make our type
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
			if vc, ok := kit.ToBool(k.Prop(vcprop, false, true)); vc && ok {
				cls = true
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
		vk := tv.Kids[idx].EmbeddedStruct(KiT_TreeView).(*TreeView)
		vk.SetSrcNode(fld, tvIdx)
		if mods {
			vk.SetClosedState(fldClosed[i])
		}
		idx++
	}
	for _, skid := range sk.Children() {
		vk := tv.Kids[idx].EmbeddedStruct(KiT_TreeView).(*TreeView)
		vk.SetSrcNode(skid, tvIdx)
		if mods {
			if vc, ok := kit.ToBool(skid.Prop(vcprop, false, true)); vc && ok {
				vk.SetClosed()
			}
		}
		idx++
	}
	tv.UpdateEnd(updt)
	pr.End()
}

// SrcNodeSignal is the function for receiving node signals from our SrcNode
func SrcNodeSignal(tvki, send ki.Ki, sig int64, data interface{}) {
	tv := tvki.EmbeddedStruct(KiT_TreeView).(*TreeView)
	if data != nil {
		dflags := data.(int64)
		if Update2DTrace {
			fmt.Printf("treeview: %v got signal: %v from node: %v  data: %v  flags %v\n", tv.PathUnique(), ki.NodeSignals(sig), send.PathUnique(), kit.BitFlagsToString(dflags, ki.FlagsN), kit.BitFlagsToString(*send.Flags(), ki.FlagsN))
		}
		if bitflag.HasMask(dflags, int64(ki.StruUpdateFlagsMask)) {
			tvIdx := tv.ViewIdx
			tv.SyncToSrc(&tvIdx)
		} else if bitflag.HasMask(dflags, int64(ki.ValUpdateFlagsMask)) {
			tv.UpdateSig()
		}
	}
}

// IsClosed returns whether this node itself closed?
func (tv *TreeView) IsClosed() bool {
	return bitflag.Has(tv.Flag, int(TreeViewFlagClosed))
}

// SetClosed sets the closed flag for this node -- call Close() method to
// close a node and update view
func (tv *TreeView) SetClosed() {
	bitflag.Set(&tv.Flag, int(TreeViewFlagClosed))
}

// SetClosedState sets the closed state based on arg
func (tv *TreeView) SetClosedState(closed bool) {
	bitflag.SetState(&tv.Flag, closed, int(TreeViewFlagClosed))
}

// HasClosedParent returns whether this node have a closed parent? if so, don't render!
func (tv *TreeView) HasClosedParent() bool {
	pcol := false
	tv.FuncUpParent(0, tv.This, func(k ki.Ki, level int, d interface{}) bool {
		_, pg := KiToNode2D(k)
		if pg == nil {
			return false
		}
		if pg.TypeEmbeds(KiT_TreeView) {
			// nw := pg.EmbeddedStruct(KiT_TreeView).(*TreeView)
			if bitflag.Has(pg.Flag, int(TreeViewFlagClosed)) {
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
//    Selection

// SelectMode returns true if keyboard movements should automatically select nodes
func (tv *TreeView) SelectMode() bool {
	smp := tv.RootView.Prop(TreeViewSelModeProp, false, false)
	if smp == nil {
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
	slp := tv.RootView.Prop(TreeViewSelProp, false, false)
	if slp == nil {
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
	tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewAllUnselected), tv.This)
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
	tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewAllSelected), tv.This)
}

// SelectAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne) -- only multiple sibling
// nodes can be selected -- otherwise the paste / drop implications don't make
// sense
func (tv *TreeView) SelectAction(mode mouse.SelectModes) {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	switch mode {
	case mouse.ExtendContinuous:
		sl := tv.SelectedViews()
		if len(sl) == 0 {
			tv.Select()
			tv.GrabFocus()
			tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewSelected), tv.This)
		} else {
			minIdx := -1
			maxIdx := 0
			for _, v := range sl {
				if minIdx < 0 {
					minIdx = v.ViewIdx
				} else {
					minIdx = kit.MinInt(minIdx, v.ViewIdx)
				}
				maxIdx = kit.MaxInt(maxIdx, v.ViewIdx)
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
			tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewSelected), tv.This)
		}
	case mouse.NoSelectMode:
		if tv.IsSelected() {
			sl := tv.SelectedViews()
			if len(sl) > 1 {
				tv.UnselectAll()
				tv.Select()
				tv.GrabFocus()
				tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewSelected), tv.This)
			}
		} else {
			tv.UnselectAll()
			tv.Select()
			tv.GrabFocus()
			tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewSelected), tv.This)
		}
	default: // anything else
		tv.Select()
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
}

// UnselectAction unselects this node (if selected) -- and emits a signal
func (tv *TreeView) UnselectAction() {
	if tv.IsSelected() {
		tv.Unselect()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewUnselected), tv.This)
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
			nn := tv.Child(0).EmbeddedStruct(KiT_TreeView).(*TreeView)
			if nn != nil {
				nn.SelectAction(selMode)
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
		tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewSelected), nn.This)
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
	myidx := tv.Index()
	if myidx < len(tv.Par.Children())-1 {
		nn := tv.Par.Child(myidx + 1).EmbeddedStruct(KiT_TreeView).(*TreeView)
		if nn != nil {
			nn.SelectAction(selMode)
			return nn
		}
	} else {
		return tv.Par.EmbeddedStruct(KiT_TreeView).(*TreeView).MoveDownSibling(selMode) // try up
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
	myidx := tv.Index()
	if myidx > 0 {
		nn := tv.Par.Child(myidx - 1).EmbeddedStruct(KiT_TreeView).(*TreeView)
		if nn != nil {
			return nn.MoveToLastChild(selMode)
		}
	} else {
		if tv.Par != nil {
			nn := tv.Par.EmbeddedStruct(KiT_TreeView).(*TreeView)
			if nn != nil {
				nn.SelectAction(selMode)
				return nn
			}
		}
	}
	return nil
}

// MoveUpAction moves the selection down to next element in the tree, using given
// select mode (from keyboard modifiers) -- and emits select event for newly selected item
func (tv *TreeView) MoveUpAction(selMode mouse.SelectModes) *TreeView {
	nn := tv.MoveUp(selMode)
	if nn != nil && nn != tv {
		nn.GrabFocus()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewSelected), nn.This)
	}
	return nn
}

// MoveToLastChild moves to the last child under me, using given select mode
// (from keyboard modifiers)
func (tv *TreeView) MoveToLastChild(selMode mouse.SelectModes) *TreeView {
	if tv.Par == nil || tv == tv.RootView {
		return nil
	}
	if !tv.IsClosed() && tv.HasChildren() {
		nn := tv.Child(-1).EmbeddedStruct(KiT_TreeView).(*TreeView)
		if nn != nil {
			return nn.MoveToLastChild(selMode)
		}
	} else {
		tv.SelectAction(selMode)
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
		bitflag.Set(&tv.Flag, int(TreeViewFlagClosed))
		tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewClosed), tv.This)
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
		bitflag.Clear(&tv.Flag, int(TreeViewFlagClosed))
		tv.RootView.TreeViewSig.Emit(tv.RootView.This, int64(TreeViewOpened), tv.This)
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

// MenuPosition returns the position for popup menus
func (tv *TreeView) MenuPosition() (pos image.Point) {
	pos.X = tv.WinBBox.Min.X + int(tv.Indent.Dots)
	pos.Y = (tv.WinBBox.Min.Y + tv.WinBBox.Max.Y) / 2
	return
}

// ActionMenu pops up a menu of various actions to perform on a node (todo:
// extend from src object methods)
func (tv *TreeView) ActionMenu() {
	var men Menu
	tv.MakeActionMenu(&men)
	pos := tv.MenuPosition()
	PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvActionMenu")
}

// MakeActionMenu makes the menu of actions that can be performed on each
// node.  TODO: check methods on src object for additional actions to
// insert.. need comment directives for that..
func (tv *TreeView) MakeActionMenu(m *Menu) {
	if len(*m) > 0 {
		return
	}
	// todo: shortcuts!
	m.AddMenuText("Add Child", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		tvv.SrcAddChild()
	})
	if !tv.IsField() && tv.RootView.This != tv.This {
		m.AddMenuText("Insert Before", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.SrcInsertBefore()
		})
		m.AddMenuText("Insert After", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.SrcInsertAfter()
		})
		m.AddMenuText("Duplicate", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.SrcDuplicate()
		})
		m.AddMenuText("Delete", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.SrcDelete()
		})
	}
	m.AddSeparator("")
	// todo: get shortcuts from keyfun
	m.AddMenuText("Copy", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		tvv.Copy(true)
	})
	if !tv.IsField() && tv.RootView.This != tv.This {
		m.AddMenuText("Cut", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.Cut()
		})
	}
	m.AddMenuText("Paste", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		tvv.Paste()
	})
}

// SrcInsertAfter inserts a new node in the source tree after this node, at
// the same (sibling) level, propmting for the type of node to insert
func (tv *TreeView) SrcInsertAfter() {
	ttl := "TreeView Insert After"
	if tv.IsField() {
		PromptDialog(tv.Viewport, ttl, "Cannot insert after fields", true, false, nil, nil)
		return
	}
	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(tv.Viewport, ttl, "Cannot insert after the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()
	NewKiDialog(tv.Viewport, reflect.TypeOf((*Node2D)(nil)).Elem(), ttl, "Number and Type of Items to Insert:", tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(DialogAccepted) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			sk := tv.SrcNode.Ptr
			par := sk.Parent()
			dlg, _ := send.(*Dialog)
			n, typ := NewKiDialogValues(dlg)
			updt := par.UpdateStart()
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name(), myidx+1+i)
				par.InsertNewChild(typ, myidx+1+i, nm)
			}
			par.UpdateEnd(updt)
		}
	})
}

// SrcInsertBefore inserts a new node in the source tree before this node, at
// the same (sibling) level, prompting for the type of node to insert
func (tv *TreeView) SrcInsertBefore() {
	ttl := "TreeView Insert Before"
	if tv.IsField() {
		PromptDialog(tv.Viewport, ttl, "Cannot insert before fields", true, false, nil, nil)
		return
	}
	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(tv.Viewport, ttl, "Cannot insert before the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()

	NewKiDialog(tv.Viewport, reflect.TypeOf((*Node2D)(nil)).Elem(), ttl, "Number and Type of Items to Insert:", tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(DialogAccepted) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			sk := tv.SrcNode.Ptr
			par := sk.Parent()
			dlg, _ := send.(*Dialog)
			n, typ := NewKiDialogValues(dlg)
			updt := par.UpdateStart()
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name(), myidx+i)
				par.InsertNewChild(typ, myidx+i, nm)
			}
			par.UpdateEnd(updt)
		}
	})
}

// SrcAddChild adds a new child node to this one in the source tree,
// propmpting the user for the type of node to add
func (tv *TreeView) SrcAddChild() {
	ttl := "TreeView Add Child"
	NewKiDialog(tv.Viewport, reflect.TypeOf((*Node2D)(nil)).Elem(), ttl, "Number and Type of Items to Add:", tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(DialogAccepted) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			sk := tv.SrcNode.Ptr
			dlg, _ := send.(*Dialog)
			n, typ := NewKiDialogValues(dlg)
			updt := sk.UpdateStart()
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name(), i)
				sk.AddNewChild(typ, nm)
			}
			sk.UpdateEnd(updt)
		}
	})
}

// SrcDelete deletes the source node corresponding to this view node in the source tree
func (tv *TreeView) SrcDelete() {
	if tv.IsField() {
		PromptDialog(tv.Viewport, "TreeView Delete", "Cannot delete fields", true, false, nil, nil)
		return
	}
	if tv.RootView.This == tv.This {
		PromptDialog(tv.Viewport, "TreeView Delete", "Cannot delete the root of the tree", true, false, nil, nil)
		return
	}
	tv.MoveDown(mouse.NoSelectMode)
	sk := tv.SrcNode.Ptr
	sk.Delete(true)
}

// SrcDuplicate duplicates the source node corresponding to this view node in
// the source tree, and inserts the duplicate after this node (as a new
// sibling)
func (tv *TreeView) SrcDuplicate() {
	if tv.IsField() {
		PromptDialog(tv.Viewport, "TreeView Duplicate", "Cannot delete fields", true, false, nil, nil)
		return
	}
	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(tv.Viewport, "TreeView Duplicate", "Cannot duplicate the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()
	nm := fmt.Sprintf("%vCopy", sk.Name())
	nwkid := sk.Clone()
	nwkid.SetName(nm)
	par.InsertChild(nwkid, myidx+1)
}

//////////////////////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeData adds mimedata for this node: a text/plain of the PathUnique, and
// an application/json of the source node
func (tv *TreeView) MimeData(md *mimedata.Mimes) {
	src := tv.SrcNode.Ptr
	*md = append(*md, mimedata.NewTextData(src.PathUnique()))
	jb, err := src.SaveJSON(true) // true = pretty for clipboard..
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: mimedata.AppJSON, Data: jb})
	} else {
		log.Printf("gi.TreeView MimeData SaveJSON error: %v\n", err)
	}
}

// NodesFromMimeData creates a slice of Ki node(s) from given mime data
func (tv *TreeView) NodesFromMimeData(md mimedata.Mimes) ki.Slice {
	sl := make(ki.Slice, 0, len(md)/2)
	for _, d := range md {
		if d.Type == mimedata.AppJSON {
			nki, err := ki.LoadNewJSON(d.Data)
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
func (tv *TreeView) Copy(reset bool) {
	sels := tv.SelectedViews()
	nitms := kit.MaxInt(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This != tv.This {
				sn.MimeData(&md)
			}
		}
	}
	oswin.TheApp.ClipBoard().Write(md)
	if reset {
		tv.UnselectAll()
	}
}

// Cut copies to clip.Board and deletes selected items
func (tv *TreeView) Cut() {
	tv.Copy(false)
	sels := tv.SelectedSrcNodes()
	tv.UnselectAll()
	for _, sn := range sels {
		sn.Delete(true)
	}
}

// Paste pastes clipboard at given node
func (tv *TreeView) Paste() {
	md := oswin.TheApp.ClipBoard().Read([]string{mimedata.AppJSON})
	if md != nil {
		tv.PasteAction(md)
	}
}

// MakePasteMenu makes the menu of options for paste events
func (tv *TreeView) MakePasteMenu(m *Menu, data interface{}) {
	if len(*m) > 0 {
		return
	}
	m.AddMenuText("Assign To", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		tvv.PasteAssign(data.(mimedata.Mimes))
	})
	m.AddMenuText("Add to Children", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		tvv.PasteChildren(data.(mimedata.Mimes), dnd.DropCopy)
	})
	if !tv.IsField() && tv.RootView.This != tv.This {
		m.AddMenuText("Insert Before", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.PasteBefore(data.(mimedata.Mimes), dnd.DropCopy)
		})
		m.AddMenuText("Insert After", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.PasteAfter(data.(mimedata.Mimes), dnd.DropCopy)
		})
	}
	m.AddMenuText("Cancel", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
	})
	// todo: compare, etc..
}

// PasteAction performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (tv *TreeView) PasteAction(md mimedata.Mimes) {
	tv.UnselectAll()
	var men Menu
	tv.MakePasteMenu(&men, md)
	pos := tv.MenuPosition()
	PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvPasteMenu")
}

// PasteAssign assigns mime data (only the first one!) to this node
func (tv *TreeView) PasteAssign(md mimedata.Mimes) {
	sl := tv.NodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	sk := tv.SrcNode.Ptr
	sk.CopyFrom(sl[0])
}

// PasteBefore inserts object(s) from mime data before this node -- mod =
// DropCopy will append _Copy on the name of the inserted object
func (tv *TreeView) PasteBefore(md mimedata.Mimes, mod dnd.DropMods) {
	ttl := "Paste Before"
	sl := tv.NodesFromMimeData(md)

	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(tv.Viewport, ttl, "Cannot insert before the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()
	updt := par.UpdateStart()
	for i, ns := range sl {
		if mod == dnd.DropCopy {
			ns.SetName(ns.Name() + "_Copy")
		}
		par.InsertChild(ns, myidx+i)
	}
	par.UpdateEnd(updt)
}

// PasteAfter inserts object(s) from mime data after this node -- mod =
// DropCopy will append _Copy on the name of the inserted objects
func (tv *TreeView) PasteAfter(md mimedata.Mimes, mod dnd.DropMods) {
	ttl := "Paste After"
	sl := tv.NodesFromMimeData(md)

	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(tv.Viewport, ttl, "Cannot after before the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()
	updt := par.UpdateStart()
	for i, ns := range sl {
		if mod == dnd.DropCopy {
			ns.SetName(ns.Name() + "_Copy")
		}
		par.InsertChild(ns, myidx+1+i)
	}
	par.UpdateEnd(updt)
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
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragNDropStart starts a drag-n-drop on this node -- it includes any other
// selected nodes as well, each as additional records in mimedata
func (tv *TreeView) DragNDropStart() {
	sels := tv.SelectedViews()
	nitms := kit.MaxInt(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This != tv.This {
				sn.MimeData(&md)
			}
		}
	}
	bi := &Bitmap{}
	bi.SetName(tv.UniqueName())
	bi.GrabRenderFrom(tv) // todo: show number of items?
	ImageClearer(bi.Pixels, 50.0)
	tv.Viewport.Win.StartDragNDrop(tv.This, md, bi)
}

// DragNDropTarget handles a drag-n-drop onto this node
func (tv *TreeView) DragNDropTarget(de *dnd.Event) {
	de.Target = tv.This
	if de.Mod == dnd.DropLink {
		de.Mod = dnd.DropCopy // link not supported -- revert to copy
	}
	de.SetProcessed()
	tv.DropAction(de.Data, de.Mod)
}

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore
func (tv *TreeView) DragNDropFinalize(mod dnd.DropMods) {
	tv.UnselectAll()
	tv.Viewport.Win.FinalizeDragNDrop(mod)
}

// DragNDropSource is called after target accepts the drop -- we just remove
// elements that were moved
func (tv *TreeView) DragNDropSource(de *dnd.Event) {
	if de.Mod != dnd.DropMove {
		return
	}
	sroot := tv.RootView.SrcNode.Ptr
	md := de.Data
	for _, d := range md {
		if d.Type == mimedata.TextPlain { // link
			path := string(d.Data)
			sn := sroot.FindPathUnique(path)
			if sn != nil {
				sn.Delete(true)
			}
		}
	}
}

// MakeDropMenu makes the menu of options for dropping on a target
func (tv *TreeView) MakeDropMenu(m *Menu, data interface{}, mod dnd.DropMods) {
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
		m.AddMenuText("Assign To", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.DropAssign(data.(mimedata.Mimes))
		})
	}
	m.AddMenuText("Add to Children", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		tvv.DropChildren(data.(mimedata.Mimes), mod) // captures mod
	})
	if !tv.IsField() && tv.RootView.This != tv.This {
		m.AddMenuText("Insert Before", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.DropBefore(data.(mimedata.Mimes), mod) // captures mod
		})
		m.AddMenuText("Insert After", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.DropAfter(data.(mimedata.Mimes), mod) // captures mod
		})
	}
	m.AddMenuText("Cancel", tv.This, data, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		tvv.DropCancel()
	})
	// todo: compare, etc..
}

// DropAction pops up a menu to determine what specifically to do with dropped items
func (tv *TreeView) DropAction(md mimedata.Mimes, mod dnd.DropMods) {
	var men Menu
	tv.MakeDropMenu(&men, md, mod)
	pos := tv.MenuPosition()
	PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvDropMenu")
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
		return tv.Par.EmbeddedStruct(KiT_TreeView).(*TreeView)
	}
	// I am rootview!
	return nil
}

// RootTreeView returns the root node of TreeView tree -- several properties
// for the overall view are stored there -- cached..
func (tv *TreeView) RootTreeView() *TreeView {
	rn := tv
	tv.FuncUp(0, tv.This, func(k ki.Ki, level int, d interface{}) bool {
		_, pg := KiToNode2D(k)
		if pg == nil {
			return false
		}
		if k.TypeEmbeds(KiT_TreeView) {
			rn = k.EmbeddedStruct(KiT_TreeView).(*TreeView)
			return true
		} else {
			return false
		}
	})
	return rn
}

func (tf *TreeView) KeyInput(kt *key.ChordEvent) {
	kf := KeyFun(kt.ChordString())
	selMode := mouse.SelectModeMod(kt.Modifiers)
	switch kf {
	case KeyFunSelectItem:
		tf.SelectAction(selMode)
		kt.SetProcessed()
	case KeyFunCancelSelect:
		tf.UnselectAll()
		kt.SetProcessed()
	case KeyFunMoveRight:
		tf.Open()
		kt.SetProcessed()
	case KeyFunMoveLeft:
		tf.Close()
		kt.SetProcessed()
	case KeyFunMoveDown:
		tf.MoveDownAction(selMode)
		kt.SetProcessed()
	case KeyFunMoveUp:
		tf.MoveUpAction(selMode)
		kt.SetProcessed()
	case KeyFunSelectMode:
		tf.SelectModeToggle()
		kt.SetProcessed()
	case KeyFunSelectAll:
		tf.SelectAll()
		kt.SetProcessed()
	case KeyFunDelete:
		tf.SrcDelete()
		kt.SetProcessed()
	case KeyFunDuplicate:
		tf.SrcDuplicate()
		kt.SetProcessed()
	case KeyFunInsert:
		tf.SrcInsertBefore()
		kt.SetProcessed()
	case KeyFunInsertAfter:
		tf.SrcInsertAfter()
		kt.SetProcessed()
	case KeyFunCopy:
		tf.Copy(true)
		kt.SetProcessed()
	case KeyFunCut:
		tf.Cut()
		kt.SetProcessed()
	case KeyFunPaste:
		tf.Paste()
		kt.SetProcessed()
	}
}

func (tv *TreeView) TreeViewEvents() {
	tv.ConnectEventType(oswin.KeyChordEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		kt := d.(*key.ChordEvent)
		tvv.KeyInput(kt)
	})
	tv.ConnectEventType(oswin.DNDEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		de := d.(*dnd.Event)
		tvv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		switch de.Action {
		case dnd.Start:
			tvv.DragNDropStart()
		case dnd.DropOnTarget:
			tvv.DragNDropTarget(de)
		case dnd.DropFmSource:
			tvv.DragNDropSource(de)
		}
	})
	wb := tv.Parts.Child(tvBranchIdx).(*CheckBox)
	wb.ButtonSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonToggled) {
			tvv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tvv.ToggleClose()
		}
	})
	lbl := tv.Parts.Child(tvLabelIdx).(*Label)
	lbl.ConnectEventType(oswin.MouseEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		lb, _ := recv.(*Label)
		tvv := lb.Parent().Parent().EmbeddedStruct(KiT_TreeView).(*TreeView)
		me := d.(*mouse.Event)
		me.SetProcessed()
		switch me.Button {
		case mouse.Left:
			switch me.Action {
			case mouse.DoubleClick:
				tvv.ToggleClose()
			case mouse.Release:
				tvv.SelectAction(me.SelectMode())
			}
		case mouse.Right:
			if me.Action == mouse.Press {
				tvv.ActionMenu()
			}
		}
	})
}

////////////////////////////////////////////////////
// Node2D interface

// qt calls the open / close thing a "branch"
// http://doc.qt.io/qt-5/stylesheet-examples.html#customizing-qtreeview

var TVBranchProps = ki.Props{
	"fill":   &Prefs.IconColor,
	"stroke": &Prefs.FontColor,
}

func (tv *TreeView) ConfigParts() {
	tv.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{}
	config.Add(KiT_CheckBox, "branch")
	config.Add(KiT_Space, "space")
	config.Add(KiT_Label, "label")
	mods, updt := tv.Parts.ConfigChildren(config, false) // not unique names

	wb := tv.Parts.Child(tvBranchIdx).(*CheckBox)
	wb.Icon = IconName("widget-wedge-down") // todo: style
	wb.IconOff = IconName("widget-wedge-right")
	if mods {
		wb.SetProp("#icon0", TVBranchProps)
		wb.SetProp("#icon1", TVBranchProps)
		tv.StylePart(wb.This)
	}

	lbl := tv.Parts.Child(tvLabelIdx).(*Label)
	lbl.Text = tv.Label()
	if mods {
		tv.StylePart(lbl.This)
	}
	tv.Parts.UpdateEnd(updt)
}

func (tv *TreeView) ConfigPartsIfNeeded() {
	if !tv.Parts.HasChildren() {
		tv.ConfigParts()
	}
	lbl := tv.Parts.Child(tvLabelIdx).(*Label)
	lbl.Text = tv.Label()
	lbl.Style.Color = tv.Style.Color
	wb := tv.Parts.Child(tvBranchIdx).(*CheckBox)
	wb.SetChecked(!tv.IsClosed())
}

func (tv *TreeView) Init2D() {
	// // optimized init -- avoid tree walking
	if tv.RootView != tv {
		tv.Viewport = tv.RootView.Viewport
	} else {
		tv.Viewport = tv.ParentViewport()
	}
	tv.Style.Defaults()
	tv.Paint.Defaults()
	tv.LayData.Defaults() // doesn't overwrite
	tv.ConfigParts()
	tv.ConnectToViewport()
}

var TreeViewProps = ki.Props{
	"indent":           units.NewValue(2, units.Ch),
	"border-width":     units.NewValue(0, units.Px),
	"border-radius":    units.NewValue(0, units.Px),
	"padding":          units.NewValue(1, units.Px),
	"margin":           units.NewValue(1, units.Px),
	"text-align":       AlignLeft,
	"vertical-align":   AlignTop,
	"background-color": "inherit",
	"#branch": ki.Props{
		"vertical-align":   AlignMiddle,
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
		"min-width": units.NewValue(16, units.Ex),
	},
	"#menu": ki.Props{
		"indicator": "none",
	},
	TreeViewSelectors[TreeViewActive]: ki.Props{},
	TreeViewSelectors[TreeViewSel]: ki.Props{
		"background-color": &Prefs.SelectColor,
	},
	TreeViewSelectors[TreeViewFocus]: ki.Props{
		"background-color": &Prefs.ControlColor,
	},
}

func (tv *TreeView) Style2D() {
	if tv.HasClosedParent() {
		bitflag.Clear(&tv.Flag, int(CanFocus))
		return
	}
	tv.SetCanFocusIfActive()
	tv.Style2DWidget()

	var pst *Style
	_, pg := KiToNode2D(tv.Par)
	if pg != nil {
		pst = &pg.Style
	}
	for i := 0; i < int(TreeViewStatesN); i++ {
		tv.StateStyles[i].CopyFrom(&tv.Style)
		tv.StateStyles[i].SetStyle(pst, tv.StyleProps(TreeViewSelectors[i]))
		tv.StateStyles[i].CopyUnitContext(&tv.Style.UnContext)
	}
	tv.Indent = units.NewValue(2, units.Ch) // default
	TreeViewFields.Style(tv, nil, tv.Props)
	TreeViewFields.ToDots(tv, &tv.Style.UnContext)
	tv.ConfigParts()
}

// TreeView is tricky for alloc because it is both a layout of its children but has to
// maintain its own bbox for its own widget.

func (tv *TreeView) Size2D() {
	tv.InitLayout2D()
	if tv.HasClosedParent() {
		return // nothing
	}
	tv.SizeFromParts() // get our size from parts
	tv.WidgetSize = tv.LayData.AllocSize
	h := tv.WidgetSize.Y
	w := tv.WidgetSize.X

	if !tv.IsClosed() {
		// we layout children under us
		for _, kid := range tv.Kids {
			_, gi := KiToNode2D(kid)
			if gi != nil {
				h += gi.LayData.AllocSize.Y
				w = Max32(w, tv.Indent.Dots+gi.LayData.AllocSize.X)
			}
		}
	}
	tv.LayData.AllocSize = Vec2D{w, h}
	tv.WidgetSize.X = w // stretch
}

func (tv *TreeView) Layout2DParts(parBBox image.Rectangle) {
	spc := tv.Style.BoxSpace()
	tv.Parts.LayData.AllocPos = tv.LayData.AllocPos.AddVal(spc)
	tv.Parts.LayData.AllocSize = tv.WidgetSize.AddVal(-2.0 * spc)
	tv.Parts.Layout2D(parBBox)
}

func (tv *TreeView) Layout2D(parBBox image.Rectangle) {
	if tv.HasClosedParent() {
		tv.LayData.AllocPosRel.X = -1000000 // put it very far off screen..
	}
	tv.ConfigParts()

	psize := tv.AddParentPos() // have to add our pos first before computing below:

	rn := tv.RootView
	// our alloc size is root's size minus our total indentation
	tv.LayData.AllocSize.X = rn.LayData.AllocSize.X - (tv.LayData.AllocPos.X - rn.LayData.AllocPos.X)
	tv.WidgetSize.X = tv.LayData.AllocSize.X

	tv.LayData.AllocPosOrig = tv.LayData.AllocPos
	tv.Style.SetUnitContext(tv.Viewport, psize) // update units with final layout
	tv.Paint.SetUnitContext(tv.Viewport, psize) // always update paint
	tv.BBox = tv.This.(Node2D).BBox2D()         // only compute once, at this point
	tv.This.(Node2D).ComputeBBox2D(parBBox, image.ZP)

	if Layout2DTrace {
		fmt.Printf("Layout: %v reduced X allocsize: %v rn: %v  pos: %v rn pos: %v\n", tv.PathUnique(), tv.WidgetSize.X, rn.LayData.AllocSize.X, tv.LayData.AllocPos.X, rn.LayData.AllocPos.X)
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", tv.PathUnique(), tv.LayData.AllocPos, tv.LayData.AllocSize, tv.VpBBox, tv.WinBBox)
	}

	tv.Layout2DParts(parBBox) // use OUR version
	h := tv.WidgetSize.Y
	if !tv.IsClosed() {
		for _, kid := range tv.Kids {
			_, gi := KiToNode2D(kid)
			if gi != nil {
				gi.LayData.AllocPosRel.Y = h
				gi.LayData.AllocPosRel.X = tv.Indent.Dots
				h += gi.LayData.AllocSize.Y
			}
		}
	}
	tv.Layout2DChildren()
}

func (tv *TreeView) BBox2D() image.Rectangle {
	// we have unusual situation of bbox != alloc
	tp := tv.LayData.AllocPos.ToPointFloor()
	ts := tv.WidgetSize.ToPointCeil()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

func (tv *TreeView) ChildrenBBox2D() image.Rectangle {
	ar := tv.BBoxFromAlloc() // need to use allocated size which includes children
	if tv.Par != nil {       // use parents children bbox to determine where we can draw
		pgi, _ := KiToNode2D(tv.Par)
		ar = ar.Intersect(pgi.ChildrenBBox2D())
	}
	return ar
}

func (tv *TreeView) Render2D() {
	if tv.HasClosedParent() {
		tv.DisconnectAllEvents()
		return // nothing
	}
	if tv.PushBounds() {
		// reset for next update
		tv.ClearFullReRender()

		if tv.IsSelected() {
			tv.Style = tv.StateStyles[TreeViewSel]
		} else if tv.HasFocus() {
			tv.Style = tv.StateStyles[TreeViewFocus]
		} else {
			tv.Style = tv.StateStyles[TreeViewActive]
		}
		tv.ConfigPartsIfNeeded()
		tv.TreeViewEvents()

		// note: this is std except using WidgetSize instead of AllocSize
		pc := &tv.Paint
		st := &tv.Style
		pc.FontStyle = st.Font
		pc.TextStyle = st.Text
		pc.StrokeStyle.SetColor(&st.Border.Color)
		pc.StrokeStyle.Width = st.Border.Width
		pc.FillStyle.SetColorSpec(&st.Background.Color)
		// tv.RenderStdBox()
		pos := tv.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
		sz := tv.WidgetSize.AddVal(-2.0 * st.Layout.Margin.Dots)
		tv.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
		tv.Render2DParts()
		tv.PopBounds()
	} else {
		tv.DisconnectAllEvents()
	}
	// we always have to render our kids b/c we could be out of scope but they could be in!
	tv.Render2DChildren()
}

func (tv *TreeView) ReRender2D() (node Node2D, layout bool) {
	if tv.NeedsFullReRender() {
		if Render2DTrace {
			fmt.Printf("Render: treeview full re-render on: %v\n", tv.PathUnique())
		}
		node = nil
		layout = false
	} else {
		if Render2DTrace {
			fmt.Printf("Render: treeview non-full re-render on: %v\n", tv.PathUnique())
		}
		node = tv.This.(Node2D)
		layout = false
	}
	return
}

func (tv *TreeView) FocusChanged2D(gotFocus bool) {
	tv.UpdateSig()
}

// TreeViewDefault is default obj that can be used when property specifies "default"
var TreeViewDefault TreeView

// TreeViewFields contain the StyledFields for TreeView type
var TreeViewFields = initTreeView()

func initTreeView() *StyledFields {
	TreeViewDefault = TreeView{}
	TreeViewDefault.Indent = units.NewValue(2, units.Ch)
	sf := &StyledFields{}
	sf.Default = &TreeViewDefault
	sf.AddField(&TreeViewDefault, "Indent")
	return sf
}
