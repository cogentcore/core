// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"image"
	"reflect"
	"strings"

	"goki.dev/girl/girl"
	"goki.dev/girl/gist"
	"goki.dev/ki/v2/ki"
	"goki.dev/mat32/v2"
)

// Node is the interface for all SVG nodes
type Node interface {
	ki.Ki

	// AsNodeBase returns a generic svg.NodeBase for our node -- gives generic
	// access to all the base-level data structures without requiring
	// interface methods.
	AsNodeBase() *NodeBase

	// Paint returns the SVG Paint style object for this node
	Paint() *girl.Paint

	// SetPos sets the *upper left* position of this element, in local dimensions
	SetPos(pos mat32.Vec2)

	// SetSize sets the overall size of this element, in local dimensions
	SetSize(sz mat32.Vec2)

	// LocalBBox returns the bounding box of node in local dimensions
	LocalBBox() mat32.Box2

	// ApplyXForm applies the given 2D transform to the geometry of this node
	// this just does a direct transform multiplication on coordinates.
	ApplyXForm(xf mat32.Mat2)

	// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
	// relative to given point.  Trans translation and point are in top-level coordinates,
	// so must be transformed into local coords first.
	// Point is upper left corner of selection box that anchors the translation and scaling,
	// and for rotation it is the center point around which to rotate
	ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2)

	// WriteGeom writes the geometry of the node to a slice of floating point numbers
	// the length and ordering of which is specific to each node type.
	// Slice must be passed and will be resized if not the correct length.
	WriteGeom(dat *[]float32)

	// ReadGeom reads the geometry of the node from a slice of floating point numbers
	// the length and ordering of which is specific to each node type.
	ReadGeom(dat []float32)

	// SVGName returns the SVG element name (e.g., "rect", "path" etc)
	SVGName() string

	// EnforceSVGName returns true if in general this element should
	// be named with its SVGName plus a unique id.
	// Groups and Markers are false.
	EnforceSVGName() bool
}

// svg.NodeBase is the base type for elements within the SVG scenegraph
type NodeBase struct {
	ki.Node

	// user-defined class name(s) used primarily for attaching CSS styles to different display elements -- multiple class names can be used to combine properties: use spaces to separate per css standard
	Class string `desc:"user-defined class name(s) used primarily for attaching CSS styles to different display elements -- multiple class names can be used to combine properties: use spaces to separate per css standard"`

	// cascading style sheet at this level -- these styles apply here and to everything below, until superceded -- use .class and #name Props elements to apply entire styles to given elements, and type for element type
	CSS ki.Props `xml:"css" desc:"cascading style sheet at this level -- these styles apply here and to everything below, until superceded -- use .class and #name Props elements to apply entire styles to given elements, and type for element type"`

	// [view: no-inline] aggregated css properties from all higher nodes down to me
	CSSAgg ki.Props `copy:"-" json:"-" xml:"-" view:"no-inline" desc:"aggregated css properties from all higher nodes down to me"`

	// bounding box for the node within the SVG Pixels image -- this one can be outside the visible range of the SVG image -- VpBBox is intersected and only shows visible portion.
	BBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"bounding box for the node within the SVG Pixels image -- this one can be outside the visible range of the SVG image -- VpBBox is intersected and only shows visible portion."`

	// visible bounding box for the node intersected with the SVG image geometry
	VisBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"visible bounding box for the node intersected with the SVG image geometry"`

	// paint style information for this node
	Pnt girl.Paint `json:"-" xml:"-" desc:"paint style information for this node"`
}

func (g *NodeBase) CopyFieldsFrom(frm any) {
	fr := frm.(*NodeBase)
	g.Pnt = fr.Pnt
}

func (g *NodeBase) AsSVGNode() *NodeBase {
	return g
}

func (g *NodeBase) SVGName() string {
	return "base"
}

func (g *NodeBase) EnforceSVGName() bool {
	return true
}

func (g *NodeBase) SetPos(pos mat32.Vec2) {
}

func (g *NodeBase) SetSize(sz mat32.Vec2) {
}

func (g *NodeBase) LocalBBox() mat32.Box2 {
	bb := mat32.Box2{}
	return bb
}

func (n *NodeBase) BaseIface() reflect.Type {
	return reflect.TypeOf((*NodeBase)(nil)).Elem()
}

func (g *NodeBase) Paint() *gist.Paint {
	return &g.Pnt.Paint
}

// SetColorProps sets color property from a string representation.
// It breaks color alpha out as opacity.  prop is either "stroke" or "fill"
func (g *NodeBase) SetColorProps(prop, color string) {
	if color[0] == '#' && len(color) == 9 {
		g.SetProp(prop, color[:7]) // exclude alpha
		alphai := 0
		fmt.Sscanf(color[7:], "%02x", &alphai)
		g.SetProp(prop+"-opacity", fmt.Sprintf("%g", float32(alphai)/255))
	} else {
		g.SetProp(prop, color)
	}
}

