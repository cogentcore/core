// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
	// "gopkg.in/go-playground/colors.v1"
	"image"
	// "log"
	// "reflect"
	// "strconv"
	// "strings"
)

////////////////////////////////////////////////////////////////////////////////////////
// 2D  Nodes

/*

Base struct node for 2D rendering tree -- renders to a bitmap using Paint
rendering functions operating on the RenderState in the parent Viewport

Rendering is done in 4 separate passes:

	1. Style2D: In a MeFirst downward pass, all properties are cached out in
	an inherited manner, and incorporating any css styles, into either the
	Paint (SVG) or Style (Widget) object for each Node.  Only done once after
	structural changes -- styles are not for dynamic changes.

	2. Size2D: DepthFirst downward pass, each node first calls
	g.Layout.Reset(), then sets their LayoutSize according to their own
	intrinsic size parameters, and/or those of its children if it is a Layout

	3. Layout2D: MeFirst downward pass (each node calls on its children at
	appropriate point) with relevant parent BBox that the children are
	constrained to render within -- they then intersect this BBox with their
	own BBox (from BBox2D) -- typically just call Layout2DBase for default
	behavior -- and add parent position to AllocPos -- Layout does all its
	sizing and positioning of children in this pass, based on the Size2D data
	gathered bottom-up and constraints applied top-down from higher levels

	4. Render2D: Render2D: Final rendering pass, each node is fully
	responsible for rendering its own children, to provide maximum flexibility
	(see Render2DChildren) -- bracket the render calls in PushBounds /
	PopBounds and a false from PushBounds indicates that VpBBox is empty and
	no rendering should occur

*/
type Node2DBase struct {
	NodeBase
	Style    Style       `desc:"styling settings for this item -- set in SetStyle2D during an initialization step, and when the structure changes"`
	Paint    Paint       `json:"-",desc:"full paint information for this node"`
	Viewport *Viewport2D `json:"-",desc:"our viewport -- set in Init2D (Base typically) and used thereafter"`
	LayData  LayoutData  `desc:"all the layout information for this item"`
}

var KiT_Node2DBase = kit.Types.AddType(&Node2DBase{}, nil)

// primary interface for all Node2D nodes
type Node2D interface {
	// get a generic Node2DBase for our node -- not otherwise possible -- don't want to interface everything that a base node can do as that would be highly redundant
	AsNode2D() *Node2DBase
	// if this is a Viewport2D-derived node, get it as a Viewport2D, else return nil
	AsViewport2D() *Viewport2D
	// if this is a Layout-derived node, get it as a Layout, else return nil
	AsLayout2D() *Layout
	// initialize a node -- setup event receiving connections etc -- must call InitNodeBase as first step set basic inits including setting Viewport and connecting node signal to parent vp -- all code here must be robust to being called repeatedly
	Init2D()
	// Style2D: In a MeFirst downward pass, all properties are cached out in an inherited manner, and incorporating any css styles, into either the Paint or Style object for each Node, depending on the type of node (SVG does Paint, Widget does Style).  Only done once after structural changes -- styles are not for dynamic changes.
	Style2D()
	// Size2D: DepthFirst downward pass, each node first calls g.Layout.Reset(), then sets their LayoutSize according to their own intrinsic size parameters, and/or those of its children if it is a Layout
	Size2D()
	// Layout2D: MeFirst downward pass (each node calls on its children at appropriate point) with relevant parent BBox that the children are constrained to render within -- they then intersect this BBox with their own BBox (from BBox2D) -- typically just call Layout2DBase for default behavior -- and add parent position to AllocPos -- Layout does all its sizing and positioning of children in this pass, based on the Size2D data gathered bottom-up and constraints applied top-down from higher levels
	Layout2D(parBBox image.Rectangle)
	// BBox2D: compute the raw bounding box of this node relative to its parent viewport -- used in setting WinBBox and VpBBox, during Layout2D
	BBox2D() image.Rectangle
	// ChildrenBBox2D: compute the bbox available to my children (content), adjusting for margins, border, padding (BoxSpace) taken up by me -- operates on the existing VpBBox for this node -- this is what is passed down as parBBox do the children's Layout2D
	ChildrenBBox2D() image.Rectangle
	// Render2D: Final rendering pass, each node is fully responsible for rendering its own children, to provide maximum flexibility (see Render2DChildren) -- bracket the render calls in PushBounds / PopBounds and a false from PushBounds indicates that VpBBox is empty and no rendering should occur
	Render2D()
	// Can this node re-render itself directly using cached data?  only for nodes that paint an opaque background first (e.g., widgets) -- optimizes local redraw when possible -- always true for sub-viewports
	CanReRender2D() bool
	// function called on node when it gets or loses focus -- focus flag has current state too
	FocusChanged2D(gotFocus bool)
}

