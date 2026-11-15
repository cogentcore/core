// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pdf

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"github.com/stretchr/testify/assert"
)

// renderPDF returns the PDF document written by the given function,
// which is not compressed, so that its content streams can be checked.
func renderPDF(t *testing.T, f func(pd *PDF, sty *styles.Paint)) string {
	t.Helper()
	ctx := units.NewContext()
	ctx.DPI = 72
	var b bytes.Buffer
	pd := New(&b, 100, 100, ctx)
	sty := styles.NewPaint()
	sty.UnitContext = *ctx
	f(pd, sty)
	assert.NoError(t, pd.Close())
	s := b.String()
	assert.True(t, strings.HasPrefix(s, "%PDF-1.7"))
	assert.True(t, strings.HasSuffix(s, "%%EOF\n"))
	return s
}

// strokeStyle sets the style to stroke only, with the given width.
func strokeStyle(sty *styles.Paint, width float32) {
	sty.Fill.Color = nil
	sty.Stroke.Color = colors.Uniform(color.RGBA{255, 0, 0, 255})
	sty.Stroke.Width.Dp(width)
	sty.ToDots()
}

// TestInfo tests the document information and language.
func TestInfo(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		pd.SetInfo("My Title", "Subject", "a, b", "Author", "Creator")
		pd.SetLang("en-US")
		w, h := pd.Size()
		assert.Equal(t, float32(100), w)
		assert.Equal(t, float32(100), h)
	})
	for _, v := range []string{"/Title(My Title)", "/Subject(Subject)", "/Keywords(a, b)", "/Author(Author)", "/Creator(Creator)", "/Producer(cogentcore/pdf)"} {
		assert.Contains(t, s, v)
	}
	assert.Contains(t, s, "/Lang(en-US)", "the language, not the creator")

	s = renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		pd.SetInfo("Grüße", "", "", "", "")
	})
	assert.Contains(t, s, "/Title(\xfe\xff", "a non-ascii title is UTF-16 with a BOM")
	assert.NotContains(t, s, "/Subject", "empty values are not written")
}

// TestPages tests documents with more than one page.
func TestPages(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		pd.NewPage(200, 50)
		pd.NewPage(300, 50)
	})
	assert.Contains(t, s, "/Count 3")
	assert.Equal(t, 3, strings.Count(s, "/Type/Page/"), "three page objects")
	assert.Contains(t, s, "/MediaBox[0 0 300 50]")
}

// TestStrokeStyles tests the operators of stroke caps, joins, and dashes.
func TestStrokeStyles(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		strokeStyle(sty, 2)
		sty.Stroke.Cap = ppath.CapRound
		sty.Stroke.Join = ppath.JoinRound
		pd.Path(*ppath.New().Line(0, 0, 10, 10), sty, math32.Identity2())

		sty.Stroke.Cap = ppath.CapSquare
		sty.Stroke.Join = ppath.JoinMiter
		sty.Stroke.MiterLimit = 4
		sty.Stroke.Dashes = []float32{3}
		sty.Stroke.DashOffset = -1
		pd.Path(*ppath.New().Line(0, 0, 10, 10), sty, math32.Identity2())

		sty.Stroke.Cap = ppath.CapButt
		sty.Stroke.Join = ppath.JoinBevel
		sty.Stroke.Dashes = nil
		pd.Path(*ppath.New().Rectangle(0, 0, 10, 10), sty, math32.Identity2())
	})
	assert.Contains(t, s, " 1 J")
	assert.Contains(t, s, " 1 j")
	assert.Contains(t, s, " 2 J")
	assert.Contains(t, s, " 0 j 4 M", "a miter join has a limit")
	assert.Contains(t, s, " [3 3] 5 d", "an odd dash array is doubled, and a negative phase is made positive")
	assert.Contains(t, s, " 0 J")
	assert.Contains(t, s, " 2 j")
	assert.Contains(t, s, " [] 0 d", "dashes are turned off")
	assert.Contains(t, s, " s\n", "a closed path is closed and stroked")
}

