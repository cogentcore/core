// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki"
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

// Update2DTrace can be set to true to obtain a trace of updates that trigger re-rendering
var Update2DTrace bool = false

// Render2DTrace can be set to true to obtain a trace of the nodes rendering
// (just printfs to stdout)
var Render2DTrace bool = false

// Layout2DTrace can be set to true to obtain a trace of all layouts (just
// printfs to stdout)
var Layout2DTrace bool = false

// Node2D is the interface for all 2D nodes -- defines the stages of building
// and rendering the 2D scenegraph
type Node2D interface {
	// nodes are Ki elements -- this comes for free by embedding ki.Node in
	// all Node2D elements
	ki.Ki

	// AsNode2D returns a generic Node2DBase for our node -- gives generic
	// access to all the base-level data structures without requiring
	// interface methods
	AsNode2D() *Node2DBase

	// AsViewport2D returns Viewport2D if this node is one (has its own
	// bitmap, used for menus, dialogs, icons, etc), else nil
	AsViewport2D() *Viewport2D

	// AsLayout2D returns Layout if this is a Layout-derived node, else nil
	AsLayout2D() *Layout

	// AsWidget returns WidgetBase if this is a WidgetBase-derived node, else nil
	AsWidget() *WidgetBase

	// Init2D initializes a node -- grabs active Viewport etc -- must call
	// InitNodeBase as first step set basic inits including setting Viewport
	// -- all code here must be robust to being called repeatedly
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
	// Layout
	Size2D()

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
	// update its bounding box information given the new position
	Move2D(delta image.Point, parBBox image.Rectangle)

	// BBox2D: compute the raw bounding box of this node relative to its
	// parent viewport -- called during Layout2D to set node BBox field, which
	// is then used in setting WinBBox and VpBBox
	BBox2D() image.Rectangle

	// Compute VpBBox and WinBBox from BBox, given parent VpBBox -- most nodes
	// call ComputeBBox2DBase but viewports require special code -- called
	// during Layout and Move
	ComputeBBox2D(parBBox image.Rectangle, delta image.Point)

	// ChildrenBBox2D: compute the bbox available to my children (content),
	// adjusting for margins, border, padding (BoxSpace) taken up by me --
	// operates on the existing VpBBox for this node -- this is what is passed
	// down as parBBox do the children's Layout2D
	ChildrenBBox2D() image.Rectangle

	// Render2D: Final rendering pass, each node is fully responsible for
	// calling Render2D on its own children, to provide maximum flexibility
	// (see Render2DChildren for default impl) -- bracket the render calls in
	// PushBounds / PopBounds and a false from PushBounds indicates that
	// VpBBox is empty and no rendering should occur.  Typically call a method
	// that sets up connections to receive window events if visible, and
	// disconnect if not.
	Render2D()

	// FocusChanged2D is called on node when it gets or loses focus -- focus
	// flag has current state too
	FocusChanged2D(gotFocus bool)

	// HasFocus2D returns true if this node has keyboard focus and should
	// receive keyboard events -- typically this just returns HasFocus based
	// on the Window-managed HasFocus flag, but some types may want to monitor
	// all keyboard activity for certain key keys..
	HasFocus2D() bool

	// FindNamedElement searches for given named element in this node or in
	// parent nodes.  Used for url(#name) references
	FindNamedElement(name string) Node2D

	// MakeContextMenu creates the context menu items (typically Action
	// elements, but it can be anything) for a given widget, typically
	// activated by the right mouse button or equivalent.  Widget has a
	// function parameter that can be set to add context items (e.g., by Views
	// or other complex widgets) to extend functionality.
	MakeContextMenu(menu *Menu)

	// ContextMenuPos returns the default position for popup menus --
	// by default in the middle of the WinBBox, but can be adapted as
	// appropriate for different widgets
	ContextMenuPos() image.Point

	// ContextMenu displays the context menu of various actions to perform on
	// a node -- returns immediately, and actions are all executed directly
	// (later) via the action signals.  Calls MakeContextMenu and
	// ContextMenuPos
	ContextMenu()
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D impl for Node2DBase (nil)

func (g *Node2DBase) AsNode2D() *Node2DBase {
	return g
}

func (g *Node2DBase) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Node2DBase) AsLayout2D() *Layout {
	return nil
}

