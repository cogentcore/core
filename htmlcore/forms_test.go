// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestFormSubmit tests submitting a form, as the search of google.com,
// with Enter in its text field and with its submit buttons.
func TestFormSubmit(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	ctx.BrowserStyle = true
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
		"https://www.example.com/search?btnI=Lucky&hl=nl&q=cogent+core&safe=on",
		"/about",
	}, opened)
}

func TestFormSelect(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	ctx.BrowserStyle = true
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
