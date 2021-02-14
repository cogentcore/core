// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"log"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
	"github.com/srwiley/rasterx"
)

// FindDefByName finds Defs item by name, using cached indexes for speed
func (sv *SVG) FindDefByName(defnm string) gi.Node2D {
	if sv.DefIdxs == nil {
		sv.DefIdxs = make(map[string]int)
	}
	idx, has := sv.DefIdxs[defnm]
	if !has {
		idx = len(sv.Defs.Kids) / 2
	}
	idx, has = sv.Defs.Kids.IndexByName(defnm, idx)
	if has {
		sv.DefIdxs[defnm] = idx
		return sv.Defs.Kids[idx].(gi.Node2D)
	}
	delete(sv.DefIdxs, defnm)
	return nil
}

func (sv *SVG) FindNamedElement(name string) gi.Node2D {
	name = strings.TrimPrefix(name, "#")
	def := sv.FindDefByName(name)
	if def != nil {
		return def.(gi.Node2D)
	}

	if sv.Par == nil {
		log.Printf("gi.SVG FindNamedElement: could not find name: %v\n", name)
		return nil
	}
	pgi, _ := gi.KiToNode2D(sv.Par)
	if pgi != nil {
		return pgi.FindNamedElement(name)
	}
	log.Printf("gi.SVG FindNamedElement: could not find name: %v\n", name)
	return nil
}

// URLName returns just the name referred to in a url(#name)
// if it is not a url(#) format then returns empty string.
func URLName(url string) string {
	if len(url) < 7 {
		return ""
	}
	if url[:5] != "url(#" {
		return ""
	}
	ref := url[5:]
	sz := len(ref)
	if ref[sz-1] == ')' {
		ref = ref[:sz-1]
	}
	return ref
}

// NodeFindURL finds a url element in the parent SVG of given node.
// Returns nil if not found.
// Works with full 'url(#Name)' string or plain name or "none"
func NodeFindURL(gii gi.Node2D, url string) gi.Node2D {
	if url == "none" {
		return nil
	}
	g := gii.AsNode2D()
	ref := URLName(url)
	if ref == "" {
		ref = url
	}
	if ref == "" {
		return nil
	}
	psvg := ParentSVG(g)
	var rv gi.Node2D
	if psvg != nil {
		rv = psvg.FindNamedElement(ref)
	} else {
		rv = g.FindNamedElement(ref)
	}
	if rv == nil {
		log.Printf("svg.NodeFindURL could not find element named: %v in parents of svg el: %v\n", url, gii.Path())
	}
	return rv
}

// NodePropURL returns a url(#name) url from given prop name on node,
// or empty string if none.  Returned value is just the 'name' part
// of the url, not the full string.
func NodePropURL(kn ki.Ki, prop string) string {
	fp, err := kn.PropTry(prop)
	if err != nil {
		return ""
	}
	fs, iss := fp.(string)
	if !iss {
		return ""
	}
	return URLName(fs)
}

// MarkerByName finds marker property of given name, or generic "marker"
// type, and if set, attempts to find that marker and return it
func MarkerByName(gii gi.Node2D, marker string) *Marker {
	url := NodePropURL(gii, marker)
	if url == "" {
		url = NodePropURL(gii, "marker")
	}
	if url == "" {
		return nil
	}
	mrkn := NodeFindURL(gii, url)
	if mrkn == nil {
		return nil
	}
	mrk, ok := mrkn.(*Marker)
	if !ok {
		log.Printf("gi.svg Found element named: %v but isn't a Marker type, instead is: %T", url, mrkn)
		return nil
	}
	return mrk
}

// GradientByName returns the gradient of given name, stored on SVG node
func GradientByName(gii gi.Node2D, grnm string) *gi.Gradient {
	gri := NodeFindURL(gii, grnm)
	if gri == nil {
		return nil
	}
	gr, ok := gri.(*gi.Gradient)
	if !ok {
		log.Printf("gi.svg Found element named: %v but isn't a Gradient type, instead is: %T", grnm, gri)
		return nil
	}
	return gr
}

// GradientApplyXForm applies the given transform to any gradients for this node,
// that are using specific coordinates (not bounding box which is automatic)
func (g *NodeBase) GradientApplyXForm(xf mat32.Mat2) {
	gii := g.This().(gi.Node2D)
	gnm := NodePropURL(gii, "fill")
	if gnm != "" {
		gr := GradientByName(gii, gnm)
		if gr != nil {
			gr.Grad.ApplyXForm(xf)
		}
	}
	gnm = NodePropURL(gii, "stroke")
	if gnm != "" {
		gr := GradientByName(gii, gnm)
		if gr != nil {
			gr.Grad.ApplyXForm(xf)
		}
	}
}