func (g *Node2DBase) AsWidget() *WidgetBase {
	return nil
}

func (g *Node2DBase) Init2D() {
	g.Viewport = g.ParentViewport()
}

func (g *Node2DBase) Style2D() {
}

func (g *Node2DBase) Size2D() {
}

func (g *Node2DBase) Layout2D(parBBox image.Rectangle, iter int) bool {
	return false
}

func (g *Node2DBase) BBox2D() image.Rectangle {
	return image.ZR
}

func (g *Node2DBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
}

func (g *Node2DBase) ChildrenBBox2D() image.Rectangle {
	return image.ZR
}

func (g *Node2DBase) Render2D() {
}

func (g *Node2DBase) Move2D(delta image.Point, parBBox image.Rectangle) {
}

func (g *Node2DBase) FocusChanged2D(gotFocus bool) {
}

func (g *Node2DBase) HasFocus2D() bool {
	return g.HasFocus()
}

// GrabFocus grabs the keyboard input focus on this item
func (g *Node2DBase) GrabFocus() {
	win := g.ParentWindow()
	if win != nil {
		win.SetFocusItem(g.This)
	}
}

// FocusNext moves the focus onto the next item
func (g *Node2DBase) FocusNext() {
	win := g.ParentWindow()
	if win != nil {
		win.SetNextFocusItem()
	}
}

// StartFocus specifies this widget to give focus to when the window opens
func (g *Node2DBase) StartFocus() {
	win := g.ParentWindow()
	if win != nil {
		win.SetStartFocus(g.This)
	}
}

// ContainsFocus returns true if this widget contains the current focus widget
// as maintained in the Window
func (g *Node2DBase) ContainsFocus() bool {
	win := g.ParentWindow()
	if win == nil {
		return false
	}
	if win.Focus == nil {
		return false
	}
	plev := win.Focus.ParentLevel(g.This)
	if plev < 0 {
		return false
	}
	return true
}

func (g *Node2DBase) FindNamedElement(name string) Node2D {
	if g.Nm == name {
		return g.This.(Node2D)
	}
	if g.Par == nil {
		return nil
	}
	if ce, ok := g.Par.ChildByName(name, -1); ok {
		return ce.(Node2D)
	}
	pni, _ := KiToNode2D(g.Par)
	if pni != nil {
		return pni.FindNamedElement(name)
	}
	return nil
}

func (g *Node2DBase) MakeContextMenu(m *Menu) {
}

func (g *Node2DBase) ContextMenuPos() (pos image.Point) {
	pos.X = (g.WinBBox.Min.X + g.WinBBox.Max.X) / 2
	pos.Y = (g.WinBBox.Min.Y + g.WinBBox.Max.Y) / 2
	return
}

func (g *Node2DBase) ContextMenu() {
	var men Menu
	g.This.(Node2D).MakeContextMenu(&men)
	if len(men) == 0 {
		return
	}
	pos := g.This.(Node2D).ContextMenuPos()
	PopupMenu(men, pos.X, pos.Y, g.Viewport, g.Nm+"-menu")
}

////////////////////////////////////////////////////////////////////////////////////////
// 2D basic infrastructure code

