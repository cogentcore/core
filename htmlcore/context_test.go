// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	"github.com/andybalholm/cascadia"
	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestInlineContainer(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadHTMLString(NewContext(), b, `<button>Test</button>`))
	if tag := b.Child(0).AsTree().Property("tag"); tag != "body" {
		t.Errorf("expected first child to be body but got %v", tag)
	}
	if !strings.Contains(b.Child(0).AsTree().Child(0).AsTree().Name, "inline") {
		t.Errorf("expected inline container for h1 but got %v", b.Child(0))
	}
}

func TestNoInlineContainer(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadHTMLString(NewContext(), b, `<h1>Test</h1>`))
	if tag := b.Child(0).AsTree().Property("tag"); tag != "body" {
		t.Errorf("expected first child to be body but got %v", tag)
	}
	if strings.Contains(b.Child(0).AsTree().Child(0).AsTree().Name, "inline") {
		t.Errorf("expected no inline container for h1 but got %v", b.Child(0))
	}
}

// TestSelectNodes tests the selectors of the user agent styles,
// and others that need to be rewritten for the selector library.
func TestSelectNodes(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<html><body><h1>Title</h1><p class="a">Text <b>bold</b></p><ul><li>one</li><li>two</li></ul><div><table><tr><td>cell</td><th>head</th></tr></table></div><ruby>x<rt>y</rt><rp>z</rp></ruby><svg><p>svg p</p></svg><center>c</center><a href="#">link</a></body></html>`))
	assert.NoError(t, err)
	ss, err := parser.Parse(browserStyles + "\nH1, p { color: red; }\ndiv, *, p { color: blue; }\np, p { color: green; }")
	assert.NoError(t, err)
	for _, rule := range ss.Rules {
		for _, sel := range rule.Selectors {
			sel = strings.TrimSpace(sel)
			want := []*html.Node{}
			if s, err := cascadia.Parse(sel); err == nil {
				want = cascadia.QueryAll(doc, s)
			}
			assert.ElementsMatch(t, want, selectNodes(doc, sel), sel)
		}
	}
	count := map[string]int{
		":root": 1, "p:not(.a)": 1, "p:is(.a)": 1, "li:where(:first-child, :last-child)": 2,
		"a:hover": 0, "a:not(:focus)": 1, "a:not(:focus-visible)": 1, "div:focus-within": 0,
		"p::before": 0, "body:has(.a)": 1, ":is(h1, ul) :is(li)": 2,
	}
	for sel, n := range count {
		assert.Len(t, selectNodes(doc, sel), n, sel)
	}
	assert.Equal(t, "html", selectNodes(doc, ":root")[0].Data)
}

func TestSpecificity(t *testing.T) {
	tests := map[string]int{
		"p": 1, "*": 0, "div p": 2, ".a": 100, "p.a.b": 201, "#x": 10000, "#x .a p": 10101,
		"div > p.direct": 102, "h2 + p": 2, "a:hover": 101, "p::before": 2, "[type=text]": 100,
	}
	for sel, want := range tests {
		assert.Equal(t, want, specificity(sel), sel)
	}
}

// TestCSSVariables tests the substitution of custom properties, with
// inheritance, fallbacks, and cycles, which are invalid.
func TestCSSVariables(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<html><head><style>
:root { --main: red; --size: 10px; --a: var(--b); --b: var(--a); --self: var(--self, blue) }
div { --size: 20px; color: var(--main); width: var(--size); height: var(--none, var(--size)) }
p { color: var(--a, green); background-color: var(--self) }
</style></head><body><div><p>x</p></div></body></html>`))
	assert.NoError(t, err)
	ctx := NewContext()
	ctx.Node = doc
	ctx.addStyleSheets(doc)
	var div, p *html.Node
	var find func(n *html.Node)
	find = func(n *html.Node) {
		switch n.Data {
		case "div":
			div = n
		case "p":
			p = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	assert.Equal(t, "red", ctx.declaration(div, "color"))
	assert.Equal(t, "20px", ctx.declaration(div, "width"))
	assert.Equal(t, "20px", ctx.declaration(div, "height"))
	assert.Equal(t, "green", ctx.declaration(p, "color"))
	assert.Equal(t, "", ctx.declaration(p, "background-color")) // a cycle is invalid, even with a fallback
}

func TestCSSColorFunction(t *testing.T) {
	tests := map[string]color.RGBA{
		"rgb(255, 0, 0)":              {255, 0, 0, 255},
		"rgba(0, 0, 255, 0.2)":        colors.FromNRGBA(0, 0, 255, 51),
		"rgb(0 0 255 / 20%)":          colors.FromNRGBA(0, 0, 255, 51),
		"rgb(100%, 50%, 0%)":          {255, 128, 0, 255},
		"hsl(120, 100%, 50%)":         {0, 255, 0, 255},
		"hsla(240deg 100% 50% / .5)":  colors.FromNRGBA(0, 0, 255, 128),
		"color(srgb 1 0 0)":           {255, 0, 0, 255},
		"color(display-p3 1 1 1)":     {255, 255, 255, 255},
		"oklch(1 0 0)":                {255, 255, 255, 255},
		"oklab(0 0 0 / 50%)":          colors.FromNRGBA(0, 0, 0, 128),
		"light-dark(#fff, #000)":      {255, 255, 255, 255},
		"light-dark(rgb(1 2 3), red)": {1, 2, 3, 255},
	}
	for v, want := range tests {
		got, ok := cssColorFunction(v)
		assert.True(t, ok, v)
		assert.Equal(t, want, got, v)
	}
}

func TestImportURL(t *testing.T) {
	for prelude, want := range map[string][2]string{
		`url("basic.css")`:         {"basic.css", ""},
		`url(a/b.css) screen`:      {"a/b.css", "screen"},
		`"print.css" print`:        {"print.css", "print"},
		`'x.css'`:                  {"x.css", ""},
		`url("y.css") layer(base)`: {"y.css", ""},
	} {
		u, m := importURL(prelude)
		assert.Equal(t, want, [2]string{u, m}, prelude)
	}
}

func TestSplitSelectors(t *testing.T) {
	assert.Equal(t, []string{"a", ".b > :is(p, table)", `[x="1,2"]`, "c:not(.d,.e)"},
		splitSelectors(`a, .b > :is(p, table) ,[x="1,2"],c:not(.d,.e)`))
}

func TestCalcPx(t *testing.T) {
	for v, want := range map[string]float32{
		"640px": 640, "40em": 640, "calc(640px - 1px)": 639, "calc((100px + 20px) * 2)": 240, "calc(1rem / 2)": 8,
	} {
		got, ok := calcPx(v)
		assert.True(t, ok, v)
		assert.Equal(t, want, got, v)
	}
	_, ok := calcPx("calc(100% - 1px)")
	assert.False(t, ok)
}

func TestMathLength(t *testing.T) {
	env := defaultLengthEnv() // an em of 16px, vw of 8px, and vh of 6px
	for v, want := range map[string]units.Value{
		"calc(10px + 2em)":         units.Dp(42),
		"calc(100vh - 100px)":      units.Dp(500),
		"min(25vw, 350px)":         units.Dp(200),
		"max(10px, 1rem)":          units.Dp(16),
		"clamp(1rem, 5vw, 2rem)":   units.Dp(32),
		"calc((100px - 20px) / 2)": units.Dp(40),
		"calc(2 * (10px + 5px))":   units.Dp(30),
	} {
		got, ok := env.length(v, units.UnitPw)
		assert.True(t, ok, v)
		assert.InDelta(t, want.Value, got.Value, 0.01, v)
		assert.Equal(t, want.Unit, got.Unit, v)
	}
	// a percentage with a length is computed with the size of the parent
	// (custom), in the layout
	uc := &units.Context{}
	uc.Defaults()
	uc.Paw = 1000
	dp := uc.Dots(units.UnitDp)
	for v, want := range map[string]float32{
		"calc(100% - 20px)":        1000 - 20*dp,
		"min(100%, 600px)":         600 * dp,
		"min(100%, 2000px)":        1000,
		"max(50%, 600px)":          600 * dp,
		"clamp(100px, 50%, 400px)": 400 * dp,
		"calc(100% - 2 * 128px)":   1000 - 256*dp,
		"calc(50% + 10px)":         500 + 10*dp,
		// with vw of 8px: clamp(20px, (800 - 768) / 4 = 8px, 128px) = 20px
		"calc(100% - 2 * clamp(1.25rem, calc((100vw - 768px) / 4), 8rem))": 1000 - 40*dp,
	} {
		got, ok := env.length(v, units.UnitPw)
		if assert.True(t, ok, v) && assert.NotNil(t, got.Custom, v) {
			assert.InDelta(t, want, got.Custom(uc), 0.01, v)
		}
	}
}

func TestMediaHeightVh(t *testing.T) {
	env := defaultLengthEnv()
	env.vh, env.fixedVh = 6, true // a 600 px viewport
	l, ok := env.length("50vh", units.UnitPh)
	assert.True(t, ok)
	assert.Equal(t, units.Dp(300), l)
	l, ok = env.length("calc(100vh - 100px)", units.UnitPh)
	assert.True(t, ok)
	assert.Equal(t, units.Dp(500), l)
}

func TestSVGSize(t *testing.T) {
	for _, c := range []struct {
		src  string
		w, h float32
	}{
		{`<svg width="16" height="16" viewBox="0 0 16 16">`, 16, 16},
		{`<?xml version="1.0"?><svg xmlns="x" viewBox="0 0 200 100">`, 200, 100},
		{`<svg width="50px" viewBox="0 0 200 100">`, 50, 25},
	} {
		w, h, ok := svgSize([]byte(c.src))
		assert.True(t, ok, c.src)
		assert.InDelta(t, c.w, w, 0.01, c.src)
		assert.InDelta(t, c.h, h, 0.01, c.src)
	}
	_, _, ok := svgSize([]byte(`<svg>`))
	assert.False(t, ok)
}

// TestSVGMaxWidth tests that an img of an svg without a width or height
// is scaled down to an absolute max-width, keeping its aspect, as for icons
// that are much larger than their max-width (the Rust snapshot).
func TestSVGMaxWidth(t *testing.T) {
	dir := t.TempDir()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="256" viewBox="0 0 512 256"><rect width="512" height="256"/></svg>`
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "icon.svg"), []byte(svg), 0666))
	b := core.NewBody()
	ctx := NewContext()
	ctx.PageURL = "file://" + dir + "/page.html"
	ctx.GetURL = func(u string) (*http.Response, error) {
		f, err := os.Open(filepath.Join(dir, filepath.Base(u)))
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: f}, nil
	}
	assert.NoError(t, ReadHTMLString(ctx, b, `<html><body><img src="icon.svg" style="max-width: 6rem"></body></html>`))
	var sv *core.SVG
	b.WalkDown(func(n tree.Node) bool {
		if s, ok := n.(*core.SVG); ok {
			sv = s
		}
		return tree.Continue
	})
	if !assert.NotNil(t, sv) {
		return
	}
	b.RenderImage(func() {
		dp := sv.Styles.UnitContext.Dots(units.UnitDp)
		sz := sv.Geom.Size.Actual.Content
		assert.InDelta(t, 96*dp, sz.X, 2, "it is as wide as its max-width")
		assert.InDelta(t, 48*dp, sz.Y, 2, "and keeps its aspect")
	})
}

// TestInlineBlocksExactFit tests inline blocks whose widths add up exactly
// to the width of their container (as the sister project links of
// Wikipedia: a logo of 31px and a link of 7px + 182px in 220px), which are
// on one line, although their sizes are each rounded up to whole dots, and
// which are aligned in it by their vertical-align: middle.
func TestInlineBlocksExactFit(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(400) })
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, b, `<html><body><ul style="width: 220px; padding: 0; margin: 0; list-style: none"><li>`+
		`<span class="logo" style="display: inline-block; width: 31px; height: 31px; vertical-align: middle; background: gray"></span>`+
		`<span class="link" style="display: inline-block; margin-left: 7px; width: 182px; vertical-align: middle">Definitions from Wiktionary</span>`+
		`</li></ul></body></html>`))
	find := func(class string) *core.WidgetBase {
		var w *core.WidgetBase
		b.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil && fmt.Sprint(wb.Property("class")) == class {
				w = wb
			}
			return tree.Continue
		})
		return w
	}
	logo, link := find("logo"), find("link")
	if !assert.NotNil(t, logo) || !assert.NotNil(t, link) {
		return
	}
	b.RenderImage(func() {
		assert.Equal(t, logo.Geom.Cell.Y, link.Geom.Cell.Y, "they are on the same line")
		assert.Greater(t, link.Geom.Pos.Total.X, logo.Geom.Pos.Total.X, "the link is after the logo")
		lc := logo.Geom.Pos.Total.Y + logo.Geom.Size.Actual.Total.Y/2
		kc := link.Geom.Pos.Total.Y + link.Geom.Size.Actual.Total.Y/2
		assert.InDelta(t, lc, kc, 2, "the link is in the middle of the line")
	})
}

// TestColumnsOfList tests columns of an element with only one child, a
// list (as for the references of Wikipedia: a div with a column-width and
// an ol): the items of the list are split into the columns, as in CSS.
func TestColumnsOfList(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(600); s.Max.X.Dp(600) })
	ctx := NewContext()
	ctx.MediaWidth = 600
	items := ""
	for i := range 6 {
		items += fmt.Sprintf(`<li class="item%d">Item %d</li>`, i, i)
	}
	assert.NoError(t, ReadHTMLString(ctx, b, `<html><body style="margin: 0"><div style="column-width: 200px"><ol>`+items+`</ol></div></body></html>`))
	var first, last *core.WidgetBase
	b.WalkDown(func(n tree.Node) bool {
		if wb := core.AsWidget(n); wb != nil {
			switch fmt.Sprint(wb.Property("class")) {
			case "item0":
				first = wb
			case "item5":
				last = wb
			}
		}
		return tree.Continue
	})
	if !assert.NotNil(t, first) || !assert.NotNil(t, last) {
		return
	}
	b.RenderImage(func() {
		assert.Greater(t, last.Geom.Pos.Total.X, first.Geom.Pos.Total.X+100, "the last item is in a column to the right")
		assert.Less(t, last.Geom.Pos.Total.Y, first.Geom.Pos.Total.Y+100, "near the top")
	})
}

func TestMediaRange(t *testing.T) {
	c := NewContext()
	c.MediaWidth = 800
	c.MediaHeight = 600
	for q, want := range map[string]bool{
		"(width >= 50rem)":                          true,
		"(width >= 64rem)":                          false,
		"only screen and (width <= 52rem)":          true,
		"(30rem < width <= 60rem)":                  true,
		"(40rem < width)":                           true,
		"(height > 700px)":                          false,
		"(min-width: 600px)":                        true,
		"only screen and (max-device-width: 480px)": false,
		"(min-device-width: 600px)":                 true,
	} {
		assert.Equal(t, want, c.mediaMatches(q), q)
	}
}

// TestMediaReducedMotion tests that pages are told that motion is reduced,
// as animations are not run: content that fades in when animated (as the
// headings of github.com) is shown.
func TestMediaReducedMotion(t *testing.T) {
	c := NewContext()
	assert.True(t, c.mediaMatches("(prefers-reduced-motion: reduce)"))
	assert.True(t, c.mediaMatches("(prefers-reduced-motion)"))
	assert.False(t, c.mediaMatches("(prefers-reduced-motion: no-preference)"))
	widgetsByID(t, `<style>@media (prefers-reduced-motion: no-preference) { .heading > * { opacity: 0 } }</style>`+
		`<div class="heading"><h2 id="h">GitHub Platform</h2></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.Equal(t, float32(1), ws["h"].Styles.Opacity, "shown")
		})
}

