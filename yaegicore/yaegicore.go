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
	"cogentcore.org/core/yaegicore/basesymbols"
	"cogentcore.org/core/yaegicore/coresymbols"
	"github.com/cogentcore/yaegi/interp"
)

// Interpreters is a map from language names (such as "Go") to functions that create a
// new [Interpreter] for that language. The base implementation is just [interp.Interpreter]
// for Go, but other packages can extend this. See the [Interpreter] interface for more information.
var Interpreters = map[string]func(options interp.Options) Interpreter{
	"Go": func(options interp.Options) Interpreter {
		return interp.New(options)
	},
}

// Interpreter is an interface that represents the functionality provided by an interpreter
// compatible with yaegicore. The base implementation is just [interp.Interpreter], but other
// packages such as yaegilab in Cogent Lab provide their own implementations with other languages
// such as Cogent Goal. See [Interpreters].
type Interpreter interface {

	// Use imports the given symbols into the interpreter.
	Use(values interp.Exports) error

	// ImportUsed imports the used symbols into the interpreter
	// and does any extra necessary configuration steps.
	ImportUsed()

	// Eval evaluates the given code in the interpreter.
	Eval(src string) (res reflect.Value, err error)
}

var autoPlanNameCounter uint64

func init() {
	htmlcore.BindTextEditor = BindTextEditor
	coresymbols.Symbols["."] = map[string]reflect.Value{} // make "." available for use
	basesymbols.Symbols["."] = map[string]reflect.Value{} // make "." available for use
}

// BindTextEditor binds the given text editor to a yaegi interpreter
// such that the contents of the text editor are interpreted as code
// of the given language, which is run in the context of the given parent widget.
// It is used as the default value of [htmlcore.BindTextEditor].
func BindTextEditor(ed *texteditor.Editor, parent core.Widget, language string) {
	oc := func() {
		inNew := Interpreters[language]
		if inNew == nil {
			core.ErrorSnackbar(ed, fmt.Errorf("no entry in yaegicore.Interpreters for language %q", language))
			return
		}
		in := inNew(interp.Options{})
		core.ExternalParent = parent
		coresymbols.Symbols["."]["b"] = reflect.ValueOf(parent)
		// the normal AutoPlanName cannot be used because the stack trace in yaegi is not helpful
		coresymbols.Symbols["cogentcore.org/core/tree/tree"]["AutoPlanName"] = reflect.ValueOf(func(int) string {
			return fmt.Sprintf("yaegi-%v", atomic.AddUint64(&autoPlanNameCounter, 1))
		})
		errors.Log(in.Use(basesymbols.Symbols))
		errors.Log(in.Use(coresymbols.Symbols))
		in.ImportUsed()

		parent.AsTree().DeleteChildren()
		str := ed.Buffer.String()
		// all Go code must be in a function for declarations to be handled correctly
		if language == "Go" && !strings.Contains(str, "func main()") {
			str = "func main() {\n" + str + "\n}"
		}
		_, err := in.Eval(str)
		if err != nil {
			core.ErrorSnackbar(ed, err, fmt.Sprintf("Error interpreting %s code", language))
			return
		}
		parent.AsWidget().Update()
	}
	ed.OnChange(func(e events.Event) { oc() })
	oc()
}
