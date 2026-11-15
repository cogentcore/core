// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg_test

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	. "cogentcore.org/core/svg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGradientGeom tests copying the transforms of the user space
// gradients of the fill and stroke of a node to the gradients, and back.
func TestGradientGeom(t *testing.T) {
	sv := openString(t, `<svg viewBox="0 0 100 100">
<defs>
  <linearGradient id="user1" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="10" y2="0"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>
  <linearGradient id="user2" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="10" y2="0"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>
  <linearGradient id="bbox3"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>
</defs>
<rect id="r1" width="10" height="10" fill="url(#user1)" stroke="url(#user2)"/>
<rect id="r2" width="10" height="10" fill="url(#bbox3)"/>
</svg>`)
	r1 := sv.FindNamedElement("r1").(*Rect)
	xf := math32.Translate2D(3, 4)
	r1.GradientFill = xf
	r1.GradientStroke = math32.Scale2D(2, 2)
	r1.GradientUpdateGeom(sv)
	assert.Equal(t, xf, sv.GradientByName(r1, "user1").Grad.AsBase().Transform)
	assert.Equal(t, math32.Scale2D(2, 2), sv.GradientByName(r1, "user2").Grad.AsBase().Transform)

	r1.GradientFill = math32.Identity2()
	r1.GradientStroke = math32.Identity2()
	r1.GradientFromGradients(sv)
	assert.Equal(t, xf, r1.GradientFill, "copied back")
	assert.Equal(t, math32.Scale2D(2, 2), r1.GradientStroke)

	r2 := sv.FindNamedElement("r2").(*Rect)
	r2.GradientFill = xf
	r2.GradientUpdateGeom(sv)
	assert.Equal(t, math32.Identity2(), sv.GradientByName(r2, "bbox3").Grad.AsBase().Transform, "not for the bounding box units")
	r2.GradientFill = math32.Identity2()
	r2.GradientFromGradients(sv)
	assert.Equal(t, math32.Identity2(), r2.GradientFill)
}

// TestMarkers tests finding the markers of nodes, by their own property
// or the generic marker property, and rendering a polygon with markers
// and opacity.
func TestMarkers(t *testing.T) {
	sv := openString(t, `<svg viewBox="0 0 100 100">
<defs>
  <marker id="m1" refX="5" refY="5" markerWidth="4" markerHeight="4" orient="auto"><path d="M0 0L10 5L0 10z" fill="red"/></marker>
  <rect id="notmarker" width="1" height="1"/>
</defs>
<polygon id="pg" points="10,10 90,10 50,90" fill="none" stroke="black" stroke-width="2" opacity="0.5" marker-start="url(#m1)" marker-mid="url(#m1)" marker-end="url(#m1)"/>
<polyline id="pl" points="10,95 90,95" stroke="black" marker="url(#m1)"/>
<line id="ln" x1="0" y1="0" x2="1" y2="1" marker-end="url(#notmarker)"/>
<line id="ln2" x1="0" y1="0" x2="1" y2="1" marker-end="url(#missing)"/>
</svg>`)
	pg := sv.FindNamedElement("pg").(Node)
	require.NotNil(t, sv.MarkerByName(pg, "marker-start"))
	pl := sv.FindNamedElement("pl").(Node)
	assert.NotNil(t, sv.MarkerByName(pl, "marker-end"), "the generic marker property")
	assert.Nil(t, sv.MarkerByName(sv.FindNamedElement("ln").(Node), "marker-end"), "not a marker")
	assert.Nil(t, sv.MarkerByName(sv.FindNamedElement("ln2").(Node), "marker-end"), "missing")
	assert.Nil(t, sv.MarkerByName(sv.FindNamedElement("ln2").(Node), "marker-start"), "none")

	img := sv.RenderImage().(*image.RGBA)
	c := img.RGBAAt(50*img.Bounds().Dx()/100, 10*img.Bounds().Dy()/100)
	assert.Greater(t, int(c.A), 0, "the stroke of the polygon")
	m1 := sv.FindNamedElement("m1").(*Marker)
	m1.BBoxes(sv, math32.Identity2())
	assert.Equal(t, "marker", m1.SVGName())
	assert.False(t, m1.EnforceSVGName())
}

// TestPatternFill tests the pattern fill of a node as an image and a
// gradient: its bounds, color model, colors, and a copy.
func TestPatternFill(t *testing.T) {
	sv := openString(t, `<svg viewBox="0 0 40 40" width="40" height="40">
<defs>
  <pattern id="pat" width="10" height="10" patternUnits="userSpaceOnUse"><rect width="5" height="5" fill="red"/></pattern>
</defs>
<rect id="r" width="40" height="40" fill="url(#pat)"/>
</svg>`)
	img := sv.RenderImage().(*image.RGBA)
	// (the 40×40 view box is centered in the 200×100 image, scaled by 2.5)
	at := func(x, y float32) color.RGBA { return img.RGBAAt(int(50+2.5*x), int(2.5*y)) }
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, at(2, 2))
	assert.Equal(t, uint8(0), at(7, 7).A)
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, at(12, 32), "tiled")

	r := sv.FindNamedElement("r").(*Rect)
	pf, ok := r.Paint.Fill.Color.(gradient.Gradient)
	require.True(t, ok, "%T", r.Paint.Fill.Color)
	assert.Equal(t, color.RGBAModel, pf.ColorModel())
	assert.True(t, pf.Bounds().Dx() > 1e6, "unbounded")
	cp := gradient.CopyOf(pf)
	require.NotNil(t, cp)
	assert.Equal(t, pf.At(12, 12), cp.At(12, 12), "the copy shares the tile")
	pat := sv.FindNamedElement("pat").(*Pattern)
	assert.Equal(t, "pattern", pat.SVGName())
	assert.False(t, pat.EnforceSVGName())
	pat.Render(sv) // patterns are not rendered themselves
}

// TestWriteLayers tests writing the groups of inkscape layers, with their
// group mode, locking, and an inkscape property of a marker.
func TestWriteLayers(t *testing.T) {
	sv := openString(t, `<svg viewBox="0 0 10 10" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" xmlns:sodipodi="http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd">
<defs><marker id="mk" inkscape:stockid="Arrow" inkscape:isstock="true" overflow="visible"><path d="M0 0L1 1"/></marker></defs>
<g id="layer1" inkscape:groupmode="layer" inkscape:label="One" sodipodi:insensitive="true"><rect width="1" height="1"/></g>
<g id="layer2" inkscape:groupmode="layer" style="display:none"><rect width="1" height="1"/></g>
</svg>`)
	out := sv.XMLString()
	assert.Contains(t, out, `inkscape:groupmode="layer"`)
	assert.Contains(t, out, `sodipodi:insensitive="true"`)
	assert.Contains(t, out, `style="display:inline"`, "a visible layer")
	assert.Contains(t, out, `inkscape:stockid="Arrow"`)
	assert.Contains(t, out, `overflow:visible`)
	sv2 := openString(t, out)
	assert.NotNil(t, sv2.FindNamedElement("layer1"))
}
