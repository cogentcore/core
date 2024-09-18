// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package goal provides the Goal Go augmented language transpiler,
// which combines the best parts of Go, bash, and Python to provide
// an integrated shell and numerical expression processing experience.
package goal

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

// Goal represents one running Goal language context.
type Goal struct {

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
	// and saved / restored from ~/.goalhist file
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

// NewGoal returns a new [Goal] with default options.
func NewGoal() *Goal {
	gl := &Goal{
		Config: exec.Config{
			Dir:    errors.Log1(os.Getwd()),
			Env:    map[string]string{},
			Buffer: false,
		},
	}
	gl.FuncToVar = true
	gl.Config.StdIO.SetFromOS()
	gl.SSH = sshclient.NewConfig(&gl.Config)
	gl.SSHClients = make(map[string]*sshclient.Client)
	gl.Commands = make(map[string]func(args ...string))
	gl.InstallBuiltins()
	return gl
}

// StartContext starts a processing context,
// setting the Ctx and Cancel Fields.
// Call EndContext when current operation finishes.
func (gl *Goal) StartContext() context.Context {
	gl.Ctx, gl.Cancel = context.WithCancel(context.Background())
	return gl.Ctx
}

// EndContext ends a processing context, clearing the
// Ctx and Cancel fields.
func (gl *Goal) EndContext() {
	gl.Ctx = nil
	gl.Cancel = nil
}

// SaveOrigStdIO saves the current Config.StdIO as the original to revert to
// after an error, and sets the StdIOWrappers to use them.
func (gl *Goal) SaveOrigStdIO() {
	gl.OrigStdIO = gl.Config.StdIO
	gl.StdIOWrappers.NewWrappers(&gl.OrigStdIO)
}

// RestoreOrigStdIO reverts to using the saved OrigStdIO
func (gl *Goal) RestoreOrigStdIO() {
	gl.Config.StdIO = gl.OrigStdIO
	gl.OrigStdIO.SetToOS()
	gl.StdIOWrappers.SetWrappers(&gl.OrigStdIO)
}

// Close closes any resources associated with the shell,
// including terminating any commands that are not running "nohup"
// in the background.
func (gl *Goal) Close() {
	gl.CloseSSH()
	// todo: kill jobs etc
}

// CloseSSH closes all open ssh client connections
func (gl *Goal) CloseSSH() {
	gl.SSHActive = ""
	for _, cl := range gl.SSHClients {
		cl.Close()
	}
	gl.SSHClients = make(map[string]*sshclient.Client)
}

// ActiveSSH returns the active ssh client
func (gl *Goal) ActiveSSH() *sshclient.Client {
	if gl.SSHActive == "" {
		return nil
	}
	return gl.SSHClients[gl.SSHActive]
}

// Host returns the name we're running commands on,
// which is empty if localhost (default).
func (gl *Goal) Host() string {
	cl := gl.ActiveSSH()
	if cl == nil {
		return ""
	}
	return "@" + gl.SSHActive + ":" + cl.Host
}

// HostAndDir returns the name we're running commands on,
// which is empty if localhost (default),
// and the current directory on that host.
func (gl *Goal) HostAndDir() string {
	host := ""
	dir := gl.Config.Dir
	home := errors.Log1(homedir.Dir())
	cl := gl.ActiveSSH()
	if cl != nil {
		host = "@" + gl.SSHActive + ":" + cl.Host + ":"
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
func (gl *Goal) SSHByHost(host string) (*sshclient.Client, error) {
	if scl, ok := gl.SSHClients[host]; ok {
		return scl, nil
	}
	return nil, fmt.Errorf("ssh connection named: %q not found", host)
}

// TotalDepth returns the sum of any unresolved paren, brace, or bracket depths.
func (gl *Goal) TotalDepth() int {
	return num.Abs(gl.ParenDepth) + num.Abs(gl.BraceDepth) + num.Abs(gl.BrackDepth)
}

// ResetCode resets the stack of transpiled code
func (gl *Goal) ResetCode() {
	gl.Chunks = nil
	gl.Lines = nil
}

// ResetDepth resets the current depths to 0
func (gl *Goal) ResetDepth() {
	gl.ParenDepth, gl.BraceDepth, gl.BrackDepth, gl.TypeDepth, gl.DeclDepth = 0, 0, 0, 0, 0
}

// DepthError reports an error if any of the parsing depths are not zero,
// to be called at the end of transpiling a complete block of code.
func (gl *Goal) DepthError() error {
	if gl.TotalDepth() == 0 {
		return nil
	}
	str := ""
	if gl.ParenDepth != 0 {
		str += fmt.Sprintf("Incomplete parentheses (), remaining depth: %d\n", gl.ParenDepth)
	}
	if gl.BraceDepth != 0 {
		str += fmt.Sprintf("Incomplete braces [], remaining depth: %d\n", gl.BraceDepth)
	}
	if gl.BrackDepth != 0 {
		str += fmt.Sprintf("Incomplete brackets {}, remaining depth: %d\n", gl.BrackDepth)
	}
	if str != "" {
		slog.Error(str)
		return errors.New(str)
	}
	return nil
}

// AddLine adds line on the stack
func (gl *Goal) AddLine(ln string) {
	gl.Lines = append(gl.Lines, ln)
}

// Code returns the current transpiled lines,
// split into chunks that should be compiled separately.
func (gl *Goal) Code() string {
	gl.AddChunk()
	if len(gl.Chunks) == 0 {
		return ""
	}
	return strings.Join(gl.Chunks, "\n")
}

// AddChunk adds current lines into a chunk of code
// that should be compiled separately.
func (gl *Goal) AddChunk() {
	if len(gl.Lines) == 0 {
		return
	}
	gl.Chunks = append(gl.Chunks, strings.Join(gl.Lines, "\n"))
	gl.Lines = nil
}

// TranspileCode processes each line of given code,
// adding the results to the LineStack
func (gl *Goal) TranspileCode(code string) {
	lns := strings.Split(code, "\n")
	n := len(lns)
	if n == 0 {
		return
	}
	for _, ln := range lns {
		hasDecl := gl.DeclDepth > 0
		tl := gl.TranspileLine(ln)
		gl.AddLine(tl)
		if gl.BraceDepth == 0 && gl.BrackDepth == 0 && gl.ParenDepth == 1 && gl.lastCommand != "" {
			gl.lastCommand = ""
			nl := len(gl.Lines)
			gl.Lines[nl-1] = gl.Lines[nl-1] + ")"
			gl.ParenDepth--
		}
		if hasDecl && gl.DeclDepth == 0 { // break at decl
			gl.AddChunk()
		}
	}
}

// TranspileCodeFromFile transpiles the code in given file
func (gl *Goal) TranspileCodeFromFile(file string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	gl.TranspileCode(string(b))
	return nil
}

// TranspileFile transpiles the given input goal file to the
// given output Go file. If no existing package declaration
// is found, then package main and func main declarations are
// added. This also affects how functions are interpreted.
func (gl *Goal) TranspileFile(in string, out string) error {
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
		gl.FuncToVar = false // use raw functions
	}
	gl.TranspileCode(code)
	gl.FuncToVar = true
	if err != nil {
		return err
	}
	gen := "// Code generated by \"goal build\"; DO NOT EDIT.\n\n"
	if hasPackage {
		gl.Lines = slices.Insert(gl.Lines, 0, gen)
	} else {
		gl.Lines = slices.Insert(gl.Lines, 0, gen, "package main", "", "func main() {", "goal := goal.NewGoal()")
		gl.Lines = append(gl.Lines, "}")
	}
	src := []byte(gl.Code())
	res, err := imports.Process(out, src, nil)
	if err != nil {
		res = src
		slog.Error(err.Error())
	} else {
		err = gl.DepthError()
	}
	werr := os.WriteFile(out, res, 0666)
	return errors.Join(err, werr)
}

// AddError adds the given error to the error stack if it is non-nil,
// and calls the Cancel function if set, to stop execution.
// This is the main way that goal errors are handled.
// It also prints the error.
func (gl *Goal) AddError(err error) error {
	if err == nil {
		return nil
	}
	gl.Errors = append(gl.Errors, err)
	logx.PrintlnError(err)
	gl.CancelExecution()
	return err
}

// TranspileConfig transpiles the .goal startup config file in the user's
// home directory if it exists.
func (gl *Goal) TranspileConfig() error {
	path, err := homedir.Expand("~/.goal")
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
	gl.TranspileCode(string(b))
	return nil
}

// AddHistory adds given line to the Hist record of commands
func (gl *Goal) AddHistory(line string) {
	gl.Hist = append(gl.Hist, line)
}

// SaveHistory saves up to the given number of lines of current history
// to given file, e.g., ~/.goalhist for the default goal program.
// If n is <= 0 all lines are saved.  n is typically 500 by default.
func (gl *Goal) SaveHistory(n int, file string) error {
	path, err := homedir.Expand(file)
	if err != nil {
		return err
	}
	hn := len(gl.Hist)
	sn := hn
	if n > 0 {
		sn = min(n, hn)
	}
	lh := strings.Join(gl.Hist[hn-sn:hn], "\n")
	err = os.WriteFile(path, []byte(lh), 0666)
	if err != nil {
		return err
	}
	return nil
}

// OpenHistory opens Hist history lines from given file,
// e.g., ~/.goalhist
func (gl *Goal) OpenHistory(file string) error {
	path, err := homedir.Expand(file)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	gl.Hist = strings.Split(string(b), "\n")
	return nil
}

// AddCommand adds given command to list of available commands
func (gl *Goal) AddCommand(name string, cmd func(args ...string)) {
	gl.Commands[name] = cmd
}

// RunCommands runs the given command(s). This is typically called
// from a Makefile-style goal script.
func (gl *Goal) RunCommands(cmds []any) error {
	for _, cmd := range cmds {
		if cmdFun, hasCmd := gl.Commands[reflectx.ToString(cmd)]; hasCmd {
			cmdFun()
		} else {
			return errors.Log(fmt.Errorf("command %q not found", cmd))
		}
	}
	return nil
}

// DeleteJob deletes the given job and returns true if successful,
func (gl *Goal) DeleteJob(cmdIO *exec.CmdIO) bool {
	idx := slices.Index(gl.Jobs, cmdIO)
	if idx >= 0 {
		gl.Jobs = slices.Delete(gl.Jobs, idx, idx+1)
		return true
	}
	return false
}

// JobIDExpand expands %n job id values in args with the full PID
// returns number of PIDs expanded
func (gl *Goal) JobIDExpand(args []string) int {
	exp := 0
	for i, id := range args {
		if id[0] == '%' {
			idx, err := strconv.Atoi(id[1:])
			if err == nil {
				if idx > 0 && idx <= len(gl.Jobs) {
					jb := gl.Jobs[idx-1]
					if jb.Cmd != nil && jb.Cmd.Process != nil {
						args[i] = fmt.Sprintf("%d", jb.Cmd.Process.Pid)
						exp++
					}
				} else {
					gl.AddError(fmt.Errorf("goal: job number out of range: %d", idx))
				}
			}
		}
	}
	return exp
}
