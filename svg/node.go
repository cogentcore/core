// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"image"
	"maps"
	"reflect"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Node is the interface for all SVG nodes
type Node interface {
	tree.Node

	// AsNodeBase returns a generic svg.NodeBase for our node -- gives generic
	// access to all the base-level data structures without requiring
	// interface methods.
	AsNodeBase() *NodeBase

	// PaintStyle returns the SVG Paint style object for this node
	PaintStyle() *styles.Paint

	// Style updates the Paint style for this node
	Style(sv *SVG)

	// Render draws the node to the svg image
	Render(sv *SVG)

	// BBoxes computes BBox and VisBBox during Render
	BBoxes(sv *SVG)

	// LocalBBox returns the bounding box of node in local dimensions
	LocalBBox() math32.Box2

	// NodeBBox returns the bounding box in image coordinates for this node
	NodeBBox(sv *SVG) image.Rectangle

	// SetNodePos sets the upper left effective position of this element, in local dimensions
	SetNodePos(pos math32.Vector2)

	// SetNodeSize sets the overall effective size of this element, in local dimensions
	SetNodeSize(sz math32.Vector2)

	// ApplyTransform applies the given 2D transform to the geometry of this node
	// this just does a direct transform multiplication on coordinates.
	ApplyTransform(sv *SVG, xf math32.Matrix2)

	// ApplyDeltaTransform applies the given 2D delta transforms to the geometry of this node
	// relative to given point.  Trans translation and point are in top-level coordinates,
	// so must be transformed into local coords first.
	// Point is upper left corner of selection box that anchors the translation and scaling,
	// and for rotation it is the center point around which to rotate
	ApplyDeltaTransform(sv *SVG, trans math32.Vector2, scale math32.Vector2, rot float32, pt math32.Vector2)

	// WriteGeom writes the geometry of the node to a slice of floating point numbers
	// the length and ordering of which is specific to each node type.
	// Slice must be passed and will be resized if not the correct length.
	WriteGeom(sv *SVG, dat *[]float32)

	// ReadGeom reads the geometry of the node from a slice of floating point numbers
	// the length and ordering of which is specific to each node type.
	ReadGeom(sv *SVG, dat []float32)

	// SVGName returns the SVG element name (e.g., "rect", "path" etc)
	SVGName() string

	// EnforceSVGName returns true if in general this element should
	// be named with its SVGName plus a unique id.
	// Groups and Markers are false.
	EnforceSVGName() bool
}

// svg.NodeBase is the base type for elements within the SVG scenegraph
type NodeBase struct {
	tree.NodeBase

	// user-defined class name(s) used primarily for attaching
	// CSS styles to different display elements.
	// Multiple class names can be used to combine properties:
	// use spaces to separate per css standard.
	Class string

	// CSS is the cascading style sheet at this level.
	// These styles apply here and to everything below, until superceded.
	// Use .class and #name Properties elements to apply entire styles
	// to given elements, and type for element type.
	CSS map[string]any `xml:"css" set:"-"`

	// CSSAgg is the aggregated css properties from all higher nodes down to this node.
	CSSAgg map[string]any `copier:"-" json:"-" xml:"-" set:"-" view:"no-inline"`

	// bounding box for the node within the SVG Pixels image.
	// This one can be outside the visible range of the SVG image.
	// VisBBox is intersected and only shows visible portion.
	BBox image.Rectangle `copier:"-" json:"-" xml:"-" set:"-"`

	// visible bounding box for the node intersected with the SVG image geometry
	VisBBox image.Rectangle `copier:"-" json:"-" xml:"-" set:"-"`

	// paint style information for this node
	Paint styles.Paint `json:"-" xml:"-" set:"-"`
}

func (g *NodeBase) AsNodeBase() *NodeBase {
	return g
}

func (g *NodeBase) SVGName() string {
	return "base"
}

func (g *NodeBase) EnforceSVGName() bool {
	return true
}

func (g *NodeBase) SetPos(pos math32.Vector2) {
}