// TestFillRuleOpacity tests even-odd fills, and the graphics states of
// opacities, of which one is made for each value.
func TestFillRuleOpacity(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		sty.Stroke.Color = nil
		sty.Fill.Color = colors.Uniform(color.RGBA{0, 128, 0, 255})
		sty.Fill.Opacity = 0.5
		sty.Fill.Rule = ppath.EvenOdd
		pd.Path(*ppath.New().Rectangle(0, 0, 10, 10), sty, math32.Identity2())
		sty.Fill.Color = colors.Uniform(color.RGBA{0, 0, 255, 255})
		pd.Path(*ppath.New().Rectangle(20, 0, 10, 10), sty, math32.Identity2())
		sty.Fill.Color = colors.Uniform(color.RGBA{128, 128, 128, 255})
		sty.Fill.Opacity = 1
		sty.Fill.Rule = ppath.NonZero
		pd.Path(*ppath.New().Rectangle(40, 0, 10, 10), sty, math32.Identity2())
	})
	assert.Contains(t, s, " f*")
	assert.Contains(t, s, "rg")
	assert.Contains(t, s, " g", "a gray is written as a gray")
	assert.Contains(t, s, "/ExtGState")
	assert.Equal(t, 1, strings.Count(s, "/CA .49804"), "one state for an opacity used twice")
}

// TestLinearGradientStops tests a gradient with more than two stops,
// which uses a stitching function of its stops.
func TestLinearGradientStops(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		g := gradient.NewLinear().AddStop(colors.White, 0).AddStop(colors.Red, 0.5).AddStop(colors.Blue, 1)
		sty.Stroke.Color = nil
		sty.Fill.Color = g
		pd.Path(*ppath.New().Rectangle(0, 0, 30, 20), sty, math32.Identity2())
	})
	assert.Contains(t, s, "/Pattern cs")
	assert.Contains(t, s, "/FunctionType 3", "stitching")
	assert.Contains(t, s, "/ShadingType 2", "axial")
}

// TestImage tests embedding images, which are embedded once each, with a
// soft mask for transparency.
func TestImage(t *testing.T) {
	opaque := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for i := range opaque.Pix {
		opaque.Pix[i] = 255
	}
	trans := image.NewRGBA(image.Rect(0, 0, 3, 3))
	trans.Set(1, 1, color.RGBA{128, 0, 0, 128})
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		pd.Image(opaque, math32.Translate2D(10, 10))
		pd.Image(opaque, math32.Translate2D(20, 10).Mul(math32.Rotate2D(0.5)))
		pd.Image(trans, math32.Identity2())
	})
	assert.Equal(t, 3, strings.Count(s, "/Subtype/Image"), "two images, each embedded once, and one mask")
	assert.Contains(t, s, "/Width 4")
	assert.Contains(t, s, "/Height 2")
	assert.Contains(t, s, "/SMask", "the transparent image has a mask")
	assert.Equal(t, 1, strings.Count(s, "/ColorSpace/DeviceGray"), "one mask")
	assert.Contains(t, s, "/Im0 Do")
	assert.Contains(t, s, "/Im2 Do")
	assert.Contains(t, s, " re W n", "clipped")
}

// TestLinksAnchors tests link annotations to URLs and anchors.
func TestLinksAnchors(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		pd.AddAnchor("top", math32.Vec2(0, 10))
		pd.AddLink("https://example.com", math32.B2(10, 10, 50, 20))
		pd.AddLink("#top", math32.B2(10, 30, 50, 40))
	})
	assert.Contains(t, s, "/URI(https://example.com)")
	assert.Contains(t, s, "/Subtype/Link")
	assert.Contains(t, s, "/Dest(top)")
	assert.Contains(t, s, "/Dests")
	assert.Contains(t, s, "/XYZ 0 90 0", "the anchor in page coordinates, from the bottom")
	assert.Contains(t, s, "/Rect[10 60 50 70]", "the link in page coordinates, from the bottom")
}

// TestLayerPane tests layers, hidden ones, and opening the layer pane.
func TestLayerPane(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		vis := pd.AddLayer("visible", true)
		hid := pd.AddLayer("hidden", false)
		pd.w.pdf.OpenLayerPane()
		strokeStyle(sty, 1)
		pd.BeginLayer(vis)
		pd.Path(*ppath.New().Line(0, 0, 10, 10), sty, math32.Identity2())
		pd.EndLayer()
		pd.BeginLayer(hid)
		pd.Path(*ppath.New().Line(0, 0, 10, 10), sty, math32.Identity2())
		pd.EndLayer()
	})
	assert.Contains(t, s, "/PageMode/UseOC")
	assert.Contains(t, s, "/OCProperties")
	assert.Equal(t, 2, strings.Count(s, " BDC"))
	assert.Equal(t, 2, strings.Count(s, " EMC"))
	assert.Contains(t, s, "/OFF[")
}

