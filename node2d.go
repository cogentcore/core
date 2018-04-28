// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strings"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
	"github.com/rcoreilly/prof"
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

	4. Render2D: Final rendering pass, each node is fully responsible for
	rendering its own children, to provide maximum flexibility (see
	Render2DChildren) -- bracket the render calls in PushBounds / PopBounds
	and a false from PushBounds indicates that VpBBox is empty and no
	rendering should occur

    * Move2D: optional pass invoked by scrollbars to move elements relative to
      their previously-assigned positions.

*/
type Node2DBase struct {
	NodeBase
	Style    Style       `json:"-" xml:"-" desc:"styling settings for this item -- set in SetStyle2D during an initialization step, and when the structure changes"`
	DefStyle *Style      `json:"-" xml:"-" desc:"default style values computed by a parent widget for us -- if set, we are a part of a parent widget and should use these as our starting styles instead of type-based defaults"`
	Paint    Paint       `json:"-" xml:"-" desc:"full paint information for this node"`
	Viewport *Viewport2D `json:"-" xml:"-" desc:"our viewport -- set in Init2D (Base typically) and used thereafter"`
	LayData  LayoutData  `json:"-" xml:"-" desc:"all the layout information for this item"`
}

var KiT_Node2DBase = kit.Types.AddType(&Node2DBase{}, Node2DBaseProps)

func (n *Node2DBase) New() ki.Ki { return &Node2DBase{} }

var Node2DBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

// set this variable to true to obtain a trace of updates that trigger re-rendering
var Update2DTrace bool = false

// set this variable to true to obtain a trace of the nodes rendering (just printfs to stdout)
var Render2DTrace bool = false

// set this variable to true to obtain a trace of all layouts (just printfs to stdout)
var Layout2DTrace bool = false

// primary interface for all Node2D nodes
type Node2D interface {
	// nodes are Ki elements -- this comes for free by embedding ki.Node in all Node2D elements
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

	// Init2D initializes a node -- sets up event receiving connections etc --
	// must call InitNodeBase as first step set basic inits including setting
	// Viewport and connecting node signal to parent vp -- all code here must
	// be robust to being called repeatedly
	Init2D()

	// Style2D: In a MeFirst downward pass, all properties are cached out in
	// an inherited manner, and incorporating any css styles, into either the
	// Paint or Style object for each Node, depending on the type of node (SVG
	// does Paint, Widget does Style).  Only done once after structural
	// changes -- styles are not for dynamic changes.
	Style2D()

	// StyleCSS: Called by widgets within their Style2D pass -- implements the
	// CSS cascade of styles for given node -- only Widget nodes have CSS
	// sheets attached
	StyleCSS(node Node2D)

	// ReStyle2D: An optional ad-hoc pass to update part styles after a parent
	// node has updated its state
	ReStyle2D()

	// Size2D: DepthFirst downward pass, each node first calls
	// g.Layout.Reset(), then sets their LayoutSize according to their own
	// intrinsic size parameters, and/or those of its children if it is a
	// Layout
	Size2D()

	// Layout2D: MeFirst downward pass (each node calls on its children at
	// appropriate point) with relevant parent BBox that the children are
	// constrained to render within -- they then intersect this BBox with
	// their own BBox (from BBox2D) -- typically just call Layout2DBase for
	// default behavior -- and add parent position to AllocPos -- Layout does
	// all its sizing and positioning of children in this pass, based on the
	// Size2D data gathered bottom-up and constraints applied top-down from
	// higher levels
	Layout2D(parBBox image.Rectangle)

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
	// VpBBox is empty and no rendering should occur
	Render2D()

	// ReRender2D: returns the node that should be re-rendered when an Update
	// signal is received for a given node, and whether a new layout pass from
	// that node down is needed) -- can be the node itself, any of its parents
	// or children, or nil to indicate that a full re-render is necessary.
	// For re-rendering to work, an opaque background should be painted first
	ReRender2D() (node Node2D, layout bool)

	// FocusChanged2D is called on node when it gets or loses focus -- focus
	// flag has current state too
	FocusChanged2D(gotFocus bool)

	// HasFocus2D returns true if this node has keyboard focus and should
	// receive keyboard events -- typically this just returns HasFocus based
	// on the Window-managed HasFocus flag, but some types may want to monitor
	// all keyboard activity for certain key keys..
	HasFocus2D() bool
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D impl for Node2DBase
// only need to redefine (override) these methods as necessary

func (g *Node2DBase) AsNode2D() *Node2DBase {
	return g
}

func (g *Node2DBase) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Node2DBase) AsLayout2D() *Layout {
	return nil
}

