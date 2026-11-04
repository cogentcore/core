// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/units"
	"github.com/andybalholm/cascadia"
	"github.com/aymerick/douceur/parser"
	"github.com/stretchr/testify/assert"
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
	ss, err := parser.Parse(userAgentStyles + browserStyles + "\nH1, p { color: red; }\ndiv, *, p { color: blue; }\np, p { color: green; }")
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
	ctx.BrowserStyle = true
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
	calcEnv.em, calcEnv.vw, calcEnv.vh = 16, 8, 6
	for v, want := range map[string]units.Value{
		"calc(100% - 20px)":        units.Pw(100),
		"calc(10px + 2em)":         units.Dp(42),
		"calc(100vh - 100px)":      units.Dp(500),
		"min(25vw, 350px)":         units.Dp(200),
		"max(10px, 1rem)":          units.Dp(16),
		"clamp(1rem, 5vw, 2rem)":   units.Dp(32),
		"calc((100px - 20px) / 2)": units.Dp(40),
		"calc(2 * (10px + 5px))":   units.Dp(30),
		"min(100%, 600px)":         units.Dp(600),
	} {
		got, ok := parseLength(v, units.UnitPw)
		assert.True(t, ok, v)
		assert.InDelta(t, want.Value, got.Value, 0.01, v)
		assert.Equal(t, want.Unit, got.Unit, v)
	}
}

func TestMediaHeightVh(t *testing.T) {
	saved := calcEnv
	defer func() { calcEnv = saved }()
	calcEnv.vh, calcEnv.fixedVh = 6, true // a 600 px viewport
	l, ok := parseLength("50vh", units.UnitPh)
	assert.True(t, ok)
	assert.Equal(t, units.Dp(300), l)
	l, ok = parseLength("calc(100vh - 100px)", units.UnitPh)
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

func TestMediaRange(t *testing.T) {
	c := NewContext()
	c.MediaWidth = 800
	c.MediaHeight = 600
	for q, want := range map[string]bool{
		"(width >= 50rem)":                 true,
		"(width >= 64rem)":                 false,
		"only screen and (width <= 52rem)": true,
		"(30rem < width <= 60rem)":         true,
		"(40rem < width)":                  true,
		"(height > 700px)":                 false,
		"(min-width: 600px)":               true,
	} {
		assert.Equal(t, want, c.mediaMatches(q), q)
	}
}

func TestGridTracks(t *testing.T) {
	assert.Equal(t, []string{"minmax(15rem,1fr)", "2rem", "minmax(0,48rem)"},
		gridTracks("[full-start left-start] minmax(15rem,1fr) [left-end] 2rem [content-start] minmax(0,48rem) [full-end right-end]"))
	assert.Equal(t, []string{"1fr", "1fr", "1fr", "auto"}, gridTracks("repeat(3, 1fr) auto"))
}

// TestCSSVariableInitial tests the space toggle of custom properties that
// are initial (as for light-dark() without native support), which are not set.
func TestCSSVariableInitial(t *testing.T) {
	ctx := NewContext()
	ctx.BrowserStyle = true
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
	ctx.BrowserStyle = true
	doc, _ := html.Parse(strings.NewReader(`<html><head><style>html { font-size: 62.5% }</style></head><body></body></html>`))
	ctx.Node = doc
	ctx.addStyleSheets(doc)
	assert.InDelta(t, 10, ctx.rootFontSize(), 0.01)
	saved := calcEnv
	defer func() { calcEnv = saved }()
	calcEnv.rem = 10
	l, ok := parseLength("2rem", units.UnitPw)
	assert.True(t, ok)
	assert.Equal(t, units.Dp(20), l)
	l, ok = parseLength("calc(1rem + 4px)", units.UnitPw)
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
