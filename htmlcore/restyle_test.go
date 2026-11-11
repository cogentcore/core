// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// elementByID returns the element of the given id in the given document.
func elementByID(doc *html.Node, id string) *html.Node {
	var found *html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && GetAttr(n, "id") == id {
			found = n
			return
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			walk(k)
		}
	}
	walk(doc)
	return found
}

// setAttr sets the given attribute of the given element.
func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

// widgetStates returns the boxes and the styles that are rendered of the
// widgets of the elements with ids in the given body, by id.
func widgetStates(bd *core.Body) map[string]string {
	m := map[string]string{}
	bd.WalkDown(func(k tree.Node) bool {
		if wb := core.AsWidget(k); wb != nil {
			if id, ok := wb.Property("id").(string); ok {
				s := &wb.Styles
				m[id] = fmt.Sprintf("pos %v size %v color %v bg %v opacity %v", wb.Geom.Pos.Total, wb.Geom.Size.Actual.Total,
					colors.AsHex(colors.ToUniform(s.Color)), colors.AsHex(colors.ToUniform(s.Background)), s.Opacity) + fmt.Sprintf(" translate %v rotate %v", s.Translate, s.Rotate)
			}
		}
		return tree.Continue
	})
	return m
}

// testRestyle reads the given HTML, changes its document with change after
// it is shown, restyles the elements that change returns, and tests that
// that is done (if restyled) or not, and that the widgets are then as
// those of the document read again.
func testRestyle(t *testing.T, src string, restyled bool, change func(doc *html.Node) []*html.Node) {
	t.Helper()
	ctx := NewContext()
	doc, err := ParseHTML(ctx, strings.NewReader(src))
	require.NoError(t, err)
	bd := core.NewBody()
	require.NoError(t, ReadDocument(ctx, bd, doc))
	var ok bool
	var before, got, want map[string]string
	bd.RenderImage(func() {
		bd.AsyncLock()
		before = widgetStates(bd)
		ok = ctx.Restyle(change(doc))
		bd.AsyncUnlock()
	}, func() {
		bd.AsyncLock()
		got = widgetStates(bd)
		bd.AsyncUnlock()
	})
	require.Equal(t, restyled, ok, "restyled")
	if !ok {
		return
	}
	fresh := core.NewBody()
	require.NoError(t, ReadDocument(NewContext(), fresh, doc))
	fresh.RenderImage(func() {}, func() { want = widgetStates(fresh) })
	assert.Equal(t, want, got)
	assert.NotEqual(t, before, got, "the change shows")
}