// KiToNode2D converts Ki to a Node2D interface and a Node2DBase obj -- nil if not.
func KiToNode2D(k ki.Ki) (Node2D, *Node2DBase) {
	if k == nil {
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

// ConnectEventType connects this node to receive a given type of GUI event
// signal from the parent window -- typically connect only visible nodes, and
// disconnect when not visible
func (g *Node2DBase) ConnectEventType(et oswin.EventType, pri EventPris, fun ki.RecvFunc) {
	win := g.ParentWindow()
	if win != nil {
		win.ConnectEventType(g.This, et, pri, fun)
	}
}

// DisconnectEventType disconnects this receiver from receiving given event
// type -- pri is priority -- pass AllPris for all priorities -- see also
// DisconnectAllEvents
func (g *Node2DBase) DisconnectEventType(et oswin.EventType, pri EventPris) {
	win := g.ParentWindow()
	if win != nil {
		win.DisconnectEventType(g.This, et, pri)
	}
}

// DisconnectAllEvents disconnects node from all window events -- typically
// disconnect when not visible -- pri is priority -- pass AllPris for all priorities
func (g *Node2DBase) DisconnectAllEvents(pri EventPris) {
	win := g.ParentWindow()
	if win != nil {
		win.DisconnectAllEvents(g.This, pri)
	}
}

// DisconnectAllEventsTree disconnect node and all of its children (and so on)
// from all events -- call for to-be-destroyed nodes (will happen in Ki
// destroy anyway, but more efficient here)
func (g *Node2DBase) DisconnectAllEventsTree(win *Window) {
	g.FuncDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		_, ni := KiToNode2D(k)
		if ni == nil {
			return false // going into a different type of thing, bail
		}
		win.DisconnectAllEvents(ni.This, AllPris)
		ni.NodeSig.DisconnectAll()
		return true
	})
}

// ConnectToViewport connects the view node's update signal to the viewport as
// a receiver, so that when the view is updated, it triggers the viewport to
// re-render it -- this is automatically called in PushBounds, and
// disconnected with DisconnectAllEvents, so it only occurs for rendered nodes
func (g *Node2DBase) ConnectToViewport() {
	if g.Viewport != nil {
		g.NodeSig.Connect(g.Viewport.This, SignalViewport2D)
	}
}

// set our window-level BBox from vp and our bbox
func (g *Node2DBase) SetWinBBox() {
	if g.Viewport != nil {
		g.WinBBox = g.VpBBox.Add(g.Viewport.WinBBox.Min)
	} else {
		g.WinBBox = g.VpBBox
	}
}

// ComputeBBox2DBase -- computes the VpBBox and WinBBox from BBox, with
// whatever delta may be in effect
func (g *Node2DBase) ComputeBBox2DBase(parBBox image.Rectangle, delta image.Point) {
	g.ObjBBox = g.BBox.Add(delta)
	g.VpBBox = parBBox.Intersect(g.ObjBBox)
	g.SetWinBBox()
}

////////////////////////////////////////////////////////////////////////////////////////
// Tree-walking code for the init, style, layout, render passes
//  typically called by Viewport but can be called by others

// FullRender2DTree does a full render of the tree
func (g *Node2DBase) FullRender2DTree() {
	updt := g.UpdateStart()
	g.Init2DTree()
	g.Style2DTree()
	g.Size2DTree()
	g.Layout2DTree()
	g.Render2DTree()
	g.UpdateEndNoSig(updt)
}

