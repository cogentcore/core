// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// TestReuseStyles tests that a page that is read again after its document
// changes, with the matches of the style rules of the context that read
// it before (see [Context.ReuseStyles]), is read as it is read whole, for
// the changes that change the matches of other elements than those that
// changed: of those in them, after them, before them (:has()), and of
// their parents (:empty), and that the rules are matched again only for
// the elements that may have other matches.
func TestReuseStyles(t *testing.T) {
	for sel, want := range map[string][2]bool{".a": {}, ".a + .b": {true, false}, ".a ~ .b .c": {true, true},
		".a+.b>.c": {true, true}, "li:nth-child(2n+1) a": {}, "[a~=b] .c": {}, ".a > .b ~ .c": {true, false}} {
		s, i := siblingCombinator(sel)
		assert.Equal(t, want, [2]bool{s, i}, sel)
	}
	page := `<html><head><style>
.open .x { color: red }
.a + .b { color: green }
.a ~ .c { background: green }
li:nth-child(2) { color: blue }
li:last-child { background: gray }
li:first-child div { background: purple }
.story > p:last-of-type + div { background: teal }
.box:empty { height: 30px; background: yellow }
.card:has(.hot) { border: 2px solid red }
.card:has(.hot) .title { color: orange }
p { margin: 4px }
</style></head><body style="width: 400px">
<div id="wrap"><span id="x" class="x">x</span></div>
<p id="pa">a</p><p id="pb" class="b">b</p><p id="pc" class="c">c</p>
<ul id="list"><li id="l1"><div id="d1">one</div></li><li id="l2">two</li><li id="l3">three</li></ul>
<div id="box" class="box"></div>
<div id="story" class="story"><p id="s1">1</p><p id="s2">2</p><div id="after2">after</div><span id="tail">tail</span></div>
<div id="card" class="card"><p id="title" class="title">title</p><p id="body">body</p></div>
<div id="filler">` + strings.Repeat(`<p>filler</p>`, 100) + `</div>
</body></html>`
	tests := []struct {
		name   string
		change func(doc *html.Node)
		reused bool // whether the matches are reused
	}{
		{"class of ancestor", func(doc *html.Node) { setAttr(elementByID(doc, "wrap"), "class", "open") }, true},
		{"sibling before", func(doc *html.Node) { setAttr(elementByID(doc, "pa"), "class", "a") }, true},
		{"insert first", func(doc *html.Node) {
			l := elementByID(doc, "list")
			ns, err := html.ParseFragment(strings.NewReader(`<li id="l0">zero</li>`), l)
			require.NoError(t, err)
			l.InsertBefore(ns[0], l.FirstChild)
		}, true},
		{"append", func(doc *html.Node) { appendHTML(t, elementByID(doc, "list"), `<li id="l4">four</li>`) }, true},
		{"remove last", func(doc *html.Node) {
			l := elementByID(doc, "list")
			l.RemoveChild(elementByID(doc, "l3"))
		}, true},
		{"last of type", func(doc *html.Node) { appendHTML(t, elementByID(doc, "story"), `<p id="s3">3</p>`) }, true},
		{"not empty", func(doc *html.Node) { appendHTML(t, elementByID(doc, "box"), `text`) }, true},
		{"has", func(doc *html.Node) { setAttr(elementByID(doc, "body"), "class", "hot") }, true},
		{"has removed", func(doc *html.Node) {
			setAttr(elementByID(doc, "body"), "class", "hot")
		}, true},
		{"style sheet", func(doc *html.Node) {
			appendHTML(t, elementByID(doc, "box"), `<style>p { color: purple }</style>`)
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			doc, err := ParseHTML(ctx, strings.NewReader(page))
			require.NoError(t, err)
			bd := core.NewBody()
			require.NoError(t, ReadDocument(ctx, bd, doc))
			if tt.name == "has removed" {
				// the class was there when it was read, and is removed
				ctx2 := ctx.NewPage()
				setAttr(elementByID(doc, "body"), "class", "hot")
				require.NoError(t, ReadDocument(ctx2, core.NewBody(), doc))
				ctx = ctx2
				setAttr(elementByID(doc, "body"), "class", "")
			} else {
				tt.change(doc)
			}
			again := core.NewBody()
			ctx2 := ctx.NewPage()
			ctx2.ReuseStyles(ctx)
			require.NoError(t, ReadDocument(ctx2, again, doc))
			if tt.reused {
				assert.Contains(t, ctx2.Reused(), "matched again")
			} else {
				assert.NotContains(t, ctx2.Reused(), "matched again")
			}
			fresh := core.NewBody()
			require.NoError(t, ReadDocument(ctx.NewPage(), fresh, doc))
			var got, want map[string]string
			again.RenderImage(func() {}, func() { got = widgetStates(again) })
			fresh.RenderImage(func() {}, func() { want = widgetStates(fresh) })
			assert.Equal(t, want, got)
		})
	}
}