func TestGridTracks(t *testing.T) {
	assert.Equal(t, []string{"minmax(15rem,1fr)", "2rem", "minmax(0,48rem)"},
		gridTracks("[full-start left-start] minmax(15rem,1fr) [left-end] 2rem [content-start] minmax(0,48rem) [full-end right-end]"))
	assert.Equal(t, []string{"1fr", "1fr", "1fr", "auto"}, gridTracks("repeat(3, 1fr) auto"))
}

// TestGridTemplateCascade tests that the grid-template shorthand and its
// longhands apply in the order of the cascade (as the page container of
// Wikipedia without a table of contents, whose more specific rule has
// grid-template-columns and grid-template-areas of one column, after a
// grid-template of two columns).
func TestGridTemplateCascade(t *testing.T) {
	ctx := NewContext()
	doc, _ := html.Parse(strings.NewReader(`<html><head><style>
.a { display: grid; grid-template: min-content 1fr / 12rem minmax(0,1fr); grid-template-areas: 'start content' }
.x .a { grid-template-columns: minmax(0,1fr); grid-template-areas: 'content' 'start' }
.b { grid-template-columns: 1fr 1fr 1fr; grid-template-areas: 'p q r' }
.x .b { grid-template: 'p' 'q' / 3rem }
</style></head><body class="x"><div class="a"></div><div class="b"></div></body></html>`))
	ctx.Node = doc
	ctx.addStyleSheets(doc)
	var a, b *html.Node
	var find func(n *html.Node)
	find = func(n *html.Node) {
		switch GetAttr(n, "class") {
		case "a":
			a = n
		case "b":
			b = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	areas, ncols, tracks := ctx.gridTemplate(a)
	assert.Equal(t, []string{"minmax(0,1fr)"}, tracks, "the longhand of the later rule")
	assert.Equal(t, 1, ncols)
	assert.Equal(t, gridArea{row: 1, col: 0, rows: 1, cols: 1}, areas["start"])
	areas, ncols, tracks = ctx.gridTemplate(b)
	assert.Equal(t, []string{"3rem"}, tracks, "the shorthand of the later rule")
	assert.Equal(t, 1, ncols)
	assert.Equal(t, gridArea{row: 1, col: 0, rows: 1, cols: 1}, areas["q"])
}

// TestCSSVariableInitial tests the space toggle of custom properties that
// are initial (as for light-dark() without native support), which are not set.
func TestCSSVariableInitial(t *testing.T) {
	ctx := NewContext()
	doc, _ := html.Parse(strings.NewReader(`<html><head><style>
:root { --light: initial; --toggle: var(--light) blue; --color: var(--toggle, red); }
p { color: var(--color); }
</style></head><body><p>x</p></body></html>`))
	ctx.Node = doc
	ctx.addStyleSheets(doc)
	var p *html.Node
	var find func(n *html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			p = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	assert.Equal(t, "red", strings.TrimSpace(ctx.declaration(p, "color")))
}

func TestRootFontSize(t *testing.T) {
	ctx := NewContext()
	doc, _ := html.Parse(strings.NewReader(`<html><head><style>html { font-size: 62.5% }</style></head><body></body></html>`))
	ctx.Node = doc
	ctx.addStyleSheets(doc)
	assert.InDelta(t, 10, ctx.rootFontSize(), 0.01)
	env := ctx.readEnv()
	assert.Equal(t, 10.0, env.rem)
	l, ok := env.length("2rem", units.UnitPw)
	assert.True(t, ok)
	assert.Equal(t, units.Dp(20), l)
	l, ok = env.length("calc(1rem + 4px)", units.UnitPw)
	assert.True(t, ok)
	assert.Equal(t, units.Dp(14), l)
}

func TestGetFileURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.html")
	assert.NoError(t, os.WriteFile(path, []byte("<p>local</p>"), 0666))
	resp, err := getURL("file://" + filepath.ToSlash(path))
	if assert.NoError(t, err) {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "<p>local</p>", string(b))
	}
	resp, err = getURL("file://" + filepath.ToSlash(filepath.Join(dir, "missing.html")))
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	}
}

