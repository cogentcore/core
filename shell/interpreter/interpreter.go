// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import (
	"log/slog"
	"reflect"

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
}

// NewInterpreter returns a new [Interpreter] initialized with the given options.
func NewInterpreter(options interp.Options) *Interpreter {
	in := &Interpreter{}
	in.Interp = interp.New(options)
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

// Prompt returns the appropriate REPL prompt to show the user.
func (in *Interpreter) Prompt() string {
	dp := in.Shell.TotalDepth()
	res := "> "
	for range dp {
		res += "\t"
	}
	return res
}

// Eval evaluates (interprets) the given code.
func (in *Interpreter) Eval(code string) error {
	in.Shell.TranspileCode(code)
	if in.Shell.TotalDepth() == 0 {
		return in.RunCode()
	}
	return nil
}

// RunCode runs the accumulated set of code lines
// and clears the stack.
func (in *Interpreter) RunCode() error {
	cmd := in.Shell.Code()
	in.Shell.ResetLines()
	_, err := in.Interp.Eval(cmd)
	if err != nil {
		slog.Error(err.Error())
	}
	return err
}
