// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yaegicore provides functions connecting
// https://github.com/cogentcore/yaegi to Cogent Core.
package yaegicore

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/yaegicore/symbols"
	"github.com/cogentcore/yaegi/interp"
)

var autoPlanNameCounter uint64

func init() {
	htmlcore.BindTextEditor = BindTextEditor
	symbols.Symbols["."] = map[string]reflect.Value{} // make "." available for use
}

// BindTextEditor binds the given text editor to a yaegi interpreter
// such that the contents of the text editor are interpreted as Go
// code, which is run in the context of the given parent widget.
// It is used as the default value of [htmlcore.BindTextEditor].
func BindTextEditor(ed *texteditor.Editor, parent core.Widget) {
	oc := func() {
		in := interp.New(interp.Options{})
		core.ExternalParent = parent
		symbols.Symbols["."]["b"] = reflect.ValueOf(parent)
		// the normal AutoPlanName cannot be used because the stack trace in yaegi is not helpful
		symbols.Symbols["cogentcore.org/core/tree/tree"]["AutoPlanName"] = reflect.ValueOf(func(int) string {
			return fmt.Sprintf("yaegi-%v", atomic.AddUint64(&autoPlanNameCounter, 1))
		})
		errors.Log(in.Use(symbols.Symbols))
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