func TestICO(t *testing.T) {
	// an ICO file with one 2x1 PNG image
	var pngBuf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	assert.NoError(t, png.Encode(&pngBuf, img))
	var ico bytes.Buffer
	ico.Write([]byte{0, 0, 1, 0, 1, 0})        // header: icon, 1 image
	ico.Write([]byte{2, 1, 0, 0, 1, 0, 32, 0}) // 2x1, planes, bpp
	binary.Write(&ico, binary.LittleEndian, uint32(pngBuf.Len()))
	binary.Write(&ico, binary.LittleEndian, uint32(22)) // offset
	ico.Write(pngBuf.Bytes())
	im, format, err := image.Decode(bytes.NewReader(ico.Bytes()))
	if assert.NoError(t, err) {
		assert.Equal(t, "ico", format)
		assert.Equal(t, image.Rect(0, 0, 2, 1), im.Bounds())
	}
	assert.True(t, looksLikeSVG([]byte(`<?xml version="1.0"?>\n<svg xmlns="x"></svg>`)))
	assert.False(t, looksLikeSVG(pngBuf.Bytes()))
}

// TestFontShorthandResets tests that the font shorthand resets the
// properties that it does not set, as the bold of a heading.
func TestFontShorthandResets(t *testing.T) {
	st := &cssState{env: defaultLengthEnv()}
	s := styles.NewStyle()
	s.Font.Weight = rich.Bold
	s.Font.Slant = rich.Italic
	setFont(st, s, nil, "100% sans-serif")
	assert.Equal(t, rich.Normal, s.Font.Weight)
	assert.Equal(t, rich.SlantNormal, s.Font.Slant)
	assert.Equal(t, "normal", st.lineHeight)
	setFont(st, s, nil, "italic bold 12px/1.5 serif")
	assert.Equal(t, rich.Bold, s.Font.Weight)
	assert.Equal(t, rich.Italic, s.Font.Slant)
	assert.Equal(t, "1.5", st.lineHeight)
	s.Font.Weight = rich.Bold
	setFont(st, s, nil, "inherit") // not a valid shorthand of values
	assert.Equal(t, rich.Bold, s.Font.Weight)
}

// TestMediaChanged tests that a page reports a change of its media only
// when the width crosses a boundary of one of its media queries or of
// its estimated number of columns.
func TestMediaChanged(t *testing.T) {
	read := func(page string) *Context {
		ctx := NewContext()
		ctx.MediaWidth = 800
		assert.NoError(t, ReadHTMLString(ctx, core.NewBody(), page))
		return ctx
	}
	ctx := read(`<html><head><style>p { color: red }
@media (max-width: 600px) { p { color: blue } }</style></head><body><p>Text</p></body></html>`)
	assert.False(t, ctx.MediaChanged(700), "within the same range")
	assert.False(t, ctx.MediaChanged(1200), "within the same range")
	assert.True(t, ctx.MediaChanged(500), "across the max-width")
	assert.Equal(t, float32(800), ctx.MediaWidth, "the width of the context is kept")

	ctx = read(`<html><head><link rel="stylesheet" media="(min-width: 1000px)" href="data:text/css,p{color:red}"></head><body><p>Text</p></body></html>`)
	assert.False(t, ctx.MediaChanged(900))
	assert.True(t, ctx.MediaChanged(1100), "across the media of a style sheet")

	ctx = read(`<html><body style="margin: 0"><div style="column-width: 300px"><p>A</p><p>B</p><p>C</p></div></body></html>`)
	assert.False(t, ctx.MediaChanged(850), "2 columns either way")
	assert.True(t, ctx.MediaChanged(1000), "3 columns")
	assert.True(t, ctx.MediaChanged(500), "1 column")

	ctx = read(`<html><body><p style="width: 50vw">Text</p></body></html>`)
	assert.False(t, ctx.MediaChanged(300), "no media queries or columns")
}

// TestTableCellWidthMinimum tests that the width of a table cell is not
// a limit, as in CSS: a cell with a width of 1% and text that does not
// wrap (as the group headers of the navigation boxes of Wikipedia) is as
// wide as its text, which does not overflow it.
func TestTableCellWidthMinimum(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(600) })
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, b, `<html><body><table style="width: 100%"><tr>`+
		`<th class="group" style="width: 1%; white-space: nowrap">Cascading style sheets</th>`+
		`<td>Some text in the other column of the table</td></tr></table></body></html>`))
	var th *core.WidgetBase
	var tx *core.Text
	b.WalkDown(func(n tree.Node) bool {
		if wb := core.AsWidget(n); wb != nil && fmt.Sprint(wb.Property("class")) == "group" {
			th = wb
			wb.WalkDown(func(n tree.Node) bool {
				if t, ok := n.(*core.Text); ok {
					tx = t
				}
				return tree.Continue
			})
		}
		return tree.Continue
	})
	if !assert.NotNil(t, th) || !assert.NotNil(t, tx) {
		return
	}
	b.RenderImage(func() {
		right := th.Geom.Pos.Total.X + th.Geom.Size.Actual.Total.X
		assert.LessOrEqual(t, tx.Geom.Pos.Total.X+tx.Geom.Size.Actual.Total.X, right+1, "the text is within its cell")
		assert.Greater(t, th.Geom.Size.Actual.Total.X, float32(100), "the cell is as wide as its text")
	})
}

