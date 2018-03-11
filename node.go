// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package Gi (GoGi) provides a complete Graphical Interface based on GoKi Tree Node structs

	The Node struct that implements the Ki interface, which
	can be used as an embedded type (or a struct field) in other structs to provide
	core tree functionality, including:
		* Parent / Child Tree structure -- each Node can ONLY have one parent
		* Paths for locating Nodes within the hierarchy -- key for many use-cases, including IO for pointers
		* Apply a function across nodes up or down a tree -- very flexible for tree walking
		* Generalized I/O -- can Save and Load the Tree as JSON, XML, etc -- including pointers which are saved using paths and automatically cached-out after loading
		* Event sending and receiving between Nodes (simlar to Qt Signals / Slots)
		* Robust updating state -- wrap updates in UpdateStart / End, and signals are blocked until the final end, at which point an update signal is sent -- works across levels
		* Properties (as a string-keyed map) with property inheritance -- css anyone!?
*/
package gi

import (
	"fmt"
	"github.com/rcoreilly/goki/ki"
	// "gopkg.in/go-playground/colors.v1"
	"image"
	"image/color"
	"log"
	"reflect"
	"strconv"
	"strings"
)

// base struct node for GoGi
type NodeBase struct {
	ki.Node
	WinBBox image.Rectangle `json:"-",desc:"2D bounding box for region occupied within parent Window object -- need to project all the way up to that -- used e.g., for event filtering"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_NodeBase = ki.KiTypes.AddType(&NodeBase{})

// todo: try to avoid introducing this interface -- not clear if we need this HasFocus function etc -- would be true if focus is not always just equality with focus object

// primary interface for all Node's -- note: need the interface for all virtual functions
// type Node interface {
// 	HasFocus(focus *Node)
// }

// func (g *Node) HasFocus(focus *Node) {
// 	return g == focus
// }

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

// base struct node for 2D rendering tree -- renders to a bitmap using Paint / Viewport rendering functions
type Node2DBase struct {
	NodeBase
	z_index int           `svg:"z-index",desc:"ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor"`
	XForm   XFormMatrix2D `json:"-",desc:"transform present when we were last rendered"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Node2DBase = ki.KiTypes.AddType(&Node2DBase{})

// primary interface for all Node2D nodes
type Node2D interface {
	// get a generic Node2DBase for our node -- not otherwise possible -- don't want to interface everything that a base node can do as that would be highly redundant
	GiNode2D() *Node2DBase
	// if this is a Viewport2D-derived node, get it as a Viewport2D, else return nil
	GiViewport2D() *Viewport2D
	// initialize a node -- setup connections etc -- should be robust to being called repeatedly
	InitNode2D(vp *Viewport2D) bool
	// get the bounding box of this node relative to its parent viewport -- used in computing EventBBox, called during Render
	Node2DBBox(vp *Viewport2D) image.Rectangle
	// Render graphics into a 2D viewport -- return value indicates whether we should keep going down -- e.g., viewport cuts off there
	Render2D(vp *Viewport2D) bool
}

// find parent viewport -- uses GiViewport2D() method on Node2D interface
func (g *Node2DBase) FindViewportParent() *Viewport2D {
	var parVp *Viewport2D
	g.FunUp(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		if level == 0 { // skip us -- only parents
			return true
		}
		gii, ok := k.(Node2D)
		if !ok {
			return false // don't keep going up
		}
		vp := gii.GiViewport2D()
		if vp != nil {
			parVp = vp
			return false // done
		}
		return true
	})
	return parVp
}

// each node notifies its parent viewport whenever it changes, causing a re-render
func SignalViewport2D(vpki, node ki.Ki, sig ki.SignalType, data interface{}) {
	gii, ok := vpki.(Node2D)
	if !ok {
		return
	}
	vp := gii.GiViewport2D()
	if vp == nil {
		return
	}
	fmt.Printf("viewport: %v rendering due to signal: %v\n", vp.PathUnique(), sig)

	parVp := vp.FindViewportParent()
	if parVp == vp {
		log.Printf("SignalViewport2D: ooops -- parent == me for viewport: %v\n", vp.PathUnique())
		return
	}
	if sig == ki.SignalChildAdded {
		vp.InitNode2D(parVp)
	}
	vp.RenderTopLevel() // render as if we are top-level
	if parVp == nil {
		vp.DrawIntoWindow() // if no parent, we must be top-level
	} else {
		vp.DrawIntoParent(parVp)
	}
}

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

// process properties and any css style sheets (todo) to get a color
func (g *NodeBase) PropColor(name string) (color.Color, bool) {
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
		return ParseHexColor(v), true
	default:
		return nil, false
	}
}

// todo: move to css

// parse Hex color -- todo: also need to lookup color names
func ParseHexColor(x string) color.Color {
	x = strings.TrimPrefix(x, "#")
	var r, g, b, a int
	a = 255
	if len(x) == 3 {
		format := "%1x%1x%1x"
		fmt.Sscanf(x, format, &r, &g, &b)
		r |= r << 4
		g |= g << 4
		b |= b << 4
	}
	if len(x) == 6 {
		format := "%02x%02x%02x"
		fmt.Sscanf(x, format, &r, &g, &b)
	}
	if len(x) == 8 {
		format := "%02x%02x%02x%02x"
		fmt.Sscanf(x, format, &r, &g, &b, &a)
	}

	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
}

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
