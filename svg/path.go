// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
)

// Path renders SVG data sequences that can render just about anything
type Path struct {
	NodeBase

	// Path data using paint/ppath representation.
	Data ppath.Path `xml:"-" set:"-"`

	// string version of the path data
	DataStr string `xml:"d"`
}

func (g *Path) SVGName() string { return "path" }

func (g *Path) SetPos(pos math32.Vector2) {
	// todo: set first point
}

func (g *Path) SetSize(sz math32.Vector2) {
	// todo: scale bbox
}

// SetData sets the path data to given string, parsing it into an optimized
// form used for rendering
func (g *Path) SetData(data string) error {
	g.DataStr = data
	var err error
	g.Data, err = ppath.ParseSVGPath(data)
	if err != nil {
		return err
	}
	return err
}

func (g *Path) LocalBBox() math32.Box2 {
	bb := g.Data.FastBounds()
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min.SetSubScalar(hlw)
	bb.Max.SetAddScalar(hlw)
	return bb
}

func (g *Path) Render(sv *SVG) {
	sz := len(g.Data)
	if sz < 2 {
		return
	}
	vis, pc := g.IsVisible(sv)
	if !vis {
		return
	}
	pc.State.Path = g.Data
	pc.PathDone()

	g.BBoxes(sv)

	// todo: use path algos for this:
	// if mrk := sv.MarkerByName(g, "marker-start"); mrk != nil {
	// 	// todo: could look for close-path at end and find angle from there..
	// 	stv, ang := PathDataStart(g.Data)
	// 	mrk.RenderMarker(sv, stv, ang, g.Paint.Stroke.Width.Dots)
	// }
	// if mrk := sv.MarkerByName(g, "marker-end"); mrk != nil {
	// 	env, ang := PathDataEnd(g.Data)
	// 	mrk.RenderMarker(sv, env, ang, g.Paint.Stroke.Width.Dots)
	// }
	// if mrk := sv.MarkerByName(g, "marker-mid"); mrk != nil {
	// 	var ptm2, ptm1, pt math32.Vector2
	// 	gotidx := 0
	// 	PathDataIterFunc(g.Data, func(idx int, cmd PathCmds, ptIndex int, cp math32.Vector2, ctrls []math32.Vector2) bool {
	// 		ptm2 = ptm1
	// 		ptm1 = pt
	// 		pt = cp
	// 		if gotidx < 2 {
	// 			gotidx++
	// 			return true
	// 		}
	// 		if idx >= sz-3 { // todo: this is approximate...
	// 			return false
	// 		}
	// 		ang := 0.5 * (math32.Atan2(pt.Y-ptm1.Y, pt.X-ptm1.X) + math32.Atan2(ptm1.Y-ptm2.Y, ptm1.X-ptm2.X))
	// 		mrk.RenderMarker(sv, ptm1, ang, g.Paint.Stroke.Width.Dots)
	// 		gotidx++
	// 		return true
	// 	})
	// }

	g.RenderChildren(sv)
}

// UpdatePathString sets the path string from the Data
func (g *Path) UpdatePathString() {
	g.DataStr = g.Data.ToSVG()
}

////////  Transforms

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Path) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	// path may have horiz, vert elements -- only gen soln is to transform
	g.Paint.Transform.SetMul(xf)
	g.SetProperty("transform", g.Paint.Transform.String())
}

// ApplyDeltaTransform applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Path) ApplyDeltaTransform(sv *SVG, trans math32.Vector2, scale math32.Vector2, rot float32, pt math32.Vector2) {
	crot := g.Paint.Transform.ExtractRot()
	if rot != 0 || crot != 0 {
		xf, lpt := g.DeltaTransform(trans, scale, rot, pt, false) // exclude self
		g.Paint.Transform.SetMulCenter(xf, lpt)
		g.SetProperty("transform", g.Paint.Transform.String())
	} else {
		xf, lpt := g.DeltaTransform(trans, scale, rot, pt, true) // include self
		g.ApplyTransformImpl(xf, lpt)
		g.GradientApplyTransformPt(sv, xf, lpt)
	}
}

// ApplyTransformImpl does the implementation of applying a transform to all points
func (g *Path) ApplyTransformImpl(xf math32.Matrix2, lpt math32.Vector2) {
	g.Data.Transform(xf)
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Path) WriteGeom(sv *SVG, dat *[]float32) {
	sz := len(g.Data)
	*dat = slicesx.SetLength(*dat, sz+6)
	for i := range g.Data {
		(*dat)[i] = float32(g.Data[i])
	}
	g.WriteTransform(*dat, sz)
	g.GradientWritePts(sv, dat)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Path) ReadGeom(sv *SVG, dat []float32) {
	sz := len(g.Data)
	g.Data = ppath.Path(dat)
	g.ReadTransform(dat, sz)
	g.GradientReadPts(sv, dat)
}
