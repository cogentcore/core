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
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/pi/filecat"
)

////////////////////////////////////////////////////////////////////////////////////////
//  TreeView -- a widget that graphically represents / manipulates a Ki Tree

// TreeView provides a graphical representation of source tree structure
// (which can be any type of Ki nodes), providing full manipulation abilities
// of that source tree (move, cut, add, etc) through drag-n-drop and
// cut/copy/paste and menu actions.
//
// There are special style Props interpreted by these nodes:
// * no-templates -- if present (assumed to be true) then style templates are
//   not used to optimize rendering speed.  Set this for nodes that have
//   styling applied differentially to individual nodes (e.g., FileNode).
type TreeView struct {
	gi.PartsWidgetBase
	SrcNode          ki.Ki                       `copy:"-" json:"-" xml:"-" desc:"Ki Node that this widget is viewing in the tree -- the source"`
	ShowViewCtxtMenu bool                        `desc:"if the object we're viewing has its own CtxtMenu property defined, should we also still show the view's own context menu?"`
	ViewIdx          int                         `desc:"linear index of this node within the entire tree -- updated on full rebuilds and may sometimes be off, but close enough for expected uses"`
	Indent           units.Value                 `xml:"indent" desc:"styled amount to indent children relative to this node"`
	OpenDepth        int                         `xml:"open-depth" desc:"styled depth for nodes be initialized as open -- nodes beyond this depth will be initialized as closed.  initial default is 4."`
	TreeViewSig      ki.Signal                   `json:"-" xml:"-" desc:"signal for TreeView -- all are emitted from the root tree view widget, with data = affected node -- see TreeViewSignals for the types"`
	StateStyles      [TreeViewStatesN]gist.Style `json:"-" xml:"-" desc:"styles for different states of the widget -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	WidgetSize       mat32.Vec2                  `desc:"just the size of our widget -- our alloc includes all of our children, but we only draw us"`
	Icon             gi.IconName                 `json:"-" xml:"icon" view:"show-name" desc:"optional icon, displayed to the the left of the text label"`
	RootView         *TreeView                   `json:"-" xml:"-" desc:"cached root of the view"`
}

var KiT_TreeView = kit.Types.AddType(&TreeView{}, nil)

// AddNewTreeView adds a new treeview to given parent node, with given name.
func AddNewTreeView(parent ki.Ki, name string) *TreeView {
	tv := parent.AddNewChild(KiT_TreeView, name).(*TreeView)
	tv.OpenDepth = 4
	return tv
}

func (tv *TreeView) Disconnect() {
	tv.PartsWidgetBase.Disconnect()
	tv.TreeViewSig.DisconnectAll()
}

func init() {
	kit.Types.SetProps(KiT_TreeView, TreeViewProps)
}

//////////////////////////////////////////////////////////////////////////////
//    End-User API

// SetRootNode sets the root view to the root of the source node that we are
// viewing, and builds-out the view of its tree.
// Calls ki.UniquifyNamesAll on source tree to ensure that node names are unique
// which is essential for proper viewing!
func (tv *TreeView) SetRootNode(sk ki.Ki) {
	updt := false
	ki.UniquifyNamesAll(sk)
	if tv.SrcNode != sk {
		updt = tv.UpdateStart()
		tv.SrcNode = sk
		sk.NodeSignal().Connect(tv.This(), SrcNodeSignalFunc) // we recv signals from source
	}
	tv.RootView = tv
	tvIdx := 0
	tv.SyncToSrc(&tvIdx, true, 0)
	tv.UpdateEnd(updt)
}

// SetSrcNode sets the source node that we are viewing,
// and builds-out the view of its tree.  It is called routinely
// via SyncToSrc during tree updating.
func (tv *TreeView) SetSrcNode(sk ki.Ki, tvIdx *int, init bool, depth int) {
	updt := false
	if tv.SrcNode != sk {
		updt = tv.UpdateStart()
		tv.SrcNode = sk
		sk.NodeSignal().Connect(tv.This(), SrcNodeSignalFunc) // we recv signals from source
	}
	tv.SyncToSrc(tvIdx, init, depth)
	tv.UpdateEnd(updt)
}

// ReSync resynchronizes the view relative to the underlying nodes
// and forces a full rerender
func (tv *TreeView) ReSync() {
	tv.SetFullReRender() //
	tvIdx := tv.ViewIdx
	tv.SyncToSrc(&tvIdx, false, 0)
	tv.UpdateSig()
}

// SyncToSrc updates the view tree to match the source tree, using
// ConfigChildren to maximally preserve existing tree elements.
// init means we are doing initial build, and depth tracks depth
// (only during init).
func (tv *TreeView) SyncToSrc(tvIdx *int, init bool, depth int) {
	// pr := prof.Start("TreeView.SyncToSrc")
	// defer pr.End()
	sk := tv.SrcNode
	nm := "tv_" + sk.Name()
	tv.SetName(nm)
	tv.ViewIdx = *tvIdx
	(*tvIdx)++
	tvPar := tv.TreeViewParent()
	if tvPar != nil {
		tv.RootView = tvPar.RootView
		if init && depth >= tv.RootView.OpenDepth {
			tv.SetClosed()
		}
	}
	vcprop := "view-closed"
	skids := *sk.Children()
	tnl := make(kit.TypeAndNameList, 0, len(skids))
	typ := ki.Type(tv.This()) // always make our type
	flds := make([]ki.Ki, 0)
	fldClosed := make([]bool, 0)
	sk.FuncFields(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		flds = append(flds, k)
		tnl.Add(typ, "tv_"+k.Name())
		ft := ki.FieldTag(sk.This(), k.Name(), vcprop)
		cls := false
		if vc, ok := kit.ToBool(ft); ok && vc {
			cls = true
		} else {
			if vcp, ok := k.PropInherit(vcprop, ki.NoInherit, ki.TypeProps); ok {
				if vc, ok := kit.ToBool(vcp); vc && ok {
					cls = true
				}
			}
		}
		fldClosed = append(fldClosed, cls)
		return true
	})
	for _, skid := range skids {
		tnl.Add(typ, "tv_"+skid.Name())
	}
	mods, updt := tv.ConfigChildren(tnl) // false = don't use unique names -- needs to!
	if mods {
		tv.SetFullReRender()
		// fmt.Printf("got mod on %v\n", tv.Path())
	}
	idx := 0
	for i, fld := range flds {
		vk := tv.Kids[idx].Embed(KiT_TreeView).(*TreeView)
		vk.SetSrcNode(fld, tvIdx, init, depth+1)
		if mods {
			vk.SetClosedState(fldClosed[i])
		}
		idx++
	}
	for _, skid := range *sk.Children() {
		if len(tv.Kids) <= idx {
			break
		}
		vk := tv.Kids[idx].Embed(KiT_TreeView).(*TreeView)
		vk.SetSrcNode(skid, tvIdx, init, depth+1)
		if mods {
			if vcp, ok := skid.PropInherit(vcprop, ki.NoInherit, ki.TypeProps); ok {
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
}

// SrcNodeSignalFunc is the function for receiving node signals from our SrcNode
func SrcNodeSignalFunc(tvki, send ki.Ki, sig int64, data interface{}) {
	tv := tvki.Embed(KiT_TreeView).(*TreeView)
	// always keep name updated in case that changed
	if data != nil {
		dflags := data.(int64)
		if gi.Update2DTrace {
			fmt.Printf("treeview: %v got signal: %v from node: %v  data: %v  flags %v\n", tv.Path(), ki.NodeSignals(sig), send.Path(), kit.BitFlagsToString(dflags, ki.FlagsN), kit.BitFlagsToString(send.Flags(), ki.FlagsN))
		}
		if tv.This() == tv.RootView.This() && tv.HasFlag(int(TreeViewFlagUpdtRoot)) {
			tv.SetFullReRender() // re-render for any updates on root node
		}
		if bitflag.HasAnyMask(dflags, int64(ki.StruUpdateFlagsMask)) {
			if tv.This() == tv.RootView.This() {
				tv.SetFullReRender() // re-render for struct updates on root node
			}
			tvIdx := tv.ViewIdx
			if gi.Update2DTrace {
				fmt.Printf("treeview: structupdate for node, idx: %v  %v", tvIdx, tv.Path())
			}
			tv.SyncToSrc(&tvIdx, false, 0)
		} else {
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
			return ki.Break
		}
		if ki.TypeEmbeds(pg, KiT_TreeView) {
			// nw := pg.Embed(KiT_TreeView).(*TreeView)
			if pg.HasFlag(int(TreeViewFlagClosed)) {
				pcol = true
				return ki.Break
			}
		}
		return ki.Continue
	})
	return pcol
}

// Label returns the display label for this node, satisfying the Labeler interface
func (tv *TreeView) Label() string {
	if lbl, has := gi.ToLabeler(tv.SrcNode); has {
		return lbl
	}
	return tv.SrcNode.Name()
}

// UpdateInactive updates the Inactive state based on SrcNode -- returns true if
// inactive.  The inactivity of individual nodes only affects display properties
// typically, and not overall functional behavior, which is controlled by
// inactivity of the root node (i.e, make the root inactive to make entire tree
// read-only and non-modifiable)
func (tv *TreeView) UpdateInactive() bool {
	tv.ClearInactive()
	if tv.SrcNode == nil {
		tv.SetInactive()
	} else {
		if inact, err := tv.SrcNode.PropTry("inactive"); err == nil {
			if bo, ok := kit.ToBool(inact); bo && ok {
				tv.SetInactive()
			}
		}
	}
	return tv.IsInactive()
}

// RootIsInactive returns the inactive status of the root node, which is what
// controls the functional inactivity of the tree -- if individual nodes
// are inactive that only affects display typically.
func (tv *TreeView) RootIsInactive() bool {
	if tv.RootView == nil {
		return true
	}
	return tv.RootView.IsInactive()
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

	// means that some kind of edit operation has taken place
	// by the user via the gui -- we don't track the details, just that
	// changes have happened
	TreeViewChanged

	// a node was inserted into the tree (Paste, DND)
	// in this case, the data is the *source node* that was inserted
	TreeViewInserted

	// a node was deleted from the tree (Cut, DND Move)
	TreeViewDeleted

	TreeViewSignalsN
)

//go:generate stringer -type=TreeViewSignals

// TreeViewFlags extend NodeBase NodeFlags to hold TreeView state
type TreeViewFlags int

//go:generate stringer -type=TreeViewFlags

var KiT_TreeViewFlags = kit.Enums.AddEnumExt(gi.KiT_NodeFlags, TreeViewFlagsN, kit.BitFlag, nil)

const (
	// TreeViewFlagClosed means node is toggled closed (children not visible)
	TreeViewFlagClosed TreeViewFlags = TreeViewFlags(gi.NodeFlagsN) + iota

	// TreeViewFlagChanged is updated on the root node whenever a gui edit is
	// made through the tree view on the tree -- this does not track any other
	// changes that might have occurred in the tree itself.
	// Also emits a TreeViewChanged signal on the root node.
	TreeViewFlagChanged

	// TreeViewFlagNoTemplate -- this node is not using a style template -- should
	// be restyled on any full re-render change
	TreeViewFlagNoTemplate

	// TreeViewFlagUpdtRoot -- for any update signal that comes from the source
	// root node, do a full update of the treeview.  This increases responsiveness
	// of the updating and makes it easy to trigger a full update by updating the root
	// node, but can be slower when not needed
	TreeViewFlagUpdtRoot

	TreeViewFlagsN
)

// TreeViewStates are mutually-exclusive tree view states -- determines appearance
type TreeViewStates int32

const (
	// TreeViewActive is normal state -- there but not being interacted with
	TreeViewActive TreeViewStates = iota

	// TreeViewSel is selected
	TreeViewSel

	// TreeViewFocus is in focus -- will respond to keyboard input
	TreeViewFocus

	// TreeViewInactive is inactive -- if SrcNode is nil, or source has "inactive" property
	// set, or treeview node has inactive property set directly
	TreeViewInactive

	TreeViewStatesN
)

//go:generate stringer -type=TreeViewStates

// TreeViewSelectors are Style selector names for the different states:
var TreeViewSelectors = []string{":active", ":selected", ":focus", ":inactive"}

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
	smp, err := tv.RootView.PropTry(TreeViewSelModeProp)
	if err != nil {
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
	slp, err := tv.RootView.PropTry(TreeViewSelProp)
	if err != nil {
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
		sn = append(sn, v.SrcNode)
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
	if tv.Viewport == nil {
		return
	}
	wupdt := tv.TopUpdateStart()
	sl := tv.SelectedViews()
	tv.SetSelectedViews(nil) // clear in advance
	for _, v := range sl {
		v.ClearSelected()
		v.UpdateSig()
	}
	tv.TopUpdateEnd(wupdt)
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewAllUnselected), tv.This())
}

// SelectAll all items in view
func (tv *TreeView) SelectAll() {
	if tv.Viewport == nil {
		return
	}
	wupdt := tv.TopUpdateStart()
	tv.UnselectAll()
	nn := tv.RootView
	nn.Select()
	for nn != nil {
		nn = nn.MoveDown(mouse.SelectQuiet)
	}
	tv.TopUpdateEnd(wupdt)
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewAllSelected), tv.This())
}

// SelectUpdate updates selection to include this node, using selectmode
// from mouse event (ExtendContinuous, ExtendOne).  Returns true if this node selected
func (tv *TreeView) SelectUpdate(mode mouse.SelectModes) bool {
	if mode == mouse.NoSelect {
		return false
	}
	wupdt := tv.TopUpdateStart()
	sel := false
	switch mode {
	case mouse.SelectOne:
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
					nn = nn.MoveDown(mouse.SelectQuiet) // just select
					cidx = nn.ViewIdx
				}
			} else if tv.ViewIdx > maxIdx {
				for cidx > maxIdx {
					nn = nn.MoveUp(mouse.SelectQuiet) // just select
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
	case mouse.SelectQuiet:
		tv.Select()
		// not sel -- no signal..
	case mouse.UnselectQuiet:
		tv.Unselect()
		// not sel -- no signal..
	}
	tv.TopUpdateEnd(wupdt)
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
	if tv.IsClosed() || !tv.HasChildren() { // next sibling
		return tv.MoveDownSibling(selMode)
	} else {
		if tv.HasChildren() {
			nn := tv.Child(0).Embed(KiT_TreeView).(*TreeView)
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
		nn := tv.Par.Child(myidx + 1).Embed(KiT_TreeView).(*TreeView)
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
	myidx, ok := tv.IndexInParent()
	if ok && myidx > 0 {
		nn := tv.Par.Child(myidx - 1).Embed(KiT_TreeView).(*TreeView)
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
	wupdt := tv.TopUpdateStart()
	mvMode := selMode
	if selMode == mouse.SelectOne {
		mvMode = mouse.NoSelect
	} else if selMode == mouse.ExtendContinuous || selMode == mouse.ExtendOne {
		mvMode = mouse.SelectQuiet
	}
	fnn := tv.MoveUp(mvMode)
	if fnn != nil && fnn != tv {
		for i := 1; i < TreeViewPageSteps; i++ {
			nn := fnn.MoveUp(mvMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		if selMode == mouse.SelectOne {
			fnn.SelectUpdate(selMode)
		}
		fnn.GrabFocus()
		fnn.ScrollToMe()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), fnn.This())
	}
	tv.TopUpdateEnd(wupdt)
	return fnn
}

// MovePageDownAction moves the selection up to previous TreeViewPageSteps elements in the tree,
// using given select mode (from keyboard modifiers) -- and emits select event for newly selected item
func (tv *TreeView) MovePageDownAction(selMode mouse.SelectModes) *TreeView {
	wupdt := tv.TopUpdateStart()
	mvMode := selMode
	if selMode == mouse.SelectOne {
		mvMode = mouse.NoSelect
	} else if selMode == mouse.ExtendContinuous || selMode == mouse.ExtendOne {
		mvMode = mouse.SelectQuiet
	}
	fnn := tv.MoveDown(mvMode)
	if fnn != nil && fnn != tv {
		for i := 1; i < TreeViewPageSteps; i++ {
			nn := fnn.MoveDown(mvMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		if selMode == mouse.SelectOne {
			fnn.SelectUpdate(selMode)
		}
		fnn.GrabFocus()
		fnn.ScrollToMe()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), fnn.This())
	}
	tv.TopUpdateEnd(wupdt)
	return fnn
}

// MoveToLastChild moves to the last child under me, using given select mode
// (from keyboard modifiers)
func (tv *TreeView) MoveToLastChild(selMode mouse.SelectModes) *TreeView {
	if tv.Par == nil || tv == tv.RootView {
		return nil
	}
	if !tv.IsClosed() && tv.HasChildren() {
		nnk, err := tv.Children().ElemFromEndTry(0)
		if err == nil {
			nn := nnk.Embed(KiT_TreeView).(*TreeView)
			return nn.MoveToLastChild(selMode)
		}
	} else {
		tv.SelectUpdate(selMode)
		return tv
	}
	return nil
}

// MoveHomeAction moves the selection up to top of the tree,
// using given select mode (from keyboard modifiers)
// and emits select event for newly selected item
func (tv *TreeView) MoveHomeAction(selMode mouse.SelectModes) *TreeView {
	tv.RootView.SelectUpdate(selMode)
	tv.RootView.GrabFocus()
	tv.RootView.ScrollToMe()
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), tv.RootView.This())
	return tv.RootView
}

// MoveEndAction moves the selection to the very last node in the tree
// using given select mode (from keyboard modifiers) -- and emits select event
// for newly selected item
func (tv *TreeView) MoveEndAction(selMode mouse.SelectModes) *TreeView {
	wupdt := tv.TopUpdateStart()
	mvMode := selMode
	if selMode == mouse.SelectOne {
		mvMode = mouse.NoSelect
	} else if selMode == mouse.ExtendContinuous || selMode == mouse.ExtendOne {
		mvMode = mouse.SelectQuiet
	}
	fnn := tv.MoveDown(mvMode)
	if fnn != nil && fnn != tv {
		for {
			nn := fnn.MoveDown(mvMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		if selMode == mouse.SelectOne {
			fnn.SelectUpdate(selMode)
		}
		fnn.GrabFocus()
		fnn.ScrollToMe()
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), fnn.This())
	}
	tv.TopUpdateEnd(wupdt)
	return fnn
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
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewOpened), tv.This())
		tv.UpdateEnd(updt)
	} else if !tv.HasChildren() {
		// non-children nodes get double-click open for example
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewOpened), tv.This())
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

// OpenAll opens the given node and all of its sub-nodes
func (tv *TreeView) OpenAll() {
	wupdt := tv.TopUpdateStart()
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	tv.FuncDownMeFirst(0, tv.This(), func(k ki.Ki, level int, d interface{}) bool {
		tvki := k.Embed(KiT_TreeView)
		if tvki != nil {
			tvki.(*TreeView).SetClosedState(false)
		}
		return ki.Continue
	})
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewOpened), tv.This())
	tv.UpdateEnd(updt)
	tv.TopUpdateEnd(wupdt)
}

// CloseAll closes the given node and all of its sub-nodes
func (tv *TreeView) CloseAll() {
	wupdt := tv.TopUpdateStart()
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	tv.FuncDownMeFirst(0, tv.This(), func(k ki.Ki, level int, d interface{}) bool {
		tvki := k.Embed(KiT_TreeView)
		if tvki != nil {
			tvki.(*TreeView).SetClosedState(true)
			return ki.Continue
		}
		return ki.Break
	})
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewClosed), tv.This())
	tv.UpdateEnd(updt)
	tv.TopUpdateEnd(wupdt)
}

// OpenParents opens all the parents of this node, so that it will be visible
func (tv *TreeView) OpenParents() {
	wupdt := tv.TopUpdateStart()
	updt := tv.RootView.UpdateStart()
	tv.RootView.SetFullReRender()
	tv.FuncUpParent(0, tv.This(), func(k ki.Ki, level int, d interface{}) bool {
		tvki := k.Embed(KiT_TreeView)
		if tvki != nil {
			tvki.(*TreeView).SetClosedState(false)
			return ki.Continue
		}
		return ki.Break
	})
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewOpened), tv.This())
	tv.RootView.UpdateEnd(updt)
	tv.TopUpdateEnd(wupdt)
}

// FindSrcNode finds TreeView node for given source node, or nil if not found
func (tv *TreeView) FindSrcNode(kn ki.Ki) *TreeView {
	var ttv *TreeView
	tv.FuncDownMeFirst(0, tv.This(), func(k ki.Ki, level int, d interface{}) bool {
		tvki := k.Embed(KiT_TreeView)
		if tvki != nil {
			tvk := tvki.(*TreeView)
			if tvk.SrcNode == kn {
				ttv = tvk
				return ki.Break
			}
		}
		return ki.Continue
	})
	return ttv
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
	// note: root inactivity is relevant factor here..
	if CtxtMenuView(tv.SrcNode, tv.RootIsInactive(), tv.Viewport, m) { // our viewed obj's menu
		if tv.ShowViewCtxtMenu {
			m.AddSeparator("sep-tvmenu")
			CtxtMenuView(tv.This(), tv.RootIsInactive(), tv.Viewport, m)
		}
	} else {
		CtxtMenuView(tv.This(), tv.RootIsInactive(), tv.Viewport, m)
	}
}

// IsRootOrField returns true if given node is either the root of the
// tree or a field -- various operations can not be performed on these -- if
// string is passed, then a prompt dialog is presented with that as the name
// of the operation being attempted -- otherwise it silently returns (suitable
// for context menu UpdateFunc).
func (tv *TreeView) IsRootOrField(op string) bool {
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView IsRootOrField nil SrcNode in: %v\n", tv.Path())
		return false
	}
	if sk.IsField() {
		if op != "" {
			gi.PromptDialog(tv.Viewport, gi.DlgOpts{Title: "TreeView " + op, Prompt: fmt.Sprintf("Cannot %v fields", op)}, gi.AddOk, gi.NoCancel, nil, nil)
		}
		return true
	}
	if tv.This() == tv.RootView.This() {
		if op != "" {
			gi.PromptDialog(tv.Viewport, gi.DlgOpts{Title: "TreeView " + op, Prompt: fmt.Sprintf("Cannot %v the root of the tree", op)}, gi.AddOk, gi.NoCancel, nil, nil)
		}
		return true
	}
	return false
}

// SrcInsertAfter inserts a new node in the source tree after this node, at
// the same (sibling) level, prompting for the type of node to insert
func (tv *TreeView) SrcInsertAfter() {
	tv.SrcInsertAt(1, "Insert After")
}

// SrcInsertBefore inserts a new node in the source tree before this node, at
// the same (sibling) level, prompting for the type of node to insert
func (tv *TreeView) SrcInsertBefore() {
	tv.SrcInsertAt(0, "Insert Before")
}

// SrcInsertAt inserts a new node in the source tree at given relative offset
// from this node, at the same (sibling) level, prompting for the type of node to insert
func (tv *TreeView) SrcInsertAt(rel int, actNm string) {
	if tv.IsRootOrField(actNm) {
		return
	}
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", actNm, tv.Path())
		return
	}
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	myidx += rel
	gi.NewKiDialog(tv.Viewport, sk.BaseIface(),
		gi.DlgOpts{Title: actNm, Prompt: "Number and Type of Items to Insert:"},
		tv.Par.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				tvv, _ := recv.Embed(KiT_TreeView).(*TreeView)
				par := tvv.SrcNode
				dlg, _ := send.(*gi.Dialog)
				n, typ := gi.NewKiDialogValues(dlg)
				updt := par.UpdateStart()
				var ski ki.Ki
				for i := 0; i < n; i++ {
					nm := fmt.Sprintf("New%v%v", typ.Name(), myidx+rel+i)
					par.SetChildAdded()
					nki := par.InsertNewChild(typ, myidx+i, nm)
					if i == n-1 {
						ski = nki
					}
					tv.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), nki.This())
				}
				tvv.SetChanged()
				par.UpdateEnd(updt)
				if ski != nil {
					if tvk := tvv.ChildByName("tv_"+ski.Name(), 0); tvk != nil {
						stv, _ := tvk.Embed(KiT_TreeView).(*TreeView)
						stv.SelectAction(mouse.SelectOne)
					}
				}
			}
		})
}

// SrcAddChild adds a new child node to this one in the source tree,
// prompting the user for the type of node to add
func (tv *TreeView) SrcAddChild() {
	ttl := "Add Child"
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", ttl, tv.Path())
		return
	}
	gi.NewKiDialog(tv.Viewport, sk.BaseIface(),
		gi.DlgOpts{Title: ttl, Prompt: "Number and Type of Items to Add:"},
		tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				tvv, _ := recv.Embed(KiT_TreeView).(*TreeView)
				sk := tvv.SrcNode
				dlg, _ := send.(*gi.Dialog)
				n, typ := gi.NewKiDialogValues(dlg)
				updt := sk.UpdateStart()
				sk.SetChildAdded()
				var ski ki.Ki
				for i := 0; i < n; i++ {
					nm := fmt.Sprintf("New%v%v", typ.Name(), i)
					nki := sk.AddNewChild(typ, nm)
					if i == n-1 {
						ski = nki
					}
					tv.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), nki.This())
				}
				tvv.SetChanged()
				sk.UpdateEnd(updt)
				if ski != nil {
					tvv.Open()
					if tvk := tvv.ChildByName("tv_"+ski.Name(), 0); tvk != nil {
						stv, _ := tvk.Embed(KiT_TreeView).(*TreeView)
						stv.SelectAction(mouse.SelectOne)
					}
				}
			}
		})
}

// SrcDelete deletes the source node corresponding to this view node in the source tree
func (tv *TreeView) SrcDelete() {
	ttl := "Delete"
	if tv.IsRootOrField(ttl) {
		return
	}
	if tv.MoveDown(mouse.SelectOne) == nil {
		tv.MoveUp(mouse.SelectOne)
	}
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", ttl, tv.Path())
		return
	}
	sk.Delete(true)
	tv.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewDeleted), sk.This())
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
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", ttl, tv.Path())
		return
	}
	if tv.Par == nil {
		return
	}
	tvpar := tv.Par.Embed(KiT_TreeView).(*TreeView)
	par := tvpar.SrcNode
	if par == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", ttl, tvpar.Path())
		return
	}
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	updt := par.UpdateStart()
	nm := fmt.Sprintf("%v_Copy", sk.Name())
	nwkid := sk.Clone()
	nwkid.SetName(nm)
	par.SetChildAdded()
	par.InsertChild(nwkid, myidx+1)
	tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), nwkid.This())
	par.UpdateEnd(updt)
	tvpar.SetChanged()
	if tvk := tvpar.ChildByName("tv_"+nm, 0); tvk != nil {
		stv, _ := tvk.Embed(KiT_TreeView).(*TreeView)
		stv.SelectAction(mouse.SelectOne)
	}
}

// SrcEdit pulls up a StructViewDialog window on the source object viewed by this node
func (tv *TreeView) SrcEdit() {
	if tv.SrcNode == nil {
		log.Printf("TreeView SrcEdit nil SrcNode in: %v\n", tv.Path())
		return
	}
	tynm := kit.NonPtrType(ki.Type(tv.SrcNode)).Name()
	StructViewDialog(tv.Viewport, tv.SrcNode, DlgOpts{Title: tynm}, nil, nil)
}

// SrcGoGiEditor pulls up a new GoGiEditor window on the source object viewed by this node
func (tv *TreeView) SrcGoGiEditor() {
	if tv.SrcNode == nil {
		log.Printf("TreeView SrcGoGiEditor nil SrcNode in: %v\n", tv.Path())
		return
	}
	GoGiEditorDialog(tv.SrcNode)
}

//////////////////////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeData adds mimedata for this node: a text/plain of the Path, and
// an application/json of the source node.
// satisfies Clipper.MimeData interface
func (tv *TreeView) MimeData(md *mimedata.Mimes) {
	sroot := tv.RootView.SrcNode
	src := tv.SrcNode
	*md = append(*md, mimedata.NewTextData(src.PathFrom(sroot)))
	var buf bytes.Buffer
	err := src.WriteJSON(&buf, ki.Indent) // true = pretty for clipboard..
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: filecat.DataJson, Data: buf.Bytes()})
	} else {
		log.Printf("gi.TreeView MimeData SaveJSON error: %v\n", err)
	}
}

// NodesFromMimeData creates a slice of Ki node(s) from given mime data
// and also a corresponding slice of original paths
func (tv *TreeView) NodesFromMimeData(md mimedata.Mimes) (ki.Slice, []string) {
	ni := len(md) / 2
	sl := make(ki.Slice, 0, ni)
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == filecat.DataJson {
			nki, err := ki.ReadNewJSON(bytes.NewReader(d.Data))
			if err == nil {
				sl = append(sl, nki)
			} else {
				log.Printf("TreeView NodesFromMimeData: JSON load error: %v\n", err)
			}
		} else if d.Type == filecat.TextPlain { // paths
			pl = append(pl, string(d.Data))
		}
	}
	return sl, pl
}

// Copy copies to clip.Board, optionally resetting the selection
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Copy(reset bool) {
	sels := tv.SelectedViews()
	nitms := ints.MaxInt(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.This().(gi.Clipper).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This() != tv.This() {
				sn.This().(gi.Clipper).MimeData(&md)
			}
		}
	}
	oswin.TheApp.ClipBoard(tv.ParentWindow().OSWin).Write(md)
	if reset {
		tv.UnselectAll()
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

// Paste pastes clipboard at given node
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Paste() {
	md := oswin.TheApp.ClipBoard(tv.ParentWindow().OSWin).Read([]string{filecat.DataJson})
	if md != nil {
		tv.PasteMenu(md)
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
	if tv.SrcNode == nil {
		log.Printf("TreeView PasteMenu nil SrcNode in: %v\n", tv.Path())
		return
	}
	var men gi.Menu
	tv.MakePasteMenu(&men, md)
	pos := tv.ContextMenuPos()
	gi.PopupMenu(men, pos.X, pos.Y, tv.Viewport, "tvPasteMenu")
}

// PasteAssign assigns mime data (only the first one!) to this node
func (tv *TreeView) PasteAssign(md mimedata.Mimes) {
	sl, _ := tv.NodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView PasteAssign nil SrcNode in: %v\n", tv.Path())
		return
	}
	sk.CopyFrom(sl[0])
	tv.SetChanged()
}

// PasteBefore inserts object(s) from mime data before this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeView) PasteBefore(md mimedata.Mimes, mod dnd.DropMods) {
	tv.PasteAt(md, mod, 0, "Paste Before")
}

// PasteAfter inserts object(s) from mime data after this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeView) PasteAfter(md mimedata.Mimes, mod dnd.DropMods) {
	tv.PasteAt(md, mod, 1, "Paste After")
}

// This is a kind of hack to prevent moved items from being deleted, using DND
const TreeViewTempMovedTag = `_\&MOVED\&`

// PasteAt inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeView) PasteAt(md mimedata.Mimes, mod dnd.DropMods, rel int, actNm string) {
	sl, pl := tv.NodesFromMimeData(md)

	if tv.Par == nil {
		return
	}
	tvpar := tv.Par.Embed(KiT_TreeView).(*TreeView)
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", actNm, tv.Path())
		return
	}
	par := sk.Parent()
	if par == nil {
		gi.PromptDialog(tv.Viewport, gi.DlgOpts{Title: actNm, Prompt: "Cannot insert after the root of the tree"}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	myidx += rel
	sroot := tv.RootView.SrcNode
	updt := par.UpdateStart()
	sz := len(sl)
	var ski ki.Ki
	for i, ns := range sl {
		orgpath := pl[i]
		if mod != dnd.DropMove {
			if cn := par.ChildByName(ns.Name(), 0); cn != nil {
				ns.SetName(ns.Name() + "_Copy")
			}
		}
		par.SetChildAdded()
		par.InsertChild(ns, myidx+i)
		npath := ns.PathFrom(sroot)
		if mod == dnd.DropMove && npath == orgpath { // we will be nuked immediately after drag
			ns.SetName(ns.Name() + TreeViewTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			ski = ns
		}
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), ns.This())
	}
	par.UpdateEnd(updt)
	tvpar.SetChanged()
	if ski != nil {
		if tvk := tvpar.ChildByName("tv_"+ski.Name(), 0); tvk != nil {
			stv, _ := tvk.Embed(KiT_TreeView).(*TreeView)
			stv.SelectAction(mouse.SelectOne)
		}
	}
}

// PasteChildren inserts object(s) from mime data at end of children of this
// node
func (tv *TreeView) PasteChildren(md mimedata.Mimes, mod dnd.DropMods) {
	sl, _ := tv.NodesFromMimeData(md)

	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView PasteChildren nil SrcNode in: %v\n", tv.Path())
		return
	}
	updt := sk.UpdateStart()
	sk.SetChildAdded()
	for _, ns := range sl {
		sk.AddChild(ns)
		tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), ns.This())
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
	tv.This().(gi.Clipper).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This() != tv.This() {
				sn.This().(gi.Clipper).MimeData(&md)
			}
		}
	}
	sp := &gi.Sprite{}
	sp.GrabRenderFrom(tv) // todo: show number of items?
	gi.ImageClearer(sp.Pixels, 50.0)
	tv.ParentWindow().StartDragNDrop(tv.This(), md, sp)
}

// DragNDropTarget handles a drag-n-drop onto this node
func (tv *TreeView) DragNDropTarget(de *dnd.Event) {
	de.Target = tv.This()
	if de.Mod == dnd.DropLink {
		de.Mod = dnd.DropCopy // link not supported -- revert to copy
	}
	de.SetProcessed()
	tv.This().(gi.DragNDropper).Drop(de.Data, de.Mod)
}

// DragNDropExternal handles a drag-n-drop external drop onto this node
func (tv *TreeView) DragNDropExternal(de *dnd.Event) {
	de.Target = tv.This()
	if de.Mod == dnd.DropLink {
		de.Mod = dnd.DropCopy // link not supported -- revert to copy
	}
	de.SetProcessed()
	tv.This().(gi.DragNDropper).DropExternal(de.Data, de.Mod)
}

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore
func (tv *TreeView) DragNDropFinalize(mod dnd.DropMods) {
	if tv.Viewport == nil {
		return
	}
	tv.UnselectAll()
	tv.ParentWindow().FinalizeDragNDrop(mod)
}

// DragNDropFinalizeDefMod is called to finalize actions on the Source node prior to
// performing target actions -- uses default drop mod in place when event was dropped.
func (tv *TreeView) DragNDropFinalizeDefMod() {
	win := tv.ParentWindow()
	if win == nil {
		return
	}
	tv.UnselectAll()
	win.FinalizeDragNDrop(win.EventMgr.DNDDropMod)
}

// Dragged is called after target accepts the drop -- we just remove
// elements that were moved
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (tv *TreeView) Dragged(de *dnd.Event) {
	if de.Mod != dnd.DropMove {
		return
	}
	sroot := tv.RootView.SrcNode
	md := de.Data
	for _, d := range md {
		if d.Type == filecat.TextPlain { // link
			path := string(d.Data)
			sn := sroot.FindPath(path)
			if sn != nil {
				sn.Delete(true)
				tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewDeleted), sn.This())
			}
			sn = sroot.FindPath(path + TreeViewTempMovedTag)
			if sn != nil {
				psplt := strings.Split(path, "/")
				orgnm := psplt[len(psplt)-1]
				sn.SetName(orgnm)
				sn.UpdateSig()
			}
		}
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

// DropExternal is not handled by base case but could be in derived
func (tv *TreeView) DropExternal(md mimedata.Mimes, mod dnd.DropMods) {
	tv.DropCancel()
}

// DropAssign assigns mime data (only the first one!) to this node
func (tv *TreeView) DropAssign(md mimedata.Mimes) {
	tv.PasteAssign(md)
	tv.DragNDropFinalize(dnd.DropCopy)
}

// DropBefore inserts object(s) from mime data before this node
func (tv *TreeView) DropBefore(md mimedata.Mimes, mod dnd.DropMods) {
	tv.PasteBefore(md, mod)
	tv.DragNDropFinalize(mod)
}

// DropAfter inserts object(s) from mime data after this node
func (tv *TreeView) DropAfter(md mimedata.Mimes, mod dnd.DropMods) {
	tv.PasteAfter(md, mod)
	tv.DragNDropFinalize(mod)
}

// DropChildren inserts object(s) from mime data at end of children of this node
func (tv *TreeView) DropChildren(md mimedata.Mimes, mod dnd.DropMods) {
	tv.PasteChildren(md, mod)
	tv.DragNDropFinalize(mod)
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
	if ki.TypeEmbeds(tv.Par, KiT_TreeView) {
		return tv.Par.Embed(KiT_TreeView).(*TreeView)
	}
	// I am rootview!
	return nil
}

// RootTreeView returns the root node of TreeView tree -- typically cached in
// RootView on each node, but this can be used if that cached value needs
// to be updated for any reason.
func (tv *TreeView) RootTreeView() *TreeView {
	rn := tv
	tv.FuncUp(0, tv.This(), func(k ki.Ki, level int, d interface{}) bool {
		_, pg := gi.KiToNode2D(k)
		if pg == nil {
			return false
		}
		if ki.TypeEmbeds(k, KiT_TreeView) {
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
		fmt.Printf("TreeView KeyInput: %v\n", tv.Path())
	}
	kf := gi.KeyFun(kt.Chord())
	selMode := mouse.SelectModeBits(kt.Modifiers)

	if selMode == mouse.SelectOne {
		if tv.SelectMode() {
			selMode = mouse.ExtendContinuous
		}
	}

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
	case gi.KeyFunHome:
		tv.MoveHomeAction(selMode)
		kt.SetProcessed()
	case gi.KeyFunEnd:
		tv.MoveEndAction(selMode)
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
		tv.This().(gi.Clipper).Copy(true)
		kt.SetProcessed()
	}
	if !tv.RootIsInactive() && !kt.IsProcessed() {
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
			tv.This().(gi.Clipper).Cut()
			kt.SetProcessed()
		case gi.KeyFunPaste:
			tv.This().(gi.Clipper).Paste()
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
		if recv == nil {
			return
		}
		de := d.(*dnd.Event)
		tvv := recv.Embed(KiT_TreeView).(*TreeView)
		switch de.Action {
		case dnd.Start:
			tvv.DragNDropStart()
		case dnd.DropOnTarget:
			tvv.DragNDropTarget(de)
		case dnd.DropFmSource:
			tvv.This().(gi.DragNDropper).Dragged(de)
		case dnd.External:
			tvv.DragNDropExternal(de)
		}
	})
	tv.ConnectEvent(oswin.DNDFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		if recv == nil {
			return
		}
		de := d.(*dnd.FocusEvent)
		tvv := recv.Embed(KiT_TreeView).(*TreeView)
		switch de.Action {
		case dnd.Enter:
			tvv.ParentWindow().DNDSetCursor(de.Mod)
		case dnd.Exit:
			tvv.ParentWindow().DNDNotCursor()
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
			if tvvi == nil || tvvi.This() == nil { // deleted
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
	if icc := tv.Parts.ChildByName("branch", 0); icc != nil {
		return icc.(*gi.CheckBox), true
	}
	return nil, false
}

// IconPart returns the icon in parts, if it exists
func (tv *TreeView) IconPart() (*gi.Icon, bool) {
	if icc := tv.Parts.ChildByName("icon", 1); icc != nil {
		return icc.(*gi.Icon), true
	}
	return nil, false
}

// LabelPart returns the label in parts, if it exists
func (tv *TreeView) LabelPart() (*gi.Label, bool) {
	if lbl := tv.Parts.ChildByName("label", 1); lbl != nil {
		return lbl.(*gi.Label), true
	}
	return nil, false
}

func (tv *TreeView) ConfigParts() {
	tv.Parts.Lay = gi.LayoutHoriz
	tv.Parts.Sty.Template = "giv.TreeView.Parts"
	config := kit.TypeAndNameList{}
	if tv.HasChildren() {
		config.Add(gi.KiT_CheckBox, "branch")
	}
	if tv.Icon.IsValid() {
		config.Add(gi.KiT_Icon, "icon")
	}
	config.Add(gi.KiT_Label, "label")
	mods, updt := tv.Parts.ConfigChildren(config)
	if tv.HasChildren() {
		if wb, ok := tv.BranchPart(); ok {
			if wb.Sty.Template != "giv.TreeView.Branch" {
				wb.SetProp("#icon0", TVBranchProps)
				wb.SetProp("#icon1", TVBranchProps)
				wb.SetProp("no-focus", true) // note: cannot be in compiled props
				wb.Sty.Template = "giv.TreeView.Branch"
				// unfortunately StylePart only handles default Style obj -- not
				// these special styles.. todo: fix this somehow
				if bprpi, err := tv.PropTry("#branch"); err == nil {
					switch pr := bprpi.(type) {
					case map[string]interface{}:
						wb.SetIconProps(ki.Props(pr))
					case ki.Props:
						wb.SetIconProps(pr)
					}
				} else {
					tprops := *kit.Types.Properties(ki.Type(tv), true) // true = makeNew
					if bprpi, ok := kit.TypeProp(tprops, gi.WidgetDefPropsKey+"#branch"); ok {
						switch pr := bprpi.(type) {
						case map[string]interface{}:
							wb.SetIconProps(ki.Props(pr))
						case ki.Props:
							wb.SetIconProps(pr)
						}
					}
				}
				tv.StylePart(gi.Node2D(wb))
				wb.Style2D() // this is key for getting styling to take effect on first try
			}
		}
	}
	if tv.Icon.IsValid() {
		if ic, ok := tv.IconPart(); ok {
			// this only works after a second redraw..
			// ic.Sty.Template = "giv.TreeView.Icon"
			set, _ := ic.SetIcon(string(tv.Icon))
			if set || tv.NeedsFullReRender() || tv.RootView.NeedsFullReRender() || mods {
				tv.StylePart(gi.Node2D(ic))
			}
		}
	}
	if lbl, ok := tv.LabelPart(); ok {
		// this does not work! even with redraws
		// lbl.Sty.Template = "giv.TreeView.Label"
		lbl.Props = nil
		// if tv.HasFlag(int(TreeViewFlagNoTemplate)) {
		// 	lbl.Redrawable = true // this prevents select highlight from rendering properly
		// }
		tv.Sty.Font.CopyNonDefaultProps(lbl.This()) // copy our properties to label
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
	if tv.Icon.IsValid() {
		if ic, ok := tv.IconPart(); ok {
			ic.SetIcon(string(tv.Icon))
		}
	}
	if lbl, ok := tv.LabelPart(); ok {
		ltxt := tv.Label()
		if lbl.Text != ltxt {
			lbl.SetText(ltxt)
		}
	}
	if tv.HasChildren() {
		if wb, ok := tv.BranchPart(); ok {
			wb.SetChecked(!tv.IsClosed())
		}
	}
}

var TreeViewProps = ki.Props{
	"EnumType:Flag":    KiT_TreeViewFlags,
	"indent":           units.NewCh(4),
	"spacing":          units.NewCh(.5),
	"border-width":     units.NewPx(0),
	"border-radius":    units.NewPx(0),
	"padding":          units.NewPx(0),
	"margin":           units.NewPx(1),
	"text-align":       gist.AlignLeft,
	"vertical-align":   gist.AlignTop,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": "inherit",
	"#icon": ki.Props{
		"width":   units.NewEm(1),
		"height":  units.NewEm(1),
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
		"fill":    &gi.Prefs.Colors.Icon,
		"stroke":  &gi.Prefs.Colors.Font,
	},
	"#branch": ki.Props{
		"icon":             "wedge-down",
		"icon-off":         "wedge-right",
		"margin":           units.NewPx(0),
		"padding":          units.NewPx(0),
		"background-color": color.Transparent,
		"max-width":        units.NewEm(.8),
		"max-height":       units.NewEm(.8),
	},
	"#space": ki.Props{
		"width": units.NewEm(0.5),
	},
	"#label": ki.Props{
		"margin":    units.NewPx(0),
		"padding":   units.NewPx(0),
		"min-width": units.NewCh(16),
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
	TreeViewSelectors[TreeViewInactive]: ki.Props{
		"background-color": "highlight-10",
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
		{"sep-open", ki.BlankProp{}},
		{"OpenAll", ki.Props{}},
		{"CloseAll", ki.Props{}},
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
	tv.Sty.Template = "giv.TreeView." + ki.Type(tv).Name()
	tv.LayState.Defaults() // doesn't overwrite
	tv.ConfigParts()
	// tv.ConnectToViewport()
}

func (tv *TreeView) StyleTreeView() {
	tv.UpdateInactive()
	if !tv.HasChildren() {
		tv.SetClosed()
	}
	if tv.HasClosedParent() {
		tv.ClearFlag(int(gi.CanFocus))
		return
	}
	tv.StyMu.Lock()
	tv.SetCanFocusIfActive()
	hasTempl, saveTempl := false, false
	_, noTempl := tv.PropInherit("no-templates", ki.NoInherit, ki.TypeProps)
	tv.SetFlagState(noTempl, int(TreeViewFlagNoTemplate))
	if !noTempl {
		hasTempl, saveTempl = tv.Sty.FromTemplate()
	}
	if !hasTempl || saveTempl {
		tv.Style2DWidget()
	}
	if hasTempl && saveTempl {
		tv.Sty.SaveTemplate()
	}
	pst := &(tv.Par.(gi.Node2D).AsWidget().Sty)
	if hasTempl && !saveTempl {
		for i := 0; i < int(TreeViewStatesN); i++ {
			tv.StateStyles[i].Template = tv.Sty.Template + TreeViewSelectors[i]
			tv.StateStyles[i].FromTemplate()
		}
	} else {
		for i := 0; i < int(TreeViewStatesN); i++ {
			tv.StateStyles[i].CopyFrom(&tv.Sty)
			tv.StateStyles[i].SetStyleProps(pst, tv.StyleProps(TreeViewSelectors[i]), tv.Viewport)
			tv.StateStyles[i].CopyUnitContext(&tv.Sty.UnContext)
		}
	}
	if hasTempl && saveTempl {
		for i := 0; i < int(TreeViewStatesN); i++ {
			tv.StateStyles[i].Template = tv.Sty.Template + TreeViewSelectors[i]
			tv.StateStyles[i].SaveTemplate()
		}
	}
	val, has := tv.Props["open-depth"]
	if has {
		if iv, ok := kit.ToInt(val); ok {
			tv.OpenDepth = int(iv)
		}
	}
	tv.Indent.SetFmInheritProp("indent", tv.This(), ki.NoInherit, ki.TypeProps)
	tv.Indent.ToDots(&tv.Sty.UnContext)
	tv.Parts.Sty.InheritFields(&tv.Sty)
	if spc, ok := tv.PropInherit("spacing", ki.NoInherit, ki.TypeProps); ok {
		tv.Parts.SetProp("spacing", spc) // parts is otherwise not typically styled
	}
	tv.StyMu.Unlock()
	tv.ConfigParts()
}

func (tv *TreeView) Style2D() {
	tv.StyleTreeView()
	tv.LayState.SetFromStyle(&tv.Sty.Layout) // also does reset
}

// TreeView is tricky for alloc because it is both a layout of its children but has to
// maintain its own bbox for its own widget.

func (tv *TreeView) Size2D(iter int) {
	tv.InitLayout2D()
	if tv.HasClosedParent() {
		return // nothing
	}
	tv.SizeFromParts(iter) // get our size from parts
	tv.WidgetSize = tv.LayState.Alloc.Size
	h := mat32.Ceil(tv.WidgetSize.Y)
	w := tv.WidgetSize.X

	if !tv.IsClosed() {
		// we layout children under us
		for _, kid := range tv.Kids {
			gis := kid.(gi.Node2D).AsWidget()
			if gis == nil || gis.This() == nil {
				continue
			}
			h += mat32.Ceil(gis.LayState.Alloc.Size.Y)
			w = mat32.Max(w, tv.Indent.Dots+gis.LayState.Alloc.Size.X)
		}
	}
	tv.LayState.Alloc.Size = mat32.Vec2{w, h}
	tv.WidgetSize.X = w // stretch
}

func (tv *TreeView) Layout2DParts(parBBox image.Rectangle, iter int) {
	spc := tv.Sty.BoxSpace()
	tv.Parts.LayState.Alloc.Pos = tv.LayState.Alloc.Pos.AddScalar(spc)
	tv.Parts.LayState.Alloc.PosOrig = tv.Parts.LayState.Alloc.Pos
	tv.Parts.LayState.Alloc.Size = tv.WidgetSize.AddScalar(-2.0 * spc)
	tv.Parts.Layout2D(parBBox, iter)
}

func (tv *TreeView) Layout2D(parBBox image.Rectangle, iter int) bool {
	if tv.HasClosedParent() {
		tv.LayState.Alloc.PosRel.X = -1000000 // put it very far off screen..
	}
	tv.ConfigPartsIfNeeded()

	psize := tv.AddParentPos() // have to add our pos first before computing below:

	rn := tv.RootView
	// our alloc size is root's size minus our total indentation
	tv.LayState.Alloc.Size.X = rn.LayState.Alloc.Size.X - (tv.LayState.Alloc.Pos.X - rn.LayState.Alloc.Pos.X)
	tv.WidgetSize.X = tv.LayState.Alloc.Size.X

	tv.LayState.Alloc.PosOrig = tv.LayState.Alloc.Pos
	gi.SetUnitContext(&tv.Sty, tv.Viewport, psize) // update units with final layout
	for i := 0; i < int(TreeViewStatesN); i++ {
		tv.StateStyles[i].CopyUnitContext(&tv.Sty.UnContext)
	}
	tv.BBox = tv.This().(gi.Node2D).BBox2D() // only compute once, at this point
	tv.This().(gi.Node2D).ComputeBBox2D(parBBox, image.ZP)

	if gi.Layout2DTrace {
		fmt.Printf("Layout: %v reduced X allocsize: %v rn: %v  pos: %v rn pos: %v\n", tv.Path(), tv.WidgetSize.X, rn.LayState.Alloc.Size.X, tv.LayState.Alloc.Pos.X, rn.LayState.Alloc.Pos.X)
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", tv.Path(), tv.LayState.Alloc.Pos, tv.LayState.Alloc.Size, tv.VpBBox, tv.WinBBox)
	}

	tv.Layout2DParts(parBBox, iter) // use OUR version
	h := mat32.Ceil(tv.WidgetSize.Y)
	if !tv.IsClosed() {
		for _, kid := range tv.Kids {
			if kid == nil || kid.This() == nil {
				continue
			}
			ni := kid.(gi.Node2D).AsWidget()
			if ni == nil {
				continue
			}
			ni.LayState.Alloc.PosRel.Y = h
			ni.LayState.Alloc.PosRel.X = tv.Indent.Dots
			h += mat32.Ceil(ni.LayState.Alloc.Size.Y)
		}
	}
	return tv.Layout2DChildren(iter)
}

func (tv *TreeView) BBox2D() image.Rectangle {
	// we have unusual situation of bbox != alloc
	tp := tv.LayState.Alloc.PosOrig.ToPointFloor()
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

func (tv *TreeView) IsVisible() bool {
	if tv == nil || tv.This() == nil || tv.Viewport == nil {
		return false
	}
	if tv.RootView == nil || tv.RootView.This() == nil {
		return false
	}
	if tv.RootView.Par == nil || tv.RootView.Par.This() == nil {
		return false
	}
	if tv.This() == tv.RootView.This() { // root is ALWAYS visible so updates there work
		return true
	}
	if tv.IsInvisible() {
		return false
	}
	return tv.RootView.Par.This().(gi.Node2D).IsVisible()
}

func (tv *TreeView) PushBounds() bool {
	if tv == nil || tv.This() == nil {
		return false
	}
	if !tv.This().(gi.Node2D).IsVisible() {
		return false
	}
	if tv.VpBBox.Empty() && tv.This() != tv.RootView.This() { // root must always connect!
		tv.ClearFullReRender()
		return false
	}
	rs := tv.Render()
	rs.PushBounds(tv.VpBBox)
	tv.ConnectToViewport()
	if gi.Render2DTrace {
		fmt.Printf("Render: %v at %v\n", tv.Path(), tv.VpBBox)
	}
	return true
}

func (tv *TreeView) Render2D() {
	if tv.HasClosedParent() {
		tv.DisconnectAllEvents(gi.AllPris)
		return // nothing
	}
	// restyle on re-render -- this is not actually necessary
	// if tv.HasFlag(int(TreeViewFlagNoTemplate)) && (tv.NeedsFullReRender() || tv.RootView.NeedsFullReRender()) {
	// 	fmt.Printf("restyle: %v\n", tv.Nm)
	// 	tv.StyleTreeView()
	// 	tv.ConfigParts()
	// }
	// fmt.Printf("tv rend: %v\n", tv.Nm)
	if tv.PushBounds() {
		if !tv.VpBBox.Empty() { // we are root and just here for the connections :)
			tv.UpdateInactive()
			if tv.IsSelected() {
				tv.Sty = tv.StateStyles[TreeViewSel]
			} else if tv.HasFocus() {
				tv.Sty = tv.StateStyles[TreeViewFocus]
			} else if tv.IsInactive() {
				tv.Sty = tv.StateStyles[TreeViewInactive]
			} else {
				tv.Sty = tv.StateStyles[TreeViewActive]
			}
			tv.ConfigPartsIfNeeded()
			tv.This().(gi.Node2D).ConnectEvents2D()

			// note: this is std except using WidgetSize instead of AllocSize
			rs, pc, st := tv.RenderLock()
			pc.FontStyle = st.Font
			pc.StrokeStyle.SetColor(&st.Border.Color)
			pc.StrokeStyle.Width = st.Border.Width
			pc.FillStyle.SetColorSpec(&st.Font.BgColor)
			// tv.RenderStdBox()
			pos := tv.LayState.Alloc.Pos.AddScalar(st.Layout.Margin.Dots)
			sz := tv.WidgetSize.AddScalar(-2.0 * st.Layout.Margin.Dots)
			tv.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
			tv.RenderUnlock(rs)
			tv.Render2DParts()
		}
		tv.PopBounds()
	} else {
		tv.DisconnectAllEvents(gi.AllPris)
	}
	// we always have to render our kids b/c we could be out of scope but they could be in!
	tv.Render2DChildren()
	tv.ClearFullReRender()
}

func (tv *TreeView) ConnectEvents2D() {
	tv.TreeViewEvents()
}

func (tv *TreeView) FocusChanged2D(change gi.FocusChanges) {
	switch change {
	case gi.FocusLost:
		tv.UpdateSig()
	case gi.FocusGot:
		if tv.This() == tv.RootView.This() {
			sl := tv.SelectedViews()
			if len(sl) > 0 {
				fsl := sl[0]
				if fsl != tv {
					fsl.GrabFocus()
					return
				}
			}
		}
		tv.ScrollToMe()
		tv.EmitFocusedSignal()
		tv.UpdateSig()
	case gi.FocusInactive: // don't care..
	case gi.FocusActive:
	}
}