// ParXForm returns the full compounded 2D transform matrix for all
// of the parents of this node.  If self is true, then include our
// own xform too.
func (g *NodeBase) ParXForm(self bool) mat32.Mat2 {
	pars := []Node{}
	xf := mat32.Identity2D()
	nb := g
	for {
		if nb.Par == nil {
			break
		}
		if ki.TypeEmbeds(nb.Par, TypeSVG) {
			top := nb.Par.Embed(TypeSVG).(*SVG)
			xf = top.Pnt.XForm
			break
		}
		psvg := nb.Par.(Node)
		pars = append(pars, psvg)
		nb = psvg.AsSVGNode()
	}
	np := len(pars)
	for i := np - 1; i >= 0; i-- {
		n := pars[i]
		xf = n.AsSVGNode().Pnt.XForm.Mul(xf)
	}
	if self {
		xf = g.Pnt.XForm.Mul(xf)
	}
	return xf
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// this just does a direct transform multiplication on coordinates.
func (g *NodeBase) ApplyXForm(xf mat32.Mat2) {
}

// DeltaXForm computes the net transform matrix for given delta xform parameters
// and the transformed version of the reference point.  If self is true, then
// include the current node self transform, otherwise don't.  Groups do not
// but regular rendering nodes do.
func (g *NodeBase) DeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2, self bool) (mat32.Mat2, mat32.Vec2) {
	mxi := g.ParXForm(self)
	mxi = mxi.Inverse()
	lpt := mxi.MulVec2AsPt(pt)
	ldel := mxi.MulVec2AsVec(trans)
	xf := mat32.Scale2D(scale.X, scale.Y).Rotate(rot)
	xf.X0 = ldel.X
	xf.Y0 = ldel.Y
	return xf, lpt
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *NodeBase) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
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

// WriteXForm writes the node transform to slice at starting index.
// slice must already be allocated sufficiently.
func (g *NodeBase) WriteXForm(dat []float32, idx int) {
	dat[idx+0] = g.Pnt.XForm.XX
	dat[idx+1] = g.Pnt.XForm.YX
	dat[idx+2] = g.Pnt.XForm.XY
	dat[idx+3] = g.Pnt.XForm.YY
	dat[idx+4] = g.Pnt.XForm.X0
	dat[idx+5] = g.Pnt.XForm.Y0
}

// ReadXForm reads the node transform from slice at starting index.
func (g *NodeBase) ReadXForm(dat []float32, idx int) {
	g.Pnt.XForm.XX = dat[idx+0]
	g.Pnt.XForm.YX = dat[idx+1]
	g.Pnt.XForm.XY = dat[idx+2]
	g.Pnt.XForm.YY = dat[idx+3]
	g.Pnt.XForm.X0 = dat[idx+4]
	g.Pnt.XForm.Y0 = dat[idx+5]
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *NodeBase) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 6)
	g.WriteXForm(*dat, 0)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *NodeBase) ReadGeom(dat []float32) {
	g.ReadXForm(dat, 0)
}

// FirstNonGroupNode returns the first item that is not a group
// recursing into groups until a non-group item is found.
func FirstNonGroupNode(kn ki.Ki) ki.Ki {
	var ngn ki.Ki
	kn.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d any) bool {
		if k == nil || k.This() == nil || k.IsDeleted() || k.IsDestroyed() {
			return ki.Break
		}
		if _, isgp := k.(*Group); isgp {
			return ki.Continue
		}
		ngn = k
		return ki.Break
	})
	return ngn
}

//////////////////////////////////////////////////////////////////
// Standard Node infrastructure

// todo: remove if not needed:

// Init does any needed initialization
func (g *NodeBase) Init() {
}

// Style styles the Paint values directly from node properties
func (g *NodeBase) Style(sv *SVG) {
	pc := &g.Pnt
	pc.Defaults()
	ctxt := sv.(gist.Context)
	pc.StyleSet = false // this is always first call, restart

	var parCSSAgg ki.Props
	if g.Par != sv.This() {
		pn := g.Par.(Node)
		parCSSAgg = pn.AsNodeBase().CSSAgg
		pp := pn.Paint()
		pc.CopyStyleFrom(pp)
		pc.SetStyleProps(pp, *g.Properties(), ctxt)
	} else {
		pc.SetStyleProps(nil, *g.Properties(), ctxt)
	}
	pc.ToDotsImpl(&pc.UnContext) // we always inherit parent's unit context -- SVG sets it once-and-for-all

	if pagg != nil {
		AggCSS(&g.CSSAgg, pagg)
	} else {
		g.CSSAgg = nil
	}
	AggCSS(&g.CSSAgg, g.CSS)
	g.StyleCSS(gii, g.CSSAgg)

	pc.Off = !pc.Display || pc.HasNoStrokeOrFill()
}