// TestLayoutSettles tests that the layout of a page in a frame that
// scrolls (as in a browser) settles, so that the content of the article of
// the Wikipedia snapshot (in a grid of columns, in which the heights of the
// text follow widths that change in each iteration of the layout) is not
// followed by blank space down to the footer.
func TestLayoutSettles(t *testing.T) {
	file := filepath.Join(snapshotsDir, "wikipedia", "index.html")
	src, err := os.ReadFile(file)
	if err != nil {
		t.Skip("no Wikipedia snapshot")
	}
	dir, _ := filepath.Abs(filepath.Dir(file))
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(800))
		s.Padding.Zero()
	})
	scroller := core.NewFrame(b)
	scroller.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
		s.Padding.Zero()
	})
	ctx := NewContext()
	ctx.MediaWidth = 1200
	ctx.PageURL = "file://" + dir + "/index.html"
	ctx.GetURL = func(u string) (*http.Response, error) {
		pu, err := url.Parse(u)
		if err != nil || pu.Scheme != "file" {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return getURL(u)
	}
	assert.NoError(t, ReadHTMLString(ctx, scroller, string(src)))
	b.RenderImage(func() {
		var text, output *core.WidgetBase
		scroller.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			switch {
			case cwb.Name == "mw-content-text":
				text = cwb
			case text != nil && output == nil && strings.Contains(fmt.Sprint(cwb.Property("class")), "mw-parser-output"):
				output = cwb
			}
			return tree.Continue
		})
		if !assert.NotNil(t, text) || !assert.NotNil(t, output) {
			return
		}
		blank := (text.Geom.Pos.Total.Y + text.Geom.Size.Actual.Total.Y) - (output.Geom.Pos.Total.Y + output.Geom.Size.Actual.Total.Y)
		assert.Less(t, blank, float32(100), "no blank space after the content of the article")
	})
}

// TestReadDocument tests reading a parsed document, changing it (as the
// scripts of a page do), and reading it again with a new context, and
// that the widgets of its elements have their nodes (NodeOf).
func TestReadDocument(t *testing.T) {
	ctx := NewContext()
	doc, err := ParseHTML(ctx, strings.NewReader(`<html><body><p id="p">First</p><button id="b"><span>Go</span></button></body></html>`))
	if !assert.NoError(t, err) {
		return
	}
	texts := func(b *core.Body) []string {
		var ts []string
		b.WalkDown(func(n tree.Node) bool {
			if tx, ok := n.(*core.Text); ok {
				ts = append(ts, tx.Text)
			}
			return tree.Continue
		})
		return ts
	}
	b := core.NewBody()
	assert.NoError(t, ReadDocument(ctx, b, doc))
	assert.Contains(t, texts(b), "First")
	var button *core.Text
	b.WalkDown(func(n tree.Node) bool {
		if tx, ok := n.(*core.Text); ok && strings.Contains(tx.Text, "Go") {
			button = tx
		}
		return tree.Continue
	})
	if assert.NotNil(t, button) {
		// the text of a run of inline content is in the frame of its
		// element, which has the node
		n := NodeOf(button)
		if assert.NotNil(t, n) {
			assert.Equal(t, "button", n.Data)
		}
	}

	// change the text of the paragraph, as a script does, and read it again
	var p *html.Node
	var find func(n *html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && GetAttr(n, "id") == "p" {
			p = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	if !assert.NotNil(t, p) {
		return
	}
	p.FirstChild.Data = "Second"
	b2 := core.NewBody()
	assert.NoError(t, ReadDocument(ctx.NewPage(), b2, doc))
	assert.Contains(t, texts(b2), "Second")
	assert.NotContains(t, texts(b2), "First")
}

// TestScriptingNoscript tests that the content of a noscript element is
// shown without scripting (as htmlcore reads pages by default) and not
// with it, as in browsers with and without JavaScript.
func TestScriptingNoscript(t *testing.T) {
	src := `<html><body><p>Page</p><noscript><p>Enable JavaScript</p></noscript></body></html>`
	for _, scripting := range []bool{false, true} {
		ctx := NewContext()
		ctx.Scripting = scripting
		b := core.NewBody()
		assert.NoError(t, ReadHTMLString(ctx, b, src))
		found := false
		b.WalkDown(func(n tree.Node) bool {
			if tx, ok := n.(*core.Text); ok && strings.Contains(tx.Text, "Enable JavaScript") {
				found = true
			}
			return tree.Continue
		})
		assert.Equal(t, !scripting, found, "noscript content shown, with scripting %v", scripting)
		assert.Equal(t, scripting, ctx.NewPage().Scripting, "NewPage keeps Scripting")
	}
	ctx := NewContext()
	ctx.MediaWidth = 800
	assert.True(t, ctx.MediaMatches("(min-width: 600px)"))
	assert.False(t, ctx.MediaMatches("(max-width: 600px)"))
}

// TestLinkedStyleNotCSS tests that a link of a style sheet with an empty
// href, which would be the page itself, and a style sheet that is served
// as HTML are not added, as in browsers (the CSS parser does not end on
// a page of HTML, as for a page of GitHub).
func TestLinkedStyleNotCSS(t *testing.T) {
	ctx := NewContext()
	ctx.PageURL = "https://example.com/page"
	var gets []string
	ctx.GetURL = func(u string) (*http.Response, error) {
		gets = append(gets, u)
		pu, _ := url.Parse(u)
		ct := "text/css"
		if strings.HasSuffix(u, ".html") {
			ct = "text/html; charset=utf-8"
		}
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{"Content-Type": {ct}},
			Body: io.NopCloser(strings.NewReader(`p { color: rgb(255, 0, 0) }`)), Request: &http.Request{URL: pu}}, nil
	}
	b := core.NewBody()
	assert.NoError(t, ReadHTMLString(ctx, b, `<html><head><link rel="stylesheet" href=""><link rel="stylesheet" href="/style.html"></head><body><p>Text</p></body></html>`))
	assert.Equal(t, []string{"https://example.com/style.html"}, gets, "no request for the empty href")
	assert.Empty(t, ctx.declaration(findTag(b, "p"), "color"), "the style sheet served as HTML is not added")
}

// findTag returns the node of the first widget for an element of the given tag.
func findTag(b *core.Body, tag string) *html.Node {
	var found *html.Node
	b.WalkDown(func(n tree.Node) bool {
		if wb := core.AsWidget(n); wb != nil && found == nil {
			if nd, ok := wb.Property("node").(*html.Node); ok && nd.Data == tag {
				found = nd
			}
		}
		return tree.Continue
	})
	return found
}

// TestUnset tests that unset is inherit for an inherited property (as
// color) and initial for one that is not, as in CSS.
func TestUnset(t *testing.T) {
	ctx := NewContext()
	b := core.NewBody()
	assert.NoError(t, ReadHTMLString(ctx, b, `<html><body style="color: rgb(0, 128, 0)"><p style="color: unset; padding: 10px; padding: unset">Text</p></body></html>`))
	b.RenderImage(func() {
		var p *core.WidgetBase
		b.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil && wb.Property("tag") == "p" && p == nil {
				p = wb
			}
			return tree.Continue
		})
		if assert.NotNil(t, p) {
			assert.Equal(t, "#008000", colors.AsHex(colors.ToUniform(p.Styles.Color)), "the color is inherited")
			assert.Zero(t, p.Styles.Padding.Top.Value, "the padding is initial")
		}
	})
}

// TestStrayCSS tests that style sheets with semicolons and blocks where
// rules are expected (as CSS-in-JS on the Guardian makes) are parsed
// without them, instead of the parser never ending.
func TestStrayCSS(t *testing.T) {
	style := `a{color:red} ; {x:y} b;c{color:blue} /* {;} */ @import "s;{.css";
		@media screen{ ;d{color:green;} {e:f} } {flex-direction:column;}`
	done := make(chan *css.Stylesheet)
	go func() { done <- parseCSS(style) }()
	select {
	case ss := <-done:
		var rules []string
		for _, r := range ss.Rules {
			rules = append(rules, strings.Join(r.Selectors, ",")+r.Prelude)
			for _, e := range r.Rules {
				rules = append(rules, " "+strings.Join(e.Selectors, ","))
			}
		}
		assert.Equal(t, []string{"aa", "cc", "\"s;{.css\"", "screen", " d"}, rules)
	case <-time.After(5 * time.Second):
		t.Fatal("parsing did not end")
	}
	assert.Equal(t, "a{b:c}@media x{d{e:f}}", removeStrayCSS("a{b:c};@media x{;d{e:f}{g:h}}"))
}