func (g *Node2DBase) Init2D() {
	g.Init2DBase()
}

func (g *Node2DBase) Style2D() {
	g.Style2DSVG() // base is used for SVG -- Widget overrides
	pc := &g.Paint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

func (g *Node2DBase) StyleCSS(node Node2D) {
	// nop for regular nodes -- only Widget, Viewport, and Layout nodes can cascade
}

func (g *Node2DBase) ReStyle2D() {
	g.ReStyle2DSVG()
}

func (g *Node2DBase) Size2D() {
	g.InitLayout2D()
	g.LayData.AllocSize.SetPoint(g.BBox2D().Size()) // get size from bbox -- minimal case
}

func (g *Node2DBase) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, false) // no style
	g.Layout2DChildren()
}

func (g *Node2DBase) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *Node2DBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	g.ComputeBBox2DBase(parBBox, delta)
}

func (g *Node2DBase) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox // pass-thru
}

func (g *Node2DBase) Move2D(delta image.Point, parBBox image.Rectangle) {
	g.Move2DBase(delta, parBBox)
	g.Move2DChildren(delta)
}

func (g *Node2DBase) Render2D() {
	if g.PushBounds() {
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *Node2DBase) ReRender2D() (node Node2D, layout bool) {
	node = g.This.(Node2D)
	layout = false
	return
}

func (g *Node2DBase) FocusChanged2D(gotFocus bool) {
}

func (g *Node2DBase) HasFocus2D() bool {
	return g.HasFocus()
}

// check for interface implementation
var _ Node2D = &Node2DBase{}

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

// redefining this here gives access to the much faster ParentWindow method!
func (g *Node2DBase) ReceiveEventType(et oswin.EventType, fun ki.RecvFunc) {
	win := g.ParentWindow()
	if win != nil {
		win.ReceiveEventType(g.This, et, fun)
	}
}

// handles basic node initialization -- Init2D can then do special things
func (g *Node2DBase) Init2DBase() {
	g.Viewport = g.ParentViewport()
	if g.Viewport != nil { // default for most cases -- delete connection of not
		// fmt.Printf("node %v connect to viewport %v\n", g.Nm, g.Viewport.Nm)
		g.NodeSig.Connect(g.Viewport.This, SignalViewport2D)
	}
	g.Style.Defaults()
	g.Paint.Defaults()
	g.LayData.Defaults() // doesn't overwrite
}

// Style2DSVG styles the Paint values directly from node properties -- for
// SVG-style nodes -- no relevant default styling here -- parents can just set
// props directly as needed
func (g *Node2DBase) Style2DSVG() {
	gii, _ := g.This.(Node2D)
	if g.Viewport == nil { // robust
		gii.Init2D()
	}
	pg := g.CopyParentPaint() // svg always inherits all paint settings from parent
	g.Paint.StyleSet = false  // this is always first call, restart
	if pg != nil {
		g.Paint.SetStyle(&pg.Paint, g.Properties())
	} else {
		g.Paint.SetStyle(nil, g.Properties())
	}
	g.Paint.SetUnitContext(g.Viewport, Vec2DZero)
}

// ReStyle2DSVG does a basic re-style using only current obj props and update of
// inherited values -- called for Parts when parent styles change
func (g *Node2DBase) ReStyle2DSVG() {
	g.Paint.StyleSet = false  // allow inherts
	pg := g.CopyParentPaint() // svg always inherits all paint settings from parent
	g.Paint.SetStyle(&pg.Paint, g.Properties())
	g.Paint.StyleSet = true
}

// DefaultStyle2DWidget retrieves default style object for the type, from type
// properties -- selector is optional selector for state etc.  Property key is
// "__DefStyle" + selector -- if part != nil, then use that obj for getting the
// default style starting point when creating a new style
func (g *Node2DBase) DefaultStyle2DWidget(selector string, part *Node2DBase) *Style {
	tprops := kit.Types.Properties(g.Type(), true) // true = makeNew
	styprops := tprops
	if selector != "" {
		sp, ok := tprops[selector]
		if !ok {
			log.Printf("gi.DefaultStyle2DWidget: did not find props for style selector: %v for node type: %v\n", selector, g.Type().Name())
		} else {
			spm, ok := sp.(ki.Props)
			if !ok {
				log.Printf("gi.DefaultStyle2DWidget: looking for a ki.Props for style selector: %v, instead got type: %T, for node type: %v\n", selector, spm, g.Type().Name())
			} else {
				styprops = spm
			}
		}
	}
	var dsty *Style
	pnm := "__DefStyle" + selector
	dstyi, ok := tprops[pnm]
	if !ok || RebuildDefaultStyles {
		dsty = &Style{}
		dsty.Defaults()
		if selector != "" {
			var baseStyle *Style
			if part != nil {
				baseStyle = part.DefaultStyle2DWidget("", nil)
			} else {
				baseStyle = g.DefaultStyle2DWidget("", nil)
			}
			*dsty = *baseStyle
		}
		dsty.SetStyle(nil, styprops)
		dsty.IsSet = false // keep as non-set
		tprops[pnm] = dsty
	} else {
		dsty, _ = dstyi.(*Style)
	}
	return dsty
}

// Style2DWidget styles the Style values from node properties and optional
// base-level defaults -- for Widget-style nodes
func (g *Node2DBase) Style2DWidget() {
	if !RebuildDefaultStyles && g.DefStyle != nil {
		g.Style.CopyFrom(g.DefStyle)
	} else {
		g.Style.CopyFrom(g.DefaultStyle2DWidget("", nil))
	}
	g.Style.IsSet = false // this is always first call, restart

	gii, _ := g.This.(Node2D)
	if g.Viewport == nil { // robust
		gii.Init2D()
	}
	_, pg := KiToNode2D(g.Par)
	if pg != nil {
		g.Style.SetStyle(&pg.Style, g.Properties())
	} else {
		g.Style.SetStyle(nil, g.Properties())
	}
	// then css:
	// if g.Viewport != nil {
	// 	g.Viewport.StyleCSStoMe(gii)
	// }
	g.Style.SetUnitContext(g.Viewport, Vec2DZero) // todo: test for use of el-relative
	g.Paint.PropsNil = true                       // not using paint props
	g.Paint.SetUnitContext(g.Viewport, Vec2DZero)
	g.LayData.SetFromStyle(&g.Style.Layout) // also does reset
}

func ApplyCSS(node Node2D, key string, css ki.Props) bool {
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(ki.Props) // must be a props map
	if !ok {
		return false
	}

	nb := node.AsNode2D()
	_, pg := KiToNode2D(node.Parent())
	if pg != nil {
		nb.Style.SetStyle(&pg.Style, pmap)
	} else {
		nb.Style.SetStyle(nil, pmap)
	}
	return true
}

func StyleCSSWidget(node Node2D, css ki.Props) {
	tyn := strings.ToLower(node.Type().Name()) // type is most general, first
	ApplyCSS(node, tyn, css)
	cln := "." + strings.ToLower(node.AsNode2D().Class) // then class
	ApplyCSS(node, cln, css)
	idnm := "#" + strings.ToLower(node.Name()) // then name
	ApplyCSS(node, idnm, css)
}

// StylePart sets the style properties for a child in parts (or any other
// child) based on its name -- only call this when new parts were created --
// name of properties is #partname (lower cased) and it should contain a
// ki.Props which is then added to the part's props -- this provides built-in
// defaults for parts, so it is separate from the CSS process
func (g *Node2DBase) StylePart(pk ki.Ki) {
	pgi, pg := KiToNode2D(pk)
	if pg.DefStyle != nil { // already set
		return
	}
	stynm := "#" + strings.ToLower(pk.Name())
	// this is called on US (the parent object) so we store the #partname
	// default style within our type properties..  that's good -- HOWEVER we
	// cannot put any sub-selector properties within these part styles -- must
	// all be in the base-level.. hopefully that works..
	pdst := g.DefaultStyle2DWidget(stynm, pg)
	pg.DefStyle = pdst // will use this as starting point for all styles now..

	if vp := pgi.AsViewport2D(); vp != nil {
		// this is typically an icon -- copy fill and stroke params to it
		styprops := kit.Types.Properties(g.Type(), true)
		sp := ki.SubProps(styprops, stynm)
		if sp != nil {
			if fill, ok := sp["fill"]; ok {
				pg.SetProp("fill", fill)
			}
			if stroke, ok := sp["stroke"]; ok {
				pg.SetProp("stroke", stroke)
			}
		}
		sp = ki.SubProps(g.Properties(), stynm)
		if sp != nil {
			if fill, ok := sp["fill"]; ok {
				pg.SetProp("fill", fill)
			}
			if stroke, ok := sp["stroke"]; ok {
				pg.SetProp("stroke", stroke)
			}
		}
	}
}

// ReStyle2DWidget does a basic re-style using only current obj props and update of
// inherited values -- called for Parts when parent styles change
func (g *Node2DBase) ReStyle2DWidget() {
	g.Style.IsSet = false // allow inherts
	_, pg := KiToNode2D(g.Par)
	if pg != nil {
		g.Style.SetStyle(&pg.Style, g.Properties())
	} else {
		g.Style.SetStyle(nil, g.Properties())
	}
	g.Style.IsSet = true
}

// CopyParentPaint copy our paint from our parents -- called during Style for
// SVG
func (g *Node2DBase) CopyParentPaint() *Node2DBase {
	_, pg := KiToNode2D(g.Par)
	if pg != nil {
		g.Paint.CopyFrom(&pg.Paint)
	}
	return pg
}

func (g *Node2DBase) InitLayout2D() {
	g.LayData.SetFromStyle(&g.Style.Layout)
}

// get our bbox from Layout allocation
func (g *Node2DBase) BBoxFromAlloc() image.Rectangle {
	return RectFromPosSize(g.LayData.AllocPos, g.LayData.AllocSize)
}

// set our window-level BBox from vp and our bbox
func (g *Node2DBase) SetWinBBox() {
	if g.Viewport != nil {
		g.WinBBox = g.VpBBox.Add(g.Viewport.WinBBox.Min)
	} else {
		g.WinBBox = g.VpBBox
	}
}

// add the position of our parent to our layout position -- layout
// computations are all relative to parent position, so they are finally
// cached out at this stage also returns the size of the parent for setting
// units context relative to parent objects
func (g *Node2DBase) AddParentPos() Vec2D {
	_, pg := KiToNode2D(g.Par)
	if pg != nil {
		if !g.IsField() {
			g.LayData.AllocPos = pg.LayData.AllocPos.Add(g.LayData.AllocPosRel)
		}
		return pg.LayData.AllocSize
	}
	return Vec2DZero
}

// ComputeBBox2DBase -- computes the VpBBox and WinBBox from BBox, with whatever delta may be in effect
func (g *Node2DBase) ComputeBBox2DBase(parBBox image.Rectangle, delta image.Point) {
	g.VpBBox = parBBox.Intersect(g.BBox.Add(delta))
	g.SetWinBBox()
}

// basic Layout2D functions -- good for most cases
func (g *Node2DBase) Layout2DBase(parBBox image.Rectangle, initStyle bool) {
	psize := g.AddParentPos()
	g.LayData.AllocPosOrig = g.LayData.AllocPos
	if initStyle {
		g.Style.SetUnitContext(g.Viewport, psize) // update units with final layout
	}
	g.Paint.SetUnitContext(g.Viewport, psize) // always update paint
	g.BBox = g.This.(Node2D).BBox2D()         // only compute once, at this point
	// note: if other styles are maintained, they also need to be updated!
	g.This.(Node2D).ComputeBBox2D(parBBox, image.ZP) // other bboxes from BBox
	// typically Layout2DChildren must be called after this!
	if Layout2DTrace {
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", g.PathUnique(), g.LayData.AllocPos, g.LayData.AllocSize, g.VpBBox, g.WinBBox)
	}
}

func (g *Node2DBase) Move2DBase(delta image.Point, parBBox image.Rectangle) {
	g.LayData.AllocPos = g.LayData.AllocPosOrig.Add(NewVec2DFmPoint(delta))
	g.This.(Node2D).ComputeBBox2D(parBBox, delta)
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
	if Render2DTrace {
		fmt.Printf("Render: %v at %v\n", g.PathUnique(), g.VpBBox)
	}
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
	updt := g.UpdateStart()
	g.Init2DTree()
	g.Style2DTree()
	g.Size2DTree()
	g.Layout2DTree()
	g.Render2DTree()
	g.UpdateEndNoSig(updt)
}

// re-render of the tree -- after it has already been initialized and styled
// -- just does layout and render passes
func (g *Node2DBase) ReRender2DTree() {
	ld := g.LayData // save our current layout data
	updt := g.UpdateStart()
	g.Style2DTree()
	g.Size2DTree()
	g.LayData = ld // restore
	g.Layout2DTree()
	g.Render2DTree()
	g.UpdateEndNoSig(updt)
}

// initialize scene graph tree from node it is called on -- only needs to be
// done once but must be robust to repeated calls -- use a flag if necessary
// -- needed after structural updates to ensure all nodes are updated
func (g *Node2DBase) Init2DTree() {
	pr := prof.Start("Node2D.Init2DTree")
	g.FuncDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, _ := KiToNode2D(k)
		if gii == nil {
			return false
		}
		gii.Init2D()
		return true
	})
	pr.End()
}