// AggCSS aggregates css properties
func AggCSS(agg *ki.Props, css ki.Props) {
	if *agg == nil {
		*agg = make(ki.Props, len(css))
	}
	for key, val := range css {
		(*agg)[key] = val
	}
}

// ApplyCSS applies css styles to given node,
// using key to select sub-props from overall properties list
func (g *NodeBase) ApplyCSS(sv *SVG, key string, css ki.Props) bool {
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(ki.Props) // must be a props map
	if !ok {
		return false
	}
	pc := &g.Pnt
	ctxt := sv.(gist.Context)
	if g.Par != sv.This() {
		pp := g.Par.(Node).Paint()
		pc.SetStyleProps(pp, pmap, ctxt)
	} else {
		pc.SetStyleProps(nil, pmap, ctxt)
	}
	return true
}

// StyleCSS applies css style properties to given SVG node
// parsing out type, .class, and #name selectors
func (g *NodeBase) StyleCSS(sv *SVG, css ki.Props) {
	tyn := strings.ToLower(g.Type.Name) // type is most general, first
	g.ApplyCSS(sv, tyn, css)
	cln := "." + strings.ToLower(g.Class) // then class
	g.ApplyCSS(sv, cln, css)
	idnm := "#" + strings.ToLower(g.Name()) // then name
	ApplyCSS(sv, idnm, css)
}

// ParentSVG returns the parent SVG viewport
func ParentSVG(g Node) *SVG {
	return g.ParentByType(SVGType, ki.Embeds)
}

// todo: set a flag for this:

// IsDefs returns true if is in the Defs of parent SVG viewport
func IsDefs(g *NodeBase) bool {
	sv := ParentSVG(g)
	if sv == nil {
		return false
	}
	rv := false
	g.FuncUpParent(0, nil, func(k ki.Ki, level int, d any) bool {
		if k == nil || k.This() == nil || k.IsDeleted() || k.IsDestroyed() {
			return ki.Break
		}
		if k.Parent() == sv.This() {
			return ki.Break
		}
		if k.Parent() == sv.Defs.This() {
			rv = true
			return ki.Break
		}
		return ki.Continue

	})
	return rv
}

// LocalBBoxToWin converts a local bounding box to SVG coordinates
func (g *NodeBase) LocalBBoxToWin(bb mat32.Box2) image.Rectangle {
	mxi := g.ParXForm(true) // include self
	return bb.MulMat2(mxi).ToRect()
}

func (g *NodeBase) NodeBBox() image.Rectangle {
	rs := &g.Viewport.Render
	return rs.LastRenderBBox
}

// LocalLineWidth returns the line width in local coordinates
func (g *NodeBase) LocalLineWidth() float32 {
	pc := &g.Pnt
	if !pc.StrokeStyle.On {
		return 0
	}
	return pc.StrokeStyle.Width.Dots
}

// ComputeBBox is called by default in render to compute bounding boxes for
// gui interaction -- can only be done in rendering because that is when all
// the proper xforms are all in place -- VpBBox is intersected with parent SVG
func (g *NodeBase) ComputeBBox(sv *SVG) {
	if g.This() == nil {
		return
	}
	ni := g.This().(Node)
	g.BBox = ni.NodeBBox()
	g.BBox.Canon()
	g.VisBBox = sv.Geom.SizeRect().Intersect(g.BBox)
}

// PushXForm checks our bounding box and visibility, returning false if
// out of bounds.  If visible, pushes our xform.
// Must be called as first step in Render.
func (g *NodeBase) PushXForm(sv *SVG) (bool, *girl.State) {
	g.BBox = image.Rectangle{}
	if g.Pnt.Off || g == nil || g.This() == nil {
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

	if nvis && !IsDefs(g.AsNode2D()) {
		return false, nil
	}

	rs := &mvp.Render
	pc := &g.Pnt
	rs.PushXFormLock(pc.XForm)

	return true, rs
}

func (g *NodeBase) RenderChildren() {
	for _, kid := range sv.Kids {
		ni := kid.(Node)
		ni.Render()
	}
}

func (g *NodeBase) Render() {
	vis, rs := g.PushXForm()
	if !vis {
		return
	}
	// pc := &g.Pnt
	// render path elements, then compute bbox, then fill / stroke
	g.ComputeBBox()
	g.RenderChildren()
	rs.PopXFormLock()
}
