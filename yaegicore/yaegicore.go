// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yaegicore provides functions connecting
// https://github.com/traefik/yaegi to Cogent Core.
package yaegicore

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
	"github.com/traefik/yaegi/interp"
)

func init() {
	htmlcore.BindTextEditor = BindTextEditor
}

// BindTextEditor binds the given text editor to a yaegi interpreter
// such that the contents of the text editor are interpreted as Go
// code, which is run in the context of the given parent widget node.
// It is used as the default value of [htmlcore.BindTextEditor].
func BindTextEditor(ed *texteditor.Editor, parent tree.Node) {
	in := interp.New(interp.Options{})
	ed.OnInput(func(e events.Event) {
		_, err := in.Eval(ed.Buffer.String())
		core.ErrorSnackbar(ed, err, "Error interpreting Go code")
	})
}
