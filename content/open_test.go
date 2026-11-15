// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"io"
	"testing"
	"testing/fstest"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/text/csl"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openFS is the content of [TestOpen]: pages with headings, an id, an
// embedded page, a citation, and the generated references page.
var openFS = fstest.MapFS{
	"home.md": {Data: []byte("+++\nURL = \"\"\n+++\n\nWelcome to [[Alpha]].\n")},
	"alpha.md": {Data: []byte(`+++
Categories = ["Docs"]
+++

The lead of alpha, citing [[@Smith20]].

## First

Text of the first.

{id="table_one"}
| A | B |
|---|---|
| 1 | 2 |

## Second

<embed-page src="beta" title="Embedded beta">
`)},
	"beta.md": {Data: []byte(`+++
Categories = ["Docs"]
+++

The lead of beta.

## Beta heading

More of beta.
`)},
	"references.md": {Data: []byte("# References\n\n<p id=\"Smith20\">Smith, J. (2020). A paper.</p>\n")},
}

// heldOpen calls the given function with the lock of the given content.
func heldOpen(ct *Content, fun func()) func() {
	return func() {
		ct.AsyncLock()
		defer ct.AsyncUnlock()
		fun()
	}
}

// TestOpen tests opening pages, their headings, ids and references, a
// redirect to the most similar page, the history, a new tab, the search
// of the pages, and the toolbar.
func TestOpen(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b := core.NewBody("content-open")
	ct := NewContent(b)
	ct.References = csl.NewKeyList([]csl.Item{
		{CitationKey: "Smith20", Type: csl.ArticleJournal, Title: "A paper", Author: []csl.Name{{Family: "Smith", Given: "J"}}, Issued: csl.Date{DateParts: [][]any{{"2020"}}}},
	})
	ct.SetSource(openFS)
	var tb *core.Toolbar
	b.AddTopBar(func(bar *core.Frame) {
		tb = core.NewToolbar(bar)
		tb.Maker(func(p *tree.Plan) {
			ct.MakeToolbar(p)
			ct.MakeToolbarSearch(p)
			ct.MakeToolbarPDF(p)
		})
	})
	button := func(tip string) *core.Button {
		for k := range tb.All() {
			if bt, ok := k.(*core.Button); ok && bt.Tooltip == tip {
				return bt
			}
		}
		t.Fatalf("no button %q", tip)
		return nil
	}
	core.AssertRender(t, b, "open", heldOpen(ct, func() {
		assert.Equal(t, "", ct.current.Page.URL, "the home page")
		assert.False(t, ct.historyHasBack())
		ct.Open("alpha")
	}), heldOpen(ct, func() {
		assert.Equal(t, "alpha", ct.current.Page.URL)
		assert.True(t, ct.historyHasBack())
		assert.False(t, ct.historyHasForward())
		embedded := false
		for k := range ct.rightFrame.All() {
			if tx, ok := k.(*core.Text); ok && tx.Text == "Embedded beta" {
				embedded = true
			}
		}
		assert.True(t, embedded, "the embedded page")
		ct.Open("alpha#second")
		assert.Equal(t, "second", ct.current.Heading, "the same page")
		ct.Open("alpha#no-such-heading")
		ct.Open("alpha#table_one")
		ct.Open("alpha#table_one:missing")
		ct.Open("alfa") // redirected
	}), heldOpen(ct, func() {
		assert.Equal(t, "alpha", ct.current.Page.URL, "the most similar page")
		ct.Open("ref://Smith20")
	}), heldOpen(ct, func() {
		assert.Equal(t, "references", ct.current.Page.URL)
		ct.historyBack()
	}), heldOpen(ct, func() {
		assert.Equal(t, "alpha", ct.current.Page.URL, "back")
		assert.True(t, ct.historyHasForward())
		ct.historyForward()
	}), heldOpen(ct, func() {
		assert.Equal(t, "references", ct.current.Page.URL, "forward")
		button("Back").Send(events.Click)
	}), heldOpen(ct, func() {
		assert.Equal(t, "alpha", ct.current.Page.URL, "the back button")
		button("Forward").Send(events.Click)
		button("Home").Send(events.Click)
	}), heldOpen(ct, func() {
		assert.Equal(t, "", ct.current.Page.URL, "the home button")
		var items []core.ChooserItem
		ct.MenuSearch(&items)
		require.Len(t, items, len(ct.pages))
		for _, it := range items {
			if it.Text == "Beta" {
				it.Func()
			}
		}
	}), heldOpen(ct, func() {
		assert.Equal(t, "beta", ct.current.Page.URL, "from the search")
		refs := ct.PageRefs(ct.pagesByURL["alpha"])
		require.NotNil(t, refs)
		assert.NotEmpty(t, refs.Children)

		n := ct.tabs.NumTabs()
		var mods key.Modifiers
		mods.SetFlag(true, key.Control)
		ct.OpenEvent("alpha", events.NewKey(events.KeyChord, 0, 0, mods))
		assert.Equal(t, n+1, ct.tabs.NumTabs(), "a new tab")
		ct.OpenEvent("beta", nil)
	}), heldOpen(ct, func() {
		assert.Equal(t, "beta", ct.current.Page.URL)
		loc := &Location{Page: ct.current.Page, Heading: "x", URL: "beta#x"}
		assert.Equal(t, `"beta#x" (page: "beta" head: "x")`, loc.String())
		assert.Equal(t, `"u"`, (&Location{URL: "u"}).String())
	}))
}

