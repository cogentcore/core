// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mvmcore provides functions connecting the mvm Go interpreter
// (https://github.com/cogentcore/mvm, a fork of github.com/mvm-sh/mvm) to
// Cogent Core, such that content code blocks can be edited and run.
package mvmcore

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync/atomic"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/mvmcore/basesymbols"
	"cogentcore.org/core/mvmcore/coresymbols"
	"cogentcore.org/core/text/textcore"
	"github.com/cogentcore/mvm/interp"
	"github.com/cogentcore/mvm/lang/golang"
	"github.com/cogentcore/mvm/stdlib"
	_ "github.com/cogentcore/mvm/stdlib/core" // the core standard library
)

// Interpreters is a map from language names (such as "Go") to functions that
// make a new [Interpreter] for that language, writing its output to out. The
// base implementation is an mvm interpreter for Go (see [NewGo]), but other
// packages can add other languages, as cogentcore.org/lab/mvmlab does Goal.
var Interpreters = map[string]func(out io.Writer) (Interpreter, error){
	"Go": func(out io.Writer) (Interpreter, error) { return NewGo(out) },
}

// Interpreter is an interpreter of a language, which runs code.
type Interpreter interface {

	// Eval runs the given code.
	Eval(src string) (reflect.Value, error)
}

// DotPackage is the import path of the package of the names that code can
// use without a package: b, the parent widget of the code, and the names of
// [content.CodeSymbols]. Its symbols are [DotValues].
const DotPackage = "cogentcore.org/core/mvmcore/dot"

var (
	autoPlanNameCounter uint64

	// interpreterParent is the parent widget of the code, b, which is
	// updated before the code runs.
	interpreterParent = new(*core.Frame)

	// interpOutput is the output of the code. It is global because the
	// Goal interpreter of a page is used for all of its code blocks.
	interpOutput bytes.Buffer

	// currentGoal is the Goal interpreter of the current page, which runs
	// all of its Goal code blocks, so that they share their variables.
	currentGoal Interpreter
)

func init() {
	content.BindTextEditor = BindTextEditor
	content.NewPageInitFunc = ResetGoalInterpreter
}

// Load adds the symbols of [basesymbols] and [coresymbols] to their Values,
// once, with tree.AutoPlanName replaced by one that works for interpreted
// code (see [autoPlanName]). It must be called before importing them.
func Load() {
	basesymbols.Load()
	coresymbols.Load()
	coresymbols.Values["cogentcore.org/core/tree"]["AutoPlanName"] = reflect.ValueOf(autoPlanName)
}

// Go is an mvm interpreter of Go code.
type Go struct {
	*interp.Interp
}

// NewGo returns a new interpreter of Go code, writing its output to out,
// with the symbols of [basesymbols.Values], [coresymbols.Values], the mvm
// core standard library and [DotPackage] imported, and with all of these
// packages usable without an import.
func NewGo(out io.Writer) (*Go, error) {
	Load()
	in := interp.NewInterpreter(golang.GoSpec)
	in.SetIO(strings.NewReader(""), out, out)
	basesymbols.Import(in, stdlib.Values, basesymbols.Values, coresymbols.Values, map[string]map[string]reflect.Value{DotPackage: DotValues()})
	return &Go{in}, nil
}

// DotValues returns the values of the names of [DotPackage].
func DotValues() map[string]reflect.Value {
	values := map[string]reflect.Value{
		"b": reflect.ValueOf(interpreterParent),
	}
	// Each is bound as a variable holding the value, a pointer to it, as mvm
	// binds package variables: a pointer value bound as itself would be
	// taken for a variable of its element type.
	for name, v := range content.CodeSymbols {
		rv := reflect.New(reflect.TypeOf(v))
		rv.Elem().Set(reflect.ValueOf(v))
		values[name] = rv
	}
	return values
}

// Eval runs the given Go code. Code without a main function is run in one,
// so that it can be a list of statements, which then also handles type
// declarations correctly; code with a main function, with declarations
// outside of it, runs it. The names of [DotPackage] are imported in the
// same source, as each Eval makes a new main function.
func (g *Go) Eval(src string) (reflect.Value, error) {
	lines := 1 // added before the code
	if !strings.Contains(src, "func main()") {
		src = "func main() {\n" + src + "\n}"
		lines++
	}
	v, err := g.Interp.Eval("code", fmt.Sprintf("import . %q\n", DotPackage)+src)
	if err == nil {
		return v, nil
	}
	if d, ok := g.Diagnose(err); ok && d.Source == "code" && d.Line > lines {
		return v, &Error{Line: d.Line - lines, Col: d.Col, Msg: d.Msg, Err: err}
	}
	return v, err
}

