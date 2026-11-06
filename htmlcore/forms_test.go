// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestFormSubmit tests submitting a form, as the search of google.com,
// with Enter in its text field and with its submit buttons.
func TestFormSubmit(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	ctx.PageURL = "https://www.example.com/page"
	var opened []string
	ctx.OpenURL = func(url string, e events.Event) { opened = append(opened, url) }
	ReadHTMLString(ctx, b, `<form action="/search"><input type="hidden" name="hl" value="nl">
<input name="q" value=""><input type="checkbox" name="safe" checked>
<input type="submit" name="btnG" value="Search"> <input type="submit" name="btnI" value="Lucky"></form>
<a href="/about" style="display:inline-block">About</a>`)
	var tf *core.TextField
	var buttons, links []*core.Frame
	b.WalkDown(func(n tree.Node) bool {
		switch w := n.(type) {
		case *core.TextField:
			tf = w
		case *core.Frame:
			switch w.Property("tag") {
			case "input":
				buttons = append(buttons, w)
			case "a":
				links = append(links, w)
			}
		}
		return tree.Continue
	})
	if !assert.NotNil(t, tf) || !assert.Len(t, buttons, 2) || !assert.Len(t, links, 1) {
		return
	}
	tf.SetText("cogent core")
	tf.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
	buttons[1].Send(events.Click)
	links[0].Send(events.Click)
	assert.Equal(t, []string{
		"https://www.example.com/search?hl=nl&q=cogent+core&safe=on",
		"https://www.example.com/search?hl=nl&q=cogent+core&safe=on&btnI=Lucky",
		"/about",
	}, opened)
}

// TestFormPost tests submitting forms with the POST method, as the
// cookie consent of google.com, in each encoding.
func TestFormPost(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	ctx.PageURL = "https://consent.example.com/ml"
	type post struct{ url, contentType, body string }
	var posts []post
	ctx.PostURL = func(url, contentType string, body []byte, e events.Event) {
		posts = append(posts, post{url, contentType, string(body)})
	}
	ReadHTMLString(ctx, b, `<form action="/save" method="POST"><input type="hidden" name="set_eom" value="true">
<input type="hidden" name="continue" value="https://www.example.com/?q=a&b"><button>Reject all</button></form>
<form action="/save" method="post" enctype="multipart/form-data"><input type="hidden" name="x" value="1"><button>Multi</button></form>
<form method="post" enctype="text/plain"><input type="hidden" name="x" value="1"><button>Text</button></form>`)
	var buttons []*core.Frame
	b.WalkDown(func(n tree.Node) bool {
		if w, ok := n.(*core.Frame); ok && w.Property("tag") == "button" {
			buttons = append(buttons, w)
		}
		return tree.Continue
	})
	if !assert.Len(t, buttons, 3) {
		return
	}
	for _, bt := range buttons {
		bt.Send(events.Click)
	}
	if !assert.Len(t, posts, 3) {
		return
	}
	assert.Equal(t, post{"https://consent.example.com/save", "application/x-www-form-urlencoded",
		"set_eom=true&continue=https%3A%2F%2Fwww.example.com%2F%3Fq%3Da%26b"}, posts[0])
	assert.Equal(t, "https://consent.example.com/save", posts[1].url)
	assert.Contains(t, posts[1].contentType, "multipart/form-data; boundary=")
	assert.Contains(t, posts[1].body, "name=\"x\"\r\n\r\n1\r\n")
	assert.Equal(t, post{"https://consent.example.com/ml", "text/plain", "x=1\r\n"}, posts[2])
}

func TestFormSelect(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	ctx.PageURL = "https://www.example.com/page"
	var opened []string
	ctx.OpenURL = func(url string, e events.Event) { opened = append(opened, url) }
	ReadHTMLString(ctx, b, `<form action="/go"><select name="lang"><option value="en">English</option>
<option value="fr" selected>Français</option><option>Deutsch</option></select>
<input type="submit" value="Go"></form>`)
	var ch *core.Chooser
	var submit *core.Frame
	b.WalkDown(func(n tree.Node) bool {
		switch w := n.(type) {
		case *core.Chooser:
			ch = w
		case *core.Frame:
			if w.Property("tag") == "input" {
				submit = w
			}
		}
		return tree.Continue
	})
	if !assert.NotNil(t, ch) || !assert.NotNil(t, submit) {
		return
	}
	assert.Len(t, ch.Items, 3)
	assert.Equal(t, "Français", ch.CurrentItem.Text)
	submit.Send(events.Click)
	ch.SetCurrentIndex(2)
	submit.Send(events.Click)
	assert.Equal(t, []string{"https://www.example.com/go?lang=fr", "https://www.example.com/go?lang=Deutsch"}, opened)
}

func TestDetails(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	ReadHTMLString(ctx, b, `<details><summary>More</summary><p>Hidden text</p></details>
<details open><summary>Open</summary><p>Shown text</p></details>`)
	var summaries []*core.Frame
	texts := map[string]*core.Text{}
	b.WalkDown(func(n tree.Node) bool {
		switch w := n.(type) {
		case *core.Frame:
			if w.Property("tag") == "summary" {
				summaries = append(summaries, w)
			}
		case *core.Text:
			texts[w.Text] = w
		}
		return tree.Continue
	})
	if !assert.Len(t, summaries, 2) || !assert.NotNil(t, texts["Hidden text"]) {
		return
	}
	visible := func(tx *core.Text) bool {
		for w := tx.AsWidget(); w != nil; {
			if w.Styles.Display == styles.DisplayNone {
				return false
			}
			p, ok := w.Parent.(core.Widget)
			if !ok {
				break
			}
			w = p.AsWidget()
		}
		return true
	}
	b.RenderImage(func() {
		assert.False(t, visible(texts["Hidden text"]))
		assert.True(t, visible(texts["Shown text"]))
		assert.NotNil(t, texts["▸ "], "a closed marker")
		b.AsyncLock()
		defer b.AsyncUnlock()
		summaries[0].Send(events.Click)
	}, func() {
		assert.True(t, visible(texts["Hidden text"]), "opened by a click")
	})
}
