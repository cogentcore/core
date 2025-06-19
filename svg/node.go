// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"maps"
	"reflect"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Node is the interface for all SVG nodes.
type Node interface {
	tree.Node

	// AsNodeBase returns the [NodeBase] for our node, which gives
	// access to all the base-level data structures and methods
	// without requiring interface methods.
	AsNodeBase() *NodeBase

	// BBoxes computes BBox and VisBBox, prior to render.
	BBoxes(sv *SVG, parTransform math32.Matrix2)

	// Render draws the node to the svg image.
	Render(sv *SVG)

	// LocalBBox returns the bounding box of node in local dimensions.
	LocalBBox(sv *SVG) math32.Box2

	// SetNodePos sets the upper left effective position of this element, in local dimensions.
	SetNodePos(pos math32.Vector2)

	// SetNodeSize sets the overall effective size of this element, in local dimensions.
	SetNodeSize(sz math32.Vector2)

	// ApplyTransform applies the given 2D transform to the geometry of this node
	// this just does a direct transform multiplication on coordinates.
	ApplyTransform(sv *SVG, xf math32.Matrix2)

	// SVGName returns the SVG element name (e.g., "rect", "path" etc).
	SVGName() string

	// EnforceSVGName returns true if in general this element should
	// be named with its SVGName plus a unique id.
	// Groups and Markers are false.
	EnforceSVGName() bool
}

// NodeBase is the base type for all elements within an SVG tree.
// It implements the [Node] interface and contains the core functionality.
type NodeBase struct {
	tree.NodeBase

	// Class contains user-defined class name(s) used primarily for attaching
	// CSS styles to different display elements.
	// Multiple class names can be used to combine properties;
	// use spaces to separate per css standard.
	Class string

	// CSS is the cascading style sheet at this level.
	// These styles apply here and to everything below, until superceded.
	// Use .class and #name Properties elements to apply entire styles
	// to given elements, and type for element type.
	CSS map[string]any `xml:"css" set:"-"`

	// CSSAgg is the aggregated css properties from all higher nodes down to this node.
	CSSAgg map[string]any `copier:"-" json:"-" xml:"-" set:"-" display:"no-inline"`

	// BBox is the bounding box for the node within the SVG Pixels image.
	// This one can be outside the visible range of the SVG image.
	// VisBBox is intersected and only shows visible portion.
	BBox math32.Box2 `copier:"-" json:"-" xml:"-" set:"-"`

	// VisBBox is the visible bounding box for the node intersected with the SVG image geometry.
	VisBBox math32.Box2 `copier:"-" json:"-" xml:"-" set:"-"`

	// Paint is the paint style information for this node.
	Paint styles.Paint `json:"-" xml:"-" set:"-"`

	// GradientFill contains the fill gradient geometry to use for linear and radial
	// gradients of UserSpaceOnUse type applied to this node.
	// These values are updated and copied to gradients of the appropriate type to keep
	// the gradients sync'd with updates to the node as it is transformed.
	GradientFill math32.Matrix2 `json:"-" xml:"-" set:"-" display:"-"`

	// GradientStroke contains the stroke gradient geometry to use for linear and radial
	// gradients of UserSpaceOnUse type applied to this node.
	// These values are updated and copied to gradients of the appropriate type to keep
	// the gradients sync'd with updates to the node as it is transformed.
	GradientStroke math32.Matrix2 `json:"-" xml:"-" set:"-" display:"-"`

	// isDef is whether this is in [SVG.Defs].
	isDef bool
}

func (g *NodeBase) AsNodeBase() *NodeBase         { return g }
func (g *NodeBase) SVGName() string               { return "base" }
func (g *NodeBase) EnforceSVGName() bool          { return true }
func (g *NodeBase) SetPos(pos math32.Vector2)     {}
func (g *NodeBase) SetSize(sz math32.Vector2)     {}
func (g *NodeBase) PaintStyle() *styles.Paint     { return &g.Paint }
func (g *NodeBase) LocalBBox(sv *SVG) math32.Box2 { return math32.Box2{} }
func (g *NodeBase) BaseInterface() reflect.Type   { return reflect.TypeOf((*NodeBase)(nil)).Elem() }
func (g *NodeBase) SetTransformProperty()         { g.SetProperty("transform", g.Paint.Transform.String()) }

func (g *NodeBase) Init() {
	g.Paint.Defaults()
	g.GradientFill = math32.Identity2()
	g.GradientStroke = math32.Identity2()
	g.Paint.Stroke.Width.Px(1) // dp is not understood by svg..
}

// SetColorProperties sets color property from a string representation.
// It breaks color alpha out as opacity.  prop is either "stroke" or "fill"
func (g *NodeBase) SetColorProperties(prop, color string) {
	if NameFromURL(color) != "" {
		return
	}
	if color == "none" || color == "" {
		g.SetProperty(prop, "none")
		g.DeleteProperty(prop + "-opacity")
		if prop == "stroke" {
			g.DeleteProperty("stroke-width")
		}
		return
	}
	clr, _ := colors.FromString(color)
	g.SetProperty(prop+"-opacity", fmt.Sprintf("%g", float32(clr.A)/255))
	// we have consumed the A via opacity, so we reset it to 255
	clr.A = 255
	g.SetProperty(prop, colors.AsHex(clr))
}

