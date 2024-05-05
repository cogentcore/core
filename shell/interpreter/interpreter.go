// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"cogentcore.org/core/shell"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Interpreter represents one running shell context
type Interpreter struct {
	// the cosh shell
	Shell *shell.Shell

	// the yaegi interpreter
	Interp *interp.Interpreter

	// the stack of un-evaluated lines, based on unfinished delimiters
	LineStack []string
}

func NewInterpreter() *Interpreter {
	in := &Interpreter{}
	in.Interp = interp.New(interp.Options{})
	in.Interp.Use(stdlib.Symbols)
	in.Interp.Use(interp.Exports{
		"cogentcore.org/core/shell/shell": map[string]reflect.Value{
			"Exec":   reflect.ValueOf(shell.Exec),
			"Output": reflect.ValueOf(shell.Output),
		},
	})
	in.Interp.ImportUsed()
	in.Shell = shell.NewShell()
	return in
}

func (in *Interpreter) Prompt() string {
	dp := in.Shell.TotalDepth()
	if dp == 0 {
		return "> "
	}
	return fmt.Sprintf("%d> ", dp)
}

func (in *Interpreter) Eval(ln string) error {
	eln := in.Shell.TranspileLine(ln)
	in.PushLine(eln)
	if in.Shell.TotalDepth() == 0 {
		return in.RunStack()
	}
	return nil
}

// PushLine pushes line on the stack
func (in *Interpreter) PushLine(ln string) {
	in.LineStack = append(in.LineStack, ln)
}

// RunStack runs the stacked set of lines
// and clears the stack.
func (in *Interpreter) RunStack() error {
	cmd := strings.Join(in.LineStack, "\n")
	in.LineStack = nil
	_, err := in.Interp.Eval(cmd)
	if err != nil {
		slog.Error(err.Error())
	}
	return err
}

// SymbolByName returns the reflect.Value for given symbol name
// from the current Globals, Symbols (must call GetSymbols first)
func (in *Interpreter) SymbolByName(name string) (bool, reflect.Value) {
	globs := in.Interp.Globals()
	syms := in.Interp.Symbols("main") // note: cannot use ""

	nmpath := ""
	dotIdx := strings.Index(name, ".")
	if dotIdx > 0 {
		nmpath = name[:dotIdx]
		name = name[dotIdx+1:]
	}
	for path, sy := range syms {
		if nmpath != "" && path != nmpath {
			continue
		}
		for nm, v := range sy {
			if nm == name {
				return true, v
			}
		}
	}

	for nm, v := range globs {
		if nm == name {
			return true, v
		}
	}
	return false, reflect.Value{}
}
