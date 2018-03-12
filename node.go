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
	"strconv"
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

////////////////////////////////////////////////////////////////////////////////////////
// 3D

// basic component node for 3D rendering -- has a 3D transform
type Node3DBase struct {
	NodeBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Node3DBase = ki.KiTypes.AddType(&Node3DBase{})

////////////////////////////////////////////////////////////////////////////////////////
//    Property checking

// IMPORTANT: we do NOT use inherit = true for property checks, because the paint stack captures all the relevant inheritance anyway!

// check for the display: none (false) property -- though spec says it is not inherited, it affects all children, so in fact it really is -- we terminate render when encountered so we don't need inherits version
func (g *NodeBase) PropDisplay() bool {
	p := g.Prop("display", false) // false = inherit
	if p == nil {
		return true
	}
	switch v := p.(type) {
	case string:
		if v == "none" {
			return false
		}
	case bool:
		return v
	}
	return true
}

// check for the visible: none (false) property
func (g *NodeBase) PropVisible() bool {
	p := g.Prop("visible", true) // true= inherit
	if p == nil {
		return true
	}
	switch v := p.(type) {
	case string:
		if v == "none" {
			return false
		}
	case bool:
		return v
	}
	return true
}

// check for the visible: none (false) property
func (g *NodeBase) GiPropBool(name string) (bool, bool) {
	p := g.Prop(name, false)
	if p == nil {
		return false, false
	}
	switch v := p.(type) {
	case string:
		if v == "none" || v == "false" || v == "off" {
			return false, true
		} else {
			return true, true
		}
	case bool:
		return v, true
	}
	return false, false
}

// process properties and any css style sheets (todo) to get a number property of the given name -- returns false if property has not been set
func (g *NodeBase) PropNumber(name string) (float64, bool) {
	p := g.Prop(name, false) // false = inherit
	if p == nil {
		return 0, false
	}
	switch v := p.(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.Printf("NodeBase %v PropNumber convert from string err: %v", g.PathUnique(), err)
			return 0, false
		}
		return f, true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}

// process property as a number -- if not locally set, apply given default
func (g *NodeBase) PropNumberDefault(name string, def float64) float64 {
	if val, got := g.PropNumber(name); got {
		return val
	}
	return def
}

// process properties and any css style sheets (todo) to get an enumerated type as a string -- returns true if value is present
func (g *NodeBase) PropEnum(name string) (string, bool) {
	p := g.Prop(name, false) // false = inherit
	if p == nil {
		return "", false
	}
	switch v := p.(type) {
	case string:
		return v, (len(v) > 0)
	default:
		return "", false
	}
}

// process property as an enum string -- if not locally set, apply given default
func (g *NodeBase) PropEnumDefault(name string, def string) string {
	if val, got := g.PropEnum(name); got {
		return val
	}
	return def
}

// process properties and any css style sheets (todo) to get a color
func (g *NodeBase) PropColor(name string) (*Color, bool) {
	p := g.Prop(name, false) // false = inherit
	if p == nil {
		return nil, false
	}
	switch v := p.(type) {
	case string:
		// fmt.Printf("got color: %v for name: %v\n", v, name)
		// cl, err := colors.Parse(v) // not working
		if v == "none" {
			return nil, true
		}
		c, err := ColorFromString(v)
		if err == nil {
			return &c, true
		}
		return nil, false
	default:
		return nil, false
	}
}

// process properties and any css style sheets (todo) to get a color
func (g *NodeBase) PropColorDefault(name string, def *Color) *Color {
	if val, got := g.PropColor(name); got {
		return val
	}
	return def
}

// todo: use g.Viewport to get % lengths etc

// process properties and any css style sheets (todo) to get a length property of the given name -- returns false if property has not been set -- automatically deals with units such as px, em etc
func (g *Node2DBase) PropLength(name string) (float64, bool) {
	p := g.Prop(name, false) // false = inherit
	if p == nil {
		return 0, false
	}
	switch v := p.(type) {
	case string:
		// todo: need to parse units from string!
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.Printf("NodeBase %v PropLength convert from string err: %v", g.PathUnique(), err)
			return 0, false
		}
		return f, true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}