// ParentTransform returns the full compounded 2D transform matrix for all
// of the parents of this node. If self is true, then include our
// own transform too.
func (g *NodeBase) ParentTransform(self bool) math32.Matrix2 {
	pars := []Node{}
	xf := math32.Identity2()
	n := g.This.(Node)
	for {
		if n.AsTree().Parent == nil {
			break
		}
		n = n.AsTree().Parent.(Node)
		pars = append(pars, n)
	}
	np := len(pars)
	if np > 0 {
		xf = pars[np-1].AsNodeBase().PaintStyle().Transform
	}
	for i := np - 2; i >= 0; i-- {
		n := pars[i]
		xf.SetMul(n.AsNodeBase().PaintStyle().Transform)
	}
	if self {
		xf.SetMul(g.Paint.Transform)
	}
	return xf
}

// ApplyTransform applies the given 2D transform to the geometry of this node.
func (g *NodeBase) ApplyTransform(sv *SVG, xf math32.Matrix2) {
}

// DeltaTransform computes the net transform matrix for given delta transform parameters,
// operating around given reference point which serves as the effective origin for rotation.
func (g *NodeBase) DeltaTransform(trans math32.Vector2, scale math32.Vector2, rot float32, pt math32.Vector2) math32.Matrix2 {
	mxi := g.ParentTransform(true).Inverse()
	lpt := mxi.MulVector2AsPoint(pt)
	ltr := mxi.MulVector2AsVector(trans)
	xf := math32.Translate2D(lpt.X, lpt.Y).Scale(scale.X, scale.Y).Rotate(rot).Translate(ltr.X, ltr.Y).Translate(-lpt.X, -lpt.Y)
	return xf
}

// SVGWalkDown does [tree.NodeBase.WalkDown] on given node using given walk function
// with SVG Node parameters.
func SVGWalkDown(n Node, fun func(sn Node, snb *NodeBase) bool) {
	n.AsTree().WalkDown(func(n tree.Node) bool {
		sn := n.(Node)
		return fun(sn, sn.AsNodeBase())
	})
}

// SVGWalkDownNoDefs does [tree.Node.WalkDown] on given node using given walk function
// with SVG Node parameters. Automatically filters Defs nodes (IsDef) and MetaData,
// i.e., it only processes concrete graphical nodes.
func SVGWalkDownNoDefs(n Node, fun func(sn Node, snb *NodeBase) bool) {
	n.AsTree().WalkDown(func(cn tree.Node) bool {
		sn := cn.(Node)
		snb := sn.AsNodeBase()
		_, md := sn.(*MetaData)
		if snb.isDef || md {
			return tree.Break
		}
		return fun(sn, snb)
	})
}

// FirstNonGroupNode returns the first item that is not a group
// recursing into groups until a non-group item is found.
func FirstNonGroupNode(n Node) Node {
	var ngn Node
	SVGWalkDownNoDefs(n, func(sn Node, snb *NodeBase) bool {
		if _, isgp := sn.(*Group); isgp {
			return tree.Continue
		}
		ngn = sn
		return tree.Break
	})
	return ngn
}

//////// Standard Node infrastructure

// Style styles the Paint values directly from node properties
func (g *NodeBase) Style(sv *SVG) {
	pc := &g.Paint
	pc.Defaults()
	ctxt := colors.Context(sv)
	pc.StyleSet = false // this is always first call, restart

	var parCSSAgg map[string]any
	if g.Parent != nil { // && g.Par != sv.Root.This
		pn := g.Parent.(Node)
		parCSSAgg = pn.AsNodeBase().CSSAgg
		pp := pn.AsNodeBase().PaintStyle()
		pc.CopyStyleFrom(pp)
		pc.SetProperties(pp, g.Properties, ctxt)
	} else {
		pc.SetProperties(nil, g.Properties, ctxt)
	}
	pc.ToDotsImpl(&pc.UnitContext) // we always inherit parent's unit context -- SVG sets it once-and-for-all

	if parCSSAgg != nil {
		AggCSS(&g.CSSAgg, parCSSAgg)
	} else {
		g.CSSAgg = nil
	}
	AggCSS(&g.CSSAgg, g.CSS)
	g.StyleCSS(sv, g.CSSAgg)
	pc.Stroke.Opacity *= pc.Opacity // applies to all
	pc.Fill.Opacity *= pc.Opacity
	pc.Off = (pc.Stroke.Color == nil && pc.Fill.Color == nil)
}

// AggCSS aggregates css properties
func AggCSS(agg *map[string]any, css map[string]any) {
	if *agg == nil {
		*agg = make(map[string]any)
	}
	maps.Copy(*agg, css)
}

