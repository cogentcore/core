// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"reflect"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
)

////////////////////////////////////////////////////////////////////////////////////////
// 2D  Nodes

/*
Base struct node for 2D rendering tree -- renders to a bitmap using Paint
rendering functions operating on the RenderState in the parent Viewport

For Widget / Layout nodes, rendering is done in 5 separate passes:

	0. Init2D: In a MeFirst downward pass, Viewport pointer is set, styles are
	initialized, and any other widget-specific init is done.

	1. Style2D: In a MeFirst downward pass, all properties are cached out in
	an inherited manner, and incorporating any css styles, into either the
	Paint (SVG) or Style (Widget) object for each Node.  Only done once after
	structural changes -- styles are not for dynamic changes.

	2. Size2D: DepthFirst downward pass, each node first calls
	g.Layout.Reset(), then sets their LayoutSize according to their own
	intrinsic size parameters, and/or those of its children if it is a Layout.

	3. Layout2D: MeFirst downward pass (each node calls on its children at
	appropriate point) with relevant parent BBox that the children are
	constrained to render within -- they then intersect this BBox with their
	own BBox (from BBox2D) -- typically just call Layout2DBase for default
	behavior -- and add parent position to AllocPos. Layout does all its
	sizing and positioning of children in this pass, based on the Size2D data
	gathered bottom-up and constraints applied top-down from higher levels.
	Typically only a single iteration is required but multiple are supported
	(needed for word-wrapped text or flow layouts).

	4. Render2D: Final rendering pass, each node is fully responsible for
	rendering its own children, to provide maximum flexibility (see
	Render2DChildren) -- bracket the render calls in PushBounds / PopBounds
	and a false from PushBounds indicates that VpBBox is empty and no
	rendering should occur.  Nodes typically connect / disconnect to receive
	events from the window based on this visibility here.

    * Move2D: optional pass invoked by scrollbars to move elements relative to
      their previously-assigned positions.

    * SVG nodes skip the Size and Layout passes, and render directly into
      parent SVG viewport

*/
type Node2DBase struct {
	NodeBase
	Viewport *Viewport2D `json:"-" xml:"-" view:"-" desc:"our viewport -- set in Init2D (Base typically) and used thereafter"`
}

var KiT_Node2DBase = kit.Types.AddType(&Node2DBase{}, Node2DBaseProps)

var Node2DBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

// Update2DTrace reports a trace of updates that trigger re-rendering -- can be set in PrefsDebug from prefs gui
var Update2DTrace bool = false

// Render2DTrace reports a trace of the nodes rendering
// (just printfs to stdout) -- can be set in PrefsDebug from prefs gui
var Render2DTrace bool = false

// Layout2DTrace reports a trace of all layouts (just
// printfs to stdout) -- can be set in PrefsDebug from prefs gui
var Layout2DTrace bool = false