////////////////////////////////////////////////////////////////////////////////////////
// 2D basic infrastructure code

// convert Ki to a Node2D interface and a Node2DBase obj -- nil if not
func KiToNode2D(k ki.Ki) (Node2D, *Node2DBase) {
	if k == nil {
		return nil, nil
	}
	gii, ok := k.(Node2D)
	if ok {
		return gii, gii.AsNode2D()
	}
	return nil, nil
}

// handles basic node initialization -- Init2D can then do special things
func (g *Node2DBase) Init2DBase() {
	g.Viewport = g.FindViewportParent()
	if g.Viewport != nil { // default for most cases -- delete connection of not
		// fmt.Printf("node %v connect to viewport %v\n", g.Name, g.Viewport.Name)
		g.NodeSig.Connect(g.Viewport.This, SignalViewport2D)
	}
	g.Style.Defaults()
	g.Paint.Defaults()
	g.LayData.Defaults() // doesn't overwrite
}

// style the Paint values directly from node properties -- for SVG-style nodes
func (g *Node2DBase) Style2DSVG() {
	gii, _ := g.This.(Node2D)
	if g.Viewport == nil { // robust
		gii.Init2D()
	}
	pg := g.CopyParentPaint() // svg always inherits all paint settings from parent
	g.Paint.SetStyle(&pg.Paint, &PaintDefault, g.KiProps())
	g.Paint.SetUnitContext(g.Viewport, Vec2DZero) // svn only has to set units here once
}

// style the Style values from node properties -- for Widget-style nodes
func (g *Node2DBase) Style2DWidget() {
	gii, _ := g.This.(Node2D)
	if g.Viewport == nil { // robust
		gii.Init2D()
	}
	_, pg := KiToNode2D(g.Parent)
	if pg != nil {
		g.Style.SetStyle(&pg.Style, &StyleDefault, g.KiProps())
	} else {
		g.Style.SetStyle(nil, &StyleDefault, g.KiProps())
	}
	g.Style.SetUnitContext(g.Viewport, Vec2DZero) // todo: test for use of el-relative
	g.LayData.SetFromStyle(&g.Style.Layout)       // also does reset
}

// find parent viewport -- uses AsViewport2D() method on Node2D interface
func (g *Node2DBase) FindViewportParent() *Viewport2D {
	var parVp *Viewport2D
	g.FunUpParent(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, ok := k.(Node2D)
		if !ok {
			return false // don't keep going up
		}
		vp := gii.AsViewport2D()
		if vp != nil {
			parVp = vp
			return false // done
		}
		return true
	})
	return parVp
}

// copy our paint from our parents -- called during Style for SVG
func (g *Node2DBase) CopyParentPaint() *Node2DBase {
	_, pg := KiToNode2D(g.Parent)
	if pg != nil {
		g.Paint = pg.Paint
	}
	return pg
}

func (g *Node2DBase) InitLayout2D() {
	g.LayData.SetFromStyle(&g.Style.Layout)
}

// get our bbox from Layout allocation
func (g *Node2DBase) BBoxFromAlloc() image.Rectangle {
	tp := g.LayData.AllocPos
	ts := g.LayData.AllocSize
	return image.Rect(int(tp.X), int(tp.Y), int(tp.X+ts.X), int(tp.Y+ts.Y))
}

