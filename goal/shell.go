// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package shell provides the Cogent Shell (cosh), which combines the best parts
// of Go and bash to provide an integrated shell experience that allows you to
// easily run terminal commands while using Go for complicated logic.
package shell

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/base/num"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/sshclient"
	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/base/stringsx"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/tools/imports"
)

// Shell represents one running shell context.
type Shell struct {

	// Config is the [exec.Config] used to run commands.
	Config exec.Config

	// StdIOWrappers are IO wrappers sent to the interpreter, so we can
	// control the IO streams used within the interpreter.
	// Call SetWrappers on this with another StdIO object to update settings.
	StdIOWrappers exec.StdIO

	// ssh connection, configuration
	SSH *sshclient.Config

	// collection of ssh clients
	SSHClients map[string]*sshclient.Client

	// SSHActive is the name of the active SSH client
	SSHActive string

	// depth of delim at the end of the current line. if 0, was complete.
	ParenDepth, BraceDepth, BrackDepth, TypeDepth, DeclDepth int

	// Chunks of code lines that are accumulated during Transpile,
	// each of which should be evaluated separately, to avoid
	// issues with contextual effects from import, package etc.
	Chunks []string

	// current stack of transpiled lines, that are accumulated into
	// code Chunks
	Lines []string

	// stack of runtime errors
	Errors []error

	// Builtins are all the builtin shell commands
	Builtins map[string]func(cmdIO *exec.CmdIO, args ...string) error

	// commands that have been defined, which can be run in Exec mode.
	Commands map[string]func(args ...string)

	// Jobs is a stack of commands running in the background
	// (via Start instead of Run)
	Jobs stack.Stack[*exec.CmdIO]

	// Cancel, while the interpreter is running, can be called
	// to stop the code interpreting.
	// It is connected to the Ctx context, by StartContext()
	// Both can be nil.
	Cancel func()

	// Ctx is the context used for cancelling current shell running
	// a single chunk of code, typically from the interpreter.
	// We are not able to pass the context around so it is set here,
	// in the StartContext function. Clear when done with ClearContext.
	Ctx context.Context

	// original standard IO setings, to restore
	OrigStdIO exec.StdIO

	// Hist is the accumulated list of command-line input,
	// which is displayed with the history builtin command,
	// and saved / restored from ~/.coshhist file
	Hist []string

	// FuncToVar translates function definitions into variable definitions,
	// which is the default for interactive use of random code fragments
	// without the complete go formatting.
	// For pure transpiling of a complete codebase with full proper Go formatting
	// this should be turned off.
	FuncToVar bool

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
	sh.FuncToVar = true
	sh.Config.StdIO.SetFromOS()
	sh.SSH = sshclient.NewConfig(&sh.Config)
	sh.SSHClients = make(map[string]*sshclient.Client)
	sh.Commands = make(map[string]func(args ...string))
	sh.InstallBuiltins()
	return sh
}

// StartContext starts a processing context,
// setting the Ctx and Cancel Fields.
// Call EndContext when current operation finishes.
func (sh *Shell) StartContext() context.Context {
	sh.Ctx, sh.Cancel = context.WithCancel(context.Background())
	return sh.Ctx
}

// EndContext ends a processing context, clearing the
// Ctx and Cancel fields.
func (sh *Shell) EndContext() {
	sh.Ctx = nil
	sh.Cancel = nil
}

// SaveOrigStdIO saves the current Config.StdIO as the original to revert to
// after an error, and sets the StdIOWrappers to use them.
func (sh *Shell) SaveOrigStdIO() {
	sh.OrigStdIO = sh.Config.StdIO
	sh.StdIOWrappers.NewWrappers(&sh.OrigStdIO)
}

