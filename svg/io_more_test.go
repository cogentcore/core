// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg_test

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"strings"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/math32"
	. "cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pngDataURI returns a data URI of a png image of the given size and color.
func pngDataURI(w, h int, c color.RGBA) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	b, mime := imagex.ToBase64(img, imagex.PNG)
	return "data:" + mime + ";base64," + string(b)
}

// moreSVG has the less common elements and attributes: a nested svg, an
// image with data, text with lists of positions, marker attributes, use
// with a size, pattern and symbol view boxes, inkscape and sodipodi
// metadata, flow text, filters, masks, links, and an unknown element.
var moreSVG = `<svg width="20cm" height="10cm" viewBox="0 0 200 100" preserveAspectRatio="xMinYMin" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:sodipodi="http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:cc="http://creativecommons.org/ns#" xmlns:dc="http://purl.org/dc/elements/1.1/">
<metadata id="meta1"><rdf:RDF><cc:Work><dc:format>image/svg+xml</dc:format><dc:type/></cc:Work></rdf:RDF></metadata>
<sodipodi:namedview id="nv2" pagecolor="#ffffff"><inkscape:grid id="grid3"/><sodipodi:guide id="guide4"/></sodipodi:namedview>
<defs>
  <inkscape:perspective id="persp5"/>
  <inkscape:path-effect id="pe6" effect="spiro"/>
  <marker id="mk7" refX="1" refY="2" markerWidth="4" markerHeight="5" markerUnits="userSpaceOnUse" viewBox="0 0 10 10" orient="45">
    <path d="M0 0L10 5L0 10z"/>
  </marker>
  <filter id="flt8"><feGaussianBlur id="blur9" stdDeviation="2"/></filter>
  <pattern id="pat10" width="10" height="10" viewBox="0 0 5 5" preserveAspectRatio="none" patternUnits="userSpaceOnUse">
    <rect width="5" height="5" fill="red"/>
  </pattern>
  <symbol id="sym11" viewBox="0 0 20 20" preserveAspectRatio="xMidYMid slice"><circle cx="10" cy="10" r="10"/></symbol>
  <mask id="mask12"><rect width="200" height="100" fill="white"/></mask>
</defs>
<svg id="nested13" x="10" y="10" width="50" height="40" viewBox="0 0 100 80"><rect width="100" height="80" fill="blue"/></svg>
<image id="img14" x="5" y="6" width="20" height="10" preserveAspectRatio="none" xlink:href="` + pngDataURI(4, 2, color.RGBA{0, 255, 0, 255}) + `"/>
<image id="img15" x="1" y="1" width="2" height="2" href="other.png"/>
<text id="text16" x="10 20 30" y="50 51" dx="1 2" dy="3" rotate="10 20" textLength="40" lengthAdjust="spacingAndGlyphs">abc<tspan id="tspan17" x="40" y="60" lengthAdjust="spacing">d</tspan></text>
<use id="use18" xlink:href="#sym11" x="100" y="10" width="30" height="30" transform="scale(2)"/>
<flowRoot id="flow19"><flowRegion><rect width="10" height="10"/></flowRegion><flowPara>flowing</flowPara></flowRoot>
<a id="link20" href="https://example.com"><rect width="5" height="5" mask="url(#mask12)" filter="url(#flt8)"/></a>
<switch id="switch21"><rect width="3" height="3"/></switch>
<unknown id="unknown22"><rect width="1" height="1"/></unknown>
<polyline id="pl23" points="0,0 10,10" marker-end="url(#mk7)" stroke="black"/>
<rect id="patterned24" x="150" y="50" width="40" height="40" fill="url(#pat10)"/>
</svg>`

