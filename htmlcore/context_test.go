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

	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
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
		got, ok := env.length(v, units.UnitPw)
		assert.True(t, ok, v)
		assert.InDelta(t, want.Value, got.Value, 0.01, v)
		assert.Equal(t, want.Unit, got.Unit, v)
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

func TestGridTracks(t *testing.T) {
	assert.Equal(t, []string{"minmax(15rem,1fr)", "2rem", "minmax(0,48rem)"},
		gridTracks("[full-start left-start] minmax(15rem,1fr) [left-end] 2rem [content-start] minmax(0,48rem) [full-end right-end]"))
	assert.Equal(t, []string{"1fr", "1fr", "1fr", "auto"}, gridTracks("repeat(3, 1fr) auto"))
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