// set our window-level BBox from vp and our bbox
func (g *Node2DBase) SetWinBBox() {
	if g.Viewport != nil {
		g.WinBBox = g.VpBBox.Add(image.Point{g.Viewport.WinBBox.Min.X, g.Viewport.WinBBox.Min.Y})
	} else {
		g.WinBBox = g.VpBBox
	}
}

// add the position of our parent to our layout position -- layout
// computations are all relative to parent position, so they are finally
// cached out at this stage also returns the size of the parent for setting
// units context relative to parent objects
func (g *Node2DBase) AddParentPos() Vec2D {
	_, pg := KiToNode2D(g.Parent)
	if pg != nil {
		g.LayData.AllocPos = g.LayData.AllocPos.Add(pg.LayData.AllocPos)
		return pg.LayData.AllocSize
	}
	return Vec2DZero
}

// ComputeBBox -- computes the VpBBox and WinBBox for node -- returns parent
// size from AddParentPos
func (g *Node2DBase) ComputeBBox(parBBox image.Rectangle) Vec2D {
	psize := g.AddParentPos()
	gii := g.This.(Node2D)
	g.VpBBox = parBBox.Intersect(gii.BBox2D())
	g.SetWinBBox()
	return psize
}

// basic Layout2D functions -- good for most cases
func (g *Node2DBase) Layout2DBase(parBBox image.Rectangle, initStyle bool) {
	psize := g.ComputeBBox(parBBox)
	if initStyle {
		g.Style.SetUnitContext(g.Viewport, psize) // update units with final layout
	}
	g.Paint.SetUnitContext(g.Viewport, psize) // always update paint
	// note: if other styles are maintained, they also need to be updated!
	// also Layout2DChildren must be called after this!
}

// if non-empty, push our bounding-box bounds onto the bounds stack -- this
// limits our drawing to our own bounding box, automatically -- must be called
// as first step in Render2D returns whether the new bounds are empty or not
// -- if empty then don't render!
func (g *Node2DBase) PushBounds() bool {
	if g.VpBBox.Empty() {
		return false
	}
	rs := &g.Viewport.Render
	rs.PushBounds(g.VpBBox)
	// fmt.Printf("node %v pushed bounds %v\n", g.Name, g.VpBBox)
	return true
}

// pop our bounding-box bounds -- last step in Render2D after rendering children
func (g *Node2DBase) PopBounds() {
	rs := &g.Viewport.Render
	rs.PopBounds()
}

// set minimum and preferred width -- will get at least this amount -- max unspecified
func (g *Node2DBase) SetMinPrefWidth(val units.Value) {
	g.SetProp("width", val)
	g.SetProp("min-width", val)
}

// set minimum and preferred height-- will get at least this amount -- max unspecified
func (g *Node2DBase) SetMinPrefHeight(val units.Value) {
	g.SetProp("height", val)
	g.SetProp("min-height", val)
}

// set stretchy max width -- can grow to take up avail room
func (g *Node2DBase) SetStretchMaxWidth() {
	g.SetProp("max-width", units.NewValue(-1, units.Px))
}

// set stretchy max height -- can grow to take up avail room
func (g *Node2DBase) SetStretchMaxHeight() {
	g.SetProp("max-height", units.NewValue(-1, units.Px))
}

// set all width options (width, min-width, max-width) to a fixed width value
func (g *Node2DBase) SetFixedWidth(val units.Value) {
	g.SetProp("width", val)
	g.SetProp("min-width", val)
	g.SetProp("max-width", val)
}

// set all height options (height, min-height, max-height) to a fixed height value
func (g *Node2DBase) SetFixedHeight(val units.Value) {
	g.SetProp("height", val)
	g.SetProp("min-height", val)
	g.SetProp("max-height", val)
}

////////////////////////////////////////////////////////////////////////////////////////
// Tree-walking code for the init, style, layout, render passes
//  typically called by Viewport but can be called by others

