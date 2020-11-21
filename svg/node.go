// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"image"
	"log"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// svg.NodeBase is an element within the SVG sub-scenegraph -- does not use
// layout logic -- just renders into parent SVG viewport
type NodeBase struct {
	gi.Node2DBase
	Pnt girl.Paint `json:"-" xml:"-" desc:"full paint information for this node"`
}

var KiT_NodeBase = kit.Types.AddType(&NodeBase{}, NodeBaseProps)

var NodeBaseProps = ki.Props{
	"base-type":     true, // excludes type from user selections
	"EnumType:Flag": gi.KiT_NodeFlags,
}

func (g *NodeBase) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*NodeBase)
	g.Node2DBase.CopyFieldsFrom(&fr.Node2DBase)
	g.Pnt = fr.Pnt
}

func (g *NodeBase) AsSVGNode() *NodeBase {
	return g
}

// Paint satisfies the painter interface
func (g *NodeBase) Paint() *gist.Paint {
	return &g.Pnt.Paint
}

// Init2DBase handles basic node initialization -- Init2D can then do special things
func (g *NodeBase) Init2DBase() {
	g.BBoxMu.Lock()
	g.Viewport = g.ParentViewport()
	g.Pnt.Defaults()
	g.BBoxMu.Unlock()
	g.ConnectToViewport()
}

func (g *NodeBase) Init2D() {
	g.Init2DBase()
	g.SetFlag(int(gi.NoLayout))
}

// StyleSVG styles the Paint values directly from node properties -- no
// relevant default styling here -- parents can just set props directly as
// needed
func StyleSVG(gii gi.Node2D) {
	g := gii.AsNode2D()
	mvp := g.ViewportSafe()
	if mvp == nil { // robust
		gii.Init2D()
	}

	pntr, ok := gii.(gist.Painter)
	if !ok {
		return
	}
	pc := pntr.Paint()

	// todo: do StyleMu for SVG nodes, then can access viewport directly
	mvp = g.ViewportSafe()

	mvp.SetCurStyleNode(gii)
	defer mvp.SetCurStyleNode(nil)

	pc.StyleSet = false // this is always first call, restart

	pp := g.ParentPaint()
	if pp != nil {
		pc.CopyStyleFrom(pp)
		pc.SetStyleProps(pp, *gii.Properties(), g.Viewport)
	} else {
		pc.SetStyleProps(nil, *gii.Properties(), g.Viewport)
	}
	// pc.SetUnitContext(g.Viewport, mat32.Vec2Zero)
	pc.ToDotsImpl(&pc.UnContext) // we always inherit parent's unit context -- SVG sets it once-and-for-all

	pagg := g.ParentCSSAgg()
	if pagg != nil {
		gi.AggCSS(&g.CSSAgg, *pagg)
	} else {
		g.CSSAgg = nil
	}
	gi.AggCSS(&g.CSSAgg, g.CSS)
	StyleCSS(gii, g.CSSAgg)
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	} else {
		pc.Off = false
	}
}

// ApplyCSSSVG applies css styles to given node, using key to select sub-props
// from overall properties list
func ApplyCSSSVG(node gi.Node2D, key string, css ki.Props) bool {
	pntr, ok := node.(gist.Painter)
	if !ok {
		return false
	}
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(ki.Props) // must be a props map
	if !ok {
		return false
	}
	nb := node.AsNode2D()
	pc := pntr.Paint()

	if pgi, _ := gi.KiToNode2D(node.Parent()); pgi != nil {
		if pp, ok := pgi.(gist.Painter); ok {
			pc.SetStyleProps(pp.Paint(), pmap, nb.Viewport)
		} else {
			pc.SetStyleProps(nil, pmap, nb.Viewport)
		}
	} else {
		pc.SetStyleProps(nil, pmap, nb.Viewport)
	}
	return true
}

