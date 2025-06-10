// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"log"
	"strings"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

// note: this code provides convenience methods for updating gradient data
// when dynamically updating svg nodes, as in an animation, svg drawing app, etc.
// It is an awkward design that the gradient needs to be updated for each node
// for most sensible use-cases, because the ObjectBoundingBox is too limited,
// so a unique UserSpaceOnUse gradient is typically needed for each node.

// Gradient is used for holding a specified color gradient.
// The name is the id for lookup in url
type Gradient struct {
	NodeBase

	// the color gradient
	Grad gradient.Gradient `json:"-"` // note: cannot do drag-n-drop

	// name of another gradient to get stops from
	StopsName string
}

// IsRadial returns true if given gradient is a raidal type.
func (gr *Gradient) IsRadial() bool {
	_, ok := gr.Grad.(*gradient.Radial)
	return ok
}

// GradientTypeName returns the SVG-style type name of gradient:
// linearGradient or radialGradient
func (gr *Gradient) GradientTypeName() string {
	if gr.IsRadial() {
		return "radialGradient"
	}
	return "linearGradient"
}

// fromGeom sets gradient geom for UserSpaceOnUse gradient units
// from given geom.
func (gr *Gradient) fromGeom(xf math32.Matrix2) { // gg *GradientGeom) {
	gb := gr.Grad.AsBase()
	if gb.Units != gradient.UserSpaceOnUse {
		return
	}
	gb.Transform = xf
}

// toGeom copies gradient points to given geom
// for UserSpaceOnUse gradient units.
func (gr *Gradient) toGeom(xf *math32.Matrix2) {
	gb := gr.Grad.AsBase()
	if gb.Units != gradient.UserSpaceOnUse {
		return
	}
	*xf = gb.Transform
}

////////  SVG gradient management

// GradientByName returns the gradient of given name, stored on SVG node
func (sv *SVG) GradientByName(n Node, grnm string) *Gradient {
	gri := sv.NodeFindURL(n, grnm)
	if gri == nil {
		return nil
	}
	gr, ok := gri.(*Gradient)
	if !ok {
		log.Printf("SVG Found element named: %v but isn't a Gradient type, instead is: %T", grnm, gri)
		return nil
	}
	return gr
}

// GradientApplyTransform applies given transform to node's gradient geometry.
// This should ONLY be called when the node's transform is _not_ being updated,
// and instead its geometry values (pos etc) are being transformed directly by
// this transform, because the node's transform will be applied to the gradient
// when it is being rendered.
func (g *NodeBase) GradientApplyTransform(sv *SVG, xf math32.Matrix2) {
	g.GradientFill = xf.Mul(g.GradientFill)
	g.GradientStroke = xf.Mul(g.GradientStroke)
	g.GradientUpdateGeom(sv)
}

// GradientGeomDefault sets the initial default gradient geometry from node.
func (g *NodeBase) GradientGeomDefault(sv *SVG, gr gradient.Gradient) {
	gi := g.This.(Node)
	if rg, ok := gr.(*gradient.Radial); ok {
		lbb := gi.LocalBBox(sv)
		ctr := lbb.Center()
		sz := lbb.Size()
		rad := 0.5 * max(sz.X, sz.Y)
		rg.Center = ctr
		rg.Focal = ctr
		rg.Radius.Set(rad, rad)
		return
	}
	lg := gr.(*gradient.Linear)
	lbb := gi.LocalBBox(sv)
	lg.Start = lbb.Min
	lg.End = math32.Vec2(lbb.Max.X, lbb.Min.Y) // L-R
}

// GradientUpdateGeom updates the geometry of UserSpaceOnUse gradients
// in use by this node for "fill" and "stroke" by copying its current
// GradientGeom points to those gradients.
func (g *NodeBase) GradientUpdateGeom(sv *SVG) {
	gi := g.This.(Node)
	gnm := NodePropURL(gi, "fill")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.fromGeom(g.GradientFill)
			// fmt.Println("set fill:", g.GradientFill)
		}
	}
	gnm = NodePropURL(gi, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.fromGeom(g.GradientStroke)
		}
	}
}

// GradientFromGradients updates the geometry of UserSpaceOnUse gradients
// in use by this node for "fill" and "stroke" by copying its current
// GradientGeom points to those gradients.
func (g *NodeBase) GradientFromGradients(sv *SVG) {
	gi := g.This.(Node)
	gnm := NodePropURL(gi, "fill")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.toGeom(&g.GradientFill)
		}
	}
	gnm = NodePropURL(gi, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.toGeom(&g.GradientStroke)
		}
	}
}

// GradientFromGradients updates the geometry of UserSpaceOnUse gradients
// for all nodes, for "fill" and "stroke" properties, by copying the current
// GradientGeom points to those gradients. This can be done after loading
// for cases where node transforms will be updated dynamically.
func (sv *SVG) GradientFromGradients() {
	sv.Root.WalkDown(func(n tree.Node) bool {
		nb := n.(Node).AsNodeBase()
		nb.GradientFromGradients(sv)
		return tree.Continue
	})
}

