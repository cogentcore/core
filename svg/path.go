// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/base/errors"
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
	bb := g.Data.FastBounds()
	csz := bb.Size()
	if csz.X == 0 || csz.Y == 0 {
		return
	}
	sc := sz.Div(csz)
	g.Data.Transform(math32.Scale2D(sc.X, sc.Y))
}

// SetData sets the path data to given string, parsing it into an optimized
// form used for rendering
func (g *Path) SetData(data string) error {
	d, err := ppath.ParseSVGPath(data)
	if errors.Log(err) != nil {
		return err
	}
	g.DataStr = data
	g.Data = d
	return err
}

func (g *Path) LocalBBox(sv *SVG) math32.Box2 {
	bb := g.Data.FastBounds()
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min.SetSubScalar(hlw)
	bb.Max.SetAddScalar(hlw)
	return bb
}

func (g *Path) Render(sv *SVG) {
	sz := len(g.Data)
	if sz < 2 || !g.IsVisible(sv) {
		return
	}
	pc := g.Painter(sv)
	pc.State.Path = g.Data.Clone() // note: yes this Clone() is absolutely necessary.
	pc.Draw()

	g.PushContext(sv)
	mrk_start := sv.MarkerByName(g, "marker-start")
	mrk_end := sv.MarkerByName(g, "marker-end")
	mrk_mid := sv.MarkerByName(g, "marker-mid")

	if mrk_start != nil || mrk_end != nil || mrk_mid != nil {
		pos := g.Data.Coords()
		dir := g.Data.CoordDirections()
		np := len(pos)
		if mrk_start != nil && np > 1 {
			dir0 := pos[1].Sub(pos[0])
			ang := ppath.Angle(dir0) // dir[0]: has average but last 2 works better
			mrk_start.RenderMarker(sv, pos[0], ang, g.Paint.Stroke.Width.Dots)
		}
		if mrk_end != nil && np > 1 {
			dirn := pos[np-1].Sub(pos[np-2])
			ang := ppath.Angle(dirn) // dir[np-1]: see above
			mrk_end.RenderMarker(sv, pos[np-1], ang, g.Paint.Stroke.Width.Dots)
		}
		if mrk_mid != nil && np > 2 {
			for i := 1; i < np-2; i++ {
				ang := ppath.Angle(dir[i])
				mrk_mid.RenderMarker(sv, pos[i], ang, g.Paint.Stroke.Width.Dots)
			}
		}
	}
	pc.PopContext()
}

// UpdatePathString sets the path string from the Data
func (g *Path) UpdatePathString() {
	g.DataStr = g.Data.ToSVG()
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Path) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	g.Data.Transform(xf)
	g.GradientApplyTransform(sv, xf)
}
