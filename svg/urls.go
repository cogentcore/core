// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"log"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
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

// NameFromURL returns just the name referred to in a url(#name)
// if it is not a url(#) format then returns empty string.
func NameFromURL(url string) string {
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

// NameToURL returns url as: url(#name)
func NameToURL(nm string) string {
	return "url(#" + nm + ")"
}

// NodeFindURL finds a url element in the parent SVG of given node.
// Returns nil if not found.
// Works with full 'url(#Name)' string or plain name or "none"
func NodeFindURL(gii gi.Node2D, url string) gi.Node2D {
	if url == "none" {
		return nil
	}
	g := gii.AsNode2D()
	ref := NameFromURL(url)
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
	return NameFromURL(fs)
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

//////////////////////////////////////////////////////////////////////////////
//  Gradient management utilities for updating geometry

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

// DeleteNodeGradient deletes the node-specific gradient on given node
// of given name, which can be a full url(# name or just the bare name.
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
	unm := NameFromURL(grnm)
	psvg.Defs.DeleteChildByName(unm, ki.DestroyKids)
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
	gr, url := psvg.AddNewGradient(radial)
	gr.StopsName = stops
	bbox := gii.(NodeSVG).SVGLocalBBox()
	gr.Grad.SetGradientPoints(bbox)
	UpdateGradientStops(gr)
	return gr, url
}

// AddNewGradient adds a new gradient, either linear or radial,
// with a new unique id
func (sv *SVG) AddNewGradient(radial bool) (*gi.Gradient, string) {
	gnm := ""
	if radial {
		gnm = "radialGradient"
	} else {
		gnm = "linearGradient"
	}
	updt := sv.UpdateStart()
	id := sv.NewUniqueId()
	gnm = NameId(gnm, id)
	sv.SetChildAdded()
	gr := sv.Defs.AddNewChild(gi.KiT_Gradient, gnm).(*gi.Gradient)
	url := NameToURL(gnm)
	if radial {
		gr.Grad.NewRadialGradient()
	} else {
		gr.Grad.NewLinearGradient()
	}
	sv.UpdateEnd(updt)
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
		trgst = "radialGradient"
	} else {
		trgst = "linearGradient"
	}
	url := "url(#" + trgst
	if strings.HasPrefix(pstr, url) {
		gr := GradientByName(gii, pstr)
		gr.StopsName = stops
		UpdateGradientStops(gr)
		return gr, NameToURL(gr.Nm)
	}
	if strings.HasPrefix(pstr, "url(#") { // wrong kind
		DeleteNodeGradient(gii, pstr)
	}
	gr, url := AddNewNodeGradient(gii, radial, stops)
	gii.SetProp(prop, url)
	return gr, url
}

// UpdateNodeGradientPoints updates the points for node based on current bbox
func UpdateNodeGradientPoints(gii gi.Node2D, prop string) {
	ps := gii.Prop(prop)
	if ps == nil {
		return
	}
	pstr := ps.(string)
	url := "url(#"
	if !strings.HasPrefix(pstr, url) {
		return
	}
	gr := GradientByName(gii, pstr)
	if gr == nil {
		return
	}
	bbox := gii.(NodeSVG).SVGLocalBBox()
	gr.Grad.SetGradientPoints(bbox)
	gr.Grad.Gradient.Matrix = rasterx.Identity
}

// CloneNodeGradientProp creates a new clone of the existing gradient for node
// if set for given property key ("fill" or "stroke").
// returns new gradient.
func CloneNodeGradientProp(gii gi.Node2D, prop string) *gi.Gradient {
	ps := gii.Prop(prop)
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
	gr := GradientByName(gii, pstr)
	if gr == nil {
		return nil
	}
	ngr, url := AddNewNodeGradient(gii, radial, gr.StopsName)
	gii.SetProp(prop, url)
	ngr.Grad.CopyFrom(&gr.Grad)
	return gr
}

// DeleteNodeGradientProp deletes any existing gradient for node
// if set for given property key ("fill" or "stroke").
// Returns true if deleted.
func DeleteNodeGradientProp(gii gi.Node2D, prop string) bool {
	ps := gii.Prop(prop)
	if ps == nil {
		return false
	}
	pstr := ps.(string)
	if !strings.HasPrefix(pstr, "url(#radialGradient") && !strings.HasPrefix(pstr, "url(#linearGradient") {
		return false
	}
	return DeleteNodeGradient(gii, pstr)
}

const SVGRefCountKey = "SVGRefCount"

func IncRefCount(k ki.Ki) {
	rc := k.Prop(SVGRefCountKey).(int)
	rc++
	k.SetProp(SVGRefCountKey, rc)
}

// RemoveOrphanedDefs removes any items from Defs that are not actually referred to
// by anything in the current SVG tree.  Returns true if items were removed.
// Does not remove gradients with StopsName = "" with extant stops -- these
// should be removed manually, as they are not automatically generated.
func (sv *SVG) RemoveOrphanedDefs() bool {
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	refkey := SVGRefCountKey
	for _, k := range sv.Defs.Kids {
		k.SetProp(refkey, 0)
	}
	sv.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		pr := k.Properties()
		for _, v := range *pr {
			ps := kit.ToString(v)
			if !strings.HasPrefix(ps, "url(#") {
				continue
			}
			nm := NameFromURL(ps)
			el := sv.FindDefByName(nm)
			if el != nil {
				IncRefCount(el)
			}
		}
		if gr, isgr := k.(*gi.Gradient); isgr {
			if gr.StopsName != "" {
				el := sv.FindDefByName(gr.StopsName)
				if el != nil {
					IncRefCount(el)
				}
			} else {
				if gr.Grad.Gradient != nil && len(gr.Grad.Gradient.Stops) > 0 {
					IncRefCount(k) // keep us around
				}
			}
		}
		return ki.Continue
	})
	sz := len(sv.Defs.Kids)
	del := false
	for i := sz - 1; i >= 0; i-- {
		k := sv.Defs.Kids[i]
		rc := k.Prop(refkey).(int)
		if rc == 0 {
			fmt.Printf("Deleting unused item: %s\n", k.Name())
			sv.Defs.Kids.DeleteAtIndex(i)
			del = true
		} else {
			k.DeleteProp(refkey)
		}
	}
	sv.UpdateEnd(updt)
	return del
}