// TestFloatColumns tests floats with a percentage width, as the columns
// of craigslist.org (float: left; width: 23%), and a sidebar beside a main
// column, as on slashdot.org (a float with a width of 100% and a right
// margin inside it, and a float after it pulled back beside it by a
// negative left margin).
func TestFloatColumns(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Padding.Zero(); s.Min.X.Dp(600); s.Max.X.Dp(600) })
	ctx := NewContext()
	ctx.MediaWidth = 600
	assert.NoError(t, ReadHTMLString(ctx, b, `<html><body>
<div style="overflow: hidden"><div class="c1" style="float: left; width: 23%; overflow: hidden">A<br>a<br>a</div>`+
		`<div class="c2" style="float: left; width: 23%; overflow: hidden">B<br>b</div></div>
<p class="after" style="margin: 0">after</p>
<div><div style="float: left; width: 100%"><div class="main" style="margin-right: 200px">main</div></div>`+
		`<aside class="rail" style="float: left; width: 200px; margin-left: -200px; overflow: hidden">rail</aside></div>
</body></html>`))
	find := func(class string) *core.WidgetBase {
		var w *core.WidgetBase
		b.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil && fmt.Sprint(wb.Property("class")) == class {
				w = wb
			}
			return tree.Continue
		})
		return w
	}
	c1, c2, after, main, rail := find("c1"), find("c2"), find("after"), find("main"), find("rail")
	if !assert.NotNil(t, c1) || !assert.NotNil(t, c2) || !assert.NotNil(t, after) || !assert.NotNil(t, main) || !assert.NotNil(t, rail) {
		return
	}
	b.RenderImage(func() {
		w := core.AsWidget(c1.Parent).Geom.Size.Actual.Content.X
		assert.InDelta(t, 0.23*w, c1.Geom.Size.Actual.Total.X, 2, "a column is 23% wide")
		assert.InDelta(t, c1.Geom.Pos.Total.Y, c2.Geom.Pos.Total.Y, 1, "the columns are beside each other")
		assert.InDelta(t, c1.Geom.Pos.Total.Y+c1.Geom.Size.Actual.Total.Y, after.Geom.Pos.Total.Y, 1, "what is after them is below the tallest")
		assert.InDelta(t, main.Geom.Pos.Total.Y, rail.Geom.Pos.Total.Y, 1, "the rail is beside the main column")
		rp := core.AsWidget(rail.Parent).Geom
		assert.InDelta(t, rp.Pos.Content.X+rp.Size.Actual.Content.X, rail.Geom.Pos.Content.X+rail.Geom.Size.Actual.Content.X, 2, "at the right")
		assert.False(t, rail.Geom.TotalBBox.Empty(), "and visible")
	})
}

// TestBlockLinkClick tests that clicking on the text of links that are
// blocks (as the links in the columns of craigslist.org, which have
// display: block) opens them: the text, which can be pressed (to select
// it), gets the click instead of the frame of the link.
func TestBlockLinkClick(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(1000) })
	ctx := NewContext()
	var opened []string
	ctx.OpenURL = func(url string, e events.Event) { opened = append(opened, url) }
	assert.NoError(t, ReadHTMLString(ctx, b, `<style>.box{overflow:hidden;float:left;width:23%}.box a{display:block}</style>
<div style="overflow:hidden"><div class="box"><ul><li><a href="/auburn">auburn</a></li><li><a href="/bham">birmingham</a></li></ul></div>
<div class="box"><ul><li><a href="/bn">bloomington</a></li></ul></div></div>
<a href="/block" style="display:block">block link</a>`))
	var links []*core.WidgetBase
	b.WalkDown(func(n tree.Node) bool {
		if wb := core.AsWidget(n); wb != nil && wb.Property("href") != nil {
			links = append(links, wb)
		}
		return tree.Continue
	})
	if !assert.Len(t, links, 4) {
		return
	}
	b.RenderImage(func() {
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		for _, l := range links {
			bb := l.Geom.TotalBBox
			pos := image.Pt(bb.Min.X+5, (bb.Min.Y+bb.Max.Y)/2) // on its text
			ev.MouseButton(events.MouseDown, events.Left, pos, 0)
			ev.MouseButton(events.MouseUp, events.Left, pos, 0)
		}
	}, func() {
		assert.Equal(t, []string{"/auburn", "/bham", "/bn", "/block"}, opened)
	})
}

// TestSVGVars tests that the var() functions in the attributes of an svg
// element (as in the icons of BBC News) are replaced by the values of the
// custom properties for its elements, which the SVG reader does not know.
func TestSVGVars(t *testing.T) {
	ctx := NewContext()
	b := core.NewBody()
	doc, err := ParseHTML(ctx, strings.NewReader(`<div style="--fg: #ff0000; --bg: blue"><svg viewBox="0 0 10 10" width="10" height="10">`+
		`<rect class="a" width="5" height="5" fill="var(--fg)"/>`+
		`<rect class="b" width="5" height="5" style="fill: var(--bg); stroke: var(--none)"/>`+
		`<rect class="c" width="5" height="5" fill="var(--none, green)" stroke="var(--none)"/></svg></div>`))
	assert.NoError(t, err)
	assert.NoError(t, ReadDocument(ctx, b, doc))
	var svg *html.Node
	walkNodes(doc, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "svg" {
			svg = n
		}
	})
	if !assert.NotNil(t, svg) {
		return
	}
	var sb strings.Builder
	assert.NoError(t, html.Render(&sb, ctx.svgForRender(svg)))
	s := sb.String()
	assert.NotContains(t, s, "var(")
	assert.Contains(t, s, `fill="#ff0000"`)
	assert.Contains(t, s, `style="fill: blue"`, "the invalid declaration is removed")
	assert.Contains(t, s, `fill="green"`, "the fallback")
	assert.NotContains(t, s, "stroke", "attributes with invalid var() functions are removed")
	assert.NoError(t, core.NewSVG(b).ReadString(s))
}

// TestZoom tests that the sizes of a page are multiplied by the zoom of
// the context, as in browsers: lengths, font sizes, and images.
func TestZoom(t *testing.T) {
	sizes := func(zoom float32) (box, font float32) {
		b := core.NewBody()
		b.Styler(func(s *styles.Style) { s.Min.X.Dp(800) })
		ctx := NewContext()
		ctx.Zoom = zoom
		assert.NoError(t, ReadHTMLString(ctx, b, `<div class="box" style="width: 100px; height: 40px; padding: 10px; font-size: 20px">text</div>`))
		var w *core.WidgetBase
		b.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil && fmt.Sprint(wb.Property("class")) == "box" {
				w = wb
			}
			return tree.Continue
		})
		b.RenderImage(func() {
			box = w.Geom.Size.Actual.Total.X
			font = w.Styles.Font.Size.Dots
		})
		return
	}
	box1, font1 := sizes(0)
	box2, font2 := sizes(2)
	assert.InDelta(t, 2*box1, box2, 2, "lengths")
	assert.InDelta(t, 2*font1, font2, 0.5, "font sizes")
}

