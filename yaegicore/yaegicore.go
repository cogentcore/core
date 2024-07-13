// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yaegicore provides functions connecting
// https://github.com/traefik/yaegi to Cogent Core.
package yaegicore

import (
	"reflect"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/yaegicore/symbols"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func init() {
	htmlcore.BindTextEditor = BindTextEditor
	delete(stdlib.Symbols, "errors/errors") // we have our own errors package
}

// BindTextEditor binds the given text editor to a yaegi interpreter
// such that the contents of the text editor are interpreted as Go
// code, which is run in the context of the given parent widget.
// It is used as the default value of [htmlcore.BindTextEditor].
func BindTextEditor(ed *texteditor.Editor, parent core.Widget) {
	oc := func() {
		in := interp.New(interp.Options{})
		symbols.Symbols["cogentcore.org/core/core/core"]["ExternalParent"].Set(reflect.ValueOf(parent))
		errors.Log(in.Use(stdlib.Symbols))
		errors.Log(in.Use(symbols.Symbols))
		errors.Log(in.Use(interp.Exports{
			".": map[string]reflect.Value{
				"parent": reflect.ValueOf(parent),
			},
		}))
		in.ImportUsed()

		parent.AsTree().DeleteChildren()
		str := ed.Buffer.String()
		// all code must be in a function for declarations to be handled correctly
		if !strings.Contains(str, "func main()") {
			str = "func main() {\n" + str + "\n}"
		}
		_, err := in.Eval(str)
		if err != nil {
			core.ErrorSnackbar(ed, err, "Error interpreting Go code")
			return
		}
		parent.AsWidget().Update()
	}
	ed.OnChange(func(e events.Event) { oc() })
	oc()
}
