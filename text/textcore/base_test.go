// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
)

// func TestBase(t *testing.T) {
// 	b := core.NewBody()
// 	NewBase(b)
// 	b.AssertRender(t, "basic")
// }
//
// func TestBaseSetText(t *testing.T) {
// 	b := core.NewBody()
// 	NewBase(b).Lines.SetString("Hello, world!")
// 	b.AssertRender(t, "set-text")
// }

func TestBaseSetLanguage(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	ed.Lines.SetLanguage(fileinfo.Go).SetString(`package main

func main() {
	fmt.Println("Hello, world!")
}
`)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Ch(40)
	})
	b.AssertRender(t, "set-lang")
}

/*
//go:embed editor.go
var myFile embed.FS

func TestBaseOpenFS(t *testing.T) {
	b := core.NewBody()
	errors.Log(NewBase(b).Lines.OpenFS(myFile, "editor.go"))
	b.AssertRender(t, "open-fs")
}

func TestBaseOpen(t *testing.T) {
	b := core.NewBody()
	errors.Log(NewBase(b).Lines.Open("editor.go"))
	b.AssertRender(t, "open")
}

func TestBaseMulti(t *testing.T) {
	b := core.NewBody()
	tb := NewLines().SetString("Hello, world!")
	NewBase(b).SetLines(tb)
	NewBase(b).SetLines(tb)
	b.AssertRender(t, "multi")
}

func TestBaseChange(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
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
	})
}

func TestBaseInput(t *testing.T) {
	b := core.NewBody()
	ed := NewBase(b)
	n := 0
	text := ""
	ed.OnInput(func(e events.Event) {
		n++
		text = ed.Lines.String()
	})
	b.AssertRender(t, "input", func() {
		ed.HandleEvent(events.NewKey(events.KeyChord, 'G', 0, 0))
		assert.Equal(t, 1, n)
		assert.Equal(t, "G\n", text)
		ed.HandleEvent(events.NewKey(events.KeyChord, 'o', 0, 0))
		assert.Equal(t, 2, n)
		assert.Equal(t, "Go\n", text)
		ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		assert.Equal(t, 3, n)
		assert.Equal(t, "Go\n\n", text)
	})
}

*/