// Node2D is the interface for all 2D nodes -- defines the stages of building
// and rendering the 2D scenegraph
type Node2D interface {
	// nodes are Ki elements -- this comes for free by embedding ki.Node in
	// all Node2D elements.
	ki.Ki

	// AsNode2D returns a generic Node2DBase for our node -- gives generic
	// access to all the base-level data structures without requiring
	// interface methods.
	AsNode2D() *Node2DBase

	// AsViewport2D returns Viewport2D if this node is one (has its own
	// bitmap, used for menus, dialogs, icons, etc), else nil.
	AsViewport2D() *Viewport2D

	// AsLayout2D returns Layout if this is a Layout-derived node, else nil
	AsLayout2D() *Layout

	// AsWidget returns WidgetBase if this is a WidgetBase-derived node, else nil.
	AsWidget() *WidgetBase

	// Init2D initializes a node -- grabs active Viewport etc -- must call
	// InitNodeBase as first step set basic inits including setting Viewport
	// -- all code here must be robust to being called repeatedly.
	Init2D()

	// Style2D: In a MeFirst downward pass, all properties are cached out in
	// an inherited manner, and incorporating any css styles, into either the
	// Paint or Style object for each Node, depending on the type of node (SVG
	// does Paint, Widget does Style).  Only done once after structural
	// changes -- styles are not for dynamic changes.
	Style2D()

	// Size2D: DepthFirst downward pass, each node first calls
	// g.Layout.Reset(), then sets their LayoutSize according to their own
	// intrinsic size parameters, and/or those of its children if it is a
	// Layout.
	Size2D(iter int)

	// Layout2D: MeFirst downward pass (each node calls on its children at
	// appropriate point) with relevant parent BBox that the children are
	// constrained to render within -- they then intersect this BBox with
	// their own BBox (from BBox2D) -- typically just call Layout2DBase for
	// default behavior -- and add parent position to AllocPos, and then
	// return call to Layout2DChildren. Layout does all its sizing and
	// positioning of children in this pass, based on the Size2D data gathered
	// bottom-up and constraints applied top-down from higher levels.
	// Typically only a single iteration is required (iter = 0) but multiple
	// are supported (needed for word-wrapped text or flow layouts) -- return
	// = true indicates another iteration required (pass this up the chain).
	Layout2D(parBBox image.Rectangle, iter int) bool

	// Move2D: optional MeFirst downward pass to move all elements by given
	// delta -- used for scrolling -- the layout pass assigns canonical
	// positions, saved in AllocPosOrig and BBox, and this adds the given
	// delta to that AllocPosOrig -- each node must call ComputeBBox2D to
	// update its bounding box information given the new position.
	Move2D(delta image.Point, parBBox image.Rectangle)

	// BBox2D: compute the raw bounding box of this node relative to its
	// parent viewport -- called during Layout2D to set node BBox field, which
	// is then used in setting WinBBox and VpBBox.
	BBox2D() image.Rectangle

	// Compute VpBBox and WinBBox from BBox, given parent VpBBox -- most nodes
	// call ComputeBBox2DBase but viewports require special code -- called
	// during Layout and Move.
	ComputeBBox2D(parBBox image.Rectangle, delta image.Point)

	// ChildrenBBox2D: compute the bbox available to my children (content),
	// adjusting for margins, border, padding (BoxSpace) taken up by me --
	// operates on the existing VpBBox for this node -- this is what is passed
	// down as parBBox do the children's Layout2D.
	ChildrenBBox2D() image.Rectangle

	// Render2D: Final rendering pass, each node is fully responsible for
	// calling Render2D on its own children, to provide maximum flexibility
	// (see Render2DChildren for default impl) -- bracket the render calls in
	// PushBounds / PopBounds and a false from PushBounds indicates that
	// VpBBox is empty and no rendering should occur.  Typically call
	// ConnectEvents2D to set up connections to receive window events if
	// visible, and disconnect if not.
	Render2D()

	// ConnectEvents2D: setup connections to window events -- called in
	// Render2D if in bounds.  It can be useful to create modular methods for
	// different event types that can then be mix-and-matched in any more
	// specialized types.
	ConnectEvents2D()

	// FocusChanged2D is called on node for changes in focus -- see the
	// FocusChanges values.
	FocusChanged2D(change FocusChanges)

	// HasFocus2D returns true if this node has keyboard focus and should
	// receive keyboard events -- typically this just returns HasFocus based
	// on the Window-managed HasFocus flag, but some types may want to monitor
	// all keyboard activity for certain key keys..
	HasFocus2D() bool

	// FindNamedElement searches for given named element in this node or in
	// parent nodes.  Used for url(#name) references.
	FindNamedElement(name string) Node2D

	// MakeContextMenu creates the context menu items (typically Action
	// elements, but it can be anything) for a given widget, typically
	// activated by the right mouse button or equivalent.  Widget has a
	// function parameter that can be set to add context items (e.g., by Views
	// or other complex widgets) to extend functionality.
	MakeContextMenu(menu *Menu)

	// ContextMenuPos returns the default position for popup menus --
	// by default in the middle of the WinBBox, but can be adapted as
	// appropriate for different widgets.
	ContextMenuPos() image.Point

	// ContextMenu displays the context menu of various actions to perform on
	// a node -- returns immediately, and actions are all executed directly
	// (later) via the action signals.  Calls MakeContextMenu and
	// ContextMenuPos.
	ContextMenu()

	// IsVisible provides the definitive answer as to whether a given node
	// is currently visible.  It is only entirely valid after a render pass
	// for widgets in a visible window, but it checks the window and viewport
	// for their visibility status as well, which is available always.
	// Non-visible nodes are automatically not rendered and not connected to
	// window events.  The Invisible flag is one key element of the IsVisible
	// calculus -- it is set by e.g., TabView for invisible tabs, and is also
	// set if a widget is entirely out of render range.  But again, use
	// IsVisible as the main end-user method.
	// For robustness, it recursively calls the parent -- this is typically
	// a short path -- propagating the Invisible flag properly can be
	// very challenging without mistakenly overwriting invisibility at various
	// levels.
	IsVisible() bool

	// IsDirectWinUpload returns true if this is a node that does a direct window upload
	// e.g., for gi3d.Scene which renders directly to the window texture for maximum efficiency
	IsDirectWinUpload() bool

	// DirectWinUpload does a direct upload of contents to the window
	// Called at the appropriate point during the overall window publish update process
	// For e.g., gi3d.Scene which renders directly to the window texture for maximum efficiency
	// Returns true if this is a type of node that does this (even if it didn't do it)
	DirectWinUpload() bool
}