// TestSelectorIndex tests that selecting the nodes of selectors with the
// index of the nodes by id, class, and element name selects the same nodes
// as matching them against all of the nodes.
func TestSelectorIndex(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<html><body><div id="main" class="a b"><p class="a">x <span class="c a">y</span></p>
<ul class="list"><li class="item first">1</li><li class="item">2</li></ul><svg><linearGradient id="g"></linearGradient><rect class="r"/></svg>
<p id="p2" class="b">z</p><a href="#" class="a-b_c">link</a></div></body></html>`))
	assert.NoError(t, err)
	sels := []string{"p", "P", ".a", "#main", "div.a.b", "#main > .a", ".a span.c", "p.a:not(.b)", ".list .item:first-child",
		"li.item + li", "ul > li.first ~ li", "div :is(.a, .b)", ".a-b_c", "a[href='#'].a-b_c", "span:not(.a)", "#p2.b",
		"svg rect.r", "linearGradient", "lineargradient", "#g", "*", ".missing", "#missing", "p .a", "[class~=a]",
		"div p:not([id])", ".b > p", ".item.first", ".a\\:b", "html|p"}
	c := NewContext()
	for _, sel := range sels {
		assert.ElementsMatch(t, selectNodes(doc, sel), c.selectNodes(doc, sel), sel)
	}
	kind, key := selectorKey("div > .x :is(.y) #z.w")
	assert.Equal(t, "#z", string(kind)+key)
	kind, key = selectorKey("ul li:not(.a)")
	assert.Equal(t, "tli", string(kind)+key)
}

// TestUnrendered tests that the elements that the Unrendered function of
// the context reports are not displayed (as the light children of shadow
// hosts that are in no slot, as the closed menus of MDN).
func TestUnrendered(t *testing.T) {
	ctx := NewContext()
	ctx.Unrendered = func(n *html.Node) bool { return GetAttr(n, "slot") == "dropdown" }
	b := core.NewBody()
	require.NoError(t, ReadHTMLString(ctx, b, `<mdn-dropdown><button id="open">About</button><div slot="dropdown" id="panel">Get to know MDN better</div></mdn-dropdown>`))
	ids := map[string]bool{}
	b.WalkDown(func(k tree.Node) bool {
		if id, ok := k.AsTree().Property("id").(string); ok {
			ids[id] = true
		}
		return tree.Continue
	})
	assert.True(t, ids["open"])
	assert.False(t, ids["panel"], "not rendered")
}

// TestGridColumnSpan tests the spans of grid-column (as the 12-column
// grids of github.com, whose features span 6 and then 4 columns, in two
// rows, which were all in one row).
func TestGridColumnSpan(t *testing.T) {
	assert.Equal(t, 6, gridSpan("auto / span 6", 12))
	assert.Equal(t, 4, gridSpan("span 4", 12))
	assert.Equal(t, 2, gridSpan("1 / 3", 12))
	assert.Equal(t, 12, gridSpan("1 / -1", 12))
	assert.Equal(t, 1, gridSpan("auto", 12))
	cell := `<section style="grid-column: auto / span %d"><div style="height: 20px"></div></section>`
	b := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), b, `<div style="display: grid; grid-template-columns: repeat(12, minmax(0, 1fr)); width: 1200px">`+
		fmt.Sprintf(cell+cell+cell+cell+cell, 6, 6, 4, 4, 4)+`</div>`))
	var secs []*core.WidgetBase
	b.RenderImage(func() {
		b.WalkDown(func(k tree.Node) bool {
			if wb := core.AsWidget(k); wb != nil && wb.Property("tag") == "section" {
				secs = append(secs, wb)
			}
			return tree.Continue
		})
	})
	require.Len(t, secs, 5)
	dp := b.Styles.UnitContext.Dots(units.UnitDp)
	assert.InDelta(t, 600*dp, secs[0].Geom.Size.Actual.Total.X, 3, "half")
	assert.InDelta(t, 400*dp, secs[2].Geom.Size.Actual.Total.X, 3, "a third")
	assert.Greater(t, secs[2].Geom.Pos.Total.Y, secs[0].Geom.Pos.Total.Y, "in the second row")
}

// TestGridPlacement tests children of a grid placed by the numbers of
// their lines (as the cells of the repository grid of github.com), in
// any order, with an empty cell between them.
func TestGridPlacement(t *testing.T) {
	cell := func(id string, r, c int) string {
		return fmt.Sprintf(`<div id="%s" style="grid-row: %d; grid-column: %d; height: 20px"></div>`, id, r, c)
	}
	widgetsByID(t, `<div style="display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); width: 300px">`+
		cell("c", 2, 1)+cell("a", 1, 1)+cell("b", 1, 3)+`</div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			pos := func(id string) math32.Vector2 { return ws[id].Geom.Pos.Total.Sub(ws["a"].Geom.Pos.Total) }
			assert.InDelta(t, 200*dp, pos("b").X, 3, "in the third column")
			assert.InDelta(t, 0, pos("b").Y, 1)
			assert.InDelta(t, 0, pos("c").X, 1, "in the second row")
			assert.Greater(t, pos("c").Y, float32(10))
			assert.InDelta(t, 100*dp, ws["c"].Geom.Size.Actual.Total.X, 3, "one column wide, in a last row that is not full")
		})
}

// TestGridMinmaxZero tests that the columns of minmax(0, 1fr) are equal,
// whatever the widths of their content (as the repository grid of
// github.com, whose long hidden texts widened their columns).
func TestGridMinmaxZero(t *testing.T) {
	long := strings.Repeat("a long text of a hidden panel ", 5)
	widgetsByID(t, `<div style="display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); width: 300px">`+
		`<div id="a">`+long+`</div><div id="b">b</div><div id="c">c</div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			for _, id := range []string{"a", "b", "c"} {
				assert.InDelta(t, 100*dp, ws[id].Geom.Size.Actual.Total.X, 3, id)
			}
		})
}

// TestGridMinmaxZeroAuto tests that a column of minmax(0, auto) beside
// another column has the rest of the width of the grid, whatever the
// width of its content, which overflows it (as the documentation of
// pkg.go.dev, beside its navigation of 21.5%, whose long lines of code
// made the page wider than the window).
func TestGridMinmaxZeroAuto(t *testing.T) {
	long := strings.Repeat("x", 300)
	widgetsByID(t, `<div id="grid" style="display: grid; grid-template-columns: 100px minmax(0, auto); width: 500px">`+
		`<nav id="nav">nav</nav><article id="doc"><pre>`+long+`</pre></article></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 500*dp, ws["grid"].Geom.Size.Actual.Total.X, 2, "the grid")
			assert.InDelta(t, 100*dp, ws["nav"].Geom.Size.Actual.Total.X, 2)
			assert.InDelta(t, 400*dp, ws["doc"].Geom.Size.Actual.Total.X, 3, "the rest of the width")
		})
}

// TestTableFullWidthCell tests that a cell of width: 100% in a table with
// another column has the width that the other column leaves it, and that
// the table, with margins, is as wide as its container, not wider (as the
// message boxes of Wikipedia, which were wider than the article, covering
// the column beside it).
func TestTableFullWidthCell(t *testing.T) {
	long := strings.Repeat("This article needs more citations for verification. ", 4)
	widgetsByID(t, `<div id="col" style="width: 960px"><table id="tb" style="margin: 0 10%; border: 1px solid gray; border-left: 10px solid blue; box-sizing: border-box">`+
		`<tr><td id="img" style="padding: 2px 8px">I</td><td id="txt" style="padding: 4px 8px; width: 100%">`+long+`</td></tr></table></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, ws["col"].Geom.Size.Actual.Total.X, ws["tb"].Geom.Size.Actual.Total.X, 2, "the table with its margins")
			assert.Less(t, ws["img"].Geom.Size.Actual.Total.X, 40*dp, "the other column")
			assert.Greater(t, ws["txt"].Geom.Size.Actual.Total.X, 650*dp, "the rest of the width")
		})
}

// TestGridOutOfFlowChild tests that a child of a grid that is out of its
// flow (as a visually hidden heading) does not take a track, so the
// children after it are in their tracks (as the 8fr and 5fr columns of
// the testimonials of github.com, which were swapped).
func TestGridOutOfFlowChild(t *testing.T) {
	widgetsByID(t, `<div style="display: grid; grid-template-columns: minmax(0px, 8fr) minmax(0px, 5fr); width: 1300px; position: relative">`+
		`<h2 style="position: absolute; width: 1px; height: 1px; overflow: hidden">Testimonials</h2><div id="q">quotes</div><div id="s">stats</div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 800*dp, ws["q"].Geom.Size.Actual.Total.X, 3)
			assert.InDelta(t, 500*dp, ws["s"].Geom.Size.Actual.Total.X, 3)
		})
}

// TestGridAutoRowColumn tests a child of a grid with a column and an auto
// row, before children placed in row 1 (as the top stories of bbc.com):
// it is in row 1, where its columns are free, between them.
func TestGridAutoRowColumn(t *testing.T) {
	widgetsByID(t, `<div style="display: grid; grid-template-columns: repeat(24, 1fr); width: 1200px">`+
		`<div id="mid" style="grid-area: auto / 7 / auto / span 12; height: 20px"></div>`+
		`<div id="left" style="grid-area: 1 / 1 / span 1 / span 6; height: 20px"></div>`+
		`<div id="right" style="grid-area: 1 / 19 / span 1 / span 6; height: 20px"></div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			l, m, r := ws["left"].Geom, ws["mid"].Geom, ws["right"].Geom
			assert.InDelta(t, l.Pos.Total.Y, m.Pos.Total.Y, 1, "in the same row")
			assert.InDelta(t, l.Pos.Total.Y, r.Pos.Total.Y, 1)
			assert.InDelta(t, 600*dp, m.Size.Actual.Total.X, 6, "12 of 24 columns (with rounding)")
			assert.InDelta(t, l.Pos.Total.X+300*dp, m.Pos.Total.X, 6, "after the 6 columns of the left")
		})
}

// TestGridFrColumns tests that 1fr columns share the width equally,
// whatever the preferred widths of their content (as the top stories of
// bbc.com, whose wider photos made their columns wider), unless the
// minimum of their content is more than that.
func TestGridFrColumns(t *testing.T) {
	long := strings.Repeat("wide content of the first column ", 6)
	widgetsByID(t, `<div style="display: grid; grid-template-columns: repeat(3, 1fr); width: 900px">`+
		`<div id="a">`+long+`</div><div id="b">b</div><div id="c">c</div></div>`+
		`<div style="display: grid; grid-template-columns: repeat(3, 1fr); width: 900px">`+
		`<div id="d"><div style="width: 500px; height: 10px"></div></div><div id="e">e</div><div id="f">f</div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			for _, id := range []string{"a", "b", "c"} {
				assert.InDelta(t, 300*dp, ws[id].Geom.Size.Actual.Total.X, 3, id)
			}
			assert.InDelta(t, 500*dp, ws["d"].Geom.Size.Actual.Total.X, 3, "its minimum")
			assert.InDelta(t, 200*dp, ws["e"].Geom.Size.Actual.Total.X, 3, "the rest shared")
		})
}