// TestReadMore tests reading the less common elements and attributes,
// writing them, and reading them back.
func TestReadMore(t *testing.T) {
	sv := openString(t, moreSVG)
	types := nodeTypes(sv)
	for _, ty := range []string{"metadata", "image", "text", "tspan", "flow"} {
		assert.Contains(t, types, ty)
	}
	var defs []string
	SVGWalkDown(sv.Defs, func(sn Node, snb *NodeBase) bool {
		defs = append(defs, sn.SVGName())
		return tree.Continue
	})
	for _, ty := range []string{"filter", "marker", "pattern", "symbol", "mask"} {
		assert.Contains(t, defs, ty)
	}
	assert.InDelta(t, 20, sv.PhysicalWidth.Value, 1e-5)
	assert.Equal(t, math32.Vec2(200, 100), sv.Root.ViewBox.Size)

	img := sv.FindNamedElement("img14").(*Image)
	assert.Equal(t, math32.Vec2(5, 6), img.Pos)
	assert.Equal(t, math32.Vec2(20, 10), img.Size)
	require.NotNil(t, img.Pixels, "the image of the data")
	assert.Equal(t, image.Pt(4, 2), img.Pixels.Image.Bounds().Size())
	img15 := sv.FindNamedElement("img15").(*Image)
	assert.Nil(t, img15.Pixels, "a url is not loaded")
	assert.Equal(t, "other.png", img15.Filename, "but kept")

	tx := sv.FindNamedElement("text16").(*Text)
	assert.Equal(t, []float32{10, 20, 30}, tx.CharPosX)
	assert.Equal(t, []float32{50, 51}, tx.CharPosY)
	assert.Equal(t, []float32{1, 2}, tx.CharPosDX)
	assert.Equal(t, []float32{3}, tx.CharPosDY)
	assert.Equal(t, []float32{10, 20}, tx.CharRots)
	assert.Equal(t, float32(40), tx.TextLength)
	assert.True(t, tx.AdjustGlyphs)
	assert.False(t, sv.FindNamedElement("tspan17").(*Text).AdjustGlyphs)

	mk := sv.FindNamedElement("mk7").(*Marker)
	assert.Equal(t, math32.Vec2(1, 2), mk.RefPos)
	assert.Equal(t, math32.Vec2(4, 5), mk.Size)
	assert.Equal(t, "45", mk.Orient)
	assert.Equal(t, math32.Vec2(10, 10), mk.ViewBox.Size)
	assert.Equal(t, UserSpaceOnUse, mk.Units)

	img2 := sv.RenderImage()
	require.NotNil(t, img2)

	// a use is a group with the transform, and a copy of the symbol
	us := sv.FindNamedElement("use18").(*Group)
	assert.Equal(t, math32.Scale2D(2, 2).Translate(100, 10), us.Paint.Transform)
	assert.Equal(t, 1, us.NumChildren())

	out := sv.XMLString()
	for _, s := range []string{"<image", "data:image/png;base64,", "<marker", "<pattern", "<symbol"} {
		assert.Contains(t, out, s)
	}
	sv2 := openString(t, out)
	img3 := sv2.FindNamedElement("img14").(*Image)
	require.NotNil(t, img3.Pixels, "the image data is read back")
	assert.Equal(t, image.Pt(4, 2), img3.Pixels.Image.Bounds().Size())
	mk2 := sv2.FindNamedElement("mk7").(*Marker)
	assert.Equal(t, mk.RefPos, mk2.RefPos)
	assert.Equal(t, UserSpaceOnUse, mk2.Units, "the units are written")
	assert.Equal(t, 1, strings.Count(out, "markerUnits"), "once")
	assert.Equal(t, "other.png", sv2.FindNamedElement("img15").(*Image).Filename, "the link is written")
}

// TestReadErrors tests the errors of reading and opening SVG files.
func TestReadErrors(t *testing.T) {
	sv := NewSVG(math32.Vec2(10, 10))
	assert.Error(t, sv.OpenXML("testdata/none.svg"))
	assert.Error(t, sv.OpenXML(t.TempDir()), "a directory")
	assert.Error(t, sv.OpenFS(os.DirFS("."), "none.svg"))
	assert.Error(t, sv.ReadXML(strings.NewReader(`<svg viewBox="0 0 1"></svg>`)), "a view box of 3 numbers")
	assert.Error(t, sv.ReadXML(strings.NewReader(`<svg><polygon points="1,2 3"/></svg>`)), "an odd number of points")
	assert.Error(t, sv.ReadXML(strings.NewReader(`<svg><polyline points="1,2 3"/></svg>`)), "an odd number of points")
	assert.Error(t, sv.ReadXML(strings.NewReader(`<svg><rect x="a"/></svg>`)))
	assert.NoError(t, sv.ReadXML(strings.NewReader(``)), "empty")
}

// TestGroupFilter tests reading only the group of a given id, as for the
// glyphs of a font.
func TestGroupFilter(t *testing.T) {
	sv := NewSVG(math32.Vec2(10, 10))
	sv.GroupFilter = "keep"
	require.NoError(t, sv.ReadXML(strings.NewReader(`<svg>
<g id="skip"><rect id="r1" width="1" height="1"/><g><circle id="c1" r="1"/></g></g>
<g id="keep"><rect id="r2" width="2" height="2"/></g>
<g id="skip2"><rect id="r3" width="3" height="3"/></g>
</svg>`)))
	assert.Nil(t, sv.FindNamedElement("r1"))
	assert.Nil(t, sv.FindNamedElement("c1"))
	assert.NotNil(t, sv.FindNamedElement("r2"))
	assert.Nil(t, sv.FindNamedElement("r3"))
}

// TestTextSpanSpaceBBox tests that the white space between the spans of a
// text (as in each line of a text that Inkscape saves) does not make the
// box of the text include the origin.
func TestTextSpanSpaceBBox(t *testing.T) {
	sv := openString(t, `<svg width="200" height="200" viewBox="0 0 200 200">
<text id="txt" x="100" y="100" font-size="10">
  <tspan x="100" y="100">first</tspan>
  <tspan x="100" y="120">second</tspan>
</text>
</svg>`)
	sv.RenderImage()
	tx := sv.FindNamedElement("txt").(*Text)
	var spans math32.Box2
	n := 0
	for _, k := range tx.Children {
		if kt := k.(*Text); kt.Text != "" {
			if n == 0 {
				spans = kt.BBox
			} else {
				spans = spans.Union(kt.BBox)
			}
			n++
		}
	}
	require.GreaterOrEqual(t, n, 2, "the two lines (and the space between them)")
	assert.Equal(t, spans, tx.BBox, "the box of the lines, not the origin")
	assert.Greater(t, tx.BBox.Min.Y, float32(10), "%v", tx.BBox)
}
