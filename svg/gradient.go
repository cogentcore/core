// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"log"
	"strings"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
)

/////////////////////////////////////////////////////////////////////////////
//  Gradient

// Gradient is used for holding a specified color gradient.
// The name is the id for lookup in url
type Gradient struct {
	NodeBase

	// the color gradient
	Grad gradient.Gradient

	// name of another gradient to get stops from
	StopsName string
}

// GradientTypeName returns the SVG-style type name of gradient: linearGradient or radialGradient
func (gr *Gradient) GradientTypeName() string {
	if _, ok := gr.Grad.(*gradient.Radial); ok {
		return "radialGradient"
	}
	return "linearGradient"
}

//////////////////////////////////////////////////////////////////////////////
//		SVG gradient management

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

// GradientApplyTransform applies the given transform to any gradients for this node,
// that are using specific coordinates (not bounding box which is automatic)
func (g *NodeBase) GradientApplyTransform(sv *SVG, xf math32.Matrix2) {
	gi := g.This().(Node)
	gnm := NodePropURL(gi, "fill")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.Grad.AsBase().Transform.SetMul(xf) // todo: do the Ctr, unscale version?
		}
	}
	gnm = NodePropURL(gi, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.Grad.AsBase().Transform.SetMul(xf)
		}
	}
}

// GradientApplyTransformPt applies the given transform with ctr point
// to any gradients for this node, that are using specific coordinates
// (not bounding box which is automatic)
func (g *NodeBase) GradientApplyTransformPt(sv *SVG, xf math32.Matrix2, pt math32.Vector2) {
	gi := g.This().(Node)
	gnm := NodePropURL(gi, "fill")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.Grad.AsBase().Transform.SetMulCenter(xf, pt) // todo: ctr off?
		}
	}
	gnm = NodePropURL(gi, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.Grad.AsBase().Transform.SetMulCenter(xf, pt)
		}
	}
}

// GradientWritePoints writes the gradient points to
// a slice of floating point numbers, appending to end of slice.
func GradientWritePts(gr gradient.Gradient, dat *[]float32) {
	// TODO: do we want this, and is this the right way to structure it?
	if gr == nil {
		return
	}
	gb := gr.AsBase()
	*dat = append(*dat, gb.Transform.XX)
	*dat = append(*dat, gb.Transform.YX)
	*dat = append(*dat, gb.Transform.XY)
	*dat = append(*dat, gb.Transform.YY)
	*dat = append(*dat, gb.Transform.X0)
	*dat = append(*dat, gb.Transform.Y0)

	*dat = append(*dat, gb.Box.Min.X)
	*dat = append(*dat, gb.Box.Min.Y)
	*dat = append(*dat, gb.Box.Max.X)
	*dat = append(*dat, gb.Box.Max.Y)
}

// GradientWritePts writes the geometry of the gradients for this node
// to a slice of floating point numbers, appending to end of slice.
func (g *NodeBase) GradientWritePts(sv *SVG, dat *[]float32) {
	gnm := NodePropURL(g, "fill")
	if gnm != "" {
		gr := sv.GradientByName(g, gnm)
		if gr != nil {
			GradientWritePts(gr.Grad, dat)
		}
	}
	gnm = NodePropURL(g, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(g, gnm)
		if gr != nil {
			GradientWritePts(gr.Grad, dat)
		}
	}
}

// GradientReadPoints reads the gradient points from
// a slice of floating point numbers, reading from the end.
func GradientReadPts(gr gradient.Gradient, dat []float32) {
	if gr == nil {
		return
	}
	gb := gr.AsBase()
	sz := len(dat)
	gb.Box.Min.X = dat[sz-4]
	gb.Box.Min.Y = dat[sz-3]
	gb.Box.Max.X = dat[sz-2]
	gb.Box.Max.Y = dat[sz-1]

	gb.Transform.XX = dat[sz-10]
	gb.Transform.YX = dat[sz-9]
	gb.Transform.XY = dat[sz-8]
	gb.Transform.YY = dat[sz-7]
	gb.Transform.X0 = dat[sz-6]
	gb.Transform.Y0 = dat[sz-5]
}

// GradientReadPts reads the geometry of the gradients for this node
// from a slice of floating point numbers, reading from the end.
func (g *NodeBase) GradientReadPts(sv *SVG, dat []float32) {
	gnm := NodePropURL(g, "fill")
	if gnm != "" {
		gr := sv.GradientByName(g, gnm)
		if gr != nil {
			GradientReadPts(gr.Grad, dat)
		}
	}
	gnm = NodePropURL(g, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(g, gnm)
		if gr != nil {
			GradientReadPts(gr.Grad, dat)
		}
	}
}