// FocusChanges are the kinds of changes that can be reported via
// FocusChanged2D method
type FocusChanges int32

const (
	// FocusLost means that keyboard focus is on a different widget
	// (typically) and this one lost focus
	FocusLost FocusChanges = iota

	// FocusGot means that this widget just got keyboard focus
	FocusGot

	// FocusInactive means that although this widget retains keyboard focus
	// (nobody else has it), the user has clicked on something else and
	// therefore the focus should be considered inactive (distracted), and any
	// changes should be applied as this other action could result in closing
	// of a dialog etc.  Keyboard events will still be sent to the focus
	// widget, but it is up to the widget if or how to process them (e.g., it
	// could reactivate on its own).
	FocusInactive

	// FocusActive means that the user has moved the mouse back into the
	// focused widget to resume active keyboard focus.
	FocusActive

	FocusChangesN
)

//go:generate stringer -type=FocusChanges

////////////////////////////////////////////////////////////////////////////////////////
// Node2D impl for Node2DBase (nil)

func (nb *Node2DBase) PropTag() string {
	return "style-prop" // everything that can be a style value is tagged with this
}

func (n *Node2DBase) BaseIface() reflect.Type {
	return reflect.TypeOf((*Node2D)(nil)).Elem()
}

func (nb *Node2DBase) AsNode2D() *Node2DBase {
	return nb
}

func (nb *Node2DBase) AsViewport2D() *Viewport2D {
	return nil
}

func (nb *Node2DBase) AsLayout2D() *Layout {
	return nil
}

func (nb *Node2DBase) AsWidget() *WidgetBase {
	return nil
}

func (nb *Node2DBase) Init2D() {
}

func (nb *Node2DBase) Style2D() {
}

func (nb *Node2DBase) Size2D(iter int) {
}

func (nb *Node2DBase) Layout2D(parBBox image.Rectangle, iter int) bool {
	return false
}

func (nb *Node2DBase) BBox2D() image.Rectangle {
	return image.ZR
}

func (nb *Node2DBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
}

func (nb *Node2DBase) ChildrenBBox2D() image.Rectangle {
	return image.ZR
}

func (nb *Node2DBase) Render2D() {
}

