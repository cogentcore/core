// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/shell"
	"github.com/ergochat/readline"
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
// It automatically imports the standard library and configures necessary shell
// functions. End user app must call [Interp.ImportUsed() and [Interpreter.RunConfig].
func NewInterpreter(options interp.Options) *Interpreter {
	in := &Interpreter{}
	in.Shell = shell.NewShell()
	if options.Stdin != nil {
		in.Shell.Config.StdIO.In = options.Stdin
	}
	if options.Stdout != nil {
		in.Shell.Config.StdIO.Out = options.Stdout
	}
	if options.Stderr != nil {
		in.Shell.Config.StdIO.Err = options.Stderr
	}
	in.Shell.StdIOWrappers.NewWrappers(&in.Shell.Config.StdIO)
	options.Stdout = in.Shell.StdIOWrappers.Out
	options.Stderr = in.Shell.StdIOWrappers.Err
	options.Stdin = in.Shell.StdIOWrappers.In
	in.Interp = interp.New(options)
	in.Interp.Use(stdlib.Symbols)

	in.Interp.Use(interp.Exports{
		"cogentcore.org/core/shell/shell": map[string]reflect.Value{
			"Run":         reflect.ValueOf(in.Shell.Run),
			"RunErrOK":    reflect.ValueOf(in.Shell.RunErrOK),
			"Output":      reflect.ValueOf(in.Shell.Output),
			"OutputErrOK": reflect.ValueOf(in.Shell.OutputErrOK),
			"Start":       reflect.ValueOf(in.Shell.Start),
			"AddCommand":  reflect.ValueOf(in.Shell.AddCommand),
		},
	})
	go in.MonitorSignals()
	return in
}

// Prompt returns the appropriate REPL prompt to show the user.
func (in *Interpreter) Prompt() string {
	dp := in.Shell.TotalDepth()
	if dp == 0 {
		return in.Shell.HostAndDir() + " > "
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
	source := strings.HasPrefix(code, "source")
	if in.Shell.TotalDepth() == 0 {
		nl := len(in.Shell.Lines)
		hasPrint := false
		if nl > 0 {
			ln := in.Shell.Lines[nl-1]
			if strings.Contains(strings.ToLower(ln), "print") {
				hasPrint = true
			}
		}
		v, _ := in.RunCode()
		in.Shell.Errors = nil
		if !hasPrint && v.IsValid() && !v.IsZero() && v.Kind() != reflect.Func {
			fmt.Println(v.Interface())
		}
	} else {
		fmt.Println("depth != 0", in.Shell.TotalDepth())
	}
	if source {
		in.RunCode() // run accumulated code
	}
	return nil
}

// RunCode runs the accumulated set of code lines
// and clears the stack of code lines
func (in *Interpreter) RunCode() (reflect.Value, error) {
	if len(in.Shell.Errors) > 0 {
		return reflect.Value{}, errors.Join(in.Shell.Errors...)
	}
	cmd := in.Shell.Code()
	in.Shell.ResetLines()
	ctx := in.Shell.StartContext()
	v, err := in.Interp.EvalWithContext(ctx, cmd)
	in.Shell.EndContext()
	if err != nil && !errors.Is(err, context.Canceled) {
		slog.Error(err.Error())
	}
	return v, err
}

// RunConfig runs the .cosh startup config file in the user's
// home directory if it exists.
func (in *Interpreter) RunConfig() error {
	err := in.Shell.TranspileConfig()
	if err != nil {
		errors.Log(err)
	}
	_, err = in.RunCode()
	return err
}

// MonitorSignals monitors the operating system signals to appropriately
// stop the interpreter and prevent the shell from closing on Control+C.
// It is called automatically in another goroutine in [NewInterpreter].
func (in *Interpreter) MonitorSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for {
		<-c
		in.Shell.CancelExecution()
	}
}

// Config performs final configuration after all the imports have been Use'd
func (in *Interpreter) Config() {
	in.Interp.ImportUsed()
	in.RunConfig()
}

// Interactive runs an interactive shell that allows the user to input cosh.
func (in *Interpreter) Interactive() error {
	in.Config()
	rl, err := readline.NewFromConfig(&readline.Config{
		AutoComplete: &shell.ReadlineCompleter{Shell: in.Shell},
		Undo:         true,
	})
	if err != nil {
		return err
	}
	defer rl.Close()
	log.SetOutput(rl.Stderr()) // redraw the prompt correctly after log output

	for {
		rl.SetPrompt(in.Prompt())
		line, err := rl.ReadLine()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if errors.Is(err, io.EOF) {
			os.Exit(0)
		}
		if err != nil {
			return err
		}
		in.Eval(line)
	}
}
