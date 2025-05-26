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
// The strategy here is to maintain the Fill and Stroke gradient control points
// on each node, and update those dynamically as the node is transformed,
// and then copy those back out to the gradient. If a new gradient is created,
// the existing gradient control points will be available to retain the gradient
// settings across such changes as well.

// GradientGeom maintains the gradient control points for a node.
// These are maintained and updated on the node, loaded from any saved
// UserSpaceOnUse gradients, but otherwise computed based on the shape.
type GradientGeom struct {
	Transform math32.Matrix2

	// Linear are the start and end points for linear gradient.
	Linear [2]math32.Vector2

	// Radial are the center, focal and radius points for radial gradient.
	Radial [3]math32.Vector2
}

// LinearIsNil returns true if the current linear geom has not been set.
func (gg *GradientGeom) LinearIsNil() bool {
	return gg.Linear[0] == (math32.Vector2{}) && gg.Linear[1] == (math32.Vector2{})
}

// RadialIsNil returns true if the current radial geom has not been set.
func (gg *GradientGeom) RadialIsNil() bool {
	return gg.Radial[0] == (math32.Vector2{}) && gg.Radial[1] == (math32.Vector2{}) && gg.Radial[2] == (math32.Vector2{})
}

// ApplyTransform applies the given transform to update the current gradient
// geom if they have been set.
func (gg *GradientGeom) ApplyTransform(xf math32.Matrix2) {
	xfn := gg.Transform.Inverse().Mul(xf).Mul(gg.Transform)
	if !gg.LinearIsNil() {
		gg.Linear[0] = xfn.MulVector2AsPoint(gg.Linear[0])
		gg.Linear[1] = xfn.MulVector2AsPoint(gg.Linear[1])
	}
	if !gg.RadialIsNil() {
		gg.Radial[0] = xfn.MulVector2AsPoint(gg.Radial[0])
		gg.Radial[1] = xfn.MulVector2AsPoint(gg.Radial[1])
		// note: this is not correct:
		gg.Radial[2] = math32.Scale2D(scx, scy).MulVector2AsVector(gg.Radial[2])
		// scx, scy := xfn.ExtractScale()
		// gg.Radial[2] = math32.Scale2D(scx, scy).MulVector2AsVector(gg.Radial[2]) // vector for radius
		// nr := xfn.MulVector2AsVector(gg.Radial[2])
		// tx, ty, phi, sx, sy, theta := xfn.Decompose()
		// rc := math32.Scale2D(sx, sy).Rotate(-theta).Translate(tx, ty)
		// nr := rc.MulVector2AsVector(gg.Radial[2])
		// fmt.Println(tx, ty, phi, sx, sy, theta, gg.Radial[2], nr)
		// gg.Radial[2] = nr
	}
}

// Gradient is used for holding a specified color gradient.
// The name is the id for lookup in url
type Gradient struct {
	NodeBase

	// the color gradient
	Grad gradient.Gradient

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
func (gr *Gradient) fromGeom(gg *GradientGeom) {
	gb := gr.Grad.AsBase()
	if gb.Units != gradient.UserSpaceOnUse {
		return
	}
	switch x := gr.Grad.(type) {
	case *gradient.Radial:
		x.Center = gg.Radial[0]
		x.Focal = gg.Radial[1]
		x.Radius = gg.Radial[2]
	case *gradient.Linear:
		x.Start = gg.Linear[0]
		x.End = gg.Linear[1]
	}
}

// toGeom copies gradient points to given geom
// for UserSpaceOnUse gradient units.
func (gr *Gradient) toGeom(gg *GradientGeom) {
	gb := gr.Grad.AsBase()
	if gb.Units != gradient.UserSpaceOnUse {
		return
	}
	gg.Transform = gb.Transform
	switch x := gr.Grad.(type) {
	case *gradient.Radial:
		gg.Radial[0] = x.Center
		gg.Radial[1] = x.Focal
		gg.Radial[2] = x.Radius
		return
	case *gradient.Linear:
		gg.Linear[0] = x.Start
		gg.Linear[1] = x.End
	}
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
func (g *NodeBase) GradientApplyTransform(sv *SVG, xf math32.Matrix2) {
	g.GradientFill.ApplyTransform(xf)
	g.GradientStroke.ApplyTransform(xf)
	g.GradientUpdateGeom(sv)
}

// GradientGeomDefault ensures that if current gradient geom is empty,
// there is a default GradientGeom for given property ("fill" or "stroke")
// and gradient type, based on the current local bounding box of the object.
func (g *NodeBase) GradientGeomDefault(sv *SVG, prop string, radial bool) {
	gi := g.This.(Node)
	gg := &g.GradientFill
	ogg := &g.GradientStroke
	if prop == "stroke" {
		gg = &g.GradientStroke
		ogg = &g.GradientFill
	}
	if radial {
		if !gg.RadialIsNil() {
			return
		}
		if !ogg.RadialIsNil() {
			gg.Transform = ogg.Transform
			gg.Radial = ogg.Radial
			return
		}
		gg.Transform = math32.Identity2()
		lbb := gi.LocalBBox(sv)
		ctr := lbb.Center()
		sz := lbb.Size()
		rad := 0.5 * max(sz.X, sz.Y)
		gg.Radial[0] = ctr
		gg.Radial[1] = ctr
		gg.Radial[2].Set(rad, rad)
		gg.ApplyTransform(g.Paint.Transform)
		return
	}
	if !gg.LinearIsNil() {
		return
	}
	if !ogg.LinearIsNil() {
		gg.Transform = ogg.Transform
		gg.Linear = ogg.Linear
		return
	}
	gg.Transform = math32.Identity2()
	lbb := gi.LocalBBox(sv)
	gg.Linear[0] = lbb.Min
	gg.Linear[1] = math32.Vec2(lbb.Max.X, lbb.Min.Y) // L-R
	gg.ApplyTransform(g.Paint.Transform)
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
			gr.fromGeom(&g.GradientFill)
		}
	}
	gnm = NodePropURL(gi, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.fromGeom(&g.GradientStroke)
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
	nb.GradientGeomDefault(sv, prop, radial) // ensure has one
	if prop == "fill" {
		gr.fromGeom(&nb.GradientFill)
	} else {
		gr.fromGeom(&nb.GradientStroke)
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
