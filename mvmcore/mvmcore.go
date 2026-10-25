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

	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/mvmcore/basesymbols"
	"cogentcore.org/core/mvmcore/coresymbols"
	"cogentcore.org/core/text/textcore"
	"github.com/cogentcore/mvm/goparser"
	"github.com/cogentcore/mvm/interp"
	"github.com/cogentcore/mvm/lang/golang"
	"github.com/cogentcore/mvm/stdlib"
	_ "github.com/cogentcore/mvm/stdlib/core" // the core standard library
	"github.com/cogentcore/mvm/symbol"
)

// Interpreters is a map from language names (such as "Go") to functions that
// make a new [Interpreter] for that language, writing its output to out. The
// base implementation is an mvm interpreter for Go (see [NewGo]), but other
// packages can add other languages.
var Interpreters = map[string]func(out io.Writer) (Interpreter, error){
	"Go": func(out io.Writer) (Interpreter, error) { return NewGo(out) },
}

// Interpreter is an interpreter of a language, which runs code.
type Interpreter interface {

	// Eval runs the given code.
	Eval(src string) (reflect.Value, error)
}

// dotPackage is the import path of the package of the names that code can
// use without a package: b, the parent widget of the code, and the names of
// [content.CodeSymbols].
const dotPackage = "cogentcore.org/core/mvmcore/dot"

var (
	autoPlanNameCounter uint64

	// interpreterParent is the parent widget of the code, b, which is
	// updated before the code runs.
	interpreterParent = new(*core.Frame)

	// interpOutput is the output of the code.
	interpOutput bytes.Buffer
)

// Go is an mvm interpreter of Go code.
type Go struct {
	*interp.Interp
}

// NewGo returns a new interpreter of Go code, writing its output to out,
// with the symbols of [basesymbols.Values], [coresymbols.Values] and the mvm
// core standard library imported, and with all of these packages usable
// without an import.
func NewGo(out io.Writer) (*Go, error) {
	basesymbols.Load()
	coresymbols.Load()
	coresymbols.Values["cogentcore.org/core/tree"]["AutoPlanName"] = reflect.ValueOf(autoPlanName)
	in := interp.NewInterpreter(golang.GoSpec)
	if stdlib.EmbeddedStd() == nil {
		in.SetStdlibFS(basesymbols.StdSource())
	}
	in.SetIO(strings.NewReader(""), out, out)
	in.ImportPackageValues(stdlib.Values)
	in.ImportPackageValues(basesymbols.Values)
	in.ImportPackageValues(coresymbols.Values)
	in.ImportPackageValues(map[string]map[string]reflect.Value{dotPackage: dotValues()})
	in.AutoImportPackages()

	// The Cogent Core packages take the names that they share with standard
	// library packages, as errors, whose functions base/errors also has.
	for _, values := range []map[string]map[string]reflect.Value{basesymbols.Values, coresymbols.Values} {
		for path := range values {
			if !strings.HasPrefix(path, "cogentcore.org/") {
				continue
			}
			name := goparser.PackageName(path)
			in.SymSet(name, &symbol.Symbol{Kind: symbol.Pkg, PkgPath: path, Index: symbol.UnsetAddr, Name: name, AutoImport: true})
		}
	}
	return &Go{in}, nil
}

// dotValues returns the values of the names that code can use without a
// package (see [dotPackage]).
func dotValues() map[string]reflect.Value {
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
// outside of it, runs it. The names of [dotPackage] are imported in the
// same source: importing them in an Eval of their own makes mvm fail to
// run the next one.
func (g *Go) Eval(src string) (reflect.Value, error) {
	if !strings.Contains(src, "func main()") {
		src = "func main() {\n" + src + "\n}"
	}
	return g.Interp.Eval("code", fmt.Sprintf("import . %q\n", dotPackage)+src)
}

func init() {
	content.BindTextEditor = BindTextEditor
}

// BindTextEditor binds the given text editor to an mvm interpreter such that
// the contents of the text editor are interpreted as code of the given
// language, which is run in the context of the given parent widget. It is
// the value of [content.BindTextEditor] when this package is imported.
func BindTextEditor(ed *textcore.Editor, parent *core.Frame, language string) {
	oc := func() {
		f := Interpreters[language]
		if f == nil {
			core.ErrorSnackbar(ed, fmt.Errorf("no entry in mvmcore.Interpreters for language %q", language))
			return
		}
		in, err := f(&interpOutput)
		if err != nil {
			core.ErrorSnackbar(ed, err, fmt.Sprintf("Error making the %s interpreter", language))
			return
		}
		core.ExternalParent = parent
		*interpreterParent = parent
		parent.DeleteChildren()

		interpOutput.Reset()
		_, err = in.Eval(ed.Lines.String())
		if ostr := interpOutput.String(); len(ostr) > 0 {
			out := textcore.NewEditor(parent)
			out.SetReadOnly(true)
			out.Lines.Settings.LineNumbers = false
			out.Lines.SetString(ostr)
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

// autoPlanName returns a unique name for each tree.AddChild call of the
// code, replacing tree.AutoPlanName, whose name from the call stack would
// not tell apart the calls of interpreted code.
func autoPlanName(int) string {
	return fmt.Sprintf("mvm-%v", atomic.AddUint64(&autoPlanNameCounter, 1))
}
