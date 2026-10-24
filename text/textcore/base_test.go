// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"embed"
	"testing"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// waitMarkup returns a function to pass to [core.Body.AssertRender] that
// does the background syntax highlighting of the given editor again and
// waits for its input event, so that the highlighting is always rendered.
// Otherwise it is done after a delay, which can be before or after the
// render, and its update is skipped while the editor is not shown yet.
func waitMarkup(t *testing.T, ed *Base) func() {
	inputs := make(chan struct{}, 100)
	ed.OnInput(func(e events.Event) {
		inputs <- struct{}{}
	})
	return func() {
		for len(inputs) > 0 {
			<-inputs
		}
		ed.Lines.ReMarkup()
		select {
		case <-inputs:
		case <-time.After(10 * time.Second):
			t.Error("no input from the background markup")
		}
	}
}

func TestBase(t *testing.T) {
	b := core.NewBody()
	NewBase(b)
	b.AssertRender(t, "basic")
}

func TestBaseSetText(t *testing.T) {
	b := core.NewBody()
	NewBase(b).Lines.SetString("Hello, world!")
	b.AssertRender(t, "set-text")
}

func TestBaseLayout(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	ed.Lines.SetLanguage(fileinfo.Go).SetString(`testing width
012345678 012345678 012345678 012345678
    fmt.Println("Hello, world!")
}
`)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(25)
	})
	b.AssertRender(t, "layout", waitMarkup(t, ed))
}

func TestBaseTabs(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	ed.Lines.SetLanguage(fileinfo.Go).SetString("\t\tinitial tabs are ok\nin\ttabs\n0123456\t\tdoubletab\tand\tmore")
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(25)
	})
	b.AssertRender(t, "tabs1", waitMarkup(t, ed))
}

func TestBaseSetLanguage(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	ed.Lines.SetLanguage(fileinfo.Go).SetString(`package main
//
func main() {
    fmt.Println("Hello, world!")
}
`)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(29)
	})
	b.AssertRender(t, "set-lang", waitMarkup(t, ed))
}

//go:embed base.go
var myFile embed.FS

func TestBaseOpenFS(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	errors.Log(ed.Lines.OpenFS(myFile, "base.go"))
	b.AssertRender(t, "open-fs", waitMarkup(t, ed))
}

func TestBaseOpen(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(40)
	})
	errors.Log(ed.Lines.Open("base.go"))
	b.AssertRender(t, "open", waitMarkup(t, ed))
}

func TestBaseScroll(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(40)
	})
	errors.Log(ed.Lines.Open("base.go"))
	b.AssertRender(t, "scroll-40", waitMarkup(t, ed), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ed.scrollToCenterIfHidden(textpos.Pos{Line: 40})
	})
	// ed.scrollToCenterIfHidden(textpos.Pos{Line: 42})
	// b.AssertRender(t, "scroll-42")
	// ed.scrollToCenterIfHidden(textpos.Pos{Line: 20})
	// b.AssertRender(t, "scroll-20")
}

func TestBaseMulti(t *testing.T) {
	b := core.NewBody()
	tb := lines.NewLines().SetString("Hello, world!")
	NewBase(b).SetLines(tb)
	NewBase(b).SetLines(tb)
	b.AssertRender(t, "multi")
}

// The views of a shared Lines get the updates of the background markup
// even when there are other views whose scenes are not shown, which
// must not block them.
func TestBaseMultiNotShown(t *testing.T) {
	b := core.NewBody()
	tb := lines.NewLines()
	ed := NewBase(b).SetLines(tb)
	for range 8 {
		NewBase(core.NewBody()).SetLines(tb)
	}
	inputs := make(chan struct{}, 100)
	ed.OnInput(func(e events.Event) {
		inputs <- struct{}{}
	})
	b.AssertRender(t, "multi-not-shown", func() {
		b.AsyncLock()
		tb.SetLanguage(fileinfo.Go).SetString("package main\n\nfunc main() {\n}\n")
		b.AsyncUnlock()
		for len(inputs) > 0 { // the inputs of setting the text
			<-inputs
		}
		select {
		case <-inputs:
		case <-time.After(5 * time.Second):
			t.Error("no input from the background markup")
		}
	})
}

func TestEditorChange(t *testing.T) {
	b := core.NewBody()
	ed := NewEditor(b)
	n := 0
	text := ""
	ed.OnChange(func(e events.Event) {
		n++
		text = ed.Lines.String()
	})
	b.AssertRender(t, "change", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ed.SetFocus()
		ed.HandleEvent(events.NewKey(events.KeyChord, 'G', 0, 0))
		assert.Equal(t, 0, n)
		assert.Equal(t, "", text)
		ed.HandleEvent(events.NewKey(events.KeyChord, 'o', 0, 0))
		assert.Equal(t, 0, n)
		assert.Equal(t, "", text)
		ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		assert.Equal(t, 0, n)
		assert.Equal(t, "", text)
		mods := key.Modifiers(0)
		mods.SetFlag(true, key.Control)
		ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, mods))
		assert.Equal(t, 1, n)
		assert.Equal(t, "Go\n\n", text)
	})
}

func TestEditorInput(t *testing.T) {
	b := core.NewBody()
	ed := NewEditor(b)
	n := 0
	text := ""
	ed.OnInput(func(e events.Event) {
		n++
		text = ed.Lines.String()
	})
	b.AssertRender(t, "input", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ed.SetFocus()
		ed.HandleEvent(events.NewKey(events.KeyChord, 'G', 0, 0))
		assert.Equal(t, 1, n)
		assert.Equal(t, "G\n", text)
		ed.HandleEvent(events.NewKey(events.KeyChord, 'o', 0, 0))
		assert.Equal(t, 2, n)
		assert.Equal(t, "Go\n", text)
		ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		assert.Equal(t, 4, n)
		assert.Equal(t, "Go\n\n", text)
	})
}
