// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import (
	"log/slog"
	"path/filepath"
	"reflect"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/shell"
	"github.com/mitchellh/go-homedir"
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
	in.Shell = shell.NewShell()
	if options.Stdin != nil {
		in.Shell.Config.Stdin = options.Stdin
	}
	if options.Stdout != nil {
		in.Shell.Config.Stdout = options.Stdout
	}
	if options.Stderr != nil {
		in.Shell.Config.Stderr = options.Stderr
	}
	in.Interp = interp.New(options)
	in.Interp.Use(stdlib.Symbols)
	in.Interp.Use(interp.Exports{
		"cogentcore.org/core/shell/shell": map[string]reflect.Value{
			"Exec":   reflect.ValueOf(in.Shell.Exec),
			"Output": reflect.ValueOf(in.Shell.Output),
		},
	})
	in.Interp.ImportUsed()
	return in
}

// Prompt returns the appropriate REPL prompt to show the user.
func (in *Interpreter) Prompt() string {
	dp := in.Shell.TotalDepth()
	if dp == 0 {
		hdir := errors.Log1(homedir.Dir())
		rel := errors.Log1(filepath.Rel(hdir, in.Shell.Config.Dir))
		return filepath.Join("~", rel) + " > "
	}
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