// GradientApplyXFormPt applies the given transform with ctr point
// to any gradients for this node, that are using specific coordinates
// (not bounding box which is automatic)
func (g *NodeBase) GradientApplyXFormPt(xf mat32.Mat2, pt mat32.Vec2) {
	gii := g.This().(gi.Node2D)
	gnm := NodePropURL(gii, "fill")
	if gnm != "" {
		gr := GradientByName(gii, gnm)
		if gr != nil {
			gr.Grad.ApplyXFormPt(xf, pt)
		}
	}
	gnm = NodePropURL(gii, "stroke")
	if gnm != "" {
		gr := GradientByName(gii, gnm)
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
	if gr.Gradient.IsRadial {
		*dat = append(*dat, float32(gr.Gradient.Matrix.A))
		*dat = append(*dat, float32(gr.Gradient.Matrix.B))
		*dat = append(*dat, float32(gr.Gradient.Matrix.C))
		*dat = append(*dat, float32(gr.Gradient.Matrix.D))
		*dat = append(*dat, float32(gr.Gradient.Matrix.E))
		*dat = append(*dat, float32(gr.Gradient.Matrix.F))
	} else {
		for i := 0; i < 4; i++ {
			*dat = append(*dat, float32(gr.Gradient.Points[i]))
		}
	}
}

// GradientWritePts writes the geometry of the gradients for this node
// to a slice of floating point numbers, appending to end of slice.
func (g *NodeBase) GradientWritePts(dat *[]float32) {
	gnm := NodePropURL(g, "fill")
	if gnm != "" {
		gr := GradientByName(g, gnm)
		if gr != nil {
			GradientWritePts(&gr.Grad, dat)
		}
	}
	gnm = NodePropURL(g, "stroke")
	if gnm != "" {
		gr := GradientByName(g, gnm)
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
	if gr.Gradient.IsRadial { // radial uses transform matrix
		n := 6
		gr.Gradient.Matrix.A = float64(dat[(sz-n)+0])
		gr.Gradient.Matrix.B = float64(dat[(sz-n)+1])
		gr.Gradient.Matrix.C = float64(dat[(sz-n)+2])
		gr.Gradient.Matrix.D = float64(dat[(sz-n)+3])
		gr.Gradient.Matrix.E = float64(dat[(sz-n)+4])
		gr.Gradient.Matrix.F = float64(dat[(sz-n)+5])
	} else {
		n := 4
		for i := 0; i < n; i++ {
			gr.Gradient.Points[i] = float64(dat[(sz-n)+i])
		}
	}
}

// GradientReadPts reads the geometry of the gradients for this node
// from a slice of floating point numbers, reading from the end.
func (g *NodeBase) GradientReadPts(dat []float32) {
	gnm := NodePropURL(g, "fill")
	if gnm != "" {
		gr := GradientByName(g, gnm)
		if gr != nil {
			GradientReadPts(&gr.Grad, dat)
		}
	}
	gnm = NodePropURL(g, "stroke")
	if gnm != "" {
		gr := GradientByName(g, gnm)
		if gr != nil {
			GradientReadPts(&gr.Grad, dat)
		}
	}
}

//////////////////////////////////////////////////////////////////////////////
//  Gradient management utilities for creating element-specific grads

// UpdateGradientStops copies stops from StopsName gradient if it is set
func UpdateGradientStops(gr *gi.Gradient) {
	if gr.StopsName == "" {
		return
	}
	sgr := GradientByName(gr, gr.StopsName)
	if sgr != nil {
		gr.Grad.CopyStopsFrom(&sgr.Grad)
	}
}

// DeleteNodeGradient deletes the node-specific gradient on given node.
// Returns true if deleted.
func DeleteNodeGradient(gii gi.Node2D, grnm string) bool {
	gr := GradientByName(gii, grnm)
	if gr == nil || gr.StopsName == "" {
		return false
	}
	psvg := ParentSVG(gii.AsNode2D())
	if psvg == nil {
		return false
	}
	unm := URLName(grnm)
	psvg.Defs.DeleteChildByName(unm, true)
	return true
}

// AddNewNodeGradient adds a new gradient specific to given node
// that points to given stops name.  returns the new gradient
// and the url that points to it (nil if parent svg cannot be found).
// Initializes gradient to use bounding box of object, but using userSpaceOnUse setting
func AddNewNodeGradient(gii gi.Node2D, radial bool, stops string) (*gi.Gradient, string) {
	psvg := ParentSVG(gii.AsNode2D())
	if psvg == nil {
		return nil, ""
	}
	gnm := ""
	if radial {
		gnm = "radialGradient"
	} else {
		gnm = "linearGradient"
	}
	id := psvg.NewUniqueId()
	gnm = NameId(gnm, id)
	gr := psvg.Defs.AddNewChild(gi.KiT_Gradient, gnm).(*gi.Gradient)
	gr.StopsName = stops
	bbox := gii.(NodeSVG).SVGLocalBBox()
	if radial {
		gr.Grad.NewRadialGradient(bbox)
	} else {
		gr.Grad.NewLinearGradient(bbox)
	}
	gr.Grad.Gradient.Units = rasterx.UserSpaceOnUse
	url := "url(#" + gnm + ")"
	return gr, url
}

// UpdateNodeGradientProp ensures that node has a gradient property of given type
func UpdateNodeGradientProp(gii gi.Node2D, prop string, radial bool, stops string) (*gi.Gradient, string) {
	ps := gii.Prop(prop)
	if ps == nil {
		gr, url := AddNewNodeGradient(gii, radial, stops)
		gii.SetProp(prop, url)
		return gr, url
	}
	pstr := ps.(string)
	trgst := ""
	if radial {
		trgst = "url(#radialGradient"
	} else {
		trgst = "url(#linearGradient"
	}
	url := "url(#" + trgst
	if strings.HasPrefix(pstr, url) {
		gr := GradientByName(gii, pstr)
		return gr, "url(#" + gr.Nm + ")"
	}
	if strings.HasPrefix(pstr, "url(#") { // wrong kind
		DeleteNodeGradient(gii, pstr)
	}
	gr, url := AddNewNodeGradient(gii, radial, stops)
	gii.SetProp(prop, url)
	return gr, url
}