//////////////////////////////////////////////////////////////////////////////
//  Gradient management utilities for creating element-specific grads

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
// that points to given stops name.  returns the new gradient
// and the url that points to it (nil if parent svg cannot be found).
// Initializes gradient to use bounding box of object, but using userSpaceOnUse setting
func (sv *SVG) GradientNewForNode(n Node, radial bool, stops string) (*Gradient, string) {
	gr, url := sv.GradientNew(radial)
	gr.StopsName = stops
	gr.Grad.AsBase().SetBox(n.LocalBBox())
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
	gr.SetName(NameID(gnm, id))
	url := NameToURL(gnm)
	if radial {
		gr.Grad = gradient.NewRadial()
	} else {
		gr.Grad = gradient.NewLinear()
	}
	return gr, url
}

// GradientUpdateNodeProp ensures that node has a gradient property of given type
func (sv *SVG) GradientUpdateNodeProp(n Node, prop string, radial bool, stops string) (*Gradient, string) {
	ps := n.AsTreeNode().Property(prop)
	if ps == nil {
		gr, url := sv.GradientNewForNode(n, radial, stops)
		n.AsTreeNode().SetProperty(prop, url)
		return gr, url
	}
	pstr := ps.(string)
	trgst := ""
	if radial {
		trgst = "radialGradient"
	} else {
		trgst = "linearGradient"
	}
	url := "url(#" + trgst
	if strings.HasPrefix(pstr, url) {
		gr := sv.GradientByName(n, pstr)
		gr.StopsName = stops
		sv.GradientUpdateStops(gr)
		return gr, NameToURL(gr.Nm)
	}
	if strings.HasPrefix(pstr, "url(#") { // wrong kind
		sv.GradientDeleteForNode(n, pstr)
	}
	gr, url := sv.GradientNewForNode(n, radial, stops)
	n.AsTreeNode().SetProperty(prop, url)
	return gr, url
}

// GradientUpdateNodePoints updates the points for node based on current bbox
func (sv *SVG) GradientUpdateNodePoints(n Node, prop string) {
	ps := n.AsTreeNode().Property(prop)
	if ps == nil {
		return
	}
	pstr := ps.(string)
	url := "url(#"
	if !strings.HasPrefix(pstr, url) {
		return
	}
	gr := sv.GradientByName(n, pstr)
	if gr == nil {
		return
	}
	gb := gr.Grad.AsBase()
	gb.SetBox(n.LocalBBox())
	gb.SetTransform(math32.Identity2())
}

// GradientCloneNodeProp creates a new clone of the existing gradient for node
// if set for given property key ("fill" or "stroke").
// returns new gradient.
func (sv *SVG) GradientCloneNodeProp(n Node, prop string) *Gradient {
	ps := n.AsTreeNode().Property(prop)
	if ps == nil {
		return nil
	}
	pstr := ps.(string)
	radial := false
	if strings.HasPrefix(pstr, "url(#radialGradient") {
		radial = true
	} else if !strings.HasPrefix(pstr, "url(#linearGradient") {
		return nil
	}
	gr := sv.GradientByName(n, pstr)
	if gr == nil {
		return nil
	}
	ngr, url := sv.GradientNewForNode(n, radial, gr.StopsName)
	n.AsTreeNode().SetProperty(prop, url)
	gradient.CopyFrom(ngr.Grad, gr.Grad)
	// TODO(kai): should this return ngr or gr? (used to return gr but ngr seems correct)
	return ngr
}

// GradientDeleteNodeProp deletes any existing gradient for node
// if set for given property key ("fill" or "stroke").
// Returns true if deleted.
func (sv *SVG) GradientDeleteNodeProp(n Node, prop string) bool {
	ps := n.AsTreeNode().Property(prop)
	if ps == nil {
		return false
	}
	pstr := ps.(string)
	if !strings.HasPrefix(pstr, "url(#radialGradient") && !strings.HasPrefix(pstr, "url(#linearGradient") {
		return false
	}
	return sv.GradientDeleteForNode(n, pstr)
}

// GradientUpdateAllStops removes any items from Defs that are not actually referred to
// by anything in the current SVG tree.  Returns true if items were removed.
// Does not remove gradients with StopsName = "" with extant stops -- these
// should be removed manually, as they are not automatically generated.
func (sv *SVG) GradientUpdateAllStops() {
	for _, k := range sv.Defs.Kids {
		gr, ok := k.(*Gradient)
		if ok {
			sv.GradientUpdateStops(gr)
		}
	}
}