func (nb *Node2DBase) ConnectEvents2D() {
}

func (nb *Node2DBase) Move2D(delta image.Point, parBBox image.Rectangle) {
}

func (nb *Node2DBase) FocusChanged2D(change FocusChanges) {
}

func (nb *Node2DBase) HasFocus2D() bool {
	return nb.HasFocus()
}

// GrabFocus grabs the keyboard input focus on this item
func (nb *Node2DBase) GrabFocus() {
	win := nb.ParentWindow()
	if win != nil {
		win.SetFocus(nb.This())
	}
}

// FocusNext moves the focus onto the next item
func (nb *Node2DBase) FocusNext() {
	win := nb.ParentWindow()
	if win != nil {
		win.FocusNext(win.CurFocus())
	}
}

// FocusPrev moves the focus onto the previous item
func (nb *Node2DBase) FocusPrev() {
	win := nb.ParentWindow()
	if win != nil {
		win.FocusPrev(win.CurFocus())
	}
}

// StartFocus specifies this widget to give focus to when the window opens
func (nb *Node2DBase) StartFocus() {
	win := nb.ParentWindow()
	if win != nil {
		win.SetStartFocus(nb.This())
	}
}

// ContainsFocus returns true if this widget contains the current focus widget
// as maintained in the Window
func (nb *Node2DBase) ContainsFocus() bool {
	win := nb.ParentWindow()
	if win == nil {
		return false
	}
	cur := win.CurFocus()
	if cur == nil {
		return false
	}
	if cur == nb.This() {
		return true
	}
	plev := cur.ParentLevel(nb.This())
	if plev < 0 {
		return false
	}
	return true
}

func (nb *Node2DBase) FindNamedElement(name string) Node2D {
	if nb.Nm == name {
		return nb.This().(Node2D)
	}
	if nb.Par == nil {
		return nil
	}
	if ce := nb.Par.ChildByName(name, -1); ce != nil {
		return ce.(Node2D)
	}
	if pni, _ := KiToNode2D(nb.Par); pni != nil {
		return pni.FindNamedElement(name)
	}
	return nil
}

func (nb *Node2DBase) MakeContextMenu(m *Menu) {
}

func (nb *Node2DBase) ContextMenuPos() (pos image.Point) {
	pos.X = (nb.WinBBox.Min.X + nb.WinBBox.Max.X) / 2
	pos.Y = (nb.WinBBox.Min.Y + nb.WinBBox.Max.Y) / 2
	return
}

func (nb *Node2DBase) ContextMenu() {
	var men Menu
	nb.This().(Node2D).MakeContextMenu(&men)
	if len(men) == 0 {
		return
	}
	pos := nb.This().(Node2D).ContextMenuPos()
	PopupMenu(men, pos.X, pos.Y, nb.Viewport, nb.Nm+"-menu")
}

func (nb *Node2DBase) IsVisible() bool {
	if nb == nil || nb.This() == nil || nb.IsInvisible() || nb.Viewport == nil {
		return false
	}
	if nb.Par == nil || nb.Par.This() == nil {
		return false
	}
	return nb.Par.This().(Node2D).IsVisible()
}

func (nb *Node2DBase) IsDirectWinUpload() bool {
	return false
}

func (nb *Node2DBase) DirectWinUpload() bool {
	return false
}

