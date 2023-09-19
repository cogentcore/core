// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"log"
	"strings"

	"github.com/srwiley/rasterx"
	"goki.dev/girl/gist"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

/////////////////////////////////////////////////////////////////////////////
//  Gradient

// Gradient is used for holding a specified color gradient (ColorSpec)
// name is id for lookup in url
type Gradient struct {
	NodeBase

	// the color gradient
	Grad gist.ColorSpec `desc:"the color gradient"`

	// name of another gradient to get stops from
	StopsName string `desc:"name of another gradient to get stops from"`
}

func (gr *Gradient) CopyFieldsFrom(frm any) {
	fr := frm.(*Gradient)
	gr.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	gr.Grad = fr.Grad
	gr.StopsName = fr.StopsName
}

// GradientTypeName returns the SVG-style type name of gradient: linearGradient or radialGradient
func (gr *Gradient) GradientTypeName() string {
	if gr.Grad.Gradient != nil && gr.Grad.Gradient.IsRadial {
		return "radialGradient"
	}
	return "linearGradient"
}

//////////////////////////////////////////////////////////////////////////////
//		SVG gradient mgmt

// GradientByName returns the gradient of given name, stored on SVG node
func (sv *SVG) GradientByName(gi Node, grnm string) *Gradient {
	gri := sv.NodeFindURL(gi, grnm)
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

// GradientApplyXForm applies the given transform to any gradients for this node,
// that are using specific coordinates (not bounding box which is automatic)
func (g *NodeBase) GradientApplyXForm(sv *SVG, xf mat32.Mat2) {
	gi := g.This().(Node)
	gnm := NodePropURL(gi, "fill")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.Grad.ApplyXForm(xf)
		}
	}
	gnm = NodePropURL(gi, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.Grad.ApplyXForm(xf)
		}
	}
}

// GradientApplyXFormPt applies the given transform with ctr point
// to any gradients for this node, that are using specific coordinates
// (not bounding box which is automatic)
func (g *NodeBase) GradientApplyXFormPt(sv *SVG, xf mat32.Mat2, pt mat32.Vec2) {
	gi := g.This().(Node)
	gnm := NodePropURL(gi, "fill")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.Grad.ApplyXFormPt(xf, pt)
		}
	}
	gnm = NodePropURL(gi, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(gi, gnm)
		if gr != nil {
			gr.Grad.ApplyXFormPt(xf, pt)
		}
	}
}

// GradientWritePoints writes the UserSpaceOnUse gradient points to
// a slice of floating point numbers, appending to end of slice.
func GradientWritePts(gr *gist.ColorSpec, dat *[]float32) {
	if gr.Gradient == nil {
		return
	}
	if gr.Gradient.Units == rasterx.ObjectBoundingBox {
		return
	}
	*dat = append(*dat, float32(gr.Gradient.Matrix.A))
	*dat = append(*dat, float32(gr.Gradient.Matrix.B))
	*dat = append(*dat, float32(gr.Gradient.Matrix.C))
	*dat = append(*dat, float32(gr.Gradient.Matrix.D))
	*dat = append(*dat, float32(gr.Gradient.Matrix.E))
	*dat = append(*dat, float32(gr.Gradient.Matrix.F))
	if !gr.Gradient.IsRadial {
		for i := 0; i < 4; i++ {
			*dat = append(*dat, float32(gr.Gradient.Points[i]))
		}
	}
}

// GradientWritePts writes the geometry of the gradients for this node
// to a slice of floating point numbers, appending to end of slice.
func (g *NodeBase) GradientWritePts(sv *SVG, dat *[]float32) {
	gnm := NodePropURL(g, "fill")
	if gnm != "" {
		gr := sv.GradientByName(g, gnm)
		if gr != nil {
			GradientWritePts(&gr.Grad, dat)
		}
	}
	gnm = NodePropURL(g, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(g, gnm)
		if gr != nil {
			GradientWritePts(&gr.Grad, dat)
		}
	}
}

// GradientReadPoints reads the UserSpaceOnUse gradient points from
// a slice of floating point numbers, reading from the end.
func GradientReadPts(gr *gist.ColorSpec, dat []float32) {
	if gr.Gradient == nil {
		return
	}
	if gr.Gradient.Units == rasterx.ObjectBoundingBox {
		return
	}
	sz := len(dat)
	n := 6
	if !gr.Gradient.IsRadial {
		n = 10
		for i := 0; i < 4; i++ {
			gr.Gradient.Points[i] = float64(dat[(sz-4)+i])
		}
	}
	gr.Gradient.Matrix.A = float64(dat[(sz-n)+0])
	gr.Gradient.Matrix.B = float64(dat[(sz-n)+1])
	gr.Gradient.Matrix.C = float64(dat[(sz-n)+2])
	gr.Gradient.Matrix.D = float64(dat[(sz-n)+3])
	gr.Gradient.Matrix.E = float64(dat[(sz-n)+4])
	gr.Gradient.Matrix.F = float64(dat[(sz-n)+5])
}

