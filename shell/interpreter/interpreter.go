// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

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

	// Cancel, while the interpreter is running, can be called
	// to stop the code interpreting.
	Cancel func()
}

// NewInterpreter returns a new [Interpreter] initialized with the given options.
// It automatically imports the standard library and configures necessary shell
// functions. It also calls [Interpreter.RunConfig].
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
	in.RunConfig()
	go in.MonitorSignals()
	return in
}

// Prompt returns the appropriate REPL prompt to show the user.
func (in *Interpreter) Prompt() string {
	dp := in.Shell.TotalDepth()
	host := in.Shell.Host()
	base := ""
	if host != "" {
		base = host + ":"
	}
	if dp == 0 {
		hdir := errors.Log1(homedir.Dir())
		rel := errors.Log1(filepath.Rel(hdir, in.Shell.Config.Dir))
		// if it has to go back, then it is not in home dir, so no ~
		if strings.Contains(rel, "..") {
			return base + in.Shell.Config.Dir + string(filepath.Separator) + " > "
		}
		return base + filepath.Join("~", rel) + string(filepath.Separator) + " > "
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
	ctx, cancel := context.WithCancel(context.Background())
	in.Cancel = cancel
	cmd := in.Shell.Code()
	in.Shell.ResetLines()
	_, err := in.Interp.EvalWithContext(ctx, cmd)
	in.Cancel = nil
	if err != nil && !errors.Is(err, context.Canceled) {
		slog.Error(err.Error())
	}
	return err
}

// RunConfig runs the .cosh startup config file in the user's
// home directory if it exists.
func (in *Interpreter) RunConfig() error {
	err := in.Shell.TranspileConfig()
	if err != nil {
		return errors.Log(err)
	}
	return in.RunCode()
}

// MonitorSignals monitors the operating system signals to appropriately
// stop the interpreter and prevent the shell from closing on Control+C.
// It is called automatically in another goroutine in [NewInterpreter].
func (in *Interpreter) MonitorSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for {
		<-c
		if in.Cancel != nil {
			in.Cancel()
		}
	}
}
