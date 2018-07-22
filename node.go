// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"

	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

// NodeBase is the base struct type for GoGi graphical interface system,
// containing infrastructure for both 2D and 3D scene graph nodes
type NodeBase struct {
	ki.Node
	Class   string          `desc:"user-defined class name used primarily for attaching CSS styles to different display elements"`
	CSS     ki.Props        `xml:"css" desc:"cascading style sheet at this level -- these styles apply here and to everything below, until superceded -- use .class and #name Props elements to apply entire styles to given elements, and type for element type"`
	CSSAgg  ki.Props        `json:"-" xml:"-" desc:"aggregated css properties from all higher nodes down to me"`
	BBox    image.Rectangle `json:"-" xml:"-" desc:"raw original 2D bounding box for the object within its parent viewport -- used for computing VpBBox and WinBBox -- this is not updated by Move2D, whereas VpBBox etc are"`
	ObjBBox image.Rectangle `json:"-" xml:"-" desc:"full object bbox -- this is BBox + Move2D delta, but NOT intersected with parent's parBBox -- used for computing color gradients or other object-specific geometry computations"`
	VpBBox  image.Rectangle `json:"-" xml:"-" desc:"2D bounding box for region occupied within immediate parent Viewport object that we render onto -- these are the pixels we draw into, filtered through parent bounding boxes -- used for render Bounds clipping"`
	WinBBox image.Rectangle `json:"-" xml:"-" desc:"2D bounding box for region occupied within parent Window object, projected all the way up to that -- these are the coordinates where we receive events, relative to the window"`
}

var KiT_NodeBase = kit.Types.AddType(&NodeBase{}, NodeBaseProps)

var NodeBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

// NodeFlags define gi node bitflags for tracking common high-frequency GUI
// state, mostly having to do with event processing -- use properties map for
// less frequently used information -- uses ki Flags field (64 bit capacity)
type NodeFlags int32

const (
	NodeFlagsNil NodeFlags = NodeFlags(ki.FlagsN) + iota

	// EventsConnected: this node has been connected to receive events from
	// the window -- to optimize event processing, connections are typically
	// only established for visible nodes during render, and disconnected when
	// not visible
	EventsConnected

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

	// Inactive disables interaction with widgets or other nodes -- they
	// should indicate this inactive state in an appropriate way, and not
	// process input events
	Inactive

	// InactiveEvents overrides the default behavior where inactive nodes are
	// not sent events from the Window -- for e.g., the TextField which can
	// still be selected and copied when inactive
	InactiveEvents

	// Selected indicates that this node has been selected by the user --
	// widely supported across different nodes
	Selected

	// MouseHasEntered indicates that the MouseEnteredEvent was previously
	// registered on this node
	MouseHasEntered

	// NodeDragging indicates this node is currently dragging -- win.Dragging
	// set to this node
	NodeDragging

	// Overlay indicates this node is an overlay -- affects how it renders
	Overlay

	// can extend node flags from here
	NodeFlagsN
)

//go:generate stringer -type=NodeFlags

var KiT_NodeFlags = kit.Enums.AddEnum(NodeFlagsN, true, nil) // true = bitflags

// CanFocus checks if this node can recieve keyboard focus
func (g *NodeBase) CanFocus() bool {
	return bitflag.Has(g.Flag, int(CanFocus))
}

// HasFocus checks if the current node is flagged as having keyboard focus
func (g *NodeBase) HasFocus() bool {
	return bitflag.Has(g.Flag, int(HasFocus))
}

// IsDragging tests if the current node is currently flagged as receiving
// dragging events -- flag set by window
func (g *NodeBase) IsDragging() bool {
	return bitflag.Has(g.Flag, int(NodeDragging))
}

// IsInactive tests if this node is flagged as Inactive.  if so, behave (e.g.,
// ignore events) and style appropriately
func (g *NodeBase) IsInactive() bool {
	return bitflag.Has(g.Flag, int(Inactive))
}

