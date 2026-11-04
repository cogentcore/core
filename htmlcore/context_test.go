// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"

	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
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
		"rgb(255, 0, 0)":             {255, 0, 0, 255},
		"rgba(0, 0, 255, 0.2)":       colors.FromNRGBA(0, 0, 255, 51),
		"rgb(0 0 255 / 20%)":         colors.FromNRGBA(0, 0, 255, 51),
		"rgb(100%, 50%, 0%)":         {255, 128, 0, 255},
		"hsl(120, 100%, 50%)":        {0, 255, 0, 255},
		"hsla(240deg 100% 50% / .5)": colors.FromNRGBA(0, 0, 255, 128),
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
