// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// widgetsByElement returns the widgets of the elements with ids in the
// given body, by id.
func widgetsByElement(bd *core.Body) map[string]core.Widget {
	m := map[string]core.Widget{}
	bd.WalkDown(func(k tree.Node) bool {
		if w, ok := k.(core.Widget); ok {
			if id, ok := w.AsWidget().Property("id").(string); ok {
				if _, has := m[id]; !has {
					m[id] = w
				}
			}
		}
		return tree.Continue
	})
	return m
}

// bodyTexts returns the texts of the given body, in order.
func bodyTexts(bd *core.Body) string {
	var ts []string
	bd.WalkDown(func(k tree.Node) bool {
		if tx, ok := k.(*core.Text); ok {
			ts = append(ts, tx.Text)
		}
		return tree.Continue
	})
	return strings.Join(ts, "|")
}

// rereadChange is a change of a document for [testReread]: it returns the
// elements that have changed themselves and those whose children have.
type rereadChange func(doc *html.Node) (self, children []*html.Node)

// testReread reads the given HTML, changes its document with change after
// it is shown, reads the changed elements again with [Context.Reread],
// and tests that that is done (if reread) or not, that the widgets are
// then as those of the document read again whole, and that those of the
// elements of the given ids are kept.
func testReread(t *testing.T, src string, reread bool, change rereadChange, kept ...string) {
	t.Helper()
	ctx := NewContext()
	doc, err := ParseHTML(ctx, strings.NewReader(src))
	require.NoError(t, err)
	bd := core.NewBody()
	require.NoError(t, ReadDocument(ctx, bd, doc))
	var ok bool
	var before, after map[string]core.Widget
	var got, want map[string]string
	var gotTexts, wantTexts string
	bd.RenderImage(func() {
		bd.AsyncLock()
		before = widgetsByElement(bd)
		ok = ctx.Reread(change(doc))
		bd.AsyncUnlock()
	}, func() {
		bd.AsyncLock()
		got, gotTexts = widgetStates(bd), bodyTexts(bd)
		after = widgetsByElement(bd)
		bd.AsyncUnlock()
	})
	require.Equal(t, reread, ok, "reread")
	if !ok {
		return
	}
	fresh := core.NewBody()
	require.NoError(t, ReadDocument(NewContext(), fresh, doc))
	fresh.RenderImage(func() {}, func() { want, wantTexts = widgetStates(fresh), bodyTexts(fresh) })
	assert.Equal(t, want, got)
	assert.Equal(t, wantTexts, gotTexts)
	for _, id := range kept {
		require.NotNil(t, before[id], id)
		assert.Same(t, before[id], after[id], "the widget of %s is kept", id)
	}
}

// appendHTML parses the given HTML as the children of the given element
// and adds them to it.
func appendHTML(t *testing.T, n *html.Node, src string) {
	ns, err := html.ParseFragment(strings.NewReader(src), n)
	require.NoError(t, err)
	for _, k := range ns {
		n.AppendChild(k)
	}
}

// TestReread tests that changes of the documents of pages, as scripts
// make, are made by reading the changed elements again, reconciled with
// their widgets (see [Context.Reread]), as reading the page again would
// make them, keeping the widgets of the elements that have not changed,
// and that the changes that need it are not.
func TestReread(t *testing.T) {
	const page = `<html><head><style>
.item { height: 20px; background: blue; color: white }
.item + .item { margin-top: 10px }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 4px }
.cell { height: 30px; background: green }
.hidden { display: none }
</style></head><body><div style="width: 400px">
<div id="top"><p id="intro">Intro text</p></div>
<div id="main">
<div id="list"><div id="i1" class="item">one</div><div id="i2" class="item">two</div></div>
<p id="para">Some <b id="bold">bold</b> text</p>
<div id="grid" class="grid"><div id="c1" class="cell">1</div><div id="c2" class="cell">2</div></div>
</div>
<div id="after" class="item">after</div>
</div></body></html>`
	t.Run("text", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			b := elementByID(doc, "bold")
			b.FirstChild.Data = "strong and long"
			return nil, []*html.Node{b}
		}, "top", "intro", "list", "i1", "i2", "grid", "after")
	})
	t.Run("append", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			l := elementByID(doc, "list")
			appendHTML(t, l, `<div id="i3" class="item">three</div>`)
			return nil, []*html.Node{l}
		}, "main", "i1", "i2", "para", "grid", "after")
	})
	t.Run("insert first", func(t *testing.T) {
		// the item that was first is after another now (.item + .item)
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			l := elementByID(doc, "list")
			ns, err := html.ParseFragment(strings.NewReader(`<div id="i0" class="item">zero</div>`), l)
			require.NoError(t, err)
			l.InsertBefore(ns[0], l.FirstChild)
			return nil, []*html.Node{l}
		}, "main", "i2", "para")
	})
	t.Run("remove", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			l := elementByID(doc, "list")
			l.RemoveChild(elementByID(doc, "i1"))
			return nil, []*html.Node{l}
		}, "main", "para", "grid")
	})
	t.Run("hidden", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			i := elementByID(doc, "i2")
			setAttr(i, "class", "item hidden")
			return []*html.Node{i}, nil
		}, "i1", "para")
	})
	t.Run("grid", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			g := elementByID(doc, "grid")
			appendHTML(t, g, `<div id="c3" class="cell">3</div>`)
			return nil, []*html.Node{g}
		}, "main", "list", "para")
	})
	t.Run("grid first", func(t *testing.T) {
		// the cells after it are in other places of the grid
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			g := elementByID(doc, "grid")
			ns, err := html.ParseFragment(strings.NewReader(`<div id="c0" class="cell" style="grid-column: span 2; height: 50px">0</div>`), g)
			require.NoError(t, err)
			g.InsertBefore(ns[0], g.FirstChild)
			return nil, []*html.Node{g}
		}, "main", "list", "para")
	})
	t.Run("body", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			b := elementByID(doc, "top").Parent.Parent
			appendHTML(t, b, `<div id="new" style="width: 400px">new</div>`)
			return nil, []*html.Node{b}
		}, "top", "list", "i1", "para", "after")
	})
	t.Run("html", func(t *testing.T) {
		testReread(t, page, false, func(doc *html.Node) (self, children []*html.Node) {
			h := elementByID(doc, "top").Parent.Parent.Parent
			setAttr(h, "lang", "fr")
			return []*html.Node{h}, nil
		})
	})
	t.Run("too many", func(t *testing.T) {
		big := strings.Replace(page, `<div id="after"`, `<div id="big">`+strings.Repeat(`<span>x</span>`, maxReread)+`</div><div id="after"`, 1)
		testReread(t, big, false, func(doc *html.Node) (self, children []*html.Node) {
			b := elementByID(doc, "big")
			appendHTML(t, b, `<span>y</span>`)
			return nil, []*html.Node{b}
		})
	})
	t.Run("style sheet", func(t *testing.T) {
		testReread(t, page, false, func(doc *html.Node) (self, children []*html.Node) {
			m := elementByID(doc, "main")
			appendHTML(t, m, `<style>.item { color: red }</style>`)
			return nil, []*html.Node{m}
		})
	})
}
