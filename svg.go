// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/rcoreilly/goki/ki/kit"
	// "fmt"
	"image"
)

////////////////////////////////////////////////////////////////////////////////////////
//  SVG

// SVG is a viewport for containing SVG drawing objects, correponding to the svg tag in html -- it provides its own
type SVG struct {
	Viewport2D
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_SVG = kit.Types.AddType(&SVG{}, nil)

func (g *SVG) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *SVG) AsViewport2D() *Viewport2D {
	return &g.Viewport2D
}

func (g *SVG) AsLayout2D() *Layout {
	return nil
}

func (g *SVG) Init2D() {
	g.Init2DBase()
}

func (g *SVG) Style2D() {
	// todo: check parentage, set flags to indicate if it is an svg-encapsulated svg
	// or not -- if not, use Widget styling
	g.Style2DSVG()
	g.Style2DSVG()
}

func (g *SVG) Size2D() {
	g.InitLayout2D()
	g.LayData.AllocSize.SetFromPoint(g.BBox2D().Size())
}

func (g *SVG) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // use layout styles
	// g.Layout2DChildren() // we
}

func (g *SVG) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *SVG) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox // no margin, padding, etc
}

func (g *SVG) Render2D() {
	if g.PushBounds() {
		if g.Fill {
			pc := &g.Paint
			rs := &g.Render
			pc.FillStyle.SetColor(&g.Style.Background.Color)
			pc.StrokeStyle.SetColor(nil)
			pc.DrawRectangle(rs, float64(g.VpBBox.Min.X), float64(g.VpBBox.Min.Y), float64(g.VpBBox.Max.X-g.VpBBox.Min.X), float64(g.VpBBox.Max.Y-g.VpBBox.Min.Y))
			pc.FillStrokeClear(rs)
		}
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *SVG) CanReRender2D() bool {
	// todo: could optimize by checking for an opaque fill, and same bbox
	return false
}

func (g *SVG) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &SVG{}

////////////////////////////////////////////////////////////////////////////////////////
//  todo parsing code etc
