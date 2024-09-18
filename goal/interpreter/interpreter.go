// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/goal"
	"cogentcore.org/core/yaegicore/symbols"
	"github.com/cogentcore/yaegi/interp"
	"github.com/cogentcore/yaegi/stdlib"
	"github.com/ergochat/readline"
)

// Interpreter represents one running shell context
type Interpreter struct {
	// the goal shell
	Goal *goal.Goal

	// HistFile is the name of the history file to open / save.
	// Defaults to ~/.goal-history for the default goal shell.
	// Update this prior to running Config() to take effect.
	HistFile string

	// the yaegi interpreter
	Interp *interp.Interpreter
}

func init() {
	delete(stdlib.Symbols, "errors/errors") // use our errors package instead
}

// NewInterpreter returns a new [Interpreter] initialized with the given options.
// It automatically imports the standard library and configures necessary shell
// functions. End user app must call [Interp.Config] after importing any additional
// symbols, prior to running the interpreter.
func NewInterpreter(options interp.Options) *Interpreter {
	in := &Interpreter{HistFile: "~/.goal-history"}
	in.Goal = goal.NewGoal()
	if options.Stdin != nil {
		in.Goal.Config.StdIO.In = options.Stdin
	}
	if options.Stdout != nil {
		in.Goal.Config.StdIO.Out = options.Stdout
	}
	if options.Stderr != nil {
		in.Goal.Config.StdIO.Err = options.Stderr
	}
	in.Goal.SaveOrigStdIO()
	options.Stdout = in.Goal.StdIOWrappers.Out
	options.Stderr = in.Goal.StdIOWrappers.Err
	options.Stdin = in.Goal.StdIOWrappers.In
	in.Interp = interp.New(options)
	// errors.Log(in.Interp.Use(stdlib.Symbols)) // note: yaegicore/symbols gets just a few key things
	errors.Log(in.Interp.Use(symbols.Symbols))
	in.ImportGoal()
	go in.MonitorSignals()
	return in
}

// Prompt returns the appropriate REPL prompt to show the user.
func (in *Interpreter) Prompt() string {
	dp := in.Goal.TotalDepth()
	if dp == 0 {
		return in.Goal.HostAndDir() + " > "
	}
	res := "> "
	for range dp {
		res += "    " // note: /t confuses readline
	}
	return res
}

// Eval evaluates (interprets) the given code,
// returning the value returned from the interpreter.
// HasPrint indicates whether the last line of code
// has the string print in it, which is for determining
// whether to print the result in interactive mode.
// It automatically logs any error in addition to returning it.
func (in *Interpreter) Eval(code string) (v reflect.Value, hasPrint bool, err error) {
	in.Goal.TranspileCode(code)
	source := false
	if in.Goal.SSHActive == "" {
		source = strings.HasPrefix(code, "source")
	}
	if in.Goal.TotalDepth() == 0 {
		nl := len(in.Goal.Lines)
		if nl > 0 {
			ln := in.Goal.Lines[nl-1]
			if strings.Contains(strings.ToLower(ln), "print") {
				hasPrint = true
			}
		}
		v, err = in.RunCode()
		in.Goal.Errors = nil
	}
	if source {
		v, err = in.RunCode() // run accumulated code
	}
	return
}

// RunCode runs the accumulated set of code lines
// and clears the stack of code lines.
// It automatically logs any error in addition to returning it.
func (in *Interpreter) RunCode() (reflect.Value, error) {
	if len(in.Goal.Errors) > 0 {
		return reflect.Value{}, errors.Join(in.Goal.Errors...)
	}
	in.Goal.AddChunk()
	code := in.Goal.Chunks
	in.Goal.ResetCode()
	var v reflect.Value
	var err error
	for _, ch := range code {
		ctx := in.Goal.StartContext()
		v, err = in.Interp.EvalWithContext(ctx, ch)
		in.Goal.EndContext()
		if err != nil {
			cancelled := errors.Is(err, context.Canceled)
			// fmt.Println("cancelled:", cancelled)
			in.Goal.RestoreOrigStdIO()
			in.Goal.ResetDepth()
			if !cancelled {
				in.Goal.AddError(err)
			} else {
				in.Goal.Errors = nil
			}
			break
		}
	}
	return v, err
}

// RunConfig runs the .goal startup config file in the user's
// home directory if it exists.
func (in *Interpreter) RunConfig() error {
	err := in.Goal.TranspileConfig()
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
		in.Goal.CancelExecution()
	}
}

// Config performs final configuration after all the imports have been Use'd
func (in *Interpreter) Config() {
	in.Interp.ImportUsed()
	in.RunConfig()
}

// OpenHistory opens history from the current HistFile
// and loads it into the readline history for given rl instance
func (in *Interpreter) OpenHistory(rl *readline.Instance) error {
	err := in.Goal.OpenHistory(in.HistFile)
	if err == nil {
		for _, h := range in.Goal.Hist {
			rl.SaveToHistory(h)
		}
	}
	return err
}

// SaveHistory saves last 500 (or HISTFILESIZE env value) lines of history,
// to the current HistFile.
func (in *Interpreter) SaveHistory() error {
	n := 500
	if hfs := os.Getenv("HISTFILESIZE"); hfs != "" {
		en, err := strconv.Atoi(hfs)
		if err != nil {
			in.Goal.Config.StdIO.ErrPrintf("SaveHistory: environment variable HISTFILESIZE: %q not a number: %s", hfs, err.Error())
		} else {
			n = en
		}
	}
	return in.Goal.SaveHistory(n, in.HistFile)
}

// Interactive runs an interactive shell that allows the user to input goal.
// Must have done in.Config() prior to calling.
func (in *Interpreter) Interactive() error {
	rl, err := readline.NewFromConfig(&readline.Config{
		AutoComplete: &goal.ReadlineCompleter{Goal: in.Goal},
		Undo:         true,
	})
	if err != nil {
		return err
	}
	in.OpenHistory(rl)
	defer rl.Close()
	log.SetOutput(rl.Stderr()) // redraw the prompt correctly after log output

	for {
		rl.SetPrompt(in.Prompt())
		line, err := rl.ReadLine()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if errors.Is(err, io.EOF) {
			in.SaveHistory()
			os.Exit(0)
		}
		if err != nil {
			in.SaveHistory()
			return err
		}
		if len(line) > 0 && line[0] == '!' { // history command
			hl, err := strconv.Atoi(line[1:])
			nh := len(in.Goal.Hist)
			if err != nil {
				in.Goal.Config.StdIO.ErrPrintf("history number: %q not a number: %s", line[1:], err.Error())
				line = ""
			} else if hl >= nh {
				in.Goal.Config.StdIO.ErrPrintf("history number: %d not in range: [0:%d]", hl, nh)
				line = ""
			} else {
				line = in.Goal.Hist[hl]
				fmt.Printf("h:%d\t%s\n", hl, line)
			}
		} else if line != "" && !strings.HasPrefix(line, "history") && line != "h" {
			in.Goal.AddHistory(line)
		}
		in.Goal.Errors = nil
		v, hasPrint, err := in.Eval(line)
		if err == nil && !hasPrint && v.IsValid() && !v.IsZero() && v.Kind() != reflect.Func {
			fmt.Println(v.Interface())
		}
	}
}
