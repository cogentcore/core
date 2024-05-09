// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"bytes"
	"fmt"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"github.com/mitchellh/go-homedir"
)

// Exec executes the given command string, waiting for the command to finish,
// handling the given arguments appropriately.
// If there is any error, it adds it to the shell, and triggers CancelExecution.
// It forwards output to [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (sh *Shell) Exec(cmd any, args ...any) {
	if len(sh.Errors) > 0 {
		return
	}
	scmd, sargs := sh.execArgs(cmd, args...)
	if !sh.RunBuiltin(scmd, sargs...) {
		cl := sh.ActiveSSH()
		if cl != nil {
			fmt.Println("ssh running command:", scmd)
			sh.AddError(cl.Run(scmd, sargs...))
		} else {
			sh.AddError(sh.Config.Run(scmd, sargs...))
		}
	}
}

// ExecErrOK executes the given command string, waiting for the command to finish,
// handling the given arguments appropriately.
// It does not stop execution if there is an error.
// If there is any error, it adds it to the shell. It forwards output to
// [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (sh *Shell) ExecErrOK(cmd any, args ...any) {
	scmd, sargs := sh.execArgs(cmd, args...)
	if !sh.RunBuiltin(scmd, sargs...) {
		sh.Config.Run(scmd, sargs...)
	}
}

// Start starts the given command string without waiting for it to finish,
// handling the given arguments appropriately.
// If there is any error, it adds it to the shell. It forwards output to
// [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (sh *Shell) Start(cmd any, args ...any) {
	scmd, sargs := sh.execArgs(cmd, args...)
	if !sh.RunBuiltin(scmd, sargs...) {
		sh.AddError(sh.Config.Start(scmd, sargs...))
	}
}

// Output executes the given command string, handling the given arguments
// appropriately. If there is any error, it adds it to the shell. It returns
// the stdout as a string and forwards stderr to [exec.Config.Stderr] appropriately.
func (sh *Shell) Output(cmd any, args ...any) string {
	scmd, sargs := sh.execArgs(cmd, args...)
	oldStdout := sh.Config.Stdout
	buf := &bytes.Buffer{}
	sh.Config.Stdout = buf
	if !sh.RunBuiltin(scmd, sargs...) {
		_, err := sh.Config.Exec(scmd, sargs...)
		sh.AddError(err)
	}
	sh.Config.Stdout = oldStdout
	return strings.TrimSuffix(buf.String(), "\n")
}

// OutputErrOK executes the given command string, handling the given arguments
// appropriately. If there is any error, it adds it to the shell. It returns
// the stdout as a string and forwards stderr to [exec.Config.Stderr] appropriately.
func (sh *Shell) OutputErrOK(cmd any, args ...any) string {
	scmd, sargs := sh.execArgs(cmd, args...)
	oldStdout := sh.Config.Stdout
	buf := &bytes.Buffer{}
	sh.Config.Stdout = buf
	if !sh.RunBuiltin(scmd, sargs...) {
		sh.Config.Exec(scmd, sargs...)
	}
	sh.Config.Stdout = oldStdout
	return strings.TrimSuffix(buf.String(), "\n")
}

func (sh *Shell) RunBuiltin(cmd string, args ...string) bool {
	fun, has := sh.Builtins[cmd]
	if !has {
		return false
	}
	sh.AddError(fun(args...))
	return true
}

// execArgs converts the given command and arguments into strings.
func (sh *Shell) execArgs(cmd any, args ...any) (string, []string) {
	scmd := reflectx.ToString(cmd)
	sargs := make([]string, len(args))
	for i, a := range args {
		s := reflectx.ToString(a)
		s, err := homedir.Expand(s)
		sh.AddError(err)
		sargs[i] = s
	}
	return scmd, sargs
}

// CancelExecution calls the Cancel() function if set.
func (sh *Shell) CancelExecution() {
	if sh.Cancel != nil {
		sh.Cancel()
	}
}