// ApplyCSS applies css styles to given node,
// using key to select sub-properties from overall properties list
func (g *NodeBase) ApplyCSS(sv *SVG, key string, css map[string]any) bool {
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(map[string]any) // must be a properties map
	if !ok {
		return false
	}
	pc := &g.Paint
	ctxt := colors.Context(sv)
	if g.Parent != sv.Root.This {
		pp := g.Parent.(Node).AsNodeBase().PaintStyle()
		pc.SetProperties(pp, pmap, ctxt)
	} else {
		pc.SetProperties(nil, pmap, ctxt)
	}
	return true
}

// StyleCSS applies css style properties to given SVG node
// parsing out type, .class, and #name selectors
func (g *NodeBase) StyleCSS(sv *SVG, css map[string]any) {
	tyn := strings.ToLower(g.NodeType().Name) // type is most general, first
	g.ApplyCSS(sv, tyn, css)
	cln := "." + strings.ToLower(g.Class) // then class
	g.ApplyCSS(sv, cln, css)
	idnm := "#" + strings.ToLower(g.Name) // then name
	g.ApplyCSS(sv, idnm, css)
}

func (g *NodeBase) SetNodePos(pos math32.Vector2) {
	// no-op by default
}

func (g *NodeBase) SetNodeSize(sz math32.Vector2) {
	// no-op by default
}

// LocalLineWidth returns the line width in local coordinates
func (g *NodeBase) LocalLineWidth() float32 {
	pc := &g.Paint
	if pc.Stroke.Color == nil {
		return 0
	}
	return pc.Stroke.Width.Dots
}

func (g *NodeBase) BBoxes(sv *SVG, parTransform math32.Matrix2) {
	xf := parTransform.Mul(g.Paint.Transform)
	ni := g.This.(Node)
	lbb := ni.LocalBBox(sv)
	g.BBox = lbb.MulMatrix2(xf)
	g.VisBBox = sv.Geom.Box2().Intersect(g.BBox)
}

// IsVisible checks our bounding box and visibility, returning false if
// out of bounds. Must be called as first step in Render.
func (g *NodeBase) IsVisible(sv *SVG) bool {
	if g == nil || g.This == nil || g.Paint.Off || !g.Paint.Display {
		return false
	}
	nvis := g.VisBBox == (math32.Box2{})
	if nvis && !g.isDef {
		// fmt.Println("invisible:", g.Name, "bb:", g.BBox, "vbb:", g.VisBBox, "svg:", sv.Geom.Bounds())
		return false
	}
	return true
}

// Painter returns a new Painter using my styles.
func (g *NodeBase) Painter(sv *SVG) *paint.Painter {
	return &paint.Painter{sv.painter.State, &g.Paint}
}

// PushContext checks our bounding box and visibility, returning false if
// out of bounds. If visible, pushes us as Context.
// Must be called as first step in Render.
func (g *NodeBase) PushContext(sv *SVG) bool {
	if !g.IsVisible(sv) {
		return false
	}
	pc := g.Painter(sv)
	pc.PushContext(&g.Paint, nil)
	return true
}

func (g *NodeBase) BBoxesFromChildren(sv *SVG, parTransform math32.Matrix2) {
	xf := parTransform.Mul(g.Paint.Transform)
	var bb math32.Box2
	for i, kid := range g.Children {
		ni := kid.(Node)
		ni.BBoxes(sv, xf)
		nb := ni.AsNodeBase()
		if i == 0 {
			bb = nb.BBox
		} else {
			bb = bb.Union(nb.BBox)
		}
	}
	g.BBox = bb
	g.VisBBox = sv.Geom.Box2().Intersect(g.BBox)
}

func (g *NodeBase) RenderChildren(sv *SVG) {
	for _, kid := range g.Children {
		ni := kid.(Node)
		ni.Render(sv)
	}
}

func (g *NodeBase) Render(sv *SVG) {
	if !g.IsVisible(sv) {
		return
	}
	g.RenderChildren(sv)
}

// BitCloneNode returns a bit-wise copy of just the single svg Node itself
// without any of the children, props or other state being copied, etc.
// Useful for saving and restoring state during animations or other
// manipulations. See also [CopyFrom].
func BitCloneNode(n Node) Node {
	cp := n.AsTree().NewInstance().(Node)
	BitCopyFrom(cp, n)
	return cp
}

// BitCopyFrom copies only the direct field bits and other key shape data
// (e.g., [Path.Data]) between nodes. Useful for saving and restoring
// state during animations or other manipulations.
func BitCopyFrom(to, fm any) {
	reflectx.Underlying(reflect.ValueOf(to)).Set(reflectx.Underlying(reflect.ValueOf(fm)))
	switch x := to.(type) {
	case *Path:
		x.Data = fm.(*Path).Data.Clone()
	case *Polyline:
		slicesx.CopyFrom(x.Points, fm.(*Polyline).Points)
	case *Polygon:
		slicesx.CopyFrom(x.Points, fm.(*Polygon).Points)
	}
}
