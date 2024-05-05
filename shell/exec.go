// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"bytes"
	"strings"

	"cogentcore.org/core/base/reflectx"
)

// Exec executes the given command string, handling the given arguments appropriately.
// If there is any error, it adds it to the shell. It forwards output to
// [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (sh *Shell) Exec(cmd any, args ...any) {
	scmd, sargs := execArgs(cmd, args...)
	if !sh.RunBuiltin(scmd, sargs...) {
		sh.AddError(sh.Config.Run(scmd, sargs...))
	}
}

// Output executes the given command string, handling the given arguments
// appropriately. If there is any error, it adds it to the shell. It returns
// the stdout as a string and forwards stderr to [exec.Config.Stderr] appropriately.
func (sh *Shell) Output(cmd any, args ...any) string {
	scmd, sargs := execArgs(cmd, args...)
	oldStdout := sh.Config.Stdout
	buf := &bytes.Buffer{}
	sh.Config.Stdout = buf
	if !sh.RunBuiltin(scmd, sargs...) {
		_, err := sh.Config.Exec(scmd, sargs...)
		sh.AddError(err)
	}
	sh.Config.Stdout = oldStdout
	if sh.Config.Stdout != nil {
		sh.Config.Stdout.Write(buf.Bytes())
	}
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
func execArgs(cmd any, args ...any) (string, []string) {
	scmd := reflectx.ToString(cmd)
	sargs := make([]string, len(args))
	for i, a := range args {
		sargs[i] = reflectx.ToString(a)
	}
	return scmd, sargs
}
