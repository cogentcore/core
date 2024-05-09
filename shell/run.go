// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"bytes"
	"fmt"
	"strings"
)

// Run executes the given command string, waiting for the command to finish,
// handling the given arguments appropriately.
// If there is any error, it adds it to the shell, and triggers CancelExecution.
// It forwards output to [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (sh *Shell) Run(cmd any, args ...any) {
	if len(sh.Errors) > 0 {
		return
	}
	sh.Config.StdIO.StackStart()
	scmd, sargs := sh.ExecArgs(false, cmd, args...)
	if scmd == "" {
		return
	}
	if !sh.RunBuiltin(scmd, sargs...) {
		cl := sh.ActiveSSH()
		if cl != nil {
			fmt.Println("ssh running command:", scmd)
			// todo: need to update stdouts
			sh.AddError(cl.Run(scmd, sargs...))
		} else {
			sh.AddError(sh.Config.Run(scmd, sargs...))
		}
	}
	sh.Config.StdIO.PopToStart(false) // not err
}

// RunErrOK executes the given command string, waiting for the command to finish,
// handling the given arguments appropriately.
// It does not stop execution if there is an error.
// If there is any error, it adds it to the shell. It forwards output to
// [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (sh *Shell) RunErrOK(cmd any, args ...any) {
	sh.Config.StdIO.StackStart()
	scmd, sargs := sh.ExecArgs(true, cmd, args...)
	if scmd == "" {
		return
	}
	if !sh.RunBuiltin(scmd, sargs...) {
		sh.Config.Run(scmd, sargs...)
	}
	sh.Config.StdIO.PopToStart(false)
}

// Start starts the given command string without waiting for it to finish,
// handling the given arguments appropriately.
// If there is any error, it adds it to the shell. It forwards output to
// [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (sh *Shell) Start(cmd any, args ...any) {
	scmd, sargs := sh.ExecArgs(false, cmd, args...)
	if scmd == "" {
		return
	}
	if !sh.RunBuiltin(scmd, sargs...) {
		excmd, err := sh.Config.Start(scmd, sargs...)
		if excmd != nil {
			sh.Jobs = append(sh.Jobs, excmd) // todo: add files to this
		}
		sh.AddError(err)
	}
}

// Output executes the given command string, handling the given arguments
// appropriately. If there is any error, it adds it to the shell. It returns
// the stdout as a string and forwards stderr to [exec.Config.Stderr] appropriately.
func (sh *Shell) Output(cmd any, args ...any) string {
	buf := &bytes.Buffer{}
	sh.Config.StdIO.PushOut(buf)
	sh.Run(cmd, args...)
	sh.Config.StdIO.PopOut()
	return strings.TrimSuffix(buf.String(), "\n")
}

// OutputErrOK executes the given command string, handling the given arguments
// appropriately. If there is any error, it adds it to the shell. It returns
// the stdout as a string and forwards stderr to [exec.Config.Stderr] appropriately.
func (sh *Shell) OutputErrOK(cmd any, args ...any) string {
	buf := &bytes.Buffer{}
	sh.Config.StdIO.PushOut(buf)
	sh.RunErrOK(cmd, args...)
	sh.Config.StdIO.PopOut()
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
