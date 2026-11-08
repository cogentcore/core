// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"image"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
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
		assert.True(t, summaries[0].Styles.Abilities.IsHoverable(), "for summary:hover")
	})
}

// TestToggleDetails tests that a click on a summary toggles its details
// with Context.ToggleDetails when it is set, as for an app that runs
// scripts (as the sidebar of MDN), which changes the document.
func TestToggleDetails(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	var toggled []*html.Node
	ctx.ToggleDetails = func(n *html.Node) bool { toggled = append(toggled, n); return true }
	ReadHTMLString(ctx, b, `<details id="d"><summary>More</summary><p>Hidden text</p></details>`)
	var summary *core.Frame
	b.WalkDown(func(n tree.Node) bool {
		if f, ok := n.(*core.Frame); ok && f.Property("tag") == "summary" {
			summary = f
		}
		return tree.Continue
	})
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		summary.Send(events.Click)
	}, func() {
		if assert.Len(t, toggled, 1) {
			assert.Equal(t, "d", GetAttr(toggled[0], "id"))
		}
	})
}

// TestRadioGroup tests that radio buttons are switches (as the widgets of
// the page, for the scripts), and that checking one unchecks the others
// of its group (as the appearance settings of Wikipedia).
func TestRadioGroup(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	var kinds []string
	ctx.AddWidgetHandler(func(w core.Widget) {
		if n, ok := w.AsWidget().Property("node").(*html.Node); ok && n.Data == "input" {
			kinds = append(kinds, fmt.Sprintf("%T", w))
		}
	})
	require.NoError(t, ReadHTMLString(ctx, b, `<input type="radio" name="size" id="a" checked><input type="radio" name="size" id="b"><input type="radio" name="other" id="c" checked><input type="checkbox" id="d">`))
	assert.Equal(t, []string{"*core.Switch", "*core.Switch", "*core.Switch", "*core.Switch"}, kinds)
	sw := map[string]*core.Switch{}
	b.WalkDown(func(n tree.Node) bool {
		if s, ok := n.(*core.Switch); ok {
			sw[s.Name] = s
		}
		return tree.Continue
	})
	b.RenderImage(func() {
		sw["b"].Send(events.Click)
	}, func() {
		assert.True(t, sw["b"].IsChecked())
		assert.False(t, sw["a"].IsChecked(), "unchecked by the other of its group")
		assert.True(t, sw["c"].IsChecked(), "another group")
	})
}

// TestPositionedButtonClick tests clicks with the mouse on a button that is
// positioned partly outside of its parent, with its text in an element (as
// the toggles of the contents of Wikipedia): the button gets them, not its
// text, also where it is outside of its parent.
func TestPositionedButtonClick(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	var button *core.WidgetBase
	clicks := 0
	ctx.AddWidgetHandler(func(w core.Widget) {
		if n, ok := w.AsWidget().Property("node").(*html.Node); ok && n.Data == "button" {
			button = w.AsWidget()
			button.OnClick(func(e events.Event) { clicks++ })
		}
	})
	require.NoError(t, ReadHTMLString(ctx, b, `<ul style="list-style: none; margin-left: 40px"><li style="position: relative; padding-left: 20px">`+
		`<button style="position: absolute; left: -20px; top: 0; width: 30px; height: 22px"><span>v</span></button><a href="#x">Syntax</a></li></ul>`))
	require.NotNil(t, button)
	b.RenderImage(func() {
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		bb := button.Geom.TotalBBox
		for _, x := range []int{bb.Min.X + 3, (bb.Min.X + bb.Max.X) / 2, bb.Max.X - 3} { // outside of its parent, on its text, inside
			pos := image.Pt(x, (bb.Min.Y+bb.Max.Y)/2)
			ev.MouseButton(events.MouseDown, events.Left, pos, 0)
			ev.MouseButton(events.MouseUp, events.Left, pos, 0)
		}
	}, func() {
		assert.Equal(t, 3, clicks)
	})
}