// style scene graph tree from node it is called on -- only needs to be
// done after a structural update in case inherited options changed
func (g *Node2DBase) Style2DTree() {
	pr := prof.Start("Node2D.Style2DTree")
	g.FuncDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, _ := KiToNode2D(k)
		if gii == nil {
			return false
		}
		gii.Style2D()
		return true
	})
	pr.End()
}

// StyleCSStoMe calls StyleCSS on the tree (typically from the top viewport)
// down from given node (inclusive) -- higher style settings are naturally
// overridden by lower ones
func (g *Node2DBase) StyleCSStoMe(me Node2D) {
	g.FuncDownMeFirst(0, me, func(k ki.Ki, level int, d interface{}) bool {
		gii, _ := KiToNode2D(k)
		if gii == nil {
			return false
		}
		gii.StyleCSS(me)
		if gii == me {
			return false
		} else {
			return true
		}
	})
}

// Restyle2DTree calls Restyle2D on the tree down from me (inclusive) -- called
// e.g., for Parts when parent styles change
func (g *Node2DBase) ReStyle2DTree() {
	g.FuncDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, _ := KiToNode2D(k)
		if gii == nil {
			return false
		}
		gii.ReStyle2D()
		return true
	})
}

// do the sizing as a depth-first pass
func (g *Node2DBase) Size2DTree() {
	pr := prof.Start("Node2D.Size2DTree")
	g.FuncDownDepthFirst(0, g.This,
		func(k ki.Ki, level int, d interface{}) bool {
			// this is for testing whether to process node
			_, gi := KiToNode2D(k)
			if gi == nil || gi.Paint.Off {
				fmt.Printf("bailing depth first size on %v\n", gi.PathUnique())
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
	pr.End()
}

// layout pass -- each node iterates over children for maximum control -- this starts with parent VpBBox -- can be called de novo
func (g *Node2DBase) Layout2DTree() {
	pr := prof.Start("Node2D.Layout2DTree")
	parBBox := image.ZR
	_, pg := KiToNode2D(g.Par)
	if pg != nil {
		parBBox = pg.VpBBox
	}
	g.This.(Node2D).Layout2D(parBBox) // important to use interface version to get interface!
	pr.End()
}

// move2d pass -- each node iterates over children for maximum control -- this starts with parent VpBBox and current delta -- can be called de novo
func (g *Node2DBase) Move2DTree() {
	parBBox := image.ZR
	_, pg := KiToNode2D(g.Par)
	if pg != nil {
		parBBox = pg.VpBBox
	}
	delta := g.LayData.AllocPos.Sub(g.LayData.AllocPosOrig).ToPoint()
	g.This.(Node2D).Move2D(delta, parBBox) // important to use interface version to get interface!
}

// render just calls on parent node and it takes full responsibility for
// managing the children -- this allows maximum flexibility for order etc of
// rendering
func (g *Node2DBase) Render2DTree() {
	pr := prof.Start("Node2D.Render2DTree")
	g.This.(Node2D).Render2D() // important to use interface version to get interface!
	pr.End()
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
	for _, kid := range g.Kids {
		gii, _ := KiToNode2D(kid)
		if gii != nil {
			gii.Layout2D(cbb)
		}
	}
}

// move all of node's children, giving them the ChildrenBBox2D -- default call at end of Move2D
func (g *Node2DBase) Move2DChildren(delta image.Point) {
	cbb := g.This.(Node2D).ChildrenBBox2D()
	for _, kid := range g.Kids {
		gii, _ := KiToNode2D(kid)
		if gii != nil {
			gii.Move2D(delta, cbb)
		}
	}
}

// render all of node's children -- default call at end of Render2D()
func (g *Node2DBase) Render2DChildren() {
	for _, kid := range g.Kids {
		gii, _ := KiToNode2D(kid)
		if gii != nil {
			gii.Render2D()
		}
	}
}

// report on all the bboxes for everything in the tree
func (g *Node2DBase) BBoxReport() string {
	rpt := ""
	g.FuncDownMeFirst(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, gi := KiToNode2D(k)
		if gii == nil {
			return false
		}
		rpt += fmt.Sprintf("%v: vp: %v, win: %v\n", gi.Nm, gi.VpBBox, gi.WinBBox)
		return true
	})
	return rpt
}

func (g *Node2DBase) ParentWindow() *Window {
	if g.Viewport != nil && g.Viewport.Win != nil {
		return g.Viewport.Win
	}
	wini := g.ParentByType(KiT_Window, true)
	if wini == nil {
		// log.Printf("Node %v ReceiveEventType -- cannot find parent window -- must be called after adding to the scenegraph\n", g.PathUnique())
		return nil
	}
	return wini.EmbeddedStruct(KiT_Window).(*Window)
}

// find parent viewport -- uses AsViewport2D() method on Node2D interface
func (g *Node2DBase) ParentViewport() *Viewport2D {
	var parVp *Viewport2D
	g.FuncUpParent(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
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

func (g *Node2DBase) ParentSVG() *SVG {
	pvp := g.ParentViewport()
	for pvp != nil {
		if pvp.IsSVG() {
			return pvp.This.(*SVG)
		}
		pvp = pvp.ParentViewport()
	}
	return nil
}

func (g *Node2DBase) ParentLayout() *Layout {
	var parLy *Layout
	g.FuncUpParent(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, ok := k.(Node2D)
		if !ok {
			return false // don't keep going up
		}
		ly := gii.AsLayout2D()
		if ly != nil {
			parLy = ly
			return false // done
		}
		return true
	})
	return parLy
}

// find parent that is a ReRenderAnchor
func (g *Node2DBase) ParentReRenderAnchor() Node2D {
	var par Node2D
	g.FuncUpParent(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		gii, gi := KiToNode2D(k)
		if gii == nil {
			return false // don't keep going up
		}
		if gi.IsReRenderAnchor() {
			par = gii
			return false
		}
		return true
	})
	return par
}