// Init2DTree initializes scene graph tree from node it is called on -- only
// needs to be done once but must be robust to repeated calls -- use a flag if
// necessary -- needed after structural updates to ensure all nodes are
// updated
func (g *Node2DBase) Init2DTree() {
	pr := prof.Start("Node2D.Init2DTree")
	g.FuncDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
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
func (g *Node2DBase) Style2DTree() {
	pr := prof.Start("Node2D.Style2DTree")
	g.FuncDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
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
func (g *Node2DBase) Size2DTree() {
	pr := prof.Start("Node2D.Size2DTree")
	g.FuncDownDepthFirst(0, g.This,
		func(k ki.Ki, level int, d interface{}) bool { // tests whether to process node
			nii, ni := KiToNode2D(k)
			if nii == nil {
				fmt.Printf("Encountered a non-Node2D -- might have forgotten to define AsNode2D method: %T, %v \n", k, k.PathUnique())
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
			nii.Size2D()
			return true
		})
	pr.End()
}

// Layout2DTree does layout pass -- each node iterates over children for
// maximum control -- this starts with parent VpBBox -- can be called de novo.
// Handles multiple iterations if needed.
func (g *Node2DBase) Layout2DTree() {
	if g.HasNoLayout() {
		return
	}
	pr := prof.Start("Node2D.Layout2DTree")
	parBBox := image.ZR
	_, pg := KiToNode2D(g.Par)
	if pg != nil {
		parBBox = pg.VpBBox
	}
	redo := g.This.(Node2D).Layout2D(parBBox, 0) // important to use interface version to get interface!
	if redo {
		g.This.(Node2D).Layout2D(parBBox, 1) // todo: multiple iters?
	}
	pr.End()
}

// Render2DTree just calls on parent node and it takes full responsibility for
// managing the children -- this allows maximum flexibility for order etc of
// rendering
func (g *Node2DBase) Render2DTree() {
	pr := prof.Start("Node2D.Render2DTree")
	g.This.(Node2D).Render2D() // important to use interface version to get interface!
	pr.End()
}

// Layout2DChildren does layout on all of node's children, giving them the
// ChildrenBBox2D -- default call at end of Layout2D.  Passes along whether
// any of the children need a re-layout -- typically Layout2D just returns
// this.
func (g *Node2DBase) Layout2DChildren(iter int) bool {
	redo := false
	cbb := g.This.(Node2D).ChildrenBBox2D()
	for _, kid := range g.Kids {
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
func (g *Node2DBase) Move2DChildren(delta image.Point) {
	cbb := g.This.(Node2D).ChildrenBBox2D()
	for _, kid := range g.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			nii.Move2D(delta, cbb)
		}
	}
}

// Render2DChildren renders all of node's children -- default call at end of Render2D()
func (g *Node2DBase) Render2DChildren() {
	for _, kid := range g.Kids {
		nii, _ := KiToNode2D(kid)
		if nii != nil {
			nii.Render2D()
		}
	}
}

// BBoxReport reports on all the bboxes for everything in the tree
func (g *Node2DBase) BBoxReport() string {
	rpt := ""
	g.FuncDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
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
func (g *Node2DBase) ParentWindow() *Window {
	if g.Viewport != nil && g.Viewport.Win != nil {
		return g.Viewport.Win
	}
	wini, ok := g.ParentByType(KiT_Window, true)
	if !ok {
		return nil
	}
	return wini.Embed(KiT_Window).(*Window)
}

// ParentViewport returns the parent viewport -- uses AsViewport2D() method on
// Node2D interface
func (g *Node2DBase) ParentViewport() *Viewport2D {
	var parVp *Viewport2D
	g.FuncUpParent(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
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
func (g *Node2DBase) ParentStyle() *Style {
	if g.Par == nil {
		return nil
	}
	if ps, ok := g.Par.(Styler); ok {
		return ps.Style()
	}
	return nil
}

// ParentPaint returns the Paint from parent, if available
func (g *Node2DBase) ParentPaint() *Paint {
	if g.Par == nil {
		return nil
	}
	if pp, ok := g.Par.(Painter); ok {
		return pp.Paint()
	}
	return nil
}

// ParentReRenderAnchor returns parent that is a ReRenderAnchor -- for
// optimized re-rendering
func (g *Node2DBase) ParentReRenderAnchor() Node2D {
	var par Node2D
	g.FuncUpParent(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
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

////////////////////////////////////////////////////////////////////////////////////////
// MetaData2D

// MetaData2D is used for holding meta data info
type MetaData2D struct {
	Node2DBase
	MetaData string
}

var KiT_MetaData2D = kit.Types.AddType(&MetaData2D{}, nil)
