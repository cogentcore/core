// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"embed"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gox/errors"
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

func TestEditorInput(t *testing.T) {
	b := core.NewBody()
	te := NewSoloEditor(b)
	n := 0
	text := ""
	te.OnInput(func(e events.Event) {
		n++
		text = te.Buffer.String()
	})
	b.AssertRender(t, "input", func() {
		te.HandleEvent(events.NewKey(events.KeyChord, 'G', 0, 0))
		assert.Equal(t, 1, n)
		assert.Equal(t, "G\n", text)
		te.HandleEvent(events.NewKey(events.KeyChord, 'o', 0, 0))
		assert.Equal(t, 2, n)
		assert.Equal(t, "Go\n", text)
		te.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		assert.Equal(t, 3, n)
		assert.Equal(t, "Go\n\n", text)
	})
}