// TestReuseStylesSiblings tests that the elements after one that is
// removed are matched again for the combinators of siblings (without
// selectors of their places, by which they would be anyway).
func TestReuseStylesSiblings(t *testing.T) {
	page := `<html><head><style>
.a + .b { background: green }
.a ~ .c div { background: blue }
</style></head><body style="width: 400px">
<div id="list"><p id="q1" class="a">a</p><p id="q2" class="b">b</p><div id="q3" class="c"><div id="q4">c</div></div></div>
<div id="filler">` + strings.Repeat(`<p>filler</p>`, 100) + `</div>
</body></html>`
	ctx := NewContext()
	doc, err := ParseHTML(ctx, strings.NewReader(page))
	require.NoError(t, err)
	require.NoError(t, ReadDocument(ctx, core.NewBody(), doc))
	l := elementByID(doc, "list")
	l.RemoveChild(elementByID(doc, "q1"))
	again := core.NewBody()
	ctx2 := ctx.NewPage()
	ctx2.ReuseStyles(ctx)
	require.NoError(t, ReadDocument(ctx2, again, doc))
	assert.Contains(t, ctx2.Reused(), "matched again")
	fresh := core.NewBody()
	require.NoError(t, ReadDocument(ctx.NewPage(), fresh, doc))
	var got, want map[string]string
	again.RenderImage(func() {}, func() { got = widgetStates(again) })
	fresh.RenderImage(func() {}, func() { want = widgetStates(fresh) })
	assert.Equal(t, want, got)
}

// TestReuseStylesAddedRules tests that the matches of a context are not
// reused when rules were added to it that no element was matched against
// (as for a style element that a script added, for which the elements
// were not read again), as in the adopted style sheets of a page.
func TestReuseStylesAddedRules(t *testing.T) {
	page := `<html><head></head><body style="width: 400px"><p id="p" class="hidden">p</p></body></html>`
	ctx := NewContext()
	doc, err := ParseHTML(ctx, strings.NewReader(page))
	require.NoError(t, err)
	require.NoError(t, ReadDocument(ctx, core.NewBody(), doc))
	head := elementByID(doc, "p").Parent.PrevSibling
	for head != nil && head.Data != "head" {
		head = head.PrevSibling
	}
	require.NotNil(t, head)
	appendHTML(t, head, `<style>.hidden { background: red }</style>`)
	require.True(t, ctx.addedStyle(head.LastChild)) // recorded, not matched
	again := core.NewBody()
	ctx2 := ctx.NewPage()
	ctx2.ReuseStyles(ctx)
	require.NoError(t, ReadDocument(ctx2, again, doc))
	assert.NotContains(t, ctx2.Reused(), "matched again")
	fresh := core.NewBody()
	require.NoError(t, ReadDocument(ctx.NewPage(), fresh, doc))
	var got, want map[string]string
	again.RenderImage(func() {}, func() { got = widgetStates(again) })
	fresh.RenderImage(func() {}, func() { want = widgetStates(fresh) })
	assert.Equal(t, want, got)
}