// TestCheckedSelector tests the :checked selector with the state of the
// controls from [Context.Checked], as for the CSS menus of Wikipedia: a
// menu after a hidden checkbox shows when the checkbox is checked, and the
// checked attributes of the document are restored after it is read.
func TestCheckedSelector(t *testing.T) {
	src := `<style>.menu { display: none } .cb:checked ~ .menu { display: block }</style>` +
		`<div><input type="checkbox" class="cb" id="cb"><label for="cb">Menu</label><div class="menu"><p>Item</p></div></div>`
	for _, checked := range []bool{false, true} {
		ctx := NewContext()
		var input *html.Node
		ctx.Checked = func(n *html.Node) bool {
			input = n
			return checked
		}
		bd := core.NewBody()
		require.NoError(t, ReadHTMLString(ctx, bd, src))
		shown := false
		var sw *core.Switch
		bd.WalkDown(func(k tree.Node) bool {
			if tx, ok := k.(*core.Text); ok && tx.Text == "Item" {
				shown = true
			}
			if s, ok := k.(*core.Switch); ok {
				sw = s
			}
			return tree.Continue
		})
		assert.Equal(t, checked, shown, "the menu shows when the checkbox is checked")
		require.NotNil(t, sw)
		assert.Equal(t, checked, sw.IsChecked(), "the state of the switch")
		require.NotNil(t, input)
		assert.False(t, HasAttr(input, "checked"), "the attribute is restored")
	}
}

// TestHoverSelector tests the :hover selector with the hovered elements
// from [Context.Hovered], as for the menus of pkg.go.dev: a submenu after
// a span in a menu item shows on the hover of the item, which
// [Context.HoverAffects] reports, and the document is unchanged after it
// is read.
func TestHoverSelector(t *testing.T) {
	src := `<style>.sub { display: none } .item:hover > .label:not(.closed) ~ .sub { display: block } a:hover { color: red }</style>` +
		`<ul><li class="item"><span class="label">Why Go</span><div class="sub"><p>Case studies</p></div></li><li class="other"><a href="x">Docs</a></li></ul>`
	for _, hovered := range []bool{false, true} {
		ctx := NewContext()
		var item, other *html.Node
		ctx.Hovered = func(n *html.Node) bool {
			switch GetAttr(n, "class") {
			case "item":
				item = n
				return hovered
			case "other":
				other = n
			}
			return false
		}
		bd := core.NewBody()
		require.NoError(t, ReadHTMLString(ctx, bd, src))
		shown := false
		bd.WalkDown(func(k tree.Node) bool {
			if tx, ok := k.(*core.Text); ok && tx.Text == "Case studies" {
				shown = true
			}
			return tree.Continue
		})
		assert.Equal(t, hovered, shown, "the submenu shows on the hover of the item")
		require.NotNil(t, item)
		require.NotNil(t, other)
		assert.True(t, ctx.HoverAffects(item), "the hover of the item affects its submenu")
		assert.False(t, ctx.HoverAffects(other), "the hover of the other item affects nothing")
		assert.False(t, HasAttr(item, hoverAttr), "the attribute is removed")
	}
}

// TestInputIcon tests an icon that is positioned over the start of a text
// input (as the search icon of the search field of Wikipedia): it is the
// leading icon of the text field, with the space before it as the padding
// of the field, and it is not rendered as an element over the field.
func TestInputIcon(t *testing.T) {
	icon := `url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20" fill="%23000"><path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1"/></svg>')`
	ctx := NewContext()
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(ctx, bd, `<style>.w { position: relative } .w input { padding-left: 36px }`+
		`.w .icon { position: absolute; left: 9px; top: 50%; width: 20px; height: 20px; mask-image: `+icon+` }</style>`+
		`<div class="w"><input type="search" placeholder="Search"> <span class="icon"></span></div>`))
	var tf *core.TextField
	spans := 0
	bd.WalkDown(func(k tree.Node) bool {
		if f, ok := k.(*core.TextField); ok {
			tf = f
		}
		if wb := core.AsWidget(k); wb != nil {
			if n, ok := wb.Property("node").(*html.Node); ok && n.Data == "span" {
				spans++
			}
		}
		return tree.Continue
	})
	require.NotNil(t, tf)
	assert.Contains(t, string(tf.LeadingIcon), `<path d="M8 1a7`, "the SVG of the icon")
	assert.Contains(t, string(tf.LeadingIcon), `fill="currentColor"`, "a mask has the color of the text")
	assert.Equal(t, 0, spans, "the icon element is not rendered")
	assert.Equal(t, "Search", tf.Placeholder)
	bd.RenderImage(func() {}, func() {
		dp := tf.Styles.UnitContext.Dots(units.UnitDp)
		assert.InDelta(t, 9*dp, tf.Styles.Padding.Left.Dots, 1, "the space before the icon")
		assert.False(t, tf.Styles.Text.WhiteSpace.HasWordWrap(), "an input is one line")
	})
}

