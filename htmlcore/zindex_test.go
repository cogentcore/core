// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestZIndexFlexItem tests that a flex item with a z-index, which is not
// positioned, is rendered over the items after it with lower ones, as in
// CSS (as the site header of pkg.go.dev, whose dropdown menus are over
// the sticky header of the page, a grid item with a lower z-index), and
// that the z-index of a static element that is not a flex or grid item
// has no effect.
func TestZIndexFlexItem(t *testing.T) {
	src := `<div style="display: flex; flex-direction: column; width: 300px">` +
		`<header id="header" style="z-index: 20; background: gray"><div style="position: relative; width: 100px; height: 40px">` +
		`<div id="menu" style="position: absolute; top: 40px; width: 100px; height: 100px; background: red"></div></div></header>` +
		`<main id="main" style="z-index: 10; height: 200px; background: blue"><div id="block" style="z-index: 5">text</div></main></div>`
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, src))
	ws := map[string]*core.WidgetBase{}
	bd.WalkDown(func(k tree.Node) bool {
		if wb := core.AsWidget(k); wb != nil {
			if id, ok := wb.Property("id").(string); ok {
				ws[id] = wb
			}
		}
		return tree.Continue
	})
	var c image.Point
	img := bd.RenderImage(func() {}, func() {
		require.NotNil(t, ws["menu"])
		bb := ws["menu"].Geom.TotalBBox
		c = bb.Min.Add(bb.Size().Div(2))
		assert.Equal(t, 20, ws["header"].Styles.ZIndex)
		assert.Equal(t, 0, ws["block"].Styles.ZIndex, "the z-index of a block in a block has no effect")
	})
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, color.RGBAModel.Convert(img.At(c.X, c.Y)), "the menu is over the main")
}

// TestBorderTriangle tests the borders of a box of no size with a colored
// top and transparent sides, which is a triangle in CSS (as the arrow of
// the language select of the Rust site, in its ::after): the sides of a
// border are filled between its outer and inner edges, with mitered
// corners.
func TestBorderTriangle(t *testing.T) {
	src := `<div id="tri" style="width: 0; height: 0; border-top: 20px solid red; border-right: 20px solid transparent; border-left: 20px solid transparent"></div>`
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, src))
	var tri *core.WidgetBase
	bd.WalkDown(func(k tree.Node) bool {
		if wb := core.AsWidget(k); wb != nil && wb.Property("id") == "tri" {
			tri = wb
		}
		return tree.Continue
	})
	require.NotNil(t, tri)
	var bb image.Rectangle
	img := bd.RenderImage(func() {}, func() { bb = tri.Geom.TotalBBox })
	red := color.RGBA{255, 0, 0, 255}
	at := func(fx, fy float32) color.RGBA {
		x := bb.Min.X + int(fx*float32(bb.Dx()))
		y := bb.Min.Y + int(fy*float32(bb.Dy()))
		return color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
	}
	assert.Equal(t, red, at(0.5, 0.2), "the middle of the top is red")
	assert.Equal(t, red, at(0.5, 0.8), "the point of the triangle is red")
	assert.NotEqual(t, red, at(0.1, 0.8), "the corners below its sides are not")
	assert.NotEqual(t, red, at(0.9, 0.8), "the corners below its sides are not")
	assert.NotEqual(t, red, at(0.3, 0.75), "beside its sloped edges, it is not")
	assert.NotEqual(t, red, at(0.7, 0.75), "beside its sloped edges, it is not")
}