// WinFullReRender tells the window to do a full re-render of everything on
// next publish -- call this on containers that might contain DirectUpload
// widgets.
func (nb *Node2DBase) WinFullReRender() {
	win := nb.ParentWindow()
	if win != nil {
		win.PublishFullReRender()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// 2D basic infrastructure code

// KiToNode2D converts Ki to a Node2D interface and a Node2DBase obj -- nil if not.
func KiToNode2D(k ki.Ki) (Node2D, *Node2DBase) {
	if k == nil || k.This() == nil { // this also checks for destroyed
		return nil, nil
	}
	nii, ok := k.(Node2D)
	if ok {
		return nii, nii.AsNode2D()
	}
	return nil, nil
}

// KiToNode2DBase converts Ki to a *Node2DBase -- use when known to be at
// least of this type, not-nil, etc
func KiToNode2DBase(k ki.Ki) *Node2DBase {
	return k.(Node2D).AsNode2D()
}

// ConnectEvent connects this node to receive a given type of GUI event
// signal from the parent window -- typically connect only visible nodes, and
// disconnect when not visible
func (nb *Node2DBase) ConnectEvent(et oswin.EventType, pri EventPris, fun ki.RecvFunc) {
	win := nb.ParentWindow()
	if win != nil {
		win.ConnectEvent(nb.This(), et, pri, fun)
	}
}

// DisconnectEvent disconnects this receiver from receiving given event
// type -- pri is priority -- pass AllPris for all priorities -- see also
// DisconnectAllEvents
func (nb *Node2DBase) DisconnectEvent(et oswin.EventType, pri EventPris) {
	win := nb.ParentWindow()
	if win != nil {
		win.DisconnectEvent(nb.This(), et, pri)
	}
}

// DisconnectAllEvents disconnects node from all window events -- typically
// disconnect when not visible -- pri is priority -- pass AllPris for all priorities
func (nb *Node2DBase) DisconnectAllEvents(pri EventPris) {
	win := nb.ParentWindow()
	if win != nil {
		win.DisconnectAllEvents(nb.This(), pri)
	}
}

// DisconnectAllEventsTree disconnect node and all of its children (and so on)
// from all events -- call for to-be-destroyed nodes (will happen in Ki
// destroy anyway, but more efficient here)
func (nb *Node2DBase) DisconnectAllEventsTree(win *Window) {
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
		_, ni := KiToNode2D(k)
		if ni == nil {
			return false // going into a different type of thing, bail
		}
		win.DisconnectAllEvents(ni.This(), AllPris)
		ni.NodeSig.DisconnectAll()
		return true
	})
}

// ConnectToViewport connects the view node's update signal to the viewport as
// a receiver, so that when the view is updated, it triggers the viewport to
// re-render it -- this is automatically called in PushBounds, and
// disconnected with DisconnectAllEvents, so it only occurs for rendered nodes
func (nb *Node2DBase) ConnectToViewport() {
	if nb.Viewport != nil {
		nb.NodeSig.Connect(nb.Viewport.This(), SignalViewport2D)
	}
}

// set our window-level BBox from vp and our bbox
func (nb *Node2DBase) SetWinBBox() {
	if nb.Viewport != nil {
		nb.WinBBox = nb.VpBBox.Add(nb.Viewport.WinBBox.Min)
	} else {
		nb.WinBBox = nb.VpBBox
	}
}

// ComputeBBox2DBase -- computes the VpBBox and WinBBox from BBox, with
// whatever delta may be in effect
func (nb *Node2DBase) ComputeBBox2DBase(parBBox image.Rectangle, delta image.Point) {
	nb.ObjBBox = nb.BBox.Add(delta)
	nb.VpBBox = parBBox.Intersect(nb.ObjBBox)
	nb.SetInvisibleState(nb.VpBBox == image.ZR)
	nb.SetWinBBox()
}

////////////////////////////////////////////////////////////////////////////////////////
// Tree-walking code for the init, style, layout, render passes
//  typically called by Viewport but can be called by others

// FullRender2DTree does a full render of the tree
func (nb *Node2DBase) FullRender2DTree() {
	updt := nb.UpdateStart()
	nb.Init2DTree()
	nb.Style2DTree()
	nb.Size2DTree(0)
	nb.Layout2DTree()
	nb.Render2DTree()
	nb.UpdateEndNoSig(updt)
}

