// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

// gi node flags are bitflags for tracking common high-frequency GUI state,
// mostly having to do with event processing -- use properties map for less
// frequently used information -- uses ki Flags field
type NodeFlags int32

const (
	NodeFlagsNil NodeFlags = NodeFlags(ki.FlagsN) + iota

	// CanFocus: can this node accept focus to receive keyboard input events
	// -- set by default for typical nodes that do so, but can be overridden,
	// including by the style 'can-focus' property
	CanFocus

	// HasFocus: does this node currently have the focus for keyboard input
	// events?  use tab / alt tab and clicking events to update focus -- see
	// interface on Window
	HasFocus

	// FullReRender indicates that a full re-render is required due to nature
	// of update event -- otherwise default is local re-render -- used
	// internally for nodes to determine what to do on the ReRender step
	FullReRender

	// ReRenderAnchor: this node has a static size, and repaints its
	// background -- any children under it that need to dynamically resize on
	// a ReRender (Update) can refer the update up to rerendering this node,
	// instead of going further up the tree -- e.g., true of Frame's within a
	// SplitView
	ReRenderAnchor

	// ReadOnly is for widgets that support editing, it is read-only -- this
	// must be universally supported in an appropriately-indicated way for
	// each widget
	ReadOnly

	// MouseHasEntered indicates that the MouseEnteredEvent was previously
	// registered on this node
	MouseHasEntered

	// NodeDragging indicates this node is currently dragging -- win.Dragging
	// set to this node
	NodeDragging

	// can extend node flags from here
	NodeFlagsN
)

//go:generate stringer -type=NodeFlags

var KiT_NodeFlags = kit.Enums.AddEnum(NodeFlagsN, true, nil) // true = bitflags

// base struct node for GoGi
type NodeBase struct {
	ki.Node
	BBox    image.Rectangle `json:"-" xml:"-" desc:"raw original 2D bounding box for the object within its parent viewport -- used for computing VpBBox and WinBBox -- this is not updated by Move2D, whereas VpBBox etc are"`
	VpBBox  image.Rectangle `json:"-" xml:"-" desc:"2D bounding box for region occupied within immediate parent Viewport object that we render onto -- these are the pixels we draw into, filtered through parent bounding boxes -- used for render Bounds clipping"`
	WinBBox image.Rectangle `json:"-" xml:"-" desc:"2D bounding box for region occupied within parent Window object, projected all the way up to that -- these are the coordinates where we receive events, relative to the window"`
}

var KiT_NodeBase = kit.Types.AddType(&NodeBase{}, NodeBaseProps)

var NodeBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

func (g *NodeBase) ParentWindow() *Window {
	wini := g.ParentByType(KiT_Window, true)
	if wini == nil {
		// log.Printf("Node %v ReceiveEventType -- cannot find parent window -- must be called after adding to the scenegraph\n", g.PathUnique())
		return nil
	}
	return wini.EmbeddedStruct(KiT_Window).(*Window)
}

// register this node to receive a given type of GUI event signal from the parent window
func (g *NodeBase) ReceiveEventType(et oswin.EventType, fun ki.RecvFunc) {
	win := g.ParentWindow()
	if win != nil {
		win.ReceiveEventType(g.This, et, fun)
	}
}

// disconnect node from all events
func (g *NodeBase) DisconnectAllEvents(win *Window) {
	win.DisconnectNode(g.This)
}

// disconnect node from all events - todo: need a more generic Ki version of this
func (g *NodeBase) DisconnectAllEventsTree(win *Window) {
	g.FuncDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		_, gi := KiToNode2D(k)
		if gi == nil {
			return false // going into a different type of thing, bail
		}
		gi.DisconnectAllEvents(win)
		gi.NodeSig.DisconnectAll()
		return true
	})
}

// can this node focus?
func (g *NodeBase) CanFocus() bool {
	return bitflag.Has(g.Flag, int(CanFocus))
}

// does the current node have keyboard focus
func (g *NodeBase) HasFocus() bool {
	return bitflag.Has(g.Flag, int(HasFocus))
}

// is the current node currently receiving dragging events?  set by window
func (g *NodeBase) IsDragging() bool {
	return bitflag.Has(g.Flag, int(NodeDragging))
}

// is this node ReadOnly?  if so, behave and style appropriately -- equivalent to disabled e.g. for buttons / actions
func (g *NodeBase) IsReadOnly() bool {
	return bitflag.Has(g.Flag, int(ReadOnly))
}

// set the node as read-only
func (g *NodeBase) SetReadOnly() {
	bitflag.Set(&g.Flag, int(ReadOnly))
}

// set read-only state of the node
func (g *NodeBase) SetReadOnlyState(readOnly bool) {
	bitflag.SetState(&g.Flag, readOnly, int(ReadOnly))
}

// node needs full re-render?
func (g *NodeBase) NeedsFullReRender() bool {
	return bitflag.Has(g.Flag, int(FullReRender))
}

// set node as needing a full ReRender
func (g *NodeBase) SetFullReRender() {
	bitflag.Set(&g.Flag, int(FullReRender))
}

// clear node as needing a full ReRender
func (g *NodeBase) ClearFullReRender() {
	bitflag.Clear(&g.Flag, int(FullReRender))
}

// is the current node a ReRenderAnchor?
func (g *NodeBase) IsReRenderAnchor() bool {
	return bitflag.Has(g.Flag, int(ReRenderAnchor))
}

// set node as a ReRenderAnchor
func (g *NodeBase) SetReRenderAnchor() {
	bitflag.Set(&g.Flag, int(ReRenderAnchor))
}

// set node as focus node
func (g *NodeBase) GrabFocus() {
	win := g.ParentWindow()
	if win != nil {
		win.SetFocusItem(g.This)
	}
}

// translate a point in global pixel coords into relative position within node
func (g *NodeBase) PointToRelPos(pt image.Point) image.Point {
	return pt.Sub(g.WinBBox.Min)
}

// StyleProps returns a property that contains another map of properties for a
// given styling selector, such as :normal :active :hover etc -- the
// convention is to prefix this selector with a : and use lower-case names, so
// we follow that.  Standard widgets set these props on the type, and we use
// type-based fallback, so these should exist for most.
func (g *NodeBase) StyleProps(selector string) ki.Props {
	sp := g.Prop(selector, false, true) // don't inherit (style handles that separately) but do use type props
	if sp == nil {
		return nil
	}
	spm, ok := sp.(ki.Props)
	if ok {
		return spm
	}
	log.Printf("gi.StyleProps: looking for a ki.Props for style selector: %v, instead got type: %T, for node: %v\n", selector, spm, g.PathUnique())
	return nil
}

// standard css properties on nodes apply, including visible, etc.

// see node2d.go for 2d node

////////////////////////////////////////////////////////////////////////////////////////
// 3D  -- todo: move to node3d.go when actually start writing

// basic component node for 3D rendering -- has a 3D transform
type Node3DBase struct {
	NodeBase
}

var KiT_Node3DBase = kit.Types.AddType(&Node3DBase{}, nil)