func (g *NodeBase) SetSize(sz math32.Vector2) {
}

func (g *NodeBase) LocalBBox() math32.Box2 {
	bb := math32.Box2{}
	return bb
}

func (n *NodeBase) BaseIface() reflect.Type {
	return reflect.TypeOf((*NodeBase)(nil)).Elem()
}

func (g *NodeBase) PaintStyle() *styles.Paint {
	return &g.Paint
}

// SetColorProperties sets color property from a string representation.
// It breaks color alpha out as opacity.  prop is either "stroke" or "fill"
func (g *NodeBase) SetColorProperties(prop, color string) {
	clr := errors.Log1(colors.FromString(color))
	g.SetProperty(prop+"-opacity", fmt.Sprintf("%g", float32(clr.A)/255))
	// we have consumed the A via opacity, so we reset it to 255
	clr.A = 255
	g.SetProperty(prop, colors.AsHex(clr))
}

// ParTransform returns the full compounded 2D transform matrix for all
// of the parents of this node.  If self is true, then include our
// own transform too.
func (g *NodeBase) ParTransform(self bool) math32.Matrix2 {
	pars := []Node{}
	xf := math32.Identity2()
	n := g.This().(Node)
	for {
		if n.Parent() == nil {
			break
		}
		n = n.Parent().(Node)
		pars = append(pars, n)
	}
	np := len(pars)
	if np > 0 {
		xf = pars[np-1].PaintStyle().Transform
	}
	for i := np - 2; i >= 0; i-- {
		n := pars[i]
		xf.SetMul(n.PaintStyle().Transform)
	}
	if self {
		xf.SetMul(g.Paint.Transform)
	}
	return xf
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// this just does a direct transform multiplication on coordinates.
func (g *NodeBase) ApplyTransform(sv *SVG, xf math32.Matrix2) {
}

// DeltaTransform computes the net transform matrix for given delta transform parameters
// and the transformed version of the reference point.  If self is true, then
// include the current node self transform, otherwise don't.  Groups do not
// but regular rendering nodes do.
func (g *NodeBase) DeltaTransform(trans math32.Vector2, scale math32.Vector2, rot float32, pt math32.Vector2, self bool) (math32.Matrix2, math32.Vector2) {
	mxi := g.ParTransform(self)
	mxi = mxi.Inverse()
	lpt := mxi.MulVector2AsPoint(pt)
	ldel := mxi.MulVector2AsVector(trans)
	xf := math32.Scale2D(scale.X, scale.Y).Rotate(rot)
	xf.X0 = ldel.X
	xf.Y0 = ldel.Y
	return xf, lpt
}

// ApplyDeltaTransform applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *NodeBase) ApplyDeltaTransform(sv *SVG, trans math32.Vector2, scale math32.Vector2, rot float32, pt math32.Vector2) {
}

// SetFloat32SliceLen is a utility function to set given slice of float32 values
// to given length, reusing existing where possible and making a new one as needed.
// For use in WriteGeom routines.
func SetFloat32SliceLen(dat *[]float32, sz int) {
	switch {
	case len(*dat) == sz:
	case len(*dat) < sz:
		if cap(*dat) >= sz {
			*dat = (*dat)[0:sz]
		} else {
			*dat = make([]float32, sz)
		}
	default:
		*dat = (*dat)[0:sz]
	}
}

// WriteTransform writes the node transform to slice at starting index.
// slice must already be allocated sufficiently.
func (g *NodeBase) WriteTransform(dat []float32, idx int) {
	dat[idx+0] = g.Paint.Transform.XX
	dat[idx+1] = g.Paint.Transform.YX
	dat[idx+2] = g.Paint.Transform.XY
	dat[idx+3] = g.Paint.Transform.YY
	dat[idx+4] = g.Paint.Transform.X0
	dat[idx+5] = g.Paint.Transform.Y0
}

