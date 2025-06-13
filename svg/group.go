// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/math32"
)

// Group groups together SVG elements.
// Provides a common transform for all group elements
// and shared style properties.
type Group struct {
	NodeBase
}

func (g *Group) SVGName() string { return "g" }

func (g *Group) EnforceSVGName() bool { return false }

func (g *Group) BBoxes(sv *SVG, parTransform math32.Matrix2) {
	g.BBoxesFromChildren(sv, parTransform)
}

func (g *Group) IsVisible(sv *SVG) bool {
	if g == nil || g.This == nil || !g.Paint.Display { // does not check g.Paint.Off!
		return false
	}
	nvis := g.VisBBox == math32.Box2{}
	if nvis && !g.isDef {
		return false
	}
	return true
}

func (g *Group) Render(sv *SVG) {
	if !g.IsVisible(sv) {
		return
	}
	pc := g.Painter(sv)
	pc.PushContext(&g.Paint, nil)
	g.RenderChildren(sv)
	pc.PopContext()
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Group) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	g.Paint.Transform.SetMul(xf)
	g.SetTransformProperty()
}