// Error is an error of Go code at a position in it, which
// content.ShowCodeErrors shows there.
type Error struct {

	// Line and Col are the 1-based line and column (in bytes) of the error.
	Line, Col int

	// Msg is the message of the error, without its position.
	Msg string

	// Err is the error of the interpreter.
	Err error
}

func (e *Error) Error() string { return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, e.Msg) }

func (e *Error) Unwrap() error { return e.Err }

// CodePos returns the 1-based line and column of the error in the code.
func (e *Error) CodePos() (line, col int) { return e.Line, e.Col }

// CodeMessage returns the message of the error, without its position.
func (e *Error) CodeMessage() string { return e.Msg }

// ResetGoalInterpreter makes the next Goal code block use a new Goal
// interpreter. It is the value of [content.NewPageInitFunc], so that each
// page has its own.
func ResetGoalInterpreter() {
	currentGoal = nil
}

// getInterpreter returns the interpreter for the given language: a new one,
// except for Goal, whose interpreter of the current page is used for all of
// its code blocks. new is whether it is a new one.
func getInterpreter(language string) (in Interpreter, new bool, err error) {
	if language == "Goal" && currentGoal != nil {
		return currentGoal, false, nil
	}
	f := Interpreters[language]
	if f == nil {
		return nil, false, fmt.Errorf("no entry in mvmcore.Interpreters for language %q", language)
	}
	in, err = f(&interpOutput)
	if err != nil {
		return nil, false, fmt.Errorf("making the %s interpreter: %w", language, err)
	}
	if language == "Goal" {
		currentGoal = in
	}
	return in, true, nil
}

// replayGoal runs in the new Goal interpreter in the Goal code blocks of the
// page that ran compiled before ([content.GoalReplay]), so that it has the
// variables that they made.
func replayGoal(in Interpreter) {
	replay := content.GoalReplay
	content.GoalReplay = nil
	for _, src := range replay {
		interpOutput.Reset()
		if _, err := in.Eval(src); err != nil {
			errors.Log(fmt.Errorf("mvmcore: running the compiled Goal code of the page again: %w", err))
		}
	}
	interpOutput.Reset()
}

// BindTextEditor binds the given text editor to an mvm interpreter such that
// the contents of the text editor are interpreted as code of the given
// language, which is run in the context of the given parent widget. It is
// the value of [content.BindTextEditor] when this package is imported.
func BindTextEditor(ed *textcore.Editor, parent *core.Frame, language string) {
	oc := func() {
		in, new, err := getInterpreter(language)
		if err != nil {
			core.ErrorSnackbar(ed, err)
			return
		}
		SetParent(parent)
		// a program that runs a window adds it to parent, but only while it runs
		defer func() { core.ExternalParent = nil }()
		if new && language == "Goal" {
			replayGoal(in)
		}
		parent.DeleteChildren()

		interpOutput.Reset()
		_, err = in.Eval(ed.Lines.String())
		if ostr := interpOutput.String(); len(ostr) > 0 {
			out := textcore.NewEditor(parent)
			out.SetReadOnly(true)
			out.Lines.Settings.LineNumbers = false
			out.Lines.SetString(ostr)
		}
		content.ShowCodeErrors(ed, parent, err)
		parent.Update()
	}
	ed.OnChange(func(e events.Event) { oc() })
	oc()
}

// SetParent sets the parent widget of the code that runs next, b, in which
// the widgets that it makes are added (see [core.ExternalParent]).
func SetParent(parent *core.Frame) {
	core.ExternalParent = parent
	*interpreterParent = parent
}

// autoPlanName returns a unique name for each tree.AddChild call of the
// code, replacing tree.AutoPlanName, whose name from the call stack would
// not tell apart the calls of interpreted code.
func autoPlanName(int) string {
	return fmt.Sprintf("mvm-%v", atomic.AddUint64(&autoPlanNameCounter, 1))
}
