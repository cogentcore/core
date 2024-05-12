// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package shell provides the Cogent Shell (cosh), which combines the best parts
// of Go and bash to provide an integrated shell experience that allows you to
// easily run terminal commands while using Go for complicated logic.
package shell

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/base/sshclient"
	"cogentcore.org/core/base/stack"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/tools/imports"
)

// Shell represents one running shell context.
type Shell struct {

	// Builtins are all the builtin shell commands
	Builtins map[string]func(args ...string) error

	// Config is the [exec.Config] used to run commands.
	Config exec.Config

	// ssh connection, configuration
	SSH *sshclient.Config

	// collection of ssh clients
	SSHClients map[string]*sshclient.Client

	// SSHActive is the name of the active SSH client
	SSHActive string

	// depth of parens at the end of the current line. if 0, was complete.
	ParenDepth int

	// depth of braces at the end of the current line. if 0, was complete.
	BraceDepth int

	// depth of brackets at the end of the current line. if 0, was complete.
	BrackDepth int

	// stack of transpiled lines, that are accumulated in TranspileCode
	Lines []string

	// stack of runtime errors
	Errors []error

	// commands that have been defined, which can be run in Exec mode.
	Commands map[string]func(args ...string)

	// Jobs is a stack of commands running in the background (via Start instead of Run)
	Jobs stack.Stack[*exec.CmdIO]

	// Cancel, while the interpreter is running, can be called
	// to stop the code interpreting.
	Cancel func()

	// commandArgs is a stack of args passed to a command, used for simplified
	// processing of args expressions.
	commandArgs stack.Stack[[]string]

	// isCommand is a stack of bools indicating whether the _immediate_ run context
	// is a command, which affects the way that args are processed.
	isCommand stack.Stack[bool]

	// if this is non-empty, it is the name of the last command defined.
	// triggers insertion of the AddCommand call to add to list of defined commands.
	lastCommand string
}

// NewShell returns a new [Shell] with default options.
func NewShell() *Shell {
	sh := &Shell{
		Config: exec.Config{
			Dir:    errors.Log1(os.Getwd()),
			Env:    map[string]string{},
			Buffer: false,
		},
	}
	sh.Config.StdIO.StdAll()
	sh.SSH = sshclient.NewConfig(&sh.Config)
	sh.SSHClients = make(map[string]*sshclient.Client)
	sh.Commands = make(map[string]func(args ...string))
	sh.InstallBuiltins()
	return sh
}

// Close closes any resources associated with the shell,
// including terminating any commands that are not running "nohup"
// in the background.
func (sh *Shell) Close() {
	sh.CloseSSH()
	// todo: kill jobs etc
}

// CloseSSH closes all open ssh client connections
func (sh *Shell) CloseSSH() {
	sh.SSHActive = ""
	for _, cl := range sh.SSHClients {
		cl.Close()
	}
	sh.SSHClients = make(map[string]*sshclient.Client)
}

// ActiveSSH returns the active ssh client
func (sh *Shell) ActiveSSH() *sshclient.Client {
	if sh.SSHActive == "" {
		return nil
	}
	return sh.SSHClients[sh.SSHActive]
}

// Host returns the name we're running commands on, for interactive prompt
// this is empty if localhost (default).
func (sh *Shell) Host() string {
	cl := sh.ActiveSSH()
	if cl == nil {
		return ""
	}
	return "@" + sh.SSHActive + ":" + cl.Host
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
		if sh.BraceDepth == 0 && sh.BrackDepth == 0 && sh.ParenDepth == 1 && sh.lastCommand != "" {
			sh.lastCommand = ""
			nl := len(sh.Lines)
			sh.Lines[nl-1] = sh.Lines[nl-1] + ")"
			sh.ParenDepth--
		}
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
	fmt.Println(string(src))
	res, err := imports.Process(out, src, nil)
	if err != nil {
		res = src
		slog.Error(err.Error())
	}
	return os.WriteFile(out, res, 0666)
}

// AddError adds the given error to the error stack if it is non-nil,
// and calls the Cancel function if set, to stop execution.
// This is the main way that shell errors are handled.
// It also prints the error.
func (sh *Shell) AddError(err error) error {
	if err == nil {
		return nil
	}
	sh.Errors = append(sh.Errors, err)
	logx.PrintlnError(err)
	sh.CancelExecution()
	return err
}

// TranspileConfig transpiles the .cosh startup config file in the user's
// home directory if it exists.
func (sh *Shell) TranspileConfig() error {
	path, err := homedir.Expand("~/.cosh")
	if err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	sh.TranspileCode(string(b))
	return nil
}

// AddCommand adds given command to list of available commands
func (sh *Shell) AddCommand(name string, cmd func(args ...string)) {
	sh.Commands[name] = cmd
}