// Init2DTree initializes scene graph tree from node it is called on -- only
// needs to be done once but must be robust to repeated calls -- use a flag if
// necessary -- needed after structural updates to ensure all nodes are
// updated
func (nb *Node2DBase) Init2DTree() {
	if nb.This() == nil {
		return
	}
	pr := prof.Start("Node2D.Init2DTree")
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
		nii, _ := KiToNode2D(k)
		if nii == nil {
			return false
		}
		nii.Init2D()
		return true
	})
	pr.End()
}

// Style2DTree styles scene graph tree from node it is called on -- only needs
// to be done after a structural update in case inherited options changed
func (nb *Node2DBase) Style2DTree() {
	if nb.This() == nil {
		return
	}
	// fmt.Printf("\n\n###################################\n%v\n", string(debug.Stack()))
	pr := prof.Start("Node2D.Style2DTree")
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
		nii, _ := KiToNode2D(k)
		if nii == nil {
			return false
		}
		nii.Style2D()
		return true
	})
	pr.End()
}

// Size2DTree does the sizing as a depth-first pass
func (nb *Node2DBase) Size2DTree(iter int) {
	if nb.This() == nil {
		return
	}
	pr := prof.Start("Node2D.Size2DTree")
	nb.FuncDownDepthFirst(0, nb.This(),
		func(k ki.Ki, level int, d interface{}) bool { // tests whether to process node
			nii, ni := KiToNode2D(k)
			if nii == nil {
				// fmt.Printf("Encountered a non-Node2D -- might have forgotten to define AsNode2D method: %T, %v \n", k, k.PathUnique())
				return false
			}
			if ni.HasNoLayout() {
				return false
			}
			return true
		},
		func(k ki.Ki, level int, d interface{}) bool { // this one does the work
			nii, ni := KiToNode2D(k)
			if ni == nil {
				return false
			}
			nii.Size2D(iter)
			return true
		})
	pr.End()
}

// Layout2DTree does layout pass -- each node iterates over children for
// maximum control -- this starts with parent VpBBox -- can be called de novo.
// Handles multiple iterations if needed.
func (nb *Node2DBase) Layout2DTree() {
	if nb.This() == nil || nb.HasNoLayout() {
		return
	}
	pr := prof.Start("Node2D.Layout2DTree")
	parBBox := image.ZR
	pni, _ := KiToNode2D(nb.Par)
	if pni != nil {
		parBBox = pni.ChildrenBBox2D()
	}
	nbi := nb.This().(Node2D)
	redo := nbi.Layout2D(parBBox, 0) // important to use interface version to get interface!
	if redo {
		wb := nbi.AsWidget()
		if wb != nil {
			ld := wb.LayData
			wb.Size2DTree(1)
			wb.LayData = ld
		} else {
			nb.Size2DTree(1)
		}
		nbi.Layout2D(parBBox, 1) // todo: multiple iters?
	}
	pr.End()
}

// Render2DTree just calls on parent node and it takes full responsibility for
// managing the children -- this allows maximum flexibility for order etc of
// rendering
func (nb *Node2DBase) Render2DTree() {
	if nb.This() == nil {
		return
	}
	pr := prof.Start("Node2D.Render2DTree")
	nb.This().(Node2D).Render2D() // important to use interface version to get interface!
	pr.End()
}

// Layout2DChildren does layout on all of node's children, giving them the
// ChildrenBBox2D -- default call at end of Layout2D.  Passes along whether
// any of the children need a re-layout -- typically Layout2D just returns
// this.
func (nb *Node2DBase) Layout2DChildren(iter int) bool {
	redo := false
	cbb := nb.This().(Node2D).ChildrenBBox2D()
	for _, kid := range nb.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			if nii.Layout2D(cbb, iter) {
				redo = true
			}
		}
	}
	return redo
}

