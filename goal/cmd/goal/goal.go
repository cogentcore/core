// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command cosh is an interactive cli for running and compiling Cogent Shell (cosh).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/cli"
	"cogentcore.org/core/shell"
	"cogentcore.org/core/shell/interpreter"
	"github.com/cogentcore/yaegi/interp"
)

//go:generate core generate -add-types -add-funcs

// Config is the configuration information for the cosh cli.
type Config struct {

	// Input is the input file to run/compile.
	// If this is provided as the first argument,
	// then the program will exit after running,
	// unless the Interactive mode is flagged.
	Input string `posarg:"0" required:"-"`

	// Expr is an optional expression to evaluate, which can be used
	// in addition to the Input file to run, to execute commands
	// defined within that file for example, or as a command to run
	// prior to starting interactive mode if no Input is specified.
	Expr string `flag:"e,expr"`

	// Args is an optional list of arguments to pass in the run command.
	// These arguments will be turned into an "args" local variable in the shell.
	// These are automatically processed from any leftover arguments passed, so
	// you should not need to specify this flag manually.
	Args []string `cmd:"run" posarg:"leftover" required:"-"`

	// Interactive runs the interactive command line after processing any input file.
	// Interactive mode is the default mode for the run command unless an input file
	// is specified.
	Interactive bool `cmd:"run" flag:"i,interactive"`
}

func main() { //types:skip
	opts := cli.DefaultOptions("cosh", "An interactive tool for running and compiling Cogent Shell (cosh).")
	cli.Run(opts, &Config{}, Run, Build)
}

// Run runs the specified cosh file. If no file is specified,
// it runs an interactive shell that allows the user to input cosh.
func Run(c *Config) error { //cli:cmd -root
	in := interpreter.NewInterpreter(interp.Options{})
	in.Config()
	if len(c.Args) > 0 {
		in.Eval("args := cosh.StringsToAnys(" + fmt.Sprintf("%#v)", c.Args))
	}

	if c.Input == "" {
		return Interactive(c, in)
	}
	code := ""
	if errors.Log1(fsx.FileExists(c.Input)) {
		b, err := os.ReadFile(c.Input)
		if err != nil && c.Expr == "" {
			return err
		}
		code = string(b)
	}
	if c.Expr != "" {
		if code != "" {
			code += "\n"
		}
		code += c.Expr + "\n"
	}

	_, _, err := in.Eval(code)
	if err == nil {
		err = in.Shell.DepthError()
	}
	if c.Interactive {
		return Interactive(c, in)
	}
	return err
}

// Interactive runs an interactive shell that allows the user to input cosh.
func Interactive(c *Config, in *interpreter.Interpreter) error {
	if c.Expr != "" {
		in.Eval(c.Expr)
	}
	in.Interactive()
	return nil
}

// Build builds the specified input cosh file, or all .cosh files in the current
// directory if no input is specified, to corresponding .go file name(s).
// If the file does not already contain a "package" specification, then
// "package main; func main()..." wrappers are added, which allows the same
// code to be used in interactive and Go compiled modes.
func Build(c *Config) error {
	var fns []string
	if c.Input != "" {
		fns = []string{c.Input}
	} else {
		fns = fsx.Filenames(".", ".cosh")
	}
	var errs []error
	for _, fn := range fns {
		ofn := strings.TrimSuffix(fn, filepath.Ext(fn)) + ".go"
		err := shell.NewShell().TranspileFile(fn, ofn)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
