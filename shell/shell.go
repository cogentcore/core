// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package shell provides the Cogent Shell (cosh), which combines the best parts
// of Go and bash to provide an integrated shell experience that allows you to
// easily run terminal commands while using Go for complicated logic.
package shell

import (
	"os"
	"slices"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"golang.org/x/tools/imports"
)

// Shell represents one running shell context.
type Shell struct {

	// Builtins are all the builtin shell commands
	Builtins map[string]func(args ...string) error

	// Config is the [exec.Config] used to run commands.
	Config exec.Config

	// depth of parens at the end of the current line. if 0, was complete.
	ParenDepth int

	// depth of braces at the end of the current line. if 0, was complete.
	BraceDepth int

	// depth of brackets at the end of the current line. if 0, was complete.
	BrackDepth int

	// stack of transpiled lines
	Lines []string

	// stack of runtime errors
	Errors []error
}

// NewShell returns a new [Shell] with default options.
func NewShell() *Shell {
	sh := &Shell{
		Config: exec.Config{
			Dir:    errors.Log1(os.Getwd()),
			Env:    map[string]string{},
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Stdin:  os.Stdin,
		},
	}
	sh.InstallBuiltins()
	return sh
}

// TotalDepth returns the sum of any unresolved paren, brace, or bracket depths.
func (sh *Shell) TotalDepth() int {
	return sh.ParenDepth + sh.BraceDepth + sh.BrackDepth
}

// ResetLines resets the stack of transpiled lines
func (sh *Shell) ResetLines() {
	sh.Lines = nil
}

// AddLine adds line on the stack
func (sh *Shell) AddLine(ln string) {
	sh.Lines = append(sh.Lines, ln)
}

// Code returns the current transpiled lines
func (sh *Shell) Code() string {
	if len(sh.Lines) == 0 {
		return ""
	}
	return strings.Join(sh.Lines, "\n")
}

// TranspileCode processes each line of given code,
// adding the results to the LineStack
func (sh *Shell) TranspileCode(code string) {
	lns := strings.Split(code, "\n")
	for _, ln := range lns {
		tl := sh.TranspileLine(ln)
		sh.AddLine(tl)
	}
}

// TranspileFile transpiles the given input cosh file to the given output Go file,
// adding package main and func main declarations.
func (sh *Shell) TranspileFile(in string, out string) error {
	b, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	sh.TranspileCode(string(b))
	sh.Lines = slices.Insert(sh.Lines, 0, "package main", "", "func main() {", "shell := shell.NewShell()")
	sh.Lines = append(sh.Lines, "}")
	src := []byte(sh.Code())
	res, err := imports.Process(out, src, nil)
	if err != nil {
		return err
	}
	return os.WriteFile(out, res, 0666)
}

// AddError adds the given error to the error stack if it is non-nil.
// This is the main way that shell errors are handled. It also prints
// the error.
func (sh *Shell) AddError(err error) error {
	if err == nil {
		return nil
	}
	sh.Errors = append(sh.Errors, err)
	logx.PrintlnError(err)
	return err
}
