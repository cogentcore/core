// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yaegicore provides functions connecting
// https://github.com/traefik/yaegi to Cogent Core.
package yaegicore

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/texteditor"
	"github.com/traefik/yaegi/interp"
)

// BindTextEditor binds the given text editor to a yaegi interpreter
// such that the contents of the text editor are interpreted as Go
// code, which is run in the context of the given parent widget.
func BindTextEditor(ed *texteditor.Editor, parent core.Widget) {
	in := interp.New(interp.Options{})
	ed.OnChange(func(e events.Event) {
		in.Eval(ed.Buffer.String())
	})
}