// ReadTransform reads the node transform from slice at starting index.
func (g *NodeBase) ReadTransform(dat []float32, idx int) {
	g.Paint.Transform.XX = dat[idx+0]
	g.Paint.Transform.YX = dat[idx+1]
	g.Paint.Transform.XY = dat[idx+2]
	g.Paint.Transform.YY = dat[idx+3]
	g.Paint.Transform.X0 = dat[idx+4]
	g.Paint.Transform.Y0 = dat[idx+5]
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *NodeBase) WriteGeom(sv *SVG, dat *[]float32) {
	SetFloat32SliceLen(dat, 6)
	g.WriteTransform(*dat, 0)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *NodeBase) ReadGeom(sv *SVG, dat []float32) {
	g.ReadTransform(dat, 0)
}

// SVGWalkPre does [tree.Node.WalkPre] on given node using given walk function
// with SVG Node parameters.  Automatically filters
// nil or deleted items.  Return [tree.Continue] (true) to continue,
// and [tree.Break] (false) to terminate.
func SVGWalkPre(n Node, fun func(kni Node, knb *NodeBase) bool) {
	n.WalkDown(func(k tree.Node) bool {
		kni := k.(Node)
		if kni == nil || kni.This() == nil {
			return tree.Break
		}
		return fun(kni, kni.AsNodeBase())
	})
}

// SVGWalkPreNoDefs does [tree.Node.WalkPre] on given node using given walk function
// with SVG Node parameters.  Automatically filters
// nil or deleted items, and Defs nodes (IsDef) and MetaData,
// i.e., it only processes concrete graphical nodes.
// Return [tree.Continue] (true) to continue, and [tree.Break] (false) to terminate.
func SVGWalkPreNoDefs(n Node, fun func(kni Node, knb *NodeBase) bool) {
	n.WalkDown(func(k tree.Node) bool {
		kni := k.(Node)
		if kni == nil || kni.This() == nil {
			return tree.Break
		}
		if kni.Is(IsDef) || kni.NodeType() == MetaDataType {
			return tree.Break
		}
		return fun(kni, kni.AsNodeBase())
	})
}

// FirstNonGroupNode returns the first item that is not a group
// recursing into groups until a non-group item is found.
func FirstNonGroupNode(n Node) Node {
	var ngn Node
	SVGWalkPreNoDefs(n, func(kni Node, knb *NodeBase) bool {
		if _, isgp := kni.This().(*Group); isgp {
			return tree.Continue
		}
		ngn = kni
		return tree.Break
	})
	return ngn
}

// NodesContainingPoint returns all Nodes with Bounding Box that contains
// given point, optionally only those that are terminal nodes (no leaves).
// Excludes the starting node.
func NodesContainingPoint(n Node, pt image.Point, leavesOnly bool) []Node {
	var cn []Node
	SVGWalkPre(n, func(kni Node, knb *NodeBase) bool {
		if kni.This() == n.This() {
			return tree.Continue
		}
		if leavesOnly && kni.HasChildren() {
			return tree.Continue
		}
		if knb.Paint.Off {
			return tree.Break
		}
		if pt.In(knb.BBox) {
			cn = append(cn, kni)
		}
		return tree.Continue
	})
	return cn
}

//////////////////////////////////////////////////////////////////
// Standard Node infrastructure

