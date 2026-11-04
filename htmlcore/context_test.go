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
	"github.com/aymerick/douceur/parser"
	selcss "github.com/ericchiang/css"
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

// TestSelectNodes tests that the selectors of the user agent styles select
// the same nodes as the selector library selects.
func TestSelectNodes(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<html><body><h1>Title</h1><p>Text <b>bold</b></p><ul><li>one</li><li>two</li></ul><div><table><tr><td>cell</td><th>head</th></tr></table></div><ruby>x<rt>y</rt><rp>z</rp></ruby><svg><p>svg p</p></svg><center>c</center></body></html>`))
	assert.NoError(t, err)
	ss, err := parser.Parse(userAgentStyles + browserStyles + "\nH1, p { color: red; }\ndiv, *, p { color: blue; }\np, p { color: green; }")
	assert.NoError(t, err)
	for _, rule := range ss.Rules {
		for _, sel := range rule.Selectors {
			sel = strings.TrimSpace(sel)
			want := []*html.Node{}
			if s, err := selcss.Parse(sel); err == nil {
				want = s.Select(doc)
			}
			for i, n := range want { // :root is the html element
				if n.Type == html.DocumentNode {
					want[i] = n.FirstChild
				}
			}
			assert.ElementsMatch(t, want, selectNodes(doc, sel), sel)
		}
	}
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