// TestSVGTransform tests the CSS transforms of SVG elements as SVG
// transforms: in px and degrees, and none for those that cannot be (as
// translate(-.5em, 0) on bicycling.com, which logged errors).
func TestSVGTransform(t *testing.T) {
	env := NewContext().readEnv()
	for _, c := range []struct {
		css, svg string
		ok       bool
	}{
		{"translate(-.5em, 0)", "translate(-8,0)", true},
		{"translate(10px, 5px) rotate(45deg)", "translate(10,5) rotate(45)", true},
		{"rotate(0.25turn) scale(2)", "rotate(90) scale(2)", true},
		{"translateX(-1rem)", "translatex(-16)", true},
		{"none", "none", true},
		{"translate(-50%, -50%)", "", false},
		{"translate3d(0, 0, 0)", "", false},
		{"translate(calc(-1 * 3px))", "", false},
	} {
		got, ok := svgTransform(env, c.css)
		assert.Equal(t, c.ok, ok, c.css)
		if ok {
			assert.Equal(t, c.svg, got, c.css)
		}
	}
}

// TestLineHeightCalc tests a line height of a calc() of numbers, as that of
// the small text of Tailwind 4 (calc(1.25 / .875), through var()), which
// is a number, not a length in px (the text was 1.4px high, cut off by its
// boxes on fietsenwinkel.nl).
func TestLineHeightCalc(t *testing.T) {
	widgetsByID(t, `<style>:root { --lh: calc(1.25 / .875) } .text-sm { font-size: 14px; line-height: var(--tw-leading, var(--lh)) }</style>`+
		`<p id="p" class="text-sm">text</p>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 1.25/0.875, ws["p"].Styles.Text.LineHeight, 0.01)
			assert.InDelta(t, 20*dp, ws["p"].Geom.Size.Actual.Content.Y, 2, "one line of 20px")
		})
}

// TestEmptyInlineRun tests that a run of inline content that is only empty
// elements (as the span with a tabindex before the payload of the reCAPTCHA
// challenge) makes no line, as in browsers: it pushed the challenge down.
func TestEmptyInlineRun(t *testing.T) {
	widgetsByID(t, `<div id="a"><span tabindex="0"></span><div id="b">text</div></div>`+
		`<div id="c"><span>&nbsp;</span></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, ws["a"].Geom.Pos.Total.Y, ws["b"].Geom.Pos.Total.Y, 1, "no line before the block")
			assert.Greater(t, ws["c"].Geom.Size.Actual.Content.Y, 10*dp, "a no-break space is content")
		})
}

// TestFixedHeightOverflow tests that a box with a height and visible
// overflow is of that height, with its content overflowing it (as the
// instructions of the reCAPTCHA challenge, which made it too high for its
// frame), and that the height of a table cell is a minimum.
func TestFixedHeightOverflow(t *testing.T) {
	widgetsByID(t, `<div id="w" style="height: 20px; padding: 10px"><div id="d" style="height: 50px">tall</div></div><div id="n">next</div>`+
		`<table><tr><td id="td" style="height: 20px"><div style="height: 50px">cell</div></td></tr></table>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 40*dp, ws["w"].Geom.Size.Actual.Total.Y, 2, "the height and padding")
			assert.InDelta(t, 50*dp, ws["d"].Geom.Size.Actual.Total.Y, 2, "the content keeps its height")
			assert.InDelta(t, ws["w"].Geom.Pos.Total.Y+40*dp, ws["n"].Geom.Pos.Total.Y, 2, "the next box is after the height")
			assert.Greater(t, ws["td"].Geom.Size.Actual.Total.Y, 49*dp, "the content of a cell grows it")
		})
}

// TestBreakEmptyRow tests that a line break that ends an empty row of
// inline widgets (as the second of two after the logo of google.com)
// makes an empty line, as in browsers, and that a last one does not.
func TestBreakEmptyRow(t *testing.T) {
	widgetsByID(t, `<div id="a"><img width="20" height="20" src="data:image/gif;base64,R0lGODlhAQABAAAAACw="><br><br></div>`+
		`<div id="b"><img width="20" height="20" src="data:image/gif;base64,R0lGODlhAQABAAAAACw="><br></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			ha, hb := ws["a"].Geom.Size.Actual.Total.Y, ws["b"].Geom.Size.Actual.Total.Y
			assert.Greater(t, ha, hb+10*dp, "the second break makes a line")
			assert.Less(t, hb, 30*dp, "a last break makes no line")
		})
}

// TestBlockWidthFixedChild tests that a block of an auto width is as wide
// as its containing block, which a wider child of a fixed width overflows,
// as in CSS, and does not widen (as the blue instructions of the reCAPTCHA
// challenge, which made it scroll in width in its frame).
func TestBlockWidthFixedChild(t *testing.T) {
	widgetsByID(t, `<div style="width: 300px"><div id="w" style="padding: 24px"><div id="c" style="width: 400px">x</div></div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 300*dp, ws["w"].Geom.Size.Actual.Total.X, 2, "the block is as wide as its container")
			assert.InDelta(t, 400*dp, ws["c"].Geom.Size.Actual.Total.X, 2, "its child keeps its width")
		})
}

// TestTextTransformAndTruncation tests text-transform, and the lines of
// text clamped by -webkit-line-clamp and cut by text-overflow: ellipsis
// (as the cards of the products in the results of google.com, whose
// titles were not clamped, which made them much higher).
func TestTextTransformAndTruncation(t *testing.T) {
	long := strings.Repeat("words that do not fit ", 10)
	widgetsByID(t, `<div style="width: 200px">`+
		`<div id="up" style="text-transform: uppercase">Uitverkoop</div>`+
		`<div id="cap" style="text-transform: capitalize">gesponsorde <b>pro</b>ducten</div>`+
		`<div id="clamp" style="display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden">`+long+`</div>`+
		`<div id="cut" style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis">`+long+`</div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			text := func(id string) *core.Text {
				var tx *core.Text
				ws[id].WalkDown(func(n tree.Node) bool {
					if t, ok := n.(*core.Text); ok && tx == nil {
						tx = t
					}
					return tree.Continue
				})
				require.NotNil(t, tx, id)
				return tx
			}
			assert.Contains(t, text("up").Text, "UITVERKOOP")
			assert.Contains(t, text("cap").Text, "Gesponsorde")
			assert.Contains(t, text("cap").Text, "Pro</b>ducten", "the word continues in the next node")
			assert.Equal(t, 2, text("clamp").Styles.Text.LineClamp)
			line := text("up").Geom.Size.Actual.Content.Y // (one line)
			assert.InDelta(t, 2*line, text("clamp").Geom.Size.Actual.Content.Y, 3, "two lines")
			cut := text("cut")
			assert.True(t, cut.Styles.Text.Ellipsis)
			assert.LessOrEqual(t, cut.Geom.Size.Actual.Total.X, 201*dp, "cut to the width")
			assert.Greater(t, cut.Geom.Size.Actual.Total.X, 100*dp)
		})
}

