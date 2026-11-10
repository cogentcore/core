// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

// TestSVGPageStyles tests that the rules of the style sheets of a page
// apply to the elements of its SVGs (as icons whose fill and visibility
// are set by classes), with the precedence of CSS over the presentation
// attributes, and the units of CSS transforms.
func TestSVGPageStyles(t *testing.T) {
	ctx := NewContext()
	b := core.NewBody()
	doc, err := ParseHTML(ctx, strings.NewReader(`<style>.icon .a { fill: #ff0000 } .icon .hidden { visibility: hidden } .icon .moved { transform: translate(5px, 0) rotate(0deg) }</style>`+
		`<div style="margin: 0"><svg class="icon" viewBox="0 0 20 10" width="200" height="100">`+
		`<rect class="a" width="10" height="10" fill="blue"/>`+
		`<rect class="hidden" x="10" width="10" height="10" fill="green"/>`+
		`<rect class="moved" x="10" width="5" height="10" fill="#0000ff"/></svg></div>`))
	assert.NoError(t, err)
	assert.NoError(t, ReadDocument(ctx, b, doc))
	var svg *html.Node
	walkNodes(doc, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "svg" {
			svg = n
		}
	})
	var sb strings.Builder
	ctx.Node = svg
	assert.NoError(t, html.Render(&sb, ctx.svgForRender(svg)))
	s := sb.String()
	assert.Contains(t, s, `style="fill: #ff0000"`)
	assert.NotContains(t, s, `fill="blue"`, "the style has precedence over the attribute")
	assert.Contains(t, s, `style="display: none"`)
	assert.Contains(t, s, `style="transform: translate(5,0) rotate(0)"`)

	var sw *core.SVG
	b.WalkDown(func(n tree.Node) bool {
		if w, ok := n.(*core.SVG); ok {
			sw = w
		}
		return tree.Continue
	})
	if !assert.NotNil(t, sw) {
		return
	}
	var bb image.Rectangle
	img := b.RenderImage(func() { bb = sw.Geom.ContentBBox })
	// at returns the color at the given fraction of the svg
	at := func(x, y float32) color.RGBA {
		p := image.Pt(bb.Min.X+int(x*float32(bb.Dx())), bb.Min.Y+int(y*float32(bb.Dy())))
		return color.RGBAModel.Convert(img.At(p.X, p.Y)).(color.RGBA)
	}
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, at(0.25, 0.5), "the fill of the style sheet")
	assert.Equal(t, color.RGBA{0, 0, 255, 255}, at(0.875, 0.5), "the transformed rect")
	assert.NotEqual(t, color.RGBA{0, 128, 0, 255}, at(0.55, 0.5), "the hidden rect")
}