// TestOutlineDict tests that the outline dictionary has the right type.
func TestOutlineDict(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		pd.w.AddOutline("One", 1, 10)
		pd.w.AddOutline("Two", 2, 30)
	})
	assert.Contains(t, s, "/Type/Outlines")
	assert.NotContains(t, s, "/Type:")
	assert.Contains(t, s, "/Title(One)")
}

// TestCompression tests compressed content streams.
func TestCompression(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		pd.w.pdf.SetCompression(true)
		pd.w.pdf.SetFontSubsetting(false)
		strokeStyle(sty, 1)
		pd.Path(*ppath.New().Line(0, 0, 10, 10), sty, math32.Identity2())
	})
	assert.Contains(t, s, "/Filter/FlateDecode")
}

// TestTextStyles tests text with a stroke color, a background, and bold
// and italic standard fonts.
func TestTextStyles(t *testing.T) {
	s := renderPDF(t, func(pd *PDF, sty *styles.Paint) {
		prv := UseStandardFonts()
		defer RestorePreviousFonts(prv)
		sh := shaped.NewShaper()
		rsty := &sty.Font
		var tx rich.Text
		stroke := rsty.Clone().SetStrokeColor(colors.Blue)
		tx.AddSpan(stroke, []rune("outlined "))
		bg := rsty.Clone().SetBackground(colors.Yellow)
		tx.AddSpan(bg, []rune("marked "))
		bi := rsty.Clone().SetWeight(rich.Bold).SetSlant(rich.Italic)
		tx.AddSpan(bi, []rune("bold italic "))
		serif := rsty.Clone().SetFamily(rich.Serif).SetSlant(rich.Italic)
		tx.AddSpan(serif, []rune("serif"))
		lns := sh.WrapLines(tx, rsty, &sty.Text, math32.Vec2(300, 100))
		pd.Text(sty, math32.Identity2(), math32.Vec2(5, 5), lns)
	})
	assert.Contains(t, s, " BT")
	assert.Contains(t, s, " ET")
	assert.Contains(t, s, " Tr", "a render mode for the stroke")
	assert.Contains(t, s, "/BaseFont/Helvetica-BoldOblique")
	assert.Contains(t, s, "/BaseFont/Times-Italic")
	assert.Contains(t, s, "1 1 0 rg", "the yellow background")
}

// TestStandardFontNames tests the names of the standard fonts.
func TestStandardFontNames(t *testing.T) {
	names := map[string]*rich.Style{
		"Helvetica":             rich.NewStyle(),
		"Times-Roman":           rich.NewStyle().SetFamily(rich.Serif),
		"Times-BoldItalic":      rich.NewStyle().SetFamily(rich.Serif).SetWeight(rich.Bold).SetSlant(rich.Italic),
		"Courier-Bold":          rich.NewStyle().SetFamily(rich.Monospace).SetWeight(rich.Bold),
		"ZapfChancery-Oblique":  rich.NewStyle().SetFamily(rich.Cursive).SetSlant(rich.Italic),
		"Symbol":                rich.NewStyle().SetFamily(rich.Math),
		"ZapfDingbats":          rich.NewStyle().SetFamily(rich.Emoji),
		"Helvetica-BoldOblique": rich.NewStyle().SetWeight(rich.Bold).SetSlant(rich.Italic),
	}
	for name, sty := range names {
		assert.Equal(t, name, standardFontName(sty))
	}
}

// TestEncoding tests the helpers that encode values.
func TestEncoding(t *testing.T) {
	assert.Equal(t, "\xfe\xff\x00a\x00\xe9", utf8toutf16("aé"))
	assert.Equal(t, "\x00a", utf8toutf16("a", false))
	assert.Equal(t, "\xfe\xff\x20\xac", utf8toutf16("€"))
	assert.Equal(t, "a\\(b\\)\\\\", escape(`a(b)\`))
	assert.Equal(t, "1.5", dec(1.5).String())
	assert.Equal(t, "3000000000.0", dec(3e9).String())
}
