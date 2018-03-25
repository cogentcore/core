// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki"
	"image"
	"log"
	"reflect"
	// "strconv"
)

// node flags are bitflags for tracking common high-frequency GUI state,
// mostly having to do with event processing -- use properties map for less
// frequently used information
type NodeFlags int32

const (
	NodeFlagsNil NodeFlags = iota
	// can this node accept focus to receive keyboard input events -- set by default for typical nodes that do so, but can be overridden, including by the style 'can-focus' property
	CanFocus
	// does this node currently have the focus for keyboard input events?  use tab / alt tab and clicking events to update focus -- see interface on Window
	HasFocus
	// indicates that the MouseEnteredEvent was previously registered on this node
	MouseHasEntered
	// this node is currently dragging -- win.Dragging set to this node
	NodeDragging
	// can extend node flags from here
	NodeFlagsN
)

//go:generate stringer -type=EventType

var KiT_NodeFlags = ki.Enums.AddEnum(NodeFlagsNil, true, nil) // true = bitflags

// base struct node for GoGi
type NodeBase struct {
	ki.Node
	NodeFlags int64           `desc:"bitwise flags set according to NodeFlags type"`
	WinBBox   image.Rectangle `json:"-" desc:"2D bounding box for region occupied within parent Window object -- need to project all the way up to that -- used e.g., for event filtering"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_NodeBase = ki.Types.AddType(&NodeBase{}, nil)

func (g *NodeBase) ParentWindow() *Window {
	wini := g.FindParentByType(reflect.TypeOf(Window{})) // todo: will not work for derived -- need interface
	if wini == nil {
		log.Printf("Node %v ReceiveEventType -- cannot find parent window -- must be called after adding to the scenegraph\n", g.PathUnique())
		return nil
	}
	return wini.(*Window)
}

// register this node to receive a given type of GUI event signal from the parent window
func (g *NodeBase) ReceiveEventType(et EventType, fun ki.RecvFun) {
	win := g.ParentWindow()
	if win != nil {
		win.ReceiveEventType(g.This, et, fun)
	}
}

// disconnect node from all eventts
func (g *NodeBase) DisconnectAllEvents() {
	win := g.ParentWindow()
	if win != nil {
		win.DisconnectNode(g.This)
	}
}

// zero-out the window bbox -- for nodes that are not visible
func (g *NodeBase) ZeroWinBBox() {
	g.WinBBox = image.ZR
}

// does the current node have keyboard focus
func (g *NodeBase) HasFocus() bool {
	return ki.HasBitFlag(g.NodeFlags, int(HasFocus))
}

// is the current node currently receiving dragging events?  set by window
func (g *NodeBase) IsDragging() bool {
	return ki.HasBitFlag(g.NodeFlags, int(NodeDragging))
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

// standard css properties on nodes apply, including visible, etc.

// see node2d.go for 2d node

////////////////////////////////////////////////////////////////////////////////////////
// 3D  -- todo: move to node3d.go when actually start writing

// basic component node for 3D rendering -- has a 3D transform
type Node3DBase struct {
	NodeBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Node3DBase = ki.Types.AddType(&Node3DBase{}, nil)
