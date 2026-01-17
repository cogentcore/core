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
	b.AssertRender(t, "layout")
}

func TestBaseTabs(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	ed.Lines.SetLanguage(fileinfo.Go).SetString("\t\tinitial tabs are ok\nin\ttabs\n0123456\t\tdoubletab\tand\tmore")
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(25)
	})
	b.AssertRender(t, "tabs1")
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
	b.AssertRender(t, "set-lang")
}

//go:embed base.go
var myFile embed.FS

func TestBaseOpenFS(t *testing.T) {
	b := core.NewBody()
	errors.Log(NewBase(b).Lines.OpenFS(myFile, "base.go"))
	b.AssertRender(t, "open-fs")
}

func TestBaseOpen(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(40)
	})
	errors.Log(ed.Lines.Open("base.go"))
	b.AssertRender(t, "open")
}

func TestBaseScroll(t *testing.T) {
	t.Skip("todo: unreliable, https://github.com/cogentcore/core/issues/1641")
	b := core.NewBody()
	ed := NewBase(b)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(40)
	})
	errors.Log(ed.Lines.Open("base.go"))
	b.AssertRender(t, "scroll-40", func() {
		ed.scrollToCenterIfHidden(textpos.Pos{Line: 40})
		time.Sleep(300 * time.Millisecond)
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
		time.Sleep(300 * time.Millisecond)
	})
}

func TestEditorInput(t *testing.T) {
	t.Skip("todo: unreliable, https://github.com/cogentcore/core/issues/1641")
	b := core.NewBody()
	ed := NewEditor(b)
	n := 0
	text := ""
	ed.OnInput(func(e events.Event) {
		n++
		text = ed.Lines.String()
	})
	b.AssertRender(t, "input", func() {
		ed.HandleEvent(events.NewKey(events.KeyChord, 'G', 0, 0))
		assert.Equal(t, 1, n)
		assert.Equal(t, "G\n\n", text)
		ed.HandleEvent(events.NewKey(events.KeyChord, 'o', 0, 0))
		assert.Equal(t, 2, n)
		assert.Equal(t, "Go\n\n", text)
		ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		assert.Equal(t, 4, n)
		assert.Equal(t, "Go\n\n", text)
		time.Sleep(300 * time.Millisecond)
	})
}