// Move2dChildren moves all of node's children, giving them the ChildrenBBox2D
// -- default call at end of Move2D
func (nb *Node2DBase) Move2DChildren(delta image.Point) {
	cbb := nb.This().(Node2D).ChildrenBBox2D()
	for _, kid := range nb.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			nii.Move2D(delta, cbb)
		}
	}
}

// Render2DChildren renders all of node's children -- default call at end of Render2D()
func (nb *Node2DBase) Render2DChildren() {
	for _, kid := range nb.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			nii.Render2D()
		}
	}
}

// BBoxReport reports on all the bboxes for everything in the tree
func (nb *Node2DBase) BBoxReport() string {
	rpt := ""
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
		nii, ni := KiToNode2D(k)
		if nii == nil {
			return false
		}
		rpt += fmt.Sprintf("%v: vp: %v, win: %v\n", ni.Nm, ni.VpBBox, ni.WinBBox)
		return true
	})
	return rpt
}

// ParentWindow returns the parent window for this node
func (nb *Node2DBase) ParentWindow() *Window {
	if nb.Viewport != nil && nb.Viewport.Win != nil {
		return nb.Viewport.Win
	}
	wini, err := nb.ParentByTypeTry(KiT_Window, true)
	if err != nil {
		// log.Println(err)
		return nil
	}
	return wini.Embed(KiT_Window).(*Window)
}

// ParentViewport returns the parent viewport -- uses AsViewport2D() method on
// Node2D interface
func (nb *Node2DBase) ParentViewport() *Viewport2D {
	var parVp *Viewport2D
	nb.FuncUpParent(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
		nii, ok := k.(Node2D)
		if !ok {
			return false // don't keep going up
		}
		vp := nii.AsViewport2D()
		if vp != nil {
			parVp = vp
			return false // done
		}
		return true
	})
	return parVp
}

// ParentStyle returns parent's style or nil if not avail
func (nb *Node2DBase) ParentStyle() *Style {
	if nb.Par == nil {
		return nil
	}
	if ps, ok := nb.Par.(Styler); ok {
		return ps.Style()
	}
	return nil
}

// ParentPaint returns the Paint from parent, if available
func (nb *Node2DBase) ParentPaint() *Paint {
	if nb.Par == nil {
		return nil
	}
	if pp, ok := nb.Par.(Painter); ok {
		return pp.Paint()
	}
	return nil
}

// ParentReRenderAnchor returns parent that is a ReRenderAnchor -- for
// optimized re-rendering
func (nb *Node2DBase) ParentReRenderAnchor() Node2D {
	var par Node2D
	nb.FuncUpParent(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
		nii, ni := KiToNode2D(k)
		if nii == nil {
			return false // don't keep going up
		}
		if ni.IsReRenderAnchor() {
			par = nii
			return false
		}
		return true
	})
	return par
}

// ParentLayout returns the parent layout
func (nb *Node2DBase) ParentLayout() *Layout {
	ly := nb.ParentByType(KiT_Layout, true)
	if ly == nil {
		return nil
	}
	return ly.Embed(KiT_Layout).(*Layout)
}

// ParentScrollLayout returns the parent layout that has active scrollbars
func (nb *Node2DBase) ParentScrollLayout() *Layout {
	lyk := nb.ParentByType(KiT_Layout, true)
	if lyk == nil {
		return nil
	}
	ly := lyk.Embed(KiT_Layout).(*Layout)
	if ly.HasAnyScroll() {
		return ly
	}
	return ly.ParentScrollLayout()
}

// ScrollToMe tells my parent layout (that has scroll bars) to scroll to keep
// this widget in view -- returns true if scrolled
func (nb *Node2DBase) ScrollToMe() bool {
	ly := nb.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollToItem(nb.This().(Node2D))
}

////////////////////////////////////////////////////////////////////////////////////////
// MetaData2D

// MetaData2D is used for holding meta data info
type MetaData2D struct {
	Node2DBase
	MetaData string
}

var KiT_MetaData2D = kit.Types.AddType(&MetaData2D{}, nil)