// TestLoadEachPage tests loading each of the pages, with a function after
// them.
func TestLoadEachPage(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b := core.NewBody("content-load-each")
	ct := NewContent(b).SetSource(openFS)
	done := make(chan struct{})
	core.AssertRender(t, b, "load-each", func() {
		require.NoError(t, ct.LoadEachPage(func() { close(done) }))
		select {
		case <-done:
		case <-time.After(20 * time.Second):
			t.Error("not all pages are loaded")
		}
	})
	assert.Equal(t, "references", ct.current.Page.URL, "the last one")
}

// TestCompiledCode tests that a Go code block compiled into the app runs
// directly, and again with another compiled block when it is edited, and
// that a panic in it is shown as an error.
func TestCompiledCode(t *testing.T) {
	defer func(c map[string]CodeFunc, bte func(*textcore.Editor, *core.Frame, string)) {
		CompiledCode, BindTextEditor = c, bte
	}(CompiledCode, BindTextEditor)
	CompiledCode = map[string]CodeFunc{}
	BindTextEditor = nil
	src1 := `core.NewText(b).SetText("one")`
	src2 := `core.NewText(b).SetText("two")`
	AddCompiledCode(map[string]CodeFunc{
		CodeKey(src1): func(b *core.Frame, out io.Writer) {
			core.NewText(b).SetText("one")
			io.WriteString(out, "printed\n")
		},
		CodeKey(src2): func(b *core.Frame, out io.Writer) { panic("two fails") },
	})
	b := core.NewBody()
	ed := textcore.NewEditor(b)
	ed.Lines.SetString(src1)
	fr := core.NewFrame(b)
	BindCode(ed, fr, "Go")
	require.NotEmpty(t, fr.Children)
	assert.Equal(t, "one", fr.Child(0).(*core.Text).Text)
	assert.Len(t, fr.Children, 2, "and the output")

	ed.Lines.SetString(src2)
	ed.SendChange()
	assert.NotEmpty(t, fr.Children, "the error")

	ed.Lines.SetString("not compiled")
	ed.SendChange() // no interpreter: an error snackbar

	interpreted := ""
	BindTextEditor = func(ed *textcore.Editor, parent *core.Frame, language string) { interpreted = language }
	ed.Lines.SetString(src1)
	ed.SendChange()
	ed.Lines.SetString("edited")
	ed.SendChange()
	assert.Equal(t, "Go", interpreted, "edited code is interpreted")

	ed2 := textcore.NewEditor(b)
	ed2.Lines.SetString("fmt.Println(1)")
	BindCode(ed2, core.NewFrame(b), "Go")
	BindTextEditor = nil
	BindCode(ed2, core.NewFrame(b), "Python") // not run
}

// TestOpenEventNoTabs tests opening pages in a new tab, without tabs, as
// before the content is shown: a page has no URL other than on the web
// (which panicked), and a link and a reference are opened directly.
func TestOpenEventNoTabs(t *testing.T) {
	fsys := fstest.MapFS{}
	for name, f := range openFS {
		fsys["content/"+name] = f
	}
	ct := NewContent(core.NewBody()).SetContent(fsys)
	require.Nil(t, ct.tabs)
	var mods key.Modifiers
	mods.SetFlag(true, key.Meta)
	e := events.NewKey(events.KeyChord, 0, 0, mods)
	assert.NotPanics(t, func() { ct.OpenEvent("beta#beta-heading", e) })
	ct.OpenEvent("https://example.com", e)
	ct.OpenEvent("ref://Smith20", e)
	assert.Equal(t, "references", ct.current.Page.URL)
	ct.Open("http://example.com") // opened in the browser
	assert.Equal(t, "references", ct.current.Page.URL)

	defer func(c map[string]*GoalPage) { CompiledGoal = c }(CompiledGoal)
	CompiledGoal = map[string]*GoalPage{}
	AddCompiledGoal(map[string]*GoalPage{"a.md": {}})
	assert.Contains(t, CompiledGoal, "a.md")
}