// TestFlexWrapZeroBasisMinWidth tests that an item of flex: 1 1 0 with a
// min-width in a row that wraps is in a line by its min-width, not by its
// content, and grows to fill the line (as the content of the page of a
// repository of GitHub, of min-width: 1px, beside which its sidebar did
// not fit, so that it was below it).
func TestFlexWrapZeroBasisMinWidth(t *testing.T) {
	widgetsByID(t, `<div style="display: flex; flex-wrap: wrap; width: 600px">`+
		`<div id="c" style="flex: 1 1 0; width: 100%; min-width: 1px"><div style="width: 500px">wide</div></div>`+
		`<div id="p"><div style="width: 200px">pane</div></div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, ws["c"].Geom.Pos.Total.Y, ws["p"].Geom.Pos.Total.Y, 1, "on one line")
			assert.InDelta(t, 400*dp, ws["c"].Geom.Size.Actual.Total.X, 2, "the rest of the line")
		})
}

// TestVisibleInHidden tests that an element with visibility: visible in
// one with visibility: hidden gets the events at it, as in CSS, and that
// a hidden one does not (the other ways to sign in with two factors on
// GitHub, items made visible in a hidden list, could not be clicked).
func TestVisibleInHidden(t *testing.T) {
	widgetsByID(t, `<ul id="list" style="visibility: hidden; padding: 0"><li id="vis" style="visibility: visible">shown</li><li id="hid">hidden</li></ul>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			center := func(id string) image.Point {
				r := ws[id].Geom.TotalBBox
				return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
			}
			in := func(w core.Widget, id string) bool {
				for wb := w.AsWidget(); wb != nil; {
					if wb == ws[id] {
						return true
					}
					p, ok := wb.Parent.(core.Widget)
					if !ok {
						return false
					}
					wb = p.AsWidget()
				}
				return false
			}
			sc := ws["list"].Scene
			assert.True(t, in(sc.WidgetAt(center("vis")), "vis"), "the visible item gets the events")
			hw := sc.WidgetAt(center("hid"))

			assert.False(t, in(hw, "hid"), "the hidden item does not")
		})
}

// TestZeroHeightFlexEnd tests that a flex row of a height of 0 is that high,
// with its items at its end, above it, as in CSS (as the edition bar of
// nytimes.com, which was as high as its content, over the logo below it).
func TestZeroHeightFlexEnd(t *testing.T) {
	widgetsByID(t, `<section id="s" style="display: flex; height: 0; align-items: flex-end"><div id="c"><div>Today's</div><div>Paper</div></div></section><div id="after">after</div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			s, c := ws["s"].Geom, ws["c"].Geom
			assert.Less(t, s.Size.Actual.Total.Y, 2*dp, "of a height of 0")
			assert.InDelta(t, s.Pos.Total.Y+s.Size.Actual.Total.Y, c.Pos.Total.Y+c.Size.Actual.Total.Y, 2, "its item ends at its end")
			assert.InDelta(t, s.Pos.Total.Y+s.Size.Actual.Total.Y, ws["after"].Geom.Pos.Total.Y, 2, "the next block is right after it")
		})
}

// TestGridMinmaxFrMin tests that a track of minmax(length, fr) is at least
// the length (as the first, empty column of the grid of the results of
// google.com, of minmax(64px, 1fr), which was narrower, so that the header
// was to the left of where it is in browsers).
func TestGridMinmaxFrMin(t *testing.T) {
	widgetsByID(t, `<div id="g" style="display: grid; width: 560px; column-gap: 20px; grid-template-columns: minmax(64px, 1fr) repeat(8, 36px) minmax(0, 1fr)"><div id="i" style="grid-column: 2 / -1">header</div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			x := ws["i"].Geom.Pos.Total.X - ws["g"].Geom.Pos.Total.X
			// (at least: core grows it from its minimum by its share of the
			// rest, where CSS takes the larger of them)
			assert.GreaterOrEqual(t, x, 84*dp-1, "after the first column of at least 64px and its gap")
		})
}

// TestGridRowAfterFixedHeight tests that a row of a grid is after the
// height of the row before it, when its last cell has a smaller fixed
// height (as the actions of the height of a line of the page headers of
// github.com, after a title of two lines, which the description under it
// overlapped).
func TestGridRowAfterFixedHeight(t *testing.T) {
	widgetsByID(t, `<div style="display: grid; width: 600px; grid-template-columns: auto auto auto auto 1fr; grid-template-areas: 'c c c c c' 'l b t r a' 'd d d d d'">`+
		`<div id="t" style="grid-area: t; display: flex"><div><h1>a title</h1></div></div>`+
		`<div style="grid-area: a; display: flex; height: 20px"><button>action</button></div>`+
		`<div id="d" style="grid-area: d">the description under it</div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			tb := ws["t"].Geom.Pos.Total.Y + ws["t"].Geom.Size.Actual.Total.Y
			assert.GreaterOrEqual(t, ws["d"].Geom.Pos.Total.Y, tb-1, "the description is under the title")
		})
}

// TestContainIntrinsicSize tests that a box with size containment is of its
// contain-intrinsic size (as the rows of the file trees of GitHub pull
// requests, of contain: strict and contain-intrinsic-height: auto 32px,
// which were of no height, over each other), and that without size
// containment the intrinsic size does nothing.
func TestContainIntrinsicSize(t *testing.T) {
	widgetsByID(t, `<div style="width: 200px">`+
		`<div id="a" style="contain: strict; contain-intrinsic-height: auto 32px">a</div>`+
		`<div id="b" style="contain: strict; contain-intrinsic-height: auto 32px">b</div>`+
		`<div id="c" style="contain-intrinsic-height: 100px">c</div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 32*dp, ws["a"].Geom.Size.Actual.Total.Y, 1)
			assert.GreaterOrEqual(t, ws["b"].Geom.Pos.Total.Y, ws["a"].Geom.Pos.Total.Y+32*dp-1, "the rows are not over each other")
			assert.Less(t, ws["c"].Geom.Size.Actual.Total.Y, 50*dp, "no containment")
		})
}

// TestFixedTableColumns tests that the columns of a table with the fixed
// layout are as wide as the pixel widths of its col elements (as the line
// numbers of the diffs of GitHub, of width="52", which were narrower), and
// that a visually hidden header (thead.sr-only, positioned and clipped)
// is not a row of it.
func TestFixedTableColumns(t *testing.T) {
	widgetsByID(t, `<style>.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:rect(0 0 0 0)}</style>`+
		`<table id="t" style="table-layout: fixed; width: 600px"><thead class="sr-only"><tr><th id="h">Original file line number</th><th>line</th></tr></thead>`+
		`<colgroup><col width="52"><col></colgroup><tbody><tr><td id="n">1325</td><td id="l">cell image.Point</td></tr></tbody></table>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 52*dp, ws["n"].Geom.Size.Actual.Total.X, float64(2*dp), "the column of the line numbers")
			assert.Nil(t, ws["h"], "the hidden header is not in the table")
			assert.InDelta(t, ws["t"].Geom.Pos.Content.Y, ws["n"].Geom.Pos.Total.Y, float64(3*dp), "the first row is at the top")
		})
}

// TestCustomPropertyIndex tests the custom properties of an element with
// many declarations (whose custom properties are indexed, as those of the
// root of nytimes.com): the last declaration of each is its value, and
// those of an element in it override them.
func TestCustomPropertyIndex(t *testing.T) {
	var rules strings.Builder
	for i := range 2 * minCustomIndex {
		fmt.Fprintf(&rules, "--v%d: %dpx; ", i, i)
	}
	ctx := NewContext()
	doc, err := ParseHTML(ctx, strings.NewReader(`<html><head><style>:root { `+rules.String()+` --w: 10px; --w: 20px }
		#e { --v40: 5px }</style></head><body><div id="d" style="width: var(--w); height: var(--v40)">d</div>
		<div id="e"><p id="f" style="height: var(--v40)">f</p></div></body></html>`))
	require.NoError(t, err)
	require.NoError(t, ReadDocument(ctx, core.NewBody(), doc))
	d := elementByID(doc, "d")
	assert.Equal(t, "20px", ctx.ComputedValue(d, "width"))
	assert.Equal(t, "40px", ctx.ComputedValue(d, "height"))
	assert.Equal(t, "5px", ctx.ComputedValue(elementByID(doc, "f"), "height"))
}

// TestDisplayUnset tests that display: unset (and initial) is inline, which
// a flex item is as a block (as the widgets of the masthead of
// nytimes.com, which had no box, so that its ticker of width: 100% was as
// wide as the row, over the logo).
func TestDisplayUnset(t *testing.T) {
	widgetsByID(t, `<section id="bar" style="display: flex; justify-content: space-between; width: 1000px">`+
		`<div id="left">left</div><div id="w" style="display: unset"><div id="t" style="width: 100%; min-width: 105px">ticker</div></div></section>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			require.NotNil(t, ws["w"], "it has a box")
			assert.Less(t, ws["t"].Geom.Size.Actual.Total.X, 200*dp, "the ticker is narrow")
			assert.Greater(t, ws["t"].Geom.Pos.Total.X, ws["bar"].Geom.Pos.Total.X+800*dp, "at the end of the row")
		})
}