// SetInactive sets the node as inactive
func (g *NodeBase) SetInactive() {
	bitflag.Set(&g.Flag, int(Inactive))
}

// SetInactiveState set flag as inactive or not based on inact arg
func (g *NodeBase) SetInactiveState(inact bool) {
	bitflag.SetState(&g.Flag, inact, int(Inactive))
}

// SetCanFocusIfActive sets CanFocus flag only if node is active (inactive
// nodes don't need focus typically)
func (g *NodeBase) SetCanFocusIfActive() {
	bitflag.SetState(&g.Flag, !g.IsInactive(), int(CanFocus))
}

// IsSelected tests if this node is flagged as Selected
func (g *NodeBase) IsSelected() bool {
	return bitflag.Has(g.Flag, int(Selected))
}

// SetSelected sets the node as selected
func (g *NodeBase) SetSelected() {
	bitflag.Set(&g.Flag, int(Selected))
}

// ClearSelected sets the node as not selected
func (g *NodeBase) ClearSelected() {
	bitflag.Clear(&g.Flag, int(Selected))
}

// SetSelectedState set flag as selected or not based on sel arg
func (g *NodeBase) SetSelectedState(sel bool) {
	bitflag.SetState(&g.Flag, sel, int(Selected))
}

// NeedsFullReRender checks if node has said it needs full re-render
func (g *NodeBase) NeedsFullReRender() bool {
	return bitflag.Has(g.Flag, int(FullReRender))
}

// SetFullReRender sets node as needing a full ReRender
func (g *NodeBase) SetFullReRender() {
	bitflag.Set(&g.Flag, int(FullReRender))
}

// ClearFullReRender clears node as needing a full ReRender
func (g *NodeBase) ClearFullReRender() {
	bitflag.Clear(&g.Flag, int(FullReRender))
}

// IsReRenderAnchor returns whethers the current node is a ReRenderAnchor
func (g *NodeBase) IsReRenderAnchor() bool {
	return bitflag.Has(g.Flag, int(ReRenderAnchor))
}

// SetReRenderAnchor sets node as a ReRenderAnchor
func (g *NodeBase) SetReRenderAnchor() {
	bitflag.Set(&g.Flag, int(ReRenderAnchor))
}

// IsOverlay returns whether node is an overlay -- lives in special viewport
// and renders without bounds
func (g *NodeBase) IsOverlay() bool {
	return bitflag.Has(g.Flag, int(Overlay))
}

// SetOverlay flags this node as an overlay -- lives in special viewport and
// renders without bounds
func (g *NodeBase) SetAsOverlay() {
	bitflag.Set(&g.Flag, int(Overlay))
}

// translate a point in global pixel coords into relative position within node
func (g *NodeBase) PointToRelPos(pt image.Point) image.Point {
	return pt.Sub(g.WinBBox.Min)
}

// StyleProps returns a property that contains another map of properties for a
// given styling selector, such as :normal :active :hover etc -- the
// convention is to prefix this selector with a : and use lower-case names, so
// we follow that.
func (g *NodeBase) StyleProps(selector string) ki.Props {
	sp := g.Prop(selector, false, true) // yeah, use types
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

// AggCSS aggregates css properties
func AggCSS(agg *ki.Props, css ki.Props) {
	if *agg == nil {
		*agg = make(ki.Props, len(css))
	}
	for key, val := range css {
		(*agg)[key] = val
	}
}

// SetStdXMLAttr sets standard attributes of node given XML-style name /
// attribute values (e.g., from parsing XML / SVG files) -- returns true if handled
func (g *NodeBase) SetStdXMLAttr(name, val string) bool {
	switch name {
	case "id":
		g.SetName(val)
		return true
	case "class":
		g.Class = val
		return true
	case "style":
		SetStylePropsXML(val, &g.Props)
		return true
	}
	return false
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