// RestoreOrigStdIO reverts to using the saved OrigStdIO
func (sh *Shell) RestoreOrigStdIO() {
	sh.Config.StdIO = sh.OrigStdIO
	sh.OrigStdIO.SetToOS()
	sh.StdIOWrappers.SetWrappers(&sh.OrigStdIO)
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

// Host returns the name we're running commands on,
// which is empty if localhost (default).
func (sh *Shell) Host() string {
	cl := sh.ActiveSSH()
	if cl == nil {
		return ""
	}
	return "@" + sh.SSHActive + ":" + cl.Host
}

// HostAndDir returns the name we're running commands on,
// which is empty if localhost (default),
// and the current directory on that host.
func (sh *Shell) HostAndDir() string {
	host := ""
	dir := sh.Config.Dir
	home := errors.Log1(homedir.Dir())
	cl := sh.ActiveSSH()
	if cl != nil {
		host = "@" + sh.SSHActive + ":" + cl.Host + ":"
		dir = cl.Dir
		home = cl.HomeDir
	}
	rel := errors.Log1(filepath.Rel(home, dir))
	// if it has to go back, then it is not in home dir, so no ~
	if strings.Contains(rel, "..") {
		return host + dir + string(filepath.Separator)
	}
	return host + filepath.Join("~", rel) + string(filepath.Separator)
}

// SSHByHost returns the SSH client for given host name, with err if not found
func (sh *Shell) SSHByHost(host string) (*sshclient.Client, error) {
	if scl, ok := sh.SSHClients[host]; ok {
		return scl, nil
	}
	return nil, fmt.Errorf("ssh connection named: %q not found", host)
}

// TotalDepth returns the sum of any unresolved paren, brace, or bracket depths.
func (sh *Shell) TotalDepth() int {
	return num.Abs(sh.ParenDepth) + num.Abs(sh.BraceDepth) + num.Abs(sh.BrackDepth)
}

// ResetCode resets the stack of transpiled code
func (sh *Shell) ResetCode() {
	sh.Chunks = nil
	sh.Lines = nil
}

// ResetDepth resets the current depths to 0
func (sh *Shell) ResetDepth() {
	sh.ParenDepth, sh.BraceDepth, sh.BrackDepth, sh.TypeDepth, sh.DeclDepth = 0, 0, 0, 0, 0
}

// DepthError reports an error if any of the parsing depths are not zero,
// to be called at the end of transpiling a complete block of code.
func (sh *Shell) DepthError() error {
	if sh.TotalDepth() == 0 {
		return nil
	}
	str := ""
	if sh.ParenDepth != 0 {
		str += fmt.Sprintf("Incomplete parentheses (), remaining depth: %d\n", sh.ParenDepth)
	}
	if sh.BraceDepth != 0 {
		str += fmt.Sprintf("Incomplete braces [], remaining depth: %d\n", sh.BraceDepth)
	}
	if sh.BrackDepth != 0 {
		str += fmt.Sprintf("Incomplete brackets {}, remaining depth: %d\n", sh.BrackDepth)
	}
	if str != "" {
		slog.Error(str)
		return errors.New(str)
	}
	return nil
}

// AddLine adds line on the stack
func (sh *Shell) AddLine(ln string) {
	sh.Lines = append(sh.Lines, ln)
}

// Code returns the current transpiled lines,
// split into chunks that should be compiled separately.
func (sh *Shell) Code() string {
	sh.AddChunk()
	if len(sh.Chunks) == 0 {
		return ""
	}
	return strings.Join(sh.Chunks, "\n")
}

// AddChunk adds current lines into a chunk of code
// that should be compiled separately.
func (sh *Shell) AddChunk() {
	if len(sh.Lines) == 0 {
		return
	}
	sh.Chunks = append(sh.Chunks, strings.Join(sh.Lines, "\n"))
	sh.Lines = nil
}

// TranspileCode processes each line of given code,
// adding the results to the LineStack
func (sh *Shell) TranspileCode(code string) {
	lns := strings.Split(code, "\n")
	n := len(lns)
	if n == 0 {
		return
	}
	for _, ln := range lns {
		hasDecl := sh.DeclDepth > 0
		tl := sh.TranspileLine(ln)
		sh.AddLine(tl)
		if sh.BraceDepth == 0 && sh.BrackDepth == 0 && sh.ParenDepth == 1 && sh.lastCommand != "" {
			sh.lastCommand = ""
			nl := len(sh.Lines)
			sh.Lines[nl-1] = sh.Lines[nl-1] + ")"
			sh.ParenDepth--
		}
		if hasDecl && sh.DeclDepth == 0 { // break at decl
			sh.AddChunk()
		}
	}
}

// TranspileCodeFromFile transpiles the code in given file
func (sh *Shell) TranspileCodeFromFile(file string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	sh.TranspileCode(string(b))
	return nil
}

// TranspileFile transpiles the given input cosh file to the
// given output Go file. If no existing package declaration
// is found, then package main and func main declarations are
// added. This also affects how functions are interpreted.
func (sh *Shell) TranspileFile(in string, out string) error {
	b, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	code := string(b)
	lns := stringsx.SplitLines(code)
	hasPackage := false
	for _, ln := range lns {
		if strings.HasPrefix(ln, "package ") {
			hasPackage = true
			break
		}
	}
	if hasPackage {
		sh.FuncToVar = false // use raw functions
	}
	sh.TranspileCode(code)
	sh.FuncToVar = true
	if err != nil {
		return err
	}
	gen := "// Code generated by \"cosh build\"; DO NOT EDIT.\n\n"
	if hasPackage {
		sh.Lines = slices.Insert(sh.Lines, 0, gen)
	} else {
		sh.Lines = slices.Insert(sh.Lines, 0, gen, "package main", "", "func main() {", "shell := shell.NewShell()")
		sh.Lines = append(sh.Lines, "}")
	}
	src := []byte(sh.Code())
	res, err := imports.Process(out, src, nil)
	if err != nil {
		res = src
		slog.Error(err.Error())
	} else {
		err = sh.DepthError()
	}
	werr := os.WriteFile(out, res, 0666)
	return errors.Join(err, werr)
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

// AddHistory adds given line to the Hist record of commands
func (sh *Shell) AddHistory(line string) {
	sh.Hist = append(sh.Hist, line)
}

// SaveHistory saves up to the given number of lines of current history
// to given file, e.g., ~/.coshhist for the default cosh program.
// If n is <= 0 all lines are saved.  n is typically 500 by default.
func (sh *Shell) SaveHistory(n int, file string) error {
	path, err := homedir.Expand(file)
	if err != nil {
		return err
	}
	hn := len(sh.Hist)
	sn := hn
	if n > 0 {
		sn = min(n, hn)
	}
	lh := strings.Join(sh.Hist[hn-sn:hn], "\n")
	err = os.WriteFile(path, []byte(lh), 0666)
	if err != nil {
		return err
	}
	return nil
}

// OpenHistory opens Hist history lines from given file,
// e.g., ~/.coshhist
func (sh *Shell) OpenHistory(file string) error {
	path, err := homedir.Expand(file)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sh.Hist = strings.Split(string(b), "\n")
	return nil
}

// AddCommand adds given command to list of available commands
func (sh *Shell) AddCommand(name string, cmd func(args ...string)) {
	sh.Commands[name] = cmd
}

// RunCommands runs the given command(s). This is typically called
// from a Makefile-style cosh script.
func (sh *Shell) RunCommands(cmds []any) error {
	for _, cmd := range cmds {
		if cmdFun, hasCmd := sh.Commands[reflectx.ToString(cmd)]; hasCmd {
			cmdFun()
		} else {
			return errors.Log(fmt.Errorf("command %q not found", cmd))
		}
	}
	return nil
}

// DeleteJob deletes the given job and returns true if successful,
func (sh *Shell) DeleteJob(cmdIO *exec.CmdIO) bool {
	idx := slices.Index(sh.Jobs, cmdIO)
	if idx >= 0 {
		sh.Jobs = slices.Delete(sh.Jobs, idx, idx+1)
		return true
	}
	return false
}

// JobIDExpand expands %n job id values in args with the full PID
// returns number of PIDs expanded
func (sh *Shell) JobIDExpand(args []string) int {
	exp := 0
	for i, id := range args {
		if id[0] == '%' {
			idx, err := strconv.Atoi(id[1:])
			if err == nil {
				if idx > 0 && idx <= len(sh.Jobs) {
					jb := sh.Jobs[idx-1]
					if jb.Cmd != nil && jb.Cmd.Process != nil {
						args[i] = fmt.Sprintf("%d", jb.Cmd.Process.Pid)
						exp++
					}
				} else {
					sh.AddError(fmt.Errorf("cosh: job number out of range: %d", idx))
				}
			}
		}
	}
	return exp
}
