// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"embed"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/errors"
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
