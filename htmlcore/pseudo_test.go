// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"regexp"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

// TestPseudoContent tests the text of ::before and ::after pseudo-elements
// (as the brackets of the edit links of Wikipedia): strings with escapes
// and alternative text, attr(), quotes, display: none, and the cascade.
func TestPseudoContent(t *testing.T) {
	n := &html.Node{Type: html.ElementNode, Data: "a", Attr: []html.Attribute{{Key: "title", Val: "T"}}}
	assert.Equal(t, "[", contentText(n, `"[" / ""`))
	assert.Equal(t, "•x", contentText(n, `"\2022 x"`), "the space after an escape is part of it")
	assert.Equal(t, "(T)", contentText(n, `"(" attr(title) ")"`))
	assert.Equal(t, "“q", contentText(n, `open-quote "q"`))
	assert.Equal(t, "", contentText(n, `none`))
	b, kind, ok := pseudoSelector(".x a::before")
	assert.Equal(t, [3]any{".x a", "before", true}, [3]any{b, kind, ok})

	bd := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, bd, `<style>.edit a::before { content: "[" / "" } .edit a::after { content: "]" }`+
		`.edit a.plain::after { content: none } .gone::before { content: "no"; display: none } .gone::after { content: "#"; visibility: hidden } .q:before { content: '\'' }</style>`+
		`<p>Syntax <span class="edit"><a href="#e">edit</a></span> <span class="edit"><a class="plain" href="#p">plain</a></span>`+
		` <span class="gone">gone</span> <span class="q">q</span></p><p><span class="edit"><a class="box" style="display: inline-block" href="#b">box</a></span></p>`))
	var text string
	bd.WalkDown(func(k tree.Node) bool {
		if tx, ok := k.(*core.Text); ok {
			text += tx.Text
		}
		return tree.Continue
	})
	plain := html.UnescapeString(regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, ""))
	assert.Contains(t, plain, "[edit]")
	assert.Contains(t, plain, "[plain gone", "content: none after it")
	assert.Contains(t, plain, " gone ", "display: none")
	assert.Contains(t, plain, "'q")
	assert.Contains(t, plain, "[box]", "an inline block, which is a widget")
}

// TestListItemDisplay tests that a li element with a display other than
// list-item (as the inline-flex items of breadcrumbs, with a ::after
// separator) is a box of its display, without a marker.
func TestListItemDisplay(t *testing.T) {
	bd := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, bd, `<style>.bc li { display: inline-flex } .bc li:not(:last-child)::after { content: ">" }</style>`+
		`<ol class="bc"><li><a href="/a">A</a></li><li><a href="/b">B</a></li></ol><ul><li>item</li></ul>`))
	markers := 0
	var texts []string
	bd.WalkDown(func(k tree.Node) bool {
		if tx, ok := k.(*core.Text); ok {
			if tx.Name == "marker" {
				markers++
			}
			texts = append(texts, tx.Text)
		}
		return tree.Continue
	})
	assert.Equal(t, 1, markers, "only the list-item has a marker")
	assert.Contains(t, texts, "&gt;")
}

// TestInlineBlockInHeading tests an inline block in a heading in an inline
// element (as the edit links of Wikipedia): it has the font size of the
// inline element, it is on the baseline of the heading (not at the bottom
// of the line), and its ::before and ::after have their space and color.
func TestInlineBlockInHeading(t *testing.T) {
	bd := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, bd, `<style>.es a { display: inline-block; line-height: 1 } .es a::before { content: "["; margin-right: 0.5em; color: #54595d } .es a::after { content: "]"; margin-inline: 0.25em 0 }</style>`+
		`<div><h2 style="display: inline; font-size: 48px">Syntax</h2><span class="es" style="font-size: small"><a href="#e">edit</a></span></div>`))
	var big, small *core.Text
	bd.RenderImage(func() {}, func() {
		bd.WalkDown(func(k tree.Node) bool {
			if tx, ok := k.(*core.Text); ok {
				if strings.Contains(tx.Text, "Syntax") {
					big = tx
				} else if strings.Contains(tx.Text, "edit") {
					small = tx
				}
			}
			return tree.Continue
		})
	})
	if assert.NotNil(t, big) && assert.NotNil(t, small) {
		assert.Contains(t, small.Text, "<span style=\"color: #54595d\">[ ⁠</span>", "the margin of ::before is an en space, in its color")
		assert.Contains(t, small.Text, "edit⁠ ]", "the last margin declaration of the side applies")
		bb, sb := big.Geom.TotalBBox, small.Geom.TotalBBox
		assert.Less(t, sb.Dy(), bb.Dy()/2, "the font size of the span")
		assert.Less(t, sb.Max.Y, bb.Max.Y, "on the baseline, above the descent of the heading")
		assert.Greater(t, sb.Min.Y, bb.Min.Y)
	}
}