// StyleCSS applies css style properties to given SVG node, parsing
// out type, .class, and #name selectors
func StyleCSS(node gi.Node2D, css ki.Props) {
	tyn := strings.ToLower(node.Type().Name()) // type is most general, first
	ApplyCSSSVG(node, tyn, css)
	cln := "." + strings.ToLower(node.AsNode2D().Class) // then class
	ApplyCSSSVG(node, cln, css)
	idnm := "#" + strings.ToLower(node.Name()) // then name
	ApplyCSSSVG(node, idnm, css)
}

func (g *NodeBase) Style2D() {
	StyleSVG(g.This().(gi.Node2D))
}

// ParentSVG returns the parent SVG viewport
func (g *NodeBase) ParentSVG() *SVG {
	pvp := g.ParentViewport()
	for pvp != nil {
		if pvp.IsSVG() {
			return pvp.This().Embed(KiT_SVG).(*SVG)
		}
		pvp = pvp.ParentViewport()
	}
	return nil
}

func (g *NodeBase) Size2D(iter int) {
}

func (g *NodeBase) Layout2D(parBBox image.Rectangle, iter int) bool {
	return false
}

func (g *NodeBase) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	return rs.LastRenderBBox
}

func (g *NodeBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
}

func (g *NodeBase) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

// ComputeBBoxSVG is called by default in render to compute bounding boxes for
// gui interaction -- can only be done in rendering because that is when all
// the proper xforms are all in place -- VpBBox is intersected with parent SVG
func (g *NodeBase) ComputeBBoxSVG() {
	if g.This() == nil {
		return
	}
	g.BBoxMu.Lock()
	g.BBox = g.This().(gi.Node2D).BBox2D()
	g.ObjBBox = g.BBox // no diff
	pbbox := g.Viewport.This().(gi.Node2D).ChildrenBBox2D()
	g.VpBBox = pbbox.Intersect(g.ObjBBox)
	g.BBoxMu.Unlock()
	g.SetWinBBox()

	if gi.Render2DTrace {
		fmt.Printf("Render: %v at %v\n", g.PathUnique(), g.VpBBox)
	}
}

func (g *NodeBase) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := g.Render()
	rs.PushXFormLock(pc.XForm)
	// render path elements, then compute bbox, then fill / stroke
	g.ComputeBBoxSVG()
	g.Render2DChildren()
	rs.PopXFormLock()
}

func (g *NodeBase) Move2D(delta image.Point, parBBox image.Rectangle) {
}

// FindSVGURL finds a url element in the parent SVG -- returns nil if not
// found -- can pass full 'url(#Name)' string
func (g *NodeBase) FindSVGURL(url string) gi.Node2D {
	if url == "none" {
		return nil
	}
	url = strings.TrimPrefix(url, "url(")
	url = strings.TrimSuffix(url, ")")
	rv := g.FindNamedElement(url)
	if rv == nil {
		log.Printf("gi.svg FindSVGURL could not find element named: %v in parents of svg el: %v\n", url, g.PathUnique())
	}
	return rv
}

// Marker checks for a marker property of given name, or generic "marker"
// type, and if set, attempts to find that marker and return it
func (g *NodeBase) Marker(marker string) *Marker {
	ms, ok := g.Props[marker]
	if !ok {
		ms, ok = g.Props["marker"]
		if !ok {
			return nil
		}
	}
	mnm, ok := ms.(string)
	if !ok {
		mrk, ok := ms.(*Marker)
		if !ok {
			log.Printf("gi.svg Marker property should be a string url or pointer to Marker element, instead is: %T\n", ms)
			return nil
		}
		return mrk
	}
	mrkn := g.FindSVGURL(mnm)
	if mrkn != nil {
		mrk, ok := mrkn.(*Marker)
		if !ok {
			log.Printf("gi.svg Found element named: %v but isn't a Marker type, instead is: %T", mnm, mrkn)
			return nil
		}
		return mrk
	}
	return nil
}