// GradientReadPts reads the geometry of the gradients for this node
// from a slice of floating point numbers, reading from the end.
func (g *NodeBase) GradientReadPts(sv *SVG, dat []float32) {
	gnm := NodePropURL(g, "fill")
	if gnm != "" {
		gr := sv.GradientByName(g, gnm)
		if gr != nil {
			GradientReadPts(&gr.Grad, dat)
		}
	}
	gnm = NodePropURL(g, "stroke")
	if gnm != "" {
		gr := sv.GradientByName(g, gnm)
		if gr != nil {
			GradientReadPts(&gr.Grad, dat)
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
		gr.Grad.CopyStopsFrom(&sgr.Grad)
	}
}

// GradientDeleteForNode deletes the node-specific gradient on given node
// of given name, which can be a full url(# name or just the bare name.
// Returns true if deleted.
func (sv *SVG) GradientDeleteForNode(gi Node, grnm string) bool {
	gr := sv.GradientByName(gi, grnm)
	if gr == nil || gr.StopsName == "" {
		return false
	}
	unm := NameFromURL(grnm)
	sv.Defs.DeleteChildByName(unm, ki.DestroyKids)
	return true
}

// GradientNewForNode adds a new gradient specific to given node
// that points to given stops name.  returns the new gradient
// and the url that points to it (nil if parent svg cannot be found).
// Initializes gradient to use bounding box of object, but using userSpaceOnUse setting
func (sv *SVG) GradientNewForNode(gi Node, radial bool, stops string) (*Gradient, string) {
	gr, url := sv.GradientNew(radial)
	gr.StopsName = stops
	bbox := gi.(Node).LocalBBox()
	gr.Grad.SetGradientPoints(bbox)
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
	id := sv.NewUniqueId()
	gnm = NameId(gnm, id)
	gr := sv.Defs.NewChild(GradientType, gnm).(*Gradient)
	url := NameToURL(gnm)
	if radial {
		gr.Grad.NewRadialGradient()
	} else {
		gr.Grad.NewLinearGradient()
	}
	return gr, url
}

// GradientUpdateNodeProp ensures that node has a gradient property of given type
func (sv *SVG) GradientUpdateNodeProp(gi Node, prop string, radial bool, stops string) (*Gradient, string) {
	ps := gi.Prop(prop)
	if ps == nil {
		gr, url := sv.GradientNewForNode(gi, radial, stops)
		gi.SetProp(prop, url)
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
		gr := sv.GradientByName(gi, pstr)
		gr.StopsName = stops
		sv.GradientUpdateStops(gr)
		return gr, NameToURL(gr.Nm)
	}
	if strings.HasPrefix(pstr, "url(#") { // wrong kind
		sv.GradientDeleteForNode(gi, pstr)
	}
	gr, url := sv.GradientNewForNode(gi, radial, stops)
	gi.SetProp(prop, url)
	return gr, url
}

// GradientUpdateNodePoints updates the points for node based on current bbox
func (sv *SVG) GradientUpdateNodePoints(gi Node, prop string) {
	ps := gi.Prop(prop)
	if ps == nil {
		return
	}
	pstr := ps.(string)
	url := "url(#"
	if !strings.HasPrefix(pstr, url) {
		return
	}
	gr := sv.GradientByName(gi, pstr)
	if gr == nil {
		return
	}
	bbox := gi.(Node).LocalBBox()
	gr.Grad.SetGradientPoints(bbox)
	gr.Grad.Gradient.Matrix = rasterx.Identity
}

// GradientCloneNodeProp creates a new clone of the existing gradient for node
// if set for given property key ("fill" or "stroke").
// returns new gradient.
func (sv *SVG) GradientCloneNodeProp(gi Node, prop string) *Gradient {
	ps := gi.Prop(prop)
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
	gr := sv.GradientByName(gi, pstr)
	if gr == nil {
		return nil
	}
	ngr, url := sv.GradientNewForNode(gi, radial, gr.StopsName)
	gi.SetProp(prop, url)
	ngr.Grad.CopyFrom(&gr.Grad)
	return gr
}

// GradientDeleteNodeProp deletes any existing gradient for node
// if set for given property key ("fill" or "stroke").
// Returns true if deleted.
func (sv *SVG) GradientDeleteNodeProp(gi Node, prop string) bool {
	ps := gi.Prop(prop)
	if ps == nil {
		return false
	}
	pstr := ps.(string)
	if !strings.HasPrefix(pstr, "url(#radialGradient") && !strings.HasPrefix(pstr, "url(#linearGradient") {
		return false
	}
	return sv.GradientDeleteForNode(gi, pstr)
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