// Style styles the Paint values directly from node properties
func (g *NodeBase) Style(sv *SVG) {
	pc := &g.Paint
	pc.Defaults()
	ctxt := colors.Context(sv)
	pc.StyleSet = false // this is always first call, restart

	var parCSSAgg map[string]any
	if g.Par != nil { // && g.Par != sv.Root.This()
		pn := g.Par.(Node)
		parCSSAgg = pn.AsNodeBase().CSSAgg
		pp := pn.PaintStyle()
		pc.CopyStyleFrom(pp)
		pc.SetStyleProperties(pp, g.Properties(), ctxt)
	} else {
		pc.SetStyleProperties(nil, g.Properties(), ctxt)
	}
	pc.ToDotsImpl(&pc.UnitContext) // we always inherit parent's unit context -- SVG sets it once-and-for-all

	if parCSSAgg != nil {
		AggCSS(&g.CSSAgg, parCSSAgg)
	} else {
		g.CSSAgg = nil
	}
	AggCSS(&g.CSSAgg, g.CSS)
	g.StyleCSS(sv, g.CSSAgg)

	pc.StrokeStyle.Opacity *= pc.FontStyle.Opacity // applies to all
	pc.FillStyle.Opacity *= pc.FontStyle.Opacity

	pc.Off = !pc.Display || (pc.StrokeStyle.Color == nil && pc.FillStyle.Color == nil)
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
	if g.Par != sv.Root.This() {
		pp := g.Par.(Node).PaintStyle()
		pc.SetStyleProperties(pp, pmap, ctxt)
	} else {
		pc.SetStyleProperties(nil, pmap, ctxt)
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
	idnm := "#" + strings.ToLower(g.Name()) // then name
	g.ApplyCSS(sv, idnm, css)
}

// IsDefs returns true if is in the Defs of parent SVG
func (g *NodeBase) IsDefs() bool {
	return g.Flags.HasFlag(IsDef)
}

// LocalBBoxToWin converts a local bounding box to SVG coordinates
func (g *NodeBase) LocalBBoxToWin(bb math32.Box2) image.Rectangle {
	mxi := g.ParTransform(true) // include self
	return bb.MulMatrix2(mxi).ToRect()
}

func (g *NodeBase) NodeBBox(sv *SVG) image.Rectangle {
	rs := &sv.RenderState
	return rs.LastRenderBBox
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
	if pc.StrokeStyle.Color == nil {
		return 0
	}
	return pc.StrokeStyle.Width.Dots
}

// ComputeBBox is called by default in render to compute bounding boxes for
// gui interaction -- can only be done in rendering because that is when all
// the proper transforms are all in place -- VpBBox is intersected with parent SVG
func (g *NodeBase) BBoxes(sv *SVG) {
	if g.This() == nil {
		return
	}
	ni := g.This().(Node)
	g.BBox = ni.NodeBBox(sv)
	g.BBox.Canon()
	g.VisBBox = sv.Geom.SizeRect().Intersect(g.BBox)
}

// PushTransform checks our bounding box and visibility, returning false if
// out of bounds.  If visible, pushes our transform.
// Must be called as first step in Render.
func (g *NodeBase) PushTransform(sv *SVG) (bool, *paint.Context) {
	g.BBox = image.Rectangle{}
	if g.Paint.Off || g == nil || g.This() == nil {
		return false, nil
	}
	ni := g.This().(Node)
	// if g.IsInvisible() { // just the Invisible flag
	// 	return false, nil
	// }
	lbb := ni.LocalBBox()
	g.BBox = g.LocalBBoxToWin(lbb)
	g.VisBBox = sv.Geom.SizeRect().Intersect(g.BBox)
	nvis := g.VisBBox == image.Rectangle{}
	// g.SetInvisibleState(nvis) // don't set

	if nvis && !g.IsDefs() {
		return false, nil
	}

	rs := &sv.RenderState
	rs.PushTransform(g.Paint.Transform)

	pc := &paint.Context{rs, &g.Paint}
	return true, pc
}

func (g *NodeBase) RenderChildren(sv *SVG) {
	for _, kid := range g.Kids {
		ni := kid.(Node)
		ni.Render(sv)
	}
}

func (g *NodeBase) Render(sv *SVG) {
	vis, rs := g.PushTransform(sv)
	if !vis {
		return
	}
	// pc := &g.Paint
	// render path elements, then compute bbox, then fill / stroke
	g.BBoxes(sv)
	g.RenderChildren(sv)
	rs.PopTransform()
}

// NodeFlags extend [tree.Flags] to hold SVG node state.
type NodeFlags tree.Flags //enums:bitflag

const (
	IsDef NodeFlags = NodeFlags(tree.FlagsN) + iota
)