// TestInputButtonLabelClick tests that a mouse click on the label of an
// input submit button (as the Compare selected revisions button of the
// history of Wikipedia) submits its form: the label is not selectable
// text, which would get the press.
func TestInputButtonLabelClick(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	ctx.PageURL = "https://www.example.com/page"
	var opened []string
	ctx.OpenURL = func(url string, e events.Event) { opened = append(opened, url) }
	require.NoError(t, ReadHTMLString(ctx, b, `<form action="/compare"><input type="hidden" name="diff" value="2"><input type="submit" value="Compare selected revisions"></form>`))
	var label *core.Text
	b.WalkDown(func(n tree.Node) bool {
		if tx, ok := n.(*core.Text); ok && tx.Text == "Compare selected revisions" {
			label = tx
		}
		return tree.Continue
	})
	require.NotNil(t, label)
	b.RenderImage(func() {
		pos := label.Geom.TotalBBox.Min.Add(label.Geom.TotalBBox.Size().Div(2))
		b.Scene.RenderWindow().SystemWindow.Events().MouseButton(events.MouseDown, events.Left, pos, 0)
		b.Scene.RenderWindow().SystemWindow.Events().MouseButton(events.MouseUp, events.Left, pos, 0)
	}, func() {
		assert.Equal(t, []string{"https://www.example.com/compare?diff=2"}, opened)
	})
}

// TestButtonHover tests that buttons (button elements and input buttons,
// as the Compare selected revisions button of Wikipedia) are darkened
// when the mouse is over them, by the state layer of their hover state.
func TestButtonHover(t *testing.T) {
	b := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), b, `<form><input type="submit" value="Compare"> <button>Go</button></form>`))
	var buttons []*core.Frame
	b.WalkDown(func(n tree.Node) bool {
		if fr, ok := n.(*core.Frame); ok && (fr.Property("tag") == "input" || fr.Property("tag") == "button") {
			buttons = append(buttons, fr)
		}
		return tree.Continue
	})
	require.Len(t, buttons, 2)
	b.RenderImage(func() {}, func() {
		for _, bt := range buttons {
			assert.True(t, bt.Styles.Abilities.IsHoverable(), bt.Property("tag"))
			bt.SetState(true, states.Hovered)
			bt.Style()
			assert.Greater(t, bt.Styles.StateLayer, float32(0), "a state layer when hovered")
		}
	})
}

// TestTransparentControlHover tests that the element after a transparent
// checkbox (as the hidden checkboxes of the menu buttons of Wikipedia,
// over their labels) shows the hover of the checkbox, with a state layer.
func TestTransparentControlHover(t *testing.T) {
	b := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), b, `<div style="position: relative"><input type="checkbox" id="cb" style="opacity: 0; position: absolute"><label for="cb" id="lb" style="display: inline-block">Menu</label></div>`))
	var sw *core.Switch
	var label *core.WidgetBase
	b.WalkDown(func(n tree.Node) bool {
		if s, ok := n.(*core.Switch); ok {
			sw = s
		}
		if wb := core.AsWidget(n); wb != nil && wb.Property("id") == "lb" && label == nil {
			label = wb
		}
		return tree.Continue
	})
	require.NotNil(t, sw)
	require.NotNil(t, label)
	b.RenderImage(func() {}, func() {
		sw.Send(events.MouseEnter)
		assert.True(t, label.StateIs(states.Hovered), "the label shows the hover")
		label.Style()
		assert.Greater(t, label.Styles.StateLayer, float32(0))
		sw.Send(events.MouseLeave)
		assert.False(t, label.StateIs(states.Hovered))
	})
}