////////  Gradient management utilities for creating element-specific grads

// GradientUpdateStops copies stops from StopsName gradient if it is set
func (sv *SVG) GradientUpdateStops(gr *Gradient) {
	if gr.StopsName == "" {
		return
	}
	sgr := sv.GradientByName(gr, gr.StopsName)
	if sgr != nil {
		gr.Grad.AsBase().CopyStopsFrom(sgr.Grad.AsBase())
	}
}

// GradientDeleteForNode deletes the node-specific gradient on given node
// of given name, which can be a full url(# name or just the bare name.
// Returns true if deleted.
func (sv *SVG) GradientDeleteForNode(n Node, grnm string) bool {
	gr := sv.GradientByName(n, grnm)
	if gr == nil || gr.StopsName == "" {
		return false
	}
	unm := NameFromURL(grnm)
	sv.Defs.DeleteChildByName(unm)
	return true
}

// GradientNewForNode adds a new gradient specific to given node
// that points to given stops name. Returns the new gradient
// and the url that points to it (nil if parent svg cannot be found).
// Initializes gradient to use current GradientFill or Stroke with UserSpaceOnUse.
func (sv *SVG) GradientNewForNode(n Node, prop string, radial bool, stops string) (*Gradient, string) {
	gr, url := sv.GradientNew(radial)
	gr.StopsName = stops
	gr.Grad.AsBase().Units = gradient.UserSpaceOnUse
	nb := n.AsNodeBase()
	nb.GradientGeomDefault(sv, gr.Grad)
	if prop == "fill" {
		nb.GradientFill = math32.Identity2()
	} else {
		nb.GradientStroke = math32.Identity2()
	}
	sv.GradientUpdateStops(gr)
	return gr, url
}

// GradientNew adds a new gradient, either linear or radial,
// with a new unique id
func (sv *SVG) GradientNew(radial bool) (*Gradient, string) {
	gnm := ""
	if radial {
		gnm = "radialGradient"
	} else {
		gnm = "linearGradient"
	}
	gr := NewGradient(sv.Defs)
	id := sv.NewUniqueID()
	gnm = NameID(gnm, id)
	gr.SetName(gnm)
	url := NameToURL(gnm)
	if radial {
		gr.Grad = gradient.NewRadial()
	} else {
		gr.Grad = gradient.NewLinear()
	}
	return gr, url
}

// GradientUpdateNodeProp ensures that node has a gradient property of given type.
func (sv *SVG) GradientUpdateNodeProp(n Node, prop string, radial bool, stops string) (*Gradient, string) {
	nb := n.AsNodeBase()
	ps := nb.Property(prop)
	if ps == nil {
		gr, url := sv.GradientNewForNode(n, prop, radial, stops)
		nb.SetProperty(prop, url)
		nb.SetProperty(prop+"-opacity", "1")
		return gr, url
	}
	pstr, ok := ps.(string)
	if !ok {
		pstr = *ps.(*string)
	}
	trgst := ""
	if radial {
		trgst = "radialGradient"
	} else {
		trgst = "linearGradient"
	}
	url := "url(#" + trgst
	if strings.HasPrefix(pstr, url) {
		gr := sv.GradientByName(n, pstr)
		if gr != nil {
			gr.StopsName = stops
			sv.GradientUpdateStops(gr)
			return gr, NameToURL(gr.Name)
		} else {
			fmt.Println("not found:", pstr, url)
		}
	}
	if strings.HasPrefix(pstr, "url(#") { // wrong kind
		sv.GradientDeleteForNode(n, pstr)
	}
	gr, url := sv.GradientNewForNode(n, prop, radial, stops)
	nb.SetProperty(prop, url)
	return gr, url
}

// GradientUpdateAllStops removes any items from Defs that are not actually referred to
// by anything in the current SVG tree.  Returns true if items were removed.
// Does not remove gradients with StopsName = "" with extant stops -- these
// should be removed manually, as they are not automatically generated.
func (sv *SVG) GradientUpdateAllStops() {
	for _, k := range sv.Defs.Children {
		gr, ok := k.(*Gradient)
		if ok {
			sv.GradientUpdateStops(gr)
		}
	}
}

// GradientDuplicateNode duplicates any existing gradients
// for the given node, in fill or stroke.
// Must be called when duplicating a node.
func (sv *SVG) GradientDuplicateNode(n Node) {
	nb := n.AsNodeBase()

	setGr := func(prop string) {
		v, ok := nb.Properties[prop]
		if !ok {
			return
		}
		s, ok := v.(string)
		if !ok {
			return
		}
		nm := NameFromURL(s)
		if nm == "" {
			return
		}
		gri := sv.FindDefByName(nm)
		if gri == nil {
			return
		}
		gr := gri.(*Gradient)
		_, ngu := sv.GradientNewForNode(n, prop, gr.IsRadial(), gr.StopsName)
		nb.Properties["fill"] = ngu
	}
	setGr("fill")
	setGr("stroke")
}
