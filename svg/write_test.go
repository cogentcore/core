// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg_test

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"cogentcore.org/core/math32"
	_ "cogentcore.org/core/paint/renderers"
	. "cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// richSVG has one of each kind of element that is written correctly,
// with gradients, styles, transforms, markers, and text (see the
// TestWrite* tests below for those that are not).
const richSVG = `<svg width="200" height="100" viewBox="0 0 200 100" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
<title>Rich</title>
<desc>All elements</desc>
<defs>
  <linearGradient id="stops1">
    <stop offset="0" stop-color="#ff0000"/>
    <stop offset="1" stop-color="#0000ff"/>
  </linearGradient>
  <linearGradient id="lin2" xlink:href="#stops1" x1="0" y1="0" x2="1" y2="0" gradientTransform="rotate(10)"/>
  <radialGradient id="rad3" cx="0.5" cy="0.5" r="0.5" fx="0.4" fy="0.4" gradientUnits="userSpaceOnUse">
    <stop offset="0" stop-color="white"/>
    <stop offset="1" stop-color="black"/>
  </radialGradient>
  <marker id="arrow4" refX="5" refY="5" orient="auto">
    <path d="M0 0L10 5L0 10z"/>
  </marker>
</defs>
<g id="g6" transform="translate(10,5)" opacity="0.8">
  <rect id="rect7" x="1" y="2" width="30" height="20" rx="3" fill="url(#lin2)" stroke="#008000"/>
  <circle id="circle8" cx="50" cy="20" r="10" fill="url(#rad3)" stroke="black"/>
  <ellipse id="ellipse9" cx="80" cy="20" rx="15" ry="8" style="fill: #ffff00"/>
</g>
<line id="line10" x1="0" y1="50" x2="60" y2="60" stroke="black" marker-end="url(#arrow4)"/>
<polyline id="polyline11" points="70,50 80,60 90,50 100,60" fill="none" stroke="black" marker-start="url(#arrow4)" marker-mid="url(#arrow4)"/>
<polygon id="polygon12" points="110,50 130,50 120,70" fill="#800080" marker-end="url(#arrow4)"/>
<path id="path13" d="M140 50 Q150 40 160 50 C170 60 180 40 190 50" stroke="black" fill="none" marker-start="url(#arrow4)" marker-end="url(#arrow4)" marker-mid="url(#arrow4)"/>
<text id="text14" x="10" y="90" font-size="12" fill="black">a &lt;b&gt; &amp; "c"</text>
</svg>`

// sortStyles returns the given XML with the declarations of each style
// attribute sorted.
func sortStyles(xml string) string {
	lines := strings.Split(xml, "\n")
	for i, ln := range lines {
		if v, ok := strings.CutPrefix(ln, `style="`); ok {
			decls := strings.Split(strings.TrimSuffix(v, `;"`), ";")
			slices.Sort(decls)
			lines[i] = `style="` + strings.Join(decls, ";") + `;"`
		}
	}
	return strings.Join(lines, "\n")
}

// openString returns a new SVG read from the given source.
func openString(t *testing.T, src string) *SVG {
	sv := NewSVG(math32.Vec2(200, 100))
	require.NoError(t, sv.ReadXML(strings.NewReader(src)))
	return sv
}

// nodeTypes returns the type names of the nodes of the tree, in order.
func nodeTypes(sv *SVG) []string {
	var types []string
	SVGWalkDown(sv.Root, func(sn Node, snb *NodeBase) bool {
		types = append(types, sn.SVGName())
		return tree.Continue
	})
	return types
}

// TestWriteRoundTrip tests that writing an SVG and reading it back
// gives the same elements, and the same output when written again.
func TestWriteRoundTrip(t *testing.T) {
	sv := openString(t, richSVG)
	assert.Equal(t, "Rich", sv.Title)
	assert.Equal(t, "All elements", sv.Desc)

	out := sv.XMLString()
	for _, s := range []string{`viewBox="0 0 200 100"`, `<linearGradient`, `<radialGradient`, `<marker`,
		`<rect`, `<circle`, `<ellipse`, `<line`, `<polyline`, `<polygon`, `<path`, `<text`,
		`&lt;b&gt; &amp;`, `id="rect7"`} {
		assert.Contains(t, out, s)
	}

	sv2 := openString(t, out)
	assert.Equal(t, nodeTypes(sv), nodeTypes(sv2))
	// note: the declarations of a style are written in map order
	assert.Equal(t, sortStyles(out), sortStyles(sv2.XMLString()), "writing is stable")

	r := sv2.FindNamedElement("rect7").(*Rect)
	assert.Equal(t, math32.Vec2(1, 2), r.Pos)
	assert.Equal(t, math32.Vec2(30, 20), r.Size)
	assert.Equal(t, float32(3), r.Radius.X)

	var b bytes.Buffer
	require.NoError(t, sv.WriteXML(&b, true))
	assert.Contains(t, b.String(), "\n  <", "indented")

	fn := filepath.Join(t.TempDir(), "rich.svg")
	require.NoError(t, sv.SaveXML(fn))
	sv3 := NewSVG(math32.Vec2(200, 100))
	require.NoError(t, sv3.OpenXML(fn))
	assert.Equal(t, nodeTypes(sv), nodeTypes(sv3))
	sv4 := NewSVG(math32.Vec2(200, 100))
	require.NoError(t, sv4.OpenFS(os.DirFS(filepath.Dir(fn)), "rich.svg"))
	assert.Equal(t, nodeTypes(sv), nodeTypes(sv4))
	assert.Error(t, sv4.SaveXML(filepath.Join(t.TempDir(), "no", "dir", "x.svg")))

	cl := sv.CloneSVG(math32.Vec2(100, 50))
	assert.Equal(t, nodeTypes(sv), nodeTypes(cl))
	assert.Equal(t, image.Pt(100, 50), cl.Geom.Size)
}