// TestRestyle tests that changes of the styles of elements (by their
// style and class attributes and their states) are made by styling their
// widgets again (see [Context.Restyle]), as reading the page again would
// make them, and that the changes that need it are not.
func TestRestyle(t *testing.T) {
	const page = `<html><head><style>
.box { width: 100px; height: 20px; background: blue }
.open .item { color: red; width: 60px }
.open ~ .after { background: green; margin-left: 30px }
.wide { width: 200px }
.moved { transform: translateX(40px) }
.gone { display: none }
.wrap:has(.on) .target { width: 150px }
.wrap2:has(.on2) { width: 170px }
.marked::before { content: "x" }
.v { width: var(--w, 10px); height: 10px; background: red }
.wv { --w: 90px }
</style></head><body><div style="width: 400px">
<div id="a" class="box">a</div>
<div id="menu" class="menu"><div id="item" class="item box">item</div></div>
<div id="after" class="after box">after</div>
<div class="wrap"><div id="target" class="target box">t</div><div id="trigger">x</div></div>
<div id="wrap2" class="wrap2 box"><div><div id="trigger2">deep</div></div></div>
<div id="vars"><div id="v" class="v"></div></div>
<p id="p">text <span id="span">span</span> more</p>
</div></body></html>`
	t.Run("style width", func(t *testing.T) {
		testRestyle(t, page, true, func(doc *html.Node) []*html.Node {
			a := elementByID(doc, "a")
			setAttr(a, "style", "width: 250px; background: orange")
			return []*html.Node{a}
		})
	})
	t.Run("class of descendants and siblings", func(t *testing.T) {
		testRestyle(t, page, true, func(doc *html.Node) []*html.Node {
			m := elementByID(doc, "menu")
			setAttr(m, "class", "menu open")
			return []*html.Node{m}
		})
	})
	t.Run("transform", func(t *testing.T) {
		testRestyle(t, page, true, func(doc *html.Node) []*html.Node {
			a := elementByID(doc, "a")
			setAttr(a, "class", "box moved")
			return []*html.Node{a}
		})
	})
	// :has() in the subject compound, of an ancestor (only the ancestors
	// and their previous siblings are matched again, see relationalChanges)
	for _, c := range []struct{ id, class string }{{"trigger2", "on2"}} {
		t.Run("subject has "+c.class, func(t *testing.T) {
			testRestyle(t, page, true, func(doc *html.Node) []*html.Node {
				tr := elementByID(doc, c.id)
				setAttr(tr, "class", c.class)
				return []*html.Node{tr}
			})
		})
	}
	t.Run("has", func(t *testing.T) {
		testRestyle(t, page, true, func(doc *html.Node) []*html.Node {
			tr := elementByID(doc, "trigger")
			setAttr(tr, "class", "on")
			return []*html.Node{tr}
		})
	})
	t.Run("custom property", func(t *testing.T) {
		testRestyle(t, page, true, func(doc *html.Node) []*html.Node {
			v := elementByID(doc, "vars")
			setAttr(v, "style", "--w: 70px")
			return []*html.Node{v}
		})
	})
	t.Run("custom property of a class", func(t *testing.T) {
		// (of a style sheet, not of the style attribute, as the lookups of
		// custom properties are memoized, see Context.varMemo)
		testRestyle(t, page, true, func(doc *html.Node) []*html.Node {
			v := elementByID(doc, "vars")
			setAttr(v, "class", "wv")
			return []*html.Node{v}
		})
	})
	t.Run("display", func(t *testing.T) {
		testRestyle(t, page, false, func(doc *html.Node) []*html.Node {
			a := elementByID(doc, "a")
			setAttr(a, "class", "box gone")
			return []*html.Node{a}
		})
	})
	t.Run("pseudo-element", func(t *testing.T) {
		testRestyle(t, page, false, func(doc *html.Node) []*html.Node {
			a := elementByID(doc, "a")
			setAttr(a, "class", "box marked")
			return []*html.Node{a}
		})
	})
	t.Run("inline", func(t *testing.T) {
		testRestyle(t, page, false, func(doc *html.Node) []*html.Node {
			sp := elementByID(doc, "span")
			setAttr(sp, "style", "color: green")
			return []*html.Node{sp}
		})
	})
}

// TestSubjectStart tests the start of the subject compound of selectors,
// after their last combinator outside of parentheses, brackets and quotes.
func TestSubjectStart(t *testing.T) {
	for sel, want := range map[string]string{
		".a:has(.b)":             ".a:has(.b)",
		".a:has(.b) .c":          ".c",
		"div > .a:has(> .b ~ c)": ".a:has(> .b ~ c)",
		`a[title="x y"]:has(b)`:  `a[title="x y"]:has(b)`,
		".x + .y:not(:has(.z))":  ".y:not(:has(.z))",
	} {
		assert.Equal(t, want, sel[subjectStart(sel):], sel)
	}
}

// TestHasCompound tests the compound of the :has() selectors of selectors
// whose :has() is not in their subject, without them.
func TestHasCompound(t *testing.T) {
	for sel, want := range map[string]struct {
		compound string
		siblings bool
	}{
		".a:has(.b) .c":                 {".a", false},
		"body:has(.modal.open) > .page": {"body", false},
		".x .a:has(> .b ~ .c) + .d":     {".a", true},
		":has(.b) .c":                   {"*", false},
		".a:has(.b) .c:has(.d)":         {"", false}, // in two compounds
		".a:not(:has(.b)) .c":           {".a", false},
		".a.b:where(:not(:has(.c))) .x": {".a.b", false},
	} {
		cs, sib := hasCompound(sel)
		if want.compound == "" {
			assert.Nil(t, cs, sel)
			continue
		}
		if assert.NotNil(t, cs, sel) {
			assert.Equal(t, want.siblings, sib, sel)
		}
	}
}