// full render of the tree
func (g *Node2DBase) FullRender2DTree() {
	parBBox := image.ZR
	_, pg := KiToNode2D(g.Parent)
	if pg != nil {
		parBBox = pg.VpBBox
	}
	g.Init2DTree()
	g.Style2DTree()
	g.Size2DTree()
	g.Layout2DTree(parBBox)
	g.Render2DTree()
}

// re-render of the tree -- after it has already been initialized and styled
// -- just does layout and render passes
func (g *Node2DBase) ReRender2DTree() {
	parBBox := image.ZR
	_, pg := KiToNode2D(g.Parent)
	if pg != nil {
		parBBox = pg.VpBBox
	}
	g.Size2DTree()
	g.Layout2DTree(parBBox)
	g.Render2DTree()
}

// initialize scene graph tree from node it is called on -- only needs to be
// done once but must be robust to repeated calls -- use a flag if necessary
// -- needed after structural updates to ensure all nodes are updated
func (g *Node2DBase) Init2DTree() {
	g.FunDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, _ := KiToNode2D(k)
		if gii == nil {
			return false
		}
		gii.Init2D()
		return true
	})
}

// style scene graph tree from node it is called on -- only needs to be
// done after a structural update in case inherited options changed
func (g *Node2DBase) Style2DTree() {
	g.FunDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, _ := KiToNode2D(k)
		if gii == nil {
			return false // going into a different type of thing, bail
		}
		gii.Style2D()
		return true
	})
}

// do the sizing as a depth-first pass
func (g *Node2DBase) Size2DTree() {
	g.FunDownDepthFirst(0, g.This,
		func(k ki.Ki, level int, d interface{}) bool {
			// this is for testing whether to process node
			_, gi := KiToNode2D(k)
			if gi == nil || gi.Paint.Off {
				return false
			}
			return true
		},
		func(k ki.Ki, level int, d interface{}) bool {
			// this one does the work
			gii, gi := KiToNode2D(k)
			if gi == nil || gi.Paint.Off {
				return false
			}
			gii.Size2D()
			return true
		})
}

// layout pass -- each node iterates over children for maximum control -- must
func (g *Node2DBase) Layout2DTree(parBBox image.Rectangle) {
	g.This.(Node2D).Layout2D(parBBox) // important to use interface version to get interface!
}

// render just calls on parent node and it takes full responsibility for
// managing the children -- this allows maximum flexibility for order etc of
// rendering
func (g *Node2DBase) Render2DTree() {
	g.This.(Node2D).Render2D() // important to use interface version to get interface!
}

// this provides a basic widget box-model subtraction of margin and padding to
// children -- call in ChildrenBBox2D for most widgets
func (g *Node2DBase) ChildrenBBox2DWidget() image.Rectangle {
	nb := g.VpBBox
	spc := int(g.Style.BoxSpace())
	nb.Min.X += spc
	nb.Min.Y += spc
	nb.Max.X -= spc
	nb.Max.Y -= spc
	return nb
}

// layout on all of node's children, giving them the ChildrenBBox2D -- default call at end of Layout2D
func (g *Node2DBase) Layout2DChildren() {
	cbb := g.This.(Node2D).ChildrenBBox2D()
	for _, kid := range g.Children {
		gii, _ := KiToNode2D(kid)
		if gii != nil {
			gii.Layout2D(cbb)
		}
	}
}

// render all of node's children -- default call at end of Render2D()
func (g *Node2DBase) Render2DChildren() {
	for _, kid := range g.Children {
		gii, _ := KiToNode2D(kid)
		if gii != nil {
			gii.Render2D()
		}
	}
}

// report on all the bboxes for everything in the tree
func (g *Node2DBase) BBoxReport() string {
	rpt := ""
	g.FunDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, gi := KiToNode2D(k)
		if gii == nil {
			return false // going into a different type of thing, bail
		}
		rpt += fmt.Sprintf("%v: vp: %v, win: %v\n", gi.Name, gi.VpBBox, gi.WinBBox)
		return true
	})
	return rpt
}
