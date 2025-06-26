// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yaegicore provides functions connecting
// https://github.com/cogentcore/yaegi to Cogent Core.
package yaegicore

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/textcore"
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

var currentGoalInterpreter Interpreter

// interpreterParent is used to store the parent widget ("b") for the interpreter.
// It exists (as a double pointer) such that it can be updated after-the-fact, such
// as in Cogent Lab/Goal where interpreters are re-used across multiple text editors,
// wherein the parent widget must be remotely controllable with a double pointer to
// keep the parent widget up-to-date.
var interpreterParent = new(*core.Frame)

// interpOutput is the output buffer for catching yaegi stdout.
// It must be a global variable because Goal re-uses the same interpreter,
// so it cannot be a local variable in [BindTextEditor].
var interpOutput bytes.Buffer

// getInterpreter returns a new interpreter for the given language,
// or [currentGoalInterpreter] if the language is "Goal" and it is non-nil.
func getInterpreter(language string) (in Interpreter, new bool, err error) {
	if language == "Goal" && currentGoalInterpreter != nil {
		return currentGoalInterpreter, false, nil
	}

	f := Interpreters[language]
	if f == nil {
		return nil, false, fmt.Errorf("no entry in yaegicore.Interpreters for language %q", language)
	}
	in = f(interp.Options{Stdout: &interpOutput})

	if language == "Goal" {
		currentGoalInterpreter = in
	}
	return in, true, nil
}

// BindTextEditor binds the given text editor to a yaegi interpreter
// such that the contents of the text editor are interpreted as code
// of the given language, which is run in the context of the given parent widget.
// It is used as the default value of [htmlcore.BindTextEditor].
func BindTextEditor(ed *textcore.Editor, parent *core.Frame, language string) {
	oc := func() {
		in, new, err := getInterpreter(language)
		if err != nil {
			core.ErrorSnackbar(ed, err)
			return
		}
		core.ExternalParent = parent
		*interpreterParent = parent
		coresymbols.Symbols["."]["b"] = reflect.ValueOf(interpreterParent).Elem()
		// the normal AutoPlanName cannot be used because the stack trace in yaegi is not helpful
		coresymbols.Symbols["cogentcore.org/core/tree/tree"]["AutoPlanName"] = reflect.ValueOf(func(int) string {
			return fmt.Sprintf("yaegi-%v", atomic.AddUint64(&autoPlanNameCounter, 1))
		})
		if new {
			errors.Log(in.Use(basesymbols.Symbols))
			errors.Log(in.Use(coresymbols.Symbols))
			in.ImportUsed()
		}

		parent.DeleteChildren()
		str := ed.Lines.String()
		// all Go code must be in a function for declarations to be handled correctly
		if language == "Go" && !strings.Contains(str, "func main()") {
			str = "func main() {\n" + str + "\n}"
		}
		interpOutput.Reset()
		_, err = in.Eval(str)
		ostr := interpOutput.String()
		if len(ostr) > 0 {
			out := textcore.NewEditor(parent)
			out.Styler(func(s *styles.Style) {
				s.SetReadOnly(true)
			})
			out.Lines.Settings.LineNumbers = false
			out.Lines.SetText([]byte(ostr))
		}
		if err != nil {
			core.ErrorSnackbar(ed, err, fmt.Sprintf("Error interpreting %s code", language))
			return
		}
		parent.Update()
	}
	ed.OnChange(func(e events.Event) { oc() })
	oc()
}
