// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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
	t.Run("remove from body", func(t *testing.T) {
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
	// the elements in one that is not displayed, in an element of more than
	// can be read again (as the video players in the hidden dialogs of
	// npr.org, in its main element), have no widgets to read again
	inHidden := strings.Replace(page, `<div id="after"`, `<div id="big">`+strings.Repeat(`<span>x</span>`, maxReread)+
		`<div id="dialog" class="hidden"><div id="d1" class="hidden">one</div><p id="d2">two</p></div></div><div id="after"`, 1)
	t.Run("in hidden", func(t *testing.T) {
		testReread(t, inHidden, true, func(doc *html.Node) (self, children []*html.Node) {
			d1, d2 := elementByID(doc, "d1"), elementByID(doc, "d2")
			setAttr(d1, "class", "item")
			appendHTML(t, d2, `<b>more</b>`)
			return []*html.Node{d1}, []*html.Node{d2}
		}, "big", "list", "after")
	})
	t.Run("hidden shown", func(t *testing.T) {
		testReread(t, inHidden, false, func(doc *html.Node) (self, children []*html.Node) {
			d := elementByID(doc, "dialog")
			setAttr(d, "class", "")
			return []*html.Node{d}, nil
		})
	})
	t.Run("html", func(t *testing.T) {
		// styled again, with all of the elements, as the selectors of the
		// rules may select elements by its attributes (as :lang)
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
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
	t.Run("large page", func(t *testing.T) {
		// on a page of many elements, more of them may be read again than
		// on a small one (as the diffs that the pull requests of GitHub
		// load into a page of 35000 elements), which is still far faster
		// than reading it all again
		rest := `<div id="rest">` + strings.Repeat(`<div><span>r</span><span>s</span></div>`, 2*maxReread) + `</div>`
		big := strings.Replace(page, `<div id="after"`, `<div id="big">`+strings.Repeat(`<span>x</span>`, maxReread)+`</div>`+rest+`<div id="after"`, 1)
		testReread(t, big, true, func(doc *html.Node) (self, children []*html.Node) {
			b := elementByID(doc, "big")
			appendHTML(t, b, `<span>y</span>`)
			return nil, []*html.Node{b}
		})
	})
	t.Run("unshown only", func(t *testing.T) {
		// changes of elements that are not rendered only (as a script that
		// a page changes, between the views of GitHub), which crashed
		testReread(t, strings.Replace(page, `<div id="after"`, `<script id="s">1</script><div id="after"`, 1), true, func(doc *html.Node) (self, children []*html.Node) {
			s := elementByID(doc, "s")
			setAttr(s, "data-x", "1")
			return []*html.Node{s}, []*html.Node{s}
		})
	})
	t.Run("style rule added", func(t *testing.T) {
		// a rule added to a style sheet, as CSS-in-JS libraries add them
		// (as emotion on nytimes.com), restyles the elements that it
		// selects, keeping all of the widgets
		styled := strings.Replace(page, `<style>`, `<style id="css">`, 1)
		testReread(t, styled, true, func(doc *html.Node) (self, children []*html.Node) {
			st := elementByID(doc, "css")
			st.AppendChild(&html.Node{Type: html.TextNode, Data: ".item { background: red; height: 30px }"})
			return nil, []*html.Node{st}
		}, "top", "intro", "list", "i1", "i2", "para", "grid", "c1", "after")
	})
	t.Run("new style element", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			head := elementByID(doc, "intro").Parent.Parent.Parent.Parent.FirstChild
			for head.Type != html.ElementNode {
				head = head.NextSibling
			}
			appendHTML(t, head, `<style>.cell { background: orange }</style>`)
			return nil, []*html.Node{head}
		}, "top", "list", "i1", "grid", "c1", "c2", "after")
	})
	t.Run("style rule replaced", func(t *testing.T) {
		styled := strings.Replace(page, `<style>`, `<style id="css">`, 1)
		testReread(t, styled, false, func(doc *html.Node) (self, children []*html.Node) {
			st := elementByID(doc, "css")
			st.FirstChild.Data = ".item { background: red }"
			return nil, []*html.Node{st}
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

// TestRereadFallback tests the reasons that a reread and a restyle report
// when they can not make their changes, for the page to be read whole (as
// the verbose console of cogent web shows, as nytimes.com was read whole
// about 20 times as it loads, and its elements resized each time).
func TestRereadFallback(t *testing.T) {
	ctx := NewContext()
	doc, err := ParseHTML(ctx, strings.NewReader(`<html><head><style>.gone { display: none }</style></head><body>`+
		`<div id="big">`+strings.Repeat(`<span>x</span>`, maxReread)+`</div><div id="a">a</div></body></html>`))
	require.NoError(t, err)
	bd := core.NewBody()
	require.NoError(t, ReadDocument(ctx, bd, doc))
	bd.RenderImage(func() {
		bd.AsyncLock()
		defer bd.AsyncUnlock()
		big := elementByID(doc, "big")
		appendHTML(t, big, `<span>y</span>`)
		assert.False(t, ctx.Reread(nil, []*html.Node{big}))
		assert.Contains(t, ctx.Fallback(), "elements to read again")
		assert.Contains(t, ctx.Fallback(), "div#big")

		a := elementByID(doc, "a")
		setAttr(a, "class", "gone")
		assert.False(t, ctx.Restyle([]*html.Node{a}))
		assert.Contains(t, ctx.Fallback(), `display of div#a.gone from "block" to "none"`)
		setAttr(a, "class", "")
		assert.True(t, ctx.Restyle([]*html.Node{a}))
		assert.Equal(t, "", ctx.Fallback())
	})
}

// TestRereadBodyClass tests that a class added to the body element styles
// it again, and the elements that the rules of the class select by it (as
// .modal-open .dialog), without reading the page again, as the scripts of
// npr.org do as it loads, which read all of it again each time.
func TestRereadBodyClass(t *testing.T) {
	const page = `<html><head><style>
.box { width: 100px; height: 20px; background: blue }
.dark { background: black; color: white }
.dark .box { background: green }
.other .box { background: red }
</style></head><body><div style="width: 400px">
<div id="a" class="box">a</div><div id="b"><div id="c" class="box">c</div></div><p id="p">text</p>
</div></body></html>`
	testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
		body := elementByID(doc, "a").Parent.Parent
		setAttr(body, "class", "dark")
		return []*html.Node{body}, nil
	}, "p")
}

// TestRereadHTMLChildren tests that an element added to the html element
// after the body, as scripts add to measure things (as on npr.org), and
// removed, reads only the children of the html element again, with the
// body kept as it is.
func TestRereadHTMLChildren(t *testing.T) {
	const page = `<html><head><style>.box { width: 100px; height: 20px; background: blue }</style></head>` +
		`<body><div id="a" class="box">a</div><p id="p">text</p></body></html>`
	htmlOf := func(doc *html.Node) *html.Node { return elementByID(doc, "a").Parent.Parent }
	t.Run("add", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			h := htmlOf(doc)
			// (made here: a fragment parsed in the html element is in a
			// body of its own)
			h.AppendChild(&html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div,
				Attr: []html.Attribute{{Key: "id", Val: "probe"}, {Key: "style", Val: "position: absolute; width: 50px; height: 50px"}}})
			return nil, []*html.Node{h}
		}, "a", "p")
	})
	t.Run("remove", func(t *testing.T) {
		// (in the body, where the parser puts it, which is read again,
		// but for its first element; p, which is last now, is read again
		// for its margins)
		probe := strings.Replace(page, `</body>`, `</body><div id="probe" style="width: 50px; height: 50px"></div>`, 1)
		testReread(t, probe, true, func(doc *html.Node) (self, children []*html.Node) {
			pr := elementByID(doc, "probe")
			h := pr.Parent
			h.RemoveChild(pr)
			return nil, []*html.Node{h}
		}, "a")
	})
}

