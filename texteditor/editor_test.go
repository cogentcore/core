// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"embed"
	"testing"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/stretchr/testify/assert"
)

func TestEditor(t *testing.T) {
	b := core.NewBody()
	NewSoloEditor(b)
	b.AssertRender(t, "basic")
}

func TestEditorSetText(t *testing.T) {
	b := core.NewBody()
	NewSoloEditor(b).Buffer.SetTextString("Hello, world!")
	b.AssertRender(t, "set-text")
}

func TestEditorSetLang(t *testing.T) {
	b := core.NewBody()
	NewSoloEditor(b).Buffer.SetLang("go").SetTextString(`package main

	func main() {
		fmt.Println("Hello, world!")
	}
	`)
	b.AssertRender(t, "set-lang")
}

//go:embed editor.go
var myFile embed.FS

func TestEditorOpenFS(t *testing.T) {
	b := core.NewBody()
	errors.Log(NewSoloEditor(b).Buffer.OpenFS(myFile, "editor.go"))
	b.AssertRender(t, "open-fs")
}

func TestEditorOpen(t *testing.T) {
	b := core.NewBody()
	errors.Log(NewSoloEditor(b).Buffer.Open("editor.go"))
	b.AssertRender(t, "open")
}

func TestEditorMulti(t *testing.T) {
	b := core.NewBody()
	tb := NewBuffer().SetTextString("Hello, world!")
	NewEditor(b).SetBuffer(tb)
	NewEditor(b).SetBuffer(tb)
	b.AssertRender(t, "multi")
}

func TestEditorChange(t *testing.T) {
	b := core.NewBody()
	ed := NewSoloEditor(b)
	n := 0
	text := ""
	ed.OnChange(func(e events.Event) {
		n++
		text = ed.Buffer.String()
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

func TestEditorInput(t *testing.T) {
	b := core.NewBody()
	ed := NewSoloEditor(b)
	n := 0
	text := ""
	ed.OnInput(func(e events.Event) {
		n++
		text = ed.Buffer.String()
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
