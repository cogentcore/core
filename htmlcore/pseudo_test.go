// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"regexp"
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
		` <span class="gone">gone</span> <span class="q">q</span></p>`))
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
}
