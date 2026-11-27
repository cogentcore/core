// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runePoint returns the window position of the middle of the rune of the
// given index of the given text.
func runePoint(tx *Text, i int) image.Point {
	bb := tx.paintText.RuneBounds(i).Translate(tx.Geom.Pos.Content).ToRect()
	return bb.Min.Add(bb.Max).Div(2)
}

// TestTextSelect tests selecting text by sliding the mouse over it, back
// past where it started and forward again, and by double and triple
// clicking, and copying the selection.
func TestTextSelect(t *testing.T) {
	b := NewBody()
	tx := NewText(b).SetText("Hello there world")
	var pt func(i int) image.Point
	sel := func() string {
		return string(tx.richText.Join()[tx.selectRange.Start:tx.selectRange.End])
	}
	mouse := func(typ events.Types, p image.Point) {
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		if typ == events.MouseMove {
			ev.MouseMove(p)
		} else {
			ev.MouseButton(typ, events.Left, p, 0)
		}
		time.Sleep(15 * time.Millisecond)
	}
	prev := TimingSettings.SlideStartTime
	TimingSettings.SlideStartTime = 0
	defer func() { TimingSettings.SlideStartTime = prev }()
	notMulti := func() { // not a multiple click with the clicks before
		time.Sleep(TimingSettings.DoubleClickInterval + 50*time.Millisecond)
	}
	AssertRender(t, b, "text/select", Locked(b, func() {
		pt = func(i int) image.Point { return runePoint(tx, i) }
		pt(0) // (computed under the lock, then used without it)
	}), func() {
		b.AsyncLock()
		from, to := pt(6), pt(11)
		b.AsyncUnlock()
		DragMouse(b, from, to)
	}, Locked(b, func() {
		assert.Equal(t, "there", sel())
		SendKeyFunction(tx, keymap.Copy)
		assert.Equal(t, "there", tx.Clipboard().Read([]string{mimedata.TextPlain}).Text(mimedata.TextPlain))
		assert.False(t, tx.hasSelection(), "reset by copying")
	}), func() {
		// back past the start, and forward again
		notMulti()
		b.AsyncLock()
		p8, p3, p10 := pt(8), pt(3), pt(10)
		b.AsyncUnlock()
		mouse(events.MouseDown, p8)
		for _, p := range []image.Point{p8.Add(image.Pt(-6, 0)), p3, p10} {
			mouse(events.MouseMove, p)
		}
		mouse(events.MouseUp, p10)
	}, Locked(b, func() {
		assert.Equal(t, "er", sel(), "from where the selection started")
	}), func() {
		b.AsyncLock()
		p := pt(14)
		b.AsyncUnlock()
		mouse(events.MouseDown, p)
		mouse(events.MouseUp, p)
		mouse(events.MouseDown, p)
		mouse(events.MouseUp, p)
	}, Locked(b, func() {
		assert.Equal(t, "world", sel(), "a double click selects the word")
	}), func() {
		b.AsyncLock()
		p := pt(2)
		b.AsyncUnlock()
		for range 3 {
			mouse(events.MouseDown, p)
			mouse(events.MouseUp, p)
		}
	}, Locked(b, func() {
		assert.Equal(t, "Hello there world", sel(), "a triple click selects all")
		m := NewScene()
		tx.contextMenu(m, image.Point{})
		_, isFunc := m.Child(0).(*FuncButton)
		assert.True(t, isFunc, "the copy button")
	}))
}

// TestTextLinks tests finding links in text, their tooltips, and copying
// them from the context menu, and searching text.
func TestTextLinks(t *testing.T) {
	b := NewBody()
	tx := NewText(b).SetText(`Go to <a href="https://example.com">the site</a> now`)
	AssertRender(t, b, "text/links", Locked(b, func() {
		require.Len(t, tx.Links, 1)
		p := runePoint(tx, 7)
		tl, bb := tx.findLink(p)
		require.NotNil(t, tl)
		assert.Equal(t, "https://example.com", tl.URL)
		tip, tpos := tx.WidgetTooltip(p)
		assert.Equal(t, "https://example.com", tip)
		assert.Equal(t, bb.Min, tpos)
		tx.Tooltip = "plain"
		tip, _ = tx.WidgetTooltip(runePoint(tx, 0))
		assert.Equal(t, "plain", tip, "not over the link")
		tip, _ = tx.WidgetTooltip(image.Pt(-1, -1))
		assert.Equal(t, "plain", tip)

		m := NewScene()
		tx.contextMenu(m, p)
		ButtonByText(m, "Copy link").Send(events.Click)
		assert.Equal(t, "https://example.com", tx.Clipboard().Read([]string{mimedata.TextPlain}).Text(mimedata.TextPlain))

		tx.HandleEvent(events.NewMouseMove(events.NoButton, p, p, 0))
		assert.NotNil(t, tx.PaintText())
		assert.Equal(t, "Go to the site now", string(tx.TextRunes()))
		matches, ok := tx.Search("O", IgnoreCase)
		assert.True(t, ok)
		assert.Len(t, matches, 3)
		matches, _ = tx.Search("O", UseCase)
		assert.Empty(t, matches)
	}))
}

// TestTextLinkContextMenu tests that the context menu of a right click on
// a link that is not at the center of the text has Copy link: the context
// menu event made from the mouse event has its position.
func TestTextLinkContextMenu(t *testing.T) {
	b := NewBody()
	tx := NewText(b).SetText(`<a href="https://example.com">Link</a> and then a lot more words after it`)
	var p image.Point
	RenderImage(b, Locked(b, func() {
		p = runePoint(tx, 1)
	}), func() {
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseButton(events.MouseDown, events.Right, p, 0)
		ev.MouseButton(events.MouseUp, events.Right, p, 0)
	}, Locked(b, func() {
		m := TopScene(b)
		require.NotEqual(t, b.Scene, m, "the menu")
		assert.NotNil(t, ButtonByText(m, "Copy link"))
		m.Close()
	}))
}