// TestRenderMarkers tests rendering the elements with markers, and
// saving the rendering to image and pdf files.
func TestRenderMarkers(t *testing.T) {
	sv := openString(t, richSVG)
	img := sv.RenderImage()
	assert.Equal(t, image.Pt(200, 100), img.Bounds().Size())
	// the purple polygon is drawn
	r, g, b, _ := img.At(120, 55).RGBA()
	assert.Greater(t, r, b/2)
	assert.Less(t, g, r)

	dir := t.TempDir()
	require.NoError(t, sv.SaveImage(filepath.Join(dir, "rich.png")))
	require.NoError(t, sv.SaveImageSize(filepath.Join(dir, "small.png"), 100, 50))
	require.NoError(t, sv.SavePDF(filepath.Join(dir, "rich.pdf")))
	for _, f := range []string{"rich.png", "small.png", "rich.pdf"} {
		st, err := os.Stat(filepath.Join(dir, f))
		require.NoError(t, err)
		assert.Greater(t, st.Size(), int64(100), f)
	}
	assert.Equal(t, image.Pt(200, 100), sv.Geom.Size, "the size is restored")

	img, err := SVGToImage([]byte(richSVG), math32.Vec2(50, 25))
	assert.NoError(t, err)
	assert.Equal(t, image.Pt(50, 25), img.Bounds().Size())
}

// roundTrip returns the SVG read from the given source, and that read
// from its written XML.
func roundTrip(t *testing.T, body string) (*SVG, *SVG) {
	src := `<svg viewBox="0 0 200 100" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">` + body + `</svg>`
	sv := openString(t, src)
	out := sv.XMLString()
	sv2 := NewSVG(math32.Vec2(200, 100))
	require.NoError(t, sv2.ReadXML(strings.NewReader(out)), out)
	return sv, sv2
}

// TestWriteSymbol tests writing a symbol, which must be valid XML.
func TestWriteSymbol(t *testing.T) {
	sv := openString(t, `<svg xmlns="http://www.w3.org/2000/svg"><defs><symbol id="s1"><circle r="4"/></symbol></defs></svg>`)
	out := sv.XMLString()
	assert.Contains(t, out, "<symbol")
	roundTrip(t, `<defs><symbol id="s1"><circle r="4"/></symbol></defs>`)
}

// TestWriteStopOpacity tests writing the opacity of a gradient stop.
func TestWriteStopOpacity(t *testing.T) {
	sv, sv2 := roundTrip(t, `<defs><linearGradient id="g1"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue" stop-opacity="0.5"/></linearGradient></defs><rect fill="url(#g1)" width="10" height="10"/>`)
	stops := sv.FindNamedElement("g1").(*Gradient).Grad.AsBase().Stops
	assert.Equal(t, float32(0.5), stops[1].Opacity, "read")
	stops2 := sv2.FindNamedElement("g1").(*Gradient).Grad.AsBase().Stops
	assert.Equal(t, float32(0.5), stops2[1].Opacity)
}

// TestWriteMarker tests writing the view box and size of a marker.
func TestWriteMarker(t *testing.T) {
	sv, sv2 := roundTrip(t, `<defs><marker id="m1" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0 0L10 5L0 10z"/></marker></defs>`)
	m := sv.FindNamedElement("m1").(*Marker)
	assert.Equal(t, math32.Vec2(6, 6), m.Size, "read")
	assert.Equal(t, math32.Vec2(10, 10), m.ViewBox.Size)
	m2 := sv2.FindNamedElement("m1").(*Marker)
	assert.Equal(t, m.Size, m2.Size)
	assert.Equal(t, m.ViewBox, m2.ViewBox)
}

// TestWriteStyleSheet tests writing a style sheet, whose rules must be
// kept.
func TestWriteStyleSheet(t *testing.T) {
	sv, sv2 := roundTrip(t, `<style>.thick { stroke-width: 3 }</style><rect id="r1" class="thick" width="10" height="10" stroke="black"/>`)
	sv.RenderImage()
	sv2.RenderImage()
	assert.Equal(t, float32(3), sv.FindNamedElement("r1").AsNodeBase().Paint.Stroke.Width.Value, "read")
	assert.Equal(t, float32(3), sv2.FindNamedElement("r1").AsNodeBase().Paint.Stroke.Width.Value)
}

// textNodes returns the number of text nodes of the SVG.
func textNodes(sv *SVG) int {
	n := 0
	SVGWalkDown(sv.Root, func(sn Node, snb *NodeBase) bool {
		if _, ok := sn.(*Text); ok {
			n++
		}
		return tree.Continue
	})
	return n
}

// TestWriteTspans tests that writing text with tspans keeps the same
// structure when read back.
func TestWriteTspans(t *testing.T) {
	sv, sv2 := roundTrip(t, `<text x="10" y="90">ab<tspan x="80">cd</tspan></text>`)
	assert.Equal(t, 3, textNodes(sv), "the text and its two spans")
	assert.Equal(t, 3, textNodes(sv2))
}

// TestUpdateBBoxesText tests updating the bounding boxes of text before
// the SVG is rendered.
func TestUpdateBBoxesText(t *testing.T) {
	sv := openString(t, `<svg xmlns="http://www.w3.org/2000/svg"><text x="10" y="20">hi</text></svg>`)
	assert.NotPanics(t, sv.UpdateBBoxes)
}
