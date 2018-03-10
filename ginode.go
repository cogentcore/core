// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package Gi (GoGi) provides a complete Graphical Interface based on GoKi Tree Node structs

	The GiNode struct that implements the Ki interface, which
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
	"image/color"
	"log"
	"reflect"
	"strconv"
	"strings"
)

// basic component node for GoGi
type GiNode struct {
	ki.Node
}

// standard css properties on nodes apply, including visible, etc.

// basic component node for 2D rendering
type GiNode2D struct {
	GiNode
	z_index int           `svg:"z-index",desc:"ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor"`
	XForm   XFormMatrix2D `desc:"the transform present when we were last rendered"`
}

// primary interface for all GiNode2D's
type GiNode2DI interface {
	// initialize a node -- setup connections etc -- should be robust to being called repeatedly
	InitNode2D(vp *Viewport2D) bool
	// Render graphics into a 2D viewport -- return value indicates whether we should keep going down -- e.g., viewport cuts off there
	Render2D(vp *Viewport2D) bool
	// Get the GiNode2D representation of the object
	Node2D() *GiNode2D
	// Initialize a new GiNode2D
}

// each node notifies its parent viewport whenever it changes, causing a re-render
func SignalViewport2D(vpki, node ki.Ki, sig ki.SignalType, data interface{}) {
	vp, ok := (vpki).(*Viewport2D)
	if !ok {
		return
	}
	fmt.Printf("viewport: %v rendering due to signal: %v\n", vp.Name, sig)
	var parVp *Viewport2D
	parVpki := vp.FindParentByType(reflect.TypeOf(Viewport2D{}))
	if parVpki != nil {
		parVp = (parVpki).(*Viewport2D)
	}
	if sig == ki.SignalChildAdded {
		vp.InitNode2D(parVp)
	}
	vp.RenderTopLevel()
	if parVp == nil {
		vp.DrawIntoWindow() // if no parent, we must be top-level
	} else {
		vp.DrawIntoParent(parVp)
	}
}

// basic component node for 3D rendering -- has a 3D transform
type GiNode3D struct {
	GiNode
}

////////////////////////////////////////////////////////////////////////////////////////
//    Property checking

// todo: could use an IGiNode interface for these common things.. probably more "chanto"

// IMPORTANT: we do NOT use inherit = true for property checks, because the paint stack captures all the relevant inheritance anyway!

// check for the display: none (false) property -- though spec says it is not inherited, it affects all children, so in fact it really is -- we terminate render when encountered so we don't need inherits version
func (g *GiNode) PropDisplay() bool {
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
func (g *GiNode) PropVisible() bool {
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

// process properties and any css style sheets (todo) to get a number property of the given name -- returns false if property has not been set
func (g *GiNode) PropNumber(name string) (float64, bool) {
	p := g.Prop(name, false) // false = inherit
	if p == nil {
		return 0, false
	}
	switch v := p.(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.Printf("GiNode %v PropNumber convert from string err: %v", g.PathUnique(), err)
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
func (g *GiNode) PropEnum(name string) (string, bool) {
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
func (g *GiNode) PropColor(name string) (color.Color, bool) {
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
func (g *GiNode2D) PropLength(name string) (float64, bool) {
	p := g.Prop(name, false) // false = inherit
	if p == nil {
		return 0, false
	}
	switch v := p.(type) {
	case string:
		// todo: need to parse units from string!
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.Printf("GiNode %v PropLength convert from string err: %v", g.PathUnique(), err)
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
