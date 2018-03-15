// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package Gi (GoGi) provides a complete Graphical Interface based on GoKi Tree Node structs

	2D and 3D scenegraphs supported, each rendering to respective Viewport2D or 3D
	which in turn can be integrated within the other type of scenegraph.
	Within 2D scenegraph, the following are supported
		* SVG-based rendering nodes for basic shapes, paths, curves, arcs etc, with SVG / CSS properties
		* Widget nodes for GUI actions (Buttons, etc), with support for full SVG-based rendering of styles, using Qt-based naming and functionality, including TreeView, TableView
		* Layouts for placing widgets, based on QtQuick model
		* Primary geometry is managed in terms of inherited position offsets from top-left,
		  in a widget-like manner, but svg-based transforms also supported.
*/
package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki"
	// "gopkg.in/go-playground/colors.v1"
	"image"
	"log"
	"reflect"
	// "strconv"
)

// base struct node for GoGi
type NodeBase struct {
	ki.Node
	WinBBox image.Rectangle `json:"-",desc:"2D bounding box for region occupied within parent Window object -- need to project all the way up to that -- used e.g., for event filtering"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_NodeBase = ki.KiTypes.AddType(&NodeBase{})

// register this node to receive a given type of GUI event signal from the parent window
func (g *NodeBase) ReceiveEventType(et EventType, fun ki.RecvFun) {
	wini := g.FindParentByType(reflect.TypeOf(Window{})) // todo: will not work for derived types!
	if wini == nil {
		log.Printf("Node %v ReceiveEventType -- cannot find parent window -- must be called after adding to the scenegraph\n", g.PathUnique())
		return
	}
	win := wini.(*Window)
	win.ReceiveEventType(g.This, et, fun)
}

// zero-out the window bbox -- for nodes that are not visible
func (g *NodeBase) ZeroWinBBox() {
	g.WinBBox = image.ZR
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
var KiT_Node3DBase = ki.KiTypes.AddType(&Node3DBase{})