// TestRereadKeepsUnchanged tests that the elements that have not changed,
// in an element whose children have changed, are kept as they are without
// being read again (see [Context.keptElements]), so that elements added to
// a large body (as the scripts of npr.org add, many times as it loads) do
// not read all of it again (here in a div, as the body of a page that
// scrolls has a scrollbar only once it is laid out again), and that the
// result is as the page read anew,
// also for the rules of siblings that the change affects.
func TestRereadKeepsUnchanged(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<html><head><style>.item { height: 1px; background: blue }
.item + .item { margin-top: 1px }
.first:first-child { height: 3px }
.added { height: 2px }</style></head><body><div id="wrap" style="width: 400px">`)
	for i := range 1500 {
		fmt.Fprintf(&b, `<div id="d%d" class="item"><span>x</span></div>`, i)
	}
	b.WriteString(`</div></body></html>`)
	page := b.String()
	t.Run("append to body", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			wrap := elementByID(doc, "wrap")
			// (not of the class item: the rule for an item after an item
			// does not apply to one added at the end, as read again, also
			// when all of the items are read again)
			appendHTML(t, wrap, `<div id="new" class="added"><span>x</span></div>`)
			return nil, []*html.Node{wrap}
		}, "d0", "d700", "d1499")
	})
	t.Run("insert before the first", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			d0 := elementByID(doc, "d0")
			ns, err := html.ParseFragment(strings.NewReader(`<div id="new" class="item first"><span>x</span></div>`), d0.Parent)
			require.NoError(t, err)
			for _, n := range ns {
				d0.Parent.InsertBefore(n, d0)
			}
			return nil, []*html.Node{d0.Parent}
		}, "d700", "d1499")
	})
	t.Run("remove one", func(t *testing.T) {
		testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
			d := elementByID(doc, "d5")
			p := d.Parent
			p.RemoveChild(d)
			return nil, []*html.Node{p}
		}, "d0", "d700", "d1499")
	})
}

// TestRereadAddedRules tests that the rules added to a style sheet (as
// CSS-in-JS libraries add them, as on nytimes.com) restyle the elements
// that they select, found by the keys of their selectors, and the elements
// in them for the custom properties that they set, as the page read anew.
func TestRereadAddedRules(t *testing.T) {
	const page = `<html><head><style id="css">.box { width: 100px; height: 20px }
.label { color: var(--c, black) }</style></head><body><div style="width: 400px">
<div id="a" class="box"><div id="la" class="label">a</div></div><div id="b" class="box other">b</div>
<p id="p">text</p></div></body></html>`
	testReread(t, page, true, func(doc *html.Node) (self, children []*html.Node) {
		st := elementByID(doc, "css")
		st.AppendChild(&html.Node{Type: html.TextNode, Data: "#a { --c: red; background: blue } .other { height: 40px } p { color: green }"})
		return nil, []*html.Node{st}
	}, "a", "b", "p")
}
